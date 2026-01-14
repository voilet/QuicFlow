<template>
  <div class="group-management">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>分组管理</span>
          <el-button type="primary" @click="handleCreate">
            <el-icon><Plus /></el-icon>
            新建分组
          </el-button>
        </div>
      </template>

      <!-- 分组列表 -->
      <el-table
        v-loading="loading"
        :data="groups"
        stripe
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="分组名称" min-width="150" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="tags" label="标签" width="150">
          <template #default="{ row }">
            <el-tag v-if="row.tags" size="small">{{ row.tags }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="设备数量" width="120">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleViewClients(row)">
              {{ getClientCount(row.id) }} 台设备
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="primary" @click="handleManageClients(row)">管理设备</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 分组表单对话框 -->
    <el-dialog
      v-model="formVisible"
      :title="currentGroupId ? '编辑分组' : '新建分组'"
      width="600px"
      @close="handleClose"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
      >
        <el-form-item label="分组名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入分组名称" />
        </el-form-item>

        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="请输入分组描述"
          />
        </el-form-item>

        <el-form-item label="标签">
          <el-input
            v-model="form.tags"
            placeholder="请输入标签，多个标签用逗号分隔"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 设备管理对话框 -->
    <el-dialog
      v-model="clientDialogVisible"
      :title="`管理设备 - ${currentGroup?.name || ''}`"
      width="800px"
    >
      <div class="client-management">
        <!-- 添加设备 -->
        <div class="add-client-section">
          <div style="margin-bottom: 10px">
            <el-input
              v-model="clientSearchKeyword"
              placeholder="搜索设备ID或主机名"
              clearable
              @input="handleClientSearch"
              style="width: 100%"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </div>
          
          <div style="border: 1px solid #dcdfe6; border-radius: 4px; max-height: 300px; overflow-y: auto; padding: 8px">
            <el-checkbox-group v-model="selectedClientIds" style="width: 100%">
              <div
                v-for="client in filteredAvailableClients"
                :key="client.client_id"
                style="padding: 8px; border-bottom: 1px solid #f0f0f0; display: flex; align-items: center; justify-content: space-between"
              >
                <el-checkbox :label="client.client_id">
                  <div style="display: flex; flex-direction: column; margin-left: 8px">
                    <span style="font-weight: 500">{{ client.client_id }}</span>
                    <span style="font-size: 12px; color: #909399">
                      {{ client.hostname || '未知主机' }}
                      <el-tag v-if="client.group_id" size="small" type="info" style="margin-left: 8px">已有分组</el-tag>
                    </span>
                  </div>
                </el-checkbox>
              </div>
              <div v-if="filteredAvailableClients.length === 0" style="padding: 20px; text-align: center; color: #909399">
                没有可用的设备
              </div>
            </el-checkbox-group>
          </div>
          
          <div style="margin-top: 10px; display: flex; justify-content: space-between; align-items: center">
            <span style="font-size: 12px; color: #909399">
              已选择 {{ selectedClientIds.length }} 个设备
            </span>
            <div>
              <el-button
                @click="handleSelectAll"
                size="small"
                :disabled="filteredAvailableClients.length === 0"
              >
                全选
              </el-button>
              <el-button
                @click="handleClearSelection"
                size="small"
                :disabled="selectedClientIds.length === 0"
              >
                清空
              </el-button>
              <el-button
                type="primary"
                @click="handleAddClients"
                :disabled="selectedClientIds.length === 0"
              >
                添加到分组
              </el-button>
            </div>
          </div>
        </div>

        <!-- 分组中的设备列表 -->
        <el-divider>分组中的设备</el-divider>
        <el-table
          v-loading="clientsLoading"
          :data="groupClients"
          stripe
        >
          <el-table-column prop="client_id" label="客户端ID" min-width="150" />
          <el-table-column prop="hostname" label="主机名" width="150" />
          <el-table-column prop="ip" label="IP地址" width="150" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="handleRemoveClient(row)">
                移除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { groupApi } from '@/api/task'
import api from '@/api'
import dayjs from 'dayjs'

const loading = ref(false)
const clientsLoading = ref(false)
const groups = ref([])
const groupClients = ref([])
const availableClients = ref([])
const filteredAvailableClients = ref([])
const clientFilterKeyword = ref('')
const clientSearchKeyword = ref('')
const formVisible = ref(false)
const clientDialogVisible = ref(false)
const currentGroupId = ref(null)
const currentGroup = ref(null)
const selectedClientIds = ref([])
const submitting = ref(false)

const formRef = ref(null)

const form = reactive({
  name: '',
  description: '',
  tags: ''
})

const rules = {
  name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }]
}

// 客户端数量缓存
const clientCountCache = ref({})

// 格式化日期
const formatDate = (date) => {
  return date ? dayjs(date).format('YYYY-MM-DD HH:mm:ss') : '-'
}

// 获取分组下的设备数量
const getClientCount = (groupId) => {
  return clientCountCache.value[groupId] || 0
}

// 加载分组列表
const loadGroups = async () => {
  loading.value = true
  try {
    const res = await groupApi.listGroups()
    if (res.success) {
      groups.value = res.data || []
      // 加载每个分组的设备数量
      for (const group of groups.value) {
        loadGroupClientCount(group.id)
      }
    }
  } catch (error) {
    ElMessage.error('加载分组列表失败')
  } finally {
    loading.value = false
  }
}

// 加载分组设备数量
const loadGroupClientCount = async (groupId) => {
  try {
    const res = await groupApi.getGroupClients(groupId)
    if (res.success) {
      clientCountCache.value[groupId] = res.data?.length || 0
    }
  } catch (error) {
    // 忽略错误
  }
}

// 加载可用客户端列表
const loadAvailableClients = async () => {
  try {
    const res = await api.getClients({ offset: 0, limit: 1000 })
    availableClients.value = res.clients || []
    // 过滤掉已经在当前分组中的设备
    const groupClientIds = new Set(groupClients.value.map(c => c.client_id))
    filteredAvailableClients.value = availableClients.value.filter(
      c => !groupClientIds.has(c.client_id)
    )
    // 如果有搜索关键词，应用搜索过滤
    if (clientSearchKeyword.value.trim()) {
      handleClientSearch()
    }
  } catch (error) {
    ElMessage.error('加载客户端列表失败')
  }
}

// 搜索客户端
const handleClientSearch = () => {
  const keyword = clientSearchKeyword.value.trim()
  const groupClientIds = new Set(groupClients.value.map(c => c.client_id))
  
  if (!keyword) {
    filteredAvailableClients.value = availableClients.value.filter(
      c => !groupClientIds.has(c.client_id)
    )
    return
  }
  
  const lowerKeyword = keyword.toLowerCase()
  filteredAvailableClients.value = availableClients.value.filter(c => {
    if (groupClientIds.has(c.client_id)) {
      return false
    }
    return c.client_id.toLowerCase().includes(lowerKeyword) ||
           (c.hostname && c.hostname.toLowerCase().includes(lowerKeyword))
  })
}

// 全选
const handleSelectAll = () => {
  selectedClientIds.value = filteredAvailableClients.value.map(c => c.client_id)
}

// 清空选择
const handleClearSelection = () => {
  selectedClientIds.value = []
}

// 加载分组下的客户端
const loadGroupClients = async (groupId) => {
  clientsLoading.value = true
  try {
    const res = await groupApi.getGroupClients(groupId)
    if (res.success) {
      groupClients.value = res.data || []
    }
  } catch (error) {
    ElMessage.error('加载分组设备失败')
  } finally {
    clientsLoading.value = false
  }
}

// 新建分组
const handleCreate = () => {
  currentGroupId.value = null
  form.name = ''
  form.description = ''
  form.tags = ''
  formVisible.value = true
}

// 编辑分组
const handleEdit = (row) => {
  currentGroupId.value = row.id
  form.name = row.name || ''
  form.description = row.description || ''
  form.tags = row.tags || ''
  formVisible.value = true
}

// 删除分组
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm('确定要删除此分组吗？删除后任务将无法关联到此分组。', '警告', {
      type: 'warning'
    })
    await groupApi.deleteGroup(row.id)
    ElMessage.success('删除成功')
    loadGroups()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 查看设备
const handleViewClients = (row) => {
  handleManageClients(row)
}

// 管理设备
const handleManageClients = async (row) => {
  currentGroup.value = row
  clientDialogVisible.value = true
  selectedClientIds.value = []
  clientFilterKeyword.value = ''
  clientSearchKeyword.value = ''
  await loadGroupClients(row.id)
  await loadAvailableClients()
}

// 添加设备到分组
const handleAddClients = async () => {
  if (selectedClientIds.value.length === 0) {
    ElMessage.warning('请选择要添加的设备')
    return
  }

  try {
    await groupApi.addGroupClients(currentGroup.value.id, selectedClientIds.value)
    ElMessage.success('设备添加成功')
    // 刷新分组设备列表和可用设备列表
    await loadGroupClients(currentGroup.value.id)
    await loadGroupClientCount(currentGroup.value.id)
    await loadAvailableClients()
    selectedClientIds.value = []
    clientSearchKeyword.value = ''
  } catch (error) {
    ElMessage.error('添加设备失败')
  }
}

// 从分组移除设备
const handleRemoveClient = async (row) => {
  try {
    await ElMessageBox.confirm('确定要从分组中移除此设备吗？', '提示', {
      type: 'warning'
    })
    await groupApi.removeGroupClient(currentGroup.value.id, row.client_id)
    ElMessage.success('设备移除成功')
    // 刷新分组设备列表和可用设备列表
    await loadGroupClients(currentGroup.value.id)
    await loadGroupClientCount(currentGroup.value.id)
    await loadAvailableClients()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('移除设备失败')
    }
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      if (currentGroupId.value) {
        await groupApi.updateGroup(currentGroupId.value, form)
        ElMessage.success('更新成功')
      } else {
        await groupApi.createGroup(form)
        ElMessage.success('创建成功')
      }
      handleClose()
      loadGroups()
    } catch (error) {
      ElMessage.error(currentGroupId.value ? '更新失败' : '创建失败')
    } finally {
      submitting.value = false
    }
  })
}

// 关闭对话框
const handleClose = () => {
  formVisible.value = false
  formRef.value?.resetFields()
}

onMounted(() => {
  loadGroups()
})
</script>

<style scoped>
.group-management {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.client-management {
  padding: 10px 0;
}

.add-client-section {
  margin-bottom: 20px;
}
</style>
