# Watcher backend smoke test
# Covers: login, material, inbound, outbound lock/receive, stocks, threshold alarm,
# geofence, approval adjust, transport/inventory/inspection lists
# Usage:
#   powershell -File tools/scripts/smoke.ps1
#   powershell -File tools/scripts/smoke.ps1 -AutoStart
param(
    [string]$BaseUrl = 'http://127.0.0.1:8080',
    [switch]$AutoStart,
    [string]$MysqlBasedir = $env:WATCHER_MYSQL_BASEDIR,
    [int]$MysqlPort = 13306,
    [int]$HttpPort = 18080,
    [switch]$KeepAlive
)

$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$smokeDir = Join-Path $repoRoot '.smoke'
$mysqlData = Join-Path $smokeDir 'mysql-data'
$mysqlPidFile = Join-Path $smokeDir 'mysqld.pid'
$webhttpPidFile = Join-Path $smokeDir 'webhttp.pid'
$webhttpLog = Join-Path $smokeDir 'webhttp.log'
$script:startedMysql = $false
$script:startedWebhttp = $false

function Write-Step([string]$msg) { Write-Host "== $msg ==" -ForegroundColor Cyan }

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Path,
        [hashtable]$Headers = $null,
        [object]$Body = $null,
        [string]$Label = ''
    )
    $uri = "$script:apiBase$Path"
    $params = @{ Method = $Method; Uri = $uri }
    if ($Headers) { $params.Headers = $Headers }
    if ($null -ne $Body) {
        $params.ContentType = 'application/json'
        if ($Body -is [string]) { $params.Body = $Body }
        else { $params.Body = ($Body | ConvertTo-Json -Depth 8 -Compress) }
    }
    try {
        $resp = Invoke-RestMethod @params
    } catch {
        $detail = $_.Exception.Message
        if ($_.ErrorDetails -and $_.ErrorDetails.Message) { $detail = $_.ErrorDetails.Message }
        throw "API failed [$Label $Method $Path]: $detail"
    }
    if ($null -eq $resp) { throw "empty response [$Label $Method $Path]" }
    if ($resp.PSObject.Properties.Name -contains 'code' -and $resp.code -and $resp.code -ne 'OK' -and $resp.code -ne '0') {
        throw "biz failed [$Label $Method $Path] code=$($resp.code) message=$($resp.message)"
    }
    return $resp
}

function Test-Health([string]$root) {
    try {
        $h = Invoke-RestMethod -Method GET "$root/healthz" -TimeoutSec 3
        return ($h.status -eq 'ok')
    } catch { return $false }
}

function Resolve-MysqlBasedir {
    if ($MysqlBasedir -and (Test-Path (Join-Path $MysqlBasedir 'bin\mysqld.exe'))) { return $MysqlBasedir }
    $candidates = @(
        'D:\SoftWare\mysql-5.7.35-winx64',
        'C:\Program Files\MySQL\MySQL Server 8.0',
        'C:\Program Files\MySQL\MySQL Server 5.7',
        'C:\mysql'
    )
    foreach ($c in $candidates) {
        if (Test-Path (Join-Path $c 'bin\mysqld.exe')) { return $c }
    }
    $cmd = Get-Command mysqld -ErrorAction SilentlyContinue
    if ($cmd) {
        return (Split-Path (Split-Path $cmd.Source -Parent) -Parent)
    }
    return $null
}

function Wait-Port([string]$HostName, [int]$Port, [int]$TimeoutSec = 60) {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $c = New-Object System.Net.Sockets.TcpClient
            $iar = $c.BeginConnect($HostName, $Port, $null, $null)
            $ok = $iar.AsyncWaitHandle.WaitOne(500)
            if ($ok -and $c.Connected) { $c.Close(); return }
            $c.Close()
        } catch {}
        Start-Sleep -Milliseconds 400
    }
    throw "wait $HostName`:$Port timeout (${TimeoutSec}s)"
}

function Start-SmokeMysql {
    $basedir = Resolve-MysqlBasedir
    if (-not $basedir) {
        throw 'mysqld not found. Install MySQL or set WATCHER_MYSQL_BASEDIR / -MysqlBasedir'
    }
    $mysqld = Join-Path $basedir 'bin\mysqld.exe'
    $mysql = Join-Path $basedir 'bin\mysql.exe'
    New-Item -ItemType Directory -Force -Path $smokeDir | Out-Null

    if (-not (Test-Path (Join-Path $mysqlData 'mysql'))) {
        Write-Step "init ephemeral MySQL datadir ($mysqlData)"
        & $mysqld --no-defaults --basedir=$basedir --datadir=$mysqlData --initialize-insecure
        if ($LASTEXITCODE -ne 0) { throw "mysqld --initialize-insecure failed exit=$LASTEXITCODE" }
    }

    Write-Step "start ephemeral MySQL :$MysqlPort"
    $proc = Start-Process -FilePath $mysqld -ArgumentList @(
        '--no-defaults',
        "--basedir=$basedir",
        "--datadir=$mysqlData",
        "--port=$MysqlPort",
        '--bind-address=127.0.0.1',
        '--skip-ssl',
        '--character-set-server=utf8mb4',
        '--collation-server=utf8mb4_unicode_ci'
    ) -PassThru -WindowStyle Hidden
    $proc.Id | Set-Content -Path $mysqlPidFile -Encoding ascii
    $script:startedMysql = $true
    Wait-Port '127.0.0.1' $MysqlPort 90

    $sql = @"
CREATE DATABASE IF NOT EXISTS watcher CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON watcher.* TO 'watcher'@'localhost' IDENTIFIED BY 'watcher';
GRANT ALL PRIVILEGES ON watcher.* TO 'watcher'@'127.0.0.1' IDENTIFIED BY 'watcher';
GRANT ALL PRIVILEGES ON watcher.* TO 'watcher'@'%' IDENTIFIED BY 'watcher';
FLUSH PRIVILEGES;
"@
    $tmpSql = Join-Path $smokeDir 'bootstrap.sql'
    Set-Content -Path $tmpSql -Value $sql -Encoding ASCII
    & $mysql --no-defaults -uroot --host=127.0.0.1 --port=$MysqlPort --protocol=tcp --ssl-mode=DISABLED -e "source $($tmpSql -replace '\\','/')"
    if ($LASTEXITCODE -ne 0) {
        & $mysql --no-defaults -uroot --host=127.0.0.1 --port=$MysqlPort --protocol=tcp --skip-ssl -e "source $($tmpSql -replace '\\','/')"
        if ($LASTEXITCODE -ne 0) {
            # fallback: pipe SQL via stdin
            Get-Content $tmpSql -Raw | & $mysql --no-defaults -uroot --host=127.0.0.1 --port=$MysqlPort --protocol=tcp --skip-ssl
            if ($LASTEXITCODE -ne 0) { throw "bootstrap watcher user failed exit=$LASTEXITCODE" }
        }
    }
    return "watcher:watcher@tcp(127.0.0.1:$MysqlPort)/watcher?charset=utf8mb4&parseTime=True&loc=Local"
}

function Start-SmokeWebhttp([string]$dsn, [int]$port) {
    Write-Step "start webhttp :$port"
    New-Item -ItemType Directory -Force -Path $smokeDir | Out-Null
    $stamp = Get-Date -Format 'yyyyMMddHHmmss'
    $outLog = Join-Path $smokeDir ("webhttp-$stamp.log")
    $errLog = Join-Path $smokeDir ("webhttp-$stamp.err.log")
    $script:webhttpLog = $outLog

    $prevDsn = $env:WATCHER_MYSQL_DSN
    $prevAddr = $env:WATCHER_HTTP_ADDR
    $env:WATCHER_MYSQL_DSN = $dsn
    $env:WATCHER_HTTP_ADDR = ":$port"
    try {
        $p = Start-Process -FilePath 'go' -ArgumentList @('run', './backend/cmd/webhttp') `
            -WorkingDirectory $repoRoot -PassThru -WindowStyle Hidden `
            -RedirectStandardOutput $outLog -RedirectStandardError $errLog
    } finally {
        if ($null -eq $prevDsn) { Remove-Item Env:WATCHER_MYSQL_DSN -ErrorAction SilentlyContinue } else { $env:WATCHER_MYSQL_DSN = $prevDsn }
        if ($null -eq $prevAddr) { Remove-Item Env:WATCHER_HTTP_ADDR -ErrorAction SilentlyContinue } else { $env:WATCHER_HTTP_ADDR = $prevAddr }
    }
    $p.Id | Set-Content -Path $webhttpPidFile -Encoding ascii
    $script:startedWebhttp = $true

    $root = "http://127.0.0.1:$port"
    $deadline = (Get-Date).AddSeconds(120)
    while ((Get-Date) -lt $deadline) {
        if ($p.HasExited) {
            $err = ''
            if (Test-Path $errLog) { $err = Get-Content $errLog -Raw -ErrorAction SilentlyContinue }
            throw "webhttp exited early code=$($p.ExitCode) err=$err (see $webhttpLog)"
        }
        if (Test-Health $root) { return $root }
        Start-Sleep -Milliseconds 500
    }
    throw "webhttp healthz timeout (see $webhttpLog / $errLog)"
}

function Stop-SmokeProcesses {
    if ($KeepAlive) {
        Write-Host "KeepAlive: leave ephemeral MySQL/webhttp under $smokeDir" -ForegroundColor Yellow
        return
    }
    if ($script:startedWebhttp -and (Test-Path $webhttpPidFile)) {
        $id = Get-Content $webhttpPidFile | Select-Object -First 1
        if ($id) { Stop-Process -Id ([int]$id) -Force -ErrorAction SilentlyContinue }
        Remove-Item $webhttpPidFile -Force -ErrorAction SilentlyContinue
    }
    if ($script:startedMysql -and (Test-Path $mysqlPidFile)) {
        $id = Get-Content $mysqlPidFile | Select-Object -First 1
        if ($id) { Stop-Process -Id ([int]$id) -Force -ErrorAction SilentlyContinue }
        Remove-Item $mysqlPidFile -Force -ErrorAction SilentlyContinue
    }
}

try {
    $root = $BaseUrl.TrimEnd('/')
    if (-not (Test-Health $root)) {
        if (-not $AutoStart) {
            throw "webhttp not ready: $root/healthz . Start it first, or pass -AutoStart"
        }
        $dsn = Start-SmokeMysql
        $root = Start-SmokeWebhttp $dsn $HttpPort
    }
    $script:apiBase = "$root/api/v1"
    Write-Host "SMOKE target $script:apiBase" -ForegroundColor Green

    Write-Step 'login'
    $login = Invoke-Api POST '/auth/login' -Body @{ username = 'admin'; password = 'Admin@123'; mode = 'local' } -Label 'login'
    $token = $login.data.accessToken
    if (-not $token) { throw 'login missing accessToken' }
    $h = @{ Authorization = "Bearer $token"; 'X-Client' = 'admin' }

    Write-Step 'me / warehouses'
    $me = Invoke-Api GET '/me' -Headers $h -Label 'me'
    if (-not $me.data) { throw 'me missing data' }
    $wh = Invoke-Api GET '/warehouses' -Headers $h -Label 'warehouses'
    if (-not $wh.data -or @($wh.data).Count -lt 1) { throw 'warehouses empty' }

    Write-Step 'material'
    $matCode = 'M' + ([guid]::NewGuid().ToString('N').Substring(0, 8))
    $mat = Invoke-Api POST '/materials' -Headers $h -Body @{
        code = $matCode; name = 'smoke-material'; unit = 'pcs'; wasteCode = 'HW01'; category = 'smoke'
    } -Label 'material'
    $matId = $mat.data.ID
    if (-not $matId) { throw 'material missing ID' }

    Write-Step 'inbound issue+confirm'
    $in = Invoke-Api POST '/inbound/orders' -Headers $h -Body @{
        warehouseId = 'WH01'; materialId = $matId; qty = 2
    } -Label 'inbound.create'
    $inId = $in.data.ID
    $codes = Invoke-Api POST "/inbound/orders/$inId/issue-codes" -Headers $h -Body @{ count = 2 } -Label 'issue-codes'
    $codeList = @($codes.data)
    if ($codeList.Count -lt 2) { throw "issue-codes count=$($codeList.Count)" }
    $code1 = $codeList[0].Code
    $code2 = $codeList[1].Code
    Invoke-Api POST "/inbound/orders/$inId/confirm" -Headers $h -Body @{
        uniqueCode = $code1; clientRequestId = [guid]::NewGuid().Guid
    } -Label 'inbound.confirm1' | Out-Null
    Invoke-Api POST "/inbound/orders/$inId/confirm" -Headers $h -Body @{
        uniqueCode = $code2; clientRequestId = [guid]::NewGuid().Guid
    } -Label 'inbound.confirm2' | Out-Null

    Write-Step 'stocks after inbound'
    $stocks = Invoke-Api GET '/stocks?warehouseId=WH01' -Headers $h -Label 'stocks'
    $bal = @($stocks.data) | Where-Object { $_.MaterialID -eq $matId } | Select-Object -First 1
    if (-not $bal -or [double]$bal.Qty -lt 2) { throw "stock after inbound qty=$($bal.Qty)" }

    Write-Step 'outbound lock+receive'
    $out = Invoke-Api POST '/outbound/orders' -Headers $h -Body @{
        warehouseId = 'WH01'; materialId = $matId; qty = 1
    } -Label 'outbound.create'
    $outId = $out.data.ID
    Invoke-Api POST "/outbound/orders/$outId/confirm" -Headers $h -Body @{
        action = 'lock'; uniqueCode = $code1; clientRequestId = [guid]::NewGuid().Guid
    } -Label 'outbound.lock' | Out-Null
    Invoke-Api POST "/outbound/orders/$outId/confirm" -Headers $h -Body @{
        action = 'receive'; uniqueCode = $code1; clientRequestId = [guid]::NewGuid().Guid
    } -Label 'outbound.receive' | Out-Null

    Write-Step 'alarm threshold evaluate + ack'
    Invoke-Api PUT '/thresholds/WH01' -Headers $h -Body @{
        metric = 'temperature'; upper = 30; lower = -10
    } -Label 'threshold' | Out-Null
    Invoke-Api POST '/monitor/evaluate' -Headers $h -Body @{
        warehouseId = 'WH01'; deviceId = 'DEV-SMOKE'; metrics = @{ temperature = 99 }
    } -Label 'evaluate' | Out-Null
    $alarms = Invoke-Api GET '/alarms?page=1&pageSize=5' -Headers $h -Label 'alarms'
    $alarmList = @()
    if ($alarms.data.list) { $alarmList = @($alarms.data.list) }
    elseif ($alarms.data) { $alarmList = @($alarms.data) }
    if ($alarmList.Count -lt 1) { throw 'no alarm after threshold breach' }
    $alarmId = $alarmList[0].ID
    Invoke-Api POST "/alarms/$alarmId/ack" -Headers $h -Body @{} -Label 'alarm.ack' | Out-Null

    Write-Step 'geofence evaluate'
    Invoke-Api PUT '/geofences' -Headers $h -Body @{
        warehouseId = 'WH01'
        name = 'smoke-circle'
        shapeType = 'circle'
        circle = @{ lat = 31.2; lng = 121.5; radiusM = 100 }
        hysteresisM = 5
        enabled = $true
    } -Label 'geofence.upsert' | Out-Null
    $geoAlarms = Invoke-Api POST '/geofences/evaluate' -Headers $h -Body @{
        warehouseId = 'WH01'; deviceId = 'DEV-SMOKE-GPS'; lat = 32.0; lng = 122.0
    } -Label 'geofence.eval'
    if ($null -eq $geoAlarms.data) { throw 'geofence evaluate empty' }

    Write-Step 'approval stock adjust'
    $adj = Invoke-Api POST '/stocks/adjustments' -Headers $h -Body @{
        warehouseId = 'WH01'; materialId = $matId; delta = 1; reason = 'smoke-adjust'
    } -Label 'adjust.create'
    $orderId = $adj.data.approvalOrder.ID
    if (-not $orderId) { throw 'adjust missing approvalOrder' }
    $todo = Invoke-Api GET '/approvals/todo?page=1&pageSize=20' -Headers $h -Label 'todo'
    Invoke-Api POST "/approvals/$orderId/act" -Headers $h -Body @{ action = 'pass'; comment = 'L1' } -Label 'approve.L1' | Out-Null
    Invoke-Api POST "/approvals/$orderId/act" -Headers $h -Body @{ action = 'pass'; comment = 'L2' } -Label 'approve.L2' | Out-Null

    Write-Step 'transport / inventory / inspection list'
    Invoke-Api GET '/transport/units' -Headers $h -Label 'transport.units' | Out-Null
    Invoke-Api GET '/transport/tasks?page=1&pageSize=5' -Headers $h -Label 'transport.tasks' | Out-Null
    Invoke-Api GET '/inventory/tasks?page=1&pageSize=5' -Headers $h -Label 'inventory.tasks' | Out-Null
    Invoke-Api GET '/inspection/plans' -Headers $h -Label 'inspection.plans' | Out-Null
    Invoke-Api GET '/approvals/templates' -Headers $h -Label 'approval.templates' | Out-Null
    Invoke-Api GET ("/unique-codes/" + [uri]::EscapeDataString($code2)) -Headers $h -Label 'unique-code' | Out-Null

    Write-Host "SMOKE OK code1=$code1 code2=$code2 alarm=$alarmId todoTotal=$($todo.data.total) mat=$matCode" -ForegroundColor Green
    exit 0
}
catch {
    Write-Host "SMOKE FAIL: $($_.Exception.Message)" -ForegroundColor Red
    if (Test-Path $webhttpLog) {
        Write-Host '--- webhttp.log (tail) ---' -ForegroundColor Yellow
        Get-Content $webhttpLog -Tail 40 -ErrorAction SilentlyContinue
    }
    $errLog = Join-Path $smokeDir 'webhttp.err.log'
    if (Test-Path $errLog) {
        Write-Host '--- webhttp.err.log (tail) ---' -ForegroundColor Yellow
        Get-Content $errLog -Tail 40 -ErrorAction SilentlyContinue
    }
    exit 1
}
finally {
    Stop-SmokeProcesses
}