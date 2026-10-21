<template>
  <el-container style="min-height:100vh">
    <el-aside width="220px" class="aside">
      <div class="brand">危废管控 Admin</div>
      <el-menu router :default-active="$route.path" background-color="#102a43" text-color="#d9e2ec" active-text-color="#2bb0ed">
        <el-menu-item index="/">概览</el-menu-item>
        <el-menu-item v-if="has('user:view')" index="/users">人员</el-menu-item>
        <el-menu-item v-if="has('material:view')" index="/materials">物资</el-menu-item>
        <el-menu-item v-if="has('inbound:edit')" index="/inbound">入库</el-menu-item>
        <el-menu-item v-if="has('approval:view')" index="/approvals">审批</el-menu-item>
        <el-menu-item v-if="has('transport:view')" index="/transport">运输</el-menu-item>
        <el-menu-item v-if="has('inventory:view')" index="/inventory">盘点</el-menu-item>
        <el-menu-item v-if="has('inspection:view')" index="/inspection">巡检</el-menu-item>
        <el-menu-item v-if="has('alarm:view')" index="/alarms">告警</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span>{{ user.displayName || user.username }}</span>
        <el-button link type="danger" @click="logout">退出</el-button>
      </el-header>
      <el-main><router-view /></el-main>
    </el-container>
  </el-container>
</template>
<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { hasPerm, clearSession } from '../auth'
const router = useRouter()
const user = computed(() => {
  try { return JSON.parse(localStorage.getItem('user') || '{}') } catch { return {} }
})
function has(p) { return hasPerm(p) }
function logout() { clearSession(); router.push('/login') }
</script>
<style scoped>
.aside { background:#102a43; }
.brand { color:#fff; padding:16px; font-weight:700; }
.header { display:flex; justify-content:flex-end; align-items:center; gap:12px; border-bottom:1px solid #eee; }
</style>
