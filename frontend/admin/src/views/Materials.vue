<template>
  <div>
    <div class="bar"><h3>物资档案</h3></div>
    <el-form inline>
      <el-form-item label="编码"><el-input v-model="form.code" /></el-form-item>
      <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="单位"><el-input v-model="form.unit" style="width:80px" /></el-form-item>
      <el-form-item label="危废代码"><el-input v-model="form.wasteCode" /></el-form-item>
      <el-button type="primary" @click="create">新建</el-button>
      <el-button @click="load">刷新</el-button>
    </el-form>
    <el-table :data="list" border>
      <el-table-column prop="ID" label="ID" width="70" />
      <el-table-column prop="Code" label="编码" />
      <el-table-column prop="Name" label="名称" />
      <el-table-column prop="Unit" label="单位" />
      <el-table-column prop="WasteCode" label="危废代码" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'
const list = ref([])
const form = reactive({ code: '', name: '', unit: '只', wasteCode: '' })
async function load() { list.value = (await http.get('/materials')).list || [] }
async function create() {
  if (!form.code || !form.name) { ElMessage.warning('请填写编码与名称'); return }
  await http.post('/materials', { ...form })
  form.code = ''; form.name = ''
  load()
}
onMounted(load)
</script>
<style scoped>.bar{display:flex;justify-content:space-between;align-items:center}</style>
