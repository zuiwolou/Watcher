<template>
  <div>
    <h3>盘点任务</h3>
    <el-button type="primary" @click="create">创建任务</el-button>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="Name" label="名称" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button size="small" @click="submit(row.ID)">提交</el-button>
          <el-button size="small" type="success" @click="post(row.ID)">过账</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const list = ref([])
async function load() { list.value = (await http.get('/inventory/tasks')).list || [] }
async function create() {
  await http.post('/inventory/tasks', { name: '盘点'+Date.now().toString().slice(-4), warehouseId: 'WH01' })
  load()
}
async function submit(id) { await http.post(`/inventory/tasks/${id}/submit`); load() }
async function post(id) { await http.post(`/inventory/tasks/${id}/post`); load() }
onMounted(load)
</script>
