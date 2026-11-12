# Watcher API / 证书申请说明（自动生成）

| 项 | 值 |
| --- | --- |
| 生成时间 | 2026-11-11 11:15:03 |
| 仓库 | `E:\Work\Watcher` |
| 用途 | 按下列**本机/工程信息**去开放平台申请 AK / 证书，再填回配置 |

> 详细步骤见 [docs/guides/证书与密钥申请指南.md](../guides/证书与密钥申请指南.md)。  
> 配置模板：[configs/external_credentials.example.yaml](../../configs/external_credentials.example.yaml) → 复制为 `external_credentials.yaml` 填写。

---

## 0. 当前填写状态

| 百度 JavaScript API AK | **待填** |
| 百度 Android SDK AK | **待填** |
| 百度 iOS SDK AK | **待填** |
| APNs Key ID | **待填** |
| FCM Project ID | **待填** |

（无 `configs/external_credentials.yaml` 时全部显示待填，属正常。）

---

## 1. 本地开发 — 百度地图 JS AK（优先做这个）

### 1.1 控制台填写

| 字段 | 填写内容 |
| --- | --- |
| 应用类型 | 浏览器端（JavaScript API） |
| 应用名称 | 建议 `watcher-web-local` |
| **域名白名单** | 见下方列表（**不要**填 admin.example.com） |

**建议白名单（复制到百度控制台）：**

- `localhost`
- `192.168.0.114`
- `192.168.0.100`

### 1.2 本地访问地址（白名单只填主机名，下面仅供对照）

| 端 | 本机 URL | 白名单主机名 |
| --- | --- | --- |
| Admin | `http://localhost:5173` | `localhost` |
| Portal | `http://localhost:5174` | `localhost` |

本机局域网 IP（真机/同事访问你电脑时，白名单可追加）：

- `192.168.0.114`
- `192.168.0.100`

### 1.3 申请后填到哪里

1. `configs/external_credentials.yaml` → `baidu_map.javascript_api_ak`（本地 Admin/Portal 可共用一把）
2. 复制 `frontend/admin/.env.example` → `.env`，设置：
   - `VITE_BAIDU_JAVASCRIPT_API_AK=<JavaScript_API_的AK>`
   - `VITE_MAP_PROVIDER=baidu`（未接 SDK 前可仍用 `leaflet`）
3. Portal 同样：`frontend/portal/.env`

未填 AK 时系统继续用 Leaflet/OSM，**不影响其它功能开发**。

---

## 2. APP — 百度 Android SDK / iOS SDK 的 AK

### 2.1 Android SDK → AK

| 字段 | 本工程当前值 |
| --- | --- |
| 应用类型 | **Android SDK**（控制台选项名） |
| 申请到的密钥名 | **AK** |
| **包名（applicationId）** | `com.example.watcher_app` |
| **Debug 签名 SHA1** | `9B:11:27:33:A7:86:56:9E:DE:F3:62:99:E4:C6:EE:DE:B6:FC:E2:74` |
| Release SHA1 | 使用正式 `.jks` 时再 `keytool -list -v` 获取 |

手动取 SHA1 命令：

```
& "C:\Program Files\Java\jre1.8.0_491\bin\keytool.exe" -list -v -keystore "C:\Users\71426\.android\debug.keystore" -alias androiddebugkey -storepass android -keypass android
```

申请后填：`baidu_map.android_sdk_ak`，运行：

```
flutter run --dart-define=BAIDU_ANDROID_SDK_AK=<Android_SDK_的AK> --dart-define=MAP_PROVIDER=baidu
```

### 2.2 iOS SDK → AK

| 字段 | 本工程当前值 |
| --- | --- |
| 应用类型 | **iOS SDK** |
| 申请到的密钥名 | **AK** |
| **Bundle ID** | `com.example.watcherApp` |

申请后填：`baidu_map.ios_sdk_ak`；运行加 `--dart-define=BAIDU_IOS_SDK_AK=...`。上架前请把 Bundle ID 改成公司域名风格并重新申请。

---

## 3. Apple APNs（控制台密钥名）

| 申请到的材料 | 配置字段 |
| --- | --- |
| **Key ID** | `apns.key_id` |
| **Team ID** | `apns.team_id` |
| **AuthKey_XXXXX.p8** | `apns.auth_key_path`（路径，勿入库） |
| Bundle ID | `apns.bundle_id` = `com.example.watcherApp` |

Identifiers 中须开启 Push Notifications。`.p8` 只放本机/服务器，**不要提交 Git**。

---

## 4. Firebase Cloud Messaging（FCM）

| 申请到的材料 | 配置字段 |
| --- | --- |
| **项目 ID** (Project ID) | `fcm.project_id` |
| **服务账号 JSON** | `fcm.service_account_json_path` |
| **google-services.json** | `fcm.google_services_json_path` → 拷到 `mobile/android/app/` |
| Android 包名 | `com.example.watcher_app` |

---

## 5. 配置文件一览（填空用）

| 文件 | 作用 |
| --- | --- |
| `configs/external_credentials.example.yaml` | 总模板（可提交） |
| `configs/external_credentials.yaml` | 你的实填文件（勿提交） |
| `frontend/admin/.env.example` → `.env` | Admin 地图 AK |
| `frontend/portal/.env.example` → `.env` | Portal 地图 AK |
| `mobile/config/credentials.example.json` | APP 说明用；真密钥优先 dart-define |
| `backend/configs/config.example.yaml` 中 `maps` / `push` / `tls` | 后端占位 |

重新生成本说明：

```powershell
powershell -File tools/scripts/gen-api-key-guide.ps1
```

打包时会自动调用本脚本，输出到发布包 `docs/API密钥申请说明.md`。

---

## 6. 推荐办理顺序（本地开发）

1. 百度 **浏览器端** AK，白名单 `localhost`（+ 可选局域网 IP）
2. （可选）Android debug SHA1 + 包名 申请百度 Android AK
3. （可选）iOS Bundle ID 申请百度 iOS AK
4. 需要真机系统通知时再办 APNs / FCM
5. 有正式域名后再把白名单改成生产域名，或另建生产 AK
