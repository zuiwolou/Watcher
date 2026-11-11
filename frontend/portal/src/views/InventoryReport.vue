<template>
  <div>
    <h2>盘点报表</h2>
    <el-table :data="tasks" border @row-click="onRow">
      <el-table-column prop="ID" label="ID" width="70" />
      <el-table-column prop="Name" label="任务" />
      <el-table-column prop="WarehouseID" label="仓" width="100" />
      <el-table-column prop="Status" label="状态" width="110" />
      <el-table-column prop="CreatedAt" label="创建时间" width="180" />
    </el-table>
    <h3 v-if="taskId" style="margin-top:20px">明细 #{{ taskId }}</h3>
    <el-table v-if="taskId" :data="lines" border>
      <el-table-column prop="UniqueCode" label="唯一码" />
      <el-table-column prop="MaterialID" label="物资ID" width="90" />
      <el-table-column prop="Scanned" label="已扫" width="80" />
      <el-table-column prop="DiffType" label="差异" width="100" />
      <el-table-column prop="Note" label="备注" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const tasks = ref([])
const lines = ref([])
const taskId = ref(0)
async function load() {
  tasks.value = (await http.get('/inventory/tasks?page=1&pageSize=50')).list || []
}
async function onRow(row) {
  taskId.value = row.ID
  lines.value = (await http.get(`/inventory/tasks/${row.ID}/lines`)) || []
  if (!Array.isArray(lines.value)) lines.value = []
}
onMounted(load)
</script>
