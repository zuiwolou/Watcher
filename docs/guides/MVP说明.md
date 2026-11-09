# 全栈 MVP v0.3 说明

| 版本 | 0.3.0-fullstack-mvp |
| --- | --- |
| 日期 | 2026-11-09 |
| Git tag | `v0.3.0-mvp` |
| OpenAPI | `contracts/openapi/openapi.yaml` **0.3.0**（与 webhttp 路由对齐） |
| 缺口跟踪 | [`缺口评估与迭代计划.md`](../plan/缺口评估与迭代计划.md) |
| 范围 | 后端闭环（Outbox/Device/审计/RequireClient/Schema 热更新）+ Admin/Portal/APP 核心能力 + 单测/SQLite 集成测 + CI |

## 相对 0.2 的增量

| 域 | 交付 |
| --- | --- |
| 后端 | IntegrationOutbox Worker（HTTP/SFTP drop）；Device CRUD；操作审计；`RequireAnyClient(admin\|portal\|app)`；mqtt Schema Profile + NSQ `config.mqtt.updated` 热加载（30s 轮询兜底） |
| Admin | 角色权限矩阵；集成配置/Outbox；围栏地图（Leaflet）；审批模板与进度 |
| Portal | WSS Protobuf 解码刷新；监测曲线/地图；Refresh Token；强制改密 |
| APP | 相机扫码/定位；离线 Outbox；展示 Tab；本地推送 + 轮询；厂商 token 钩子 |
| 质量 | JWT/中间件/围栏/审批单测；入库→出库→签收/审批/幂等 SQLite 集成测；GitHub Actions（Go test+build、Admin/Portal build、Flutter analyze 可选） |
| 冒烟 | `tools/scripts/smoke.ps1 -AutoStart`（临时 MySQL:13306 + webhttp:18080） |

## 已交付

### 后端服务

| 进程 | 端口 | 说明 |
| --- | --- | --- |
| `webhttp` | 8080 | REST + JWT/Casbin + 业务同进程 + `/v1/stream` 反代 |
| `history` | 8082 | 历史写入/查询；可订 NSQ |
| `push` | 8083 | WSS Protobuf 下行；订 NSQ |
| `mqtt` | — | EMQX 订阅 → Canonical → NSQ/history；Schema 热更新 |

业务：仓储主链路、运输、盘点、多级审批、巡检、告警、围栏（圆/多边形）、LDAP、MinIO、监管 Outbox、设备台账。

### 前端

| 应用 | 命令 | 端口 |
| --- | --- | --- |
| Admin | `cd frontend/admin && npm i && npm run dev` | 5173 |
| Portal | `cd frontend/portal && npm i && npm run dev` | 5174 |

### APP

`mobile/` Flutter：登录、告警、入库/出库/签收（扫码+离线 Outbox）、展示与本地推送。

```bash
cd mobile
flutter pub get
flutter run --dart-define=API_BASE=http://10.0.2.2:8080/api/v1
```

### 验证

```powershell
# 单元 + SQLite 集成（与 CI 一致）
go test ./... -count=1 -timeout 120s
# 或 make test

# 已有 webhttp 时直接跑；否则加 -AutoStart（临时 MySQL:13306 + webhttp:18080）
powershell -File tools/scripts/smoke.ps1 -AutoStart
```

CI：`.github/workflows/ci.yml`（Go tidy/test/build；Admin/Portal `npm run build`；Flutter analyze 非阻断）。

前端生产构建：`frontend/admin`、`frontend/portal` 均可 `npm run build`。  
Flutter：`flutter analyze`（需镜像时可设 `PUB_HOSTED_URL=https://pub.flutter-io.cn`）。

## 启动顺序

1. `docker compose -f deploy/compose/docker-compose.yml up -d`  
2. `make run-webhttp` / `run-history` / `run-push` / `run-mqtt`（分终端）  
3. 前端 `npm run dev`  
4. （可选）Flutter  

默认账号：`admin` / `Admin@123`

## 已知简化（可接受的 MVP 缺口）

- biz 仍与 webhttp 同进程（目录在 `backend/internal/biz`，无独立 cmd/gRPC）  
- 监管 Outbox 为 HTTP/SFTP drop 骨架，未接真实省平台字段映射  
- Admin 围栏以 Leaflet 为主；百度 AK 联调留待下迭代  
- APP 厂商推送通道未接真证书；离线队列为本地 Outbox  
- Schema Admin dry-run / 高级编辑 UI 仍可加深  

## 下一迭代建议

完整缺口清单与分阶段计划见 [`缺口评估与迭代计划.md`](../plan/缺口评估与迭代计划.md)。

摘要：真环境联调与性能冒烟、盘点差异过账、围栏定位告警闭环、监管入站 OpenAPI、APP 连扫/上传/WSS/iOS、百度地图 AK、厂商推送证书、拆 `svc-biz` gRPC、监管字段映射配置化、冻结 tag（如 `v0.3.0-mvp`）。
