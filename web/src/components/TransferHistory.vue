<template>
  <div class="transfer-history">
    <!-- 筛选工具栏 -->
    <div class="toolbar">
      <el-radio-group v-model="filterType" size="small" @change="loadTransfers">
        <el-radio-button label="all">全部</el-radio-button>
        <el-radio-button label="upload">上传</el-radio-button>
        <el-radio-button label="download">下载</el-radio-button>
      </el-radio-group>

      <el-select
        v-model="filterStatus"
        placeholder="状态筛选"
        size="small"
        style="width: 120px"
        @change="loadTransfers"
      >
        <el-option label="全部状态" value="" />
        <el-option label="已完成" value="completed" />
        <el-option label="进行中" value="transferring" />
        <el-option label="失败" value="failed" />
        <el-option label="已取消" value="cancelled" />
      </el-select>

      <el-button
        size="small"
        :icon="RefreshRight"
        @click="loadTransfers"
      >
        刷新
      </el-button>
    </div>

    <!-- 传输列表 -->
    <el-table
      :data="transfers"
      v-loading="loading"
      stripe
      style="width: 100%"
    >
      <el-table-column prop="file_name" label="文件名" min-width="200">
        <template #default="{ row }">
          <div class="file-name-cell">
            <el-icon class="file-icon">
              <component :is="getTypeIcon(row.transfer_type)" />
            </el-icon>
            <span>{{ row.file_name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="transfer_type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.transfer_type === 'upload' ? 'success' : 'primary'" size="small">
            {{ row.transfer_type === 'upload' ? '上传' : '下载' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="file_size" label="大小" width="120">
        <template #default="{ row }">
          {{ formatSize(row.file_size) }}
        </template>
      </el-table-column>
      <el-table-column prop="progress" label="进度" width="150">
        <template #default="{ row }">
          <el-progress
            :percentage="row.progress || 0"
            :status="getProgressStatus(row.status)"
            :stroke-width="8"
          />
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)" size="small">
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="开始时间" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column prop="completed_at" label="完成时间" width="180">
        <template #default="{ row }">
          {{ row.completed_at ? formatDateTime(row.completed_at) : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.status === 'failed'"
            size="small"
            type="primary"
            :icon="RefreshRight"
            @click="retryTransfer(row)"
          >
            重试
          </el-button>
          <el-button
            v-if="row.status === 'transferring'"
            size="small"
            type="danger"
            @click="cancelTransfer(row)"
          >
            取消
          </el-button>
          <el-button
            v-if="row.status === 'completed'"
            size="small"
            :icon="Download"
            @click="downloadFile(row)"
          >
            下载
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadTransfers"
        @current-change="loadTransfers"
      />
    </div>

    <!-- 空状态 -->
    <el-empty
      v-if="!loading && transfers.length === 0"
      description="暂无传输记录"
      :image-size="100"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Upload, Download, RefreshRight
} from '@element-plus/icons-vue'
import { fileTransferApi } from '@/api/file'

interface TransferItem {
  id: string
  task_id: string
  file_name: string
  file_size: number
  transfer_type: 'upload' | 'download'
  status: string
  progress: number
  bytes_transferred: number
  created_at: string
  completed_at?: string
  error_message?: string
}

const emit = defineEmits<{
  retry: [transfer: TransferItem]
}>()

const loading = ref(false)
const transfers = ref<TransferItem[]>([])
const filterType = ref<string>('all')
const filterStatus = ref<string>('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 加载传输记录
async function loadTransfers() {
  loading.value = true
  try {
    const data = await fileTransferApi.getTransfers({
      type: filterType.value === 'all' ? undefined : filterType.value,
      status: filterStatus.value || undefined,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    })
    transfers.value = data.items || []
    total.value = data.total || 0
  } catch (error: any) {
    console.error('Failed to load transfers:', error)
    ElMessage.error('加载传输记录失败')
    transfers.value = []
  } finally {
    loading.value = false
  }
}

// 重试传输
function retryTransfer(transfer: TransferItem) {
  emit('retry', transfer)
}

// 取消传输
async function cancelTransfer(transfer: TransferItem) {
  try {
    await ElMessageBox.confirm(
      `确定要取消 "${transfer.file_name}" 的传输吗？`,
      '确认取消',
      { type: 'warning' }
    )

    if (transfer.transfer_type === 'upload') {
      await fileTransferApi.cancelUpload(transfer.task_id)
    } else {
      await fileTransferApi.cancelDownload(transfer.task_id)
    }

    ElMessage.success('已取消')
    loadTransfers()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('取消失败')
    }
  }
}

// 下载文件
async function downloadFile(transfer: TransferItem) {
  try {
    // 直接通过任务ID下载文件
    const blob = await fileTransferApi.downloadFile(transfer.task_id)

    // 创建下载链接
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = transfer.file_name
    document.body.appendChild(a)
    a.click()

    // 清理
    window.URL.revokeObjectURL(url)
    document.body.removeChild(a)

    ElMessage.success(`${transfer.file_name} 下载成功`)
  } catch (error: any) {
    console.error('Download error:', error)
    ElMessage.error(`${transfer.file_name} 下载失败: ${error.message}`)
  }
}

// 获取类型图标
function getTypeIcon(type: string) {
  return type === 'upload' ? Upload : Download
}

// 获取进度状态
function getProgressStatus(status: string) {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'exception'
    default: return undefined
  }
}

// 获取状态类型
function getStatusType(status: string) {
  const types: Record<string, string> = {
    completed: 'success',
    failed: 'danger',
    transferring: 'warning',
    paused: 'info',
    pending: 'info',
    cancelled: 'info'
  }
  return types[status] || ''
}

// 获取状态文本
function getStatusText(status: string) {
  const texts: Record<string, string> = {
    pending: '等待中',
    transferring: '传输中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return texts[status] || status
}

// 格式化大小
function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

// 格式化日期时间
function formatDateTime(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

onMounted(() => {
  loadTransfers()
})
</script>

<style scoped>
.transfer-history {
  min-height: 400px;
}

.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-icon {
  font-size: 16px;
  color: #909399;
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}
</style>
