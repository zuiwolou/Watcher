<template>
  <div>
    <header class="top">
      <strong>危废管控 Portal</strong>
      <nav>
        <router-link to="/">工作台</router-link>
        <router-link to="/alarms">告警</router-link>
        <router-link to="/materials">物资</router-link>
      </nav>
      <el-button link type="danger" @click="logout">退出</el-button>
    </header>
    <main class="main"><router-view /></main>
  </div>
</template>
<script setup>
import { useRouter } from 'vue-router'
const router = useRouter()
function logout() { localStorage.clear(); router.push('/login') }
</script>
<style scoped>
.top{display:flex;align-items:center;gap:24px;padding:12px 20px;background:#102a43;color:#fff}
nav{display:flex;gap:16px;flex:1}
nav a{color:#9fb3c8;text-decoration:none}
nav a.router-link-active{color:#2bb0ed}
.main{padding:20px}
</style>
