<template>
  <div>
    <div class="bar"><h3>物资档案</h3>
      <el-button type="primary" @click="create">新建演示物资</el-button>
    </div>
    <el-table :data="list" border>
      <el-table-column prop="Code" label="编码" />
      <el-table-column prop="Name" label="名称" />
      <el-table-column prop="Unit" label="单位" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const list = ref([])
async function load() { list.value = (await http.get('/materials')).list || [] }
async function create() {
  const n = Date.now().toString().slice(-6)
  await http.post('/materials', { code: 'M'+n, name: '废铅蓄电池'+n, unit: '只' })
  load()
}
onMounted(load)
</script>
<style scoped>.bar{display:flex;justify-content:space-between;align-items:center}</style>
