import { createRouter, createWebHistory } from 'vue-router'
import Login from './views/Login.vue'
import Shell from './views/Shell.vue'
import Workbench from './views/Workbench.vue'
import Alarms from './views/Alarms.vue'
import Materials from './views/Materials.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    {
      path: '/',
      component: Shell,
      children: [
        { path: '', component: Workbench },
        { path: 'alarms', component: Alarms },
        { path: 'materials', component: Materials }
      ]
    }
  ]
})

router.beforeEach((to) => {
  if (to.path !== '/login' && !localStorage.getItem('accessToken')) return '/login'
  return true
})

export default router
