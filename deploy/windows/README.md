# Windows 部署说明

## 一键打包

```powershell
.\pack.ps1
```

产物：`dist/watcher-windows-amd64/`（含 `bin\`、`configs\`、`sql\`、`web\`、`start-all.ps1`）。

## 初始化数据库

```powershell
mysql -u watcher -p watcher -e "source $PWD\sql\schema\watcher_schema.sql"
mysql -u watcher -p watcher -e "source $PWD\sql\data\watcher_data.sql"
```

## 运行

1. 编辑 `configs\config.yaml`（DSN、JWT、MinIO）。
2. 中间件：Docker Desktop Compose 或本机安装。
3. `powershell -File .\start-all.ps1`
4. IIS / Nginx 托管 `web\admin`、`web\portal`，反代 `/api` 与 `/v1/stream`。
5. 注册 Windows 服务（需 [NSSM](https://nssm.cc/)，管理员 PowerShell）：

```powershell
powershell -File deploy\windows\install-nssm-services.ps1 -Root <发布包根目录> `
  -InternalToken '你的内部令牌' `
  -WsOrigins 'https://你的域名'
```

6. 环境变量模板：仓库 `configs/runtime.env.example`。

默认账号：`admin` / `Admin@123`（首次登录后请改密）。
