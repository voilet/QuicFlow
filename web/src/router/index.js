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
  },
  {
    path: '/terminal',
    name: 'Terminal',
    component: () => import('@/views/Terminal.vue'),
    meta: { title: 'SSH 终端' }
  },
  {
    path: '/audit',
    name: 'AuditLog',
    component: () => import('@/views/AuditLog.vue'),
    meta: { title: '命令审计' }
  },
  {
    path: '/recordings',
    name: 'Recordings',
    component: () => import('@/views/Recordings.vue'),
    meta: { title: '会话录像' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
