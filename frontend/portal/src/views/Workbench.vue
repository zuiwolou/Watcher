<template>
  <div>
    <h2>工作台</h2>
    <el-row :gutter="16">
      <el-col :span="8"><el-card shadow="hover">未确认告警：{{ raised }}</el-card></el-col>
      <el-col :span="8"><el-card shadow="hover">物资条目：{{ materials }}</el-card></el-col>
      <el-col :span="8"><el-card shadow="hover">实时通道：{{ wsState }}</el-card></el-col>
    </el-row>
  </div>
</template>
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import http from '../api'
const raised = ref(0)
const materials = ref(0)
const wsState = ref('未连接')
let ws
onMounted(async () => {
  const alarms = await http.get('/alarms', { params: { status: 'raised' } })
  raised.value = alarms.total || (alarms.list || []).length
  const mats = await http.get('/materials')
  materials.value = mats.total || (mats.list || []).length
  const token = localStorage.getItem('accessToken')
  try {
    ws = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/v1/stream?token=${token}`)
    ws.binaryType = 'arraybuffer'
    ws.onopen = () => { wsState.value = '已连接' }
    ws.onclose = () => { wsState.value = '断开' }
    ws.onmessage = () => { wsState.value = '收到帧 ' + new Date().toLocaleTimeString() }
  } catch {
    wsState.value = '不可用'
  }
})
onUnmounted(() => { try { ws?.close() } catch (_) {} })
</script>
