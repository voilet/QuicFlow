<template>
  <div class="command-history">
    <!-- 筛选条件 -->
    <el-card shadow="never" class="filter-card">
      <el-form :inline="true" :model="filters">
        <el-form-item label="客户端ID">
          <el-select
            v-model="filters.client_id"
            placeholder="全部客户端"
            clearable
            filterable
            style="width: 200px"
          >
            <el-option
              v-for="client in clients"
              :key="client.client_id"
              :label="client.client_id"
              :value="client.client_id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="命令状态">
          <el-select
            v-model="filters.status"
            placeholder="全部状态"
            clearable
            style="width: 150px"
          >
            <el-option label="等待执行" value="pending" />
            <el-option label="执行中" value="executing" />
            <el-option label="执行成功" value="completed" />
            <el-option label="执行失败" value="failed" />
            <el-option label="执行超时" value="timeout" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :icon="Search" @click="searchCommands">
            查询
          </el-button>
          <el-button :icon="Refresh" @click="resetFilters">重置</el-button>
          <el-button
            type="success"
            :icon="Download"
            @click="exportCommands"
          >
            导出
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 命令列表 -->
    <el-card shadow="never" class="list-card">
      <template #header>
        <div class="card-header">
          <span>命令历史（共 {{ total }} 条）</span>
          <el-button
            type="primary"
            :icon="Refresh"
            @click="loadCommands"
            :loading="loading"
            size="small"
          >
            刷新
          </el-button>
        </div>
      </template>

      <el-table
        :data="commands"
        v-loading="loading"
        stripe
        style="width: 100%"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-content">
              <el-descriptions :column="2" border size="small">
                <el-descriptions-item label="命令ID" :span="2">
                  <el-tag>{{ row.command_id }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="命令参数" :span="2">
                  <pre class="json-code">{{ formatJSON(row.payload) }}</pre>
                </el-descriptions-item>
                <el-descriptions-item label="执行结果" :span="2" v-if="row.result">
                  <pre class="json-code">{{ formatJSON(row.result) }}</pre>
                </el-descriptions-item>
                <el-descriptions-item label="错误信息" :span="2" v-if="row.error">
                  <el-alert type="error" :closable="false">
                    {{ row.error }}
                  </el-alert>
                </el-descriptions-item>
                <el-descriptions-item label="创建时间">
                  {{ formatTime(row.created_at) }}
                </el-descriptions-item>
                <el-descriptions-item label="完成时间">
                  {{ row.completed_at ? formatTime(row.completed_at) : '-' }}
                </el-descriptions-item>
                <el-descriptions-item label="超时设置">
                  {{ formatDuration(row.timeout) }}
                </el-descriptions-item>
                <el-descriptions-item label="执行时长" v-if="row.completed_at">
                  {{ calculateDuration(row.created_at, row.completed_at) }}
                </el-descriptions-item>
              </el-descriptions>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="client_id" label="客户端ID" width="180">
          <template #default="{ row }">
            <el-tag>{{ row.client_id }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="command_type" label="命令类型" width="150">
          <template #default="{ row }">
            <el-tag type="info">{{ row.command_type }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="完成时间" width="180">
          <template #default="{ row }">
            {{ row.completed_at ? formatTime(row.completed_at) : '-' }}
          </template>
        </el-table-column>

        <el-table-column label="执行时长" width="100">
          <template #default="{ row }">
            {{ row.completed_at ? calculateDuration(row.created_at, row.completed_at) : '-' }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              :icon="View"
              @click="viewDetail(row)"
            >
              详情
            </el-button>
            <el-button
              type="success"
              size="small"
              :icon="Refresh"
              @click="retryCommand(row)"
              v-if="row.status === 'failed' || row.status === 'timeout'"
            >
              重试
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="命令详情"
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-if="selectedCommand" class="command-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="命令ID" :span="2">
            <el-tag>{{ selectedCommand.command_id }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="客户端">
            <el-tag>{{ selectedCommand.client_id }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="命令类型">
            <el-tag type="info">{{ selectedCommand.command_type }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态" :span="2">
            <el-tag :type="getStatusType(selectedCommand.status)">
              {{ getStatusText(selectedCommand.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ formatTime(selectedCommand.created_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="完成时间">
            {{ selectedCommand.completed_at ? formatTime(selectedCommand.completed_at) : '-' }}
          </el-descriptions-item>
        </el-descriptions>

        <div class="detail-section">
          <h4>命令参数</h4>
          <el-input
            type="textarea"
            :rows="6"
            :value="formatJSON(selectedCommand.payload)"
            readonly
          />
        </div>

        <div class="detail-section" v-if="selectedCommand.result">
          <h4>执行结果</h4>
          <el-input
            type="textarea"
            :rows="6"
            :value="formatJSON(selectedCommand.result)"
            readonly
          />
        </div>

        <div class="detail-section" v-if="selectedCommand.error">
          <h4>错误信息</h4>
          <el-alert type="error" :closable="false">
            {{ selectedCommand.error }}
          </el-alert>
        </div>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">关闭</el-button>
        <el-button
          type="primary"
          :icon="Refresh"
          @click="refreshDetail"
        >
          刷新
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Refresh,
  Download,
  View
} from '@element-plus/icons-vue'
import api from '@/api'
import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'

dayjs.extend(duration)

const route = useRoute()
const router = useRouter()

const clients = ref([])
const commands = ref([])
const loading = ref(false)
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const dialogVisible = ref(false)
const selectedCommand = ref(null)

// 筛选条件
const filters = reactive({
  client_id: route.query.client_id || '',
  status: ''
})

// 加载客户端列表
async function loadClients() {
  try {
    const res = await api.getClients()
    clients.value = res.clients || []
  } catch (error) {
    console.error('加载客户端列表失败', error)
  }
}

// 加载命令列表
async function loadCommands() {
  loading.value = true
  try {
    const params = {
      ...filters,
      page: currentPage.value,
      page_size: pageSize.value
    }
    const res = await api.getCommands(params)
    commands.value = res.commands || []
    total.value = res.total || 0
  } catch (error) {
    ElMessage.error('加载命令列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索命令
function searchCommands() {
  currentPage.value = 1
  loadCommands()
}

// 重置筛选
function resetFilters() {
  filters.client_id = ''
  filters.status = ''
  currentPage.value = 1
  loadCommands()
}

// 导出命令
function exportCommands() {
  ElMessage.info('导出功能开发中...')
}

// 查看详情
async function viewDetail(command) {
  try {
    const res = await api.getCommand(command.command_id)
    selectedCommand.value = res.command
    dialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取命令详情失败')
  }
}

// 刷新详情
async function refreshDetail() {
  if (selectedCommand.value) {
    await viewDetail(selectedCommand.value)
    ElMessage.success('刷新成功')
  }
}

// 重试命令
async function retryCommand(command) {
  try {
    await ElMessageBox.confirm(
      `确定要重新下发此命令吗？`,
      '确认重试',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const payload = typeof command.payload === 'string'
      ? JSON.parse(command.payload)
      : command.payload

    await api.sendCommand({
      client_id: command.client_id,
      command_type: command.command_type,
      payload,
      timeout: 30
    })

    ElMessage.success('命令已重新下发')
    loadCommands()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('重试失败')
    }
  }
}

// 分页处理
function handleSizeChange() {
  currentPage.value = 1
  loadCommands()
}

function handleCurrentChange() {
  loadCommands()
}

// 格式化时间
function formatTime(timestamp) {
  return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
}

// 格式化JSON
function formatJSON(obj) {
  if (typeof obj === 'string') {
    try {
      obj = JSON.parse(obj)
    } catch (e) {
      return obj
    }
  }
  return JSON.stringify(obj, null, 2)
}

// 格式化时长
function formatDuration(nanoseconds) {
  const seconds = Math.floor(nanoseconds / 1000000000)
  return `${seconds}s`
}

// 计算执行时长
function calculateDuration(start, end) {
  const diff = dayjs(end).diff(dayjs(start))
  const dur = dayjs.duration(diff)
  const seconds = dur.asSeconds()
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`
  }
  const minutes = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${minutes}m${secs}s`
}

// 获取状态类型
function getStatusType(status) {
  const types = {
    pending: 'warning',
    executing: 'info',
    completed: 'success',
    failed: 'danger',
    timeout: 'danger'
  }
  return types[status] || 'info'
}

// 获取状态文本
function getStatusText(status) {
  const texts = {
    pending: '等待执行',
    executing: '执行中',
    completed: '执行成功',
    failed: '执行失败',
    timeout: '执行超时'
  }
  return texts[status] || status
}

onMounted(() => {
  loadClients()
  loadCommands()
  // 自动刷新
  setInterval(loadCommands, 15000)
})
</script>

<style scoped>
.command-history {
  width: 100%;
}

.filter-card {
  margin-bottom: 20px;
}

.list-card {
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.expand-content {
  padding: 20px;
}

.json-code {
  margin: 0;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  max-height: 300px;
  overflow: auto;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.command-detail {
  padding: 10px 0;
}

.detail-section {
  margin-top: 20px;
}

.detail-section h4 {
  margin-bottom: 10px;
  color: #303133;
}
</style>
