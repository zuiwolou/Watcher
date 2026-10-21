<template>
  <div>
    <h3>巡检</h3>
    <el-button type="primary" @click="bootstrap">一键创建计划+路线+点+任务</el-button>
    <el-table :data="tasks" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="RouteID" label="路线" />
      <el-table-column prop="AssigneeID" label="执行人" />
      <el-table-column prop="Status" label="状态" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const tasks = ref([])
async function load() { tasks.value = (await http.get('/inspection/tasks')).list || [] }
async function bootstrap() {
  const plan = await http.post('/inspection/plans', { name: '日常巡检', cycleCron: '0 8 * * *' })
  const route = await http.post('/inspection/routes', { planId: plan.ID, name: '仓区A' })
  await http.post('/inspection/points', { routeId: route.ID, seq: 1, name: '门口', lat: 31.2, lng: 121.5, radiusM: 200 })
  await http.post('/inspection/tasks', { planId: plan.ID, routeId: route.ID, assigneeId: 1 })
  load()
}
onMounted(load)
</script>
