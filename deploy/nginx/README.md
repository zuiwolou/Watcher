# Nginx 反代（占位）

生产建议：

- 静态：`frontend/admin/dist`、`frontend/portal/dist`
- 反代 `/api/` → `webhttp:8080`
- 反代 `/api/v1/stream` → WSS（经 webhttp 鉴权后到 push）

配置样例后续补齐。
