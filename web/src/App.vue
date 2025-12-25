<template>
  <el-container class="app-container">
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
      </el-menu>
    </el-aside>

    <!-- 主内容区 -->
    <el-container>
      <!-- 顶部导航栏 -->
      <el-header class="app-header">
        <div class="header-content">
          <span class="header-title">{{ pageTitle }}</span>
          <div class="header-actions">
            <el-tag type="success">
              <el-icon><Connection /></el-icon>
              服务器已连接
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
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const pageTitle = computed(() => {
  const titles = {
    '/': '客户端管理',
    '/command': '命令下发',
    '/history': '命令历史'
  }
  return titles[route.path] || 'QUIC 命令管理系统'
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
