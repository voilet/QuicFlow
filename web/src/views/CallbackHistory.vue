<template>
  <div class="callback-history-page">
    <el-page-header @back="goBack" title="返回发布管理">
      <template #content>
        <span class="page-title">回调历史记录</span>
      </template>
    </el-page-header>

    <!-- 筛选条件 -->
    <el-card class="filter-card" shadow="never">
      <el-form :inline="true" :model="filters" class="filter-form">
        <el-form-item label="项目">
          <el-select
            v-model="filters.project_id"
            placeholder="全部项目"
            clearable
            style="width: 200px"
            @change="handleProjectChange"
          >
            <el-option
              v-for="project in projects"
              :key="project.id"
              :label="project.name"
              :value="project.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="事件类型">
          <el-select
            v-model="filters.event_type"
            placeholder="全部事件"
            clearable
            style="width: 150px"
          >
            <el-option label="金丝雀开始" value="canary_started" />
            <el-option label="金丝雀完成" value="canary_completed" />
            <el-option label="全量完成" value="full_completed" />
          </el-select>
        </el-form-item>

        <el-form-item label="渠道">
          <el-select
            v-model="filters.channel"
            placeholder="全部渠道"
            clearable
            style="width: 120px"
          >
            <el-option label="飞书" value="feishu" />
            <el-option label="钉钉" value="dingtalk" />
            <el-option label="企业微信" value="wechat" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>

        <el-form-item label="状态">
          <el-select
            v-model="filters.status"
            placeholder="全部状态"
            clearable
            style="width: 120px"
          >
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="重试中" value="retrying" />
            <el-option label="等待" value="pending" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadHistory">
            查询
          </el-button>
          <el-button :icon="Refresh" @click="resetFilters">
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 历史列表 -->
    <el-card v-loading="loading" shadow="never">
      <el-table :data="historyList" stripe style="width: 100%">
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column prop="event_type" label="事件" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="getEventTagType(row.event_type)">
              {{ getEventTypeName(row.event_type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="channel" label="渠道" width="100">
          <template #default="{ row }">
            <div class="channel-cell">
              <el-icon :class="['channel-icon', `channel-icon-${row.channel}`]">
                <component :is="getChannelIcon(row.channel)" />
              </el-icon>
              <span>{{ getChannelName(row.channel) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)" size="small">
              {{ getStatusName(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">
            <span v-if="row.duration">{{ row.duration }}ms</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column prop="retry_count" label="重试" width="80">
          <template #default="{ row }">
            <el-badge v-if="row.retry_count > 0" :value="row.retry_count" type="warning" />
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="项目/版本" min-width="200">
          <template #default="{ row }">
            <div v-if="row.request">
              <div class="project-name">{{ row.request.project?.name || '-' }}</div>
              <div class="version-name text-muted">
                {{ row.request.version?.name || '-' }}
              </div>
            </div>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column prop="error" label="错误信息" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              v-if="row.error"
              :content="row.error"
              placement="top"
              :show-after="500"
            >
              <span class="error-text">{{ truncateText(row.error, 50) }}</span>
            </el-tooltip>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button text :icon="View" @click="viewDetail(row)">
              详情
            </el-button>
            <el-button
              v-if="row.status === 'failed'"
              text
              type="primary"
              :icon="RefreshRight"
              @click="retryCallback(row)"
              :loading="row._retrying"
            >
              重试
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadHistory"
          @current-change="loadHistory"
        />
      </div>
    </el-card>

    <!-- 详情抽屉 -->
    <el-drawer
      v-model="detailVisible"
      title="回调详情"
      size="600px"
      direction="rtl"
    >
      <template v-if="currentDetail">
        <!-- 基本信息 -->
        <el-descriptions :column="2" border>
          <el-descriptions-item label="事件类型">
            <el-tag :type="getEventTagType(currentDetail.event_type)" size="small">
              {{ getEventTypeName(currentDetail.event_type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="渠道">
            {{ getChannelName(currentDetail.channel) }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusTagType(currentDetail.status)" size="small">
              {{ getStatusName(currentDetail.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">
            {{ currentDetail.duration }}ms
          </el-descriptions-item>
          <el-descriptions-item label="重试次数">
            {{ currentDetail.retry_count }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ formatTime(currentDetail.created_at) }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 错误信息 -->
        <template v-if="currentDetail.error">
          <h4 class="detail-section-title">错误信息</h4>
          <el-alert type="error" :closable="false" show-icon>
            {{ currentDetail.error }}
          </el-alert>
        </template>

        <!-- 请求负载 -->
        <h4 class="detail-section-title">请求负载</h4>
        <div class="json-viewer">
          <pre>{{ formatJson(currentDetail.request) }}</pre>
        </div>

        <!-- 响应内容 -->
        <template v-if="currentDetail.response">
          <h4 class="detail-section-title">响应内容</h4>
          <div class="json-viewer">
            <pre>{{ currentDetail.response }}</pre>
          </div>
        </template>

        <!-- 操作按钮 -->
        <div class="detail-actions">
          <el-button
            v-if="currentDetail.status === 'failed'"
            type="primary"
            :icon="RefreshRight"
            @click="retryCallback(currentDetail)"
            :loading="currentDetail._retrying"
          >
            重新发送
          </el-button>
          <el-button @click="detailVisible = false">关闭</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search, Refresh, View, RefreshRight,
  ChatDotRound, Notification, Message, Link
} from '@element-plus/icons-vue'
import api from '@/api'

const router = useRouter()

// 数据状态
const loading = ref(false)
const projects = ref([])
const historyList = ref([])
const detailVisible = ref(false)
const currentDetail = ref(null)

// 筛选条件
const filters = reactive({
  project_id: '',
  task_id: '',
  event_type: '',
  channel: '',
  status: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 渠道名称映射
const channelNames = {
  feishu: '飞书',
  dingtalk: '钉钉',
  wechat: '企业微信',
  custom: '自定义'
}

// 事件类型名称映射
const eventTypeNames = {
  canary_started: '金丝雀开始',
  canary_completed: '金丝雀完成',
  full_completed: '全量完成'
}

// 状态名称映射
const statusNames = {
  pending: '等待',
  success: '成功',
  failed: '失败',
  retrying: '重试中'
}

// 获取渠道图标
const getChannelIcon = (channel) => {
  switch (channel) {
    case 'feishu': return ChatDotRound
    case 'dingtalk': return Notification
    case 'wechat': return Message
    case 'custom': return Link
    default: return Message
  }
}

// 获取渠道名称
const getChannelName = (channel) => channelNames[channel] || channel

// 获取事件类型名称
const getEventTypeName = (eventType) => eventTypeNames[eventType] || eventType

// 获取事件标签类型
const getEventTagType = (eventType) => {
  switch (eventType) {
    case 'canary_started': return 'warning'
    case 'canary_completed': return 'success'
    case 'full_completed': return ''
    default: return 'info'
  }
}

// 获取状态名称
const getStatusName = (status) => statusNames[status] || status

// 获取状态标签类型
const getStatusTagType = (status) => {
  switch (status) {
    case 'success': return 'success'
    case 'failed': return 'danger'
    case 'retrying': return 'warning'
    case 'pending': return 'info'
    default: return 'info'
  }
}

// 格式化时间
const formatTime = (time) => {
  if (!time) return '-'
  const date = new Date(time)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 格式化 JSON
const formatJson = (obj) => {
  if (!obj) return ''
  try {
    return JSON.stringify(obj, null, 2)
  } catch {
    return String(obj)
  }
}

// 截断文本
const truncateText = (text, maxLength) => {
  if (!text) return ''
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

// 返回上一页
const goBack = () => {
  router.push('/release')
}

// 加载项目列表
const loadProjects = async () => {
  try {
    const res = await api.getProjects()
    if (res.success) {
      projects.value = res.projects || []
    }
  } catch (error) {
    console.error('Failed to load projects:', error)
  }
}

// 处理项目切换
const handleProjectChange = () => {
  pagination.page = 1
  loadHistory()
}

// 加载回调历史
const loadHistory = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize
    }

    // 添加筛选条件
    if (filters.task_id) params.task_id = filters.task_id
    if (filters.event_type) params.event_type = filters.event_type
    if (filters.channel) params.channel = filters.channel
    if (filters.status) params.status = filters.status

    const res = await api.getCallbackHistory(params)
    if (res.success) {
      historyList.value = res.data?.items || []
      pagination.total = res.data?.total || 0
    } else {
      historyList.value = []
      pagination.total = 0
    }
  } catch (error) {
    console.error('Failed to load callback history:', error)
    historyList.value = []
  } finally {
    loading.value = false
  }
}

// 重置筛选条件
const resetFilters = () => {
  filters.project_id = ''
  filters.task_id = ''
  filters.event_type = ''
  filters.channel = ''
  filters.status = ''
  pagination.page = 1
  loadHistory()
}

// 查看详情
const viewDetail = async (row) => {
  try {
    const res = await api.getCallbackHistoryDetail(row.id)
    if (res.success) {
      currentDetail.value = res.data
      detailVisible.value = true
    }
  } catch (error) {
    ElMessage.error('获取详情失败')
  }
}

// 重试回调
const retryCallback = async (row) => {
  try {
    await ElMessageBox.confirm('确定要重新发送此回调吗？', '确认重试', {
      type: 'warning'
    })

    row._retrying = true

    const res = await api.retryCallbackHistory(row.id)
    if (res.success) {
      ElMessage.success('回调重试成功')
      // 关闭详情抽屉
      detailVisible.value = false
      // 刷新列表
      loadHistory()
    } else {
      ElMessage.error(res.error || '重试失败')
    }

  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('重试失败')
    }
  } finally {
    row._retrying = false
  }
}

onMounted(() => {
  loadProjects()
  loadHistory()

  // 从路由参数获取筛选条件
  const query = router.currentRoute.value.query
  if (query.task_id) {
    filters.task_id = query.task_id
  }
  if (query.project_id) {
    filters.project_id = query.project_id
  }
})
</script>

<style scoped>
.callback-history-page {
  padding: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
}

.filter-card {
  margin: 20px 0;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.channel-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}

.channel-icon {
  font-size: 16px;
}

.channel-icon-feishu {
  color: #00d6b9;
}

.channel-icon-dingtalk {
  color: #0089ff;
}

.channel-icon-wechat {
  color: #07c160;
}

.channel-icon-custom {
  color: #909399;
}

.project-name {
  font-weight: 500;
}

.version-name {
  font-size: 12px;
}

.text-muted {
  color: var(--el-text-color-secondary);
}

.error-text {
  color: var(--el-color-danger);
  font-size: 13px;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.detail-section-title {
  margin: 20px 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.json-viewer {
  background: var(--el-fill-color-light);
  border-radius: 4px;
  padding: 12px;
  overflow: auto;
  max-height: 300px;
}

.json-viewer pre {
  margin: 0;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}

.detail-actions {
  margin-top: 24px;
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}
</style>
