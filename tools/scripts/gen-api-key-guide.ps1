# 根据当前工程信息生成「API / 证书申请说明」
# 用法:
#   powershell -File tools/scripts/gen-api-key-guide.ps1
#   powershell -File tools/scripts/gen-api-key-guide.ps1 -OutDir dist\watcher-windows-amd64\docs
param(
    [string]$OutDir = '',
    [string]$AdminHost = '',
    [string]$PortalHost = ''
)

$ErrorActionPreference = 'Stop'
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..\..')
Set-Location $RepoRoot

$stamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
$genDir = Join-Path $RepoRoot 'docs\generated'
New-Item -ItemType Directory -Force -Path $genDir | Out-Null

# --- 读取工程标识 ---
$androidAppId = 'com.example.watcher_app'
$androidGradle = Join-Path $RepoRoot 'mobile\android\app\build.gradle.kts'
if (Test-Path $androidGradle) {
    $m = Select-String -Path $androidGradle -Pattern 'applicationId\s*=\s*"([^"]+)"' | Select-Object -First 1
    if ($m) { $androidAppId = $m.Matches[0].Groups[1].Value }
}

$iosBundleId = 'com.example.watcherApp'
$iosPbx = Join-Path $RepoRoot 'mobile\ios\Runner.xcodeproj\project.pbxproj'
if (Test-Path $iosPbx) {
    $m = Select-String -Path $iosPbx -Pattern 'PRODUCT_BUNDLE_IDENTIFIER = ([^;]+);' |
        Where-Object { $_.Line -notmatch 'RunnerTests' } | Select-Object -First 1
    if ($m) { $iosBundleId = $m.Matches[0].Groups[1].Value.Trim() }
}

$adminPort = 5173
$portalPort = 5174
$adminVite = Join-Path $RepoRoot 'frontend\admin\vite.config.js'
$portalVite = Join-Path $RepoRoot 'frontend\portal\vite.config.js'
if (Test-Path $adminVite) {
    $m = Select-String -Path $adminVite -Pattern 'port:\s*(\d+)' | Select-Object -First 1
    if ($m) { $adminPort = [int]$m.Matches[0].Groups[1].Value }
}
if (Test-Path $portalVite) {
    $m = Select-String -Path $portalVite -Pattern 'port:\s*(\d+)' | Select-Object -First 1
    if ($m) { $portalPort = [int]$m.Matches[0].Groups[1].Value }
}

# --- 本机信息 ---
$localHost = 'localhost'
$lanIps = @()
try {
    $lanIps = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
        Where-Object { $_.IPAddress -notlike '127.*' -and $_.PrefixOrigin -ne 'WellKnown' } |
        Select-Object -ExpandProperty IPAddress -Unique
} catch {
    try {
        $lanIps = [System.Net.Dns]::GetHostAddresses([System.Net.Dns]::GetHostName()) |
            Where-Object { $_.AddressFamily -eq 'InterNetwork' -and $_.IPAddressToString -notlike '127.*' } |
            ForEach-Object { $_.IPAddressToString }
    } catch { $lanIps = @() }
}

# --- Debug SHA1 ---
$debugSha1 = '(未取到：请安装 JDK 后手动 keytool)'
$debugKeystore = Join-Path $env:USERPROFILE '.android\debug.keystore'
function Find-Keytool {
    $cmd = Get-Command keytool -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $cands = @(
        (Join-Path $env:JAVA_HOME 'bin\keytool.exe'),
        "$env:LOCALAPPDATA\Programs\Android\Android Studio\jbr\bin\keytool.exe",
        'C:\Program Files\Android\Android Studio\jbr\bin\keytool.exe'
    )
    $cands += @(Get-ChildItem 'C:\Program Files\Java\*\bin\keytool.exe' -ErrorAction SilentlyContinue | ForEach-Object FullName)
    $cands += @(Get-ChildItem 'C:\Program Files\Eclipse Adoptium\*\bin\keytool.exe' -ErrorAction SilentlyContinue | ForEach-Object FullName)
    foreach ($p in $cands) {
        if ($p -and (Test-Path $p)) { return $p }
    }
    return $null
}
$keytoolExe = Find-Keytool
$debugSha1Cmd = if ($keytoolExe) {
    "& `"$keytoolExe`" -list -v -keystore `"$debugKeystore`" -alias androiddebugkey -storepass android -keypass android"
} else {
    'keytool -list -v -keystore "%USERPROFILE%\.android\debug.keystore" -alias androiddebugkey -storepass android -keypass android'
}
if (Test-Path $debugKeystore) {
    if ($keytoolExe) {
        try {
            $out = & $keytoolExe -list -v -keystore $debugKeystore -alias androiddebugkey -storepass android -keypass android 2>&1 | Out-String
            if ($out -match 'SHA1:\s*([0-9A-Fa-f:]+)') {
                $debugSha1 = $Matches[1]
            } elseif ($out -match 'SHA1:\s*(.+)') {
                $debugSha1 = $Matches[1].Trim()
            }
        } catch {
            $debugSha1 = "(keytool 执行失败: $($_.Exception.Message))"
        }
    } else {
        $debugSha1 = '(未找到 keytool，请安装 JDK/Android Studio JBR 后重跑本脚本)'
    }
} else {
    $debugSha1 = '(尚无 debug.keystore：先执行一次 flutter run 生成后再跑本脚本)'
}

# --- 可选：读取用户已填主机 ---
$credFile = Join-Path $RepoRoot 'configs\external_credentials.yaml'
if ((-not $AdminHost -or -not $PortalHost) -and (Test-Path $credFile)) {
    $raw = Get-Content $credFile -Raw -ErrorAction SilentlyContinue
    if ($raw) {
        if (-not $AdminHost -and $raw -match 'admin_public_host:\s*"?([^"\r\n#]+)"?') {
            $AdminHost = $Matches[1].Trim().Trim('"')
        }
        if (-not $PortalHost -and $raw -match 'portal_public_host:\s*"?([^"\r\n#]+)"?') {
            $PortalHost = $Matches[1].Trim().Trim('"')
        }
    }
}
if (-not $AdminHost) { $AdminHost = $localHost }
if (-not $PortalHost) { $PortalHost = $localHost }

$whitelistLines = @($localHost)
foreach ($ip in $lanIps) {
    if ($ip -and ($whitelistLines -notcontains $ip)) { $whitelistLines += $ip }
}
if ($AdminHost -and ($whitelistLines -notcontains $AdminHost)) { $whitelistLines += $AdminHost }
if ($PortalHost -and ($whitelistLines -notcontains $PortalHost)) { $whitelistLines += $PortalHost }

$wlMarkdown = ($whitelistLines | ForEach-Object { "- ``$_``" }) -join "`n"
$lanMarkdown = if ($lanIps.Count -gt 0) {
    ($lanIps | ForEach-Object { "- ``$_``" }) -join "`n"
} else { '- （未检测到局域网 IPv4）' }

$fillStatus = @()
function Test-Filled([string]$label, [string]$pathPattern) {
    $has = $false
    if (Test-Path $credFile) {
        $t = Get-Content $credFile -Raw
        if ($t -match $pathPattern) {
            $val = $Matches[1].Trim().Trim('"')
            if ($val -ne '' -and $val -ne 'false') { $has = $true }
        }
    }
    if ($has) { $script:fillStatus += "| $label | 已填 |" } else { $script:fillStatus += "| $label | **待填** |" }
}
Test-Filled '百度 JavaScript API AK' 'javascript_api_ak:\s*"?([^"\r\n#]*)"?'
Test-Filled '百度 Android SDK AK' 'android_sdk_ak:\s*"?([^"\r\n#]*)"?'
Test-Filled '百度 iOS SDK AK' 'ios_sdk_ak:\s*"?([^"\r\n#]*)"?'
Test-Filled 'APNs Key ID' '(?m)^\s*key_id:\s*"?([^"\r\n#]*)"?'
Test-Filled 'FCM Project ID' '(?m)^\s*project_id:\s*"?([^"\r\n#]*)"?'
$fillTable = $fillStatus -join "`n"

$doc = @"
# Watcher API / 证书申请说明（自动生成）

| 项 | 值 |
| --- | --- |
| 生成时间 | $stamp |
| 仓库 | ``$RepoRoot`` |
| 用途 | 按下列**本机/工程信息**去开放平台申请 AK / 证书，再填回配置 |

> 详细步骤见 [`docs/guides/证书与密钥申请指南.md`](../guides/证书与密钥申请指南.md)。  
> 配置模板：[`configs/external_credentials.example.yaml`](../../configs/external_credentials.example.yaml) → 复制为 ``external_credentials.yaml`` 填写。

---

## 0. 当前填写状态

$fillTable

（无 ``configs/external_credentials.yaml`` 时全部显示待填，属正常。）

---

## 1. 本地开发 — 百度地图 JS AK（优先做这个）

### 1.1 控制台填写

| 字段 | 填写内容 |
| --- | --- |
| 应用类型 | 浏览器端（JavaScript API） |
| 应用名称 | 建议 ``watcher-web-local`` |
| **域名白名单** | 见下方列表（**不要**填 admin.example.com） |

**建议白名单（复制到百度控制台）：**

$wlMarkdown

### 1.2 本地访问地址（白名单只填主机名，下面仅供对照）

| 端 | 本机 URL | 白名单主机名 |
| --- | --- | --- |
| Admin | ``http://localhost:$adminPort`` | ``localhost`` |
| Portal | ``http://localhost:$portalPort`` | ``localhost`` |

本机局域网 IP（真机/同事访问你电脑时，白名单可追加）：

$lanMarkdown

### 1.3 申请后填到哪里

1. ``configs/external_credentials.yaml`` → ``baidu_map.javascript_api_ak``（本地 Admin/Portal 可共用一把）
2. 复制 ``frontend/admin/.env.example`` → ``.env``，设置：
   - ``VITE_BAIDU_JAVASCRIPT_API_AK=<JavaScript_API_的AK>``
   - ``VITE_MAP_PROVIDER=baidu``（未接 SDK 前可仍用 ``leaflet``）
3. Portal 同样：``frontend/portal/.env``

未填 AK 时系统继续用 Leaflet/OSM，**不影响其它功能开发**。

---

## 2. APP — 百度 Android SDK / iOS SDK 的 AK

### 2.1 Android SDK → AK

| 字段 | 本工程当前值 |
| --- | --- |
| 应用类型 | **Android SDK**（控制台选项名） |
| 申请到的密钥名 | **AK** |
| **包名（applicationId）** | ``$androidAppId`` |
| **Debug 签名 SHA1** | ``$debugSha1`` |
| Release SHA1 | 使用正式 ``.jks`` 时再 ``keytool -list -v`` 获取 |

手动取 SHA1 命令：

``````
$debugSha1Cmd
``````

申请后填：``baidu_map.android_sdk_ak``，运行：

``````
flutter run --dart-define=BAIDU_ANDROID_SDK_AK=<Android_SDK_的AK> --dart-define=MAP_PROVIDER=baidu
``````

### 2.2 iOS SDK → AK

| 字段 | 本工程当前值 |
| --- | --- |
| 应用类型 | **iOS SDK** |
| 申请到的密钥名 | **AK** |
| **Bundle ID** | ``$iosBundleId`` |

申请后填：``baidu_map.ios_sdk_ak``；运行加 ``--dart-define=BAIDU_IOS_SDK_AK=...``。上架前请把 Bundle ID 改成公司域名风格并重新申请。

---

## 3. Apple APNs（控制台密钥名）

| 申请到的材料 | 配置字段 |
| --- | --- |
| **Key ID** | ``apns.key_id`` |
| **Team ID** | ``apns.team_id`` |
| **AuthKey_XXXXX.p8** | ``apns.auth_key_path``（路径，勿入库） |
| Bundle ID | ``apns.bundle_id`` = ``$iosBundleId`` |

Identifiers 中须开启 Push Notifications。``.p8`` 只放本机/服务器，**不要提交 Git**。

---

## 4. Firebase Cloud Messaging（FCM）

| 申请到的材料 | 配置字段 |
| --- | --- |
| **项目 ID** (Project ID) | ``fcm.project_id`` |
| **服务账号 JSON** | ``fcm.service_account_json_path`` |
| **google-services.json** | ``fcm.google_services_json_path`` → 拷到 ``mobile/android/app/`` |
| Android 包名 | ``$androidAppId`` |

---

## 5. 配置文件一览（填空用）

| 文件 | 作用 |
| --- | --- |
| ``configs/external_credentials.example.yaml`` | 总模板（可提交） |
| ``configs/external_credentials.yaml`` | 你的实填文件（勿提交） |
| ``frontend/admin/.env.example`` → ``.env`` | Admin 地图 AK |
| ``frontend/portal/.env.example`` → ``.env`` | Portal 地图 AK |
| ``mobile/config/credentials.example.json`` | APP 说明用；真密钥优先 dart-define |
| ``backend/configs/config.example.yaml`` 中 ``maps`` / ``push`` / ``tls`` | 后端占位 |

重新生成本说明：

``````powershell
powershell -File tools/scripts/gen-api-key-guide.ps1
``````

打包时会自动调用本脚本，输出到发布包 ``docs/API密钥申请说明.md``。

---

## 6. 推荐办理顺序（本地开发）

1. 百度 **浏览器端** AK，白名单 ``localhost``（+ 可选局域网 IP）
2. （可选）Android debug SHA1 + 包名 申请百度 Android AK
3. （可选）iOS Bundle ID 申请百度 iOS AK
4. 需要真机系统通知时再办 APNs / FCM
5. 有正式域名后再把白名单改成生产域名，或另建生产 AK
"@

$outMain = Join-Path $genDir 'API密钥申请说明.md'
Set-Content -Path $outMain -Value $doc -Encoding UTF8
Write-Host "wrote: $outMain"

if ($OutDir) {
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $outPack = Join-Path $OutDir 'API密钥申请说明.md'
    Set-Content -Path $outPack -Value $doc -Encoding UTF8
    Write-Host "wrote: $outPack"
}

# 同时写一份纯文本「白名单速查」方便复制
$quick = @"
# 百度 JS 域名白名单（本机生成 $stamp）
$($whitelistLines -join "`n")

# Android
applicationId=$androidAppId
debugSHA1=$debugSha1

# iOS
bundleId=$iosBundleId
"@
$quickPath = Join-Path $genDir '申请速查.txt'
Set-Content -Path $quickPath -Value $quick -Encoding UTF8
Write-Host "wrote: $quickPath"
if ($OutDir) {
    Copy-Item $quickPath (Join-Path $OutDir '申请速查.txt') -Force
}

Write-Host ''
Write-Host 'GEN API KEY GUIDE OK'
