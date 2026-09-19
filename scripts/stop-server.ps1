# Tat process dang chiem cong 8090 (server agentx cu).

$conn = Get-NetTCPConnection -LocalPort 8090 -State Listen -ErrorAction SilentlyContinue

if (-not $conn) {
    Write-Host "Khong co server nao dang chay tren cong 8090."
    exit 0
}

$procIds = $conn.OwningProcess | Sort-Object -Unique
foreach ($procId in $procIds) {
    Write-Host "Dang tat process PID" $procId "..."
    Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
}

Write-Host "Da tat server tren cong 8090. Gio co the chay run-server.ps1 lai."
