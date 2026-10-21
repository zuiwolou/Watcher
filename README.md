# 智能危废管控系统 — 全栈 MVP 0.2

按职责划分的 monorepo。目录索引：[docs/guides/目录说明.md](docs/guides/目录说明.md)

详情：[MVP说明](docs/guides/MVP说明.md) · [实施计划](docs/plan/实施计划.md) · [开发上手](docs/guides/开发上手.md)

## 快速启动

```bash
# 1) 中间件
docker compose -f deploy/compose/docker-compose.yml up -d

# 2) 后端（多终端）
go run ./backend/cmd/webhttp    # :8080
go run ./backend/cmd/history    # :8082
go run ./backend/cmd/push       # :8083
go run ./backend/cmd/mqtt       # EMQX

# 3) 前端
cd frontend/admin && npm i && npm run dev    # :5173
cd frontend/portal && npm i && npm run dev   # :5174

# 4) 冒烟
powershell -File tools/scripts/smoke.ps1

# 5) 一键打包（默认 Windows amd64）
.\pack.ps1
```

账号：`admin` / `Admin@123`

## 文档

- [系统设计 V5.1](docs/design/智能危废管控系统设计文档.md)
- [需求-后端](docs/requirements/需求-后端.md) · [需求-前端](docs/requirements/需求-前端.md) · [需求-APP](docs/requirements/需求-APP.md)
- [OpenAPI](contracts/openapi/openapi.yaml)
- [默认 SQL](sql/README.md)（`watcher_schema.sql` · `watcher_data.sql`）
