# 默认 SQL（Watcher）

| 文件 | 功能 |
| --- | --- |
| [`schema/watcher_schema.sql`](schema/watcher_schema.sql) | **表结构**（全量 DDL） |
| [`data/watcher_data.sql`](data/watcher_data.sql) | **表数据**（角色、权限、管理员、默认仓等） |

## 执行

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS watcher DEFAULT CHARACTER SET utf8mb4;"
mysql -u watcher -p watcher < sql/schema/watcher_schema.sql
mysql -u watcher -p watcher < sql/data/watcher_data.sql
```

Windows：

```powershell
mysql -u watcher -pwatcher watcher -e "source $PWD/sql/schema/watcher_schema.sql"
mysql -u watcher -pwatcher watcher -e "source $PWD/sql/data/watcher_data.sql"
```

## 默认账号

| 项 | 值 |
| --- | --- |
| 用户名 | `admin` |
| 密码 | `Admin@123` |
| 角色 | `admin`（全权限）；另有 `readonly` |
| 仓范围 | `WH01` |
| 客户端 | portal / admin / app |

## 说明

- 与当前 GORM 模型对齐；空库初始化优先跑这两份 SQL。
- 应用启动仍可能 `AutoMigrate` + Seed（幂等）。
