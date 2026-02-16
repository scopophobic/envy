@echo off
set ENVO_CALLER_DIR=%CD%
cd /d "%~dp0cli"
go run ./cmd/envo %*
