# One-line install for Envo CLI (Windows PowerShell).
# Usage: irm https://raw.githubusercontent.com/OWNER/REPO/main/install.ps1 | iex
#
# Or: Invoke-RestMethod -Uri ... | Invoke-Expression

$ErrorActionPreference = "Stop"

$Repo = if ($env:ENVO_REPO) { $env:ENVO_REPO } else { "envo/cli" }
$Version = if ($env:ENVO_VERSION) { $env:ENVO_VERSION } else { "latest" }

if ($Version -eq "latest") {
  $BaseUrl = "https://github.com/$Repo/releases/latest/download"
} else {
  $BaseUrl = "https://github.com/$Repo/releases/download/$Version"
}

# Prefer amd64 on Windows (arm64 is less common)
$Arch = "amd64"
$Asset = "envo-windows-$Arch.exe"
$Url = "$BaseUrl/$Asset"

$InstallDir = if ($env:ENVO_INSTALL_DIR) { $env:ENVO_INSTALL_DIR } else { "$env:USERPROFILE\bin" }
$BinPath = Join-Path $InstallDir "envo.exe"

if (-not (Test-Path $InstallDir)) {
  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Write-Host "Installing envo to $BinPath"
Write-Host "Downloading $Url ..."

try {
  Invoke-WebRequest -Uri $Url -OutFile $BinPath -UseBasicParsing
} catch {
  Write-Error "Download failed. Ensure the release exists and the asset is named: $Asset"
  throw
}

Write-Host "Installed: $BinPath"

# Add to user PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
  Write-Host ""
  Write-Host "Added $InstallDir to your user PATH. Restart the terminal, then run: envo whoami"
} else {
  Write-Host ""
  Write-Host "Run: envo whoami"
}
