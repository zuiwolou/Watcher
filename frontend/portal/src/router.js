import { createRouter, createWebHistory } from 'vue-router'
import { mustChangePassword } from './auth'
import Login from './views/Login.vue'
import ChangePassword from './views/ChangePassword.vue'
import Shell from './views/Shell.vue'
import Workbench from './views/Workbench.vue'
import Alarms from './views/Alarms.vue'
import Materials from './views/Materials.vue'
import Monitor from './views/Monitor.vue'
import MapView from './views/MapView.vue'
import Geofence from './views/Geofence.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/change-password', component: ChangePassword },
    {
      path: '/',
      component: Shell,
      children: [
        { path: '', component: Workbench },
        { path: 'alarms', component: Alarms },
        { path: 'materials', component: Materials },
        { path: 'monitor', component: Monitor },
        { path: 'map', component: MapView },
        { path: 'geofences', component: Geofence }
      ]
    }
  ]
})

router.beforeEach((to) => {
  const authed = !!localStorage.getItem('accessToken')
  if (to.path === '/login') {
    if (authed && mustChangePassword()) return '/change-password'
    if (authed) return '/'
    return true
  }
  if (!authed) return '/login'
  if (mustChangePassword() && to.path !== '/change-password') return '/change-password'
  if (to.path === '/change-password' && !mustChangePassword()) return '/'
  return true
})

export default router
