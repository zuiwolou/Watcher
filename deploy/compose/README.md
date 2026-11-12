# Compose 依赖与应用

## 仅中间件

```bash
docker compose -f deploy/compose/docker-compose.yml up -d
```

中间件：MySQL、Redis、NSQ、etcd、MinIO、EMQX。

## 中间件 + 应用容器（E5.2）

```bash
export WATCHER_INTERNAL_TOKEN=change-me
export WATCHER_WS_ORIGINS=https://localhost
docker compose -f deploy/compose/docker-compose.yml -f deploy/compose/docker-compose.apps.yml up -d --build
```

镜像定义见 [`deploy/docker/Dockerfile`](../docker/Dockerfile)（`SERVICE=webhttp|history|push|mqtt`）。

## 本机二进制（不跑应用容器）

```bash
make run-webhttp   # :8080
make run-history   # :8082
make run-push      # :8083
make run-mqtt      # 连接 EMQX
```

## 规范目录（Linux 裸机 / 发布包）

```
/opt/watcher/
  bin/          webhttp history push mqtt
  configs/      config.yaml
  web/admin     web/portal
  sql/
```

配合 [`deploy/systemd/*.service`](../systemd/) 或 Windows [`install-nssm-services.ps1`](../windows/install-nssm-services.ps1)。
