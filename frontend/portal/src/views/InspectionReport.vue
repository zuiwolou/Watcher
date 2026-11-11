<template>
  <div>
    <h2>巡检报表</h2>
    <el-table :data="tasks" border>
      <el-table-column prop="ID" label="ID" width="70" />
      <el-table-column prop="PlanID" label="计划" width="90" />
      <el-table-column prop="RouteID" label="路线" width="90" />
      <el-table-column prop="Status" label="状态" width="120" />
      <el-table-column prop="AssigneeID" label="执行人" width="100" />
      <el-table-column prop="CreatedAt" label="创建时间" width="180" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const tasks = ref([])
onMounted(async () => {
  tasks.value = (await http.get('/inspection/tasks?page=1&pageSize=50')).list || []
})
</script>
