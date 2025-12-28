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
              <div class="stat-value">{{ totalClients }}</div>
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
              type="success"
              :icon="Position"
              @click="batchSendCommand"
              :disabled="selectedClients.length === 0"
            >
              批量下发 ({{ selectedClients.length }})
            </el-button>
            <el-button
              type="warning"
              link
              @click="selectAllClients"
              v-if="clients.length > 0"
            >
              全选
            </el-button>
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
        ref="tableRef"
        :data="clients"
        v-loading="loading"
        stripe
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
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

        <el-table-column label="操作" width="480" fixed="right">
          <template #default="{ row }">
            <el-button
              type="success"
              size="small"
              :icon="Monitor"
              @click="getHardwareInfo(row.client_id)"
              :loading="hardwareLoading[row.client_id]"
            >
              硬件信息
            </el-button>
            <el-popover
              placement="top"
              :width="200"
              trigger="hover"
            >
              <template #reference>
                <el-button
                  type="warning"
                  size="small"
                  :icon="Odometer"
                  @click="runDiskBenchmark(row.client_id)"
                  :loading="benchmarkLoading[row.client_id]"
                >
                  磁盘测试
                </el-button>
              </template>
              <div class="benchmark-options">
                <el-switch
                  v-model="benchmarkConcurrent"
                  active-text="并发"
                  inactive-text="顺序"
                  style="margin-bottom: 8px;"
                />
                <div class="option-tip">{{ benchmarkConcurrent ? '同时测试所有磁盘' : '依次测试每块磁盘' }}</div>
              </div>
            </el-popover>
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[50, 100, 200, 500]"
          :total="totalClients"
          :disabled="loading"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>

      <div v-if="clients.length === 0 && !loading" class="empty-state">
        <el-empty description="暂无客户端连接" />
      </div>
    </el-card>

    <!-- 硬件信息对话框 -->
    <el-dialog
      v-model="hardwareDialogVisible"
      :title="`硬件信息 - ${currentClientId}`"
      width="900px"
      top="5vh"
    >
      <div v-if="hardwareInfo" class="hardware-info">
        <!-- 主机信息 -->
        <el-card shadow="never" class="info-section">
          <template #header>
            <span class="section-title">主机信息</span>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="主机名">{{ hardwareInfo.host?.hostname }}</el-descriptions-item>
            <el-descriptions-item label="操作系统">{{ hardwareInfo.host?.os }}</el-descriptions-item>
            <el-descriptions-item label="平台">{{ hardwareInfo.host?.platform }} {{ hardwareInfo.host?.platform_version }}</el-descriptions-item>
            <el-descriptions-item label="内核版本">{{ hardwareInfo.host?.kernel_version }}</el-descriptions-item>
            <el-descriptions-item label="架构">{{ hardwareInfo.host?.kernel_arch }}</el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ formatUptime(hardwareInfo.host?.uptime) }}</el-descriptions-item>
            <el-descriptions-item label="虚拟化">{{ hardwareInfo.host?.virtualization_system || '无' }} ({{ hardwareInfo.host?.virtualization_role || '-' }})</el-descriptions-item>
            <el-descriptions-item label="主机ID">{{ hardwareInfo.host?.host_id }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <!-- CPU 信息 -->
        <el-card shadow="never" class="info-section">
          <template #header>
            <span class="section-title">CPU 信息</span>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="型号" :span="2">{{ hardwareInfo.model_name }}</el-descriptions-item>
            <el-descriptions-item label="物理核心">{{ hardwareInfo.cpu_core_count }}</el-descriptions-item>
            <el-descriptions-item label="逻辑处理器">{{ hardwareInfo.cpu_thread_count }}</el-descriptions-item>
            <el-descriptions-item label="频率">{{ hardwareInfo.physical_cpu_frequency_mhz }} MHz</el-descriptions-item>
            <el-descriptions-item label="内核报告CPU数">{{ hardwareInfo.num_cpu_kernel }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <!-- 内存信息 -->
        <el-card shadow="never" class="info-section">
          <template #header>
            <span class="section-title">内存信息</span>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="总容量">{{ hardwareInfo.memory?.total_gb_rounded }} GB</el-descriptions-item>
            <el-descriptions-item label="内存条数量">{{ hardwareInfo.memory?.count }}</el-descriptions-item>
          </el-descriptions>
          <el-table v-if="hardwareInfo.memory?.modules?.length" :data="hardwareInfo.memory.modules" size="small" class="sub-table">
            <el-table-column prop="locator" label="插槽" width="100" />
            <el-table-column prop="size" label="容量" width="120" />
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="manufacturer" label="制造商" />
          </el-table>
        </el-card>

        <!-- 磁盘信息 -->
        <el-card shadow="never" class="info-section">
          <template #header>
            <span class="section-title">磁盘信息 (总容量: {{ hardwareInfo.total_disk_capacity_tb?.toFixed(2) }} TB)</span>
          </template>
          <el-table :data="hardwareInfo.disks" size="small">
            <el-table-column prop="device" label="设备" width="100" />
            <el-table-column prop="model" label="型号" min-width="150" />
            <el-table-column prop="kind" label="类型" width="80">
              <template #default="{ row }">
                <el-tag :type="row.kind === 'SSD' ? 'success' : row.kind === 'NVMe' ? 'warning' : 'info'" size="small">
                  {{ row.kind }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="容量" width="100">
              <template #default="{ row }">
                {{ row.size_rounded_tb >= 1 ? row.size_rounded_tb.toFixed(2) + ' TB' : (row.size_rounded_bytes / 1024 / 1024 / 1024).toFixed(0) + ' GB' }}
              </template>
            </el-table-column>
            <el-table-column label="系统盘" width="80">
              <template #default="{ row }">
                <el-tag v-if="row.is_system_disk" type="danger" size="small">是</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 网卡信息 -->
        <el-card shadow="never" class="info-section">
          <template #header>
            <span class="section-title">网卡信息 (主MAC: {{ hardwareInfo.mac }})</span>
          </template>
          <el-table :data="hardwareInfo.nic_infos" size="small">
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="mac_address" label="MAC地址" width="150" />
            <el-table-column prop="ip_address" label="IPv4" width="130" />
            <el-table-column prop="ipv6" label="IPv6" min-width="200">
              <template #default="{ row }">
                <span class="ipv6-text">{{ row.ipv6 || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="speed" label="速率" width="120" />
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'up' ? 'success' : 'danger'" size="small">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="物理" width="70">
              <template #default="{ row }">
                <el-icon v-if="row.is_physical" color="#67c23a"><CircleCheck /></el-icon>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- DMI 信息 -->
        <el-card shadow="never" class="info-section">
          <template #header>
            <span class="section-title">DMI/BIOS 信息</span>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="系统厂商">{{ hardwareInfo.dmi?.sys_vendor }}</el-descriptions-item>
            <el-descriptions-item label="产品名称">{{ hardwareInfo.dmi?.product_name }}</el-descriptions-item>
            <el-descriptions-item label="产品UUID">{{ hardwareInfo.dmi?.product_uuid }}</el-descriptions-item>
            <el-descriptions-item label="BIOS厂商">{{ hardwareInfo.dmi?.bios_vendor }}</el-descriptions-item>
            <el-descriptions-item label="BIOS版本">{{ hardwareInfo.dmi?.bios_version }}</el-descriptions-item>
            <el-descriptions-item label="BIOS日期">{{ hardwareInfo.dmi?.bios_date }}</el-descriptions-item>
            <el-descriptions-item label="机箱类型">{{ hardwareInfo.dmi?.chassis_type }}</el-descriptions-item>
            <el-descriptions-item label="机箱厂商">{{ hardwareInfo.dmi?.chassis_vendor }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </div>

      <template #footer>
        <el-button @click="hardwareDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="copyHardwareInfo">复制JSON</el-button>
      </template>
    </el-dialog>

    <!-- 磁盘测试对话框 -->
    <el-dialog
      v-model="benchmarkDialogVisible"
      :title="`磁盘 IO 性能测试 - ${currentBenchmarkClientId}`"
      width="1000px"
      top="5vh"
    >
      <div v-if="benchmarkResult" class="benchmark-info">
        <el-alert
          :title="`测试完成时间: ${benchmarkResult.tested_at} | 测试磁盘数: ${benchmarkResult.total_disks} | 模式: ${benchmarkResult.message?.includes('concurrent') ? '并发' : '顺序'}`"
          type="success"
          :closable="false"
          class="benchmark-summary"
        />

        <div v-for="(disk, index) in benchmarkResult.results" :key="index" class="disk-result">
          <el-card shadow="never" class="info-section">
            <template #header>
              <div class="disk-header">
                <span class="section-title">{{ disk.device }} - {{ disk.model }}</span>
                <el-tag :type="disk.kind === 'NVMe' ? 'warning' : disk.kind === 'SSD' ? 'success' : 'info'">
                  {{ disk.kind }}
                </el-tag>
              </div>
            </template>

            <el-row :gutter="20">
              <!-- 顺序读写 -->
              <el-col :span="12">
                <div class="perf-card seq-read">
                  <div class="perf-title">顺序读 (1M)</div>
                  <div class="perf-metrics">
                    <div class="metric">
                      <span class="metric-value">{{ formatNumber(disk.seq_read_bw_mbps) }}</span>
                      <span class="metric-unit">MB/s</span>
                    </div>
                    <div class="metric-secondary">
                      <span>IOPS: {{ formatNumber(disk.seq_read_iops) }}</span>
                      <span>延迟: {{ formatLatency(disk.seq_read_latency_us) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
              <el-col :span="12">
                <div class="perf-card seq-write">
                  <div class="perf-title">顺序写 (1M)</div>
                  <div class="perf-metrics">
                    <div class="metric">
                      <span class="metric-value">{{ formatNumber(disk.seq_write_bw_mbps) }}</span>
                      <span class="metric-unit">MB/s</span>
                    </div>
                    <div class="metric-secondary">
                      <span>IOPS: {{ formatNumber(disk.seq_write_iops) }}</span>
                      <span>延迟: {{ formatLatency(disk.seq_write_latency_us) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>

            <el-row :gutter="20" style="margin-top: 15px;">
              <!-- 随机读写 -->
              <el-col :span="12">
                <div class="perf-card rand-read">
                  <div class="perf-title">随机读 (4K)</div>
                  <div class="perf-metrics">
                    <div class="metric">
                      <span class="metric-value">{{ formatNumber(disk.rand_read_iops) }}</span>
                      <span class="metric-unit">IOPS</span>
                    </div>
                    <div class="metric-secondary">
                      <span>带宽: {{ formatNumber(disk.rand_read_bw_mbps) }} MB/s</span>
                      <span>延迟: {{ formatLatency(disk.rand_read_latency_us) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
              <el-col :span="12">
                <div class="perf-card rand-write">
                  <div class="perf-title">随机写 (4K)</div>
                  <div class="perf-metrics">
                    <div class="metric">
                      <span class="metric-value">{{ formatNumber(disk.rand_write_iops) }}</span>
                      <span class="metric-unit">IOPS</span>
                    </div>
                    <div class="metric-secondary">
                      <span>带宽: {{ formatNumber(disk.rand_write_bw_mbps) }} MB/s</span>
                      <span>延迟: {{ formatLatency(disk.rand_write_latency_us) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>

            <el-row :gutter="20" style="margin-top: 15px;">
              <!-- 混合读写 -->
              <el-col :span="12">
                <div class="perf-card mixed">
                  <div class="perf-title">混合随机读写 (70R/30W, 4K)</div>
                  <div class="perf-metrics">
                    <div class="metric">
                      <span class="metric-value">{{ formatNumber(disk.mixed_iops) }}</span>
                      <span class="metric-unit">IOPS</span>
                    </div>
                    <div class="metric-secondary">
                      <span>带宽: {{ formatNumber(disk.mixed_bw_mbps) }} MB/s</span>
                      <span>延迟: {{ formatLatency(disk.mixed_latency_us) }}</span>
                    </div>
                  </div>
                </div>
              </el-col>
              <el-col :span="12">
                <div class="perf-card test-info">
                  <div class="perf-title">测试信息</div>
                  <div class="perf-metrics">
                    <div class="info-item">
                      <span class="info-label">测试路径:</span>
                      <span class="info-value">{{ disk.test_path }}</span>
                    </div>
                    <div class="info-item">
                      <span class="info-label">测试大小:</span>
                      <span class="info-value">{{ disk.test_size }}</span>
                    </div>
                    <div class="info-item">
                      <span class="info-label">测试耗时:</span>
                      <span class="info-value">{{ disk.duration }} 秒</span>
                    </div>
                  </div>
                </div>
              </el-col>
            </el-row>
          </el-card>
        </div>
      </div>

      <div v-else class="benchmark-loading">
        <el-empty description="暂无测试结果" />
      </div>

      <template #footer>
        <el-button @click="benchmarkDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="copyBenchmarkResult">复制JSON</el-button>
      </template>
    </el-dialog>
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
  Document,
  Monitor,
  Odometer,
  Position
} from '@element-plus/icons-vue'
import api from '@/api'
import dayjs from 'dayjs'

const router = useRouter()

const tableRef = ref()
const clients = ref([])
const loading = ref(false)
const totalConnections = ref(0)
const messagesSent = ref(0)
const selectedClients = ref([])

// 分页相关
const currentPage = ref(1)
const pageSize = ref(100)
const totalClients = ref(0)

// 硬件信息相关
const hardwareDialogVisible = ref(false)
const hardwareInfo = ref(null)
const hardwareLoading = ref({})
const currentClientId = ref('')

// 磁盘测试相关
const benchmarkDialogVisible = ref(false)
const benchmarkResult = ref(null)
const benchmarkLoading = ref({})
const currentBenchmarkClientId = ref('')
const benchmarkConcurrent = ref(true) // 默认并发测试

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
    const offset = (currentPage.value - 1) * pageSize.value
    const res = await api.getClients({
      offset: offset,
      limit: pageSize.value
    })
    clients.value = res.clients || []
    totalClients.value = res.total || 0
    totalConnections.value = res.total || 0
  } catch (error) {
    ElMessage.error('加载客户端列表失败')
  } finally {
    loading.value = false
  }
}

// 处理每页条数变化
function handleSizeChange(size) {
  pageSize.value = size
  currentPage.value = 1 // 重置到第一页
  loadClients()
}

// 处理页码变化
function handlePageChange(page) {
  currentPage.value = page
  loadClients()
}

// 处理选择变化
function handleSelectionChange(selection) {
  selectedClients.value = selection
}

// 全选所有客户端
function selectAllClients() {
  if (tableRef.value) {
    clients.value.forEach(row => {
      tableRef.value.toggleRowSelection(row, true)
    })
  }
}

// 批量下发命令
function batchSendCommand() {
  if (selectedClients.value.length === 0) {
    ElMessage.warning('请先选择客户端')
    return
  }

  // 将选中的客户端 ID 传递给命令页面
  const clientIds = selectedClients.value.map(c => c.client_id).join('\n')
  router.push({
    path: '/command',
    query: { client_ids: clientIds }
  })
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

// 获取硬件信息
async function getHardwareInfo(clientId) {
  hardwareLoading.value[clientId] = true
  currentClientId.value = clientId

  try {
    const res = await api.sendCommand({
      client_id: clientId,
      command_type: 'hardware.info',
      payload: {},
      timeout: 30
    })

    if (res.success && res.command_id) {
      // 等待命令执行完成
      await pollCommandResult(res.command_id)
    } else {
      ElMessage.error(res.message || '发送命令失败')
    }
  } catch (error) {
    ElMessage.error('获取硬件信息失败: ' + (error.message || '未知错误'))
  } finally {
    hardwareLoading.value[clientId] = false
  }
}

// 轮询命令结果
async function pollCommandResult(commandId) {
  const maxAttempts = 30
  const interval = 1000

  for (let i = 0; i < maxAttempts; i++) {
    try {
      const res = await api.getCommand(commandId)

      if (res.success && res.command) {
        const cmd = res.command

        if (cmd.status === 'completed') {
          hardwareInfo.value = cmd.result
          hardwareDialogVisible.value = true
          return
        } else if (cmd.status === 'failed' || cmd.status === 'timeout') {
          ElMessage.error(cmd.error || '命令执行失败')
          return
        }
      }
    } catch (error) {
      // 继续轮询
    }

    await new Promise(resolve => setTimeout(resolve, interval))
  }

  ElMessage.error('获取硬件信息超时')
}

// 格式化运行时间
function formatUptime(seconds) {
  if (!seconds) return '-'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) {
    return `${days}天 ${hours}小时 ${minutes}分钟`
  } else if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`
  } else {
    return `${minutes}分钟`
  }
}

// 复制硬件信息 JSON
function copyHardwareInfo() {
  if (!hardwareInfo.value) return

  const text = JSON.stringify(hardwareInfo.value, null, 2)
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

// 执行磁盘测试
async function runDiskBenchmark(clientId) {
  benchmarkLoading.value[clientId] = true
  currentBenchmarkClientId.value = clientId

  try {
    ElMessage.info('磁盘测试已开始，预计需要几分钟时间...')

    const res = await api.sendCommand({
      client_id: clientId,
      command_type: 'disk.benchmark',
      payload: {
        test_size: '1G',
        runtime: 30,
        concurrent: benchmarkConcurrent.value
      },
      timeout: 600 // 10分钟超时
    })

    if (res.success && res.command_id) {
      // 等待命令执行完成（长时间轮询）
      await pollBenchmarkResult(res.command_id)
    } else {
      ElMessage.error(res.message || '发送命令失败')
    }
  } catch (error) {
    ElMessage.error('执行磁盘测试失败: ' + (error.message || '未知错误'))
  } finally {
    benchmarkLoading.value[clientId] = false
  }
}

// 轮询磁盘测试结果（较长超时）
async function pollBenchmarkResult(commandId) {
  const maxAttempts = 600 // 最多等待10分钟
  const interval = 1000

  for (let i = 0; i < maxAttempts; i++) {
    try {
      const res = await api.getCommand(commandId)

      if (res.success && res.command) {
        const cmd = res.command

        if (cmd.status === 'completed') {
          benchmarkResult.value = cmd.result
          benchmarkDialogVisible.value = true
          ElMessage.success('磁盘测试完成')
          return
        } else if (cmd.status === 'failed' || cmd.status === 'timeout') {
          ElMessage.error(cmd.error || '磁盘测试失败')
          return
        }
      }
    } catch (error) {
      // 继续轮询
    }

    await new Promise(resolve => setTimeout(resolve, interval))
  }

  ElMessage.error('磁盘测试超时')
}

// 格式化数字
function formatNumber(num) {
  if (num === undefined || num === null) return '-'
  return num.toFixed(2)
}

// 格式化延迟（μs转换为合适单位）
function formatLatency(us) {
  if (us === undefined || us === null) return '-'
  if (us < 1000) {
    return us.toFixed(2) + ' μs'
  } else if (us < 1000000) {
    return (us / 1000).toFixed(2) + ' ms'
  } else {
    return (us / 1000000).toFixed(2) + ' s'
  }
}

// 复制磁盘测试结果 JSON
function copyBenchmarkResult() {
  if (!benchmarkResult.value) return

  const text = JSON.stringify(benchmarkResult.value, null, 2)
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
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

/* 分页样式 */
.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 硬件信息对话框样式 */
.hardware-info {
  max-height: 70vh;
  overflow-y: auto;
}

.info-section {
  margin-bottom: 15px;
}

.info-section:last-child {
  margin-bottom: 0;
}

.section-title {
  font-weight: bold;
  color: #303133;
}

.sub-table {
  margin-top: 10px;
}

.ipv6-text {
  font-size: 12px;
  word-break: break-all;
}

/* 磁盘测试对话框样式 */
.benchmark-info {
  max-height: 70vh;
  overflow-y: auto;
}

.benchmark-summary {
  margin-bottom: 15px;
}

.disk-result {
  margin-bottom: 15px;
}

.disk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.perf-card {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 15px;
  height: 100%;
}

.perf-card.seq-read {
  background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%);
}

.perf-card.seq-write {
  background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%);
}

.perf-card.rand-read {
  background: linear-gradient(135deg, #fff3e0 0%, #ffe0b2 100%);
}

.perf-card.rand-write {
  background: linear-gradient(135deg, #fce4ec 0%, #f8bbd9 100%);
}

.perf-card.mixed {
  background: linear-gradient(135deg, #f3e5f5 0%, #e1bee7 100%);
}

.perf-card.test-info {
  background: linear-gradient(135deg, #eceff1 0%, #cfd8dc 100%);
}

.perf-title {
  font-size: 14px;
  font-weight: bold;
  color: #606266;
  margin-bottom: 10px;
}

.perf-metrics {
  text-align: center;
}

.metric {
  margin-bottom: 8px;
}

.metric-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.metric-unit {
  font-size: 14px;
  color: #606266;
  margin-left: 5px;
}

.metric-secondary {
  display: flex;
  justify-content: space-around;
  font-size: 12px;
  color: #909399;
}

.info-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 13px;
}

.info-label {
  color: #606266;
}

.info-value {
  color: #303133;
  font-weight: 500;
  word-break: break-all;
  text-align: right;
  max-width: 200px;
}

.benchmark-loading {
  padding: 40px 0;
}

.benchmark-options {
  text-align: center;
}

.benchmark-options .option-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
