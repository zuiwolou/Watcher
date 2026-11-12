<template>
  <div>
    <div class="bar">
      <h3>告警</h3>
      <div>
        <el-select v-model="status" clearable placeholder="状态" style="width:140px;margin-right:8px" @change="load">
          <el-option label="全部" value="" />
          <el-option label="raised" value="raised" />
          <el-option label="acked" value="acked" />
          <el-option label="closed" value="closed" />
        </el-select>
        <el-select v-model="level" clearable placeholder="级别" style="width:140px;margin-right:8px" @change="load">
          <el-option label="全部" value="" />
          <el-option label="urgent" value="urgent" />
          <el-option label="important" value="important" />
          <el-option label="info" value="info" />
        </el-select>
        <el-button @click="load">刷新</el-button>
        <el-button type="warning" @click="goThreshold">去阈值评估</el-button>
      </div>
    </div>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="100" />
      <el-table-column prop="Title" label="标题" />
      <el-table-column prop="Level" label="级别" width="100" />
      <el-table-column prop="Status" label="状态" width="100" />
      <el-table-column prop="DeviceID" label="设备" width="120" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="ack(row.ID)" :disabled="row.Status!=='raised'">确认</el-button>
          <el-button size="small" type="danger" @click="close(row.ID)" :disabled="row.Status==='closed'">关闭</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'
const router = useRouter()
const list = ref([])
const status = ref('raised')
const level = ref('')
async function load() {
  const params = { page: 1, pageSize: 50 }
  if (status.value) params.status = status.value
  if (level.value) params.level = level.value
  list.value = (await http.get('/alarms', { params })).list || []
}
async function ack(id) { await http.post(`/alarms/${id}/ack`); load() }
async function close(id) { await http.post(`/alarms/${id}/close`); load() }
function goThreshold() { router.push('/thresholds') }
onMounted(load)
</script>
<style scoped>
.bar { display:flex; justify-content:space-between; align-items:center; }
h3 { margin:0; }
</style>
