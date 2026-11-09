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
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" style="width:140px">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-form-item label="出库单ID"><el-input v-model="outboundId" style="width:100px" /></el-form-item>
      <el-form-item label="单元ID"><el-input v-model="unitId" style="width:100px" /></el-form-item>
      <el-form-item label="司机用户ID"><el-input v-model="driverId" style="width:100px" /></el-form-item>
      <el-button type="primary" @click="createTask">创建任务</el-button>
    </el-form>
    <el-table :data="tasks" border>
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="TaskNo" label="任务号" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button size="small" @click="checkin(row.ID)">打卡</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const units = ref([])
const tasks = ref([])
const warehouseId = ref('')
const outboundId = ref('')
const unitId = ref('')
const driverId = ref('')
async function load() {
  units.value = (await http.get('/transport/units')) || []
  tasks.value = (await http.get('/transport/tasks')).list || []
}
async function createUnit() {
  const { value } = await ElMessageBox.prompt('车牌/编号', '新建运输单元')
  await http.post('/transport/units', { plateNo: value, type: 'truck' })
  load()
}
async function createTask() {
  await http.post('/transport/tasks', {
    outboundOrderId: Number(outboundId.value), unitId: Number(unitId.value),
    driverUserId: Number(driverId.value), warehouseId: warehouseId.value
  })
  load()
}
async function checkin(taskId) {
  const { value } = await ElMessageBox.prompt('节点类型 depart|waypoint|arrive', '运输打卡', { inputValue: 'waypoint' })
  await http.post('/transport/checkins', {
    taskId, nodeType: value, clientRequestId: crypto.randomUUID()
  })
  load()
}
onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  load()
})
</script>
