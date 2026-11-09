<template>
  <div>
    <div class="bar">
      <h2>告警中心</h2>
      <div>
        <el-tag v-if="liveHint" type="danger" style="margin-right:8px">{{ liveHint }}</el-tag>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="Title" label="标题" />
      <el-table-column prop="Level" label="级别" width="100" />
      <el-table-column prop="Metric" label="指标" width="120" />
      <el-table-column prop="Status" label="状态" width="100" />
      <el-table-column prop="DeviceID" label="设备" width="120" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button size="small" @click="ack(row.ID)" :disabled="row.Status!=='raised'">确认</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import http from '../api'
import { realtime } from '../realtime'

const list = ref([])
const liveHint = ref('')
let off

async function load() {
  list.value = (await http.get('/alarms', { params: { page: 1, pageSize: 50 } })).list || []
}

async function ack(id) {
  await http.post(`/alarms/${id}/ack`)
  load()
}

onMounted(() => {
  load()
  realtime.start()
  off = realtime.on((evt) => {
    if (evt.type === 'message' && evt.envelope?.msgType === 'ALARM') {
      liveHint.value = `新告警 ${evt.envelope.alarm?.title || ''} @ ${new Date().toLocaleTimeString()}`
      // soft refresh list
      load()
    }
  })
})
onUnmounted(() => { off?.() })
</script>
<style scoped>
.bar{display:flex;justify-content:space-between;align-items:center}
h2{margin:0}
</style>
