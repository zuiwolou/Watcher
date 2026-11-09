<template>
  <div>
    <h3>巡检</h3>
    <el-form inline>
      <el-form-item label="计划名"><el-input v-model="planName" style="width:140px" /></el-form-item>
      <el-form-item label="路线"><el-input v-model="routeName" style="width:120px" /></el-form-item>
      <el-form-item label="点位"><el-input v-model="pointName" style="width:100px" /></el-form-item>
      <el-form-item label="纬度"><el-input-number v-model="lat" :step="0.001" /></el-form-item>
      <el-form-item label="经度"><el-input-number v-model="lng" :step="0.001" /></el-form-item>
      <el-form-item label="执行人ID"><el-input v-model="assigneeId" style="width:90px" /></el-form-item>
      <el-button type="primary" @click="bootstrap">创建计划链路</el-button>
    </el-form>
    <el-table :data="tasks" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="RouteID" label="路线" />
      <el-table-column prop="AssigneeID" label="执行人" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button size="small" @click="checkin(row)">打卡</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import http from '../api'
const tasks = ref([])
const planName = ref('日常巡检')
const routeName = ref('仓区A')
const pointName = ref('门口')
const lat = ref(31.2)
const lng = ref(121.5)
const assigneeId = ref('')
const lastPointId = ref(0)
async function load() { tasks.value = (await http.get('/inspection/tasks')).list || [] }
async function bootstrap() {
  const plan = await http.post('/inspection/plans', { name: planName.value, cycleCron: '0 8 * * *' })
  const route = await http.post('/inspection/routes', { planId: plan.ID, name: routeName.value })
  const pt = await http.post('/inspection/points', {
    routeId: route.ID, seq: 1, name: pointName.value, lat: lat.value, lng: lng.value, radiusM: 200
  })
  lastPointId.value = pt.ID
  await http.post('/inspection/tasks', {
    planId: plan.ID, routeId: route.ID, assigneeId: Number(assigneeId.value || 0)
  })
  load()
}
async function checkin(row) {
  let pointId = lastPointId.value
  if (!pointId) {
    const { value } = await ElMessageBox.prompt('点位ID', '巡检打卡')
    pointId = Number(value)
  }
  await http.post(`/inspection/tasks/${row.ID}/checkin`, {
    pointId, status: 'done', lat: lat.value, lng: lng.value, clientRequestId: crypto.randomUUID()
  })
  load()
}
onMounted(load)
</script>
