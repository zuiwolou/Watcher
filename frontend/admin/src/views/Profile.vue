<template>
  <div style="max-width:640px">
    <h3>个人中心</h3>
    <el-descriptions :column="1" border v-loading="loading">
      <el-descriptions-item label="用户 ID">{{ me.id }}</el-descriptions-item>
      <el-descriptions-item label="用户名">{{ me.username }}</el-descriptions-item>
      <el-descriptions-item label="显示名">{{ me.displayName || '-' }}</el-descriptions-item>
      <el-descriptions-item label="客户端">{{ (me.clients || []).join(', ') || '-' }}</el-descriptions-item>
      <el-descriptions-item label="仓 Scope">{{ (me.warehouseIds || []).join(', ') || '-' }}</el-descriptions-item>
      <el-descriptions-item label="权限数">{{ (me.permissions || []).length }}</el-descriptions-item>
    </el-descriptions>
    <el-collapse style="margin-top:16px">
      <el-collapse-item title="权限列表" name="perms">
        <el-tag v-for="p in me.permissions || []" :key="p" style="margin:4px">{{ p }}</el-tag>
      </el-collapse-item>
    </el-collapse>
    <el-button type="primary" style="margin-top:16px" @click="$router.push('/change-password')">修改密码</el-button>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const me = ref({})
const loading = ref(false)
onMounted(async () => {
  loading.value = true
  try { me.value = (await http.get('/me')) || {} } finally { loading.value = false }
})
</script>
