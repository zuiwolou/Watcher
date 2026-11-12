<template>
  <div>
    <div class="bar">
      <h2>监测曲线</h2>
      <el-button link type="primary" @click="$router.push('/alarms')">查看告警 →</el-button>
    </div>
    <el-form inline>
      <el-form-item label="仓ID"><el-input v-model="warehouseId" style="width:120px" /></el-form-item>
      <el-form-item label="指标">
        <el-select v-model="metric" style="width:140px">
          <el-option label="temperature" value="temperature" />
          <el-option label="humidity" value="humidity" />
          <el-option label="waterLeak" value="waterLeak" />
          <el-option label="intrusion" value="intrusion" />
        </el-select>
      </el-form-item>
      <el-button type="primary" @click="loadHistory">查询历史</el-button>
      <el-tag type="success" style="margin-left:8px">实时点 {{ liveCount }}</el-tag>
    </el-form>

    <el-row :gutter="12" style="margin-top:12px">
      <el-col :span="14">
        <el-card shadow="never">
          <template #header>历史曲线 · {{ metric }}</template>
          <MetricChart :series="histSeries" :title="metric" />
        </el-card>
        <el-card shadow="never" style="margin-top:12px">
          <template #header>实时曲线（Protobuf TELEMETRY）</template>
          <MetricChart :series="liveSeries" :title="metric + ' live'" />
        </el-card>
      </el-col>
      <el-col :span="10">
        <el-table :data="rows" border max-height="520" size="small">
          <el-table-column prop="Metric" label="指标" width="100" />
          <el-table-column prop="Value" label="值" width="90" />
          <el-table-column prop="DeviceID" label="设备" />
          <el-table-column prop="Ts" label="时间" width="170" />
        </el-table>
      </el-col>
    </el-row>
  </div>
</template>
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import http from '../api'
import { realtime } from '../realtime'
import MetricChart from '../components/MetricChart.vue'

const warehouseId = ref('WH01')
const metric = ref('temperature')
const rows = ref([])
const histSeries = ref([])
const liveSeries = ref([])
const liveCount = ref(0)
let off

function metricValue(tel) {
  if (!tel) return null
  if (metric.value === 'temperature') return tel.temperature
  if (metric.value === 'humidity') return tel.humidity
  if (metric.value === 'waterLeak') return tel.waterLeak
  if (metric.value === 'intrusion') return tel.intrusion
  return tel.custom?.[metric.value]
}

async function loadHistory() {
  const list = (await http.get('/history/telemetry', {
    params: { warehouseId: warehouseId.value, metric: metric.value }
  })) || []
  rows.value = Array.isArray(list) ? list : (list.list || [])
  const sorted = [...rows.value].sort((a, b) => new Date(a.Ts) - new Date(b.Ts))
  histSeries.value = sorted.map((r) => ({ t: new Date(r.Ts).getTime(), v: Number(r.Value) }))
}

onMounted(() => {
  loadHistory()
  realtime.start()
  off = realtime.on((evt) => {
    if (evt.type !== 'message') return
    const env = evt.envelope
    if (env.msgType !== 'TELEMETRY') return
    if (warehouseId.value && env.warehouseId && env.warehouseId !== warehouseId.value) return
    const v = metricValue(env.telemetry)
    if (v == null) return
    liveSeries.value = [...liveSeries.value, { t: env.tsUnixMs, v: Number(v) }].slice(-80)
    liveCount.value = liveSeries.value.length
  })
})
onUnmounted(() => { off?.() })
</script>
<style scoped>
.bar{display:flex;justify-content:space-between;align-items:center}
h2{margin:0}
</style>
