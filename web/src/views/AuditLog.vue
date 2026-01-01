<template>
  <div class="audit-log-container">
    <div class="header">
      <h2>命令审计日志</h2>
      <div class="stats" v-if="stats">
        <el-tag type="info">总命令: {{ stats.total_commands }}</el-tag>
        <el-tag type="success">总会话: {{ stats.total_sessions }}</el-tag>
        <el-tag type="warning">总客户端: {{ stats.total_clients }}</el-tag>
      </div>
    </div>

    <!-- 过滤条件 -->
    <el-card class="filter-card">
      <el-form :inline="true" :model="filter" class="filter-form">
        <el-form-item label="客户端 ID">
          <el-input v-model="filter.client_id" placeholder="输入客户端 ID" clearable />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="filter.username" placeholder="输入用户名" clearable />
        </el-form-item>
        <el-form-item label="命令">
          <el-input v-model="filter.command" placeholder="命令关键字" clearable />
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            @change="handleDateChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchCommands">
            <el-icon><Search /></el-icon>
            查询
          </el-button>
          <el-button @click="resetFilter">重置</el-button>
          <el-button type="success" @click="exportData">
            <el-icon><Download /></el-icon>
            导出
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 命令列表 -->
    <el-card class="commands-card">
      <el-table
        :data="commands"
        v-loading="loading"
        stripe
        border
        style="width: 100%"
        :default-sort="{ prop: 'executed_at', order: 'descending' }"
      >
        <el-table-column prop="executed_at" label="执行时间" width="180">
          <template #default="scope">
            {{ formatTime(scope.row.executed_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="client_id" label="客户端 ID" width="150" />
        <el-table-column prop="username" label="用户名" width="100" />
        <el-table-column prop="command" label="命令" min-width="300">
          <template #default="scope">
            <code class="command-text">{{ scope.row.command }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="exit_code" label="退出码" width="80" align="center">
          <template #default="scope">
            <el-tag :type="scope.row.exit_code === 0 ? 'success' : 'danger'" size="small">
              {{ scope.row.exit_code }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="duration_ms" label="耗时" width="100" align="right">
          <template #default="scope">
            {{ scope.row.duration_ms ? scope.row.duration_ms + 'ms' : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="remote_ip" label="来源 IP" width="130" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="scope">
            <el-button size="small" link @click="showDetail(scope.row)">
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        class="pagination"
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[20, 50, 100, 200]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailVisible" title="命令详情" width="600px">
      <el-descriptions :column="1" border v-if="selectedCommand">
        <el-descriptions-item label="ID">{{ selectedCommand.id }}</el-descriptions-item>
        <el-descriptions-item label="会话 ID">{{ selectedCommand.session_id }}</el-descriptions-item>
        <el-descriptions-item label="客户端 ID">{{ selectedCommand.client_id }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ selectedCommand.username }}</el-descriptions-item>
        <el-descriptions-item label="执行时间">{{ formatTime(selectedCommand.executed_at) }}</el-descriptions-item>
        <el-descriptions-item label="来源 IP">{{ selectedCommand.remote_ip }}</el-descriptions-item>
        <el-descriptions-item label="退出码">
          <el-tag :type="selectedCommand.exit_code === 0 ? 'success' : 'danger'">
            {{ selectedCommand.exit_code }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="耗时">{{ selectedCommand.duration_ms }}ms</el-descriptions-item>
        <el-descriptions-item label="命令">
          <code class="command-detail">{{ selectedCommand.command }}</code>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Download } from '@element-plus/icons-vue'
import api, { request } from '../api'

const loading = ref(false)
const commands = ref([])
const stats = ref(null)
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(50)
const dateRange = ref(null)
const detailVisible = ref(false)
const selectedCommand = ref(null)

const filter = reactive({
  client_id: '',
  username: '',
  command: '',
  start_time: '',
  end_time: ''
})

const fetchCommands = async () => {
  loading.value = true
  try {
    const params = {
      ...filter,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }
    // Remove empty params
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null) {
        delete params[key]
      }
    })

    const res = await request.get('/audit/commands', { params })
    // res is already response.data due to interceptor
    if (res.success) {
      commands.value = res.commands || []
      total.value = res.count || 0
    }
  } catch (err) {
    ElMessage.error('获取命令列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const fetchStats = async () => {
  try {
    const res = await request.get('/audit/stats')
    // res is already response.data due to interceptor
    if (res.success) {
      stats.value = res.stats
    }
  } catch (err) {
    console.error('获取统计信息失败:', err)
  }
}

const handleDateChange = (val) => {
  if (val) {
    filter.start_time = val[0].toISOString()
    filter.end_time = val[1].toISOString()
  } else {
    filter.start_time = ''
    filter.end_time = ''
  }
}

const resetFilter = () => {
  filter.client_id = ''
  filter.username = ''
  filter.command = ''
  filter.start_time = ''
  filter.end_time = ''
  dateRange.value = null
  currentPage.value = 1
  fetchCommands()
}

const handleSizeChange = () => {
  currentPage.value = 1
  fetchCommands()
}

const handlePageChange = () => {
  fetchCommands()
}

const showDetail = (row) => {
  selectedCommand.value = row
  detailVisible.value = true
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const exportData = async () => {
  try {
    window.open('/api/audit/export?format=csv', '_blank')
  } catch (err) {
    ElMessage.error('导出失败: ' + err.message)
  }
}

onMounted(() => {
  fetchCommands()
  fetchStats()
})
</script>

<style scoped>
.audit-log-container {
  padding: 20px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h2 {
  margin: 0;
}

.stats {
  display: flex;
  gap: 10px;
}

.filter-card {
  margin-bottom: 20px;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.commands-card {
  margin-bottom: 20px;
}

.command-text {
  background-color: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  word-break: break-all;
}

.command-detail {
  background-color: #1e1e1e;
  color: #d4d4d4;
  padding: 10px;
  border-radius: 4px;
  display: block;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
