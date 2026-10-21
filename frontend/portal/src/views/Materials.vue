<template>
  <div>
    <h2>物资总览</h2>
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
onMounted(async () => { list.value = (await http.get('/materials')).list || [] })
</script>
