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
  getClients() {
    return request.get('/clients')
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

  // 多播命令（同时发送到多个客户端）
  sendMultiCommand(data) {
    return request.post('/command/multi', data)
  },

  getCommand(commandId) {
    return request.get(`/command/${commandId}`)
  },

  getCommands(params) {
    return request.get('/commands', { params })
  }
}

export default api
