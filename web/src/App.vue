<template>
  <!-- 初始化引导页面（无布局） -->
  <router-view v-if="$route.meta.hideLayout" />

  <!-- 主应用布局 -->
  <el-container v-else class="app-container">
    <!-- 侧边栏 -->
    <el-aside width="200px" class="app-aside">
      <div class="logo">
        <h2>QUIC 命令管理</h2>
      </div>
      <el-menu
        :default-active="$route.path"
        router
        class="el-menu-vertical"
      >
        <el-menu-item index="/">
          <el-icon><Monitor /></el-icon>
          <span>客户端管理</span>
        </el-menu-item>
        <el-menu-item index="/command">
          <el-icon><DocumentAdd /></el-icon>
          <span>命令下发</span>
        </el-menu-item>
        <el-menu-item index="/history">
          <el-icon><Document /></el-icon>
          <span>命令历史</span>
        </el-menu-item>
        <el-menu-item index="/terminal">
          <el-icon><Monitor /></el-icon>
          <span>SSH 终端</span>
        </el-menu-item>
        <el-menu-item index="/audit">
          <el-icon><List /></el-icon>
          <span>命令审计</span>
        </el-menu-item>
        <el-menu-item index="/recordings">
          <el-icon><VideoCamera /></el-icon>
          <span>会话录像</span>
        </el-menu-item>
        <el-menu-item index="/release">
          <el-icon><Upload /></el-icon>
          <span>发布管理</span>
        </el-menu-item>
        <el-menu-item index="/setup" class="setup-menu-item">
          <el-icon><Setting /></el-icon>
          <span>数据库设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主内容区 -->
    <el-container>
      <!-- 顶部导航栏 -->
      <el-header class="app-header">
        <div class="header-content">
          <span class="header-title">{{ pageTitle }}</span>
          <div class="header-actions">
            <el-tag :type="dbStatus.type">
              <el-icon><component :is="dbStatus.icon" /></el-icon>
              {{ dbStatus.text }}
            </el-tag>
          </div>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="app-main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { request } from '@/api'

const route = useRoute()

// 数据库状态
const dbInitialized = ref(null) // null=检查中, true=已初始化, false=未初始化

const pageTitle = computed(() => {
  const titles = {
    '/': '客户端管理',
    '/command': '命令下发',
    '/history': '命令历史',
    '/terminal': 'SSH 终端',
    '/audit': '命令审计',
    '/recordings': '会话录像',
    '/release': '发布管理',
    '/setup': '数据库设置'
  }
  return titles[route.path] || 'QUIC 命令管理系统'
})

const dbStatus = computed(() => {
  if (dbInitialized.value === null) {
    return { type: 'info', text: '检查中...', icon: 'Loading' }
  } else if (dbInitialized.value) {
    return { type: 'success', text: '数据库已连接', icon: 'Connection' }
  } else {
    return { type: 'warning', text: '数据库未配置', icon: 'WarningFilled' }
  }
})

// 检查数据库状态
async function checkDatabaseStatus() {
  try {
    const res = await request.get('/setup/status')
    if (res.success) {
      dbInitialized.value = res.status.initialized
    }
  } catch (e) {
    dbInitialized.value = false
  }
}

onMounted(() => {
  checkDatabaseStatus()
})
</script>

<style scoped>
.app-container {
  height: 100vh;
}

.app-aside {
  background: #304156;
  color: #fff;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #263445;
}

.logo h2 {
  margin: 0;
  font-size: 18px;
  color: #fff;
}

.el-menu-vertical {
  border: none;
}

.app-header {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  padding: 0 20px;
}

.header-content {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 18px;
  font-weight: 500;
}

.app-main {
  background: #f0f2f5;
  padding: 20px;
}

/* 过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB',
    'Microsoft YaHei', Arial, sans-serif;
}
</style>
