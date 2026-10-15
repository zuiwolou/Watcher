# 智能危废管控系统 — 设计交付包

面向废铅蓄电池暂存仓的全场景监控与转运管控：**系统设计文档 + APP 高保真原型**。

## 交付物

| 内容 | 路径 |
| --- | --- |
| 系统设计文档（架构 / 组网 / 九大模块 / 接口 / 部署 / 监管对接） | [docs/智能危废管控系统设计文档.md](docs/智能危废管控系统设计文档.md) |
| 架构与组网示意图补充 | [docs/架构与组网示意图.md](docs/架构与组网示意图.md) |
| APP 原型入口 | [prototypes/index.html](prototypes/index.html) |

## 快速查看原型

用浏览器直接打开 `prototypes/index.html`（无需构建）。顶部导航可切换九大功能模块页面。

| 模块 | 页面 |
| --- | --- |
| 首页总览 | `prototypes/index.html` |
| 仓储管理 | `prototypes/pages/warehouse.html` |
| 物资管理 | `prototypes/pages/materials.html` |
| 库存盘点 | `prototypes/pages/inventory.html` |
| 预警管理 | `prototypes/pages/alerts.html` |
| 运输打卡 | `prototypes/pages/checkin.html` |
| 运输单元管理 | `prototypes/pages/units.html` |
| 运输定位跟踪 | `prototypes/pages/tracking.html` |
| 软电子围栏 | `prototypes/pages/geofence.html` |
| 智能监测 | `prototypes/pages/monitor.html` |

## 设计覆盖要点

1. **多模组网**：局域网 / 物联网 / 公网灵活部署，传感器→网关→MQTT→后台→APP 全链路。
2. **后台与 APP**：HTTPS REST + WSS 推送 + JWT；设备侧 MQTTS Topic 规范。
3. **九大 APP 模块**：交互逻辑、操作流程、数据流转均在设计文档第 5 章与对应原型中体现。
4. **合规与对接**：台账、联单字段、告警与轨迹摘要；预留 `/api/v1/open/` 监管 OpenAPI。

## 建议下一步

- 评审设计文档后冻结接口字段与唯一码规则  
- 按原型进行 UI 视觉精修与真机扫码/地图 SDK 联调  
- 选定试点仓体完成组网勘测与网关开箱流程验证  
