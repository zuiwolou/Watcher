<template>
  <div>
    <h3>运输</h3>
    <el-button type="primary" @click="createUnit">新建单元</el-button>
    <el-table :data="units" border style="margin:12px 0">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="PlateNo" label="车牌/编号" />
      <el-table-column prop="Type" label="类型" />
    </el-table>
    <el-form inline>
      <el-form-item label="出库单ID"><el-input v-model="outboundId" style="width:100px" /></el-form-item>
      <el-form-item label="单元ID"><el-input v-model="unitId" style="width:100px" /></el-form-item>
      <el-form-item label="司机用户ID"><el-input v-model="driverId" style="width:100px" /></el-form-item>
      <el-button type="primary" @click="createTask">创建任务</el-button>
    </el-form>
    <el-table :data="tasks" border>
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="TaskNo" label="任务号" />
      <el-table-column prop="Status" label="状态" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const units = ref([])
const tasks = ref([])
const outboundId = ref('')
const unitId = ref('')
const driverId = ref('1')
async function load() {
  units.value = (await http.get('/transport/units')) || []
  tasks.value = (await http.get('/transport/tasks')).list || []
}
async function createUnit() {
  await http.post('/transport/units', { plateNo: '沪A'+Date.now().toString().slice(-5), type: 'truck' })
  load()
}
async function createTask() {
  await http.post('/transport/tasks', {
    outboundOrderId: Number(outboundId.value), unitId: Number(unitId.value), driverUserId: Number(driverId.value), warehouseId: 'WH01'
  })
  load()
}
onMounted(load)
</script>
