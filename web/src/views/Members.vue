<template>
  <div class="members-page">
    <el-page-header @back="goBack" title="返回发布管理">
      <template #content>
        <span class="page-title">成员管理</span>
      </template>
      <template #extra>
        <el-button type="primary" :icon="Plus" @click="openAddDialog">
          添加成员
        </el-button>
      </template>
    </el-page-header>

    <!-- 项目选择 -->
    <el-card class="project-selector" shadow="never">
      <el-select
        v-model="selectedProjectId"
        placeholder="请选择项目"
        @change="loadMembers"
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
          <span>管理项目成员及其权限。维护者可部署所有环境，访客仅可查看。</span>
        </div>
      </template>
    </el-alert>

    <!-- 成员列表 -->
    <el-card v-loading="loading" shadow="never">
      <el-empty v-if="!selectedProjectId" description="请先选择项目" />
      <el-empty v-else-if="members.length === 0" description="暂无成员">
        <el-button type="primary" @click="openAddDialog">添加第一个成员</el-button>
      </el-empty>
      <div v-else>
        <!-- 统计卡片 -->
        <div class="member-stats">
          <div class="stat-card">
            <div class="stat-value">{{ members.length }}</div>
            <div class="stat-label">总成员</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ maintainerCount }}</div>
            <div class="stat-label">维护者</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ viewerCount }}</div>
            <div class="stat-label">访客</div>
          </div>
        </div>

        <!-- 成员列表 -->
        <div class="member-list">
          <div
            v-for="member in members"
            :key="member.id"
            class="member-item"
          >
            <!-- 成员信息 -->
            <div class="member-info">
              <div class="member-avatar">
                <el-avatar :size="48" :src="member.user?.avatar">
                  {{ member.user?.display_name?.charAt(0) || member.user?.username?.charAt(0) }}
                </el-avatar>
              </div>
              <div class="member-details">
                <div class="member-name">{{ member.user?.display_name || member.user?.username }}</div>
                <div class="member-email">{{ member.user?.email }}</div>
                <div class="member-meta">
                  <span class="meta-text">
                    <el-icon><Clock /></el-icon>
                    加入于 {{ formatDate(member.added_at) }}
                  </span>
                  <span v-if="member.added_by" class="meta-text">
                    由 {{ member.added_by_name }} 邀请
                  </span>
                </div>
              </div>
            </div>

            <!-- 角色和操作 -->
            <div class="member-role-section">
              <el-select
                :model-value="member.role"
                @change="(val) => handleRoleChange(member, val)"
                :disabled="member.is_owner"
                size="small"
              >
                <el-option label="所有者" value="owner" :disabled="true" />
                <el-option label="维护者" value="maintainer" />
                <el-option label="访客" value="viewer" />
              </el-select>
              <el-tag v-if="member.is_owner" type="warning" size="small" effect="plain">
                所有者
              </el-tag>
              <el-button
                v-if="!member.is_owner"
                text
                type="danger"
                :icon="Delete"
                size="small"
                @click="handleRemove(member)"
              >
                移除
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 添加成员对话框 -->
    <el-dialog
      v-model="addDialogVisible"
      title="添加成员"
      width="500px"
      @close="resetAddForm"
    >
      <el-form
        ref="addFormRef"
        :model="addForm"
        :rules="addFormRules"
        label-width="100px"
      >
        <el-form-item label="搜索用户" prop="user_id">
          <el-select
            v-model="addForm.user_id"
            filterable
            remote
            reserve-keyword
            :remote-method="searchUsers"
            :loading="searching"
            placeholder="输入用户名或邮箱搜索"
            style="width: 100%"
          >
            <el-option
              v-for="user in searchResults"
              :key="user.id"
              :label="`${user.display_name || user.username} (${user.email})`"
              :value="user.id"
            >
              <div class="user-option">
                <el-avatar :size="24" :src="user.avatar">
                  {{ user.display_name?.charAt(0) || user.username?.charAt(0) }}
                </el-avatar>
                <div class="user-option-info">
                  <div class="user-option-name">{{ user.display_name || user.username }}</div>
                  <div class="user-option-email">{{ user.email }}</div>
                </div>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item label="角色" prop="role">
          <el-radio-group v-model="addForm.role">
            <el-radio value="maintainer">
              <div class="role-option">
                <div class="role-name">维护者</div>
                <div class="role-desc">可部署所有环境、修改配置</div>
              </div>
            </el-radio>
            <el-radio value="viewer">
              <div class="role-option">
                <div class="role-name">访客</div>
                <div class="role-desc">仅可查看</div>
              </div>
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 权限说明 -->
        <el-alert type="info" :closable="false">
          <template #default>
            <div class="permission-help">
              <div class="permission-row">
                <strong>维护者：</strong>
                <span>修改项目配置、部署所有环境、管理成员</span>
              </div>
              <div class="permission-row">
                <strong>访客：</strong>
                <span>查看项目、查看日志，无法操作部署</span>
              </div>
            </div>
          </template>
        </el-alert>
      </el-form>

      <template #footer>
        <el-button @click="addDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAdd" :loading="adding">
          添加
        </el-button>
      </template>
    </el-dialog>

    <!-- 权限说明对话框 -->
    <el-dialog
      v-model="permissionDialogVisible"
      title="权限说明"
      width="500px"
    >
      <div class="permission-matrix">
        <el-table :data="permissionData" border style="width: 100%">
          <el-table-column prop="action" label="操作" width="150" />
          <el-table-column prop="maintainer" label="维护者" />
          <el-table-column prop="viewer" label="访客" />
        </el-table>
      </div>
      <template #footer>
        <el-button type="primary" @click="permissionDialogVisible = false">知道了</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Delete, InfoFilled, Clock
} from '@element-plus/icons-vue'
import api from '@/api'

const router = useRouter()

// 数据状态
const loading = ref(false)
const projects = ref([])
const members = ref([])

// 搜索状态
const searching = ref(false)
const searchResults = ref([])

// 对话框状态
const addDialogVisible = ref(false)
const permissionDialogVisible = ref(false)
const adding = ref(false)
const addFormRef = ref(null)

// 添加成员表单
const addForm = reactive({
  user_id: '',
  role: 'viewer'
})

// 表单验证规则
const addFormRules = {
  user_id: [
    { required: true, message: '请选择用户', trigger: 'change' }
  ],
  role: [
    { required: true, message: '请选择角色', trigger: 'change' }
  ]
}

// 权限矩阵数据
const permissionData = [
  { action: '查看项目', maintainer: '✓', viewer: '✓' },
  { action: '查看日志', maintainer: '✓', viewer: '✓' },
  { action: '修改配置', maintainer: '✓', viewer: '✗' },
  { action: '部署生产', maintainer: '✓', viewer: '✗' },
  { action: '部署其他', maintainer: '✓', viewer: '✗' },
  { action: '管理成员', maintainer: '✓', viewer: '✗' }
]

// 计算属性
const maintainerCount = computed(() =>
  members.value.filter(m => m.role === 'maintainer' || m.role === 'owner').length
)

const viewerCount = computed(() =>
  members.value.filter(m => m.role === 'viewer').length
)

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
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

// 加载成员列表
const loadMembers = async () => {
  if (!selectedProjectId.value) {
    members.value = []
    return
  }

  loading.value = true
  try {
    const res = await api.getProjectMembers(selectedProjectId.value)
    if (res.success) {
      members.value = res.data || []
    } else {
      members.value = []
    }
  } catch (error) {
    console.error('Failed to load members:', error)
    members.value = []
  } finally {
    loading.value = false
  }
}

// 搜索用户
const searchUsers = async (query) => {
  if (!query) {
    searchResults.value = []
    return
  }

  searching.value = true
  try {
    const res = await api.searchUsers(query)
    if (res.success) {
      searchResults.value = res.data || []
    } else {
      searchResults.value = []
    }
  } catch (error) {
    console.error('Failed to search users:', error)
    searchResults.value = []
  } finally {
    searching.value = false
  }
}

// 打开添加对话框
const openAddDialog = () => {
  if (!selectedProjectId.value) {
    ElMessage.warning('请先选择项目')
    return
  }
  resetAddForm()
  addDialogVisible.value = true
}

// 重置添加表单
const resetAddForm = () => {
  addForm.user_id = ''
  addForm.role = 'viewer'
  searchResults.value = []
  addFormRef.value?.clearValidate()
}

// 添加成员
const handleAdd = async () => {
  await addFormRef.value?.validate()

  adding.value = true
  try {
    await api.addProjectMember(selectedProjectId.value, {
      user_id: addForm.user_id,
      role: addForm.role
    })
    ElMessage.success('添加成功')
    addDialogVisible.value = false
    loadMembers()
  } catch (error) {
    ElMessage.error(error.message || '添加失败')
  } finally {
    adding.value = false
  }
}

// 角色变更
const handleRoleChange = async (member, newRole) => {
  try {
    await api.updateProjectMember(selectedProjectId.value, member.user_id, {
      role: newRole
    })
    member.role = newRole
    ElMessage.success('角色已更新')
  } catch (error) {
    ElMessage.error('更新失败')
  }
}

// 移除成员
const handleRemove = async (member) => {
  try {
    await ElMessageBox.confirm(
      `确定要移除成员 "${member.user?.display_name || member.user?.username}" 吗？`,
      '确认移除',
      { type: 'warning' }
    )
    await api.removeProjectMember(selectedProjectId.value, member.user_id)
    ElMessage.success('移除成功')
    loadMembers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('移除失败')
    }
  }
}

// 打开权限说明
const openPermissionDialog = () => {
  permissionDialogVisible.value = true
}

onMounted(() => {
  loadProjects()
  // 从路由参数获取项目 ID
  const projectId = router.currentRoute.value.query.project_id
  if (projectId) {
    selectedProjectId.value = projectId
    loadMembers()
  }
})

// 选择的项目 ID
const selectedProjectId = ref('')
</script>

<style scoped>
.members-page {
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

/* 统计卡片 */
.member-stats {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  flex: 1;
  padding: 16px;
  background: linear-gradient(135deg, var(--el-color-primary-light-9), var(--el-fill-color-lighter));
  border-radius: 8px;
  text-align: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--el-color-primary);
  margin-bottom: 4px;
}

.stat-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

/* 成员列表 */
.member-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.member-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  transition: all 0.3s;
}

.member-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.member-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.member-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.member-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.member-email {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.member-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.meta-text {
  display: flex;
  align-items: center;
  gap: 4px;
}

.member-role-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* 添加成员对话框 */
.user-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-option-info {
  display: flex;
  flex-direction: column;
}

.user-option-name {
  font-size: 14px;
  font-weight: 500;
}

.user-option-email {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.role-name {
  font-weight: 500;
}

.role-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.permission-help {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.permission-row {
  font-size: 13px;
}

.permission-row strong {
  color: var(--el-text-color-primary);
}
</style>
