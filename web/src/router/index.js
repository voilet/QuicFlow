import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'ClientList',
    component: () => import('@/views/ClientList.vue')
  },
  {
    path: '/command',
    name: 'CommandSend',
    component: () => import('@/views/CommandSend.vue')
  },
  {
    path: '/history',
    name: 'CommandHistory',
    component: () => import('@/views/CommandHistory.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
