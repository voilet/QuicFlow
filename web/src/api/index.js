import axios from 'axios'
import { ElMessage } from 'element-plus'

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

// 响应拦截器
request.interceptors.response.use(
  response => response.data,
  error => {
    ElMessage.error(error.message || '请求失败')
    return Promise.reject(error)
  }
)

// API 接口
export const api = {
  // 客户端管理
  getClients(params = {}) {
    return request.get('/clients', { params })
  },

  getClient(clientId) {
    return request.get(`/clients/${clientId}`)
  },

  // 消息发送
  sendMessage(data) {
    return request.post('/send', data)
  },

  broadcast(data) {
    return request.post('/broadcast', data)
  },

  // 命令管理
  sendCommand(data) {
    return request.post('/command', data)
  },

  // 多播命令（同时发送到多个客户端）- 等待所有完成
  sendMultiCommand(data) {
    return request.post('/command/multi', data)
  },

  /**
   * 流式多播命令 (SSE) - 实时返回结果
   * @param {Object} data - 请求数据 {client_ids, command_type, payload, timeout}
   * @param {Function} onResult - 收到单个结果时的回调 (result) => {}
   * @param {Function} onStart - 任务开始时的回调 (eventData) => {}，包含 task_id
   * @param {Function} onComplete - 全部完成时的回调 (summary) => {}
   * @param {Function} onError - 发生错误时的回调 (error) => {}
   * @returns {Object} - 返回可取消的对象，可调用 .close() 取消
   */
  sendStreamCommand(data, onResult, onStart, onComplete, onError) {
    // 使用 fetch + ReadableStream 实现 SSE (因为需要 POST)
    const controller = new AbortController()

    fetch('/api/command/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
      signal: controller.signal,
    })
    .then(response => {
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      function processStream() {
        return reader.read().then(({ done, value }) => {
          if (done) {
            return
          }

          buffer += decoder.decode(value, { stream: true })

          // 解析 SSE 格式: "data: {...}\n\n"
          const lines = buffer.split('\n\n')
          buffer = lines.pop() // 保留未完成的部分

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              try {
                const eventData = JSON.parse(line.slice(6))

                if (eventData.type === 'start' && onStart) {
                  onStart(eventData)
                } else if (eventData.type === 'result' && onResult) {
                  onResult(eventData.result)
                } else if (eventData.type === 'complete' && onComplete) {
                  onComplete(eventData.summary)
                }
              } catch (e) {
                console.error('Failed to parse SSE event:', e, line)
              }
            }
          }

          return processStream()
        })
      }

      return processStream()
    })
    .catch(error => {
      if (error.name !== 'AbortError' && onError) {
        onError(error)
      }
    })

    // 返回一个可以取消的对象
    return {
      close: () => controller.abort()
    }
  },

  getCommand(commandId) {
    return request.get(`/command/${commandId}`)
  },

  getCommands(params) {
    return request.get('/commands', { params })
  },

  // 取消多播任务
  cancelMultiCommand(taskId) {
    return request.post(`/command/multi/${taskId}/cancel`)
  },

  // ===== 终端 API =====

  // 获取终端会话列表
  getTerminalSessions() {
    return request.get('/terminal/sessions')
  },

  // 关闭终端会话
  closeTerminalSession(sessionId) {
    return request.delete(`/terminal/sessions/${sessionId}`)
  },

  // ===== 审计 API =====

  // 获取审计命令列表
  getAuditCommands(params = {}) {
    return request.get('/audit/commands', { params })
  },

  // 获取会话的审计命令
  getAuditCommandsBySession(sessionId) {
    return request.get(`/audit/commands/${sessionId}`)
  },

  // 获取审计统计
  getAuditStats() {
    return request.get('/audit/stats')
  },

  // 导出审计日志
  exportAuditLogs(format = 'json') {
    return `/api/audit/export?format=${format}`
  },

  // 清理旧审计记录
  cleanupAuditRecords(days = 90) {
    return request.delete('/audit/cleanup', { params: { days } })
  },

  // ===== 录像 API =====

  // 获取录像列表
  getRecordings(params = {}) {
    return request.get('/recordings', { params })
  },

  // 获取录像详情
  getRecording(id) {
    return request.get(`/recordings/${id}`)
  },

  // 下载录像
  getRecordingDownloadUrl(id) {
    return `/api/recordings/${id}/download`
  },

  // 删除录像
  deleteRecording(id) {
    return request.delete(`/recordings/${id}`)
  },

  // 获取录像统计
  getRecordingStats() {
    return request.get('/recordings/stats')
  },

  // 清理旧录像
  cleanupRecordings(days = 30) {
    return request.delete('/recordings/cleanup', { params: { days } })
  }
}

export { request }
export default api
