<template>
  <div :class="['terminal-page', { 'fullscreen': isFullscreen }]">
    <!-- 顶部工具栏 -->
    <div class="terminal-toolbar">
      <div class="toolbar-left">
        <!-- 状态筛选 -->
        <el-radio-group v-model="clientStatusFilter" @change="onStatusFilterChange" size="default" class="status-filter">
          <el-radio-button value="online">
            <el-icon><CircleCheck /></el-icon>
            在线
          </el-radio-button>
          <el-radio-button value="all">
            <el-icon><List /></el-icon>
            全部
          </el-radio-button>
        </el-radio-group>

        <!-- 客户端选择器 -->
        <el-select
          v-model="selectedClientId"
          placeholder="选择客户端"
          filterable
          remote
          :remote-method="searchClients"
          :loading="loadingClients"
          @visible-change="onSelectVisibleChange"
          @focus="onSelectFocus"
          size="default"
          class="client-select"
          :popper-class="'client-select-dropdown'"
        >
          <template #header>
            <div class="select-header">
              <span>共 {{ clientTotal }} 台设备</span>
              <el-link v-if="hasMoreClients" type="primary" @click="loadMoreClients" :loading="loadingMore">
                加载更多
              </el-link>
            </div>
          </template>
          <el-option
            v-for="client in clients"
            :key="client.client_id"
            :label="client.client_id"
            :value="client.client_id"
            :disabled="!client.online"
          >
            <div class="client-option">
              <div class="client-option-main">
                <el-tag :type="client.online ? 'success' : 'info'" size="small" effect="dark">
                  {{ client.online ? '●' : '○' }}
                </el-tag>
                <span class="client-id">{{ client.client_id }}</span>
              </div>
              <div class="client-option-sub">
                <span class="client-hostname">{{ client.hostname || '未知主机' }}</span>
                <span class="client-os">{{ client.os || '' }}</span>
              </div>
            </div>
          </el-option>
          <template #empty>
            <div class="select-empty">
              <el-empty description="暂无客户端" :image-size="60" />
            </div>
          </template>
        </el-select>
        <el-button @click="fetchClients" :loading="loadingClients" size="default" title="刷新列表">
          <el-icon><Refresh /></el-icon>
        </el-button>
        <el-button
          type="success"
          @click="openNewTerminal"
          :disabled="!selectedClientId"
          size="default"
        >
          <el-icon><Plus /></el-icon>
          新建终端
        </el-button>
      </div>
      <div class="toolbar-right">
        <el-button @click="toggleFullscreen" size="default">
          <el-icon>
            <FullScreen v-if="!isFullscreen" />
            <Close v-else />
          </el-icon>
          {{ isFullscreen ? '退出全屏' : '全屏' }}
        </el-button>
      </div>
    </div>

    <!-- 终端标签页 -->
    <div class="terminal-tabs-container">
      <el-tabs
        v-if="terminalTabs.length > 0"
        v-model="activeTabId"
        type="card"
        closable
        @tab-remove="closeTerminal"
        @tab-change="handleTabChange"
        class="terminal-tabs"
      >
        <el-tab-pane
          v-for="tab in terminalTabs"
          :key="tab.id"
          :label="tab.label"
          :name="tab.id"
        >
          <template #label>
            <span class="tab-label">
              <el-tag :type="tab.connected ? 'success' : 'info'" size="small" effect="dark">
                {{ tab.connected ? '●' : '○' }}
              </el-tag>
              {{ tab.clientId }}
            </span>
          </template>

          <!-- xterm.js 终端容器 -->
          <div class="xterm-container" :ref="el => setTerminalRef(tab.id, el)"></div>
        </el-tab-pane>
      </el-tabs>

      <!-- 空状态 -->
      <div v-if="terminalTabs.length === 0" class="terminals-wrapper">
        <div class="empty-state">
          <div class="empty-content">
            <!-- 主视觉区域 -->
            <div class="empty-hero">
              <div class="empty-icon-wrapper">
                <div class="icon-glow"></div>
                <el-icon class="empty-icon-main"><Connection /></el-icon>
              </div>
              <h2 class="empty-title">开始使用终端</h2>
              <p class="empty-description">选择客户端并点击「新建终端」开始连接远程服务器</p>
              <div class="empty-actions">
                <el-button
                  type="primary"
                  size="large"
                  @click="fetchClients"
                  :loading="loadingClients"
                  class="action-button-primary"
                >
                  <el-icon><Refresh /></el-icon>
                  刷新客户端列表
                </el-button>
              </div>
            </div>

            <!-- 使用说明 -->
            <div class="guide-card">
              <div class="guide-card-header">
                <div class="guide-header-icon">
                  <el-icon><Document /></el-icon>
                </div>
                <div class="guide-header-content">
                  <h3 class="guide-title">SSH 终端</h3>
                  <p class="guide-subtitle">完整的交互式终端体验</p>
                </div>
              </div>

              <div class="guide-card-body">
                <div class="guide-features">
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>交互式命令</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>Tab 补全</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>Vim/Nano 编辑器</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>快捷键支持</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Plus, FullScreen, Close, Connection, Document, CircleCheck, List } from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import api from '@/api'

// 状态
const selectedClientId = ref('')
const clients = ref([])
const loadingClients = ref(false)
const loadingMore = ref(false)
const clientStatusFilter = ref('online')
const clientTotal = ref(0)
const clientOffset = ref(0)
const clientPageSize = ref(50)
const searchQuery = ref('')
const isFullscreen = ref(false)
const activeTabId = ref('')
const terminalTabs = ref([])

// 存储终端实例和 WebSocket
const terminals = {}
const fitAddons = {}
const websockets = {}
const terminalRefs = {}

let tabCounter = 0

// 是否还有更多数据可加载
const hasMoreClients = computed(() => clients.value.length < clientTotal.value)

// 设置终端容器引用
function setTerminalRef(tabId, el) {
  if (el) {
    terminalRefs[tabId] = el
  }
}

// 获取客户端列表
async function fetchClients(reset = true) {
  if (reset) {
    clientOffset.value = 0
    clients.value = []
  }
  loadingClients.value = true
  try {
    const params = {
      status: clientStatusFilter.value,
      offset: clientOffset.value,
      limit: clientPageSize.value
    }
    const res = await api.getClients(params)
    const newClients = res.clients || []

    if (reset) {
      clients.value = newClients
    } else {
      clients.value.push(...newClients)
    }

    clientTotal.value = res.total || 0

    if (clients.value.length > 0 && !selectedClientId.value) {
      const firstOnline = clients.value.find(c => c.online)
      if (firstOnline) {
        selectedClientId.value = firstOnline.client_id
      }
    }
  } catch (error) {
    ElMessage.error('获取客户端列表失败: ' + error.message)
  } finally {
    loadingClients.value = false
  }
}

// 状态筛选变更
function onStatusFilterChange() {
  fetchClients(true)
}

// 搜索客户端
let searchTimer = null
function searchClients(query) {
  searchQuery.value = query
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  searchTimer = setTimeout(() => {
    clientOffset.value = 0
    clients.value = []
    loadClients()
  }, 300)
}

// 加载更多客户端
async function loadMoreClients() {
  if (loadingMore.value || !hasMoreClients.value) return
  loadingMore.value = true
  try {
    clientOffset.value += clientPageSize.value
    const params = {
      status: clientStatusFilter.value,
      offset: clientOffset.value,
      limit: clientPageSize.value
    }
    const res = await api.getClients(params)
    const newClients = res.clients || []
    clients.value.push(...newClients)
    clientTotal.value = res.total || 0
  } catch (error) {
    ElMessage.error('加载更多失败: ' + error.message)
  } finally {
    loadingMore.value = false
  }
}

// 下拉框显示/隐藏处理
function onSelectVisibleChange(visible) {
  if (visible && clients.value.length === 0) {
    fetchClients()
  }
}

// 下拉框聚焦处理
function onSelectFocus() {
  if (clients.value.length === 0) {
    fetchClients()
  }
}

// 加载客户端
async function loadClients() {
  loadingClients.value = true
  try {
    const params = {
      status: clientStatusFilter.value,
      offset: clientOffset.value,
      limit: clientPageSize.value
    }
    const res = await api.getClients(params)
    clients.value = res.clients || []
    clientTotal.value = res.total || 0

    if (clients.value.length > 0 && !selectedClientId.value) {
      const firstOnline = clients.value.find(c => c.online)
      if (firstOnline) {
        selectedClientId.value = firstOnline.client_id
      }
    }
  } catch (error) {
    ElMessage.error('搜索客户端失败: ' + error.message)
  } finally {
    loadingClients.value = false
  }
}

// 打开新终端
async function openNewTerminal() {
  if (!selectedClientId.value) {
    ElMessage.warning('请先选择客户端')
    return
  }

  const tabId = `tab-${++tabCounter}`
  const clientId = selectedClientId.value

  terminalTabs.value.push({
    id: tabId,
    clientId: clientId,
    label: clientId,
    connected: false
  })

  activeTabId.value = tabId

  await nextTick()

  // 初始化 xterm.js 终端
  initTerminal(tabId, clientId)
}

// 初始化 xterm.js 终端
function initTerminal(tabId, clientId) {
  const container = terminalRefs[tabId]
  if (!container) {
    console.error('[Terminal] 容器不存在:', tabId)
    return
  }

  // 创建终端实例
  const term = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 13,
    fontFamily: 'Menlo, Monaco, Consolas, "Ubuntu Mono", "DejaVu Sans Mono", "Liberation Mono", "Courier New", Courier, monospace',
    fontWeight: '400',
    fontWeightBold: '700',
    letterSpacing: 0,
    lineHeight: 1.0,
    theme: {
      background: '#0a0e27',
      foreground: '#d4d4d4',
      cursor: '#ffffff',
      cursorAccent: '#0a0e27',
      selectionBackground: 'rgba(59, 130, 246, 0.4)',
      black: '#000000',
      red: '#ef4444',
      green: '#10b981',
      yellow: '#f59e0b',
      blue: '#3b82f6',
      magenta: '#a855f7',
      cyan: '#06b6d4',
      white: '#d4d4d4',
      brightBlack: '#64748b',
      brightRed: '#f87171',
      brightGreen: '#34d399',
      brightYellow: '#fbbf24',
      brightBlue: '#60a5fa',
      brightMagenta: '#c084fc',
      brightCyan: '#22d3ee',
      brightWhite: '#ffffff'
    },
    allowProposedApi: true
  })

  // 添加插件
  const fitAddon = new FitAddon()
  const webLinksAddon = new WebLinksAddon()

  term.loadAddon(fitAddon)
  term.loadAddon(webLinksAddon)

  // 挂载终端
  term.open(container)

  // 适配容器大小
  setTimeout(() => {
    fitAddon.fit()
  }, 0)

  // 存储实例
  terminals[tabId] = term
  fitAddons[tabId] = fitAddon

  // 显示连接信息和风险提示
  term.writeln('\x1b[1;33m========================================\x1b[0m')
  term.writeln('\x1b[1;33m  WARNING / 警告\x1b[0m')
  term.writeln('\x1b[1;33m========================================\x1b[0m')
  term.writeln('')
  term.writeln('\x1b[33m本终端仅供授权用户使用。\x1b[0m')
  term.writeln('\x1b[33m所有操作将被记录和监控。\x1b[0m')
  term.writeln('\x1b[33m未经授权的访问将被追究法律责任。\x1b[0m')
  term.writeln('')
  term.writeln('\x1b[1;33m========================================\x1b[0m')
  term.writeln('')
  term.writeln('正在连接到 ' + clientId + '...')
  term.writeln('')

  // 连接 WebSocket
  connectWebSocket(tabId, clientId, term)
}

// 连接 WebSocket
function connectWebSocket(tabId, clientId, term) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/terminal/ws/${clientId}`

  const ws = new WebSocket(wsUrl)
  websockets[tabId] = ws

  const tab = terminalTabs.value.find(t => t.id === tabId)

  ws.onopen = () => {
    term.writeln('\x1b[32m[OK] WebSocket 已连接\x1b[0m')

    if (tab) {
      tab.connected = true
    }

    // 获取终端尺寸
    const fitAddon = fitAddons[tabId]
    let cols = 80
    let rows = 24
    if (fitAddon) {
      const dims = fitAddon.proposeDimensions()
      if (dims) {
        cols = dims.cols
        rows = dims.rows
      }
    }

    // 发送初始化消息
    ws.send(JSON.stringify({
      type: 'init',
      cols: cols,
      rows: rows
    }))
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)

      switch (msg.type) {
        case 'output':
          // 直接写入终端（xterm.js 会处理 ANSI 序列）
          term.write(msg.data)
          break
        case 'connected':
          term.writeln('\x1b[32m[OK] SSH 会话已建立\x1b[0m')
          term.writeln('\x1b[90mSession ID: ' + msg.data + '\x1b[0m')
          term.writeln('')
          if (tab) {
            tab.connected = true
            tab.sessionId = msg.data
          }
          break
        case 'disconnected':
          term.writeln('')
          term.writeln('\x1b[33m[WARN] 连接已断开: ' + msg.data + '\x1b[0m')
          if (tab) tab.connected = false
          break
        case 'error':
          term.writeln('')
          term.writeln('\x1b[31m[ERROR] ' + msg.data + '\x1b[0m')
          if (tab) tab.connected = false
          break
        case 'pong':
          // 心跳响应，忽略
          break
        default:
          console.warn('[Terminal] 未知消息类型:', msg.type)
      }
    } catch (e) {
      console.error('[Terminal] 解析消息失败:', e)
      // 尝试直接写入原始数据
      term.write(event.data)
    }
  }

  ws.onclose = (event) => {
    if (tab && tab.connected) {
      term.writeln('')
      term.writeln('\x1b[33m[WARN] WebSocket 连接已关闭 (Code: ' + event.code + ')\x1b[0m')
    }
    if (tab) tab.connected = false
  }

  ws.onerror = (error) => {
    term.writeln('')
    term.writeln('\x1b[31m[ERROR] WebSocket 连接错误\x1b[0m')
    if (tab) tab.connected = false
  }

  // 终端输入处理 - 实时发送每个字符
  term.onData((data) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'input',
        data: data
      }))
    }
  })

  // 终端大小变化处理
  term.onResize(({ cols, rows }) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'resize',
        cols: cols,
        rows: rows
      }))
    }
  })

  // 心跳
  const pingInterval = setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'ping' }))
    } else {
      clearInterval(pingInterval)
    }
  }, 30000)

  ws._pingInterval = pingInterval
}

// 关闭终端
function closeTerminal(tabId) {
  // 关闭 WebSocket
  const ws = websockets[tabId]
  if (ws) {
    if (ws._pingInterval) clearInterval(ws._pingInterval)
    ws.close()
    delete websockets[tabId]
  }

  // 销毁终端实例
  const term = terminals[tabId]
  if (term) {
    term.dispose()
    delete terminals[tabId]
  }

  // 清理 fitAddon
  delete fitAddons[tabId]
  delete terminalRefs[tabId]

  // 移除标签
  const index = terminalTabs.value.findIndex(t => t.id === tabId)
  if (index !== -1) {
    terminalTabs.value.splice(index, 1)
  }

  // 切换到其他标签
  if (activeTabId.value === tabId) {
    if (terminalTabs.value.length > 0) {
      activeTabId.value = terminalTabs.value[Math.max(0, index - 1)].id
    } else {
      activeTabId.value = ''
    }
  }
}

// 标签切换处理
function handleTabChange(tabId) {
  nextTick(() => {
    // 重新适配终端大小
    const fitAddon = fitAddons[tabId]
    if (fitAddon) {
      fitAddon.fit()
    }
    // 聚焦终端
    const term = terminals[tabId]
    if (term) {
      term.focus()
    }
  })
}

// 全屏切换
function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
  // 延迟重新适配所有终端大小
  nextTick(() => {
    setTimeout(() => {
      Object.keys(fitAddons).forEach(tabId => {
        const fitAddon = fitAddons[tabId]
        if (fitAddon) {
          fitAddon.fit()
        }
      })
    }, 100)
  })
}

// 窗口大小变化处理
function handleResize() {
  Object.keys(fitAddons).forEach(tabId => {
    const fitAddon = fitAddons[tabId]
    if (fitAddon) {
      fitAddon.fit()
    }
  })
}

// 键盘快捷键
function handleKeydown(e) {
  if (e.ctrlKey && e.shiftKey && e.key === 'T') {
    e.preventDefault()
    if (selectedClientId.value) {
      openNewTerminal()
    }
  }
  if (e.ctrlKey && e.shiftKey && e.key === 'W') {
    e.preventDefault()
    if (activeTabId.value) {
      closeTerminal(activeTabId.value)
    }
  }
  if (e.key === 'F11') {
    e.preventDefault()
    toggleFullscreen()
  }
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
  }
}

// 组件挂载
onMounted(() => {
  fetchClients()
  document.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', handleResize)
})

// 组件卸载
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', handleResize)

  // 清理所有终端
  terminalTabs.value.forEach(tab => {
    closeTerminal(tab.id)
  })
})

// 监听活动标签变化
watch(activeTabId, (newId) => {
  if (newId) {
    handleTabChange(newId)
  }
})
</script>

<style scoped>
.terminal-page {
  height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1a2e 50%, #16213e 100%);
  border-radius: 12px;
  margin: 8px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.terminal-page.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  height: 100vh;
  margin: 0;
  border-radius: 0;
  z-index: 9999;
}

.terminal-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: rgba(30, 30, 30, 0.9);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
  flex: 1;
}

.toolbar-right {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 状态筛选器 */
.status-filter {
  flex-shrink: 0;
}

:deep(.status-filter .el-radio-button__inner) {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
}

:deep(.status-filter .el-icon) {
  font-size: 14px;
}

/* 客户端选择器 */
.client-select {
  width: 280px;
  flex-shrink: 0;
}

.client-option {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 0;
}

.client-option-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

.client-option-main .client-id {
  font-weight: 600;
  color: #ffffff;
}

.client-option-sub {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-left: 24px;
  font-size: 12px;
}

.client-option-sub .client-hostname {
  color: #94a3b8;
}

.client-option-sub .client-os {
  color: #64748b;
}

.select-header {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  font-size: 12px;
  color: #94a3b8;
}

.select-empty {
  padding: 20px;
}

:deep(.client-select-dropdown) {
  background: rgba(18, 18, 18, 0.95) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
}

/* 标签页容器 */
.terminal-tabs-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.terminal-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

:deep(.terminal-tabs .el-tabs__header) {
  margin: 0;
  background: rgba(18, 18, 18, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;
}

:deep(.terminal-tabs .el-tabs__nav-wrap::after),
:deep(.terminal-tabs .el-tabs__nav-wrap::before) {
  display: none;
}

:deep(.terminal-tabs .el-tabs__item) {
  color: #94a3b8;
  border: none !important;
  background: transparent;
  height: 36px;
  line-height: 36px;
  padding: 0 20px;
  border-radius: 8px 8px 0 0;
  margin-right: 4px;
}

:deep(.terminal-tabs .el-tabs__item:hover) {
  color: #ffffff;
  background: rgba(59, 130, 246, 0.1);
}

:deep(.terminal-tabs .el-tabs__item.is-active) {
  color: #ffffff;
  background: rgba(59, 130, 246, 0.15);
}

:deep(.terminal-tabs .el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

:deep(.terminal-tabs .el-tab-pane) {
  height: 100%;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tab-label :deep(.el-tag) {
  border: none;
  background: transparent;
  padding: 0;
}

.tab-label :deep(.el-tag--success) {
  color: #10b981;
}

.tab-label :deep(.el-tag--info) {
  color: #64748b;
}

/* xterm.js 容器 */
.xterm-container {
  width: 100%;
  height: 100%;
  padding: 8px;
  box-sizing: border-box;
  background: #0a0e27;
}

:deep(.xterm) {
  height: 100%;
}

:deep(.xterm .xterm-screen) {
  font-family: Menlo, Monaco, Consolas, "Ubuntu Mono", "DejaVu Sans Mono", "Liberation Mono", "Courier New", Courier, monospace !important;
}

:deep(.xterm-viewport) {
  overflow-y: auto !important;
}

:deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}

:deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: rgba(59, 130, 246, 0.3);
  border-radius: 4px;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: rgba(59, 130, 246, 0.5);
}

/* 终端包装器 */
.terminals-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 0;
  background: rgba(0, 0, 0, 0.3);
}

/* 空状态 */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  overflow-y: auto;
  width: 100%;
  height: 100%;
}

.empty-content {
  max-width: 700px;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.empty-hero {
  text-align: center;
  padding: 24px;
}

.empty-icon-wrapper {
  position: relative;
  display: inline-flex;
  margin-bottom: 16px;
}

.icon-glow {
  position: absolute;
  width: 80px;
  height: 80px;
  background: radial-gradient(circle, rgba(59, 130, 246, 0.2) 0%, transparent 70%);
  border-radius: 50%;
  animation: pulse-glow 2s ease-in-out infinite;
}

@keyframes pulse-glow {
  0%, 100% { opacity: 0.4; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(1.05); }
}

.empty-icon-main {
  position: relative;
  font-size: 48px;
  color: #3b82f6;
}

.empty-title {
  font-size: 24px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 8px 0;
}

.empty-description {
  font-size: 14px;
  color: #94a3b8;
  margin: 0 0 20px 0;
}

.empty-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.guide-card {
  background: rgba(30, 30, 30, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  overflow: hidden;
}

.guide-card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(16, 185, 129, 0.1) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.guide-header-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(59, 130, 246, 0.2);
  border-radius: 10px;
}

.guide-header-icon .el-icon {
  font-size: 20px;
  color: #3b82f6;
}

.guide-title {
  font-size: 18px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 4px 0;
}

.guide-subtitle {
  font-size: 13px;
  color: #94a3b8;
  margin: 0;
}

.guide-card-body {
  padding: 20px;
}

.guide-features {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.feature-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 8px;
  font-size: 13px;
  color: #10b981;
}

.feature-badge .el-icon {
  font-size: 14px;
}

/* Element Plus 组件深色主题 */
:deep(.el-select) {
  --el-fill-color-blank: rgba(30, 30, 30, 0.8);
  --el-text-color-regular: #ffffff;
  --el-border-color: rgba(255, 255, 255, 0.1);
}

:deep(.el-input__wrapper) {
  background-color: rgba(30, 30, 30, 0.8);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

:deep(.el-input__inner) {
  color: #ffffff;
}

:deep(.el-button) {
  --el-button-bg-color: rgba(30, 30, 30, 0.8);
  --el-button-border-color: rgba(255, 255, 255, 0.1);
  --el-button-text-color: #d4d4d4;
  --el-button-hover-bg-color: rgba(59, 130, 246, 0.15);
  --el-button-hover-border-color: rgba(59, 130, 246, 0.3);
  --el-button-hover-text-color: #ffffff;
}

:deep(.el-button--success) {
  --el-button-bg-color: #10b981;
  --el-button-border-color: #10b981;
  --el-button-text-color: #ffffff;
}

:deep(.el-button--success:disabled) {
  --el-button-bg-color: rgba(30, 30, 30, 0.5);
  --el-button-border-color: rgba(255, 255, 255, 0.05);
  --el-button-text-color: #64748b;
}

:deep(.el-radio-button__inner) {
  background: rgba(30, 30, 30, 0.8);
  border-color: rgba(255, 255, 255, 0.1);
  color: #d4d4d4;
}

:deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background-color: #3b82f6;
  border-color: #3b82f6;
  color: #ffffff;
}

:deep(.el-select-dropdown) {
  background: rgba(18, 18, 18, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

:deep(.el-select-dropdown__item) {
  color: #d4d4d4;
}

:deep(.el-select-dropdown__item:hover) {
  background: rgba(59, 130, 246, 0.15);
}

:deep(.el-select-dropdown__item.is-selected) {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
}

:deep(.el-select-dropdown__item.is-disabled) {
  opacity: 0.5;
}

:deep(.empty-description) {
  color: #94a3b8;
}
</style>
