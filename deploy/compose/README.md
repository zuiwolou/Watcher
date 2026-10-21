# Compose 依赖

```bash
docker compose -f deploy/compose/docker-compose.yml up -d
```

中间件：MySQL、Redis、NSQ、etcd、MinIO、EMQX。

应用进程（本机，需先起 MySQL）：

```bash
make run-webhttp   # :8080
make run-history   # :8082
make run-push      # :8083
make run-mqtt      # 连接 EMQX
```

Windows：同样可用 Compose（Docker Desktop）或分别安装中间件后运行上述二进制。
