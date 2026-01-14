<template>
  <div class="execution-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>执行记录</span>
        </div>
      </template>

      <!-- 搜索栏 -->
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="任务ID">
          <el-input
            v-model="searchForm.task_id"
            placeholder="请输入任务ID"
            clearable
            style="width: 150px"
          />
        </el-form-item>
        <el-form-item label="客户端ID">
          <el-input
            v-model="searchForm.client_id"
            placeholder="请输入客户端ID"
            clearable
            style="width: 150px"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchForm.status" placeholder="请选择状态" clearable style="width: 150px">
            <el-option label="全部" value="" />
            <el-option label="待执行" :value="1" />
            <el-option label="执行中" :value="2" />
            <el-option label="成功" :value="3" />
            <el-option label="失败" :value="4" />
            <el-option label="超时" :value="5" />
            <el-option label="取消" :value="6" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 统计卡片 -->
      <el-row :gutter="16" class="stats-row">
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="stat-item">
              <div class="stat-value">{{ stats.total || 0 }}</div>
              <div class="stat-label">总执行次数</div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="stat-item">
              <div class="stat-value success">{{ stats.success || 0 }}</div>
              <div class="stat-label">成功</div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="stat-item">
              <div class="stat-value failed">{{ stats.failed || 0 }}</div>
              <div class="stat-label">失败</div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="stat-item">
              <div class="stat-value">{{ stats.running || 0 }}</div>
              <div class="stat-label">执行中</div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 表格 -->
      <el-table
        v-loading="loading"
        :data="tableData"
        stripe
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="task_name" label="任务名称" min-width="150" />
        <el-table-column prop="client_id" label="客户端ID" width="150" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="开始时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.start_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="end_time" label="结束时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.end_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时(秒)" width="100">
          <template #default="{ row }">
            {{ row.duration || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="exit_code" label="退出码" width="100">
          <template #default="{ row }">
            {{ row.exit_code !== null ? row.exit_code : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleViewLogs(row)">查看日志</el-button>
            <el-button link type="primary" @click="handleViewDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>

    <!-- 日志对话框 -->
    <el-dialog
      v-model="logDialogVisible"
      title="执行日志"
      width="800px"
    >
      <el-tabs>
        <el-tab-pane label="输出">
          <pre class="log-content">{{ currentLogs.output || '无输出' }}</pre>
        </el-tab-pane>
        <el-tab-pane label="错误">
          <pre class="log-content error">{{ currentLogs.error_msg || '无错误' }}</pre>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { executionApi } from '@/api/task'
import dayjs from 'dayjs'

const loading = ref(false)
const tableData = ref([])
const logDialogVisible = ref(false)
const currentLogs = ref({ output: '', error_msg: '' })

const searchForm = reactive({
  task_id: '',
  client_id: '',
  status: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const stats = reactive({
  total: 0,
  success: 0,
  failed: 0,
  running: 0
})

// 格式化日期
const formatDate = (date) => {
  return date ? dayjs(date).format('YYYY-MM-DD HH:mm:ss') : '-'
}

// 获取状态类型
const getStatusType = (status) => {
  const types = {
    1: 'info',      // 待执行
    2: 'warning',   // 执行中
    3: 'success',   // 成功
    4: 'danger',    // 失败
    5: 'warning',   // 超时
    6: 'info'       // 取消
  }
  return types[status] || 'info'
}

// 获取状态文本
const getStatusText = (status) => {
  const texts = {
    1: '待执行',
    2: '执行中',
    3: '成功',
    4: '失败',
    5: '超时',
    6: '取消'
  }
  return texts[status] || '未知'
}

// 加载执行记录列表
const loadExecutions = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      task_id: searchForm.task_id || undefined,
      client_id: searchForm.client_id || undefined,
      status: searchForm.status !== '' ? searchForm.status : undefined
    }
    const res = await executionApi.listExecutions(params)
    if (res.success) {
      tableData.value = res.data.executions || []
      pagination.total = res.data.total || 0
    }
  } catch (error) {
    ElMessage.error('加载执行记录失败')
  } finally {
    loading.value = false
  }
}

// 加载统计信息
const loadStats = async () => {
  try {
    const params = {
      task_id: searchForm.task_id || undefined,
      client_id: searchForm.client_id || undefined
    }
    const res = await executionApi.getExecutionStats(params)
    if (res.success) {
      Object.assign(stats, res.data)
    }
  } catch (error) {
    console.error('加载统计信息失败:', error)
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadExecutions()
  loadStats()
}

// 重置
const handleReset = () => {
  searchForm.task_id = ''
  searchForm.client_id = ''
  searchForm.status = ''
  handleSearch()
}

// 分页变化
const handlePageChange = (page) => {
  pagination.page = page
  loadExecutions()
}

const handleSizeChange = (size) => {
  pagination.pageSize = size
  pagination.page = 1
  loadExecutions()
}

// 查看日志
const handleViewLogs = async (row) => {
  try {
    const res = await executionApi.getExecutionLogs(row.id)
    if (res.success) {
      currentLogs.value = {
        output: res.data.output || '',
        error_msg: res.data.error_msg || ''
      }
      logDialogVisible.value = true
    }
  } catch (error) {
    ElMessage.error('加载日志失败')
  }
}

// 查看详情
const handleViewDetail = async (row) => {
  // TODO: 实现详情页面
  ElMessage.info('详情功能开发中')
}

onMounted(() => {
  loadExecutions()
  loadStats()
})
</script>

<style scoped>
.execution-list {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-form {
  margin-bottom: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-item {
  text-align: center;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #409eff;
}

.stat-value.success {
  color: #67c23a;
}

.stat-value.failed {
  color: #f56c6c;
}

.stat-label {
  margin-top: 8px;
  font-size: 14px;
  color: #909399;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.log-content {
  padding: 10px;
  background: #f5f5f5;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  max-height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-content.error {
  color: #f56c6c;
}
</style>
