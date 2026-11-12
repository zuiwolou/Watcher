<template>
  <el-container style="min-height:100vh">
    <el-aside width="220px" class="aside">
      <div class="brand">危废管控 Admin</div>
      <el-menu router :default-active="$route.path" background-color="#102a43" text-color="#d9e2ec" active-text-color="#2bb0ed">
        <el-menu-item index="/">概览</el-menu-item>
        <el-menu-item v-if="has('user:view')" index="/users">人员</el-menu-item>
        <el-menu-item v-if="has('role:view')" index="/roles">角色矩阵</el-menu-item>
        <el-menu-item v-if="has('audit:view')" index="/audit">审计</el-menu-item>
        <el-menu-item v-if="has('material:view')" index="/materials">物资</el-menu-item>
        <el-menu-item v-if="has('material:view')" index="/unique-codes">唯一码</el-menu-item>
        <el-menu-item v-if="has('material:view')" index="/stocks">库存</el-menu-item>
        <el-menu-item v-if="has('inbound:edit')" index="/inbound">入库</el-menu-item>
        <el-menu-item v-if="has('outbound:edit')" index="/outbound">出库</el-menu-item>
        <el-menu-item v-if="has('approval:view')" index="/approvals">审批</el-menu-item>
        <el-menu-item v-if="has('transport:view')" index="/transport">运输</el-menu-item>
        <el-menu-item v-if="has('inventory:view')" index="/inventory">盘点</el-menu-item>
        <el-menu-item v-if="has('inspection:view')" index="/inspection">巡检</el-menu-item>
        <el-menu-item v-if="has('alarm:view')" index="/alarms">告警</el-menu-item>
        <el-menu-item v-if="has('threshold:edit')" index="/thresholds">阈值</el-menu-item>
        <el-menu-item v-if="has('geofence:view')" index="/geofences">地图围栏</el-menu-item>
        <el-menu-item v-if="has('device:edit')" index="/devices">设备台账</el-menu-item>
        <el-menu-item v-if="has('monitor:view')" index="/history">历史</el-menu-item>
        <el-menu-item v-if="has('mqtt:schema:edit')" index="/mqtt-schema">MQTT Schema</el-menu-item>
        <el-menu-item v-if="has('openapi:config')" index="/integrations">监管对接</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <el-dropdown>
          <span class="user-link">{{ user.displayName || user.username || '用户' }} ▾</span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="router.push('/profile')">个人中心</el-dropdown-item>
              <el-dropdown-item @click="router.push('/change-password')">修改密码</el-dropdown-item>
              <el-dropdown-item divided @click="logout">退出</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main><router-view /></el-main>
    </el-container>
  </el-container>
</template>
<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { hasPerm, clearSession } from '../auth'
import http from '../api'
const router = useRouter()
const user = computed(() => {
  try { return JSON.parse(localStorage.getItem('user') || '{}') } catch { return {} }
})
function has(p) { return hasPerm(p) }
async function logout() {
  const rt = localStorage.getItem('refreshToken')
  try { await http.post('/auth/logout', { refreshToken: rt }) } catch (_) { /* ignore */ }
  clearSession()
  router.push('/login')
}
</script>
<style scoped>
.aside { background:#102a43; }
.brand { color:#fff; padding:16px; font-weight:700; }
.header { display:flex; justify-content:flex-end; align-items:center; gap:12px; border-bottom:1px solid #eee; }
.user-link { cursor:pointer; color:#334e68; }
</style>
