<template>
  <div class="audit-log-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <h2>命令审计日志</h2>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row" v-if="stats">
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-icon info">
            <el-icon :size="30"><Document /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_commands }}</div>
            <div class="stat-label">总命令数</div>
          </div>
        </div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-icon success">
            <el-icon :size="30"><Connection /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_sessions }}</div>
            <div class="stat-label">总会话数</div>
          </div>
        </div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-icon warning">
            <el-icon :size="30"><Monitor /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_clients }}</div>
            <div class="stat-label">总客户端数</div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 过滤条件 -->
    <el-card class="filter-card" shadow="hover">
      <template #header>
        <div class="card-header-title">
          <el-icon><Search /></el-icon>
          <span>筛选条件</span>
        </div>
      </template>
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
    <el-card class="commands-card" shadow="hover">
      <template #header>
        <div class="card-header-title">
          <el-icon><Document /></el-icon>
          <span>命令列表</span>
          <el-tag type="info" size="small" class="record-count-tag">
            {{ total }} 条记录
          </el-tag>
        </div>
      </template>
      <div class="commands-table-wrapper">
        <el-table
          :data="commands"
          v-loading="loading"
          stripe
          class="commands-table"
          :default-sort="{ prop: 'executed_at', order: 'descending' }"
          table-layout="auto"
        >
          <el-table-column prop="executed_at" label="执行时间" width="180">
            <template #default="scope">
              <span class="table-cell-time">{{ formatTime(scope.row.executed_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="client_id" label="客户端 ID" width="150">
            <template #default="scope">
              <span class="table-cell-text">{{ scope.row.client_id }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="username" label="用户名" width="120">
            <template #default="scope">
              <span class="table-cell-text">{{ scope.row.username }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="command" label="命令" min-width="400">
            <template #default="scope">
              <code class="command-text">{{ scope.row.command }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="remote_ip" label="来源 IP" width="140">
            <template #default="scope">
              <span class="table-cell-text">{{ scope.row.remote_ip }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="scope">
              <el-button size="small" @click="showDetail(scope.row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

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
    <el-dialog 
      v-model="detailVisible" 
      title="命令详情" 
      width="700px"
      class="detail-dialog"
    >
      <el-descriptions :column="1" border v-if="selectedCommand" class="command-descriptions">
        <el-descriptions-item label="ID">{{ selectedCommand.id }}</el-descriptions-item>
        <el-descriptions-item label="会话 ID">{{ selectedCommand.session_id }}</el-descriptions-item>
        <el-descriptions-item label="客户端 ID">{{ selectedCommand.client_id }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ selectedCommand.username }}</el-descriptions-item>
        <el-descriptions-item label="执行时间">{{ formatTime(selectedCommand.executed_at) }}</el-descriptions-item>
        <el-descriptions-item label="来源 IP">{{ selectedCommand.remote_ip }}</el-descriptions-item>
        <!-- <el-descriptions-item label="退出码">
          <el-tag :type="selectedCommand.exit_code === 0 ? 'success' : 'danger'">
            {{ selectedCommand.exit_code }}
          </el-tag>
        </el-descriptions-item> -->
        <!-- <el-descriptions-item label="耗时">{{ selectedCommand.duration_ms }}ms</el-descriptions-item> -->
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
import { Search, Download, Document, Connection, Monitor } from '@element-plus/icons-vue'
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
  padding: 0;
}

.page-header {
  margin-bottom: 20px;
  padding: 0 4px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--tech-primary);
  letter-spacing: -0.02em;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  position: relative;
  overflow: hidden;
  box-shadow: var(--tech-shadow-sm);
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(64, 158, 255, 0.08),
    transparent
  );
  transition: left 0.6s ease;
}

.stat-card::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--tech-gradient-primary);
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.stat-card:hover {
  transform: translateY(-4px);
  border-color: rgba(64, 158, 255, 0.4);
  box-shadow: var(--tech-shadow-md);
}

.stat-card:hover::before {
  left: 100%;
}

.stat-card:hover::after {
  transform: scaleX(1);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
  position: relative;
  z-index: 1;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon.info {
  background: linear-gradient(135deg, rgba(144, 147, 153, 0.15) 0%, rgba(144, 147, 153, 0.08) 100%);
  border: 1px solid rgba(144, 147, 153, 0.3);
  color: var(--tech-info);
  box-shadow: 0 4px 12px rgba(144, 147, 153, 0.2);
}

.stat-icon.success {
  background: linear-gradient(135deg, rgba(103, 194, 58, 0.15) 0%, rgba(103, 194, 58, 0.08) 100%);
  border: 1px solid rgba(103, 194, 58, 0.3);
  color: var(--tech-secondary);
  box-shadow: 0 4px 12px rgba(103, 194, 58, 0.2);
}

.stat-icon.warning {
  background: linear-gradient(135deg, rgba(230, 162, 60, 0.15) 0%, rgba(230, 162, 60, 0.08) 100%);
  border: 1px solid rgba(230, 162, 60, 0.3);
  color: var(--tech-warning);
  box-shadow: 0 4px 12px rgba(230, 162, 60, 0.2);
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--tech-primary);
  margin-bottom: 6px;
  line-height: 1.2;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.stat-card:hover .stat-value {
  transform: scale(1.05);
  color: var(--tech-primary-light);
}

.stat-label {
  font-size: 14px;
  color: var(--tech-text-secondary);
  font-weight: 500;
}

.filter-card {
  margin-bottom: 20px;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
}

.filter-card :deep(.el-card__header) {
  background: var(--tech-bg-tertiary);
  border-bottom: 1px solid var(--tech-border);
  padding: 15px 20px;
}

.card-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.card-header-title .el-icon {
  color: var(--tech-primary);
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-end;
}

.commands-card {
  margin-bottom: 20px;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
}

.commands-card :deep(.el-card__header) {
  background: var(--tech-bg-tertiary);
  border-bottom: 1px solid var(--tech-border);
  padding: 15px 20px;
}

.record-count-tag {
  margin-left: auto;
}

.commands-table-wrapper {
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  overflow: hidden;
}

.commands-table {
  background: transparent;
}

.commands-table :deep(.el-table__header-wrapper) {
  background: transparent;
}

.commands-table :deep(.el-table__header) {
  background: transparent;
}

.commands-table :deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  font-weight: 600;
  padding: 12px;
  font-size: 14px;
  transition: all 0.3s ease;
}

.commands-table :deep(.el-table th:hover) {
  background-color: var(--tech-bg-tertiary);
  color: var(--tech-text-primary);
}

.commands-table :deep(.el-table td) {
  border-color: var(--tech-border);
  padding: 12px;
  transition: all 0.2s ease;
}

.commands-table :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: var(--tech-bg-tertiary);
}

.commands-table :deep(.el-table__row) {
  transition: all 0.2s ease;
}

.commands-table :deep(.el-table__row:hover) {
  background-color: var(--tech-bg-tertiary);
}

.commands-table :deep(.el-table__row:hover td) {
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
}

.table-cell-text {
  color: var(--tech-text-primary);
  font-size: 14px;
}

.table-cell-time {
  color: var(--tech-text-secondary);
  font-size: 13px;
  font-family: var(--tech-font-mono);
}

.command-text {
  font-family: var(--tech-font-mono);
  font-size: 13px;
  word-break: break-all;
  color: var(--tech-text-primary);
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
  padding: 0 4px;
}

.pagination :deep(.el-pagination .el-pager li.is-active) {
  background-color: var(--tech-primary);
  border-color: var(--tech-primary);
  color: #ffffff;
  font-weight: 600;
}

/* 对话框美化 */
.detail-dialog :deep(.el-dialog) {
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  border-radius: 8px;
}

.detail-dialog :deep(.el-dialog__header) {
  background: var(--tech-bg-tertiary);
  border-bottom: 1px solid var(--tech-border);
  padding: 20px;
}

.detail-dialog :deep(.el-dialog__title) {
  font-weight: 600;
  color: var(--tech-text-primary);
}

.command-descriptions {
  margin-top: 20px;
}

.command-descriptions :deep(.el-descriptions__label) {
  font-weight: 600;
  color: var(--tech-text-secondary);
}

.command-descriptions :deep(.el-descriptions__content) {
  color: var(--tech-text-primary);
}

.command-detail {
  background-color: var(--tech-bg-primary);
  color: var(--tech-text-primary);
  padding: 12px;
  border-radius: 4px;
  display: block;
  font-family: var(--tech-font-mono);
  white-space: pre-wrap;
  word-break: break-all;
  border: 1px solid var(--tech-border);
  font-size: 13px;
  line-height: 1.6;
}

[data-theme="dark"] .command-detail {
  background-color: #1e1e1e;
  color: #d4d4d4;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
  
  .filter-form {
    flex-direction: column;
    align-items: stretch;
  }
  
  .filter-form .el-form-item {
    margin-bottom: 0;
  }
}
</style>
