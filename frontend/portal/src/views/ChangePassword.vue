<template>
  <div class="wrap">
    <el-card style="width:420px">
      <h3>强制修改密码</h3>
      <p class="tip">首次登录或管理员要求改密后，需设置新密码才能继续使用 Portal。</p>
      <el-form label-width="90px" @submit.prevent="submit">
        <el-form-item label="原密码"><el-input v-model="oldPassword" type="password" show-password /></el-form-item>
        <el-form-item label="新密码"><el-input v-model="newPassword" type="password" show-password /></el-form-item>
        <el-form-item label="确认密码"><el-input v-model="confirm" type="password" show-password /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">提交并进入系统</el-button>
      </el-form>
    </el-card>
  </div>
</template>
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api'

const router = useRouter()
const oldPassword = ref('')
const newPassword = ref('')
const confirm = ref('')
const loading = ref(false)

async function submit() {
  if (newPassword.value.length < 8) {
    ElMessage.warning('新密码至少 8 位')
    return
  }
  if (newPassword.value !== confirm.value) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  loading.value = true
  try {
    await http.post('/auth/change-password', {
      oldPassword: oldPassword.value,
      newPassword: newPassword.value
    })
    localStorage.removeItem('mustChangePassword')
    ElMessage.success('密码已更新')
    router.replace('/')
  } finally {
    loading.value = false
  }
}
</script>
<style scoped>
.wrap { min-height:100vh; display:flex; align-items:center; justify-content:center; background:linear-gradient(160deg,#0b3d4a,#1f2933); }
.tip { color:#627d98; font-size:13px; margin:0 0 12px; }
h3 { margin:0 0 8px; color:#0b3d4a; }
</style>
