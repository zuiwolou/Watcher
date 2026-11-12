<template>
  <div>
    <header class="top">
      <strong>危废管控 Portal</strong>
      <nav>
        <router-link to="/">工作台</router-link>
        <router-link v-if="has('alarm:view')" to="/alarms">
          告警
          <el-badge v-if="alarmBadge" :value="alarmBadge" class="badge" />
        </router-link>
        <router-link v-if="has('material:view')" to="/materials">物资</router-link>
        <router-link v-if="has('material:view')" to="/unique-codes">唯一码</router-link>
        <router-link v-if="has('inventory:view')" to="/inventory">盘点</router-link>
        <router-link v-if="has('inspection:view')" to="/inspection">巡检</router-link>
        <router-link v-if="has('monitor:view')" to="/monitor">监测</router-link>
        <router-link v-if="has('track:view')" to="/map">地图</router-link>
        <router-link v-if="has('geofence:view')" to="/geofences">围栏</router-link>
      </nav>
      <span class="ws">{{ wsLabel }}</span>
      <el-dropdown>
        <span class="user-link">{{ userLabel }} ▾</span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item @click="router.push('/profile')">个人中心</el-dropdown-item>
            <el-dropdown-item @click="router.push('/change-password')">修改密码</el-dropdown-item>
            <el-dropdown-item divided @click="logout">退出</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </header>
    <main class="main"><router-view /></main>
  </div>
</template>
<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { clearSession, hasPerm } from '../auth'
import { realtime } from '../realtime'
import http from '../api'

const router = useRouter()
const wsState = ref(realtime.getState())
const alarmBadge = ref(0)
let off

const wsLabel = computed(() => {
  const map = { open: '实时已连接', connecting: '实时连接中', closed: '实时断开', error: '实时异常', idle: '实时未启', 'no-token': '未登录' }
  return map[wsState.value] || wsState.value
})
const userLabel = computed(() => {
  try {
    const u = JSON.parse(localStorage.getItem('user') || '{}')
    return u.displayName || u.username || '用户'
  } catch { return '用户' }
})

function has(p) { return hasPerm(p) }

onMounted(() => {
  realtime.start()
  off = realtime.on((evt) => {
    if (evt.type === 'state') {
      wsState.value = evt.state
      if (evt.state === 'open') {
        // reconnect snapshot: refresh alarm badge from REST
        http.get('/alarms?status=raised&page=1&pageSize=1').then((d) => {
          alarmBadge.value = d?.total || 0
        }).catch(() => {})
      }
    }
    if (evt.type === 'message' && evt.envelope?.msgType === 'ALARM') {
      alarmBadge.value = Math.min(99, alarmBadge.value + 1)
    }
  })
})
onUnmounted(() => { off?.() })

async function logout() {
  realtime.stop()
  const rt = localStorage.getItem('refreshToken')
  try {
    await http.post('/auth/logout', { refreshToken: rt })
  } catch (_) { /* ignore */ }
  clearSession()
  router.push('/login')
}
</script>
<style scoped>
.top{display:flex;align-items:center;gap:24px;padding:12px 20px;background:#102a43;color:#fff}
nav{display:flex;gap:16px;flex:1;align-items:center;flex-wrap:wrap}
nav a{color:#9fb3c8;text-decoration:none;position:relative}
nav a.router-link-active{color:#2bb0ed}
.badge{margin-left:4px}
.ws{font-size:12px;opacity:.85}
.user-link{cursor:pointer;color:#d9e2ec}
.main{padding:16px 20px}
</style>
