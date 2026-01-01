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
            
            <!-- trzsz 使用说明卡片 -->
            <div class="guide-card">
              <div class="guide-card-header">
                <div class="guide-header-icon">
                  <el-icon><Document /></el-icon>
                </div>
                <div class="guide-header-content">
                  <h3 class="guide-title">文件传输工具 trzsz</h3>
                  <p class="guide-subtitle">通过终端快速上传和下载文件</p>
                </div>
              </div>
              
              <div class="guide-card-body">
                <div class="guide-grid">
                  <div class="guide-item">
                    <div class="guide-item-icon upload">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                        <polyline points="17 8 12 3 7 8"></polyline>
                        <line x1="12" y1="3" x2="12" y2="15"></line>
                      </svg>
                    </div>
                    <div class="guide-item-content">
                      <h4 class="guide-item-title">上传文件</h4>
                      <div class="guide-item-commands">
                        <code>trz</code>
                        <code>trz -d</code>
                        <code>trz -y</code>
                      </div>
                      <p class="guide-item-desc">支持拖动上传到终端窗口</p>
                    </div>
                  </div>
                  
                  <div class="guide-item">
                    <div class="guide-item-icon download">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                        <polyline points="7 16 12 21 17 16"></polyline>
                        <line x1="12" y1="21" x2="12" y2="9"></line>
                      </svg>
                    </div>
                    <div class="guide-item-content">
                      <h4 class="guide-item-title">下载文件</h4>
                      <div class="guide-item-commands">
                        <code>tsz file</code>
                        <code>tsz -r dir</code>
                        <code>tsz -y file</code>
                      </div>
                      <p class="guide-item-desc">支持目录和断点续传</p>
                    </div>
                  </div>
                </div>
                
                <div class="guide-features">
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>支持 tmux</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>目录传输</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>断点续传</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>进度显示</span>
                  </div>
                </div>
                
                <!-- <div class="guide-footer">
                  <el-link 
                    href="https://trzsz.github.io/cn/" 
                    target="_blank" 
                    type="primary"
                    class="guide-link"
                  >
                    <span>查看完整文档</span>
                    <el-icon><ArrowRight /></el-icon>
                  </el-link>
                </div> -->
              </div>
            </div>
          </div>
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
import { Refresh, Plus, FullScreen, Close, Connection, Document, CircleCheck, ArrowRight } from '@element-plus/icons-vue'
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

  // 固定使用 dark 风格终端主题
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

  terminal.writeln('\x1b[36m╔═══════════════════════════════════════════════════════════════╗\x1b[0m')
  terminal.writeln('\x1b[36m║\x1b[0m  \x1b[33m✨ 欢迎使用 SSH 终端管理系统 ✨\x1b[0m                                    \x1b[36m║\x1b[0m')
  terminal.writeln('\x1b[36m╚═══════════════════════════════════════════════════════════════╝\x1b[0m')
  terminal.writeln('')
  terminal.writeln(`\x1b[36m🔗\x1b[0m 正在连接到 \x1b[33m${clientId}\x1b[0m...`)
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
    terminal.writeln('\x1b[32m✓\x1b[0m WebSocket 连接已建立')
    // 立即发送初始终端大小，服务器会等待这个消息
    if (fitAddons[tabId]) {
      fitAddons[tabId].fit()
    }
    ws.send(JSON.stringify({
      type: 'init',
      cols: terminal.cols,
      rows: terminal.rows
    }))
    terminal.writeln(`\x1b[36m📐\x1b[0m 终端大小: \x1b[33m${terminal.cols}x${terminal.rows}\x1b[0m`)
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
          terminal.writeln('')
          terminal.writeln('\x1b[32m╔═══════════════════════════════════════════════════════════════╗\x1b[0m')
          terminal.writeln('\x1b[32m║\x1b[0m  \x1b[32m✓ SSH 会话已成功建立\x1b[0m                                              \x1b[32m║\x1b[0m')
          terminal.writeln('\x1b[32m╚═══════════════════════════════════════════════════════════════╝\x1b[0m')
          terminal.writeln('')
          terminal.writeln('\x1b[33m⚠️  重要提示与风险提醒：\x1b[0m')
          terminal.writeln('')
          terminal.writeln('  \x1b[36m📋 操作建议：\x1b[0m')
          terminal.writeln('    • 请谨慎执行可能影响系统稳定性的命令')
          terminal.writeln('    • 建议在执行重要操作前先进行测试')
          terminal.writeln('    • 使用 \x1b[33mCtrl+C\x1b[0m 可以中断正在执行的命令')
          terminal.writeln('    • 使用 \x1b[33mCtrl+D\x1b[0m 或输入 \x1b[33mexit\x1b[0m 可以退出当前会话')
          terminal.writeln('')
          terminal.writeln('  \x1b[31m⚠️  风险警告：\x1b[0m')
          terminal.writeln('    • \x1b[31m请勿执行 rm -rf /\x1b[0m 等危险命令，可能导致数据丢失')
          terminal.writeln('    • 修改系统配置文件前请先备份')
          terminal.writeln('    • 生产环境操作需经过审批流程')
          terminal.writeln('    • 所有操作都会被记录，请遵守安全规范')
          terminal.writeln('')
          terminal.writeln('  \x1b[35m💡 实用功能：\x1b[0m')
          terminal.writeln('    • 支持文件传输：使用 \x1b[33mtrz\x1b[0m 上传，\x1b[33mtsz\x1b[0m 下载')
          terminal.writeln('    • 支持 tmux 多窗口管理')
          terminal.writeln('    • 终端窗口可调整大小，自动适配')
          terminal.writeln('')
          terminal.writeln('\x1b[36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1b[0m')
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
  background: linear-gradient(135deg, #0a0e27 0%, #1a1a2e 50%, #16213e 100%);
  border-radius: 12px;
  margin: 8px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: background-color 0.3s ease, border-color 0.3s ease;
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
  border: none;
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
  min-height: 56px;
  box-sizing: border-box;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  transition: background-color 0.3s ease, border-color 0.3s ease;
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
  background: rgba(18, 18, 18, 0.8);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0;
  padding-left: 0;
}

:deep(.terminal-tabs .el-tabs__nav-wrap) {
  padding: 8px 0;
  padding-left: 0;
}

:deep(.terminal-tabs .el-tabs__nav-wrap::after),
:deep(.terminal-tabs .el-tabs__nav-wrap::before) {
  display: none;
}

:deep(.terminal-tabs .el-tabs__nav) {
  border: none;
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
  transition: all 0.2s ease;
  font-weight: 500;
  font-size: 13px;
  cursor: pointer;
  position: relative;
}

:deep(.terminal-tabs .el-tabs__item::before) {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: transparent;
  transition: all 0.2s ease;
}

:deep(.terminal-tabs .el-tabs__item:hover) {
  color: #ffffff;
  background: rgba(59, 130, 246, 0.1);
}

:deep(.terminal-tabs .el-tabs__item.is-active) {
  color: #ffffff;
  background: rgba(59, 130, 246, 0.15);
  border-bottom: 2px solid #3b82f6;
}

:deep(.terminal-tabs .el-tabs__item.is-active::before) {
  background: linear-gradient(90deg, transparent, #3b82f6, transparent);
  opacity: 0.3;
}

:deep(.terminal-tabs .el-tabs__item .el-icon-close) {
  margin-left: 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
  padding: 2px;
}

:deep(.terminal-tabs .el-tabs__item .el-icon-close:hover) {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

:deep(.terminal-tabs .el-tabs__content) {
  display: none;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.tab-label :deep(.el-tag) {
  border: none;
  background: transparent;
  padding: 0;
  font-size: 10px;
  height: auto;
  line-height: 1;
  font-weight: 600;
}

.tab-label :deep(.el-tag--success) {
  color: #10b981;
  text-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
}

.tab-label :deep(.el-tag--info) {
  color: #64748b;
}

.terminals-wrapper {
  flex: 1;
  position: relative;
  min-height: 0;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.3);
}

.terminal-container {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: #0a0e27;
  padding: 0;
  display: none;
  box-sizing: border-box;
  border-radius: 0 0 12px 12px;
}

.terminal-container.active {
  display: block;
  animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.empty-state {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  overflow-y: auto;
  background: linear-gradient(135deg, #0a0e27 0%, #1a1a2e 50%, #16213e 100%);
  transition: background-color 0.3s ease;
}

.empty-content {
  max-width: 900px;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 主视觉区域 */
.empty-hero {
  text-align: center;
  padding: 24px 16px;
}

.empty-icon-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
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
  0%, 100% {
    opacity: 0.4;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(1.05);
  }
}

.empty-icon-main {
  position: relative;
  font-size: 48px;
  color: #3b82f6;
  z-index: 1;
  filter: drop-shadow(0 0 12px rgba(59, 130, 246, 0.4));
}

.empty-title {
  font-size: 24px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 8px 0;
  letter-spacing: -0.3px;
}

.empty-description {
  font-size: 14px;
  color: #94a3b8;
  margin: 0 0 20px 0;
  line-height: 1.5;
  max-width: 400px;
  margin-left: auto;
  margin-right: auto;
}

.empty-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.action-button-primary {
  padding: 10px 24px;
  font-size: 14px;
  font-weight: 600;
  border-radius: 6px;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.25);
}

.action-button-primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.35);
}

/* 使用说明卡片 */
.guide-card {
  background: rgba(30, 30, 30, 0.8);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  overflow: hidden;
  transition: all 0.3s ease;
  cursor: pointer;
}

.guide-card:hover {
  border-color: rgba(59, 130, 246, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  transform: translateY(-2px);
}

.guide-card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px 28px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(16, 185, 129, 0.1) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.guide-header-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(59, 130, 246, 0.2);
  border-radius: 12px;
  flex-shrink: 0;
}

.guide-header-icon .el-icon {
  font-size: 24px;
  color: #3b82f6;
}

.guide-header-content {
  flex: 1;
  text-align: left;
}

.guide-title {
  font-size: 20px;
  font-weight: 700;
  color: #ffffff;
  margin: 0 0 4px 0;
  letter-spacing: -0.3px;
}

.guide-subtitle {
  font-size: 14px;
  color: #94a3b8;
  margin: 0;
  line-height: 1.5;
}

.guide-card-body {
  padding: 28px;
}

.guide-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
  margin-bottom: 28px;
}

.guide-item {
  display: flex;
  gap: 16px;
  padding: 20px;
  background: rgba(18, 18, 18, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  transition: all 0.2s ease;
  cursor: pointer;
}

.guide-item:hover {
  background: rgba(18, 18, 18, 0.8);
  border-color: rgba(59, 130, 246, 0.2);
  transform: translateY(-2px);
}

.guide-item-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  flex-shrink: 0;
}

.guide-item-icon.upload {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.2) 0%, rgba(59, 130, 246, 0.1) 100%);
  color: #3b82f6;
}

.guide-item-icon.download {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.2) 0%, rgba(16, 185, 129, 0.1) 100%);
  color: #10b981;
}

.guide-item-icon svg {
  width: 24px;
  height: 24px;
  stroke-width: 2.5;
}

.guide-item-content {
  flex: 1;
  text-align: left;
}

.guide-item-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  margin: 0 0 12px 0;
}

.guide-item-commands {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.guide-item-commands code {
  display: inline-block;
  background: rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 6px;
  padding: 6px 12px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  color: #60a5fa;
  transition: all 0.2s ease;
}

.guide-item-commands code:hover {
  background: rgba(59, 130, 246, 0.15);
  border-color: rgba(59, 130, 246, 0.4);
  transform: translateY(-1px);
}

.guide-item-desc {
  font-size: 13px;
  color: #94a3b8;
  margin: 0;
  line-height: 1.5;
}

.guide-features {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 24px;
  padding: 20px;
  background: rgba(18, 18, 18, 0.4);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.feature-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 8px;
  font-size: 13px;
  color: #10b981;
  transition: all 0.2s ease;
}

.feature-badge:hover {
  background: rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.3);
  transform: translateY(-1px);
}

.feature-badge .el-icon {
  font-size: 16px;
}

.guide-footer {
  text-align: center;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.guide-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  padding: 10px 20px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.guide-link:hover {
  gap: 12px;
}

.guide-link .el-icon {
  transition: transform 0.2s ease;
}

.guide-link:hover .el-icon {
  transform: translateX(4px);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .terminal-page {
    margin: 4px;
    border-radius: 8px;
  }
  
  .terminal-toolbar {
    padding: 10px 16px;
    flex-wrap: wrap;
    gap: 8px;
    min-height: auto;
  }
  
  .toolbar-left {
    flex: 1 1 100%;
    order: 1;
  }
  
  .toolbar-right {
    flex: 1 1 auto;
    order: 2;
  }
  
  :deep(.terminal-tabs .el-tabs__item) {
    padding: 0 12px;
    font-size: 12px;
  }
  
  .terminal-container {
    padding: 0;
  }
  
  :deep(.xterm-viewport) {
    padding: 0 8px;
  }
  
  :deep(.xterm-screen) {
    padding: 0 8px;
  }
  
  :deep(.xterm) {
    padding: 0;
  }
  
  .empty-state {
    padding: 16px 12px;
  }
  
  .empty-hero {
    padding: 16px 12px;
  }
  
  .empty-title {
    font-size: 20px;
  }
  
  .empty-description {
    font-size: 13px;
  }
  
  .empty-icon-main {
    font-size: 40px;
  }
  
  .icon-glow {
    width: 70px;
    height: 70px;
  }
  
  .action-button-primary {
    padding: 8px 20px;
    font-size: 13px;
  }
  
  .guide-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .guide-card-header {
    flex-direction: column;
    text-align: center;
    padding: 20px;
  }
  
  .guide-header-content {
    text-align: center;
  }
  
  .guide-features {
    justify-content: center;
  }
  
  .action-button-primary {
    width: 100%;
    max-width: 300px;
  }
}

@media (max-width: 480px) {
  .terminal-toolbar {
    padding: 8px 12px;
  }
  
  .toolbar-left {
    gap: 6px;
  }
  
  :deep(.el-select) {
    width: 140px !important;
  }
  
  :deep(.el-button) {
    padding: 8px 12px;
    font-size: 12px;
  }
  
  :deep(.terminal-tabs .el-tabs__item) {
    padding: 0 8px;
    font-size: 11px;
  }
  
  .empty-content {
    gap: 24px;
  }
  
  .guide-card-body {
    padding: 20px;
  }
  
  .guide-item {
    flex-direction: column;
    text-align: center;
  }
  
  .guide-item-content {
    text-align: center;
  }
  
  .guide-item-commands {
    justify-content: center;
  }
}

/* xterm 样式调整 - 确保填满容器并美化 */
:deep(.xterm) {
  width: 100% !important;
  height: 100% !important;
  padding: 0;
  background: #0a0e27 !important;
  border-radius: 0;
  box-shadow: inset 0 0 40px rgba(0, 0, 0, 0.3);
  transition: background-color 0.3s ease;
}

:deep(.xterm-viewport) {
  width: 100% !important;
  height: 100% !important;
  overflow-y: auto !important;
  background: transparent !important;
  padding: 0 12px;
  box-sizing: border-box;
}

:deep(.xterm-viewport::-webkit-scrollbar) {
  width: 10px;
}

:deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 5px;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: rgba(59, 130, 246, 0.3);
  border-radius: 5px;
  transition: background 0.2s ease;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: rgba(59, 130, 246, 0.5);
}

:deep(.xterm-screen) {
  width: 100% !important;
  height: 100% !important;
  padding: 0 12px;
  box-sizing: border-box;
}

:deep(.xterm-helper-textarea) {
  position: absolute;
  opacity: 0;
}

:deep(.xterm-cursor-layer) {
  z-index: 2;
}

:deep(.xterm-cursor) {
  background-color: #3b82f6 !important;
  box-shadow: 0 0 10px rgba(59, 130, 246, 0.5);
  animation: blink-cursor 1s infinite;
}

@keyframes blink-cursor {
  0%, 50% {
    opacity: 1;
  }
  51%, 100% {
    opacity: 0.3;
  }
}

/* Terminal 页面固定使用深色主题 - Element Plus 组件 */
:deep(.el-select) {
  --el-fill-color-blank: rgba(30, 30, 30, 0.8);
  --el-text-color-regular: #ffffff;
  --el-border-color: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.el-select:hover) {
  --el-border-color: rgba(59, 130, 246, 0.3);
}

:deep(.el-select.is-focus) {
  --el-border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

:deep(.el-input__wrapper) {
  background-color: rgba(30, 30, 30, 0.8);
  backdrop-filter: blur(10px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.el-input__wrapper:hover) {
  background-color: rgba(30, 30, 30, 0.9);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

:deep(.el-input__inner) {
  color: #ffffff;
  font-weight: 500;
}

:deep(.el-select-dropdown) {
  background: rgba(18, 18, 18, 0.95);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

:deep(.el-select-dropdown .el-option) {
  color: #d4d4d4;
  transition: all 0.2s ease;
}

:deep(.el-select-dropdown .el-option:hover) {
  background: rgba(59, 130, 246, 0.15);
  color: #ffffff;
}

:deep(.el-select-dropdown .el-option.is-selected) {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
  font-weight: 600;
}

:deep(.el-button) {
  --el-button-bg-color: rgba(30, 30, 30, 0.8);
  --el-button-border-color: rgba(255, 255, 255, 0.1);
  --el-button-text-color: #d4d4d4;
  --el-button-hover-bg-color: rgba(59, 130, 246, 0.15);
  --el-button-hover-border-color: rgba(59, 130, 246, 0.3);
  --el-button-hover-text-color: #ffffff;
  border-radius: 8px;
  font-weight: 500;
  transition: all 0.2s ease;
  backdrop-filter: blur(10px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

:deep(.el-button:hover) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

:deep(.el-button:active) {
  transform: translateY(0);
}

:deep(.el-button--success) {
  --el-button-bg-color: #10b981;
  --el-button-border-color: #10b981;
  --el-button-text-color: #ffffff;
  --el-button-hover-bg-color: #059669;
  --el-button-hover-border-color: #059669;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3);
}

:deep(.el-button--success:hover) {
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.4);
}

:deep(.el-button--success:disabled) {
  --el-button-bg-color: rgba(30, 30, 30, 0.5);
  --el-button-border-color: rgba(255, 255, 255, 0.05);
  --el-button-text-color: #64748b;
  opacity: 0.6;
  cursor: not-allowed;
}

:deep(.el-button.is-loading) {
  pointer-events: none;
}
</style>
