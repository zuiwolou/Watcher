# Nginx 反代

| 文件 | 用途 |
| --- | --- |
| [`watcher.conf`](watcher.conf) | HTTP 开发/内网样例 |
| [`watcher-tls.conf`](watcher-tls.conf) | **生产 HTTPS**（443 + HSTS） |

要点：

1. 静态托管打包产物 `web/admin`、`web/portal`
2. `/api/` → webhttp `:8080`
3. `/v1/stream` 开启 WebSocket 升级头，反代到 webhttp（内部再转到 push）
4. TLS：把 `ssl_certificate` / `ssl_certificate_key` 换成真实证书（可与 `configs/external_credentials.yaml` 的 `tls.*` 路径一致）
5. 后端设置 `WATCHER_WS_ORIGINS=https://你的域名`（生产必填；见 `configs/runtime.env.example`）
6. 全栈统一 `WATCHER_INTERNAL_TOKEN`（mqtt/history/webhttp 内部调用）
