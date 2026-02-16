# Run Envo CLI without building. From repo root: .\envo.ps1 login
$env:ENVO_CALLER_DIR = (Get-Location).Path
Set-Location $PSScriptRoot\cli
go run ./cmd/envo @args
