# Build Envo CLI for current platform (Windows).
# Run from cli/ folder: .\build.ps1
# Output: envo.exe in cli/

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
go build -o envo.exe ./cmd/envo
Write-Host "Built: .\envo.exe"
Write-Host "Run: .\envo.exe whoami"
