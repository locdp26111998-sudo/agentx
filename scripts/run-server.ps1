# Chay server agentx
#   .\scripts\run-server.ps1
#   .\scripts\run-server.ps1 -Mode timeout
#   .\scripts\run-server.ps1 -Mode notfound

param(
    [ValidateSet("normal", "timeout", "notfound")]
    [string]$Mode = "normal"
)

$repoRoot = Split-Path $PSScriptRoot -Parent
Set-Location $repoRoot

Remove-Item Env:AGENTX_ENGINE_TIMEOUT -ErrorAction SilentlyContinue
Remove-Item Env:AGENTX_ENGINE_COMMAND -ErrorAction SilentlyContinue

if ($Mode -eq "timeout") {
    $env:AGENTX_ENGINE_TIMEOUT = "2s"
}
if ($Mode -eq "notfound") {
    $env:AGENTX_ENGINE_COMMAND = "claude-sai-ten"
}

Write-Host "agentx server started"
Write-Host "mode:" $Mode
Write-Host "folder:" $repoRoot

go run ./cmd/server
