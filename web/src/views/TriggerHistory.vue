<template>
  <div class="trigger-history-page">
    <el-page-header @back="goBack" title="返回">
      <template #content>
        <span class="page-title">Webhook 触发历史</span>
      </template>
      <template #extra>
        <el-button :icon="Refresh" @click="loadTriggers">刷新</el-button>
      </template>
    </el-page-header>

    <!-- 筛选器 -->
    <el-card class="filter-card" shadow="never">
      <el-row :gutter="16" align="middle">
        <el-col :span="5">
          <el-select
            v-model="filterSource"
            placeholder="全部来源"
            @change="loadTriggers"
            clearable
          >
            <el-option label="GitHub" value="github" />
            <el-option label="GitLab" value="gitlab" />
          </el-select>
        </el-col>
        <el-col :span="5">
          <el-select
            v-model="filterStatus"
            placeholder="全部状态"
            @change="loadTriggers"
            clearable
          >
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="跳过" value="skipped" />
          </el-select>
        </el-col>
        <el-col :span="10">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索分支、提交信息"
            :prefix-icon="Search"
            @input="handleSearch"
            clearable
          />
        </el-col>
        <el-col :span="4" class="text-right">
          <el-text type="info">共 {{ filteredTriggers.length }} 条记录</el-text>
        </el-col>
      </el-row>
    </el-card>

    <!-- 触发历史列表 -->
    <el-card v-loading="loading" shadow="never">
      <el-empty v-if="triggers.length === 0" description="暂无触发记录" />
      <el-timeline v-else class="trigger-timeline">
        <el-timeline-item
          v-for="trigger in filteredTriggers"
          :key="trigger.id"
          :timestamp="formatDateTime(trigger.triggered_at)"
          placement="top"
          :type="getStatusType(trigger.status)"
          :icon="getStatusIcon(trigger.status)"
        >
          <div class="trigger-item">
            <!-- 头部信息 -->
            <div class="trigger-header">
              <div class="trigger-title">
                <el-tag :type="getSourceTagType(trigger.source)" size="small">
                  {{ getSourceName(trigger.source) }}
                </el-tag>
                <span class="branch-name">{{ trigger.branch }}</span>
              </div>
              <div class="trigger-status">
                <el-tag :type="getStatusTagType(trigger.status)" size="small">
                  {{ getStatusName(trigger.status) }}
                </el-tag>
              </div>
            </div>

            <!-- 提交信息 -->
            <div class="trigger-commit">
              <el-icon><Document /></el-icon>
              <span class="commit-message">{{ trigger.message }}</span>
              <code class="commit-sha">{{ shortSha(trigger.commit) }}</code>
            </div>

            <!-- 提交者 -->
            <div class="trigger-committer">
              <el-icon><User /></el-icon>
              <span>{{ trigger.committer }}</span>
            </div>

            <!-- 关联任务 -->
            <div v-if="trigger.task_id" class="trigger-task">
              <el-icon><Operation /></el-icon>
              <span>已创建部署任务：</span>
              <el-link type="primary" @click="goToTask(trigger.task_id)">
                {{ trigger.task_id }}
              </el-link>
            </div>

            <!-- 失败原因 -->
            <div v-if="trigger.status === 'failed' && trigger.error" class="trigger-error">
              <el-icon><Warning /></el-icon>
              <span>{{ trigger.error }}</span>
            </div>

            <!-- 操作按钮 -->
            <div class="trigger-actions">
              <el-button
                v-if="trigger.status === 'failed'"
                size="small"
                :icon="RefreshRight"
                @click="handleRetry(trigger)"
              >
                重试
              </el-button>
              <el-button
                size="small"
                text
                @click="viewDetail(trigger)"
              >
                查看详情
              </el-button>
            </div>
          </div>
        </el-timeline-item>
      </el-timeline>
    </el-card>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="触发详情"
      width="600px"
    >
      <div v-if="currentTrigger" class="trigger-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="来源平台">
            {{ getSourceName(currentTrigger.source) }}
          </el-descriptions-item>
          <el-descriptions-item label="分支">
            {{ currentTrigger.branch }}
          </el-descriptions-item>
          <el-descriptions-item label="提交">
            <div class="commit-detail">
              <div>{{ currentTrigger.message }}</div>
              <code class="commit-sha-full">{{ currentTrigger.commit }}</code>
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="提交者">
            {{ currentTrigger.committer }}
          </el-descriptions-item>
          <el-descriptions-item label="触发时间">
            {{ formatDateTime(currentTrigger.triggered_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusTagType(currentTrigger.status)">
              {{ getStatusName(currentTrigger.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="currentTrigger.task_id" label="关联任务">
            <el-link type="primary" @click="goToTask(currentTrigger.task_id)">
              {{ currentTrigger.task_id }}
            </el-link>
          </el-descriptions-item>
          <el-descriptions-item v-if="currentTrigger.error" label="错误信息">
            <el-text type="danger">{{ currentTrigger.error }}</el-text>
          </el-descriptions-item>
        </el-descriptions>

        <!-- Webhook 负载（调试用） -->
        <el-collapse v-if="currentTrigger.payload" class="payload-collapse">
          <el-collapse-item title="Webhook 负载" name="payload">
            <pre class="payload-preview">{{ JSON.stringify(currentTrigger.payload, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  Refresh, Search, Document, User, Operation, Warning, RefreshRight,
  CircleCheck, CircleClose, InfoFilled
} from '@element-plus/icons-vue'
import api from '@/api'

const router = useRouter()
const route = useRoute()

// 数据状态
const loading = ref(false)
const triggers = ref([])
const webhookId = ref('')

// 筛选状态
const filterSource = ref('')
const filterStatus = ref('')
const searchKeyword = ref('')

// 对话框状态
const detailDialogVisible = ref(false)
const currentTrigger = ref(null)

// 计算属性 - 过滤后的触发记录
const filteredTriggers = computed(() => {
  let result = triggers.value

  // 来源筛选
  if (filterSource.value) {
    result = result.filter(t => t.source === filterSource.value)
  }

  // 状态筛选
  if (filterStatus.value) {
    result = result.filter(t => t.status === filterStatus.value)
  }

  // 关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(t =>
      t.branch.toLowerCase().includes(keyword) ||
      t.message.toLowerCase().includes(keyword) ||
      t.commit.toLowerCase().includes(keyword)
    )
  }

  return result
})

// Git 平台名称
const getSourceName = (source) => {
  const names = { github: 'GitHub', gitlab: 'GitLab' }
  return names[source] || source
}

// 状态名称
const getStatusName = (status) => {
  const names = {
    success: '成功',
    failed: '失败',
    skipped: '跳过'
  }
  return names[status] || status
}

// 获取来源标签类型
const getSourceTagType = (source) => {
  return source === 'github' ? '' : 'warning'
}

// 获取状态标签类型
const getStatusTagType = (status) => {
  const types = {
    success: 'success',
    failed: 'danger',
    skipped: 'info'
  }
  return types[status] || 'info'
}

// 获取时间线类型
const getStatusType = (status) => {
  const types = {
    success: 'success',
    failed: 'danger',
    skipped: 'info'
  }
  return types[status] || 'primary'
}

// 获取状态图标
const getStatusIcon = (status) => {
  const icons = {
    success: CircleCheck,
    failed: CircleClose,
    skipped: InfoFilled
  }
  return icons[status] || InfoFilled
}

// 格式化日期时间
const formatDateTime = (dateStr) => {
  if (!dateStr) return '-'
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

// 短 SHA
const shortSha = (sha) => {
  return sha ? sha.substring(0, 8) : '-'
}

// 搜索处理（防抖）
let searchTimeout = null
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    // 搜索是纯前端实现
  }, 300)
}

// 返回
const goBack = () => {
  if (webhookId.value) {
    router.push({ path: '/webhooks', query: { webhook_id: webhookId.value } })
  } else {
    router.push('/webhooks')
  }
}

// 跳转到任务
const goToTask = (taskId) => {
  router.push({ path: '/release', query: { task_id: taskId } })
}

// 加载触发历史
const loadTriggers = async () => {
  loading.value = true
  try {
    let res
    if (webhookId.value) {
      res = await api.getWebhookTriggers(webhookId.value)
    } else {
      // 获取所有触发记录（需要后端支持）
      res = await api.getAllTriggers()
    }

    if (res.success) {
      triggers.value = res.data || []
    } else {
      triggers.value = []
    }
  } catch (error) {
    console.error('Failed to load triggers:', error)
    ElMessage.error('加载触发历史失败')
  } finally {
    loading.value = false
  }
}

// 查看详情
const viewDetail = (trigger) => {
  currentTrigger.value = trigger
  detailDialogVisible.value = true
}

// 重试触发
const handleRetry = async (trigger) => {
  try {
    await api.retryTrigger(trigger.id)
    ElMessage.success('重试成功')
    loadTriggers()
  } catch (error) {
    ElMessage.error('重试失败')
  }
}

onMounted(() => {
  // 从路由参数获取 webhook_id
  webhookId.value = route.query.webhook_id || ''
  loadTriggers()
})
</script>

<style scoped>
.trigger-history-page {
  padding: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
}

.filter-card {
  margin-bottom: 20px;
}

.text-right {
  text-align: right;
}

.trigger-timeline {
  padding-left: 20px;
}

.trigger-item {
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
}

.trigger-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.trigger-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.branch-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--el-text-color-primary);
}

.trigger-commit {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.commit-message {
  flex: 1;
}

.commit-sha {
  font-family: monospace;
  font-size: 12px;
  color: var(--el-color-primary);
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
}

.trigger-committer {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.trigger-task {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  color: var(--el-text-color-regular);
  font-size: 13px;
}

.trigger-error {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 12px;
  background: var(--el-color-danger-light-9);
  border-radius: 4px;
  color: var(--el-color-danger);
  font-size: 13px;
  margin-bottom: 8px;
}

.trigger-actions {
  display: flex;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* 详情对话框 */
.commit-detail {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.commit-sha-full {
  font-family: monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  padding: 4px 8px;
  border-radius: 4px;
  align-self: flex-start;
}

.payload-collapse {
  margin-top: 16px;
}

.payload-preview {
  background: var(--el-fill-color-lighter);
  padding: 12px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 300px;
  overflow-y: auto;
  margin: 0;
}
</style>
