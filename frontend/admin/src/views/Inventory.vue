<template>
  <div>
    <h3>盘点任务</h3>
    <el-form inline>
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" style="width:140px">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称"><el-input v-model="name" style="width:160px" /></el-form-item>
      <el-button type="primary" @click="create">创建任务</el-button>
    </el-form>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="Name" label="名称" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="320">
        <template #default="{ row }">
          <el-button size="small" @click="scan(row.ID)">扫码</el-button>
          <el-button size="small" @click="submit(row.ID)">提交</el-button>
          <el-button size="small" type="success" @click="post(row.ID)">过账</el-button>
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
const list = ref([])
const warehouseId = ref('')
const name = ref('')
async function load() { list.value = (await http.get('/inventory/tasks')).list || [] }
async function create() {
  await http.post('/inventory/tasks', {
    name: name.value || ('盘点' + Date.now().toString().slice(-4)),
    warehouseId: warehouseId.value
  })
  load()
}
async function scan(id) {
  const { value } = await ElMessageBox.prompt('唯一码', '盘点扫码')
  await http.post(`/inventory/tasks/${id}/scan`, { uniqueCode: value, clientRequestId: crypto.randomUUID() })
  load()
}
async function submit(id) { await http.post(`/inventory/tasks/${id}/submit`); load() }
async function post(id) { await http.post(`/inventory/tasks/${id}/post`); load() }
onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  load()
})
</script>
