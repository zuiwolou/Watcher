# 智能危废管控系统 — 设计交付包（V2.1）

## 三端职责

| 端 | 职责 | 布局 / 入口 |
| --- | --- | --- |
| **网页前台** | 仅数据展示（告警、物资、监测、定位、围栏等） | **上下结构：顶栏菜单** · [web-portal](prototypes/web-portal/index.html) |
| **网页后台** | 人员权限 + 主数据/单据录入配置（不做打卡扫码） | 左侧管理台 · [web-admin](prototypes/web-admin/index.html) |
| **管控 APP** | 展示 + **现场执行**（运输打卡、盘点扫码、入出库扫码） | 移动端 · [app](prototypes/app/index.html) |

总导航：[prototypes/index.html](prototypes/index.html)

## 协作关系

```
后台开单/建盘点任务/绑车辆/配阈值围栏
        ↓
APP 扫码打卡 · 盘点实盘 · 入出库确认
        ↓
前台（顶栏）实时展示告警、库存、轨迹、报表
```

## 人员权限

- RBAC：人员 ↔ 角色 ↔ 权限点 + 仓库范围  
- 原型：[人员管理](prototypes/web-admin/users.html) · [角色权限](prototypes/web-admin/roles.html)

## 文档

- [系统设计文档 V2.1](docs/智能危废管控系统设计文档.md)
- [架构与组网示意图](docs/架构与组网示意图.md)
