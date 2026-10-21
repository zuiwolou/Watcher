# Windows 部署说明

## 一键打包

在仓库根目录：

```powershell
.\pack.ps1
# 或
make pack
```

产物：`dist/watcher-windows-amd64/` 与同名 zip（含 `bin\`、`configs\`、`web\`、`start-all.ps1`）。

可选参数：

```powershell
.\pack.ps1 -SkipFrontend          # 只打后端
.\pack.ps1 -Target linux          # 交叉编 Linux
.\pack.ps1 -Arch arm64 -NoZip
```

## 运行

1. 中间件：Docker Desktop Compose（`deploy/compose`）或各自 Windows 安装包。
2. 解压发布包，编辑 `configs\config.yaml`。
3. `powershell -File .\start-all.ps1`（工作目录为包根，以便读到 `configs\`）。
4. 用 IIS / Nginx 托管 `web\admin`、`web\portal`，反代 `/api` → `http://127.0.0.1:8080`。
5. 生产可用 NSSM 将四个 exe 注册为服务。
