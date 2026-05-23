# Radar installer for Windows
# Usage: irm https://raw.githubusercontent.com/skyhook-io/radar/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "skyhook-io/radar"
$BinaryName = "kubectl-radar.exe"
$InstallDir = "$env:LOCALAPPDATA\radar"

# Only amd64 is supported on Windows
$Arch = "amd64"

# Get latest release version
Write-Host "Fetching latest release..." -ForegroundColor Cyan
try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name -replace '^v', ''
} catch {
    Write-Host "Failed to fetch latest version: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Installing Radar v$Version..." -ForegroundColor Cyan

# Download
$Filename = "radar_v${Version}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/v$Version/$Filename"
$TmpDir = Join-Path $env:TEMP "radar-install-$(Get-Random)"
$ZipPath = Join-Path $TmpDir $Filename

New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

Write-Host "Downloading $DownloadUrl..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
} catch {
    Write-Host "Failed to download: $_" -ForegroundColor Red
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
    exit 1
}

# Extract
Write-Host "Extracting..." -ForegroundColor Cyan
Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

# Install
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Move-Item -Path (Join-Path $TmpDir $BinaryName) -Destination $InstallDir -Force

# Create radar.exe symlink/copy for convenience
$RadarExe = Join-Path $InstallDir "radar.exe"
if (-not (Test-Path $RadarExe)) {
    Copy-Item -Path (Join-Path $InstallDir $BinaryName) -Destination $RadarExe
}

# Cleanup
Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

Write-Host ""
Write-Host "Radar v$Version installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Usage:" -ForegroundColor Yellow
Write-Host "  kubectl radar          # as kubectl plugin"
Write-Host "  kubectl-radar          # standalone"
Write-Host "  radar                  # short alias"
Write-Host ""
Write-Host "Run 'radar --help' for more options." -ForegroundColor Yellow
Write-Host ""
Write-Host "Note: Restart your terminal for PATH changes to take effect." -ForegroundColor Cyan
