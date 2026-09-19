# Kiểm tra mọi configs/agents/*.yaml load được. Chạy từ repo: .\scripts\verify-agent-configs.ps1
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path))
go run ./scripts/verify-agent-configs/
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
