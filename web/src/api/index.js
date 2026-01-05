import axios from 'axios'
import { ElMessage } from 'element-plus'

// Token 管理
const TOKEN_KEY = 'x-token'

export const getToken = () => {
  return localStorage.getItem(TOKEN_KEY) || document.cookie.match(new RegExp(`(^| )${TOKEN_KEY}=([^;]*)(;|$)`))?.[2] || ''
}

export const setToken = (token) => {
  localStorage.setItem(TOKEN_KEY, token)
  document.cookie = `${TOKEN_KEY}=${token};path=/`
}

export const removeToken = () => {
  localStorage.removeItem(TOKEN_KEY)
  document.cookie = `${TOKEN_KEY}=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/`
}

// 创建 axios 实例
const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

// 请求拦截器
request.interceptors.request.use(
  config => {
    const token = getToken()
    if (token) {
      config.headers['x-token'] = token
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  response => {
    // 处理新 token
    const newToken = response.headers['new-token']
    if (newToken) {
      setToken(newToken)
    }
    return response.data
  },
  error => {
    if (error.response) {
      const { status } = error.response
      if (status === 401) {
        removeToken()
        window.location.href = '/login'
      }
      ElMessage.error(error.response.data?.msg || error.message || '请求失败')
    } else {
      ElMessage.error(error.message || '请求失败')
    }
    return Promise.reject(error)
  }
)

// API 接口
export const api = {
  // ===== 认证 API =====

  // 用户登录
  login(data) {
    return request.post('/base/login', data)
  },

  // 用户登出
  logout() {
    return request.post('/user/logout')
  },

  // 获取当前用户信息
  getUserInfo() {
    return request.get('/user/info')
  },

  // 修改密码
  changePassword(data) {
    return request.put('/user/password', data)
  },

  // 用户管理（管理员）
  getUserList(params) {
    return request.get('/user/list', { params })
  },

  createUser(data) {
    return request.post('/user/create', data)
  },

  updateUser(data) {
    return request.put('/user/update', data)
  },

  deleteUser(id) {
    return request.delete('/user/delete', { params: { id } })
  },

  resetPassword(data) {
    return request.put('/user/reset-password', data)
  },

  // 角色管理（管理员）
  getAuthorityList() {
    return request.get('/authority/list')
  },

  createAuthority(data) {
    return request.post('/authority/create', data)
  },

  updateAuthority(data) {
    return request.put('/authority/update', data)
  },

  deleteAuthority(id) {
    return request.delete('/authority/delete', { params: { id } })
  },

  copyAuthority(data) {
    return request.post('/authority/copy', data)
  },

  // 菜单管理（管理员）
  getMenuList() {
    return request.get('/menu/list')
  },

  createMenu(data) {
    return request.post('/menu/create', data)
  },

  updateMenu(data) {
    return request.put('/menu/update', data)
  },

  deleteMenu(id) {
    return request.delete('/menu/delete', { params: { id } })
  },

  getMenuByAuthority(authorityId) {
    return request.get('/menu/by-authority', { params: { authority_id: authorityId } })
  },

  setMenuAuthority(data) {
    return request.post('/menu/set-authority', data)
  },

  // 客户端管理
  getClients(params = {}) {
    return request.get('/clients', { params })
  },

  getClient(clientId) {
    return request.get(`/clients/${clientId}`)
  },

  // 获取客户端硬件信息（从数据库）
  getClientHardwareInfo(clientId) {
    return request.get(`/hardware/devices/${clientId}/hardware`)
  },

  // ===== 硬件设备管理 API =====

  // 获取设备列表
  getDevices(params = {}) {
    return request.get('/hardware/devices', { params })
  },

  // 获取单个设备
  getDevice(clientId) {
    return request.get(`/hardware/devices/${clientId}`)
  },

  // 获取设备统计
  getDeviceStats() {
    return request.get('/hardware/devices/stats')
  },

  // 按主机名搜索设备
  searchDevicesByHostname(keyword, params = {}) {
    return request.get('/hardware/devices/search/by-hostname', { params: { q: keyword, ...params } })
  },

  // 批量查询设备（按 client_id 列表）
  batchQueryDevices(clientIds) {
    return request.post('/hardware/devices/batch-query', { client_ids: clientIds })
  },

  // 批量删除设备
  batchDeleteDevices(clientIds) {
    return request.post('/hardware/devices/batch-delete', { client_ids: clientIds })
  },

  // 批量更新设备状态
  batchUpdateDeviceStatus(clientIds, status) {
    return request.post('/hardware/devices/batch-update-status', { client_ids: clientIds, status })
  },

  // 删除单个设备
  deleteDevice(clientId) {
    return request.delete(`/hardware/devices/${clientId}`)
  },

  // 更新设备状态
  updateDeviceStatus(clientId, status) {
    return request.put(`/hardware/devices/${clientId}/status`, { status })
  },

  // 获取设备历史
  getDeviceHistory(clientId, limit = 50) {
    return request.get(`/hardware/devices/${clientId}/history`, { params: { limit } })
  },

  // 标记离线设备
  markOfflineDevices(timeoutMinutes) {
    return request.post('/hardware/devices/mark-offline', { timeout_minutes: timeoutMinutes })
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
  },

  // ===== 发布系统 API =====

  // 项目管理
  getProjects() {
    return request.get('/release/projects')
  },

  getProject(id) {
    return request.get(`/release/projects/${id}`)
  },

  createProject(data) {
    return request.post('/release/projects', data)
  },

  updateProject(id, data) {
    return request.put(`/release/projects/${id}`, data)
  },

  deleteProject(id) {
    return request.delete(`/release/projects/${id}`)
  },

  // 环境管理
  getEnvironments(projectId) {
    return request.get(`/release/projects/${projectId}/environments`)
  },

  getEnvironment(id) {
    return request.get(`/release/environments/${id}`)
  },

  createEnvironment(projectId, data) {
    return request.post(`/release/projects/${projectId}/environments`, data)
  },

  updateEnvironment(id, data) {
    return request.put(`/release/environments/${id}`, data)
  },

  deleteEnvironment(id) {
    return request.delete(`/release/environments/${id}`)
  },

  // 目标管理
  getTargets(envId) {
    return request.get(`/release/environments/${envId}/targets`)
  },

  getTarget(id) {
    return request.get(`/release/targets/${id}`)
  },

  createTarget(envId, data) {
    return request.post(`/release/environments/${envId}/targets`, data)
  },

  updateTarget(id, data) {
    return request.put(`/release/targets/${id}`, data)
  },

  deleteTarget(id) {
    return request.delete(`/release/targets/${id}`)
  },

  // 流水线管理
  getPipelines(projectId) {
    return request.get(`/release/projects/${projectId}/pipelines`)
  },

  getPipeline(id) {
    return request.get(`/release/pipelines/${id}`)
  },

  createPipeline(projectId, data) {
    return request.post(`/release/projects/${projectId}/pipelines`, data)
  },

  updatePipeline(id, data) {
    return request.put(`/release/pipelines/${id}`, data)
  },

  deletePipeline(id) {
    return request.delete(`/release/pipelines/${id}`)
  },

  // 发布管理
  getReleases(params = {}) {
    return request.get('/release/deploys', { params })
  },

  getRelease(id) {
    return request.get(`/release/deploys/${id}`)
  },

  createRelease(data) {
    return request.post('/release/deploys', data)
  },

  startRelease(id) {
    return request.post(`/release/deploys/${id}/start`)
  },

  cancelRelease(id) {
    return request.post(`/release/deploys/${id}/cancel`)
  },

  rollbackRelease(id, data = {}) {
    return request.post(`/release/deploys/${id}/rollback`, data)
  },

  promoteRelease(id) {
    return request.post(`/release/deploys/${id}/promote`)
  },

  // 快捷操作
  installService(data) {
    return request.post('/release/install', data)
  },

  updateService(data) {
    return request.post('/release/update', data)
  },

  uninstallService(data) {
    return request.post('/release/uninstall', data)
  },

  // 审批管理
  getApprovals() {
    return request.get('/release/approvals')
  },

  approveRelease(id) {
    return request.post(`/release/approvals/${id}/approve`)
  },

  rejectRelease(id, data = {}) {
    return request.post(`/release/approvals/${id}/reject`, data)
  },

  // ===== 版本管理 =====

  // 获取项目版本列表
  getVersions(projectId) {
    return request.get(`/release/projects/${projectId}/versions`)
  },

  // 获取版本详情
  getVersion(id) {
    return request.get(`/release/versions/${id}`)
  },

  // 创建版本
  createVersion(projectId, data) {
    return request.post(`/release/projects/${projectId}/versions`, data)
  },

  // 更新版本
  updateVersion(id, data) {
    return request.put(`/release/versions/${id}`, data)
  },

  // 删除版本
  deleteVersion(id) {
    return request.delete(`/release/versions/${id}`)
  },

  // ===== 部署任务管理 =====

  // 获取项目部署任务列表
  getDeployTasks(projectId, params = {}) {
    return request.get(`/release/projects/${projectId}/tasks`, { params })
  },

  // 获取部署任务详情
  getDeployTask(id) {
    return request.get(`/release/tasks/${id}`)
  },

  // 创建部署任务
  createDeployTask(data) {
    return request.post('/release/tasks', data)
  },

  // 开始部署任务
  startDeployTask(id) {
    return request.post(`/release/tasks/${id}/start`)
  },

  // 取消部署任务
  cancelDeployTask(id) {
    return request.post(`/release/tasks/${id}/cancel`)
  },

  // 暂停部署任务
  pauseDeployTask(id) {
    return request.post(`/release/tasks/${id}/pause`)
  },

  // 金丝雀全量发布
  promoteDeployTask(id) {
    return request.post(`/release/tasks/${id}/promote`)
  },

  // 回滚部署任务
  rollbackDeployTask(id) {
    return request.post(`/release/tasks/${id}/rollback`)
  },

  // ===== 部署任务实时日志 =====

  // 获取部署任务的执行日志
  getDeployTaskLogs(taskId, params = {}) {
    return request.get(`/release/tasks/${taskId}/logs`, { params })
  },

  // 获取指定客户端的容器日志（按 Container ID）
  getClientContainerLogs(taskId, clientId, params = {}) {
    return request.get(`/release/tasks/${taskId}/clients/${clientId}/container-logs`, { params })
  },

  /**
   * 流式获取部署任务日志 (SSE) - 实时更新
   * @param {string} taskId - 任务 ID
   * @param {Object} params - 请求参数 {client_id?}
   * @param {Function} onLog - 收到日志时的回调 (log) => {}
   * @param {Function} onStatus - 收到状态更新时的回调 (status) => {}
   * @param {Function} onDone - 任务完成时的回调 (data) => {}
   * @param {Function} onError - 发生错误时的回调 (error) => {}
   * @returns {Object} - 返回可取消的对象，可调用 .close() 关闭连接
   */
  streamDeployTaskLogs(taskId, params, onLog, onStatus, onDone, onError) {
    const queryParams = new URLSearchParams()
    if (params.client_id) {
      queryParams.append('client_id', params.client_id)
    }

    const queryString = queryParams.toString()
    const url = `/api/release/tasks/${taskId}/logs/stream${queryString ? '?' + queryString : ''}`

    const eventSource = new EventSource(url)

    eventSource.addEventListener('log', (event) => {
      try {
        const data = JSON.parse(event.data)
        if (onLog) onLog(data)
      } catch (e) {
        console.error('Failed to parse log event:', e)
      }
    })

    eventSource.addEventListener('status', (event) => {
      try {
        const data = JSON.parse(event.data)
        if (onStatus) onStatus(data)
      } catch (e) {
        console.error('Failed to parse status event:', e)
      }
    })

    eventSource.addEventListener('done', (event) => {
      try {
        const data = JSON.parse(event.data)
        if (onDone) onDone(data)
        eventSource.close()
      } catch (e) {
        console.error('Failed to parse done event:', e)
      }
    })

    eventSource.onerror = (error) => {
      if (onError) {
        onError(error)
      }
    }

    return {
      close: () => eventSource.close()
    }
  },

  // ===== 部署日志 =====

  // 获取部署日志列表
  getDeployLogs(params = {}) {
    return request.get('/release/logs', { params })
  },

  // 获取部署日志详情
  getDeployLog(id) {
    return request.get(`/release/logs/${id}`)
  },

  // 获取项目的部署日志
  getProjectDeployLogs(projectId, params = {}) {
    return request.get(`/release/projects/${projectId}/logs`, { params })
  },

  // 获取项目的部署统计
  getProjectDeployStats(projectId, params = {}) {
    return request.get(`/release/projects/${projectId}/stats`, { params })
  },

  // 获取整体部署统计
  getDeployStats(params = {}) {
    return request.get('/release/stats', { params })
  },

  // 验证脚本语法
  validateScript(data) {
    return request.post('/release/validate-script', data)
  },

  // 获取 Git 仓库版本信息（tags、branches、commits）
  getGitVersions(data) {
    return request.post('/release/git-versions', data)
  },

  // ===== 容器日志 API =====

  // 获取容器日志（一次性获取）
  getContainerLogs(data) {
    return request.post('/containers/logs', data)
  },

  // ===== 性能分析 API =====

  // 启动 CPU 采集
  startCPUProfile(data) {
    return request.post('/profiling/cpu', data)
  },

  // 上传已采集的 CPU profile（用于标准 pprof 端点采集的数据）
  uploadCPUProfile(formData) {
    return request.post('/profiling/cpu/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 采集内存快照
  captureMemoryProfile(data) {
    return request.post('/profiling/memory', data)
  },

  // 采集 Goroutine 快照
  captureGoroutineProfile(data) {
    return request.post('/profiling/goroutine', data)
  },

  // 获取采集列表
  getProfiles(params = {}) {
    return request.get('/profiling/list', { params })
  },

  // 获取单个采集
  getProfile(id) {
    return request.get(`/profiling/profiles/${id}`)
  },

  // 获取火焰图 URL
  getFlameGraphUrl(id) {
    return `/api/profiling/flamegraph/${id}`
  },

  // 生成火焰图
  generateFlameGraph(id) {
    return request.post(`/profiling/flamegraph/${id}/generate`)
  },

  // 分析采集
  analyzeProfile(id) {
    return request.post(`/profiling/analyze/${id}`)
  },

  // 删除采集
  deleteProfile(id) {
    return request.delete(`/profiling/profiles/${id}`)
  },

  // 清理旧采集
  cleanupProfiles(days = 7) {
    return request.post('/profiling/cleanup', null, { params: { days } })
  },

  // 下载采集文件
  downloadProfile(id) {
    return `/api/profiling/profiles/${id}/download`
  },

  /**
   * 流式获取容器日志 (SSE) - 实时更新
   * @param {Object} params - 请求参数 {client_id, container_id?, container_name?, tail?, timestamps?}
   * @param {Function} onLogs - 收到日志时的回调 (event) => {}
   * @param {Function} onStart - 开始时的回调 (event) => {}
   * @param {Function} onError - 发生错误时的回调 (event) => {}
   * @returns {Object} - 返回可取消的对象，可调用 .close() 关闭连接
   */
  streamContainerLogs(params, onLogs, onStart, onError) {
    // 构建 URL 查询参数
    const queryParams = new URLSearchParams()
    queryParams.append('client_id', params.client_id)
    if (params.container_id) {
      queryParams.append('container_id', params.container_id)
    }
    if (params.container_name) {
      queryParams.append('container_name', params.container_name)
    }
    if (params.tail) {
      queryParams.append('tail', params.tail.toString())
    }
    if (params.timestamps) {
      queryParams.append('timestamps', 'true')
    }

    const url = `/api/containers/logs/stream?${queryParams.toString()}`

    // 使用 EventSource 实现 SSE (GET 请求)
    const eventSource = new EventSource(url)

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'start' && onStart) {
          onStart(data)
        } else if (data.type === 'logs' && onLogs) {
          onLogs(data)
        } else if (data.type === 'error' && onError) {
          onError(data)
        }
      } catch (e) {
        console.error('Failed to parse SSE event:', e, event.data)
      }
    }

    eventSource.onerror = (error) => {
      if (onError) {
        onError({ type: 'error', error: 'Connection error' })
      }
      // EventSource 会自动重连，如果不需要可以在这里关闭
    }

    // 返回一个可以关闭的对象
    return {
      close: () => {
        eventSource.close()
      }
    }
  }
}

export { request }
export default api
