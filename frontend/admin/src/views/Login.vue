<template>
  <div class="wrap">
    <el-card class="card">
      <h2>智能危废管控 · 后台</h2>
      <el-form @submit.prevent="onLogin">
        <el-form-item label="账号"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">登录</el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'
import { saveSession } from '../auth'

const router = useRouter()
const loading = ref(false)
const form = reactive({ username: 'admin', password: 'Admin@123' })

async function onLogin() {
  loading.value = true
  try {
    const data = await http.post('/auth/login', { ...form, mode: 'local' })
    saveSession(data)
    if (data.mustChangePassword) router.push('/change-password')
    else router.push('/')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.wrap { min-height: 100vh; display:flex; align-items:center; justify-content:center; background: linear-gradient(160deg,#0b3d4a,#102a43); }
.card { width: 380px; }
h2 { margin: 0 0 16px; color: #0b3d4a; }
</style>
