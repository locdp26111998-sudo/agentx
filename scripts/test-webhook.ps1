# Gửi tin thử tới webhook giả (server phải chạy cổng 8090).
# Router chọn agent theo page_id trong body JSON.

param(
    [string]$Text = "Xin chao",
    [string]$SenderID = "khach-test",
    [string]$PageID = "111111111"
)

$body = @{
    sender_id = $SenderID
    text      = $Text
    page_id   = $PageID
} | ConvertTo-Json -Compress

Write-Host "POST /webhook page_id=$PageID text=$Text"

try {
    $resp = Invoke-RestMethod -Uri "http://127.0.0.1:8090/webhook" `
        -Method POST -ContentType "application/json" -Body $body
    if ($resp.agent_id) {
        Write-Host "agent_id:" $resp.agent_id
    }
    Write-Host "reply:" $resp.reply
} catch {
    if ($_.ErrorDetails.Message) {
        Write-Host "error:" $_.ErrorDetails.Message
    } else {
        Write-Host "error:" $_.Exception.Message
    }
}
