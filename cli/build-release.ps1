# Build all Envo CLI binaries for GitHub Release (Windows).
# Run from repo root or from cli/
# Output: dist/envo-{os}-{arch}.exe

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

New-Item -ItemType Directory -Force -Path dist | Out-Null

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/envo-windows-amd64.exe ./cmd/envo

$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/envo-darwin-amd64 ./cmd/envo

$env:GOOS = "darwin"
$env:GOARCH = "arm64"
go build -ldflags="-s -w" -o dist/envo-darwin-arm64 ./cmd/envo

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o dist/envo-linux-amd64 ./cmd/envo

$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -ldflags="-s -w" -o dist/envo-linux-arm64 ./cmd/envo

Remove-Item Env:GOOS
Remove-Item Env:GOARCH

Write-Host "Done. Binaries in cli/dist/:"
Get-ChildItem dist
