# 一键打包 Watcher 发布目录（默认 Windows amd64）
# 用法:
#   powershell -File tools/scripts/pack.ps1
#   powershell -File tools/scripts/pack.ps1 -Target linux
#   powershell -File tools/scripts/pack.ps1 -SkipFrontend
#   powershell -File tools/scripts/pack.ps1 -Arch arm64
param(
    [ValidateSet('windows', 'linux')]
    [string]$Target = 'windows',

    [ValidateSet('amd64', 'arm64')]
    [string]$Arch = 'amd64',

    [switch]$SkipFrontend,
    [switch]$NoZip,

    [string]$OutRoot = ''
)

$ErrorActionPreference = 'Stop'

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..\..')
Set-Location $RepoRoot

$ext = if ($Target -eq 'windows') { '.exe' } else { '' }
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$pkgName = "watcher-$Target-$Arch"
if (-not $OutRoot) {
    $OutRoot = Join-Path $RepoRoot "dist\$pkgName"
}

Write-Host "== pack $pkgName =="
Write-Host "out: $OutRoot"

if (Test-Path $OutRoot) {
    Remove-Item -Recurse -Force $OutRoot
}
New-Item -ItemType Directory -Force -Path `
    (Join-Path $OutRoot 'bin'), `
    (Join-Path $OutRoot 'configs'), `
    (Join-Path $OutRoot 'web\admin'), `
    (Join-Path $OutRoot 'web\portal'), `
    (Join-Path $OutRoot 'sql'), `
    (Join-Path $OutRoot 'docs') | Out-Null

$env:CGO_ENABLED = '0'
$env:GOOS = $Target
$env:GOARCH = $Arch

$services = @('webhttp', 'history', 'push', 'mqtt')
foreach ($s in $services) {
    $out = Join-Path $OutRoot "bin\$s$ext"
    Write-Host "go build $s -> $out"
    & go build -trimpath -ldflags '-s -w' -o $out "./backend/cmd/$s"
    if ($LASTEXITCODE -ne 0) { throw "build failed: $s" }
}

# 还原本机交叉编译环境变量，避免影响后续 npm/本机 go
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

$cfgSrc = Join-Path $RepoRoot 'backend\configs\config.example.yaml'
if (-not (Test-Path $cfgSrc)) {
    $cfgSrc = Join-Path $RepoRoot 'backend\configs\config.yaml'
}
Copy-Item $cfgSrc (Join-Path $OutRoot 'configs\config.yaml')
Copy-Item -Recurse -Force (Join-Path $RepoRoot 'sql\*') (Join-Path $OutRoot 'sql\')
Copy-Item (Join-Path $RepoRoot 'contracts\openapi\openapi.yaml') (Join-Path $OutRoot 'docs\openapi.yaml') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'docs\guides\开发上手.md') (Join-Path $OutRoot 'docs\') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'docs\guides\证书与密钥申请指南.md') (Join-Path $OutRoot 'docs\') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'deploy\windows\README.md') (Join-Path $OutRoot 'docs\windows-deploy.md') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'configs\external_credentials.example.yaml') (Join-Path $OutRoot 'configs\external_credentials.example.yaml') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'configs\runtime.env.example') (Join-Path $OutRoot 'configs\runtime.env.example') -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $OutRoot 'deploy\nginx'), (Join-Path $OutRoot 'deploy\windows') | Out-Null
Copy-Item (Join-Path $RepoRoot 'deploy\nginx\watcher.conf') (Join-Path $OutRoot 'deploy\nginx\') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'deploy\nginx\watcher-tls.conf') (Join-Path $OutRoot 'deploy\nginx\') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'deploy\windows\install-nssm-services.ps1') (Join-Path $OutRoot 'deploy\windows\') -ErrorAction SilentlyContinue
Copy-Item (Join-Path $RepoRoot 'docs\qa\验收记录模板.md') (Join-Path $OutRoot 'docs\') -ErrorAction SilentlyContinue

# 生成含本机白名单 / 包名 / SHA1 的申请说明，供填写 API Key
Write-Host "gen API key application guide"
& powershell -NoProfile -File (Join-Path $RepoRoot 'tools\scripts\gen-api-key-guide.ps1') -OutDir (Join-Path $OutRoot 'docs')
if ($LASTEXITCODE -ne 0) { Write-Warning "gen-api-key-guide.ps1 failed (non-fatal)" }

if (-not $SkipFrontend) {
    foreach ($app in @('admin', 'portal')) {
        $appDir = Join-Path $RepoRoot "frontend\$app"
        if (-not (Test-Path (Join-Path $appDir 'package.json'))) {
            Write-Warning "skip frontend/${app}: package.json missing"
            continue
        }
        Write-Host "npm build frontend/${app}"
        Push-Location $appDir
        try {
            if (-not (Test-Path 'node_modules')) {
                & npm install --no-fund --no-audit
                if ($LASTEXITCODE -ne 0) { throw "npm install failed: ${app}" }
            }
            & npm run build
            if ($LASTEXITCODE -ne 0) { throw "npm build failed: ${app}" }
        } finally {
            Pop-Location
        }
        $distDir = Join-Path $appDir 'dist'
        Copy-Item -Recurse -Force (Join-Path $distDir '*') (Join-Path $OutRoot "web\$app\")
    }
}

# 发布包内启动脚本（Windows）
if ($Target -eq 'windows') {
    $startAll = @"
# 在包根目录启动全部后端（需已配置 configs\config.yaml 与 MySQL 等）
`$ErrorActionPreference = 'Stop'
`$Root = `$PSScriptRoot
Set-Location `$Root
`$env:WATCHER_ENV = if (`$env:WATCHER_ENV) { `$env:WATCHER_ENV } else { 'prod' }

`$procs = @('webhttp', 'history', 'push', 'mqtt')
foreach (`$p in `$procs) {
    `$exe = Join-Path `$Root "bin\`$p.exe"
    if (-not (Test-Path `$exe)) { throw "missing `$exe" }
    Write-Host "start `$p"
    Start-Process -FilePath `$exe -WorkingDirectory `$Root -WindowStyle Minimized
}
Write-Host 'started. webhttp default :8080'
Write-Host 'static: web\admin  web\portal  (需 Nginx/IIS 托管并反代 /api)'
"@
    Set-Content -Path (Join-Path $OutRoot 'start-all.ps1') -Value $startAll -Encoding UTF8

    $readme = @"
# Watcher 发布包 ($Target-$Arch)

构建时间: $stamp

## 内容

- ``bin\`` — webhttp / history / push / mqtt
- ``configs\config.yaml`` — 请按环境修改后使用
- ``configs\external_credentials.example.yaml`` — 地图/推送等外部凭证模板
- ``docs\API密钥申请说明.md`` — **打包时自动生成**（白名单、包名、SHA1）
- ``docs\申请速查.txt`` — 白名单/SHA1 速查
- ``sql\schema\watcher_schema.sql`` · ``sql\data\watcher_data.sql``
- ``web\admin`` · ``web\portal`` — 前端静态资源
- ``start-all.ps1`` — 一键拉起四个后端进程

## 使用

1. 编辑 ``configs\config.yaml``（MySQL DSN、JWT 等）
2. 阅读 ``docs\API密钥申请说明.md``，按说明申请百度 AK / APNs / FCM，填入凭证配置
3. 空库先执行 ``sql\schema\watcher_schema.sql`` 再执行 ``sql\data\watcher_data.sql``
4. 确保中间件已就绪（可用仓库 ``deploy\compose``）
5. ``powershell -File .\start-all.ps1``
6. 用 Nginx/IIS 托管 ``web\admin``、``web\portal``，反代 ``/api`` → ``http://127.0.0.1:8080``

默认账号：``admin`` / ``Admin@123``
"@
    Set-Content -Path (Join-Path $OutRoot 'README.md') -Value $readme -Encoding UTF8
} else {
    $startSh = @"
#!/usr/bin/env bash
set -euo pipefail
ROOT="`$(cd "`$(dirname "`$0")" && pwd)"
cd "`$ROOT"
export WATCHER_ENV="`${WATCHER_ENV:-prod}"
for p in webhttp history push mqtt; do
  echo "start `$p"
  nohup "`$ROOT/bin/`$p" >/tmp/watcher-`$p.log 2>&1 &
done
echo "started. logs in /tmp/watcher-*.log"
"@
    # LF line endings for shell script
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText((Join-Path $OutRoot 'start-all.sh'), ($startSh -replace "`r`n", "`n"), $utf8NoBom)

    $readme = @"
# Watcher release ($Target-$Arch)

Built: $stamp

- ``bin/`` services
- ``configs/config.yaml``
- ``web/admin`` ``web/portal``
- ``./start-all.sh``

Host static files behind Nginx; proxy ``/api`` to ``:8080``.
"@
    Set-Content -Path (Join-Path $OutRoot 'README.md') -Value $readme -Encoding UTF8
}

$zipPath = $null
if (-not $NoZip) {
    $zipPath = Join-Path $RepoRoot "dist\$pkgName-$stamp.zip"
    if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
    Write-Host "zip -> $zipPath"
    Compress-Archive -Path (Join-Path $OutRoot '*') -DestinationPath $zipPath -Force
}

Write-Host ''
Write-Host "PACK OK"
Write-Host "  dir: $OutRoot"
if ($zipPath) { Write-Host "  zip: $zipPath" }
