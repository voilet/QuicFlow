import axios from 'axios'
import type { AxiosResponse } from 'axios'

const api = axios.create({
  baseURL: '/api/file',
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器：添加 token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：处理错误
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.error?.message || error.message || '请求失败'
    return Promise.reject(new Error(message))
  }
)

// 类型定义
interface InitUploadRequest {
  filename: string
  file_size: number
  checksum?: string
  content_type?: string
  path?: string
  metadata?: Record<string, any>
  options?: TransferOptions
}

interface TransferOptions {
  overwrite?: boolean
  encryption?: boolean
  compression?: boolean
  verify_checksum?: boolean
  resume?: boolean
  threads?: number
}

interface InitUploadResponse {
  task_id: string
  upload_config: {
    quic_url: string
    chunk_size: number
    max_retries: number
    timeout: number
  }
  status: string
  created_at: string
}

interface UploadChunkRequest {
  task_id: string
  offset: number
  sequence: number
  data: ArrayBuffer
  checksum?: string
}

interface UploadChunkResponse {
  ack: boolean
  received: number
  total_received: number
  progress: number
}

interface CompleteUploadRequest {
  task_id: string
  checksum?: string
  metadata?: Record<string, any>
}

interface CompleteUploadResponse {
  task_id: string
  status: string
  file_info: {
    file_id?: string
    file_name: string
    file_path: string
    file_size: number
    content_type?: string
    checksum?: string
    download_url?: string
  }
  transfer_stats: {
    duration_ms: number
    average_speed: string
    peak_speed?: string
    total_bytes: number
  }
  completed_at: string
}

interface RequestDownloadRequest {
  file_id?: string
  file_path?: string
  local_path?: string
  offset?: number
  options?: TransferOptions
}

interface RequestDownloadResponse {
  task_id: string
  download_config: {
    quic_url: string
    file_info: {
      name: string
      path: string
      size: number
      checksum?: string
    }
    chunk_size: number
    timeout: number
  }
  status: string
  created_at: string
}

interface ProgressUpdate {
  task_id: string
  status: string
  progress: number
  transferred: number
  total: number
  speed: string
  eta: string
  updated_at: string
}

// API 函数
export const fileTransferApi = {
  // 初始化上传
  async initUpload(params: InitUploadRequest): Promise<InitUploadResponse> {
    const response: AxiosResponse<{ success: boolean; data: InitUploadResponse }> =
      await api.post('/upload/init', params)
    return response.data.data
  },

  // 上传分块
  async uploadChunk(taskId: string, params: UploadChunkRequest): Promise<UploadChunkResponse> {
    const formData = new FormData()
    const blob = new Blob([params.data])

    const queryParams = new URLSearchParams({
      task_id: taskId,
      offset: params.offset.toString(),
      sequence: params.sequence.toString()
    })

    const response: AxiosResponse<{ success: boolean; data: UploadChunkResponse }> =
      await api.post(`/upload/chunk?${queryParams}`, blob, {
        headers: {
          'Content-Type': 'application/octet-stream'
        }
      })
    return response.data.data
  },

  // 完成上传
  async completeUpload(params: CompleteUploadRequest): Promise<CompleteUploadResponse> {
    const response: AxiosResponse<{ success: boolean; data: CompleteUploadResponse }> =
      await api.post('/upload/complete', params)
    return response.data.data
  },

  // 取消上传
  async cancelUpload(taskId: string): Promise<void> {
    await api.delete(`/upload/${taskId}`)
  },

  // 请求下载
  async requestDownload(params: RequestDownloadRequest): Promise<RequestDownloadResponse> {
    const response: AxiosResponse<{ success: boolean; data: RequestDownloadResponse }> =
      await api.post('/download/request', params)
    return response.data.data
  },

  // 下载文件
  async downloadFile(taskId: string): Promise<Blob> {
    const response = await api.get(`/download/${taskId}`, {
      responseType: 'blob'
    })
    return response.data
  },

  // 取消下载
  async cancelDownload(taskId: string): Promise<void> {
    await api.delete(`/download/${taskId}`)
  },

  // 获取进度
  async getProgress(taskId: string): Promise<ProgressUpdate> {
    const response: AxiosResponse<{ success: boolean; data: ProgressUpdate }> =
      await api.get(`/transfer/${taskId}/progress`)
    return response.data.data
  },

  // 批量查询状态
  async getBatchStatus(taskIds: string[]): Promise<{ tasks: ProgressUpdate[] }> {
    const response: AxiosResponse<{ success: boolean; data: { tasks: ProgressUpdate[] } }> =
      await api.post('/transfer/batch-status', { task_ids: taskIds })
    return response.data.data
  },

  // 列出传输历史
  async getTransfers(params?: {
    type?: string
    status?: string
    limit?: number
    offset?: number
  }): Promise<any> {
    const response: AxiosResponse<{ success: boolean; data: any }> =
      await api.get('/transfers', { params })
    return response.data.data
  },

  // 获取任务详情
  async getTask(taskId: string): Promise<any> {
    const response: AxiosResponse<{ success: boolean; data: any }> =
      await api.get(`/transfer/${taskId}`)
    return response.data.data
  },

  // 列出文件
  async listFiles(params?: { path?: string; limit?: number }): Promise<any> {
    const response: AxiosResponse<{ success: boolean; data: any }> =
      await api.get('/list', { params })
    return response.data.data
  },

  // 获取文件信息
  async getFileInfo(fileId: string): Promise<any> {
    const response: AxiosResponse<{ success: boolean; data: any }> =
      await api.get(`/info/${fileId}`)
    return response.data.data
  },

  // 删除文件
  async deleteFile(fileId: string): Promise<void> {
    await api.delete(`/${fileId}`)
  },

  // 更新元数据
  async updateMetadata(fileId: string, params: {
    description?: string
    tags?: string[]
    metadata?: Record<string, any>
  }): Promise<void> {
    await api.patch(`/${fileId}/metadata`, params)
  },

  // 获取配额
  async getQuota(): Promise<any> {
    const response: AxiosResponse<{ success: boolean; data: any }> =
      await api.get('/quota')
    return response.data.data
  },

  // 获取配置
  async getConfig(): Promise<any> {
    const response: AxiosResponse<{ success: boolean; data: any }> =
      await api.get('/config')
    return response.data.data
  }
}

export default fileTransferApi
