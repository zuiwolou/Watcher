# Watcher APP（Flutter）

智能危废管控移动端：展示 Tab + 现场执行（扫码/拍照/定位）+ 离线 Outbox + 推送骨架。

## 运行

```bash
cd mobile
flutter pub get
flutter run --dart-define=API_BASE=http://10.0.2.2:8080/api/v1
```

真机请把 `API_BASE` 换成可达的后端地址。

## Tab（按权限动态）

| Tab | 权限 |
| --- | --- |
| 首页 | 登录后默认 |
| 告警 | `alarm:view` |
| 监测 | `monitor:view` |
| 执行 | 任一 `inbound:confirm` / `outbound:confirm` / `checkin:exec` / `inventory:scan` / `inspection:exec` |
| 我的 | 登录后默认（含 Outbox 队列与推送 Token） |

## 能力说明

- **相机扫码**：`mobile_scanner` 全屏取景，可手输兜底
- **拍照**：`image_picker` 现场留存路径（MinIO 上传可后续接）
- **定位**：`geolocator`；签收/运输/巡检可强制要求定位
- **离线 Outbox**：提交失败入队，连通后自动重放；幂等靠 `clientRequestId`
- **推送**：本地通知通道 + 设备 Token 占位；前台轮询未确认告警作摘要；厂商 FCM/APNs 可调用 `PushService.registerVendorToken` / `onRemoteMessage`

## 权限清单（Android）

`CAMERA` · `ACCESS_FINE/COARSE_LOCATION` · `POST_NOTIFICATIONS` · `INTERNET`
