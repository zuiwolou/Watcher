# MQTT 可自定义数据结构规范

| 文档版本 | V1.0 |
| --- | --- |
| 关联 | 《智能危废管控系统设计文档》V2.1+ |
| 目的 | 支持不同厂商网关/传感器自定义上报报文，经映射归一为系统标准模型后参与监测、告警、定位 |

---

## 1. 设计目标

| 目标 | 说明 |
| --- | --- |
| 可自定义 | 后台可配置 Topic 模板、报文 Schema、字段映射，无需改代码适配新设备 |
| 可归一 | 设备侧异构 JSON → 系统**规范内部模型（Canonical）** |
| 可版本化 | Schema Profile 带版本号，设备可绑定指定版本；支持灰度 |
| 可校验 | 入站按 Schema 校验，失败进死信并留审计 |
| 可双向 | 上行遥测/事件/定位；下行指令也可套用自定义结构 |

**原则**：业务规则引擎、告警阈值、前台/APP **只消费 Canonical 模型**；厂商差异全部收敛在「MQTT 接入适配层」。

---

## 2. 总体数据流

```
设备/网关 MQTT Publish（厂商自定义 JSON）
        │
        ▼
┌───────────────────┐
│ Topic 路由匹配     │  ← Topic Template + 变量抽取（tenant/warehouse/device）
└─────────┬─────────┘
          ▼
┌───────────────────┐
│ Schema Profile    │  ← 设备或产品型号绑定的报文模板
│ · 校验 raw payload│
│ · 字段映射 Mapping│
│ · 单位/类型换算   │
└─────────┬─────────┘
          ▼
┌───────────────────┐
│ Canonical Message │  ← 系统标准结构（见第 4 章）
└─────────┬─────────┘
          ▼
  时序入库 / 规则引擎 / 定位服务 / 告警推送
```

下行：

```
业务下发 Canonical Command
  → 按 Profile 反向映射为厂商 payload
  → Publish 到设备 cmd Topic
```

---

## 3. 核心配置实体

### 3.1 Schema Profile（报文模板）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| profileId | string | 唯一 ID，如 `PF-TH-VENDOR-A` |
| name | string | 显示名，如「A 厂温湿度网关」 |
| version | string | 语义化版本 `1.2.0` |
| messageType | enum | `telemetry` / `event` / `location` / `heartbeat` / `command` |
| contentType | enum | 默认 `application/json`（本期） |
| jsonSchema | object | 对 **原始报文** 的 JSON Schema（可选，用于校验） |
| mapping | MappingRule[] | raw → canonical 映射规则 |
| reverseMapping | MappingRule[] | canonical → raw（下行，可选） |
| defaults | object | 缺省字段填充 |
| status | enum | `draft` / `published` / `deprecated` |
| samplePayload | object | 样例报文，供调试 |

### 3.2 Topic Template（主题模板）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| templateId | string | |
| direction | enum | `up` / `down` |
| pattern | string | 含占位符，见下 |
| profileId | string | 匹配该 Topic 时使用的 Profile（可被设备级覆盖） |
| qosDefault | 0/1/2 | |

**占位符约定**：

```
hw/{tenantId}/wh/{warehouseId}/dev/{deviceId}/telemetry
hw/{tenantId}/wh/{warehouseId}/dev/{deviceId}/event
hw/{tenantId}/unit/{unitId}/location
hw/{tenantId}/dev/{deviceId}/cmd
```

自定义示例（厂商原有 Topic 也可登记）：

```
factoryA/{sn}/data
iot/{productKey}/{deviceName}/property/post
```

从 Topic 抽取的变量写入 Canonical 的 `meta.source`。

### 3.3 Device Binding（设备绑定）

| 字段 | 说明 |
| --- | --- |
| deviceId | 系统设备 ID |
| productModel | 可选产品型号，继承默认 Profile |
| uplinkProfiles | `{ telemetry, event, location, heartbeat }` → profileId |
| downlinkProfile | command 用 profileId |
| topicOverrides | 覆盖默认 Topic |

### 3.4 MappingRule（映射规则）

| 字段 | 说明 |
| --- | --- |
| target | Canonical 字段路径，如 `metrics.temperature` |
| source | 原始 JSONPath，如 `$.data.temp` 或 `$.Temp` |
| transform | 可选：`identity` / `number` / `bool` / `enumMap` / `scale` / `expr` |
| params | 变换参数，如 `{ "scale": 0.1, "offset": 0 }` 或枚举表 |
| required | 是否必填；缺失则整包失败或跳过（可配） |

---

## 4. 系统规范内部模型（Canonical）

所有自定义报文最终映射为下列标准信封 + 分类型 body。业务侧 **禁止** 直接依赖厂商字段名。

### 4.1 统一信封

```json
{
  "schema": "hw.canonical.v1",
  "messageType": "telemetry",
  "messageId": "uuid-or-device-seq",
  "ts": "2026-10-15T07:20:00.000Z",
  "receivedAt": "2026-10-15T07:20:01.120Z",
  "tenantId": "T-001",
  "warehouseId": "WH-A01",
  "deviceId": "TH-001",
  "unitId": null,
  "profileId": "PF-TH-VENDOR-A",
  "profileVersion": "1.2.0",
  "meta": {
    "topic": "hw/T-001/wh/WH-A01/dev/TH-001/telemetry",
    "qos": 1,
    "rssi": -67,
    "rawRef": "oss://mqtt-raw/2026/10/15/...."
  },
  "body": {}
}
```

| 字段 | 说明 |
| --- | --- |
| schema | 固定 `hw.canonical.v1`，重大不兼容升 v2 |
| ts | 业务时间（优先设备时间，否则网关/服务器时间） |
| receivedAt | Broker/接入层收到时间 |
| rawRef | 可选保留原始报文引用，便于对账 |

### 4.2 telemetry（环境监测遥测）body

```json
{
  "metrics": {
    "temperature": 28.5,
    "humidity": 62.0,
    "waterLeak": false,
    "intrusion": false,
    "smoke": null,
    "custom": {
      "batteryVoltage": 3.7
    }
  },
  "quality": {
    "battery": 86,
    "signal": -67
  }
}
```

**标准 metric 键（约定）**：

| 键 | 类型 | 单位 | 用途 |
| --- | --- | --- | --- |
| temperature | number | ℃ | 温度阈值 |
| humidity | number | %RH | 湿度阈值 |
| waterLeak | boolean | — | 水浸 |
| intrusion | boolean | — | 入侵 |
| smoke | boolean/number | — | 可选 |
| capacityUsedPct | number | % | 若网关上报库容占比 |
| custom.* | any | — | 扩展指标，不进默认阈值，可配置自定义规则 |

### 4.3 event（设备事件/告警）body

```json
{
  "eventCode": "TEMP_HIGH",
  "level": "CRITICAL",
  "text": "温度超上限",
  "metrics": { "temperature": 36.2 },
  "thresholds": { "temperatureMax": 35 },
  "ackRequired": true
}
```

说明：多数场景由**规则引擎对比 telemetry** 生成告警；`event` 用于设备主动上报的告警/门磁等。

### 4.4 location（定位）body

```json
{
  "lng": 114.085,
  "lat": 22.547,
  "alt": null,
  "speed": 42.0,
  "heading": 120,
  "accuracy": 8,
  "coordType": "GCJ02",
  "provider": "GNSS"
}
```

### 4.5 heartbeat body

```json
{
  "online": true,
  "uptimeSec": 86400,
  "firmware": "1.0.3"
}
```

### 4.6 command（下行）body（Canonical）

```json
{
  "command": "SET_REPORT_INTERVAL",
  "params": { "intervalSec": 60 },
  "expireAt": "2026-10-15T08:00:00Z"
}
```

经 `reverseMapping` 转为厂商格式后下发。

---

## 5. 自定义报文字段设计示例

### 5.1 系统推荐默认上行（可直接作为 Profile「系统默认」）

Topic：`hw/{tenantId}/wh/{warehouseId}/dev/{deviceId}/telemetry`

```json
{
  "deviceId": "TH-001",
  "warehouseId": "WH-A01",
  "ts": "2026-10-15T07:20:00Z",
  "metrics": {
    "temperature": 28.5,
    "humidity": 62.0,
    "waterLeak": false,
    "intrusion": false
  },
  "battery": 86,
  "rssi": -67
}
```

映射：几乎 identity → Canonical。

### 5.2 厂商 A：嵌套 + 异名字段

**原始报文**：

```json
{
  "sn": "A10086",
  "time": 1728976800,
  "data": {
    "temp": 285,
    "humi": 620,
    "leak": 0,
    "door": 1
  },
  "bat": 86
}
```

**Mapping（节选）**：

| target | source | transform | params |
| --- | --- | --- | --- |
| deviceId | `$.sn` | identity | — |
| ts | `$.time` | expr | `fromUnixSec` |
| metrics.temperature | `$.data.temp` | scale | `{ "scale": 0.1 }` |
| metrics.humidity | `$.data.humi` | scale | `{ "scale": 0.1 }` |
| metrics.waterLeak | `$.data.leak` | enumMap | `{ "0": false, "1": true }` |
| metrics.intrusion | `$.data.door` | enumMap | `{ "0": false, "1": true }` |
| quality.battery | `$.bat` | number | — |

### 5.3 厂商 B：数组成员多测点

**原始报文**：

```json
{
  "device": "GW-01",
  "points": [
    { "id": "T1", "type": "temp", "v": 28.5 },
    { "id": "H1", "type": "hum", "v": 62 }
  ]
}
```

**Mapping 策略**：使用 `arrayFind` 变换：

- `metrics.temperature` ← `points[?type=='temp'].v`
- `metrics.humidity` ← `points[?type=='hum'].v`

（实现可用 JMESPath 或自定义脚本沙箱，见第 7 章。）

### 5.4 定位自定义

**原始**：

```json
{ "imei": "8601...", "lon": "114.085012", "lat": "22.547001", "spd": "42" }
```

**映射到** `location` Canonical：`lng/lat/speed`，`unitId` 由设备绑定表解析。

---

## 6. Profile 配置 JSON（完整样例）

```json
{
  "profileId": "PF-TH-VENDOR-A",
  "name": "A厂温湿度网关",
  "version": "1.2.0",
  "messageType": "telemetry",
  "contentType": "application/json",
  "status": "published",
  "topicTemplates": [
    {
      "templateId": "TP-A-UP",
      "direction": "up",
      "pattern": "factoryA/{sn}/data",
      "qosDefault": 1
    }
  ],
  "jsonSchema": {
    "type": "object",
    "required": ["sn", "time", "data"],
    "properties": {
      "sn": { "type": "string" },
      "time": { "type": "integer" },
      "data": {
        "type": "object",
        "required": ["temp", "humi"],
        "properties": {
          "temp": { "type": "number" },
          "humi": { "type": "number" },
          "leak": { "type": "integer" },
          "door": { "type": "integer" }
        }
      },
      "bat": { "type": "number" }
    }
  },
  "mapping": [
    { "target": "deviceId", "source": "$.sn", "transform": "identity", "required": true },
    { "target": "ts", "source": "$.time", "transform": "fromUnixSec", "required": true },
    { "target": "metrics.temperature", "source": "$.data.temp", "transform": "scale", "params": { "scale": 0.1 }, "required": true },
    { "target": "metrics.humidity", "source": "$.data.humi", "transform": "scale", "params": { "scale": 0.1 }, "required": true },
    { "target": "metrics.waterLeak", "source": "$.data.leak", "transform": "enumMap", "params": { "map": { "0": false, "1": true } }, "required": false },
    { "target": "metrics.intrusion", "source": "$.data.door", "transform": "enumMap", "params": { "map": { "0": false, "1": true } }, "required": false },
    { "target": "quality.battery", "source": "$.bat", "transform": "number", "required": false }
  ],
  "defaults": {
    "warehouseId": null
  },
  "samplePayload": {
    "sn": "A10086",
    "time": 1728976800,
    "data": { "temp": 285, "humi": 620, "leak": 0, "door": 0 },
    "bat": 86
  }
}
```

**映射结果（Canonical body 摘要）**：

```json
{
  "metrics": {
    "temperature": 28.5,
    "humidity": 62.0,
    "waterLeak": false,
    "intrusion": false
  },
  "quality": { "battery": 86 }
}
```

---

## 7. 变换函数（transform）清单

| transform | 说明 |
| --- | --- |
| identity | 原样 |
| number / string / bool | 类型强制 |
| scale | `value * scale + offset` |
| fromUnixSec / fromUnixMs | 转 ISO8601 UTC |
| enumMap | 查表映射 |
| expr | 受限表达式，如 `value / 10`（沙箱） |
| jmespath | 复杂提取 |
| arrayFind | 数组中按条件取字段 |
| coalesce | 多 source 兜底 |

安全：`expr` 禁止访问 IO/网络；超时与复杂度限制。

---

## 8. 管理端能力（网页后台）

权限点：`mqtt:schema:view` / `mqtt:schema:edit`（建议仅系统管理员、运维）。

功能：

1. **Profile 列表**：草稿/已发布/废弃；复制版本升级。  
2. **可视化映射编辑**：左侧样例 JSON 树，右侧拖拽绑定到 Canonical 字段。  
3. **Topic 模板管理**：登记并测试匹配。  
4. **设备绑定**：为设备选择 Profile。  
5. **在线调试**：粘贴原始报文 → 预览 Canonical → 模拟入库（测试租户）。  
6. **原始报文抽样**：失败包与抽样成功包可查。

原型页：`prototypes/web-admin/mqtt-schema.html`

---

## 9. 接入服务处理伪流程

```
onMessage(topic, payloadBytes):
  vars = matchTopicTemplates(topic)
  device = resolveDevice(vars) or resolveByPayloadHint()
  profile = device.uplinkProfiles[inferredType] or template.profileId
  raw = parseJson(payloadBytes)
  validate(raw, profile.jsonSchema)           # 失败 → DLQ
  canonical = applyMapping(raw, profile.mapping, vars, device)
  enrich(canonical)                           # warehouseId、tenantId、receivedAt
  persistRawOptional(raw)
  dispatch(canonical)                         # telemetry→TSDB; location→track; event→alarm
```

---

## 10. API（后台配置）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET/POST | `/api/v1/admin/mqtt/profiles` | Profile CRUD |
| POST | `/api/v1/admin/mqtt/profiles/{id}/publish` | 发布版本 |
| POST | `/api/v1/admin/mqtt/profiles/{id}/dry-run` | 样例映射试跑 |
| GET/PUT | `/api/v1/admin/mqtt/topic-templates` | Topic 模板 |
| PUT | `/api/v1/admin/devices/{id}/mqtt-binding` | 设备绑定 Profile |
| GET | `/api/v1/admin/mqtt/dead-letters` | 解析失败队列 |

Dry-run 请求体：`{ "payload": { ... }, "topic": "..." }`  
响应：`{ "canonical": { ... }, "errors": [] }`

---

## 11. 与告警阈值的衔接

- 阈值配置仍针对 Canonical 键：`temperature` / `humidity` / `waterLeak` 等。  
- 若使用 `metrics.custom.xxx`，需在规则引擎增加**自定义指标规则**（后台可选高级项）。  
- Profile 变更后，历史数据不回溯重映射；仅影响新消息。

---

## 12. 合规与安全

- MQTTS + 设备证书/密钥；Topic 按租户隔离 ACL。  
- 原始报文若含敏感信息，按租户加密存储或短周期脱敏。  
- Profile 变更写审计日志（谁、何时、diff）。  
- 映射沙箱防注入。

---

## 13. 落地建议

1. 预置 **系统默认 Profile**（第 5.1 节），新项目开箱即用。  
2. 每接入一款网关：录样例 → 配映射 → dry-run → 绑定 1 台试跑 → 发布。  
3. 禁止业务代码写死厂商字段；扩展只加 Canonical `custom` 或升规范版本。

---

**文档结束**
