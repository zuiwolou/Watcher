# Nginx 反代

样例配置见同目录 [`watcher.conf`](watcher.conf)。

要点：

1. 静态托管打包产物 `web/admin`、`web/portal`
2. `/api/` → webhttp `:8080`
3. `/v1/stream` 开启 WebSocket 升级头，反代到 webhttp（内部再转到 push）

生产请配合 TLS（443）与 `WATCHER_WS_ORIGINS` 白名单。
