<template>
  <div class="webhooks-page">
    <el-page-header @back="goBack" title="返回发布管理">
      <template #content>
        <span class="page-title">Webhook 自动触发</span>
      </template>
      <template #extra>
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          新建 Webhook
        </el-button>
      </template>
    </el-page-header>

    <!-- 项目选择 -->
    <el-card class="project-selector" shadow="never">
      <el-select
        v-model="selectedProjectId"
        placeholder="请选择项目"
        @change="loadWebhooks"
        style="width: 300px"
      >
        <el-option
          v-for="project in projects"
          :key="project.id"
          :label="project.name"
          :value="project.id"
        />
      </el-select>
    </el-card>

    <!-- 说明卡片 -->
    <el-alert
      v-if="selectedProjectId"
      type="info"
      :closable="false"
      class="info-alert"
    >
      <template #default>
        <div class="alert-content">
          <el-icon><InfoFilled /></el-icon>
          <span>配置 Git 仓库的 Webhook 后，代码推送将自动触发部署。支持 GitHub 和 GitLab。</span>
        </div>
      </template>
    </el-alert>

    <!-- Webhook 列表 -->
    <el-card v-loading="loading" shadow="never">
      <el-empty v-if="!selectedProjectId" description="请先选择项目" />
      <el-empty v-else-if="webhooks.length === 0" description="暂无 Webhook 配置">
        <el-button type="primary" @click="openCreateDialog">创建第一个 Webhook</el-button>
      </el-empty>
      <div v-else class="webhook-list">
        <div
          v-for="webhook in webhooks"
          :key="webhook.id"
          class="webhook-item"
          :class="{ 'webhook-item-disabled': !webhook.enabled }"
        >
          <!-- Webhook 头部 -->
          <div class="webhook-header">
            <div class="webhook-title">
              <div class="webhook-icon" :class="`webhook-icon-${webhook.source}`">
                <component :is="getSourceIcon(webhook.source)" />
              </div>
              <div class="webhook-info">
                <div class="webhook-name">{{ webhook.name }}</div>
                <div class="webhook-meta">
                  <el-tag :type="webhook.enabled ? 'success' : 'info'" size="small">
                    {{ webhook.enabled ? '已启用' : '已禁用' }}
                  </el-tag>
                  <el-tag size="small" effect="plain">
                    {{ getSourceName(webhook.source) }}
                  </el-tag>
                  <span class="meta-text">{{ webhook.target_env }} 环境</span>
                  <span class="meta-text">自动部署: {{ webhook.auto_deploy ? '是' : '否' }}</span>
                </div>
              </div>
            </div>
            <div class="webhook-actions">
              <el-switch
                v-model="webhook.enabled"
                @change="toggleEnabled(webhook)"
                :loading="webhook._toggling"
              />
              <el-button text :icon="Edit" @click="openEditDialog(webhook)">编辑</el-button>
              <el-button text :icon="Connection" @click="openTestDialog(webhook)">测试</el-button>
              <el-button text :icon="Delete" type="danger" @click="handleDelete(webhook)">删除</el-button>
            </div>
          </div>

          <!-- 分支过滤 -->
          <div class="webhook-branches">
            <span class="branches-label">触发分支:</span>
            <el-tag
              v-for="branch in webhook.branch_filter"
              :key="branch"
              size="small"
              effect="plain"
            >
              {{ branch }}
            </el-tag>
          </div>

          <!-- Webhook URL -->
          <div class="webhook-url-section">
            <div class="url-label">Webhook URL:</div>
            <div class="url-row">
              <el-input
                :model-value="webhook.url"
                readonly
                size="small"
                class="url-input"
              >
                <template #append>
                  <el-button :icon="CopyDocument" @click="copyUrl(webhook.url)">
                    复制
                  </el-button>
                </template>
              </el-input>
            </div>
            <div class="secret-row">
              <span class="secret-label">Secret:</span>
              <el-input
                :model-value="webhook.secret_masked"
                readonly
                size="small"
                class="secret-input"
                type="password"
                show-password
              />
              <el-button
                size="small"
                :icon="RefreshRight"
                @click="regenerateSecret(webhook)"
              >
                重置
              </el-button>
            </div>
          </div>

          <!-- 触发历史预览 -->
          <div class="webhook-history-preview" @click="viewHistory(webhook)">
            <span class="history-label">最近触发:</span>
            <span class="history-time">{{ formatLastTrigger(webhook.last_trigger) }}</span>
            <el-icon class="history-arrow"><ArrowRight /></el-icon>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑 Webhook' : '创建 Webhook'"
      width="600px"
      @close="resetForm"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="Webhook 名称" prop="name">
          <el-input
            v-model="form.name"
            placeholder="如: main-分支自动部署"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="来源平台" prop="source">
          <el-radio-group v-model="form.source">
            <el-radio value="github">GitHub</el-radio>
            <el-radio value="gitlab">GitLab</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="分支过滤" prop="branch_filter">
          <div class="branch-filter-input">
            <el-select
              v-model="form.branch_filter"
              multiple
              filterable
              allow-create
              placeholder="输入分支名后回车添加"
              style="width: 100%"
            >
              <el-option label="main" value="main" />
              <el-option label="master" value="master" />
              <el-option label="develop" value="develop" />
              <el-option label="release/*" value="release/*" />
              <el-option label="feature/*" value="feature/*" />
            </el-select>
            <div class="form-tip">
              支持通配符 *，如 release/* 匹配所有 release 开头的分支
            </div>
          </div>
        </el-form-item>

        <el-form-item label="目标环境" prop="target_env">
          <el-select v-model="form.target_env" placeholder="选择触发后部署的环境">
            <el-option
              v-for="env in environments"
              :key="env.name"
              :label="env.display_name || env.name"
              :value="env.name"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="自动部署">
          <el-switch v-model="form.auto_deploy" />
          <span class="switch-label">
            开启后将自动创建并执行部署任务
          </span>
        </el-form-item>

        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" />
        </el-form-item>

        <!-- 配置说明 -->
        <el-alert type="info" :closable="false">
          <template #default>
            <div class="config-help">
              <p>创建后需要在 {{ form.source === 'github' ? 'GitHub' : 'GitLab' }} 仓库中配置 Webhook：</p>
              <ol>
                <li>复制下方生成的 Webhook URL 和 Secret</li>
                <li>进入仓库 Settings → Webhooks</li>
                <li>添加新 Webhook，粘贴 URL 和 Secret</li>
                <li>选择 Push events 触发事件</li>
              </ol>
            </div>
          </template>
        </el-alert>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ isEdit ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 测试对话框 -->
    <el-dialog
      v-model="testDialogVisible"
      title="测试 Webhook"
      width="500px"
    >
      <div class="test-content">
        <p>发送测试推送事件到当前 Webhook</p>
        <el-form label-width="100px">
          <el-form-item label="测试分支">
            <el-input v-model="testBranch" placeholder="main" />
          </el-form-item>
          <el-form-item label="测试提交">
            <el-input v-model="testCommit" placeholder="test: webhook test" />
          </el-form-item>
        </el-form>
        <el-alert type="info" :closable="false">
          这将模拟一次 Git Push 事件，不会实际创建部署任务
        </el-alert>
      </div>
      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleTest" :loading="testing">
          发送测试
        </el-button>
      </template>
    </el-dialog>

    <!-- Secret 重置对话框 -->
    <el-dialog
      v-model="secretDialogVisible"
      title="重置 Secret"
      width="400px"
    >
      <div class="secret-confirm">
        <el-icon class="warning-icon"><WarningFilled /></el-icon>
        <p>确定要重置 Secret 吗？</p>
        <el-alert type="error" :closable="false">
          重置后需要更新 Git 仓库中的 Webhook 配置
        </el-alert>
      </div>
      <template #footer>
        <el-button @click="secretDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmResetSecret" :loading="resetting">
          确认重置
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Edit, Delete, Connection, InfoFilled, WarningFilled,
  ArrowRight, CopyDocument, RefreshRight
} from '@element-plus/icons-vue'
import api from '@/api'

const router = useRouter()

// 数据状态
const loading = ref(false)
const projects = ref([])
const webhooks = ref([])
const environments = ref([])
const selectedProjectId = ref('')

// 对话框状态
const dialogVisible = ref(false)
const testDialogVisible = ref(false)
const secretDialogVisible = ref(false)
const isEdit = ref(false)
const currentWebhook = ref(null)
const submitting = ref(false)
const testing = ref(false)
const resetting = ref(false)
const formRef = ref(null)

// 测试表单
const testBranch = ref('main')
const testCommit = ref('test: webhook test')

// 表单数据
const form = reactive({
  name: '',
  source: 'github',
  branch_filter: ['main'],
  target_env: '',
  auto_deploy: true,
  enabled: true
})

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入 Webhook 名称', trigger: 'blur' }
  ],
  source: [
    { required: true, message: '请选择来源平台', trigger: 'change' }
  ],
  branch_filter: [
    { type: 'array', min: 1, message: '请至少选择一个分支', trigger: 'change' }
  ],
  target_env: [
    { required: true, message: '请选择目标环境', trigger: 'change' }
  ]
}

// Git 平台配置
const gitPlatforms = {
  github: { name: 'GitHub', icon: 'GitHub' },
  gitlab: { name: 'GitLab', icon: 'GitLab' }
}

// 获取来源图标
const getSourceIcon = (source) => {
  // 这里可以返回对应的图标组件
  return Connection
}

// 获取来源名称
const getSourceName = (source) => {
  return gitPlatforms[source]?.name || source
}

// 格式化最后触发时间
const formatLastTrigger = (timeStr) => {
  if (!timeStr) return '从未触发'
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  return `${days} 天前`
}

// 复制 URL
const copyUrl = async (url) => {
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

// 查看触发历史
const viewHistory = (webhook) => {
  router.push({
    path: '/trigger-history',
    query: { webhook_id: webhook.id }
  })
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

// 加载环境列表
const loadEnvironments = async () => {
  if (!selectedProjectId.value) return
  try {
    const res = await api.getEnvironments(selectedProjectId.value)
    if (res.success) {
      environments.value = res.data || []
    }
  } catch (error) {
    console.error('Failed to load environments:', error)
  }
}

// 加载 Webhook 列表
const loadWebhooks = async () => {
  if (!selectedProjectId.value) {
    webhooks.value = []
    return
  }

  loading.value = true
  try {
    const res = await api.getWebhooks(selectedProjectId.value)
    if (res.success) {
      webhooks.value = (res.data || []).map(w => ({
        ...w,
        secret_masked: w.secret ? '••••••••••••' : '',
        _toggling: false
      }))
    } else {
      webhooks.value = []
    }
  } catch (error) {
    console.error('Failed to load webhooks:', error)
    webhooks.value = []
  } finally {
    loading.value = false
  }
}

// 打开创建对话框
const openCreateDialog = () => {
  if (!selectedProjectId.value) {
    ElMessage.warning('请先选择项目')
    return
  }
  isEdit.value = false
  resetForm()
  loadEnvironments()
  dialogVisible.value = true
}

// 打开编辑对话框
const openEditDialog = (webhook) => {
  isEdit.value = true
  currentWebhook.value = webhook
  resetForm()
  loadEnvironments()

  // 填充表单
  form.id = webhook.id
  form.name = webhook.name
  form.source = webhook.source
  form.branch_filter = [...webhook.branch_filter]
  form.target_env = webhook.target_env
  form.auto_deploy = webhook.auto_deploy
  form.enabled = webhook.enabled

  dialogVisible.value = true
}

// 重置表单
const resetForm = () => {
  form.name = ''
  form.source = 'github'
  form.branch_filter = ['main']
  form.target_env = ''
  form.auto_deploy = true
  form.enabled = true
  formRef.value?.clearValidate()
}

// 切换启用状态
const toggleEnabled = async (webhook) => {
  webhook._toggling = true
  try {
    await api.updateWebhook(webhook.id, {
      ...webhook,
      branch_filter: webhook.branch_filter,
      enabled: !webhook.enabled
    })
    webhook.enabled = !webhook.enabled
    ElMessage.success('状态已更新')
  } catch (error) {
    ElMessage.error('更新失败')
  } finally {
    webhook._toggling = false
  }
}

// 删除 Webhook
const handleDelete = async (webhook) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 Webhook "${webhook.name}" 吗？`,
      '确认删除',
      { type: 'warning' }
    )
    await api.deleteWebhook(webhook.id)
    ElMessage.success('删除成功')
    loadWebhooks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 提交表单
const handleSubmit = async () => {
  await formRef.value?.validate()

  submitting.value = true
  try {
    const data = {
      name: form.name,
      source: form.source,
      branch_filter: form.branch_filter,
      target_env: form.target_env,
      auto_deploy: form.auto_deploy,
      enabled: form.enabled
    }

    if (isEdit.value) {
      await api.updateWebhook(form.id, data)
      ElMessage.success('更新成功')
    } else {
      const res = await api.createWebhook(selectedProjectId.value, data)
      if (res.success) {
        ElMessage.success('创建成功')
        // 显示生成的 Secret
        if (res.data?.secret) {
          ElMessageBox.alert(
            `请保存此 Secret，关闭后将无法查看：\n\n${res.data.secret}`,
            'Webhook Secret',
            { type: 'success' }
          )
        }
      }
    }

    dialogVisible.value = false
    loadWebhooks()
  } catch (error) {
    ElMessage.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 打开测试对话框
const openTestDialog = (webhook) => {
  currentWebhook.value = webhook
  testBranch.value = 'main'
  testCommit.value = 'test: webhook test'
  testDialogVisible.value = true
}

// 执行测试
const handleTest = async () => {
  testing.value = true
  try {
    const res = await api.testWebhook(currentWebhook.value.id)
    if (res.success) {
      ElMessage.success('测试事件已发送')
    } else {
      ElMessage.error(res.message || '测试失败')
    }
  } catch (error) {
    ElMessage.error('测试失败: ' + (error.message || error))
  } finally {
    testing.value = false
  }
}

// 重置 Secret
const regenerateSecret = (webhook) => {
  currentWebhook.value = webhook
  secretDialogVisible.value = true
}

// 确认重置 Secret
const confirmResetSecret = async () => {
  resetting.value = true
  try {
    const res = await api.regenerateWebhookSecret(currentWebhook.value.id)
    if (res.success) {
      ElMessageBox.alert(
        `新 Secret:\n\n${res.data.secret}\n\n请立即更新 Git 仓库中的 Webhook 配置`,
        'Secret 已重置',
        { type: 'success' }
      )
      loadWebhooks()
      secretDialogVisible.value = false
    }
  } catch (error) {
    ElMessage.error('重置失败')
  } finally {
    resetting.value = false
  }
}

onMounted(() => {
  loadProjects()
  // 从路由参数获取项目 ID
  const projectId = router.currentRoute.value.query.project_id
  if (projectId) {
    selectedProjectId.value = projectId
    loadWebhooks()
  }
})
</script>

<style scoped>
.webhooks-page {
  padding: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
}

.project-selector {
  margin: 20px 0;
}

.info-alert {
  margin-bottom: 20px;
}

.alert-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.webhook-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.webhook-item {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s;
}

.webhook-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.webhook-item-disabled {
  opacity: 0.6;
}

.webhook-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.webhook-title {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.webhook-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.webhook-icon-github {
  background: linear-gradient(135deg, #24292e, #1a1e22);
  color: white;
}

.webhook-icon-gitlab {
  background: linear-gradient(135deg, #fc6d26, #e6432d);
  color: white;
}

.webhook-info {
  flex: 1;
}

.webhook-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--el-text-color-primary);
}

.webhook-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-text {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.webhook-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.webhook-branches {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  margin-bottom: 12px;
}

.branches-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.webhook-url-section {
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  margin-bottom: 12px;
}

.url-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.url-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.url-input {
  flex: 1;
}

.secret-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.secret-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.secret-input {
  flex: 1;
  max-width: 300px;
}

.webhook-history-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.webhook-history-preview:hover {
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
}

.history-arrow {
  margin-left: auto;
}

/* 表单样式 */
.radio-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.platform-icon {
  width: 16px;
  height: 16px;
}

.branch-filter-input {
  width: 100%;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  line-height: 1.5;
}

.switch-label {
  margin-left: 8px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.config-help p {
  margin: 0 0 8px 0;
}

.config-help ol {
  margin: 0;
  padding-left: 20px;
}

.config-help li {
  margin: 4px 0;
}

/* 测试对话框 */
.test-content p {
  margin: 0 0 16px 0;
}

/* 删除确认对话框 */
.secret-confirm {
  text-align: center;
  padding: 20px 0;
}

.warning-icon {
  font-size: 48px;
  color: var(--el-color-warning);
  margin-bottom: 16px;
}

.secret-confirm p {
  margin: 16px 0;
}
</style>
