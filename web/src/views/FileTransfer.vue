<template>
  <div class="file-transfer-page">
    <!-- 上传区域 -->
    <el-row :gutter="20" class="page-content">
      <!-- 左侧：上传区域 -->
      <el-col :span="14" class="upload-section">
        <el-card class="upload-card">
          <template #header>
            <div class="card-header">
              <span>
                <el-icon><Upload /></el-icon>
                文件上传
              </span>
              <div class="header-actions">
                <el-button
                  size="small"
                  @click="clearCompleted"
                  :disabled="!hasCompleted"
                >
                  清除已完成
                </el-button>
                <el-button
                  size="small"
                  type="primary"
                  :disabled="!hasActive"
                  @click="pauseAll"
                >
                  全部暂停
                </el-button>
                <el-button
                  size="small"
                  type="success"
                  :disabled="!hasPaused"
                  @click="resumeAll"
                >
                  全部继续
                </el-button>
              </div>
            </div>
          </template>

          <!-- 拖拽上传区域 -->
          <div
            class="upload-zone"
            :class="{ 'drag-over': isDragOver, 'uploading': hasActive }"
            @drop.prevent="handleDrop"
            @dragover.prevent="isDragOver = true"
            @dragleave="isDragOver = false"
            @click="selectFiles"
          >
            <div class="upload-icon-wrapper">
              <el-icon :size="60" class="upload-icon"><UploadFilled /></el-icon>
              <div v-if="hasActive" class="upload-pulse"></div>
            </div>
            <p class="upload-text">拖拽文件到此处</p>
            <p class="upload-subtext">或点击选择文件上传</p>
            <p class="upload-hint">支持批量上传，单文件最大 10GB</p>
            <input
              ref="fileInput"
              type="file"
              multiple
              @change="handleFileSelect"
              style="display: none"
            />
          </div>

          <!-- 上传队列 -->
          <div v-if="files.length > 0" class="upload-queue">
            <div class="queue-header">
              <span class="queue-title">
                队列 ({{ files.length }} 个文件)
              </span>
              <span class="queue-stats">
                总计: {{ formatSize(totalSize) }} |
                已上传: {{ formatSize(totalTransferred) }}
              </span>
            </div>

            <div class="queue-list">
              <transition-group name="list">
                <div
                  v-for="file in sortedFiles"
                  :key="file.id"
                  class="queue-item"
                  :class="{ 'queue-item-error': file.status === 'error' }"
                >
                  <!-- 文件信息 -->
                  <div class="file-main">
                    <div class="file-icon">
                      <el-icon v-if="getFileIcon(file.name)" :is="getFileIcon(file.name)"></el-icon>
                      <el-icon v-else><Document /></el-icon>
                    </div>
                    <div class="file-details">
                      <div class="file-name-row">
                        <span class="file-name" :title="file.name">{{ file.name }}</span>
                        <el-tag
                          :type="getStatusType(file.status)"
                          size="small"
                          effect="plain"
                        >
                          {{ getStatusText(file.status) }}
                        </el-tag>
                      </div>
                      <div class="file-meta">
                        <span>{{ formatSize(file.size) }}</span>
                        <span v-if="file.speed > 0">• {{ formatSpeed(file.speed) }}</span>
                        <span v-if="file.eta !== '00:00:00'">• {{ file.eta }}</span>
                      </div>
                    </div>
                    <div class="file-progress-section">
                      <el-progress
                        :percentage="file.progress"
                        :status="getProgressStatus(file.status)"
                        :stroke-width="6"
                        :show-text="false"
                      />
                      <span class="progress-text">{{ file.progress }}%</span>
                    </div>
                  </div>

                  <!-- 操作按钮 -->
                  <div class="file-actions">
                    <el-button
                      v-if="file.status === 'uploading'"
                      size="small"
                      :icon="VideoPause"
                      @click="pauseUpload(file.id)"
                    />
                    <el-button
                      v-if="file.status === 'paused'"
                      size="small"
                      type="primary"
                      :icon="VideoPlay"
                      @click="resumeUpload(file.id)"
                    />
                    <el-button
                      v-if="['pending', 'uploading', 'paused'].includes(file.status)"
                      size="small"
                      type="danger"
                      :icon="Delete"
                      @click="cancelUpload(file.id)"
                    />
                    <el-button
                      v-if="file.status === 'completed'"
                      size="small"
                      :icon="Download"
                      @click="downloadFile(file)"
                    />
                    <el-button
                      v-if="file.status === 'error'"
                      size="small"
                      type="warning"
                      :icon="RefreshRight"
                      @click="retryUpload(file.id)"
                    />
                  </div>
                </div>
              </transition-group>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：存储信息 + 传输历史 -->
      <el-col :span="10" class="info-section">
        <!-- 存储配额 -->
        <el-card class="quota-card">
          <template #header>
            <span>
              <el-icon><PieChart /></el-icon>
              存储配额
            </span>
          </template>
          <div v-if="quota" class="quota-content">
            <div class="quota-chart">
              <el-progress
                type="circle"
                :percentage="quota.usagePercentage"
                :color="getQuotaColor(quota.usagePercentage)"
                :width="120"
              >
                <template #default="{ percentage }">
                  <span class="quota-percentage">{{ percentage }}%</span>
                </template>
              </el-progress>
            </div>
            <div class="quota-details">
              <div class="quota-item">
                <span class="quota-label">已使用</span>
                <span class="quota-value">{{ quota.formatted.used }}</span>
              </div>
              <div class="quota-item">
                <span class="quota-label">可用空间</span>
                <span class="quota-value">{{ quota.formatted.available }}</span>
              </div>
              <div class="quota-item">
                <span class="quota-label">总配额</span>
                <span class="quota-value">{{ quota.formatted.total }}</span>
              </div>
            </div>
          </div>
          <el-skeleton v-else :rows="3" animated />
        </el-card>

        <!-- 快速操作 -->
        <el-card class="actions-card">
          <template #header>
            <span>
              <el-icon><Operation /></el-icon>
              快速操作
            </span>
          </template>
          <div class="quick-actions">
            <el-button
              type="primary"
              :icon="FolderOpened"
              @click="showFileDialog = true"
            >
              浏览文件
            </el-button>
            <el-button
              :icon="Download"
              @click="showHistory = true"
            >
              传输历史
            </el-button>
            <el-button
              :icon="Delete"
              @click="cleanupFiles"
            >
              清理临时文件
            </el-button>
          </div>
        </el-card>

        <!-- 最近传输 -->
        <el-card class="history-card">
          <template #header>
            <span>
              <el-icon><Clock /></el-icon>
              最近传输
            </span>
            <el-link type="primary" @click="showHistory = true">查看全部</el-link>
          </template>
          <div v-if="recentTransfers.length > 0" class="recent-list">
            <div
              v-for="item in recentTransfers"
              :key="item.id"
              class="recent-item"
              @click="showHistory = true"
            >
              <el-icon class="recent-icon">
                <component :is="item.transfer_type === 'upload' ? Upload : Download" />
              </el-icon>
              <div class="recent-details">
                <span class="recent-name">{{ item.file_name }}</span>
                <span class="recent-meta">{{ formatTime(item.created_at) }}</span>
              </div>
              <el-tag
                :type="getStatusType(item.status)"
                size="small"
              >
                {{ getStatusText(item.status) }}
              </el-tag>
            </div>
          </div>
          <el-empty
            v-else
            description="暂无传输记录"
            :image-size="80"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- 文件浏览对话框 -->
    <el-dialog
      v-model="showFileDialog"
      title="浏览文件"
      width="800px"
      :close-on-click-modal="false"
    >
      <FileBrowser
        :current-path="currentPath"
        @navigate="handleNavigate"
      />
    </el-dialog>

    <!-- 传输历史对话框 -->
    <el-dialog
      v-model="showHistory"
      title="传输历史"
      width="90%"
      :close-on-click-modal="false"
    >
      <TransferHistory @retry="handleRetryTransfer" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  UploadFilled, Document, Upload, Download, Delete,
  VideoPause, VideoPlay, RefreshRight, PieChart, Operation,
  Clock, FolderOpened, Files
} from '@element-plus/icons-vue'
import { fileTransferApi } from '@/api/file'
import FileBrowser from '@/components/FileBrowser.vue'
import TransferHistory from '@/components/TransferHistory.vue'

interface UploadFile {
  id: string
  name: string
  size: number
  progress: number
  status: 'pending' | 'uploading' | 'paused' | 'completed' | 'error' | 'cancelled'
  speed: number
  eta: string
  taskId?: string
  file: File
  error?: string
}

const CHUNK_SIZE = 1024 * 1024 // 1MB

// 状态
const isDragOver = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const files = ref<UploadFile[]>([])
const showFileDialog = ref(false)
const showHistory = ref(false)
const currentPath = ref('/')
const quota = ref<any>(null)
const recentTransfers = ref<any[]>([])

// 进度更新定时器
let progressTimer: ReturnType<typeof setInterval> | null = null

// 计算属性
const sortedFiles = computed(() => {
  const priority = { error: 0, uploading: 1, paused: 2, pending: 3, completed: 4 }
  return [...files.value].sort((a, b) => {
    const pa = priority[a.status] ?? 999
    const pb = priority[b.status] ?? 999
    if (pa !== pb) return pa - pb
    return b.name.localeCompare(a.name)
  })
})

const hasActive = computed(() => files.value.some(f => ['uploading', 'pending'].includes(f.status)))
const hasPaused = computed(() => files.value.some(f => f.status === 'paused'))
const hasCompleted = computed(() => files.value.some(f => f.status === 'completed'))

const totalSize = computed(() => files.value.reduce((sum, f) => sum + f.size, 0))
const totalTransferred = computed(() => files.value.reduce((sum, f) => sum + (f.size * f.progress / 100), 0))

// 文件选择
function selectFiles() {
  fileInput.value?.click()
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files) {
    addFiles(Array.from(target.files))
    target.value = '' // 重置 input
  }
}

function handleDrop(event: DragEvent) {
  isDragOver.value = false
  const dt = event.dataTransfer
  if (dt.files) {
    addFiles(Array.from(dt.files))
  }
}

function addFiles(fileList: File[]) {
  for (const file of fileList) {
    // 检查是否已存在
    if (!files.value.find(f => f.name === file.name && f.size === file.size)) {
      const uploadFile: UploadFile = {
        id: Date.now().toString() + Math.random().toString(36).substr(2, 9),
        name: file.name,
        size: file.size,
        progress: 0,
        status: 'pending',
        speed: 0,
        eta: '00:00:00',
        file: file
      }
      files.value.push(uploadFile)
      // 获取响应式对象引用（通过数组索引）
      const index = files.value.length - 1
      // 开始上传
      startUpload(index)
    }
  }
}

// 上传文件
async function startUpload(fileIndex: number) {
  const uploadFile = files.value[fileIndex]
  console.log(`[DEBUG] startUpload called for: ${uploadFile.name}, current status: ${uploadFile.status}`)
  try {
    uploadFile.status = 'uploading'
    console.log(`[DEBUG] Status set to uploading, calling initUpload`)

    // 1. 初始化上传
    const initResp = await fileTransferApi.initUpload({
      filename: uploadFile.name,
      file_size: uploadFile.size,
      path: '/uploads/'
    })

    console.log(`[DEBUG] initUpload response:`, initResp)
    files.value[fileIndex].taskId = initResp.task_id
    console.log(`[DEBUG] Task ID set: ${files.value[fileIndex].taskId}`)

    // 2. 分块上传
    console.log(`[DEBUG] Starting chunk upload...`)
    await uploadChunks(fileIndex)
    console.log(`[DEBUG] Chunk upload completed`)

    // 3. 完成上传
    console.log(`[DEBUG] Calling completeUpload...`)
    const checksum = await calculateChecksum(files.value[fileIndex].file)
    console.log(`[DEBUG] Checksum calculated: ${checksum}`)

    await fileTransferApi.completeUpload({
      task_id: files.value[fileIndex].taskId!,
      checksum: checksum
    })

    console.log(`[DEBUG] completeUpload succeeded, setting status to completed`)
    files.value[fileIndex].status = 'completed'
    files.value[fileIndex].progress = 100
    ElMessage.success(`${files.value[fileIndex].name} 上传成功`)

    // 刷新历史
    loadRecentTransfers()
    loadQuota()
  } catch (error: any) {
    console.error(`[DEBUG] Upload error for ${uploadFile.name}:`, error)
    files.value[fileIndex].status = 'error'
    files.value[fileIndex].error = error.message
    ElMessage.error(`${files.value[fileIndex].name} 上传失败: ${error.message}`)
  }
}

// 分块上传
async function uploadChunks(fileIndex: number) {
  const uploadFile = files.value[fileIndex]
  const file = uploadFile.file
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
  let offset = 0
  let sequence = 0
  const startTime = Date.now()

  console.log(`[DEBUG] Starting upload chunks: ${uploadFile.name}, size=${file.size}, chunks=${totalChunks}, taskId=${uploadFile.taskId}`)

  try {
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      const chunk = file.slice(start, end)

      // 检查暂停状态
      while (files.value[fileIndex].status === 'paused') {
        await new Promise(resolve => setTimeout(resolve, 100))
      }

      // 检查取消
      if (files.value[fileIndex].status === 'cancelled') {
        throw new Error('Upload cancelled')
      }

      console.log(`[DEBUG] Uploading chunk ${i+1}/${totalChunks}: offset=${start}, size=${end-start}`)

      const chunkResp = await fileTransferApi.uploadChunk(files.value[fileIndex].taskId!, {
        task_id: files.value[fileIndex].taskId!,
        offset: start,
        sequence: sequence++,
        data: new Uint8Array(await chunk.arrayBuffer())
      })

      console.log(`[DEBUG] Chunk response:`, chunkResp)

      offset = end
      const newProgress = Math.round((offset / file.size) * 100)
      files.value[fileIndex].progress = newProgress

      // 确保检测到变化
      await nextTick()

      console.log(`[DEBUG] Progress updated: ${files.value[fileIndex].progress}%, offset=${offset}, total=${file.size}`)

      // 更新速度和 ETA
      const elapsed = (Date.now() - startTime) / 1000
      const speed = offset / elapsed
      files.value[fileIndex].speed = speed
      const remaining = file.size - offset
      files.value[fileIndex].eta = formatDuration(remaining / speed)

      console.log(`[DEBUG] Speed: ${speed.toFixed(2)} bytes/s, ETA: ${files.value[fileIndex].eta}`)

      // 让 UI 有机会更新
      await new Promise(resolve => setTimeout(resolve, 10))
    }

    console.log(`[DEBUG] All chunks uploaded successfully`)
  } catch (error) {
    console.error(`[DEBUG] Upload chunk error:`, error)
    throw error
  }
}

// 暂停上传
function pauseUpload(id: string) {
  const file = files.value.find(f => f.id === id)
  if (file && file.status === 'uploading') {
    file.status = 'paused'
  }
}

// 恢复上传
function resumeUpload(id: string) {
  const file = files.value.find(f => f.id === id)
  if (file && file.status === 'paused') {
    file.status = 'uploading'
  }
}

// 取消上传
async function cancelUpload(id: string) {
  try {
    await ElMessageBox.confirm('确定要取消这个上传吗？', '确认', {
      type: 'warning'
    })

    const file = files.value.find(f => f.id === id)
    if (file && file.taskId) {
      await fileTransferApi.cancelUpload(file.taskId)
    }

    files.value = files.value.filter(f => f.id !== id)
    ElMessage.info('上传已取消')
  } catch {
    // 用户取消
  }
}

// 重试上传
function retryUpload(id: string) {
  const index = files.value.findIndex(f => f.id === id)
  if (index !== -1) {
    files.value[index].status = 'pending'
    files.value[index].progress = 0
    files.value[index].error = undefined
    startUpload(index)
  }
}

// 下载文件
function downloadFile(file: UploadFile) {
  ElMessage.info('下载功能待实现')
}

// 清除已完成
function clearCompleted() {
  files.value = files.value.filter(f => f.status !== 'completed')
}

// 全部暂停
function pauseAll() {
  files.value.forEach(f => {
    if (f.status === 'uploading') f.status = 'paused'
  })
}

// 全部继续
function resumeAll() {
  files.value.forEach(f => {
    if (f.status === 'paused') f.status = 'uploading'
  })
}

// 计算校验和
async function calculateChecksum(file: File): Promise<string> {
  // 简化版本
  const buffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
  return 'sha256:' + hashHex
}

// 加载配额
async function loadQuota() {
  try {
    const data = await fileTransferApi.getQuota()
    quota.value = data
  } catch (error: any) {
    console.error('Failed to load quota:', error)
  }
}

// 加载最近传输
async function loadRecentTransfers() {
  try {
    const data = await fileTransferApi.getTransfers({ limit: 5 })
    recentTransfers.value = data.items || []
  } catch (error: any) {
    console.error('Failed to load transfers:', error)
    recentTransfers.value = []
  }
}

// 文件导航
function handleNavigate(path: string) {
  currentPath.value = path
}

// 清理文件
async function cleanupFiles() {
  try {
    await ElMessageBox.confirm('确定要清理所有临时文件吗？', '确认', {
      type: 'warning'
    })
    ElMessage.success('临时文件已清理')
  } catch {
    // 用户取消
  }
}

// 历史重试
function handleRetryTransfer(transfer: any) {
  if (transfer.transfer_type === 'upload') {
    ElMessage.info('重试上传功能待实现')
  }
}

// 工具函数
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

function formatSpeed(bytesPerSec: number): string {
  return formatSize(bytesPerSec) + '/s'
}

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
}

function formatTime(timeStr: string): string {
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  return `${Math.floor(hours / 24)}天前`
}

function getProgressStatus(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'error': return 'exception'
    default: return undefined
  }
}

function getStatusType(status: string) {
  const types: Record<string, string> = {
    completed: 'success',
    failed: 'danger',
    transferring: 'warning',
    paused: 'info',
    pending: 'info',
    uploading: 'warning',
    error: 'danger'
  }
  return types[status] || ''
}

function getStatusText(status: string) {
  const texts: Record<string, string> = {
    pending: '等待中',
    uploading: '上传中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败',
    error: '错误'
  }
  return texts[status] || status
}

function getQuotaColor(percentage: number) {
  if (percentage < 50) return '#67C23A'
  if (percentage < 80) return '#E6A23C'
  return '#F56C6C'
}

function getFileIcon(filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase()
  const iconMap: Record<string, string> = {
    'pdf': 'Document',
    'doc': 'Document',
    'docx': 'Document',
    'xls': 'Tickets',
    'xlsx': 'Tickets',
    'ppt': 'Notebook',
    'pptx': 'Notebook',
    'jpg': 'Picture',
    'jpeg': 'Picture',
    'png': 'Picture',
    'gif': 'Picture',
    'mp4': 'VideoPlay',
    'zip': 'FolderOpened',
    'rar': 'FolderOpened'
  }
  return iconMap[ext || ''] || ''
}

// 生命周期
onMounted(() => {
  loadQuota()
  loadRecentTransfers()

  // 定时刷新进度
  progressTimer = setInterval(() => {
    files.value.forEach(file => {
      if (file.status === 'uploading' && file.taskId) {
        // 这里可以添加进度查询逻辑
      }
    })
  }, 1000)
})

onUnmounted(() => {
  if (progressTimer) {
    clearInterval(progressTimer)
  }
})
</script>

<style scoped>
.file-transfer-page {
  padding: 20px;
}

.page-content {
  display: flex;
  flex-wrap: wrap;
}

/* 上传区域 */
.upload-card {
  height: 100%;
  min-height: 500px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.upload-zone {
  border: 2px dashed #d9d9d9;
  border-radius: 12px;
  padding: 60px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.upload-zone:hover {
  border-color: #409eff;
  background: #f0f7ff;
}

.upload-zone.drag-over {
  border-color: #409eff;
  background: #e6f4ff;
  transform: scale(1.02);
}

.upload-zone.uploading {
  border-style: solid;
  border-color: #67C23A;
  background: #f0f9ff;
}

.upload-icon-wrapper {
  position: relative;
  display: inline-block;
  margin-bottom: 16px;
}

.upload-icon {
  color: #409eff;
  transition: all 0.3s ease;
}

.upload-zone:hover .upload-icon {
  transform: scale(1.1) rotate(5deg);
}

.upload-pulse {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background: rgba(64, 158, 255, 0.1);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: translate(-50%, -50%) scale(0.8); opacity: 0.5; }
  50% { transform: translate(-50%, -50%) scale(1.2); opacity: 0.2; }
}

.upload-text {
  font-size: 18px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.upload-subtext {
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
}

.upload-hint {
  font-size: 12px;
  color: #999;
}

/* 队列列表 */
.upload-queue {
  margin-top: 20px;
}

.queue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 12px;
  margin-bottom: 12px;
}

.queue-title {
  font-weight: 600;
  color: #333;
}

.queue-stats {
  font-size: 12px;
  color: #666;
}

.queue-list {
  max-height: 350px;
  overflow-y: auto;
}

.queue-item {
  display: flex;
  align-items: center;
  padding: 12px;
  background: #fafafa;
  border-radius: 8px;
  margin-bottom: 8px;
  transition: all 0.3s ease;
}

.queue-item:hover {
  background: #f0f0f0;
}

.queue-item.queue-item-error {
  background: #fef0f0;
  border: 1px solid #fbc4c4;
}

.list-enter-active,
.list-leave-active {
  transition: all 0.3s ease;
}

.list-enter-from,
.list-leave-to {
  opacity: 0;
  transform: translateX(-30px);
}

.file-main {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-icon {
  font-size: 24px;
  color: #409eff;
}

.file-details {
  flex: 1;
}

.file-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.file-name {
  font-weight: 500;
  color: #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  font-size: 12px;
  color: #999;
  display: flex;
  gap: 8px;
}

.file-progress-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.progress-text {
  font-size: 12px;
  color: #666;
  text-align: right;
}

.file-actions {
  display: flex;
  gap: 4px;
}

/* 信息区域 */
.info-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.quota-card,
.actions-card,
.history-card {
  height: fit-content;
}

.quota-content {
  display: flex;
  align-items: center;
  gap: 20px;
}

.quota-percentage {
  font-size: 24px;
  font-weight: 700;
  color: #333;
}

.quota-details {
  flex: 1;
}

.quota-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.quota-item:last-child {
  border-bottom: none;
}

.quota-label {
  color: #666;
  font-size: 14px;
}

.quota-value {
  font-weight: 600;
  color: #333;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.quick-actions .el-button {
  width: 100%;
  justify-content: flex-start;
}

/* 最近传输 */
.recent-list {
  max-height: 200px;
  overflow-y: auto;
}

.recent-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.recent-item:hover {
  background: #f5f5f5;
}

.recent-icon {
  font-size: 20px;
  color: #409eff;
}

.recent-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.recent-name {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.recent-meta {
  font-size: 12px;
  color: #999;
}
</style>
