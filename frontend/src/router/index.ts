import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Keys from '../views/Keys.vue'
import Dashboard from '../views/Dashboard.vue'
const router = createRouter({ history: createWebHistory(), routes: [
  { path: '/login', component: Login, meta: { public: true } },
  { path: '/', component: Dashboard },
  { path: '/keys', component: Keys },
] })
router.beforeEach(to => {
  const token = localStorage.getItem('keymgr_access_token')
  if (!to.meta.public && !token) return '/login'
  if (to.path === '/login' && token) return '/'
})
export default router
