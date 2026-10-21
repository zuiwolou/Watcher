# 全栈 MVP v0.2 说明

| 版本 | 0.2.0-fullstack-mvp |
| --- | --- |
| 日期 | 2026-10-18 |
| 范围 | 后端 A1–A3 + 部署骨架 A4 + Vue Admin/Portal + Flutter APP 壳与核心页 + 冒烟脚本 |

## 已交付

### 后端服务

| 进程 | 端口 | 说明 |
| --- | --- | --- |
| `webhttp` | 8080 | REST + JWT/Casbin + 业务 + `/v1/stream` 反代 |
| `history` | 8082 | 历史写入/查询；可订 NSQ |
| `push` | 8083 | WSS Protobuf 下行；订 NSQ |
| `mqtt` | — | EMQX 订阅 → Canonical → NSQ/history |

业务：仓储、运输、盘点、审批、巡检、告警、围栏、LDAP、MinIO。

### 前端

| 应用 | 命令 | 端口 |
| --- | --- | --- |
| Admin | `cd frontend/admin && npm i && npm run dev` | 5173 |
| Portal | `cd frontend/portal && npm i && npm run dev` | 5174 |

### APP

`mobile/` Flutter：登录、告警确认、入库确认（手输码）。

```bash
cd mobile
flutter pub get
flutter run --dart-define=API_BASE=http://10.0.2.2:8080/api/v1
```

### 验证

```powershell
# 先确保 MySQL 账号与 backend/configs/config.yaml 一致，再启动 webhttp
powershell -File tools/scripts/smoke.ps1
```

本机若无 Docker，需自备 MySQL 并创建 `watcher` 库/用户（见开发上手）。

前端生产构建：`frontend/admin`、`frontend/portal` 均可 `npm run build`。  
Flutter：`flutter analyze` 通过（需镜像时可设 `PUB_HOSTED_URL=https://pub.flutter-io.cn`）。

## 启动顺序

1. `docker compose -f deploy/compose/docker-compose.yml up -d`  
2. `make run-webhttp` / `run-history` / `run-push` / `run-mqtt`（分终端）  
3. 前端 `npm run dev`  
4. （可选）Flutter  

默认账号：`admin` / `Admin@123`

## 已知简化（可接受的 MVP 缺口）

- biz 仍与 webhttp 同进程（目录在 `backend/internal/biz`，无独立 cmd）  
- 围栏仅圆形；推送厂商通道未接真证书  
- Portal 地图/曲线为接口预留，页面以列表为主  
- APP 扫码用手动输入代替相机插件  
- 监管 Outbox 未接真实省平台  

## 下一迭代建议

真机联调、百度地图嵌入、相机扫码、gRPC 拆 biz、Windows 安装包打磨。
