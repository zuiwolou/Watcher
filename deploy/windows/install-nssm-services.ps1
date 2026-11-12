# 使用 NSSM 将 Watcher 四服务注册为 Windows 服务
# 前置：已解压发布包；已安装 NSSM（https://nssm.cc/）；以管理员运行
#
# 用法（在发布包根目录）:
#   powershell -File deploy\windows\install-nssm-services.ps1
#   powershell -File deploy\windows\install-nssm-services.ps1 -Root D:\watcher -Nssm C:\tools\nssm.exe
param(
    [string]$Root = '',
    [string]$Nssm = 'nssm',
    [string]$InternalToken = '',
    [string]$WsOrigins = ''
)

$ErrorActionPreference = 'Stop'

if (-not $Root) {
    # 脚本在仓库 deploy/windows 时指到仓库根；在发布包内可把脚本拷到包根后改 -Root
    $here = $PSScriptRoot
    if (Test-Path (Join-Path $here '..\..\bin')) {
        $Root = Resolve-Path (Join-Path $here '..\..')
    } elseif (Test-Path (Join-Path (Get-Location) 'bin')) {
        $Root = (Get-Location).Path
    } else {
        throw '请用 -Root 指定发布包根目录（含 bin\ configs\）'
    }
}

$nssmCmd = Get-Command $Nssm -ErrorAction SilentlyContinue
if (-not $nssmCmd) {
    throw "找不到 nssm。请安装后把路径传给 -Nssm，或加入 PATH。"
}
$Nssm = $nssmCmd.Source

if (-not $InternalToken) {
    $InternalToken = if ($env:WATCHER_INTERNAL_TOKEN) { $env:WATCHER_INTERNAL_TOKEN } else { 'change-me-internal-token' }
}
if (-not $WsOrigins) {
    $WsOrigins = if ($env:WATCHER_WS_ORIGINS) { $env:WATCHER_WS_ORIGINS } else { 'https://localhost,https://127.0.0.1' }
}

$services = @(
    @{ Name = 'WatcherWebHTTP';  Exe = 'webhttp.exe' },
    @{ Name = 'WatcherHistory';  Exe = 'history.exe' },
    @{ Name = 'WatcherPush';     Exe = 'push.exe' },
    @{ Name = 'WatcherMQTT';     Exe = 'mqtt.exe' }
)

foreach ($s in $services) {
    $exe = Join-Path $Root "bin\$($s.Exe)"
    if (-not (Test-Path $exe)) { throw "missing $exe" }

    Write-Host "install $($s.Name) -> $exe"
    & $Nssm stop $s.Name 2>$null | Out-Null
    & $Nssm remove $s.Name confirm 2>$null | Out-Null
    & $Nssm install $s.Name $exe
    & $Nssm set $s.Name AppDirectory $Root
    & $Nssm set $s.Name AppEnvironmentExtra `
        "WATCHER_ENV=prod" `
        "WATCHER_INTERNAL_TOKEN=$InternalToken" `
        "WATCHER_WS_ORIGINS=$WsOrigins"
    & $Nssm set $s.Name Start SERVICE_AUTO_START
    & $Nssm set $s.Name AppStdout (Join-Path $Root "logs\$($s.Name).out.log")
    & $Nssm set $s.Name AppStderr (Join-Path $Root "logs\$($s.Name).err.log")
    & $Nssm set $s.Name AppRotateFiles 1
}

New-Item -ItemType Directory -Force -Path (Join-Path $Root 'logs') | Out-Null

Write-Host ''
Write-Host '已注册服务。启动示例：'
Write-Host '  nssm start WatcherWebHTTP'
Write-Host '  nssm start WatcherHistory'
Write-Host '  nssm start WatcherPush'
Write-Host '  nssm start WatcherMQTT'
Write-Host "WATCHER_INTERNAL_TOKEN / WATCHER_WS_ORIGINS 已写入服务环境（Origin 示例: $WsOrigins）"
