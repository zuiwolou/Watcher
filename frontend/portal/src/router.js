import { createRouter, createWebHistory } from 'vue-router'
import { mustChangePassword, hasPerm } from './auth'
import Login from './views/Login.vue'
import ChangePassword from './views/ChangePassword.vue'
import Shell from './views/Shell.vue'
import Workbench from './views/Workbench.vue'
import Alarms from './views/Alarms.vue'
import Materials from './views/Materials.vue'
import UniqueCodes from './views/UniqueCodes.vue'
import Monitor from './views/Monitor.vue'
import MapView from './views/MapView.vue'
import Geofence from './views/Geofence.vue'
import InventoryReport from './views/InventoryReport.vue'
import InspectionReport from './views/InspectionReport.vue'
import Forbidden from './views/Forbidden.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/change-password', component: ChangePassword },
    { path: '/403', component: Forbidden },
    {
      path: '/',
      component: Shell,
      children: [
        { path: '', component: Workbench },
        { path: 'alarms', component: Alarms, meta: { perm: 'alarm:view' } },
        { path: 'materials', component: Materials, meta: { perm: 'material:view' } },
        { path: 'unique-codes', component: UniqueCodes, meta: { perm: 'material:view' } },
        { path: 'inventory', component: InventoryReport, meta: { perm: 'inventory:view' } },
        { path: 'inspection', component: InspectionReport, meta: { perm: 'inspection:view' } },
        { path: 'monitor', component: Monitor, meta: { perm: 'monitor:view' } },
        { path: 'map', component: MapView, meta: { perm: 'track:view' } },
        { path: 'geofences', component: Geofence, meta: { perm: 'geofence:view' } }
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
  if (to.path === '/403') return true
  if (!authed) return '/login'
  if (mustChangePassword() && to.path !== '/change-password') return '/change-password'
  if (to.path === '/change-password' && !mustChangePassword()) return '/'
  if (to.meta.perm && !hasPerm(to.meta.perm)) return '/403'
  return true
})

export default router
