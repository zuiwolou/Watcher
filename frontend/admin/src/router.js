import { createRouter, createWebHistory } from 'vue-router'
import { hasPerm } from './auth'
import Login from './views/Login.vue'
import Layout from './views/Layout.vue'
import Dashboard from './views/Dashboard.vue'
import Users from './views/Users.vue'
import Roles from './views/Roles.vue'
import Materials from './views/Materials.vue'
import Inbound from './views/Inbound.vue'
import Outbound from './views/Outbound.vue'
import Approvals from './views/Approvals.vue'
import Transport from './views/Transport.vue'
import Inventory from './views/Inventory.vue'
import Inspection from './views/Inspection.vue'
import Alarms from './views/Alarms.vue'
import Thresholds from './views/Thresholds.vue'
import Geofences from './views/Geofences.vue'
import Stocks from './views/Stocks.vue'
import History from './views/History.vue'
import Audit from './views/Audit.vue'
import MqttSchema from './views/MqttSchema.vue'
import Integrations from './views/Integrations.vue'
import ChangePassword from './views/ChangePassword.vue'

const routes = [
  { path: '/login', component: Login },
  { path: '/change-password', component: ChangePassword },
  {
    path: '/',
    component: Layout,
    children: [
      { path: '', component: Dashboard },
      { path: 'users', component: Users, meta: { perm: 'user:view' } },
      { path: 'roles', component: Roles, meta: { perm: 'role:view' } },
      { path: 'materials', component: Materials, meta: { perm: 'material:view' } },
      { path: 'stocks', component: Stocks, meta: { perm: 'material:view' } },
      { path: 'inbound', component: Inbound, meta: { perm: 'inbound:edit' } },
      { path: 'outbound', component: Outbound, meta: { perm: 'outbound:edit' } },
      { path: 'approvals', component: Approvals, meta: { perm: 'approval:view' } },
      { path: 'transport', component: Transport, meta: { perm: 'transport:view' } },
      { path: 'inventory', component: Inventory, meta: { perm: 'inventory:view' } },
      { path: 'inspection', component: Inspection, meta: { perm: 'inspection:view' } },
      { path: 'alarms', component: Alarms, meta: { perm: 'alarm:view' } },
      { path: 'thresholds', component: Thresholds, meta: { perm: 'threshold:edit' } },
      { path: 'geofences', component: Geofences, meta: { perm: 'geofence:view' } },
      { path: 'history', component: History, meta: { perm: 'monitor:view' } },
      { path: 'audit', component: Audit, meta: { perm: 'audit:view' } },
      { path: 'mqtt-schema', component: MqttSchema, meta: { perm: 'mqtt:schema:edit' } },
      { path: 'integrations', component: Integrations, meta: { perm: 'openapi:config' } }
    ]
  }
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  if (to.path === '/login') return true
  if (!localStorage.getItem('accessToken')) return '/login'
  if (localStorage.getItem('mustChangePassword') === '1' && to.path !== '/change-password') {
    return '/change-password'
  }
  if (to.meta.perm && !hasPerm(to.meta.perm)) return '/'
  return true
})

export default router
