<template>
  <div>
    <h2>工作台</h2>
    <el-row :gutter="16">
      <el-col :span="6"><el-card shadow="hover">未确认告警：{{ raised }}</el-card></el-col>
      <el-col :span="6"><el-card shadow="hover">物资条目：{{ materials }}</el-card></el-col>
      <el-col :span="6"><el-card shadow="hover">实时通道：{{ wsState }}</el-card></el-col>
      <el-col :span="6"><el-card shadow="hover">最近遥测：{{ lastTel }}</el-card></el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="14">
        <el-card shadow="never">
          <template #header>温度实时曲线（Protobuf 推送）</template>
          <MetricChart :series="tempSeries" title="temperature" />
        </el-card>
      </el-col>
      <el-col :span="10">
        <el-card shadow="never">
          <template #header>实时告警流</template>
          <el-timeline>
            <el-timeline-item
              v-for="(a, i) in recentAlarms"
              :key="i"
              type="danger"
              :timestamp="fmt(a.tsUnixMs)"
            >
              {{ a.alarm?.title || a.alarm?.metric }} · {{ a.deviceId }}
            </el-timeline-item>
          </el-timeline>
          <div v-if="!recentAlarms.length" class="muted">等待 ALARM 帧…</div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import http from '../api'
import { realtime } from '../realtime'
import MetricChart from '../components/MetricChart.vue'

const raised = ref(0)
const materials = ref(0)
const wsState = ref(realtime.getState())
const lastTel = ref('-')
const tempSeries = ref([])
const recentAlarms = ref([])
let off

function fmt(ms) {
  if (!ms) return ''
  return new Date(ms).toLocaleTimeString()
}

function pushTemp(env) {
  const v = env.telemetry?.temperature
  if (v == null) return
  tempSeries.value = [...tempSeries.value, { t: env.tsUnixMs, v }].slice(-60)
  lastTel.value = `${env.deviceId || '-'} ${Number(v).toFixed(1)}℃`
}

async function snapshot() {
  const alarms = await http.get('/alarms', { params: { status: 'raised', page: 1, pageSize: 1 } })
  raised.value = alarms.total || (alarms.list || []).length
  const mats = await http.get('/materials', { params: { page: 1, pageSize: 1 } })
  materials.value = mats.total || (mats.list || []).length
}

onMounted(async () => {
  await snapshot()
  realtime.start()
  off = realtime.on((evt) => {
    if (evt.type === 'state') wsState.value = evt.state
    if (evt.type === 'message') {
      const env = evt.envelope
      if (env.msgType === 'TELEMETRY') pushTemp(env)
      if (env.msgType === 'ALARM') {
        recentAlarms.value = [env, ...recentAlarms.value].slice(0, 12)
        raised.value += 1
      }
    }
  })
})
onUnmounted(() => { off?.() })
</script>
<style scoped>
.muted { color:#829ab1; font-size:13px; }
</style>
