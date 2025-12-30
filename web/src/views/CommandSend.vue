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

            <!-- Shell 命令输入 -->
            <el-form-item label="Shell命令">
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
              <div class="button-group">
                <el-button
                  type="primary"
                  @click="submitCommand"
                  :loading="submitting"
                  :disabled="clientIdsList.length === 0 || streaming"
                  :icon="VideoPlay"
                >
                  {{ submitting ? '执行中...' : '批量执行' }}
                </el-button>
                <el-button
                  v-if="!streaming"
                  type="success"
                  @click="submitStreamCommand"
                  :disabled="clientIdsList.length === 0 || submitting"
                  :icon="Connection"
                >
                  流式执行 (SSE)
                </el-button>
                <el-button
                  v-else
                  type="danger"
                  @click="handleCancelStream"
                  :icon="CircleClose"
                >
                  中止任务
                </el-button>
              </div>
              <div class="button-tip">
                <el-text type="info" size="small">流式执行：结果实时返回，先完成的先显示</el-text>
              </div>
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
                <el-button
                  type="primary"
                  link
                  @click="toggleAllTerminals"
                >
                  {{ allTerminalsExpanded ? '全部收起' : '全部展开' }}
                </el-button>
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
                        <span class="terminal-actions">
                          <el-button
                            size="small"
                            :type="result.terminalExpanded ? 'info' : 'primary'"
                            link
                            @click.stop="result.terminalExpanded = !result.terminalExpanded"
                          >
                            {{ result.terminalExpanded ? '收起' : '展开' }}
                            <el-icon :class="{ 'is-rotate': result.terminalExpanded }"><ArrowDown /></el-icon>
                          </el-button>
                        </span>
                      </div>
                      <div class="terminal-body" :class="{ collapsed: !result.terminalExpanded }">
                        <!-- 标准输出 -->
                        <div v-if="result.output && result.output.stdout !== undefined && result.output.stdout !== null" class="terminal-output">
                          <pre class="terminal-text" v-html="highlightOutput(result.output.stdout)"></pre>
                        </div>
                        <div v-else-if="result.output && (result.output.stdout === undefined || result.output.stdout === null) && !result.output.stderr" class="terminal-output">
                          <pre class="terminal-text" style="color: #909399;">(无输出)</pre>
                        </div>

                        <!-- 错误输出 -->
                        <div v-if="result.output && result.output.stderr" class="terminal-output stderr">
                          <pre class="terminal-text error" v-html="highlightOutput(result.output.stderr)"></pre>
                        </div>

                        <!-- 退出码 -->
                        <div class="terminal-footer">
                          <span :class="['exit-code-badge', result.output.exit_code === 0 ? 'success' : 'error']">
                            Exit: {{ result.output.exit_code }}
                          </span>
                        </div>
                      </div>
                      <!-- 收起时显示展开提示 -->
                      <div v-if="!result.terminalExpanded" class="terminal-expand-hint" @click="result.terminalExpanded = true">
                        <span>点击展开查看完整输出</span>
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
                      <span class="terminal-actions">
                        <el-button
                          size="small"
                          :type="result.terminalExpanded ? 'info' : 'primary'"
                          link
                          @click.stop="result.terminalExpanded = !result.terminalExpanded"
                        >
                          {{ result.terminalExpanded ? '收起' : '展开' }}
                          <el-icon :class="{ 'is-rotate': result.terminalExpanded }"><ArrowDown /></el-icon>
                        </el-button>
                      </span>
                    </div>
                    <div class="terminal-body" :class="{ collapsed: !result.terminalExpanded }">
                      <pre class="terminal-text error">{{ result.error }}</pre>
                    </div>
                    <div v-if="!result.terminalExpanded" class="terminal-expand-hint" @click="result.terminalExpanded = true">
                      <span>点击展开查看完整输出</span>
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
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  VideoPlay, Loading, Connection,
  CircleCheck, CircleClose, ArrowDown
} from '@element-plus/icons-vue'
import api from '@/api'
import dayjs from 'dayjs'

const route = useRoute()

const formRef = ref()
const submitting = ref(false)
const streaming = ref(false)  // 流式执行状态
const streamController = ref(null)  // SSE 控制器，用于取消请求
const currentTaskId = ref(null)  // 当前流式任务的 task_id
const shellCommand = ref('ls -la')
const executionResults = ref([])
const pollingTimers = ref({})
const clientIdsText = ref('')  // 客户端ID文本，每行一个
const allTerminalsExpanded = ref(true)  // 全局终端展开状态

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
  command_type: 'exec_shell', // 固定为 shell 命令
  payload: JSON.stringify({ command: 'ls -la', timeout: 30 }, null, 2),
  timeout: 30
})

// 验证规则
const rules = {
  client_id: [{ required: true, message: '请选择客户端', trigger: 'change' }]
}

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

// 全部展开/收起终端
function toggleAllTerminals() {
  allTerminalsExpanded.value = !allTerminalsExpanded.value
  executionResults.value.forEach(result => {
    result.terminalExpanded = allTerminalsExpanded.value
  })
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
            terminalExpanded: true,
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

// 流式提交命令 (SSE)
async function submitStreamCommand() {
  // 验证
  if (clientIdsList.value.length === 0) {
    ElMessage.warning('请输入至少一个客户端ID')
    return
  }

  streaming.value = true
  currentTaskId.value = null
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

    // 使用流式 API
    streamController.value = api.sendStreamCommand(
      {
        client_ids: targetClients,
        command_type: form.command_type,
        payload,
        timeout: form.timeout
      },
      // onResult 回调 - 每个结果返回时调用
      (result) => {
        console.log('Stream result:', result)

        // 检查是否是不在线错误
        const offline = isOfflineError(result.error)
        // 结果返回时，从已发送列表移除
        markClientReturned(result.client_id, offline)

        // 不在线的主机不显示在结果中
        if (!offline) {
          const newResult = {
            id: `${result.client_id}-${Date.now()}`,
            client_id: result.client_id,
            status: result.status || 'completed',
            time: timestamp,
            expanded: true,
            terminalExpanded: true,
            output: parseResultOutput(result.result),
            error: result.error,
            command_id: result.command_id
          }
          // 实时添加结果（先返回的在前面）
          executionResults.value = [...executionResults.value, newResult]
        }
      },
      // onStart 回调 - 任务开始时调用（包含 task_id）
      (eventData) => {
        if (eventData.task_id) {
          currentTaskId.value = eventData.task_id
          console.log('Stream task started:', eventData.task_id)
        }
      },
      // onComplete 回调 - 全部完成时调用
      (summary) => {
        console.log('Stream complete:', summary)
        streaming.value = false
        currentTaskId.value = null
        streamController.value = null
        ElMessage.success(`执行完成: 成功 ${summary.success_count}, 失败 ${summary.failed_count}, 耗时 ${summary.duration_ms}ms`)
      },
      // onError 回调 - 发生错误时调用
      (error) => {
        console.error('Stream error:', error)
        streaming.value = false
        currentTaskId.value = null
        streamController.value = null
        ElMessage.error('流式执行失败: ' + (error.message || '未知错误'))
      }
    )
  } catch (error) {
    ElMessage.error('命令执行失败: ' + (error.message || '未知错误'))
    streaming.value = false
    currentTaskId.value = null
    streamController.value = null
  }
}

// 处理取消流式任务
async function handleCancelStream() {
  try {
    await ElMessageBox.confirm(
      '确定要中止当前任务吗？未发送的命令将被取消，已发送的命令将继续执行。',
      '确认中止',
      {
        confirmButtonText: '确定中止',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: false
      }
    )

    // 用户确认中止
    if (currentTaskId.value) {
      try {
        await api.cancelMultiCommand(currentTaskId.value)
        ElMessage.success('任务已中止')
      } catch (error) {
        ElMessage.error('中止任务失败: ' + (error.message || '未知错误'))
      }
    }

    // 关闭流式连接
    if (streamController.value) {
      streamController.value.close()
      streamController.value = null
    }

    streaming.value = false
    currentTaskId.value = null
  } catch (error) {
    // 用户取消操作，不做任何处理
    if (error !== 'cancel') {
      console.error('Cancel stream error:', error)
    }
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

// 将 ANSI 转义码转换为 HTML/CSS 样式
function ansiToHtml(text) {
  if (!text) return ''
  
  // ANSI 颜色码映射
  const ansiColors = {
    // 前景色
    '30': '#000000', '31': '#cd0000', '32': '#00cd00', '33': '#cdcd00',
    '34': '#0000ee', '35': '#cd00cd', '36': '#00cdcd', '37': '#e5e5e5',
    // 背景色
    '40': '#000000', '41': '#cd0000', '42': '#00cd00', '43': '#cdcd00',
    '44': '#0000ee', '45': '#cd00cd', '46': '#00cdcd', '47': '#ffffff'
  }
  
  // 转义 HTML 特殊字符
  const escapeHtml = (str) => {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
  }
  
  let html = ''
  let currentStyle = {
    color: null,
    backgroundColor: null,
    bold: false
  }
  
  // 处理 ANSI 码（包括 \x1b[ 和直接 [ 开头的格式）
  const ansiPattern = /(\x1b\[|\u001b\[|\[)([0-9;]*)m/g
  let lastIndex = 0
  let match
  
  const closeCurrentSpan = () => {
    if (currentStyle.color || currentStyle.backgroundColor || currentStyle.bold) {
      html += '</span>'
    }
  }
  
  const openSpan = (style) => {
    const styles = []
    if (style.color) styles.push(`color: ${style.color}`)
    if (style.backgroundColor) styles.push(`background-color: ${style.backgroundColor}`)
    if (style.bold) styles.push('font-weight: bold')
    if (styles.length > 0) {
      html += `<span style="${styles.join('; ')}">`
    }
  }
  
  while ((match = ansiPattern.exec(text)) !== null) {
    // 添加匹配前的文本
    const beforeText = text.substring(lastIndex, match.index)
    if (beforeText) {
      const escaped = escapeHtml(beforeText)
      if (currentStyle.color || currentStyle.backgroundColor || currentStyle.bold) {
        html += escaped
      } else {
        html += escaped
      }
    }
    
    // 处理 ANSI 码
    const codes = match[2].split(';').filter(c => c !== '')
    let newStyle = { ...currentStyle }
    
    for (const code of codes) {
      if (code === '0' || code === '') {
        // 重置
        newStyle = { color: null, backgroundColor: null, bold: false }
      } else if (code === '1') {
        // 粗体
        newStyle.bold = true
      } else if (code in ansiColors) {
        const color = ansiColors[code]
        if (code >= '30' && code <= '37') {
          // 前景色
          newStyle.color = color
        } else if (code >= '40' && code <= '47') {
          // 背景色
          newStyle.backgroundColor = color
        }
      }
    }
    
    // 如果样式改变，关闭旧 span，打开新 span
    if (JSON.stringify(newStyle) !== JSON.stringify(currentStyle)) {
      closeCurrentSpan()
      currentStyle = newStyle
      openSpan(currentStyle)
    }
    
    lastIndex = match.index + match[0].length
  }
  
  // 添加剩余文本
  const remainingText = text.substring(lastIndex)
  if (remainingText) {
    html += escapeHtml(remainingText)
  }
  
  // 关闭未关闭的标签
  closeCurrentSpan()
  
  return html
}

// 高亮输出内容
function highlightOutput(text) {
  // 处理非字符串类型
  if (text === null || text === undefined) return ''
  if (typeof text !== 'string') {
    text = String(text)
  }
  
  // 即使原始内容为空或只有空格，也要处理（二维码输出可能主要是空格）
  if (!text) return ''

  // 检查是否包含 ANSI 转义码（用于二维码等）
  const hasAnsi = /(\x1b\[|\u001b\[|\[)[0-9;]*m/.test(text)
  
  if (hasAnsi) {
    // 如果包含 ANSI 码，转换为 HTML
    return ansiToHtml(text)
  }

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

onMounted(() => {
  // 从 URL 参数获取客户端 ID
  if (route.query.client_ids) {
    clientIdsText.value = route.query.client_ids
  } else if (route.query.client_id) {
    clientIdsText.value = route.query.client_id
  }
})

onUnmounted(() => {
  // 清理所有轮询定时器
  Object.values(pollingTimers.value).forEach(timer => clearInterval(timer))
  // 取消流式请求
  if (streamController.value) {
    streamController.value.close()
    streamController.value = null
  }
  currentTaskId.value = null
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

/* 按钮组样式 */
.button-group {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.button-tip {
  margin-top: 8px;
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

.terminal-actions {
  margin-left: auto;
}

.terminal-actions .is-rotate {
  transform: rotate(180deg);
}

.terminal-body {
  background: #1e1e1e;
  padding: 16px;
  min-height: 60px;
}

.terminal-body.collapsed {
  display: none;
}

.terminal-expand-hint {
  background: #2a2a2a;
  padding: 12px 16px;
  text-align: center;
  color: #888;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.terminal-expand-hint:hover {
  background: #333;
  color: #aaa;
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
  line-height: 1.2;
  color: #d4d4d4;
  white-space: pre;
  word-break: keep-all;
  overflow-x: auto;
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
