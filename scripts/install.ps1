# OOF RL installer / updater.
#
# Ships in the root of the release zip. Run it from the extracted folder:
#   powershell -ExecutionPolicy Bypass -File install.ps1
#
# Installs (or updates in place) by copying the extracted files into the
# install folder. User data (database, logs, settings) lives in
# %LOCALAPPDATA%\OOF_RL and is never touched.
param(
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "Programs\OOF_RL"),
    [switch]$NoLaunch,
    [switch]$NoShortcut
)

$ErrorActionPreference = "Stop"

$sourceDir = $PSScriptRoot
$exeName = "oof_rl.exe"
$sourceExe = Join-Path $sourceDir $exeName
if (-not (Test-Path -LiteralPath $sourceExe)) {
    throw "$exeName not found next to install.ps1 - run this script from the extracted release folder."
}

# When OOF RL is already running, update whatever copy is running rather than
# the default folder (unless the caller pinned -InstallDir explicitly).
$running = Get-Process -Name "oof_rl" -ErrorAction SilentlyContinue
if ($running) {
    if (-not $PSBoundParameters.ContainsKey('InstallDir')) {
        $runningPath = ($running | Select-Object -First 1).Path
        if ($runningPath) {
            $InstallDir = Split-Path -Parent $runningPath
        }
    }
    Write-Host "Stopping running OOF RL..."
    $running | ForEach-Object { $null = $_.CloseMainWindow() }
    try {
        $running | Wait-Process -Timeout 10 -ErrorAction Stop
    } catch {
        Get-Process -Name "oof_rl" -ErrorAction SilentlyContinue | Stop-Process -Force
    }
    Start-Sleep -Milliseconds 500
}

# Installing into the extracted folder (or anywhere inside it) would
# recursively copy the folder into itself.
$resolvedInstall = [System.IO.Path]::GetFullPath($InstallDir).TrimEnd('\')
$resolvedSource = [System.IO.Path]::GetFullPath($sourceDir).TrimEnd('\')
if ($resolvedInstall -eq $resolvedSource -or
    $resolvedInstall.StartsWith($resolvedSource + '\', [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Install folder is inside the extracted folder - extract the zip somewhere else (e.g. Downloads) and run install.ps1 from there."
}

Write-Host "Installing to $InstallDir"
New-Item -ItemType Directory -Force $InstallDir | Out-Null

# Copy everything from the extracted folder; -Force overwrites the previous
# version in place.
Get-ChildItem -LiteralPath $sourceDir | ForEach-Object {
    Copy-Item -LiteralPath $_.FullName -Destination $InstallDir -Recurse -Force
}

# Files extracted by Explorer carry the Mark-of-the-Web, which makes Windows
# warn on every launch; clear it from the installed copy.
Get-ChildItem -LiteralPath $InstallDir -Recurse -File | Unblock-File -ErrorAction SilentlyContinue

if (-not $NoShortcut) {
    $shortcutPath = Join-Path ([Environment]::GetFolderPath("Programs")) "OOF RL.lnk"
    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = Join-Path $InstallDir $exeName
    $shortcut.WorkingDirectory = $InstallDir
    $shortcut.Save()
    Write-Host "Created Start Menu shortcut: $shortcutPath"
}

Write-Host "Done."
if (-not $NoLaunch) {
    Start-Process -FilePath (Join-Path $InstallDir $exeName) -WorkingDirectory $InstallDir
}
