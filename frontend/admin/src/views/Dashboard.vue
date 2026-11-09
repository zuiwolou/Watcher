<template>
  <div>
    <h2>工作台</h2>
    <el-row :gutter="16">
      <el-col :span="6"><el-card shadow="hover">待确认告警<br /><b>{{ stats.alarms }}</b></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover">物资数<br /><b>{{ stats.materials }}</b></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover">我的审批待办<br /><b>{{ stats.todo }}</b></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover">入库单<br /><b>{{ stats.inbound }}</b></el-card></el-col>
    </el-row>
  </div>
</template>
<script setup>
import { onMounted, reactive } from 'vue'
import http from '../api'
const stats = reactive({ alarms: 0, materials: 0, todo: 0, inbound: 0 })
onMounted(async () => {
  try {
    const a = await http.get('/alarms', { params: { status: 'raised' } })
    stats.alarms = a.total || (a.list || []).length
  } catch (_) {}
  try {
    const m = await http.get('/materials')
    stats.materials = m.total || (m.list || []).length
  } catch (_) {}
  try {
    const t = await http.get('/approvals/todo')
    stats.todo = t.total || (t.list || []).length
  } catch (_) {}
  try {
    const i = await http.get('/inbound/orders')
    stats.inbound = i.total || (i.list || []).length
  } catch (_) {}
})
</script>
