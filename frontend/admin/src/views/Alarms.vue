<template>
  <div>
    <h3>告警</h3>
    <el-button @click="evalDemo">模拟超阈</el-button>
    <el-button @click="load">刷新</el-button>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="Title" label="标题" />
      <el-table-column prop="Level" label="级别" width="100" />
      <el-table-column prop="Status" label="状态" width="100" />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button size="small" @click="ack(row.ID)" :disabled="row.Status!=='raised'">确认</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const list = ref([])
async function load() { list.value = (await http.get('/alarms')).list || [] }
async function evalDemo() {
  await http.put('/thresholds/WH01', { metric: 'temperature', upper: 30, lower: 0 })
  await http.post('/monitor/evaluate', { warehouseId: 'WH01', deviceId: 'DEV01', metrics: { temperature: 45 } })
  load()
}
async function ack(id) { await http.post(`/alarms/${id}/ack`); load() }
onMounted(load)
</script>
