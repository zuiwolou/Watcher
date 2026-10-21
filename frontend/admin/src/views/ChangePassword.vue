<template>
  <div class="wrap">
    <el-card style="width:400px">
      <h3>强制修改密码</h3>
      <el-form @submit.prevent="submit">
        <el-form-item label="原密码"><el-input v-model="oldPassword" type="password" /></el-form-item>
        <el-form-item label="新密码"><el-input v-model="newPassword" type="password" /></el-form-item>
        <el-button type="primary" native-type="submit">提交</el-button>
      </el-form>
    </el-card>
  </div>
</template>
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'
const router = useRouter()
const oldPassword = ref('')
const newPassword = ref('')
async function submit() {
  await http.post('/auth/change-password', { oldPassword: oldPassword.value, newPassword: newPassword.value })
  localStorage.removeItem('mustChangePassword')
  router.push('/')
}
</script>
<style scoped>
.wrap { min-height:100vh; display:flex; align-items:center; justify-content:center; }
</style>
