import { ref, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

/**
 * WebSocket 钩子
 * @param {string} url - WebSocket 连接地址
 * @param {object} options - 配置选项
 * @returns {object} WebSocket 相关方法和状态
 */
export function useWebSocket(url, options = {}) {
  const ws = ref(null)
  const connected = ref(false)
  const messages = ref([])
  const error = ref(null)

  const {
    onMessage = null,
    onError = null,
    onOpen = null,
    onClose = null,
    reconnect: shouldReconnect = true,
    reconnectInterval = 3000,
    maxReconnectAttempts = 5
  } = options

  let reconnectAttempts = 0
  let reconnectTimer = null
  let reconnectEnabled = shouldReconnect

  const connect = () => {
    try {
      // 处理 WebSocket URL
      let wsUrl = url
      if (!url.startsWith('ws://') && !url.startsWith('wss://')) {
        // 如果是相对路径，根据当前协议构建完整 URL
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const host = window.location.host
        wsUrl = `${protocol}//${host}${url.startsWith('/') ? url : '/' + url}`
      }
      ws.value = new WebSocket(wsUrl)

      ws.value.onopen = () => {
        connected.value = true
        error.value = null
        reconnectAttempts = 0
        if (onOpen) onOpen()
      }

      ws.value.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          messages.value.push(data)
          if (onMessage) {
            onMessage(data)
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.value.onerror = (err) => {
        error.value = err
        if (onError) {
          onError(err)
        } else {
          ElMessage.error('WebSocket 连接错误')
        }
      }

      ws.value.onclose = () => {
        connected.value = false
        if (onClose) {
          onClose()
        }

        // 自动重连
        if (reconnectEnabled && reconnectAttempts < maxReconnectAttempts) {
          reconnectAttempts++
          reconnectTimer = setTimeout(() => {
            console.log(`WebSocket 重连中... (${reconnectAttempts}/${maxReconnectAttempts})`)
            connect()
          }, reconnectInterval)
        } else if (reconnectAttempts >= maxReconnectAttempts) {
          ElMessage.error('WebSocket 连接失败，已达到最大重连次数')
        }
      }
    } catch (err) {
      error.value = err
      ElMessage.error('WebSocket 连接失败')
    }
  }

  const send = (data) => {
    if (ws.value && connected.value) {
      ws.value.send(typeof data === 'string' ? data : JSON.stringify(data))
    } else {
      ElMessage.warning('WebSocket 未连接')
    }
  }

  const disconnect = () => {
    reconnectEnabled = false
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    connected.value = false
  }

  // 初始化连接
  connect()

  // 组件卸载时断开连接
  onUnmounted(() => {
    disconnect()
  })

  return {
    ws,
    connected,
    messages,
    error,
    connect,
    send,
    disconnect
  }
}
