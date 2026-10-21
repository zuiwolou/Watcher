import { createRouter, createWebHistory } from 'vue-router'
import { hasPerm } from './auth'
import Login from './views/Login.vue'
import Layout from './views/Layout.vue'
import Dashboard from './views/Dashboard.vue'
import Users from './views/Users.vue'
import Materials from './views/Materials.vue'
import Inbound from './views/Inbound.vue'
import Approvals from './views/Approvals.vue'
import Transport from './views/Transport.vue'
import Inventory from './views/Inventory.vue'
import Inspection from './views/Inspection.vue'
import Alarms from './views/Alarms.vue'
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
      { path: 'materials', component: Materials, meta: { perm: 'material:view' } },
      { path: 'inbound', component: Inbound, meta: { perm: 'inbound:edit' } },
      { path: 'approvals', component: Approvals, meta: { perm: 'approval:view' } },
      { path: 'transport', component: Transport, meta: { perm: 'transport:view' } },
      { path: 'inventory', component: Inventory, meta: { perm: 'inventory:view' } },
      { path: 'inspection', component: Inspection, meta: { perm: 'inspection:view' } },
      { path: 'alarms', component: Alarms, meta: { perm: 'alarm:view' } }
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
