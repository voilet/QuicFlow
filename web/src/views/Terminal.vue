<template>
  <div :class="['terminal-page', { 'fullscreen': isFullscreen }]">
    <!-- 顶部工具栏 -->
    <div class="terminal-toolbar">
      <div class="toolbar-left">
        <el-select
          v-model="selectedClientId"
          placeholder="选择客户端"
          filterable
          size="default"
          style="width: 200px"
        >
          <el-option
            v-for="client in clients"
            :key="client.client_id"
            :label="client.client_id"
            :value="client.client_id"
          >
            <span>{{ client.client_id }}</span>
            <span style="float: right; color: #909399; font-size: 12px">
              {{ client.uptime }}
            </span>
          </el-option>
        </el-select>
        <el-button @click="fetchClients" :loading="loadingClients" size="default">
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
        </el-tab-pane>
      </el-tabs>

      <!-- 终端容器 -->
      <div class="terminals-wrapper">
        <div
          v-for="tab in terminalTabs"
          :key="tab.id"
          :ref="el => setTerminalRef(tab.id, el)"
          :class="['terminal-container', { 'active': activeTabId === tab.id }]"
        ></div>

        <!-- 空状态 -->
        <div v-if="terminalTabs.length === 0" class="empty-state">
          <el-empty description="选择客户端并点击「新建终端」开始">
            <el-button type="primary" @click="fetchClients">刷新客户端列表</el-button>
          </el-empty>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { ElMessage } from 'element-plus'
import { Refresh, Plus, FullScreen, Close } from '@element-plus/icons-vue'
import api from '@/api'

// 状态
const selectedClientId = ref('')
const clients = ref([])
const loadingClients = ref(false)
const isFullscreen = ref(false)
const activeTabId = ref('')
const terminalTabs = ref([])
const terminalRefs = ref({})

// 终端实例存储
const terminals = {}
const fitAddons = {}
const websockets = {}
const resizeObservers = {}

let tabCounter = 0

// 设置终端容器引用
function setTerminalRef(id, el) {
  if (el) {
    terminalRefs.value[id] = el
  }
}

// 获取客户端列表
async function fetchClients() {
  loadingClients.value = true
  try {
    const res = await api.getClients()
    clients.value = res.clients || []
    if (clients.value.length > 0 && !selectedClientId.value) {
      selectedClientId.value = clients.value[0].client_id
    }
  } catch (error) {
    ElMessage.error('获取客户端列表失败: ' + error.message)
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

  // 创建新标签
  terminalTabs.value.push({
    id: tabId,
    clientId: clientId,
    label: clientId,
    connected: false,
    sessionId: ''
  })

  activeTabId.value = tabId

  // 等待 DOM 更新
  await nextTick()

  // 初始化终端
  initTerminal(tabId, clientId)
}

// 初始化终端
function initTerminal(tabId, clientId) {
  const container = terminalRefs.value[tabId]
  if (!container) {
    console.error('Terminal container not found:', tabId)
    return
  }

  const terminal = new Terminal({
    fontFamily: 'Monaco, Menlo, "Courier New", monospace',
    fontSize: 14,
    lineHeight: 1.2,
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#f0f0f0',
      cursorAccent: '#1e1e1e',
      selectionBackground: 'rgba(255, 255, 255, 0.3)',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#ffffff'
    },
    cursorBlink: true,
    scrollback: 10000,
    tabStopWidth: 4,
    allowProposedApi: true
  })

  const fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  terminal.open(container)

  terminals[tabId] = terminal
  fitAddons[tabId] = fitAddon

  // 多次 fit 确保尺寸正确
  const doFit = () => {
    if (fitAddons[tabId]) {
      fitAddons[tabId].fit()
    }
  }

  // 立即 fit
  doFit()
  // 延迟 fit 确保容器完全渲染
  setTimeout(doFit, 50)
  setTimeout(doFit, 200)
  setTimeout(doFit, 500)

  // 监听容器大小变化（带防抖）
  let resizeTimeout = null
  const resizeObserver = new ResizeObserver(() => {
    if (fitAddons[tabId] && activeTabId.value === tabId) {
      if (resizeTimeout) clearTimeout(resizeTimeout)
      resizeTimeout = setTimeout(() => {
        fitAddons[tabId].fit()
        sendResize(tabId)
      }, 100)
    }
  })
  resizeObserver.observe(container)
  resizeObservers[tabId] = resizeObserver

  // 处理终端输入
  terminal.onData(data => {
    const ws = websockets[tabId]
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    }
  })

  terminal.writeln('\x1b[33m欢迎使用 SSH 终端\x1b[0m')
  terminal.writeln(`正在连接到 ${clientId}...`)
  terminal.writeln('')

  // 连接 WebSocket
  connectWebSocket(tabId, clientId)
}

// 连接 WebSocket
function connectWebSocket(tabId, clientId) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/terminal/ws/${clientId}`

  const ws = new WebSocket(wsUrl)
  websockets[tabId] = ws

  const terminal = terminals[tabId]
  const tab = terminalTabs.value.find(t => t.id === tabId)

  ws.onopen = () => {
    terminal.writeln('\x1b[32mWebSocket 连接已建立\x1b[0m')
    // 立即发送初始终端大小，服务器会等待这个消息
    if (fitAddons[tabId]) {
      fitAddons[tabId].fit()
    }
    ws.send(JSON.stringify({
      type: 'init',
      cols: terminal.cols,
      rows: terminal.rows
    }))
    terminal.writeln(`终端大小: ${terminal.cols}x${terminal.rows}`)
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      switch (msg.type) {
        case 'output':
          terminal.write(msg.data)
          break
        case 'connected':
          if (tab) {
            tab.connected = true
            tab.sessionId = msg.data
          }
          terminal.writeln('\x1b[32mSSH 会话已建立\x1b[0m')
          terminal.writeln('')
          // 多次发送 resize 确保后端正确接收尺寸
          const fitAndResize = () => {
            if (fitAddons[tabId]) {
              fitAddons[tabId].fit()
            }
            sendResize(tabId)
          }
          fitAndResize()
          setTimeout(fitAndResize, 100)
          setTimeout(fitAndResize, 300)
          setTimeout(fitAndResize, 500)
          // 聚焦终端
          terminal.focus()
          break
        case 'disconnected':
          terminal.writeln(`\r\n\x1b[33m连接已断开: ${msg.data}\x1b[0m`)
          if (tab) tab.connected = false
          break
        case 'error':
          terminal.writeln(`\r\n\x1b[31m错误: ${msg.data}\x1b[0m`)
          if (tab) tab.connected = false
          break
        case 'pong':
          break
      }
    } catch (e) {
      console.error('解析消息失败:', e)
    }
  }

  ws.onclose = () => {
    if (tab && tab.connected) {
      terminal.writeln('\r\n\x1b[33mWebSocket 连接已关闭\x1b[0m')
    }
    if (tab) tab.connected = false
  }

  ws.onerror = (error) => {
    terminal.writeln('\r\n\x1b[31m连接错误\x1b[0m')
    console.error('WebSocket error:', error)
    if (tab) tab.connected = false
  }

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

// 发送终端大小
function sendResize(tabId) {
  const ws = websockets[tabId]
  const terminal = terminals[tabId]
  if (ws && ws.readyState === WebSocket.OPEN && terminal) {
    ws.send(JSON.stringify({
      type: 'resize',
      cols: terminal.cols,
      rows: terminal.rows
    }))
  }
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

  // 销毁终端
  const terminal = terminals[tabId]
  if (terminal) {
    terminal.dispose()
    delete terminals[tabId]
  }

  // 清理 fitAddon
  delete fitAddons[tabId]

  // 清理 ResizeObserver
  const resizeObserver = resizeObservers[tabId]
  if (resizeObserver) {
    resizeObserver.disconnect()
    delete resizeObservers[tabId]
  }

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
    const fitAddon = fitAddons[tabId]
    if (fitAddon) {
      fitAddon.fit()
      sendResize(tabId)
    }
    // 聚焦终端
    const terminal = terminals[tabId]
    if (terminal) {
      terminal.focus()
    }
  })
}

// 全屏切换
function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
  nextTick(() => {
    // 重新计算所有终端大小
    Object.keys(fitAddons).forEach(tabId => {
      if (fitAddons[tabId]) {
        fitAddons[tabId].fit()
        sendResize(tabId)
      }
    })
  })
}

// 键盘快捷键
function handleKeydown(e) {
  // Ctrl+Shift+T: 新建终端
  if (e.ctrlKey && e.shiftKey && e.key === 'T') {
    e.preventDefault()
    if (selectedClientId.value) {
      openNewTerminal()
    }
  }
  // Ctrl+Shift+W: 关闭当前终端
  if (e.ctrlKey && e.shiftKey && e.key === 'W') {
    e.preventDefault()
    if (activeTabId.value) {
      closeTerminal(activeTabId.value)
    }
  }
  // F11: 全屏
  if (e.key === 'F11') {
    e.preventDefault()
    toggleFullscreen()
  }
  // Escape: 退出全屏
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
  }
}

// 组件挂载
onMounted(() => {
  fetchClients()
  document.addEventListener('keydown', handleKeydown)
})

// 组件卸载
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)

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
  background: #1e1e1e;
  border-radius: 8px;
  margin: 8px;
  overflow: hidden;
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
  padding: 8px 12px;
  background: #252526;
  border-bottom: 1px solid #3c3c3c;
  flex-shrink: 0;
  height: 48px;
  box-sizing: border-box;
}

.toolbar-left {
  display: flex;
  gap: 8px;
  align-items: center;
}

.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}

.terminal-tabs-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.terminal-tabs {
  flex-shrink: 0;
}

:deep(.terminal-tabs .el-tabs__header) {
  margin: 0;
  background: #2d2d2d;
  border-bottom: 1px solid #3c3c3c;
}

:deep(.terminal-tabs .el-tabs__nav-wrap) {
  padding: 0 8px;
}

:deep(.terminal-tabs .el-tabs__item) {
  color: #969696;
  border: none !important;
  background: transparent;
  height: 32px;
  line-height: 32px;
  padding: 0 16px;
}

:deep(.terminal-tabs .el-tabs__item.is-active) {
  color: #fff;
  background: #1e1e1e;
}

:deep(.terminal-tabs .el-tabs__item:hover) {
  color: #fff;
}

:deep(.terminal-tabs .el-tabs__content) {
  display: none;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.terminals-wrapper {
  flex: 1;
  position: relative;
  min-height: 0;
  overflow: hidden;
}

.terminal-container {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: #1e1e1e;
  padding: 4px;
  display: none;
  box-sizing: border-box;
}

.terminal-container.active {
  display: block;
}

.empty-state {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}

:deep(.empty-state .el-empty__description) {
  color: #969696;
}

/* xterm 样式调整 - 确保填满容器 */
:deep(.xterm) {
  width: 100% !important;
  height: 100% !important;
  padding: 0;
}

:deep(.xterm-viewport) {
  width: 100% !important;
  height: 100% !important;
  overflow-y: auto !important;
}

:deep(.xterm-screen) {
  width: 100% !important;
  height: 100% !important;
}

:deep(.xterm-helper-textarea) {
  position: absolute;
  opacity: 0;
}

/* 深色主题下的 Element Plus 组件 */
:deep(.el-select) {
  --el-fill-color-blank: #3c3c3c;
  --el-text-color-regular: #d4d4d4;
  --el-border-color: #3c3c3c;
}

:deep(.el-input__wrapper) {
  background-color: #3c3c3c;
  box-shadow: none;
}

:deep(.el-input__inner) {
  color: #d4d4d4;
}

:deep(.el-button) {
  --el-button-bg-color: #3c3c3c;
  --el-button-border-color: #3c3c3c;
  --el-button-text-color: #d4d4d4;
  --el-button-hover-bg-color: #4c4c4c;
  --el-button-hover-border-color: #4c4c4c;
}

:deep(.el-button--success) {
  --el-button-bg-color: #0dbc79;
  --el-button-border-color: #0dbc79;
  --el-button-text-color: #fff;
  --el-button-hover-bg-color: #23d18b;
  --el-button-hover-border-color: #23d18b;
}
</style>
