<template>
  <div>
    <h3>审计日志</h3>
    <el-button @click="load">刷新</el-button>
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
async function load() { list.value = (await http.get('/admin/audit')).list || [] }
onMounted(load)
</script>
