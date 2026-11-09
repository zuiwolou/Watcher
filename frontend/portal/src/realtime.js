import protobuf from 'protobufjs'
import protoSrc from '@contracts/proto/push/v1/envelope.proto?raw'

const MsgTypeName = {
  0: 'UNSPECIFIED',
  1: 'TELEMETRY',
  2: 'LOCATION',
  3: 'EVENT',
  4: 'ALARM',
  5: 'HEARTBEAT'
}

let PushEnvelopeType = null
let rootPromise = null

function loadTypes() {
  if (PushEnvelopeType) return Promise.resolve(PushEnvelopeType)
  if (!rootPromise) {
    rootPromise = Promise.resolve().then(() => {
      const parsed = protobuf.parse(protoSrc, { keepCase: false })
      PushEnvelopeType = parsed.root.lookupType('push.v1.PushEnvelope')
      return PushEnvelopeType
    })
  }
  return rootPromise
}

function normalize(msg) {
  const typeNum = typeof msg.msgType === 'number' ? msg.msgType : (msg.msg_type || 0)
  const out = {
    ver: msg.ver || 1,
    msgType: MsgTypeName[typeNum] || String(typeNum),
    msgTypeNum: typeNum,
    msgId: msg.msgId || msg.msg_id || '',
    tsUnixMs: Number(msg.tsUnixMs || msg.ts_unix_ms || Date.now()),
    warehouseId: msg.warehouseId || msg.warehouse_id || '',
    deviceId: msg.deviceId || msg.device_id || '',
    telemetry: null,
    location: null,
    alarm: null
  }
  const tel = msg.telemetry || msg.Telemetry
  if (tel) {
    out.telemetry = {
      temperature: tel.temperature ?? 0,
      humidity: tel.humidity ?? 0,
      waterLeak: tel.waterLeak ?? tel.water_leak ?? 0,
      intrusion: tel.intrusion ?? 0,
      custom: tel.custom || {}
    }
  }
  const loc = msg.location || msg.Location
  if (loc) {
    out.location = {
      lat: loc.lat ?? 0,
      lng: loc.lng ?? 0,
      speed: loc.speed ?? 0,
      heading: loc.heading ?? 0,
      accuracy: loc.accuracy ?? 0,
      unitId: loc.unitId || loc.unit_id || ''
    }
  }
  const alarm = msg.alarm || msg.Alarm
  if (alarm) {
    out.alarm = {
      alarmId: alarm.alarmId || alarm.alarm_id || '',
      ruleCode: alarm.ruleCode || alarm.rule_code || '',
      level: alarm.level || '',
      title: alarm.title || '',
      status: alarm.status || '',
      metric: alarm.metric || '',
      value: alarm.value ?? 0,
      threshold: alarm.threshold ?? 0
    }
  }
  return out
}

export async function decodePushEnvelope(buf) {
  const Type = await loadTypes()
  const bytes = buf instanceof ArrayBuffer ? new Uint8Array(buf) : buf
  const msg = Type.decode(bytes)
  const obj = Type.toObject(msg, {
    longs: Number,
    enums: Number,
    bytes: String,
    defaults: true,
    oneofs: true
  })
  return normalize(obj)
}

/**
 * Shared Portal realtime bus: Protobuf WSS decode + reconnect + snapshot refresh hooks.
 */
class RealtimeBus {
  constructor() {
    this.listeners = new Set()
    this.ws = null
    this.state = 'idle'
    this.lastError = ''
    this.reconnectAttempt = 0
    this.timer = null
    this.started = false
    this.latest = {
      telemetryByDevice: {},
      locationsByDevice: {},
      recentAlarms: [],
      lastMsgAt: 0
    }
  }

  on(fn) {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  emit(evt) {
    for (const fn of this.listeners) {
      try { fn(evt) } catch (_) { /* ignore */ }
    }
  }

  getState() { return this.state }

  start() {
    if (this.started) return
    this.started = true
    this.connect()
  }

  stop() {
    this.started = false
    if (this.timer) clearTimeout(this.timer)
    try { this.ws?.close() } catch (_) {}
    this.ws = null
    this.state = 'idle'
  }

  connect() {
    const token = localStorage.getItem('accessToken')
    if (!token) {
      this.state = 'no-token'
      this.emit({ type: 'state', state: this.state })
      return
    }
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${location.host}/v1/stream?token=${encodeURIComponent(token)}`
    try {
      this.ws = new WebSocket(url)
    } catch (e) {
      this.state = 'error'
      this.lastError = String(e)
      this.scheduleReconnect()
      return
    }
    this.ws.binaryType = 'arraybuffer'
    this.state = 'connecting'
    this.emit({ type: 'state', state: this.state })
    this.ws.onopen = () => {
      this.state = 'open'
      this.reconnectAttempt = 0
      this.emit({ type: 'state', state: this.state })
    }
    this.ws.onclose = () => {
      this.state = 'closed'
      this.emit({ type: 'state', state: this.state })
      this.scheduleReconnect()
    }
    this.ws.onerror = () => {
      this.state = 'error'
      this.emit({ type: 'state', state: this.state })
    }
    this.ws.onmessage = async (ev) => {
      try {
        const env = await decodePushEnvelope(ev.data)
        this.lastError = ''
        this.latest.lastMsgAt = Date.now()
        if (env.msgType === 'TELEMETRY' && env.telemetry) {
          this.latest.telemetryByDevice[env.deviceId || '_'] = { ...env, at: Date.now() }
        }
        if (env.msgType === 'LOCATION' && env.location) {
          this.latest.locationsByDevice[env.deviceId || '_'] = { ...env, at: Date.now() }
        }
        if (env.msgType === 'ALARM' && env.alarm) {
          this.latest.recentAlarms = [env, ...this.latest.recentAlarms].slice(0, 30)
        }
        this.emit({ type: 'message', envelope: env })
      } catch (e) {
        this.lastError = String(e?.message || e)
        this.emit({ type: 'decode-error', error: this.lastError })
      }
    }
  }

  scheduleReconnect() {
    if (!this.started) return
    if (this.timer) clearTimeout(this.timer)
    const delay = Math.min(30000, 1000 * (2 ** Math.min(this.reconnectAttempt, 5)))
    this.reconnectAttempt += 1
    this.timer = setTimeout(() => this.connect(), delay)
  }
}

export const realtime = new RealtimeBus()
