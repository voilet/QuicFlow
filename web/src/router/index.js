import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', hideLayout: true, public: true }
  },
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/Setup.vue'),
    meta: { title: '初始化向导', hideLayout: true, public: true }
  },
  {
    path: '/',
    name: 'ClientList',
    component: () => import('@/views/ClientList.vue'),
    meta: { title: '客户端列表' }
  },
  {
    path: '/command',
    name: 'CommandSend',
    component: () => import('@/views/CommandSend.vue'),
    meta: { title: '命令发送' }
  },
  {
    path: '/history',
    name: 'CommandHistory',
    component: () => import('@/views/CommandHistory.vue'),
    meta: { title: '命令历史' }
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
  },
  {
    path: '/release',
    name: 'Release',
    component: () => import('@/views/Release.vue'),
    meta: { title: '发布管理' }
  },
  {
    path: '/profiling',
    name: 'Profiling',
    component: () => import('@/views/Profiling.vue'),
    meta: { title: '性能分析' }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// 白名单路由
const whiteList = ['/login', '/setup']

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()

  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} - QUIC Flow`
  }

  const hasToken = userStore.token

  if (hasToken) {
    if (to.path === '/login') {
      // 已登录则跳转到首页
      next({ path: '/' })
    } else {
      // 有 token 就允许访问，不强制获取用户信息
      // 如果需要刷新用户信息，可以在页面组件中自行调用
      next()
    }
  } else {
    // 未登录
    if (whiteList.includes(to.path) || to.meta.public) {
      next()
    } else {
      next(`/login?redirect=${to.path}`)
    }
  }
})

export default router
