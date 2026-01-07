<template>
  <div class="credentials-page">
    <el-page-header @back="goBack" title="返回发布管理">
      <template #content>
        <span class="page-title">凭证中心</span>
      </template>
      <template #extra>
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          新建凭证
        </el-button>
      </template>
    </el-page-header>

    <!-- 说明卡片 -->
    <el-alert
      type="info"
      :closable="false"
      class="info-alert"
    >
      <template #default>
        <div class="alert-content">
          <el-icon><InfoFilled /></el-icon>
          <span>凭证用于存储敏感信息（如镜像仓库密码、Git SSH Key），采用 AES-256 加密存储。凭证可在全局或项目级别创建。</span>
        </div>
      </template>
    </el-alert>

    <!-- 筛选器 -->
    <el-card class="filter-card" shadow="never">
      <el-row :gutter="16" align="middle">
        <el-col :span="6">
          <el-select
            v-model="filterScope"
            placeholder="全部范围"
            @change="loadCredentials"
            clearable
          >
            <el-option label="全局凭证" value="global" />
            <el-option label="项目凭证" value="project" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-select
            v-model="filterType"
            placeholder="全部类型"
            @change="loadCredentials"
            clearable
          >
            <el-option label="Docker Registry" value="docker_registry" />
            <el-option label="Git SSH Key" value="git_ssh" />
            <el-option label="Git Token" value="git_token" />
            <el-option label="用户名密码" value="username_password" />
          </el-select>
        </el-col>
        <el-col :span="8">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索凭证名称"
            :prefix-icon="Search"
            @input="handleSearch"
            clearable
          />
        </el-col>
        <el-col :span="4" class="text-right">
          <el-text type="info">共 {{ filteredCredentials.length }} 个凭证</el-text>
        </el-col>
      </el-row>
    </el-card>

    <!-- 凭证列表 -->
    <el-card v-loading="loading" shadow="never">
      <el-empty v-if="credentials.length === 0" description="暂无凭证">
        <el-button type="primary" @click="openCreateDialog">创建第一个凭证</el-button>
      </el-empty>
      <div v-else class="credential-list">
        <div
          v-for="cred in filteredCredentials"
          :key="cred.id"
          class="credential-item"
        >
          <!-- 凭证图标和名称 -->
          <div class="credential-header">
            <div class="credential-title">
              <div class="credential-icon" :class="`credential-icon-${cred.type}`">
                <component :is="getTypeIcon(cred.type)" />
              </div>
              <div class="credential-info">
                <div class="credential-name">{{ cred.name }}</div>
                <div class="credential-meta">
                  <el-tag size="small" :type="getScopeTagType(cred.scope)">
                    {{ cred.scope === 'global' ? '全局' : '项目' }}
                  </el-tag>
                  <el-tag size="small" effect="plain">{{ getTypeName(cred.type) }}</el-tag>
                  <span class="meta-text">{{ cred.description || '无描述' }}</span>
                </div>
              </div>
            </div>
            <div class="credential-actions">
              <el-button text :icon="Edit" @click="openEditDialog(cred)">编辑</el-button>
              <el-button text :icon="Delete" type="danger" @click="handleDelete(cred)">删除</el-button>
            </div>
          </div>

          <!-- 凭证详情 -->
          <div class="credential-details">
            <div class="detail-row">
              <span class="detail-label">服务器地址:</span>
              <span class="detail-value">{{ cred.server_url || '-' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">用户名:</span>
              <span class="detail-value">{{ cred.username || '-' }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">使用次数:</span>
              <span class="detail-value">{{ cred.use_count || 0 }} 次</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">最后使用:</span>
              <span class="detail-value">{{ formatDate(cred.last_used_at) }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">创建时间:</span>
              <span class="detail-value">{{ formatDate(cred.created_at) }}</span>
            </div>
          </div>

          <!-- 关联项目 (项目凭证显示) -->
          <div v-if="cred.scope === 'project' && cred.project_name" class="credential-projects">
            <el-tag size="small" effect="plain">
              <el-icon><FolderOpened /></el-icon>
              {{ cred.project_name }}
            </el-tag>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑凭证' : '创建凭证'"
      width="600px"
      @close="resetForm"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="凭证类型" prop="type">
          <el-select
            v-model="form.type"
            placeholder="选择凭证类型"
            :disabled="isEdit"
            @change="handleTypeChange"
          >
            <el-option label="Docker Registry" value="docker_registry">
              <div class="option-with-icon">
                <el-icon><Box /></el-icon>
                <span>Docker Registry - 镜像仓库认证</span>
              </div>
            </el-option>
            <el-option label="Git SSH Key" value="git_ssh">
              <div class="option-with-icon">
                <el-icon><Key /></el-icon>
                <span>Git SSH Key - SSH 私钥克隆</span>
              </div>
            </el-option>
            <el-option label="Git Token" value="git_token">
              <div class="option-with-icon">
                <el-icon><Key /></el-icon>
                <span>Git Token - API 访问令牌</span>
              </div>
            </el-option>
            <el-option label="用户名密码" value="username_password">
              <div class="option-with-icon">
                <el-icon><Lock /></el-icon>
                <span>用户名密码 - 通用认证</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item label="凭证名称" prop="name">
          <el-input
            v-model="form.name"
            placeholder="如: Harbor-生产环境、GitHub-SSH"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="2"
            placeholder="选填，描述此凭证的用途"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="凭证范围">
          <el-radio-group v-model="form.scope">
            <el-radio label="global">全局凭证（所有项目可用）</el-radio>
            <el-radio label="project">项目凭证（仅指定项目可用）</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="form.scope === 'project'" label="选择项目" prop="project_id">
          <el-select
            v-model="form.project_id"
            placeholder="选择关联的项目"
            filterable
          >
            <el-option
              v-for="project in projects"
              :key="project.id"
              :label="project.name"
              :value="project.id"
            />
          </el-select>
        </el-form-item>

        <el-divider content-position="left">认证信息</el-divider>

        <!-- Docker Registry / 用户名密码 -->
        <template v-if="form.type === 'docker_registry' || form.type === 'username_password'">
          <el-form-item label="服务器地址" prop="server_url">
            <el-input
              v-model="form.server_url"
              :placeholder="getServerUrlPlaceholder(form.type)"
            />
            <div v-if="form.type === 'docker_registry'" class="form-tip">
              常见地址: docker.io, harbor.company.com, registry.cn-hangzhou.aliyuncs.com
            </div>
          </el-form-item>

          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" placeholder="输入用户名" />
          </el-form-item>

          <el-form-item label="密码" :prop="isEdit ? '' : 'password'">
            <el-input
              v-model="form.password"
              type="password"
              :placeholder="isEdit ? '留空则不修改密码' : '输入密码'"
              show-password
            />
          </el-form-item>
        </template>

        <!-- Git SSH Key -->
        <template v-if="form.type === 'git_ssh'">
          <el-form-item label="私钥内容" prop="ssh_key">
            <el-input
              v-model="form.ssh_key"
              type="textarea"
              :rows="6"
              placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
              class="ssh-key-input"
            />
            <div class="form-tip">
              支持 PEM 格式的私钥文件内容
            </div>
          </el-form-item>

          <el-form-item label="私钥密码">
            <el-input
              v-model="form.ssh_passphrase"
              type="password"
              placeholder="如果私钥有密码保护，请输入"
              show-password
            />
          </el-form-item>
        </template>

        <!-- Git Token -->
        <template v-if="form.type === 'git_token'">
          <el-form-item label="服务器地址" prop="server_url">
            <el-input
              v-model="form.server_url"
              placeholder="github.com, gitlab.com"
            />
          </el-form-item>

          <el-form-item label="访问令牌" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              :placeholder="getTokenPlaceholder(form.server_url)"
              show-password
            />
            <div class="form-tip">
              {{ getTokenHelp(form.server_url) }}
            </div>
          </el-form-item>
        </template>

        <!-- 安全提示 -->
        <el-alert
          type="warning"
          :closable="false"
          show-icon
        >
          <template #default>
            <span>凭证信息将被 AES-256 加密存储，请妥善保管。创建后除密码外，其他信息可修改。</span>
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

    <!-- 删除确认对话框 -->
    <el-dialog
      v-model="deleteDialogVisible"
      title="确认删除"
      width="400px"
    >
      <div class="delete-confirm">
        <el-icon class="warning-icon"><WarningFilled /></el-icon>
        <p>确定要删除凭证 <strong>{{ deleteTarget?.name }}</strong> 吗？</p>
        <el-alert type="error" :closable="false">
          <template #default>
            <span>删除后不可恢复，使用此凭证的部署任务可能失败！</span>
          </template>
        </el-alert>
        <p v-if="deleteTarget?.use_count > 0" class="usage-hint">
          此凭证已被使用 {{ deleteTarget.use_count }} 次
        </p>
      </div>
      <template #footer>
        <el-button @click="deleteDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmDelete" :loading="deleting">
          确认删除
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Edit, Delete, Search, InfoFilled, WarningFilled,
  Box, Key, Lock, FolderOpened
} from '@element-plus/icons-vue'
import api from '@/api'

const router = useRouter()

// 数据状态
const loading = ref(false)
const credentials = ref([])
const projects = ref([])

// 筛选状态
const filterScope = ref('')
const filterType = ref('')
const searchKeyword = ref('')

// 对话框状态
const dialogVisible = ref(false)
const deleteDialogVisible = ref(false)
const isEdit = ref(false)
const deleteTarget = ref(null)
const submitting = ref(false)
const deleting = ref(false)
const formRef = ref(null)

// 表单数据
const form = reactive({
  type: 'docker_registry',
  name: '',
  description: '',
  scope: 'global',
  project_id: '',
  server_url: '',
  username: '',
  password: '',
  ssh_key: '',
  ssh_passphrase: ''
})

// 表单验证规则
const formRules = {
  type: [
    { required: true, message: '请选择凭证类型', trigger: 'change' }
  ],
  name: [
    { required: true, message: '请输入凭证名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  project_id: [
    { required: true, message: '请选择项目', trigger: 'change' }
  ],
  server_url: [
    { required: true, message: '请输入服务器地址', trigger: 'blur' }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ],
  ssh_key: [
    { required: true, message: '请输入私钥内容', trigger: 'blur' }
  ]
}

// 计算属性 - 过滤后的凭证列表
const filteredCredentials = computed(() => {
  let result = credentials.value

  // 范围筛选
  if (filterScope.value) {
    result = result.filter(c => c.scope === filterScope.value)
  }

  // 类型筛选
  if (filterType.value) {
    result = result.filter(c => c.type === filterType.value)
  }

  // 关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(c =>
      c.name.toLowerCase().includes(keyword) ||
      (c.description && c.description.toLowerCase().includes(keyword)) ||
      (c.server_url && c.server_url.toLowerCase().includes(keyword))
    )
  }

  return result
})

// 凭证类型映射
const credentialTypes = {
  docker_registry: { name: 'Docker Registry', icon: Box },
  git_ssh: { name: 'Git SSH Key', icon: Key },
  git_token: { name: 'Git Token', icon: Key },
  username_password: { name: '用户名密码', icon: Lock }
}

// 获取类型图标
const getTypeIcon = (type) => {
  return credentialTypes[type]?.icon || Key
}

// 获取类型名称
const getTypeName = (type) => {
  return credentialTypes[type]?.name || type
}

// 获取范围标签类型
const getScopeTagType = (scope) => {
  return scope === 'global' ? 'success' : 'primary'
}

// 获取服务器地址占位符
const getServerUrlPlaceholder = (type) => {
  switch (type) {
    case 'docker_registry':
      return 'docker.io 或 harbor.company.com'
    case 'username_password':
      return 'https://example.com'
    default:
      return '服务器地址'
  }
}

// 获取 Token 占位符
const getTokenPlaceholder = (serverUrl) => {
  if (serverUrl?.includes('github')) {
    return 'ghp_xxxxxxxxxxxxxxxxxxxx'
  } else if (serverUrl?.includes('gitlab')) {
    return 'glpat-xxxxxxxxxxxxxxxxxxxx'
  }
  return '输入访问令牌'
}

// 获取 Token 帮助信息
const getTokenHelp = (serverUrl) => {
  if (serverUrl?.includes('github')) {
    return '在 GitHub Settings → Developer settings → Personal access tokens 中创建'
  } else if (serverUrl?.includes('gitlab')) {
    return '在 GitLab Settings → Access Tokens 中创建'
  }
  return '输入对应平台的访问令牌'
}

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return '从未'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  if (days < 7) return `${days} 天前`
  return date.toLocaleDateString('zh-CN')
}

// 返回上一页
const goBack = () => {
  router.push('/release')
}

// 加载凭证列表
const loadCredentials = async () => {
  loading.value = true
  try {
    const res = await api.getCredentials()
    if (res.success) {
      credentials.value = res.data || []
    } else {
      credentials.value = []
    }
  } catch (error) {
    console.error('Failed to load credentials:', error)
    ElMessage.error('加载凭证列表失败')
  } finally {
    loading.value = false
  }
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

// 搜索处理（防抖）
let searchTimeout = null
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    // 搜索是纯前端实现，不需要调用 API
  }, 300)
}

// 凭证类型改变
const handleTypeChange = () => {
  // 清空认证信息
  form.server_url = ''
  form.username = ''
  form.password = ''
  form.ssh_key = ''
  form.ssh_passphrase = ''
}

// 打开创建对话框
const openCreateDialog = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

// 打开编辑对话框
const openEditDialog = (cred) => {
  isEdit.value = true
  resetForm()

  // 填充表单（不包含敏感信息）
  form.id = cred.id
  form.type = cred.type
  form.name = cred.name
  form.description = cred.description
  form.scope = cred.scope
  form.project_id = cred.project_id || ''
  form.server_url = cred.server_url || ''
  form.username = cred.username || ''

  dialogVisible.value = true
}

// 重置表单
const resetForm = () => {
  form.type = 'docker_registry'
  form.name = ''
  form.description = ''
  form.scope = 'global'
  form.project_id = ''
  form.server_url = ''
  form.username = ''
  form.password = ''
  form.ssh_key = ''
  form.ssh_passphrase = ''
  formRef.value?.clearValidate()
}

// 构建提交数据
const buildSubmitData = () => {
  const data = {
    type: form.type,
    name: form.name,
    description: form.description,
    scope: form.scope
  }

  if (form.scope === 'project') {
    data.project_id = form.project_id
  }

  // 根据类型添加认证信息
  switch (form.type) {
    case 'docker_registry':
    case 'username_password':
      data.server_url = form.server_url
      data.username = form.username
      if (form.password) {
        data.password = form.password
      }
      break
    case 'git_ssh':
      data.ssh_key = form.ssh_key
      data.ssh_passphrase = form.ssh_passphrase || ''
      break
    case 'git_token':
      data.server_url = form.server_url
      data.password = form.password
      break
  }

  return data
}

// 提交表单
const handleSubmit = async () => {
  await formRef.value?.validate()

  submitting.value = true
  try {
    const data = buildSubmitData()

    if (isEdit.value) {
      await api.updateCredential(form.id, data)
      ElMessage.success('更新成功')
    } else {
      await api.createCredential(data)
      ElMessage.success('创建成功')
    }

    dialogVisible.value = false
    loadCredentials()
  } catch (error) {
    ElMessage.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 删除凭证
const handleDelete = (cred) => {
  deleteTarget.value = cred
  deleteDialogVisible.value = true
}

// 确认删除
const confirmDelete = async () => {
  deleting.value = true
  try {
    await api.deleteCredential(deleteTarget.value.id)
    ElMessage.success('删除成功')
    deleteDialogVisible.value = false
    loadCredentials()
  } catch (error) {
    ElMessage.error('删除失败')
  } finally {
    deleting.value = false
  }
}

onMounted(() => {
  loadCredentials()
  loadProjects()
})
</script>

<style scoped>
.credentials-page {
  padding: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
}

.info-alert {
  margin: 20px 0;
}

.alert-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-card {
  margin-bottom: 20px;
}

.text-right {
  text-align: right;
}

.credential-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.credential-item {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s;
  background: var(--el-bg-color);
}

.credential-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.credential-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.credential-title {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.credential-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.credential-icon-docker_registry {
  background: linear-gradient(135deg, #2496ed, #1c84c6);
  color: white;
}

.credential-icon-git_ssh,
.credential-icon-git_token {
  background: linear-gradient(135deg, #f03c3c, #c03232);
  color: white;
}

.credential-icon-username_password {
  background: linear-gradient(135deg, #6366f1, #4f46e5);
  color: white;
}

.credential-info {
  flex: 1;
}

.credential-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}

.credential-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.meta-text {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.credential-actions {
  display: flex;
  gap: 8px;
}

.credential-details {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}

.detail-row {
  display: flex;
  gap: 8px;
}

.detail-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  min-width: 80px;
}

.detail-value {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 500;
}

.credential-projects {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* 表单样式 */
.option-with-icon {
  display: flex;
  align-items: center;
  gap: 8px;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  line-height: 1.5;
}

.ssh-key-input :deep(textarea) {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
}

:deep(.el-divider__text) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

/* 删除确认对话框 */
.delete-confirm {
  text-align: center;
  padding: 20px 0;
}

.warning-icon {
  font-size: 48px;
  color: var(--el-color-warning);
  margin-bottom: 16px;
}

.delete-confirm p {
  margin: 16px 0;
}

.usage-hint {
  color: var(--el-color-warning);
  font-size: 13px;
}
</style>
