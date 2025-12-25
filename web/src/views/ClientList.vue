<template>
  <div class="client-list">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon online">
              <el-icon :size="30"><Connection /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ clients.length }}</div>
              <div class="stat-label">在线客户端</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon success">
              <el-icon :size="30"><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ totalConnections }}</div>
              <div class="stat-label">总连接数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon warning">
              <el-icon :size="30"><Message /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ messagesSent }}</div>
              <div class="stat-label">发送消息数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon info">
              <el-icon :size="30"><Clock /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ averageUptime }}</div>
              <div class="stat-label">平均在线时长</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 客户端列表 -->
    <el-card shadow="never" class="list-card">
      <template #header>
        <div class="card-header">
          <span>客户端列表</span>
          <div class="header-actions">
            <el-button
              type="primary"
              :icon="Refresh"
              @click="loadClients"
              :loading="loading"
            >
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-table
        :data="clients"
        v-loading="loading"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="client_id" label="客户端ID" min-width="200">
          <template #default="{ row }">
            <el-tag type="success">{{ row.client_id }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="remote_addr" label="远程地址" min-width="150" />

        <el-table-column label="连接时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.connected_at) }}
          </template>
        </el-table-column>

        <el-table-column label="在线时长" min-width="120">
          <template #default="{ row }">
            <el-tag type="info">{{ row.uptime }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default>
            <el-tag type="success">
              <el-icon><CircleCheck /></el-icon>
              在线
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              :icon="DocumentAdd"
              @click="sendCommand(row.client_id)"
            >
              下发命令
            </el-button>
            <el-button
              type="info"
              size="small"
              :icon="Document"
              @click="viewHistory(row.client_id)"
            >
              命令历史
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="clients.length === 0 && !loading" class="empty-state">
        <el-empty description="暂无客户端连接" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  Connection,
  CircleCheck,
  Message,
  Clock,
  Refresh,
  DocumentAdd,
  Document
} from '@element-plus/icons-vue'
import api from '@/api'
import dayjs from 'dayjs'

const router = useRouter()

const clients = ref([])
const loading = ref(false)
const totalConnections = ref(0)
const messagesSent = ref(0)

// 计算平均在线时长
const averageUptime = computed(() => {
  if (clients.value.length === 0) return '0s'
  const total = clients.value.reduce((sum, client) => {
    const uptime = client.uptime || '0s'
    return sum + parseUptimeToSeconds(uptime)
  }, 0)
  const avg = Math.floor(total / clients.value.length)
  return formatSeconds(avg)
})

// 解析uptime字符串为秒数
function parseUptimeToSeconds(uptime) {
  const match = uptime.match(/(\d+)h(\d+)m(\d+)s/)
  if (match) {
    return parseInt(match[1]) * 3600 + parseInt(match[2]) * 60 + parseInt(match[3])
  }
  return 0
}

// 格式化秒数为可读字符串
function formatSeconds(seconds) {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}h${m}m${s}s`
  if (m > 0) return `${m}m${s}s`
  return `${s}s`
}

// 格式化时间
function formatTime(timestamp) {
  return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
}

// 加载客户端列表
async function loadClients() {
  loading.value = true
  try {
    const res = await api.getClients()
    clients.value = res.clients || []
    totalConnections.value = res.total || 0
    ElMessage.success('刷新成功')
  } catch (error) {
    ElMessage.error('加载客户端列表失败')
  } finally {
    loading.value = false
  }
}

// 下发命令
function sendCommand(clientId) {
  router.push({
    path: '/command',
    query: { client_id: clientId }
  })
}

// 查看命令历史
function viewHistory(clientId) {
  router.push({
    path: '/history',
    query: { client_id: clientId }
  })
}

onMounted(() => {
  loadClients()
})
</script>

<style scoped>
.client-list {
  width: 100%;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  cursor: pointer;
  transition: transform 0.3s;
}

.stat-card:hover {
  transform: translateY(-5px);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 15px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-icon.online {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.success {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-icon.warning {
  background: linear-gradient(135deg, #ffc107 0%, #ff9800 100%);
}

.stat-icon.info {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
  margin-bottom: 5px;
}

.stat-label {
  font-size: 14px;
  color: #909399;
}

.list-card {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.empty-state {
  padding: 40px 0;
}
</style>
