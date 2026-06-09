param(
    [string]$Version = "",
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$distDir = Join-Path $repoRoot $OutputDir
$packageRoot = Join-Path $distDir "OOF_RL"
$pluginsDir = Join-Path $packageRoot "plugins"
$archiveName = if ([string]::IsNullOrWhiteSpace($Version)) { "OOF_RL.zip" } else { "OOF_RL-$Version.zip" }
$archivePath = Join-Path $distDir $archiveName
$checksumPath = "$archivePath.sha256"
$plugins = @("live", "ranks", "session", "dashboard", "autoupdate")

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
    $dest = Join-Path $pluginsDir $Plugin
    New-Item -ItemType Directory -Force $dest | Out-Null
    Copy-Item -Path (Join-Path $assets "*") -Destination $dest -Recurse -Force
}

Assert-UnderRepo $distDir
Assert-UnderRepo $packageRoot

if (Test-Path -LiteralPath $packageRoot) {
    Remove-Item -LiteralPath $packageRoot -Recurse -Force
}
if (Test-Path -LiteralPath $archivePath) {
    Remove-Item -LiteralPath $archivePath -Force
}
if (Test-Path -LiteralPath $checksumPath) {
    Remove-Item -LiteralPath $checksumPath -Force
}

New-Item -ItemType Directory -Force $pluginsDir | Out-Null
if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $distDir ".gocache"
    New-Item -ItemType Directory -Force $env:GOCACHE | Out-Null
}

Push-Location $repoRoot
try {
    $exePath = Join-Path $packageRoot "oof_rl.exe"
    $ldflags = "-H windowsgui -s -w"
    if (-not [string]::IsNullOrWhiteSpace($Version)) {
        $ldflags = "$ldflags -X OOF_RL/internal/config.AppVersion=$Version"
    }
    Invoke-Checked "go" @("build", "-ldflags=$ldflags", "-o", $exePath, ".")

    $oldGOOS = $env:GOOS
    $oldGOARCH = $env:GOARCH
    try {
        $env:GOOS = "wasip1"
        $env:GOARCH = "wasm"
        foreach ($plugin in $plugins) {
            $wasmPath = Join-Path $pluginsDir "$plugin.wasm"
            Invoke-Checked "go" @("-C", "plugins\$plugin", "build", "-buildmode=c-shared", "-o", $wasmPath, ".")
            Copy-PluginAssets $plugin
        }
    } finally {
        $env:GOOS = $oldGOOS
        $env:GOARCH = $oldGOARCH
    }

    @"
OOF RL

Run oof_rl.exe from this folder.

This release includes bundled public plugins in the plugins folder. On startup,
OOF RL copies those bundled plugins into %LOCALAPPDATA%\OOF_RL\plugins so the
public pages and Settings plugin actions are available without running developer
build commands.
"@ | Out-File -Encoding utf8 (Join-Path $packageRoot "README.txt")

    Compress-Archive -LiteralPath $packageRoot -DestinationPath $archivePath -Force
    $hash = (Get-FileHash $archivePath -Algorithm SHA256).Hash.ToLower()
    "$hash  $archiveName" | Out-File -Encoding utf8 $checksumPath

    Write-Host "Created $archivePath"
    Write-Host "Created $checksumPath"
    Get-ChildItem -LiteralPath $packageRoot -Recurse | ForEach-Object {
        Write-Host $_.FullName.Substring($packageRoot.Length + 1)
    }
} finally {
    Pop-Location
}
