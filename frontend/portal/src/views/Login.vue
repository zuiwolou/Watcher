<template>
  <div class="wrap">
    <el-card class="card">
      <h2>智能危废管控 · 前台</h2>
      <el-form @submit.prevent="login">
        <el-form-item label="账号"><el-input v-model="username" autocomplete="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="password" type="password" autocomplete="current-password" show-password /></el-form-item>
        <el-button type="primary" native-type="submit" style="width:100%" :loading="loading">登录</el-button>
      </el-form>
    </el-card>
  </div>
</template>
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api'
import { saveSession } from '../auth'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)

async function login() {
  loading.value = true
  try {
    const data = await http.post('/auth/login', {
      username: username.value,
      password: password.value,
      mode: 'local'
    })
    const clients = data.clients || []
    if (clients.length && !clients.includes('portal')) {
      ElMessage.error('当前账号无 Portal 访问权')
      return
    }
    saveSession(data)
    if (data.mustChangePassword) router.replace('/change-password')
    else router.replace('/')
  } finally {
    loading.value = false
  }
}
</script>
<style scoped>
.wrap{min-height:100vh;display:flex;align-items:center;justify-content:center;background:linear-gradient(160deg,#0b3d4a,#1f2933)}
.card{width:380px} h2{margin:0 0 16px;color:#0b3d4a}
</style>
