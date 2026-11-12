<template>
  <div>
    <div class="bar">
      <h3>审计日志</h3>
      <el-button @click="load">刷新</el-button>
    </div>
    <el-form inline style="margin-top:8px">
      <el-form-item label="动作">
        <el-input v-model="action" clearable placeholder="如 alarm.ack" style="width:160px" />
      </el-form-item>
      <el-form-item label="操作者ID">
        <el-input v-model="actorId" clearable placeholder="用户数字 ID" style="width:140px" />
      </el-form-item>
      <el-button type="primary" @click="load">查询</el-button>
    </el-form>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="70" />
      <el-table-column prop="ActorID" label="操作者" width="90" />
      <el-table-column prop="Action" label="动作" />
      <el-table-column prop="ResourceType" label="资源" />
      <el-table-column prop="ResourceID" label="资源ID" />
      <el-table-column prop="Client" label="端" width="80" />
      <el-table-column prop="IP" label="IP" />
      <el-table-column prop="CreatedAt" label="时间" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const list = ref([])
const action = ref('')
const actorId = ref('')
async function load() {
  const params = { page: 1, pageSize: 50 }
  if (action.value) params.action = action.value
  if (actorId.value) params.actorId = actorId.value
  list.value = (await http.get('/admin/audit', { params })).list || []
}
onMounted(load)
</script>
<style scoped>
.bar { display:flex; justify-content:space-between; align-items:center; }
h3 { margin:0; }
</style>
