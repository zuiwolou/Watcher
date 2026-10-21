<template>
  <div class="wrap">
    <el-card class="card">
      <h2>智能危废管控 · 前台</h2>
      <el-form @submit.prevent="login">
        <el-form-item label="账号"><el-input v-model="username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="password" type="password" /></el-form-item>
        <el-button type="primary" native-type="submit" style="width:100%">登录</el-button>
      </el-form>
    </el-card>
  </div>
</template>
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'
const router = useRouter()
const username = ref('admin')
const password = ref('Admin@123')
async function login() {
  const data = await http.post('/auth/login', { username: username.value, password: password.value, mode: 'local' })
  localStorage.setItem('accessToken', data.accessToken)
  localStorage.setItem('permissions', JSON.stringify(data.permissions || []))
  router.push('/')
}
</script>
<style scoped>
.wrap{min-height:100vh;display:flex;align-items:center;justify-content:center;background:linear-gradient(160deg,#0b3d4a,#1f2933)}
.card{width:380px} h2{margin:0 0 16px;color:#0b3d4a}
</style>
