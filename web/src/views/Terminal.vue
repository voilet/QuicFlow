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

          <!-- 常规输入/输出终端 -->
          <div class="simple-terminal">
            <!-- 输出区域 -->
            <div class="terminal-output" ref="outputRefs">
              <div v-if="tab.history.length === 0" class="terminal-welcome">
                <div class="welcome-icon">🔗</div>
                <div class="welcome-title">SSH 终端已连接</div>
                <div class="welcome-desc">
                  客户端: <span class="highlight">{{ tab.clientId }}</span><br>
                  状态: <span :class="tab.connected ? 'status-online' : 'status-offline'">
                    {{ tab.connected ? '已连接' : '未连接' }}
                  </span>
                </div>
                <div class="welcome-hint">
                  💡 在下方输入框输入命令，按 Enter 执行
                </div>
              </div>

              <div
                v-for="(entry, idx) in tab.history"
                :key="idx"
                :class="['terminal-entry', `entry-${entry.type}`]"
              >
                <!-- 命令输入 -->
                <div v-if="entry.type === 'input'" class="entry-input">
                  <span class="prompt">$</span>
                  <span class="command">{{ escapeHtml(entry.content) }}</span>
                </div>

                <!-- 命令输出 -->
                <div v-else-if="entry.type === 'output'" class="entry-output">
                  <pre class="output-text">{{ escapeHtml(entry.content) }}</pre>
                </div>

                <!-- 系统消息 -->
                <div v-else-if="entry.type === 'system'" class="entry-system">
                  {{ entry.content }}
                </div>

                <!-- 错误消息 -->
                <div v-else-if="entry.type === 'error'" class="entry-error">
                  ❌ {{ escapeHtml(entry.content) }}
                </div>
              </div>
            </div>

            <!-- 输入区域 -->
            <div class="terminal-input-area">
              <div class="input-prompt">$</div>
              <el-input
                v-model="tab.inputValue"
                :placeholder="tab.connected ? '输入命令...' : '未连接'"
                :disabled="!tab.connected"
                @keydown="handleInputKeydown($event, tab.id)"
                class="terminal-input"
                size="large"
                clearable
              />
              <el-button
                type="primary"
                @click="sendCommand(tab.id)"
                :disabled="!tab.connected || !tab.inputValue"
                :loading="tab.sending"
                size="large"
              >
                执行
              </el-button>
              <el-button
                @click="clearHistory(tab.id)"
                :disabled="tab.history.length === 0"
                size="large"
              >
                清空
              </el-button>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- 终端容器 -->
      <div class="terminals-wrapper">
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

            <!-- 使用说明 -->
            <div class="guide-card">
              <div class="guide-card-header">
                <div class="guide-header-icon">
                  <el-icon><Document /></el-icon>
                </div>
                <div class="guide-header-content">
                  <h3 class="guide-title">命令执行终端</h3>
                  <p class="guide-subtitle">远程执行 Shell 命令并查看输出结果</p>
                </div>
              </div>

              <div class="guide-card-body">
                <div class="guide-features">
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>简单输入输出</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>命令历史记录</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>多标签页支持</span>
                  </div>
                  <div class="feature-badge">
                    <el-icon><CircleCheck /></el-icon>
                    <span>实时输出显示</span>
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
const outputRefs = ref({})

// WebSocket 连接存储
const websockets = {}

let tabCounter = 0

// 是否还有更多数据可加载
const hasMoreClients = computed(() => clients.value.length < clientTotal.value)

// HTML 转义
function escapeHtml(text) {
  if (!text) return ''
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
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
    connected: false,
    sending: false,
    inputValue: '',
    history: []
  })

  activeTabId.value = tabId

  await nextTick()

  // 连接 WebSocket
  connectWebSocket(tabId, clientId)
}

// 连接 WebSocket
function connectWebSocket(tabId, clientId) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/terminal/ws/${clientId}`

  const ws = new WebSocket(wsUrl)
  websockets[tabId] = ws

  const tab = terminalTabs.value.find(t => t.id === tabId)

  ws.onopen = () => {
    addHistoryEntry(tabId, 'system', `已连接到 ${clientId}`)
    if (tab) {
      tab.connected = true
      tab.sessionId = Date.now().toString()
    }
    // 发送初始化消息
    ws.send(JSON.stringify({
      type: 'init',
      cols: 80,
      rows: 24
    }))
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      switch (msg.type) {
        case 'output':
          addHistoryEntry(tabId, 'output', msg.data)
          break
        case 'connected':
          if (tab) {
            tab.connected = true
            tab.sessionId = msg.data
          }
          addHistoryEntry(tabId, 'system', 'SSH 会话已建立')
          break
        case 'disconnected':
          addHistoryEntry(tabId, 'system', `连接已断开: ${msg.data}`)
          if (tab) tab.connected = false
          break
        case 'error':
          addHistoryEntry(tabId, 'error', msg.data)
          if (tab) tab.connected = false
          break
      }
    } catch (e) {
      console.error('解析消息失败:', e)
    }
  }

  ws.onclose = () => {
    if (tab && tab.connected) {
      addHistoryEntry(tabId, 'system', 'WebSocket 连接已关闭')
    }
    if (tab) tab.connected = false
  }

  ws.onerror = (error) => {
    addHistoryEntry(tabId, 'error', '连接错误')
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

// 添加历史记录
function addHistoryEntry(tabId, type, content) {
  const tab = terminalTabs.value.find(t => t.id === tabId)
  if (!tab) return

  tab.history.push({
    type,
    content,
    timestamp: Date.now()
  })

  // 自动滚动到底部
  nextTick(() => {
    scrollToBottom(tabId)
  })
}

// 滚动到底部
function scrollToBottom(tabId) {
  const container = outputRefs.value
  if (!container) return
  // 找到对应的输出容器
  const tabElement = document.querySelector(`[data-tab-id="${tabId}"] .terminal-output`)
  if (tabElement) {
    tabElement.scrollTop = tabElement.scrollHeight
  }
}

// 处理输入框按键
function handleInputKeydown(event, tabId) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendCommand(tabId)
  }
}

// 发送命令
function sendCommand(tabId) {
  const tab = terminalTabs.value.find(t => t.id === tabId)
  if (!tab || !tab.connected || !tab.inputValue) return

  const command = tab.inputValue.trim()
  if (!command) return

  // 添加命令到历史
  addHistoryEntry(tabId, 'input', command)

  // 清空输入框
  const inputValue = tab.inputValue
  tab.inputValue = ''

  // 发送到 WebSocket
  const ws = websockets[tabId]
  if (ws && ws.readyState === WebSocket.OPEN) {
    tab.sending = true
    ws.send(JSON.stringify({
      type: 'input',
      data: command + '\n'
    }))
    tab.sending = false
  } else {
    addHistoryEntry(tabId, 'error', 'WebSocket 未连接')
  }
}

// 清空历史
function clearHistory(tabId) {
  const tab = terminalTabs.value.find(t => t.id === tabId)
  if (tab) {
    tab.history = []
  }
}

// 关闭终端
function closeTerminal(tabId) {
  const ws = websockets[tabId]
  if (ws) {
    if (ws._pingInterval) clearInterval(ws._pingInterval)
    ws.close()
    delete websockets[tabId]
  }

  const index = terminalTabs.value.findIndex(t => t.id === tabId)
  if (index !== -1) {
    terminalTabs.value.splice(index, 1)
  }

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
    scrollToBottom(tabId)
  })
}

// 全屏切换
function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
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
})

// 组件卸载
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)

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
  flex-shrink: 0;
}

:deep(.terminal-tabs .el-tabs__header) {
  margin: 0;
  background: rgba(18, 18, 18, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
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
  display: none;
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

/* 简单终端 */
.simple-terminal {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: rgba(10, 14, 39, 0.5);
  border-radius: 0 0 12px 12px;
}

.terminal-output {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}

/* 滚动条样式 */
.terminal-output::-webkit-scrollbar {
  width: 8px;
}

.terminal-output::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
}

.terminal-output::-webkit-scrollbar-thumb {
  background: rgba(59, 130, 246, 0.3);
  border-radius: 4px;
}

.terminal-output::-webkit-scrollbar-thumb:hover {
  background: rgba(59, 130, 246, 0.5);
}

/* 欢迎界面 */
.terminal-welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  color: #94a3b8;
}

.welcome-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.welcome-title {
  font-size: 20px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 8px;
}

.welcome-desc {
  font-size: 14px;
  line-height: 1.8;
}

.welcome-desc .highlight {
  color: #3b82f6;
  font-weight: 500;
}

.welcome-desc .status-online {
  color: #10b981;
}

.welcome-desc .status-offline {
  color: #64748b;
}

.welcome-hint {
  margin-top: 20px;
  padding: 12px 20px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 8px;
  font-size: 13px;
}

/* 终端条目 */
.terminal-entry {
  margin-bottom: 8px;
}

.entry-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.entry-input .prompt {
  color: #10b981;
  font-weight: 600;
  flex-shrink: 0;
}

.entry-input .command {
  color: #ffffff;
  word-break: break-all;
}

.entry-output {
  margin-left: 20px;
}

.entry-output .output-text {
  margin: 0;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.entry-system {
  color: #64748b;
  font-style: italic;
  padding: 4px 0;
}

.entry-error {
  color: #ef4444;
  padding: 4px 0;
}

/* 输入区域 */
.terminal-input-area {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(30, 30, 30, 0.9);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;
}

.input-prompt {
  color: #10b981;
  font-weight: 600;
  font-size: 16px;
  flex-shrink: 0;
}

.terminal-input {
  flex: 1;
}

:deep(.terminal-input .el-input__wrapper) {
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
}

:deep(.terminal-input .el-input__inner) {
  color: #ffffff;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

:deep(.terminal-input .el-input__inner::placeholder) {
  color: #64748b;
}

/* 终端包装器 */
.terminals-wrapper {
  flex: 1;
  position: relative;
  min-height: 0;
  background: rgba(0, 0, 0, 0.3);
}

/* 空状态 */
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
