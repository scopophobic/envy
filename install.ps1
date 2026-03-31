# Envo CLI Installer for Windows
# Usage: irm https://raw.githubusercontent.com/scopophobic/envy/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "scopophobic/envy"
$Binary = "envo"
$InstallDir = "$env:LOCALAPPDATA\envo\bin"

function Write-Info  { param($Msg) Write-Host "  > " -NoNewline -ForegroundColor Cyan; Write-Host $Msg }
function Write-Ok    { param($Msg) Write-Host "  ✓ " -NoNewline -ForegroundColor Green; Write-Host $Msg }
function Write-Fail  { param($Msg) Write-Host "  ✗ " -NoNewline -ForegroundColor Red; Write-Host $Msg; exit 1 }

Write-Host ""
Write-Host "  Envo CLI Installer" -ForegroundColor White
Write-Host "  Secure secret management for developers"
Write-Host ""

# Detect architecture
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq "Arm64") { "arm64" } else { "amd64" }

# Get latest release version
Write-Info "Checking latest version..."
try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "envo-installer" }
    $Version = $Release.tag_name
    $VersionNum = $Version -replace "^v", ""
} catch {
    Write-Fail "Could not determine latest version. Check https://github.com/$Repo/releases"
}

# Download
$Archive = "${Binary}_${VersionNum}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Archive"
$TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
$ZipPath = Join-Path $TempDir $Archive

Write-Info "Downloading $Binary $Version for windows/$Arch..."
try {
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath -UseBasicParsing
} catch {
    Write-Fail "Download failed. Is $Version released for windows/$Arch?"
}

# Extract
Write-Info "Extracting..."
Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

# Install
Write-Info "Installing to $InstallDir..."
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path (Join-Path $TempDir "$Binary.exe") -Destination (Join-Path $InstallDir "$Binary.exe") -Force

# Clean up
Remove-Item -Recurse -Force $TempDir

# Add to PATH if not already there
$UserPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Info "Adding $InstallDir to your PATH..."
    [System.Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

Write-Ok "Installed $Binary $Version"
Write-Host ""
Write-Host "  Get started:"
Write-Host "    envo login" -ForegroundColor Cyan -NoNewline; Write-Host "                                    Sign in via browser"
Write-Host "    envo pull --org my-team --project api --env dev" -ForegroundColor Cyan -NoNewline; Write-Host "   Pull secrets to .env"
Write-Host "    envo run  --org my-team --project api --env dev -- npm start" -ForegroundColor Cyan
Write-Host "                                                   Run with secrets injected"
Write-Host ""
Write-Host "  Restart your terminal if 'envo' is not found." -ForegroundColor Yellow
Write-Host ""
