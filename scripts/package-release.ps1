param(
    [string]$Version = "",
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$distDir = Join-Path $repoRoot $OutputDir
$packageRoot = Join-Path $distDir "OOF_RL"
# WASM plugins + assets are staged here so go:embed (all:bundled/plugins in
# plugin_bundle.go) links them into the exe. This is build-time staging only:
# bundled/plugins/* is gitignored and reset to the .gitkeep placeholder when
# the build finishes, preserving the "dev builds embed only the placeholder"
# invariant.
$bundledPluginsDir = Join-Path $repoRoot "bundled\plugins"
$archiveName = if ([string]::IsNullOrWhiteSpace($Version)) { "OOF_RL.zip" } else { "OOF_RL-$Version.zip" }
$archivePath = Join-Path $distDir $archiveName
$checksumPath = "$archivePath.sha256"
$manifestPath = Join-Path $distDir "update-manifest.json"
$plugins = @("live", "ranks", "session", "dashboard")

function Invoke-Checked([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command exited with code $LASTEXITCODE"
    }
}

function Assert-UnderRepo([string]$Path) {
    $full = [System.IO.Path]::GetFullPath($Path)
    $root = [System.IO.Path]::GetFullPath($repoRoot)
    if (-not $full.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to write outside repository: $full"
    }
}

function Copy-PluginAssets([string]$Plugin) {
    $assets = Join-Path $repoRoot "plugins\$Plugin\assets"
    if (-not (Test-Path -LiteralPath $assets)) {
        return
    }
    $dest = Join-Path $bundledPluginsDir $Plugin
    New-Item -ItemType Directory -Force $dest | Out-Null
    Copy-Item -Path (Join-Path $assets "*") -Destination $dest -Recurse -Force
}

# Remove only the wasm + asset dirs this script stages for embedding, leaving
# the tracked .gitkeep placeholder so a plain `go build .` embeds nothing.
# Scoped to $plugins so it can never delete an unexpected file.
function Reset-BundledPlugins {
    foreach ($plugin in $plugins) {
        $wasm = Join-Path $bundledPluginsDir "$plugin.wasm"
        if (Test-Path -LiteralPath $wasm) {
            Remove-Item -LiteralPath $wasm -Force
        }
        $assetDir = Join-Path $bundledPluginsDir $plugin
        if (Test-Path -LiteralPath $assetDir) {
            Remove-Item -LiteralPath $assetDir -Recurse -Force
        }
    }
}

Assert-UnderRepo $distDir
Assert-UnderRepo $packageRoot
Assert-UnderRepo $bundledPluginsDir

if (Test-Path -LiteralPath $packageRoot) {
    Remove-Item -LiteralPath $packageRoot -Recurse -Force
}
if (Test-Path -LiteralPath $archivePath) {
    Remove-Item -LiteralPath $archivePath -Force
}
if (Test-Path -LiteralPath $checksumPath) {
    Remove-Item -LiteralPath $checksumPath -Force
}
if (Test-Path -LiteralPath $manifestPath) {
    Remove-Item -LiteralPath $manifestPath -Force
}

New-Item -ItemType Directory -Force $packageRoot | Out-Null
Reset-BundledPlugins
if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $distDir ".gocache"
    New-Item -ItemType Directory -Force $env:GOCACHE | Out-Null
}

Push-Location $repoRoot
try {
    # Embed Windows VERSIONINFO matching the release tag. An exe with no/stale
    # version resource is a common antivirus false-positive trigger; goversioninfo
    # writes CompanyName/ProductName/FileDescription plus the numeric version into
    # rsrc.syso, which `go build` links automatically. See docs/dev/antivirus.md.
    if (-not [string]::IsNullOrWhiteSpace($Version) -and ($Version -match '^v?(\d+)\.(\d+)\.(\d+)')) {
        $verMajor = [int]$Matches[1]; $verMinor = [int]$Matches[2]; $verPatch = [int]$Matches[3]
        $verStr = "$verMajor.$verMinor.$verPatch.0"
        # Pinned: this runs at release-build time and its output is linked into
        # the shipped exe, so "@latest" would let a compromised or broken new
        # version flow straight into a release.
        Invoke-Checked "go" @("install", "github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0")
        $gvi = Join-Path (& go env GOPATH) "bin\goversioninfo.exe"
        Invoke-Checked $gvi @(
            "-o", "rsrc.syso", "-icon", "icon.ico",
            "-ver-major", "$verMajor", "-ver-minor", "$verMinor", "-ver-patch", "$verPatch", "-ver-build", "0",
            "-product-ver-major", "$verMajor", "-product-ver-minor", "$verMinor", "-product-ver-patch", "$verPatch", "-product-ver-build", "0",
            "-file-version", $verStr, "-product-version", $verStr,
            "versioninfo.json"
        )
    }

    # Build the WASM plugins into the embed staging dir FIRST: go:embed runs at
    # compile time, so these must exist before the exe build below or the exe
    # would embed only the placeholder and ship with no public plugins.
    $oldGOOS = $env:GOOS
    $oldGOARCH = $env:GOARCH
    try {
        $env:GOOS = "wasip1"
        $env:GOARCH = "wasm"
        foreach ($plugin in $plugins) {
            $wasmPath = Join-Path $bundledPluginsDir "$plugin.wasm"
            Invoke-Checked "go" @("-C", "plugins\$plugin", "build", "-buildmode=c-shared", "-o", $wasmPath, ".")
            Copy-PluginAssets $plugin
        }
    } finally {
        $env:GOOS = $oldGOOS
        $env:GOARCH = $oldGOARCH
    }

    $exePath = Join-Path $packageRoot "oof_rl.exe"
    $ldflags = "-H windowsgui -s -w"
    if (-not [string]::IsNullOrWhiteSpace($Version)) {
        $ldflags = "$ldflags -X OOF_RL/internal/config.AppVersion=$Version"
    }
    Invoke-Checked "go" @("build", "-ldflags=$ldflags", "-o", $exePath, ".")

    Copy-Item -LiteralPath (Join-Path $PSScriptRoot "install.ps1") -Destination $packageRoot

    @"
OOF RL

Install or update (recommended): right-click install.ps1 and choose
"Run with PowerShell", or from a terminal in this folder:

    powershell -ExecutionPolicy Bypass -File install.ps1

The script copies the app to %LOCALAPPDATA%\Programs\OOF_RL (stopping a
running copy first), creates a Start Menu shortcut, and launches it. Your
data (database, logs, settings) lives in %LOCALAPPDATA%\OOF_RL and is never
touched. You can also skip the script and just run oof_rl.exe from this
folder, or copy oof_rl.exe anywhere - it is fully self-contained.

The public plugins are built into oof_rl.exe. On startup, OOF RL copies them
into %LOCALAPPDATA%\OOF_RL\plugins so the public pages are available without
running developer build commands.
"@ | Out-File -Encoding utf8 (Join-Path $packageRoot "README.txt")

    Compress-Archive -LiteralPath $packageRoot -DestinationPath $archivePath -Force
    $hash = (Get-FileHash $archivePath -Algorithm SHA256).Hash.ToLower()
    "$hash  $archiveName" | Out-File -Encoding utf8 $checksumPath

    # update-manifest.json is what the in-app update checker fetches from
    # releases/latest/download/. Only meaningful for tagged builds.
    if (-not [string]::IsNullOrWhiteSpace($Version)) {
        # A semver prerelease tag (e.g. v0.0.8-dev.1) is a dev-channel build;
        # a plain vMAJOR.MINOR.PATCH tag is stable. The channel is informational
        # in the manifest — which release the manifest is attached to is what
        # actually gates who sees it (see .github/workflows/ci.yml).
        $channel = if ($Version -match '-') { "dev" } else { "stable" }
        $manifest = [ordered]@{
            version         = $Version
            channel         = $channel
            notes_url       = "https://github.com/erosas/OOF_RL/releases/tag/$Version"
            published_at    = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
            artifact_url    = "https://github.com/erosas/OOF_RL/releases/download/$Version/$archiveName"
            artifact_name   = $archiveName
            artifact_sha256 = $hash
        }
        # WriteAllText with BOM-less UTF8: Windows PowerShell 5.1's
        # Out-File -Encoding utf8 prepends a BOM, which JSON parsers reject.
        $json = ($manifest | ConvertTo-Json)
        [System.IO.File]::WriteAllText($manifestPath, $json, (New-Object System.Text.UTF8Encoding($false)))
        Write-Host "Created $manifestPath"
    }

    Write-Host "Created $archivePath"
    Write-Host "Created $checksumPath"
    Get-ChildItem -LiteralPath $packageRoot -Recurse | ForEach-Object {
        Write-Host $_.FullName.Substring($packageRoot.Length + 1)
    }
} finally {
    # Reset staging even on failure, but never let its errors skip Pop-Location.
    try {
        Reset-BundledPlugins
    } finally {
        Pop-Location
    }
}
