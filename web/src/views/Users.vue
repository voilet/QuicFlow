<template>
  <div class="users-container">
    <!-- 页面标题和操作 -->
    <div class="page-header">
      <div class="header-left">
        <h2>用户管理</h2>
        <p class="description">管理系统用户账号和权限</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="showCreateDialog = true">
        新建用户
      </el-button>
    </div>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <el-row :gutter="16">
        <el-col :span="8">
          <el-input
            v-model="searchUsername"
            placeholder="搜索用户名"
            :prefix-icon="Search"
            clearable
            @input="handleSearch"
          />
        </el-col>
        <el-col :span="8">
          <el-input
            v-model="searchNickname"
            placeholder="搜索昵称"
            clearable
            @input="handleSearch"
          />
        </el-col>
        <el-col :span="8">
          <el-select v-model="enableFilter" placeholder="状态筛选" clearable @change="loadUsers">
            <el-option label="全部" value="" />
            <el-option label="正常" :value="1" />
            <el-option label="冻结" :value="2" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <!-- 用户统计 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: rgba(64, 158, 255, 0.1)">
              <el-icon :size="24" color="#409EFF"><User /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total }}</div>
              <div class="stat-label">总用户数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: rgba(103, 194, 58, 0.1)">
              <el-icon :size="24" color="#67C23A"><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.enabled }}</div>
              <div class="stat-label">正常用户</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: rgba(230, 162, 60, 0.1)">
              <el-icon :size="24" color="#E6A23C"><Lock /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.disabled }}</div>
              <div class="stat-label">冻结用户</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: rgba(245, 108, 108, 0.1)">
              <el-icon :size="24" color="#F56C6C"><Key /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.admins }}</div>
              <div class="stat-label">管理员</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 用户列表 -->
    <el-card class="table-card">
      <el-table
        v-loading="loading"
        :data="users"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="username" label="用户名" width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="32" :src="row.header_img">
                {{ row.username?.charAt(0)?.toUpperCase() || '?' }}
              </el-avatar>
              <span class="username">{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="nick_name" label="昵称" width="120" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enable === 1 ? 'success' : 'danger'" size="small">
              {{ row.enable === 1 ? '正常' : '冻结' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.authority_id === 1 ? 'danger' : 'primary'" size="small">
              {{ row.authority_id === 1 ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              link
              :icon="Edit"
              @click="handleEdit(row)"
            >
              编辑
            </el-button>
            <el-button
              type="warning"
              size="small"
              link
              :icon="Key"
              @click="handleResetPassword(row)"
            >
              重置密码
            </el-button>
            <el-button
              :type="row.enable === 1 ? 'warning' : 'success'"
              size="small"
              link
              @click="handleToggleStatus(row)"
            >
              {{ row.enable === 1 ? '冻结' : '启用' }}
            </el-button>
            <el-button
              type="danger"
              size="small"
              link
              :icon="Delete"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="loadUsers"
          @size-change="loadUsers"
        />
      </div>
    </el-card>

    <!-- 创建用户对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      title="新建用户"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createRules"
        label-width="100px"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="createForm.username"
            placeholder="请输入用户名"
            autocomplete="off"
          />
        </el-form-item>
        <el-form-item label="昵称" prop="nick_name">
          <el-input
            v-model="createForm.nick_name"
            placeholder="请输入昵称"
          />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="createForm.password"
            type="password"
            placeholder="请输入密码"
            show-password
            autocomplete="new-password"
          />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="createForm.email"
            placeholder="请输入邮箱"
          />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input
            v-model="createForm.phone"
            placeholder="请输入手机号"
          />
        </el-form-item>
        <el-form-item label="角色" prop="authority_id">
          <el-select v-model="createForm.authority_id" placeholder="选择角色" style="width: 100%">
            <el-option label="普通用户" :value="888" />
            <el-option label="管理员" :value="1" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="createForm.enable">
            <el-radio :label="1">正常</el-radio>
            <el-radio :label="2">冻结</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑用户"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="100px"
      >
        <el-form-item label="用户名">
          <el-input v-model="editForm.username" disabled />
        </el-form-item>
        <el-form-item label="昵称" prop="nick_name">
          <el-input
            v-model="editForm.nick_name"
            placeholder="请输入昵称"
          />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="editForm.email"
            placeholder="请输入邮箱"
          />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input
            v-model="editForm.phone"
            placeholder="请输入手机号"
          />
        </el-form-item>
        <el-form-item label="角色" prop="authority_id">
          <el-select v-model="editForm.authority_id" placeholder="选择角色" style="width: 100%">
            <el-option label="普通用户" :value="888" />
            <el-option label="管理员" :value="1" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="editForm.enable">
            <el-radio :label="1">正常</el-radio>
            <el-radio :label="2">冻结</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" :loading="updating" @click="handleUpdate">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="showPasswordDialog"
      title="重置密码"
      width="450px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-width="100px"
      >
        <el-form-item label="用户名">
          <el-input v-model="passwordForm.username" disabled />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="passwordForm.new_password"
            type="password"
            placeholder="请输入新密码"
            show-password
            autocomplete="new-password"
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input
            v-model="passwordForm.confirm_password"
            type="password"
            placeholder="请再次输入新密码"
            show-password
            autocomplete="new-password"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPasswordDialog = false">取消</el-button>
        <el-button type="primary" :loading="resetting" @click="handleConfirmResetPassword">
          确认重置
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Search, Edit, User, CircleCheck, Key, Lock, Delete
} from '@element-plus/icons-vue'
import api from '@/api'

// 数据状态
const loading = ref(false)
const creating = ref(false)
const updating = ref(false)
const resetting = ref(false)
const users = ref([])
const searchUsername = ref('')
const searchNickname = ref('')
const enableFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 统计数据
const stats = ref({
  total: 0,
  enabled: 0,
  disabled: 0,
  admins: 0
})

// 对话框状态
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showPasswordDialog = ref(false)

// 创建表单
const createFormRef = ref()
const createForm = reactive({
  username: '',
  nick_name: '',
  password: '',
  email: '',
  phone: '',
  authority_id: 888,
  enable: 1
})

const createRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名长度在 3 到 50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少 6 个字符', trigger: 'blur' }
  ]
}

// 编辑表单
const editFormRef = ref()
const editForm = reactive({
  id: 0,
  username: '',
  nick_name: '',
  email: '',
  phone: '',
  authority_id: 888,
  enable: 1
})

const editRules = {
  nick_name: [
    { required: true, message: '请输入昵称', trigger: 'blur' }
  ]
}

// 密码表单
const passwordFormRef = ref()
const passwordForm = reactive({
  user_id: 0,
  username: '',
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.new_password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少 6 个字符', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 加载用户列表
async function loadUsers() {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (searchUsername.value) {
      params.username = searchUsername.value
    }
    if (searchNickname.value) {
      params.nickname = searchNickname.value
    }
    if (enableFilter.value !== '') {
      params.enable = enableFilter.value
    }

    const res = await api.getUserList(params)
    if (res.code === 0) {
      users.value = res.data?.list || []
      total.value = res.data?.total || 0
      updateStats()
    } else {
      ElMessage.error(res.msg || '加载用户列表失败')
    }
  } catch (error) {
    console.error('Failed to load users:', error)
    ElMessage.error('加载用户列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索用户（防抖）
let searchTimer = null
function handleSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    loadUsers()
  }, 300)
}

// 更新统计数据
function updateStats() {
  stats.value.total = users.value.length
  stats.value.enabled = users.value.filter(u => u.enable === 1).length
  stats.value.disabled = users.value.filter(u => u.enable === 2).length
  stats.value.admins = users.value.filter(u => u.authority_id === 1).length
}

// 格式化日期
function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// 创建用户
async function handleCreate() {
  await createFormRef.value.validate()
  creating.value = true
  try {
    const res = await api.createUser(createForm)
    if (res.code === 0) {
      ElMessage.success('用户创建成功')
      showCreateDialog.value = false
      resetCreateForm()
      loadUsers()
    } else {
      ElMessage.error(res.msg || '创建用户失败')
    }
  } catch (error) {
    console.error('Failed to create user:', error)
    ElMessage.error('创建用户失败')
  } finally {
    creating.value = false
  }
}

// 重置创建表单
function resetCreateForm() {
  createForm.username = ''
  createForm.nick_name = ''
  createForm.password = ''
  createForm.email = ''
  createForm.phone = ''
  createForm.authority_id = 888
  createForm.enable = 1
  createFormRef.value?.resetFields()
}

// 编辑用户
function handleEdit(row) {
  editForm.id = row.id
  editForm.username = row.username
  editForm.nick_name = row.nick_name || ''
  editForm.email = row.email || ''
  editForm.phone = row.phone || ''
  editForm.authority_id = row.authority_id
  editForm.enable = row.enable
  showEditDialog.value = true
}

// 更新用户
async function handleUpdate() {
  await editFormRef.value.validate()
  updating.value = true
  try {
    const res = await api.updateUser(editForm)
    if (res.code === 0) {
      ElMessage.success('用户更新成功')
      showEditDialog.value = false
      loadUsers()
    } else {
      ElMessage.error(res.msg || '更新用户失败')
    }
  } catch (error) {
    console.error('Failed to update user:', error)
    ElMessage.error('更新用户失败')
  } finally {
    updating.value = false
  }
}

// 重置密码
function handleResetPassword(row) {
  passwordForm.user_id = row.id
  passwordForm.username = row.username
  passwordForm.new_password = ''
  passwordForm.confirm_password = ''
  showPasswordDialog.value = true
}

// 确认重置密码
async function handleConfirmResetPassword() {
  await passwordFormRef.value.validate()
  resetting.value = true
  try {
    const res = await api.resetPassword({
      user_id: passwordForm.user_id,
      new_password: passwordForm.new_password
    })
    if (res.code === 0) {
      ElMessage.success('密码重置成功')
      showPasswordDialog.value = false
    } else {
      ElMessage.error(res.msg || '重置密码失败')
    }
  } catch (error) {
    console.error('Failed to reset password:', error)
    ElMessage.error('重置密码失败')
  } finally {
    resetting.value = false
  }
}

// 切换用户状态
async function handleToggleStatus(row) {
  const action = row.enable === 1 ? '冻结' : '启用'
  try {
    await ElMessageBox.confirm(
      `确定要${action}用户 "${row.username}" 吗？`,
      '确认操作',
      { type: 'warning' }
    )

    const newEnable = row.enable === 1 ? 2 : 1
    const res = await api.updateUser({
      ...row,
      enable: newEnable
    })

    if (res.code === 0) {
      ElMessage.success(`用户已${action}`)
      loadUsers()
    } else {
      ElMessage.error(res.msg || `${action}用户失败`)
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to toggle user status:', error)
      ElMessage.error(`${action}用户失败`)
    }
  }
}

// 删除用户
async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 "${row.username}" 吗？此操作不可恢复！`,
      '确认删除',
      {
        type: 'warning',
        confirmButtonText: '确定删除',
        cancelButtonText: '取消'
      }
    )

    const res = await api.deleteUser(row.id)
    if (res.code === 0) {
      ElMessage.success('用户删除成功')
      // 如果当前页只有一条数据，返回上一页
      if (users.value.length === 1 && currentPage.value > 1) {
        currentPage.value--
      }
      loadUsers()
    } else {
      ElMessage.error(res.msg || '删除用户失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete user:', error)
      ElMessage.error('删除用户失败')
    }
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<style scoped>
.users-container {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-left h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.header-left .description {
  margin: 4px 0 0;
  font-size: 14px;
  color: var(--tech-text-secondary);
}

.search-card {
  margin-bottom: 16px;
}

.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: 8px;
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--tech-shadow-md);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--tech-text-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: 12px;
  color: var(--tech-text-secondary);
  margin-top: 2px;
}

.table-card {
  margin-bottom: 16px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.username {
  font-weight: 500;
  color: var(--tech-primary);
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

:deep(.el-table) {
  font-size: 14px;
}

:deep(.el-table th) {
  font-weight: 600;
}

:deep(.el-avatar) {
  background: var(--tech-primary);
  color: #fff;
  font-weight: 600;
}
</style>
