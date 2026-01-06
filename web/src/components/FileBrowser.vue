<template>
  <div class="file-browser">
    <!-- 路径导航 -->
    <div class="breadcrumb-bar">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item @click="navigateTo('/')">
          <el-icon><HomeFilled /></el-icon>
          根目录
        </el-breadcrumb-item>
        <el-breadcrumb-item
          v-for="(segment, index) in pathSegments"
          :key="index"
          @click="navigateTo(segment.path)"
        >
          {{ segment.name }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 文件列表 -->
    <el-table
      :data="normalizedFiles"
      v-loading="loading"
      stripe
      style="width: 100%"
      @row-click="handleRowClick"
    >
      <el-table-column prop="name" label="文件名" min-width="200">
        <template #default="{ row }">
          <div class="file-name-cell">
            <el-icon class="file-icon">
              <component :is="getFileIcon(row.name)" />
            </el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="size" label="大小" width="120">
        <template #default="{ row }">
          {{ formatSize(row.size) }}
        </template>
      </el-table-column>
      <el-table-column prop="content_type" label="类型" width="150">
        <template #default="{ row }">
          {{ row.content_type || '未知' }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="上传时间" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column prop="file_hash" label="校验和" width="120">
        <template #default="{ row }">
          <el-tooltip v-if="row.file_hash" :content="row.file_hash" placement="top">
            <span class="checksum">{{ row.file_hash.substring(0, 8) }}...</span>
          </el-tooltip>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button
            size="small"
            type="primary"
            :icon="Download"
            @click.stop="downloadFile(row)"
          >
            下载
          </el-button>
          <el-button
            size="small"
            type="danger"
            :icon="Delete"
            @click.stop="deleteFile(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 空状态 -->
    <el-empty
      v-if="!loading && normalizedFiles.length === 0"
      description="当前目录没有文件"
      :image-size="100"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  HomeFilled, Document, Picture, VideoPlay, FolderOpened,
  Download, Delete, Files
} from '@element-plus/icons-vue'
import { fileTransferApi } from '@/api/file'

interface FileItem {
  id: string
  name: string
  Name?: string
  size: number
  Size?: number
  content_type?: string
  ContentType?: string
  file_path: string
  Path?: string
  file_hash?: string
  Checksum?: string
  created_at: string
  ModTime?: string
}

const props = defineProps<{
  currentPath?: string
}>()

const emit = defineEmits<{
  navigate: [path: string]
  select: [file: FileItem]
}>()

const loading = ref(false)
const files = ref<FileItem[]>([])

// 规范化的文件列表（处理后端返回的大写字段名）
const normalizedFiles = computed(() => {
  return files.value.map(f => ({
    ...f,
    id: f.id || f.Path || '',
    name: f.name || f.Name || '',
    size: f.size || f.Size || 0,
    content_type: f.content_type || f.ContentType || '',
    file_path: f.file_path || f.Path || '',
    file_hash: f.file_hash || f.Checksum,
    created_at: f.created_at || f.ModTime || ''
  }))
})

// 路径分段
const pathSegments = computed(() => {
  if (!props.currentPath || props.currentPath === '/') return []
  const segments = props.currentPath.split('/').filter(Boolean)
  let path = ''
  return segments.map((seg, index) => {
    path += '/' + seg
    return { name: seg, path }
  })
})

// 加载文件列表
async function loadFiles() {
  loading.value = true
  try {
    const data = await fileTransferApi.listFiles({
      path: props.currentPath,
      limit: 100
    })
    files.value = data.files || data.items || []
  } catch (error: any) {
    console.error('Failed to load files:', error)
    ElMessage.error('加载文件列表失败')
    files.value = []
  } finally {
    loading.value = false
  }
}

// 导航到路径
function navigateTo(path: string) {
  emit('navigate', path)
}

// 处理行点击
function handleRowClick(row: FileItem) {
  emit('select', row)
}

// 下载文件
async function downloadFile(file: FileItem) {
  try {
    ElMessage.info('下载功能开发中')
  } catch (error: any) {
    ElMessage.error('下载失败')
  }
}

// 删除文件
async function deleteFile(file: FileItem) {
  try {
    await ElMessageBox.confirm(
      `确定要删除文件 "${file.name}" 吗？`,
      '确认删除',
      { type: 'warning' }
    )
    await fileTransferApi.deleteFile(file.id)
    ElMessage.success('删除成功')
    loadFiles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 获取文件图标
function getFileIcon(filename: string) {
  const ext = filename.split('.').pop()?.toLowerCase()
  const iconMap: Record<string, any> = {
    'pdf': Document,
    'doc': Document,
    'docx': Document,
    'xls': Files,
    'xlsx': Files,
    'ppt': Document,
    'pptx': Document,
    'jpg': Picture,
    'jpeg': Picture,
    'png': Picture,
    'gif': Picture,
    'mp4': VideoPlay,
    'zip': FolderOpened,
    'rar': FolderOpened
  }
  return iconMap[ext || ''] || Document
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
    minute: '2-digit'
  })
}

// 监听路径变化
watch(() => props.currentPath, () => {
  loadFiles()
})

onMounted(() => {
  loadFiles()
})
</script>

<style scoped>
.file-browser {
  min-height: 400px;
}

.breadcrumb-bar {
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.breadcrumb-bar :deep(.el-breadcrumb__item) {
  cursor: pointer;
}

.breadcrumb-bar :deep(.el-breadcrumb__item:hover) {
  color: #409EFF;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-icon {
  font-size: 18px;
  color: #909399;
}

.checksum {
  font-family: monospace;
  font-size: 12px;
  color: #909399;
}

:deep(.el-table) {
  cursor: pointer;
}

:deep(.el-table__row):hover {
  background-color: #f5f7fa;
}
</style>
