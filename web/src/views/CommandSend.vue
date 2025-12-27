<template>
  <div class="command-send">
    <el-row :gutter="16">
      <!-- 左侧：命令表单 -->
      <el-col :span="7">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>批量下发命令</span>
            </div>
          </template>

          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-width="100px"
            label-position="left"
            size="default"
          >
            <!-- 目标客户端输入 -->
            <el-form-item label="目标客户端">
              <div class="client-input-wrapper">
                <el-input
                  v-model="clientIdsText"
                  type="textarea"
                  :rows="4"
                  placeholder="输入客户端ID，每行一个&#10;例如:&#10;client-001&#10;client-002"
                  class="client-textarea"
                />
                <div class="form-actions">
                  <el-button type="info" link size="small" @click="clientIdsText = ''">清空</el-button>
                  <span class="select-count">{{ clientIdsList.length }} 个目标</span>
                </div>
              </div>
            </el-form-item>

            <el-form-item label="命令类型" prop="command_type">
              <el-select
                v-model="form.command_type"
                placeholder="请选择命令类型"
                style="width: 100%"
                @change="onCommandTypeChange"
              >
                <el-option
                  v-for="cmd in commandTypes"
                  :key="cmd.value"
                  :label="cmd.label"
                  :value="cmd.value"
                />
              </el-select>
            </el-form-item>

            <!-- Shell 命令输入 -->
            <el-form-item v-if="form.command_type === 'exec_shell'" label="Shell命令">
              <div class="shell-input-wrapper">
                <div class="shell-header">
                  <span class="shell-prompt">$</span>
                  <el-dropdown @command="useQuickCommand" class="quick-cmd-dropdown">
                    <el-button size="small" type="primary" link>
                      快捷命令 <el-icon><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="ls -la">ls -la</el-dropdown-item>
                        <el-dropdown-item command="pwd">pwd</el-dropdown-item>
                        <el-dropdown-item command="hostname">hostname</el-dropdown-item>
                        <el-dropdown-item command="whoami">whoami</el-dropdown-item>
                        <el-dropdown-item command="date">date</el-dropdown-item>
                        <el-dropdown-item command="uptime">uptime</el-dropdown-item>
                        <el-dropdown-item command="df -h">df -h</el-dropdown-item>
                        <el-dropdown-item command="free -m">free -m</el-dropdown-item>
                        <el-dropdown-item command="ps aux | head -20">ps aux | head</el-dropdown-item>
                        <el-dropdown-item command="netstat -tlnp">netstat -tlnp</el-dropdown-item>
                        <el-dropdown-item command="cat /etc/os-release">cat /etc/os-release</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
                <el-input
                  v-model="shellCommand"
                  type="textarea"
                  :rows="3"
                  placeholder="输入 Shell 命令，支持多行&#10;例如: ls -la /tmp"
                  class="shell-textarea"
                  @input="updatePayloadFromShell"
                />
              </div>
            </el-form-item>

            <el-form-item label="超时时间">
              <el-slider
                v-model="form.timeout"
                :min="5"
                :max="120"
                :step="5"
                :format-tooltip="(val) => `${val}秒`"
                show-stops
              />
            </el-form-item>

            <el-form-item>
              <el-button
                type="primary"
                @click="submitCommand"
                :loading="submitting"
                :disabled="clientIdsList.length === 0"
                :icon="VideoPlay"
                style="width: 100%"
              >
                {{ submitting ? '执行中...' : '批量执行' }}
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 批量下发统计 -->
        <el-card shadow="never" class="stats-card">
          <template #header>
            <div class="card-header">
              <span>执行统计</span>
              <el-button v-if="sentCount > 0" type="danger" link size="small" @click="clearStats">
                清空
              </el-button>
            </div>
          </template>

          <div class="stats-grid">
            <div class="stat-item">
              <div class="stat-value sent">{{ sentCount }}</div>
              <div class="stat-label">已发送</div>
            </div>
            <div class="stat-item">
              <div class="stat-value returned">{{ returnedCount }}</div>
              <div class="stat-label">已返回</div>
            </div>
            <div class="stat-item">
              <div class="stat-value pending">{{ pendingClients.size }}</div>
              <div class="stat-label">未执行</div>
            </div>
            <div class="stat-item">
              <div class="stat-value offline">{{ offlineClients.size }}</div>
              <div class="stat-label">不在线</div>
            </div>
          </div>

          <!-- 不在线客户端列表 -->
          <div v-if="offlineClients.size > 0" class="offline-list">
            <div class="list-header offline">
              <span>不在线客户端 ({{ offlineClients.size }})</span>
            </div>
            <div class="client-tags">
              <el-tag
                v-for="clientId in Array.from(offlineClients)"
                :key="clientId"
                size="small"
                type="info"
                class="client-tag"
              >
                {{ clientId }}
              </el-tag>
            </div>
          </div>

          <!-- 未执行客户端列表 -->
          <div v-if="pendingClients.size > 0" class="pending-list">
            <div class="list-header pending">
              <span>未执行客户端 ({{ pendingClients.size }})</span>
            </div>
            <div class="client-tags">
              <el-tag
                v-for="clientId in Array.from(pendingClients)"
                :key="clientId"
                size="small"
                type="danger"
                class="client-tag"
              >
                {{ clientId }}
              </el-tag>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：执行结果 -->
      <el-col :span="17">
        <el-card shadow="never" class="result-card">
          <template #header>
            <div class="card-header">
              <span>执行结果</span>
              <div class="header-actions" v-if="executionResults.length > 0">
                <el-button type="danger" link @click="clearResults">清空</el-button>
              </div>
            </div>
          </template>

          <div v-if="executionResults.length === 0" class="no-results">
            <el-empty description="执行命令后在此查看结果" :image-size="100" />
          </div>

          <div v-else class="results-container">
            <div
              v-for="(result, index) in executionResults"
              :key="result.id"
              class="result-item"
            >
              <!-- 结果头部 -->
              <div class="result-header" @click="result.expanded = !result.expanded">
                <div class="result-title">
                  <el-icon class="status-icon" :class="getStatusClass(result.status)">
                    <Loading v-if="result.status === 'pending' || result.status === 'executing'" class="is-loading" />
                    <CircleCheck v-else-if="result.status === 'completed'" />
                    <CircleClose v-else />
                  </el-icon>
                  <span class="client-name">{{ result.client_id }}</span>
                  <el-tag size="small" :type="getStatusType(result.status)">
                    {{ getStatusText(result.status) }}
                  </el-tag>
                  <span v-if="result.command_id" class="command-id">ID: {{ result.command_id.slice(0, 8) }}...</span>
                </div>
                <div class="result-meta">
                  <span class="time">{{ result.time }}</span>
                  <el-icon class="expand-icon" :class="{ expanded: result.expanded }">
                    <ArrowDown />
                  </el-icon>
                </div>
              </div>

              <!-- 结果内容 -->
              <el-collapse-transition>
                <div v-show="result.expanded" class="result-content">
                  <!-- 加载中 -->
                  <div v-if="result.status === 'pending' || result.status === 'executing'" class="terminal-box">
                    <div class="terminal-header">
                      <span class="terminal-dot red"></span>
                      <span class="terminal-dot yellow"></span>
                      <span class="terminal-dot green"></span>
                      <span class="terminal-title">Terminal</span>
                    </div>
                    <div class="terminal-body">
                      <div class="terminal-loading">
                        <span class="cursor-blink">_</span> 正在执行...
                      </div>
                    </div>
                  </div>

                  <!-- 执行结果 -->
                  <template v-else-if="result.output">
                    <div class="terminal-box">
                      <div class="terminal-header">
                        <span class="terminal-dot red"></span>
                        <span class="terminal-dot yellow"></span>
                        <span class="terminal-dot green"></span>
                        <span class="terminal-title">Terminal - {{ result.output.exit_code === 0 ? 'Success' : 'Error' }}</span>
                      </div>
                      <div class="terminal-body">
                        <!-- 标准输出 -->
                        <div v-if="result.output.stdout" class="terminal-output">
                          <pre class="terminal-text" v-html="highlightOutput(result.output.stdout)"></pre>
                        </div>

                        <!-- 错误输出 -->
                        <div v-if="result.output.stderr" class="terminal-output stderr">
                          <pre class="terminal-text error">{{ result.output.stderr }}</pre>
                        </div>

                        <!-- 退出码 -->
                        <div class="terminal-footer">
                          <span :class="['exit-code-badge', result.output.exit_code === 0 ? 'success' : 'error']">
                            Exit: {{ result.output.exit_code }}
                          </span>
                        </div>
                      </div>
                    </div>
                  </template>

                  <!-- 错误信息 -->
                  <div v-else-if="result.error" class="terminal-box error">
                    <div class="terminal-header">
                      <span class="terminal-dot red"></span>
                      <span class="terminal-dot yellow"></span>
                      <span class="terminal-dot green"></span>
                      <span class="terminal-title">Terminal - Error</span>
                    </div>
                    <div class="terminal-body">
                      <pre class="terminal-text error">{{ result.error }}</pre>
                    </div>
                  </div>
                </div>
              </el-collapse-transition>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  VideoPlay, Loading,
  CircleCheck, CircleClose, ArrowDown
} from '@element-plus/icons-vue'
import api from '@/api'
import dayjs from 'dayjs'

const route = useRoute()

const formRef = ref()
const submitting = ref(false)
const shellCommand = ref('ls -la')
const executionResults = ref([])
const pollingTimers = ref({})
const clientIdsText = ref('')  // 客户端ID文本，每行一个

// 统计相关
const sentCount = ref(0)  // 已发送的总数
const pendingClients = ref(new Set())  // 未执行（已发送但未返回）的客户端
const returnedCount = ref(0)  // 已返回的数量
const offlineClients = ref(new Set())  // 不在线的客户端

// 计算属性：从文本解析客户端ID列表
const clientIdsList = computed(() => {
  return clientIdsText.value
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
})

// 表单数据
const form = reactive({
  client_id: route.query.client_id || '',
  client_ids: [],
  command_type: 'exec_shell',
  payload: JSON.stringify({ command: 'ls -la', timeout: 30 }, null, 2),
  timeout: 30
})

// 验证规则
const rules = {
  client_id: [{ required: true, message: '请选择客户端', trigger: 'change' }],
  command_type: [{ required: true, message: '请选择命令类型', trigger: 'change' }]
}

// 命令类型
const commandTypes = [
  { label: '执行Shell命令', value: 'exec_shell' },
  { label: '获取状态', value: 'get_status' },
  { label: '重启服务', value: 'restart' },
  { label: '更新配置', value: 'update_config' }
]

// 清空统计
function clearStats() {
  sentCount.value = 0
  pendingClients.value = new Set()
  returnedCount.value = 0
  offlineClients.value = new Set()
}

// 标记客户端已返回
function markClientReturned(clientId, isOffline = false) {
  if (pendingClients.value.has(clientId)) {
    pendingClients.value.delete(clientId)
    // 触发响应式更新
    pendingClients.value = new Set(pendingClients.value)

    // 如果是不在线，添加到不在线列表，同时从已发送中减掉
    if (isOffline) {
      offlineClients.value.add(clientId)
      offlineClients.value = new Set(offlineClients.value)
      // 不在线的不算已发送
      sentCount.value--
    } else {
      // 只有在线且成功返回的才计入已返回
      returnedCount.value++
    }
  }
}

// 检查是否是不在线错误
function isOfflineError(error) {
  if (!error) return false
  const offlinePatterns = [
    'client not connected',
    'session not found',
    'not connected',
    'connection refused'
  ]
  const errorStr = String(error).toLowerCase()
  return offlinePatterns.some(pattern => errorStr.includes(pattern))
}

// 命令类型改变
function onCommandTypeChange(type) {
  if (type === 'exec_shell') {
    updatePayloadFromShell()
  } else if (type === 'get_status') {
    form.payload = '{}'
  } else if (type === 'restart') {
    form.payload = JSON.stringify({ delay_seconds: 5 }, null, 2)
  }
}

// 从 shell 命令更新 payload
function updatePayloadFromShell() {
  form.payload = JSON.stringify({
    command: shellCommand.value,
    timeout: form.timeout
  }, null, 2)
}

// 使用快捷命令
function useQuickCommand(command) {
  shellCommand.value = command
  updatePayloadFromShell()
}

// 清空结果
function clearResults() {
  // 停止所有轮询
  Object.values(pollingTimers.value).forEach(timer => clearInterval(timer))
  pollingTimers.value = {}
  executionResults.value = []
}

// 提交命令
async function submitCommand() {
  // 验证
  if (clientIdsList.value.length === 0) {
    ElMessage.warning('请输入至少一个客户端ID')
    return
  }

  submitting.value = true
  updatePayloadFromShell()

  // 清空之前的结果
  clearResults()

  // 重置统计
  clearStats()

  try {
    const payload = JSON.parse(form.payload)
    const timestamp = dayjs().format('HH:mm:ss')

    // 将所有目标客户端添加到统计
    const targetClients = clientIdsList.value
    sentCount.value = targetClients.length
    pendingClients.value = new Set(targetClients)

    // 批量发送命令
    const res = await api.sendMultiCommand({
      client_ids: targetClients,
      command_type: form.command_type,
      payload,
      timeout: form.timeout
    })

    console.log('Multi-command response:', res)

    // 按返回顺序添加结果（先返回的在前面）
    if (res.results && Array.isArray(res.results)) {
      const newResults = []
      res.results.forEach((r, index) => {
        // 检查是否是不在线错误
        const offline = isOfflineError(r.error)
        // 结果返回时，从已发送列表移除
        markClientReturned(r.client_id, offline)

        // 不在线的主机不显示在结果中
        if (!offline) {
          newResults.push({
            id: `${r.client_id}-${Date.now()}-${index}`,
            client_id: r.client_id,
            status: r.status || 'completed',
            time: timestamp,
            expanded: true,
            output: parseResultOutput(r.result),
            error: r.error,
            command_id: r.command_id
          })
        }
      })
      console.log('Adding results:', newResults)
      executionResults.value = [...executionResults.value, ...newResults]
    }
  } catch (error) {
    ElMessage.error('命令执行失败: ' + (error.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

// 解析结果输出
function parseResultOutput(result) {
  if (!result) return null
  let output = result
  if (typeof output === 'string') {
    try {
      output = JSON.parse(output)
    } catch (e) {}
  }
  return output
}

// 高亮输出内容
function highlightOutput(text) {
  if (!text) return ''

  // 转义 HTML 特殊字符
  const escapeHtml = (str) => {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
  }

  const lines = text.split('\n')
  const highlightedLines = lines.map(line => {
    const escapedLine = escapeHtml(line)

    // 检测 ls -la 格式的行
    // 格式: drwxr-xr-x  2 user group  4096 Jan  1 00:00 filename
    const lsMatch = line.match(/^([dlcbsp-])([rwxsStT-]{9})\s+/)

    if (lsMatch) {
      const fileType = lsMatch[1]
      const permissions = lsMatch[2]

      // 提取文件名（最后一个字段）
      const parts = line.trim().split(/\s+/)
      if (parts.length >= 9) {
        const fileName = parts.slice(8).join(' ')
        const prefix = parts.slice(0, 8).join(' ')
        const escapedPrefix = escapeHtml(prefix)
        const escapedFileName = escapeHtml(fileName)

        // 根据类型返回不同颜色
        if (fileType === 'd') {
          // 目录 - 蓝色
          return `<span class="ls-perm">${escapedPrefix}</span> <span class="ls-dir">${escapedFileName}</span>`
        } else if (fileType === 'l') {
          // 符号链接 - 青色
          return `<span class="ls-perm">${escapedPrefix}</span> <span class="ls-link">${escapedFileName}</span>`
        } else if (permissions.includes('x')) {
          // 可执行文件 - 绿色
          return `<span class="ls-perm">${escapedPrefix}</span> <span class="ls-exec">${escapedFileName}</span>`
        } else {
          // 普通文件
          return `<span class="ls-perm">${escapedPrefix}</span> <span class="ls-file">${escapedFileName}</span>`
        }
      }
    }

    // 检测 total 行
    if (line.match(/^total\s+\d+/)) {
      return `<span class="ls-total">${escapedLine}</span>`
    }

    // 检测路径
    if (line.match(/^\/[\w\-\.\/]+:?\s*$/)) {
      return `<span class="ls-path">${escapedLine}</span>`
    }

    return escapedLine
  })

  return highlightedLines.join('\n')
}

// 开始轮询命令状态
function startPolling(resultItem) {
  const commandId = resultItem.command_id
  if (!commandId) return

  const timer = setInterval(async () => {
    try {
      const res = await api.getCommand(commandId)
      if (res.command) {
        const cmd = res.command

        resultItem.status = cmd.status
        resultItem.output = parseResultOutput(cmd.result)
        resultItem.error = cmd.error

        // 如果是终态，停止轮询
        if (['completed', 'failed', 'timeout'].includes(cmd.status)) {
          clearInterval(timer)
          delete pollingTimers.value[commandId]
        }
      }
    } catch (error) {
      console.error('轮询失败', error)
    }
  }, 500)

  pollingTimers.value[commandId] = timer
}

// 获取状态类名
function getStatusClass(status) {
  return {
    pending: status === 'pending',
    executing: status === 'executing',
    success: status === 'completed',
    error: status === 'failed' || status === 'timeout',
    offline: status === 'offline'
  }
}

// 获取状态类型
function getStatusType(status) {
  const types = {
    pending: 'warning',
    executing: 'info',
    completed: 'success',
    failed: 'danger',
    timeout: 'danger',
    offline: 'info'
  }
  return types[status] || 'info'
}

// 获取状态文本
function getStatusText(status) {
  const texts = {
    pending: '等待中',
    executing: '执行中',
    completed: '完成',
    failed: '失败',
    timeout: '超时',
    offline: '不在线'
  }
  return texts[status] || status
}

onUnmounted(() => {
  // 清理所有轮询定时器
  Object.values(pollingTimers.value).forEach(timer => clearInterval(timer))
})
</script>

<style scoped>
.command-send {
  height: calc(100vh - 140px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* 客户端输入区域 */
.client-input-wrapper {
  width: 100%;
}

.client-textarea :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  background: #fafafa;
  border-color: #e4e7ed;
}

.client-textarea :deep(.el-textarea__inner):focus {
  background: #fff;
}

/* Shell 命令输入区域 */
.shell-input-wrapper {
  width: 100%;
  background: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
}

.shell-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: linear-gradient(180deg, #3d3d3d 0%, #2d2d2d 100%);
}

.shell-prompt {
  color: #67c23a;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-weight: bold;
  font-size: 14px;
}

.quick-cmd-dropdown {
  margin-left: auto;
}

.shell-textarea :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  background: #1e1e1e;
  color: #d4d4d4;
  border: none;
  border-radius: 0;
  padding: 12px;
  resize: none;
}

.shell-textarea :deep(.el-textarea__inner)::placeholder {
  color: #6a6a6a;
}

.shell-textarea :deep(.el-textarea__inner):focus {
  box-shadow: none;
}

.form-actions {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.select-count {
  font-size: 12px;
  color: #909399;
  margin-left: auto;
}

/* 统计卡片 */
.stats-card {
  margin-top: 16px;
}

.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 16px;
}

.stat-item {
  text-align: center;
  padding: 10px 6px;
  background: #f5f7fa;
  border-radius: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  margin-bottom: 2px;
}

.stat-value.sent {
  color: #409eff;
}

.stat-value.returned {
  color: #67c23a;
}

.stat-value.pending {
  color: #e6a23c;
}

.stat-value.offline {
  color: #909399;
}

.stat-label {
  font-size: 12px;
  color: #909399;
}

.offline-list,
.pending-list {
  border-top: 1px solid #ebeef5;
  padding-top: 12px;
  margin-top: 12px;
}

.list-header {
  font-size: 13px;
  margin-bottom: 8px;
  font-weight: 500;
}

.list-header.offline {
  color: #909399;
}

.list-header.pending {
  color: #e6a23c;
}

.client-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-height: 150px;
  overflow-y: auto;
}

.client-tag {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 11px;
}

/* 结果区域 */
.result-card {
  height: calc(100vh - 140px);
  display: flex;
  flex-direction: column;
}

.result-card :deep(.el-card__body) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.no-results {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.results-container {
  flex: 1;
  overflow-y: auto;
  padding-right: 8px;
}

.result-item {
  background: #fafafa;
  border-radius: 8px;
  margin-bottom: 12px;
  border: 1px solid #ebeef5;
  overflow: hidden;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.result-header:hover {
  background: #f0f0f0;
}

.result-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-icon {
  font-size: 18px;
}

.status-icon.pending,
.status-icon.executing {
  color: #e6a23c;
}

.status-icon.success {
  color: #67c23a;
}

.status-icon.error {
  color: #f56c6c;
}

.status-icon.offline {
  color: #909399;
}

.result-title .client-name {
  font-weight: 600;
  color: #303133;
}

.result-title .command-id {
  font-size: 12px;
  color: #909399;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.result-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.result-meta .time {
  font-size: 12px;
  color: #909399;
}

.expand-icon {
  transition: transform 0.3s;
  color: #909399;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.result-content {
  padding: 12px 16px 16px;
  border-top: 1px solid #ebeef5;
}

/* Terminal 样式 */
.terminal-box {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.terminal-header {
  background: linear-gradient(180deg, #3d3d3d 0%, #2d2d2d 100%);
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.terminal-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.terminal-dot.red {
  background: #ff5f56;
}

.terminal-dot.yellow {
  background: #ffbd2e;
}

.terminal-dot.green {
  background: #27ca40;
}

.terminal-title {
  margin-left: 8px;
  font-size: 12px;
  color: #999;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.terminal-body {
  background: #1e1e1e;
  padding: 16px;
  min-height: 60px;
  max-height: 400px;
  overflow-y: auto;
}

.terminal-loading {
  color: #67c23a;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
}

.cursor-blink {
  animation: blink 1s step-end infinite;
  color: #67c23a;
}

@keyframes blink {
  50% {
    opacity: 0;
  }
}

.terminal-output {
  margin-bottom: 8px;
}

.terminal-output:last-child {
  margin-bottom: 0;
}

.terminal-text {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}

.terminal-text.error {
  color: #f48771;
}

/* ls 命令语法高亮 */
.terminal-text :deep(.ls-dir) {
  color: #5c9eff;
  font-weight: bold;
}

.terminal-text :deep(.ls-link) {
  color: #00d7d7;
}

.terminal-text :deep(.ls-exec) {
  color: #5fff5f;
  font-weight: bold;
}

.terminal-text :deep(.ls-file) {
  color: #e4e4e4;
}

.terminal-text :deep(.ls-perm) {
  color: #9e9e9e;
}

.terminal-text :deep(.ls-total) {
  color: #87d787;
}

.terminal-text :deep(.ls-path) {
  color: #ffaf5f;
  font-weight: bold;
}

.terminal-output.stderr {
  border-left: 3px solid #f56c6c;
  padding-left: 12px;
  margin-left: 4px;
}

.terminal-footer {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #333;
}

.exit-code-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.exit-code-badge.success {
  background: rgba(103, 194, 58, 0.2);
  color: #67c23a;
}

.exit-code-badge.error {
  background: rgba(245, 108, 108, 0.2);
  color: #f56c6c;
}

.terminal-box.error .terminal-body {
  background: #2a1a1a;
}

/* 滚动条样式 */
.terminal-body::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.terminal-body::-webkit-scrollbar-thumb {
  background: #444;
  border-radius: 4px;
}

.terminal-body::-webkit-scrollbar-thumb:hover {
  background: #555;
}

.terminal-body::-webkit-scrollbar-track {
  background: #1e1e1e;
}

/* 加载动画 */
.is-loading {
  animation: rotating 1s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 滚动条样式 */
.results-container::-webkit-scrollbar,
.client-list::-webkit-scrollbar {
  width: 6px;
}

.results-container::-webkit-scrollbar-thumb,
.client-list::-webkit-scrollbar-thumb {
  background: #dcdfe6;
  border-radius: 3px;
}

.results-container::-webkit-scrollbar-thumb:hover,
.client-list::-webkit-scrollbar-thumb:hover {
  background: #c0c4cc;
}
</style>
