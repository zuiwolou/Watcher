<template>
  <div>
    <h3>审批 / 库存调整</h3>
    <el-form inline>
      <el-form-item label="物资ID"><el-input v-model="materialId" style="width:100px" /></el-form-item>
      <el-form-item label="调整量"><el-input-number v-model="delta" /></el-form-item>
      <el-button type="primary" @click="adjust">提交调整</el-button>
      <el-button @click="loadTodo">刷新待办</el-button>
    </el-form>
    <el-table :data="todo" border>
      <el-table-column prop="ID" label="审批单" width="90" />
      <el-table-column prop="BizType" label="类型" />
      <el-table-column prop="CurrentLevel" label="当前级" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" type="success" @click="act(row.ID,'pass')">通过</el-button>
          <el-button size="small" type="danger" @click="act(row.ID,'reject')">驳回</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const todo = ref([])
const materialId = ref('')
const delta = ref(1)
async function loadTodo() { todo.value = (await http.get('/approvals/todo')).list || [] }
async function adjust() {
  await http.post('/stocks/adjustments', { warehouseId: 'WH01', materialId: Number(materialId.value), delta: delta.value, reason: 'demo' })
  loadTodo()
}
async function act(id, action) {
  await http.post(`/approvals/${id}/act`, { action, comment: action })
  loadTodo()
}
onMounted(loadTodo)
</script>
