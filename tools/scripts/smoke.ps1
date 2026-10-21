# 后端主链路冒烟（需 webhttp 已启动 + MySQL 可用）
$ErrorActionPreference = 'Stop'
$base = 'http://127.0.0.1:8080/api/v1'

Write-Host '== login =='
$login = Invoke-RestMethod -Method POST "$base/auth/login" -ContentType 'application/json' -Body '{"username":"admin","password":"Admin@123","mode":"local"}'
$token = $login.data.accessToken
$h = @{ Authorization = "Bearer $token"; 'X-Client' = 'admin' }

Write-Host '== material =='
$mat = Invoke-RestMethod -Method POST "$base/materials" -Headers $h -ContentType 'application/json' -Body (@{code=('M'+[guid]::NewGuid().ToString('N').Substring(0,6)); name='冒烟物资'; unit='只'} | ConvertTo-Json)

Write-Host '== inbound =='
$in = Invoke-RestMethod -Method POST "$base/inbound/orders" -Headers $h -ContentType 'application/json' -Body (@{warehouseId='WH01'; materialId=$mat.data.ID; qty=1} | ConvertTo-Json)
$codes = Invoke-RestMethod -Method POST "$base/inbound/orders/$($in.data.ID)/issue-codes" -Headers $h -ContentType 'application/json' -Body '{"count":1}'
$code = $codes.data[0].Code
Invoke-RestMethod -Method POST "$base/inbound/orders/$($in.data.ID)/confirm" -Headers $h -ContentType 'application/json' -Body (@{uniqueCode=$code; clientRequestId=[guid]::NewGuid().Guid} | ConvertTo-Json) | Out-Null

Write-Host '== alarm eval =='
Invoke-RestMethod -Method PUT "$base/thresholds/WH01" -Headers $h -ContentType 'application/json' -Body '{"metric":"temperature","upper":30}' | Out-Null
Invoke-RestMethod -Method POST "$base/monitor/evaluate" -Headers $h -ContentType 'application/json' -Body '{"warehouseId":"WH01","deviceId":"DEV-SMOKE","metrics":{"temperature":99}}' | Out-Null

Write-Host 'SMOKE OK code=' $code
