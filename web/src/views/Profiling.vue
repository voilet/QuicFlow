<template>
  <div class="profiling-page">
    <el-card class="header-card">
      <div class="header-content">
        <div class="title-section">
          <h2>服务端性能分析</h2>
          <p class="subtitle">CPU、内存、Goroutine 性能分析与火焰图可视化</p>
        </div>
        <div class="actions">
          <!-- 标准采集模式 -->
          <el-button type="primary" :icon="VideoCamera" @click="showStandardCPUDialog">
            CPU 采集 (标准)
          </el-button>
          <el-button type="success" :icon="Odometer" @click="captureStandardHeap">
            堆内存 (标准)
          </el-button>
          <el-button type="warning" :icon="Connection" @click="captureStandardGoroutine">
            Goroutine (标准)
          </el-button>
          <el-divider direction="vertical" />
          <!-- 数据库存储模式 -->
          <el-button type="info" :icon="FolderOpened" @click="showCPUDialog">
            CPU 采集 (保存)
          </el-button>
          <el-button :icon="Delete" @click="showCleanupDialog">
            清理
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 实时采集状态 -->
    <el-card v-if="liveCapturing.active" class="capturing-card">
      <div class="capturing-content">
        <el-progress
          :percentage="liveCapturing.progress"
          :status="liveCapturing.status"
          :stroke-width="20"
        />
        <div class="capturing-info">
          <span>{{ liveCapturing.message }}</span>
          <el-button type="danger" size="small" @click="cancelLiveCapture">
            取消采集
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <el-statistic title="总采集数" :value="stats.total" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card stat-cpu">
          <el-statistic title="CPU 采集" :value="stats.cpu" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card stat-memory">
          <el-statistic title="内存采集" :value="stats.memory" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card stat-goroutine">
          <el-statistic title="Goroutine 采集" :value="stats.goroutine" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 采集列表 -->
    <el-card class="table-card">
      <template #header>
        <div class="table-header">
          <span>采集历史</span>
          <div class="filter-controls">
            <el-select v-model="filterType" placeholder="类型筛选" clearable @change="loadProfiles">
              <el-option label="全部类型" value="" />
              <el-option label="CPU" value="cpu" />
              <el-option label="内存" value="memory" />
              <el-option label="Goroutine" value="goroutine" />
            </el-select>
            <el-select v-model="filterStatus" placeholder="状态筛选" clearable @change="loadProfiles">
              <el-option label="全部状态" value="" />
              <el-option label="已完成" value="completed" />
              <el-option label="运行中" value="running" />
              <el-option label="失败" value="failed" />
            </el-select>
            <el-button :icon="Refresh" @click="loadProfiles">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :data="profiles" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" width="180" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.type === 'cpu'" type="primary">CPU</el-tag>
            <el-tag v-else-if="row.type === 'memory'" type="success">内存</el-tag>
            <el-tag v-else-if="row.type === 'goroutine'" type="warning">Goroutine</el-tag>
            <el-tag v-else type="info">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'completed'" type="success">已完成</el-tag>
            <el-tag v-else-if="row.status === 'running'" type="warning">运行中</el-tag>
            <el-tag v-else-if="row.status === 'failed'" type="danger">失败</el-tag>
            <el-tag v-else type="info">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="时长" width="80">
          <template #default="{ row }">
            {{ row.duration > 0 ? `${row.duration}s` : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" width="100">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_by" label="创建者" width="100" />
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'completed'"
              type="primary"
              size="small"
              :icon="View"
              @click="viewProfile(row)"
            >
              查看
            </el-button>
            <el-button
              v-if="row.status === 'completed'"
              type="success"
              size="small"
              :icon="TrendCharts"
              @click="analyzeProfile(row)"
            >
              分析
            </el-button>
            <el-button
              v-if="row.status === 'running'"
              type="warning"
              size="small"
              disabled
            >
              采集中...
            </el-button>
            <el-button
              type="danger"
              size="small"
              :icon="Delete"
              @click="deleteProfile(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadProfiles"
        @current-change="loadProfiles"
      />
    </el-card>

    <!-- 标准采集对话框 -->
    <el-dialog v-model="stdCPUDialogVisible" title="CPU 性能采集 (标准 pprof)" width="500px">
      <el-form :model="stdCPUForm" label-width="80px">
        <el-form-item label="时长">
          <el-slider v-model="stdCPUForm.duration" :min="5" :max="300" :marks="durationMarks" show-input />
          <div class="hint">采集时长：{{ stdCPUForm.duration }} 秒</div>
        </el-form-item>
        <el-alert type="info" :closable="false">
          此模式使用标准 pprof 端点实时采集服务端性能数据，兼容 go tool pprof。
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="stdCPUDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="stdCPUSubmitting" @click="startStandardCPU">
          开始采集
        </el-button>
      </template>
    </el-dialog>

    <!-- 数据库采集对话框 -->
    <el-dialog v-model="cpuDialogVisible" title="CPU 性能采集 (保存到数据库)" width="500px">
      <el-form :model="cpuForm" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="cpuForm.name" placeholder="输入采集名称" />
        </el-form-item>
        <el-form-item label="时长">
          <el-slider v-model="cpuForm.duration" :min="5" :max="300" :marks="durationMarks" show-input />
          <div class="hint">采集时长：{{ cpuForm.duration }} 秒</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cpuDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="cpuSubmitting" @click="startCPUProfile">
          开始采集
        </el-button>
      </template>
    </el-dialog>

    <!-- 清理对话框 -->
    <el-dialog v-model="cleanupDialogVisible" title="清理旧采集" width="400px">
      <el-form :model="cleanupForm" label-width="100px">
        <el-form-item label="保留天数">
          <el-input-number v-model="cleanupForm.days" :min="1" :max="365" />
        </el-form-item>
        <el-alert type="warning" :closable="false">
          将删除 {{ cleanupForm.days }} 天前的所有采集记录和文件，此操作不可恢复！
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="cleanupDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="cleanupSubmitting" @click="cleanupProfiles">
          确认清理
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看详情对话框 -->
    <el-dialog v-model="detailDialogVisible" :title="detailTitle" width="90%" top="5vh" @close="closeDetailDialog">
      <div v-if="detailProfile" class="profile-detail">
        <el-tabs v-model="activeTab" class="detail-tabs">
          <!-- 基本信息 -->
          <el-tab-pane label="基本信息" name="info">
            <el-descriptions :column="3" border>
              <el-descriptions-item label="名称">{{ detailProfile.name }}</el-descriptions-item>
              <el-descriptions-item label="类型">
                <el-tag v-if="detailProfile.type === 'cpu'" type="primary">CPU</el-tag>
                <el-tag v-else-if="detailProfile.type === 'memory'" type="success">内存</el-tag>
                <el-tag v-else-if="detailProfile.type === 'goroutine'" type="warning">Goroutine</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="状态">{{ detailProfile.status }}</el-descriptions-item>
              <el-descriptions-item label="时长">{{ detailProfile.duration > 0 ? `${detailProfile.duration}s` : '-' }}</el-descriptions-item>
              <el-descriptions-item label="文件大小">{{ formatFileSize(detailProfile.file_size) }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ formatTime(detailProfile.created_at) }}</el-descriptions-item>
            </el-descriptions>
            <div class="download-section">
              <el-button type="primary" @click="downloadProfileFile">
                <el-icon><Download /></el-icon> 下载 Profile 文件
              </el-button>
              <el-button @click="openWithPprofTool">
                <el-icon><Link /></el-icon> 使用 go tool pprof 分析
              </el-button>
            </div>
          </el-tab-pane>

          <!-- 火焰图 -->
          <el-tab-pane label="火焰图" name="flamegraph">
            <div class="flamegraph-section">
              <div class="section-header">
                <h3>火焰图可视化</h3>
                <el-space>
                  <el-button v-if="!flameGraphSvgData && !flameGraphLoading" type="primary" @click="generateFlameGraph" :loading="flameGraphLoading">
                    生成火焰图
                  </el-button>
                  <el-select v-model="flameGraphType" style="width: 150px" @change="regenerateFlameGraph">
                    <el-option label="火焰图" value="flame" />
                    <el-option label="冰柱图" value="icicle" />
                  </el-select>
                  <el-button v-if="flameGraphSvgData" @click="openFlameGraphNewTab">
                    <el-icon><FolderOpened /></el-icon> 新窗口打开
                  </el-button>
                </el-space>
              </div>

              <div v-if="flameGraphLoading" class="flamegraph-loading">
                <el-icon class="is-loading"><Loading /></el-icon>
                <span>正在生成火焰图...</span>
              </div>

              <!-- SVG 火焰图显示 -->
              <div v-show="flameGraphSvgData && !flameGraphLoading" class="flamegraph-container">
                <div
                  class="flamegraph-svg"
                  v-html="flameGraphSvgData"
                ></div>
              </div>

              <div v-if="!flameGraphLoading && !flameGraphSvgData" class="flamegraph-placeholder">
                <el-empty description="点击生成火焰图按钮查看可视化" />
              </div>

              <!-- 火焰图图例 -->
              <div v-if="flameGraphSvgData" class="flamegraph-legend">
                <div class="legend-item">
                  <div class="legend-color warm"></div>
                  <span>暖色系 (CPU 热点)</span>
                </div>
                <div class="legend-item">
                  <div class="legend-color cool"></div>
                  <span>冷色系 (内存)</span>
                </div>
                <p class="legend-hint">鼠标悬停查看函数名和占比，滚轮缩放，拖拽平移</p>
              </div>
            </div>
          </el-tab-pane>

          <!-- pprof Web 视图 -->
          <el-tab-pane label="pprof Web" name="pprofweb">
            <div class="pprofweb-section">
              <div class="section-header">
                <h3>go tool pprof Web 视图</h3>
                <el-space>
                  <el-select v-model="pprofView" style="width: 150px" @change="changePprofView">
                    <el-option label="Top 函数" value="top" />
                    <el-option label="调用图" value="graph" />
                    <el-option label="源码" value="source" />
                    <el-option label="反汇编" value="disasm" />
                  </el-select>
                  <el-button @click="refreshPprofView" :loading="pprofLoading">
                    <el-icon><Refresh /></el-icon> 刷新
                  </el-button>
                </el-space>
              </div>

              <div v-if="pprofLoading" class="loading-section">
                <el-icon class="is-loading"><Loading /></el-icon>
                <span>正在分析...</span>
              </div>

              <!-- Top 视图 -->
              <div v-else-if="pprofView === 'top' && pprofData" class="pprof-top-view">
                <el-table :data="pprofData.top_functions" size="small" max-height="500" stripe>
                  <el-table-column type="index" label="#" width="60" />
                  <el-table-column prop="name" label="函数" show-overflow-tooltip min-width="200">
                    <template #default="{ row }">
                      <el-link @click="showFunctionDetail(row)" type="primary">
                        {{ row.name }}
                      </el-link>
                    </template>
                  </el-table-column>
                  <el-table-column prop="flat" label="自身 (Flat)" width="120">
                    <template #default="{ row }">
                      {{ formatValue(row.flat) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="flat_percent" label="自身%" width="100">
                    <template #default="{ row }">
                      {{ row.flat_percent?.toFixed(2) }}%
                    </template>
                  </el-table-column>
                  <el-table-column prop="cum" label="累计 (Cum)" width="120">
                    <template #default="{ row }">
                      {{ formatValue(row.cum) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="cum_percent" label="累计%" width="100">
                    <template #default="{ row }">
                      {{ row.cum_percent?.toFixed(2) }}%
                    </template>
                  </el-table-column>
                  <el-table-column prop="call_count" label="调用次数" width="100" />
                </el-table>
              </div>

              <!-- 图形视图占位 -->
              <div v-else-if="pprofView === 'graph'" class="pprof-graph-view">
                <div class="graph-placeholder">
                  <el-icon><TrendCharts /></el-icon>
                  <p>调用图可视化</p>
                  <p class="hint">请使用火焰图标签页查看交互式调用图</p>
                  <el-button type="primary" @click="activeTab = 'flamegraph'">
                    查看火焰图
                  </el-button>
                </div>
              </div>

              <!-- 源码视图占位 -->
              <div v-else-if="pprofView === 'source'" class="pprof-source-view">
                <div class="source-placeholder">
                  <el-icon><Document /></el-icon>
                  <p>源码视图</p>
                  <p class="hint">该功能需要在服务端启用源码映射</p>
                </div>
              </div>

              <!-- 反汇编视图占位 -->
              <div v-else-if="pprofView === 'disasm'" class="pprof-disasm-view">
                <div class="disasm-placeholder">
                  <el-icon><Monitor /></el-icon>
                  <p>反汇编视图</p>
                  <p class="hint">该功能需要在服务端启用符号表</p>
                </div>
              </div>

              <el-empty v-else description="请先选择一个采集记录" />
            </div>
          </el-tab-pane>

          <!-- Top 函数 -->
          <el-tab-pane label="Top 函数" name="top">
            <div v-if="topFunctionsLoading" class="loading-section">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>正在分析...</span>
            </div>
            <div v-else-if="topFunctions.length > 0" class="top-functions-section">
              <el-table :data="topFunctions" size="small" max-height="500">
                <el-table-column type="index" label="#" width="60" />
                <el-table-column prop="name" label="函数" show-overflow-tooltip />
                <el-table-column prop="flat" label="自身 (Flat)" width="120">
                  <template #default="{ row }">
                    {{ formatValue(row.flat) }}
                  </template>
                </el-table-column>
                <el-table-column prop="cum" label="累计 (Cum)" width="120">
                  <template #default="{ row }">
                    {{ formatValue(row.cum) }}
                  </template>
                </el-table-column>
                <el-table-column prop="percentage" label="占比" width="100">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="row.percentage"
                      :color="getPercentageColor(row.percentage)"
                      :show-text="true"
                    />
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else description="暂无数据" />
          </el-tab-pane>

          <!-- 分析报告 -->
          <el-tab-pane label="分析报告" name="analysis">
            <div v-if="analysisLoading" class="loading-section">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>正在分析...</span>
            </div>
            <div v-else-if="analysisReport" class="analysis-section">
              <!-- 摘要 -->
              <el-card class="summary-card">
                <template #header>
                  <span>分析摘要</span>
                </template>
                <el-row :gutter="16">
                  <el-col :span="6">
                    <el-statistic title="总样本数" :value="analysisReport.summary.total_samples" />
                  </el-col>
                  <el-col :span="6">
                    <el-statistic title="问题数" :value="analysisReport.summary.issue_count">
                      <template #suffix>
                        <el-tag v-if="analysisReport.summary.critical_count > 0" type="danger" size="small">
                          {{ analysisReport.summary.critical_count }} 严重
                        </el-tag>
                        <el-tag v-else-if="analysisReport.summary.high_count > 0" type="warning" size="small">
                          {{ analysisReport.summary.high_count }} 高危
                        </el-tag>
                      </template>
                    </el-statistic>
                  </el-col>
                  <el-col :span="12">
                    <div class="recommendation">
                      <strong>建议：</strong>{{ analysisReport.summary.recommendations }}
                    </div>
                  </el-col>
                </el-row>
              </el-card>

              <!-- 问题列表 -->
              <el-card v-if="analysisReport.issues.length > 0" class="issues-card">
                <template #header>
                  <span>发现的问题</span>
                </template>
                <el-timeline>
                  <el-timeline-item
                    v-for="issue in analysisReport.issues"
                    :key="issue.id"
                    :type="getIssueType(issue.severity)"
                    placement="top"
                  >
                    <el-card>
                      <div class="issue-header">
                        <el-tag :type="getIssueTagType(issue.severity)" size="small">
                          {{ issue.severity }}
                        </el-tag>
                        <strong>{{ issue.title }}</strong>
                      </div>
                      <p class="issue-desc">{{ issue.description }}</p>
                      <div v-if="issue.location" class="issue-location">
                        <strong>位置：</strong>
                        <el-tag size="small" @click="copyToClipboard(issue.location)">
                          {{ issue.location }}
                        </el-tag>
                      </div>
                      <el-alert type="info" :closable="false" class="issue-suggestion">
                        <strong>建议：</strong>{{ issue.suggestion }}
                      </el-alert>
                    </el-card>
                  </el-timeline-item>
                </el-timeline>
              </el-card>
              <el-empty v-else description="未发现明显问题" />
            </div>
            <el-empty v-else-if="!analysisLoading" description="点击分析按钮生成报告" />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  VideoCamera, Odometer, Connection, Delete, Refresh, View, TrendCharts,
  Download, Link, FolderOpened, Loading, Document, Monitor
} from '@element-plus/icons-vue'
import api from '@/api'

// 数据
const profiles = ref([])
const stats = reactive({
  total: 0,
  cpu: 0,
  memory: 0,
  goroutine: 0
})
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterType = ref('')
const filterStatus = ref('')

// 实时采集状态
const liveCapturing = reactive({
  active: false,
  progress: 0,
  status: '',
  message: '',
  controller: null
})

// 标准采集对话框
const stdCPUDialogVisible = ref(false)
const stdCPUSubmitting = ref(false)
const stdCPUForm = reactive({
  duration: 30
})

// 数据库采集对话框
const cpuDialogVisible = ref(false)
const cpuSubmitting = ref(false)
const cpuForm = reactive({
  name: '',
  duration: 30
})

const durationMarks = {
  5: '5s',
  30: '30s',
  60: '1m',
  120: '2m',
  300: '5m'
}

// 清理对话框
const cleanupDialogVisible = ref(false)
const cleanupSubmitting = ref(false)
const cleanupForm = reactive({
  days: 7
})

// 详情对话框
const detailDialogVisible = ref(false)
const detailProfile = ref(null)
const activeTab = ref('info')
const flameGraphSvgData = ref('') // 火焰图 SVG 数据
const flameGraphLoading = ref(false)
const flameGraphType = ref('flame') // flame 或 icicle
const analysisReport = ref(null)
const analysisLoading = ref(false)
const topFunctions = ref([])
const topFunctionsLoading = ref(false)

// pprof Web 视图
const pprofView = ref('top')
const pprofData = ref(null)
const pprofLoading = ref(false)

// d3-flame-graph 相关 - 已移除，直接使用后端生成的 SVG

// 加载采集列表
const loadProfiles = async () => {
  loading.value = true
  try {
    const res = await api.getProfiles({
      type: filterType.value,
      status: filterStatus.value,
      page: currentPage.value,
      page_size: pageSize.value
    })
    if (res.success) {
      profiles.value = res.profiles
      total.value = res.total
      updateStats()
    }
  } catch (error) {
    ElMessage.error('加载采集列表失败')
  } finally {
    loading.value = false
  }
}

// 更新统计
const updateStats = () => {
  stats.total = profiles.value.length
  stats.cpu = profiles.value.filter(p => p.type === 'cpu').length
  stats.memory = profiles.value.filter(p => p.type === 'memory').length
  stats.goroutine = profiles.value.filter(p => p.type === 'goroutine').length
}

// 显示标准 CPU 采集对话框
const showStandardCPUDialog = () => {
  stdCPUForm.duration = 30
  stdCPUDialogVisible.value = true
}

// 开始标准 CPU 采集（实时）
const startStandardCPU = async () => {
  stdCPUSubmitting.value = true
  stdCPUDialogVisible.value = false

  // 使用 fetch 获取 pprof 数据
  const duration = stdCPUForm.duration

  try {
    // 启动采集状态
    liveCapturing.active = true
    liveCapturing.progress = 0
    liveCapturing.status = 'success'
    liveCapturing.message = `正在采集 CPU 数据... (0/${duration}s)`

    // 创建 AbortController 用于取消
    liveCapturing.controller = new AbortController()

    // 计算进度
    const progressInterval = setInterval(() => {
      if (liveCapturing.progress < 90) {
        liveCapturing.progress += 100 / (duration * 10)
        liveCapturing.message = `正在采集 CPU 数据... (${Math.floor(liveCapturing.progress / 100 * duration)}/${duration}s)`
      }
    }, 1000)

    // 发起请求
    const response = await fetch(`/debug/pprof/profile?seconds=${duration}`, {
      signal: liveCapturing.controller.signal
    })

    clearInterval(progressInterval)
    liveCapturing.progress = 100

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    // 保存 pprof 数据
    const blob = await response.blob()

    // 构建 FormData 上传到服务器
    const formData = new FormData()
    formData.append('file', blob, `cpu-std-${Date.now()}.prof`)
    formData.append('name', `cpu-std-${Date.now()}`)

    // 上传并保存到数据库（使用 CPU 类型）
    const saveRes = await api.uploadCPUProfile(formData)
    if (saveRes.success) {
      ElMessage.success('CPU 采集完成')
      loadProfiles()
    }

    // 显示火焰图
    await loadAndDisplayFlameGraph(blob)
  } catch (error) {
    if (error.name === 'AbortError') {
      ElMessage.info('采集已取消')
    } else {
      ElMessage.error(`采集失败: ${error.message}`)
    }
  } finally {
    stdCPUSubmitting.value = false
    liveCapturing.active = false
    liveCapturing.progress = 0
  }
}

// 取消实时采集
const cancelLiveCapture = () => {
  if (liveCapturing.controller) {
    liveCapturing.controller.abort()
    liveCapturing.active = false
  }
}

// 采集标准堆内存
const captureStandardHeap = async () => {
  ElMessage.info('正在采集堆内存...')
  try {
    const response = await fetch('/debug/pprof/heap')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const blob = await response.blob()
    const name = `heap-std-${Date.now()}`

    // 保存
    const saveRes = await api.captureMemoryProfile({ name })
    if (saveRes.success) {
      ElMessage.success('堆内存采集完成')
      loadProfiles()

      // 查看详情
      const profile = profiles.value.find(p => p.name === name)
      if (profile) {
        await viewProfile(profile)
        await loadAndDisplayFlameGraph(blob)
      }
    }
  } catch (error) {
    ElMessage.error(`采集失败: ${error.message}`)
  }
}

// 采集标准 Goroutine
const captureStandardGoroutine = async () => {
  ElMessage.info('正在采集 Goroutine...')
  try {
    const response = await fetch('/debug/pprof/goroutine')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const blob = await response.blob()
    const name = `goroutine-std-${Date.now()}`

    // 保存
    const saveRes = await api.captureGoroutineProfile({ name })
    if (saveRes.success) {
      ElMessage.success('Goroutine 采集完成')
      loadProfiles()

      // 查看详情
      const profile = profiles.value.find(p => p.name === name)
      if (profile) {
        await viewProfile(profile)
        await loadAndDisplayFlameGraph(blob)
      }
    }
  } catch (error) {
    ElMessage.error(`采集失败: ${error.message}`)
  }
}

// 加载并显示火焰图（从采集的 blob）
const loadAndDisplayFlameGraph = async (blob) => {
  // 首先上传到服务器，然后获取火焰图
  ElMessage.info('正在处理 profile 数据...')
  try {
    const formData = new FormData()
    formData.append('file', blob, `temp-${Date.now()}.prof`)
    formData.append('name', `temp-${Date.now()}`)

    const saveRes = await api.uploadCPUProfile(formData)
    if (saveRes.success && saveRes.profile) {
      // 切换到火焰图标签页
      activeTab.value = 'flamegraph'
      await generateFlameGraphFromProfile(saveRes.profile.id)
    }
  } catch (error) {
    ElMessage.error(`火焰图生成失败: ${error.message}`)
  }
}

// 生成火焰图（使用后端 API）
const generateFlameGraph = async () => {
  if (!detailProfile.value) return
  await generateFlameGraphFromProfile(detailProfile.value.id)
}

// 从 profile ID 生成火焰图
const generateFlameGraphFromProfile = async (profileId) => {
  flameGraphLoading.value = true
  try {
    console.log('生成火焰图, profileId:', profileId)

    // 调用后端生成火焰图
    const res = await api.generateFlameGraph(profileId)
    console.log('火焰图 API 响应:', res)

    if (res.success) {
      // 使用 API 端点获取 SVG，而不是直接使用本地文件路径
      const svgUrl = api.getFlameGraphUrl(profileId)
      console.log('使用 SVG URL:', svgUrl)
      await renderFlameGraphSVG(svgUrl)
    } else if (res.message) {
      ElMessage.error(`火焰图生成失败: ${res.message}`)
      // 显示错误提示
      flameGraphSvgData.value = `<div class="flamegraph-error">
        <p>火焰图生成失败</p>
        <p>${res.message}</p>
        <p style="margin-top: 16px; color: #666;">请尝试使用 pprof Web 标签页查看分析结果</p>
      </div>`
    } else {
      ElMessage.warning('火焰图生成失败，请稍后重试')
    }
  } catch (error) {
    console.error('火焰图生成错误:', error)
    ElMessage.error(`火焰图生成失败: ${error.message}`)
    // 显示错误提示
    flameGraphSvgData.value = `<div class="flamegraph-error">
      <p>火焰图生成失败</p>
      <p>${error.message}</p>
      <p style="margin-top: 16px; color: #666;">请尝试使用 pprof Web 标签页查看分析结果</p>
    </div>`
  } finally {
    flameGraphLoading.value = false
  }
}

// 渲染火焰图 SVG
const renderFlameGraphSVG = async (svgPath) => {
  try {
    console.log('加载 SVG, 路径:', svgPath)

    // 获取 SVG 内容
    const response = await fetch(svgPath)
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
    const svgText = await response.text()
    console.log('SVG 内容长度:', svgText.length)

    if (!svgText || svgText.trim().length === 0) {
      throw new Error('SVG 内容为空')
    }

    // 检查是否是有效的 SVG
    if (!svgText.includes('<svg') && !svgText.includes('<?xml')) {
      console.warn('响应不是 SVG 格式:', svgText.substring(0, 200))
      // 直接显示内容
      flameGraphSvgData.value = `<pre style="white-space: pre-wrap; background: #f5f7fa; padding: 16px; border-radius: 4px;">${svgText}</pre>`
      return
    }

    // 直接显示 SVG（后端已生成完整的交互式火焰图）
    flameGraphSvgData.value = svgText
    console.log('火焰图渲染成功')
  } catch (error) {
    console.error('SVG 加载失败:', error)
    flameGraphSvgData.value = `<div class="flamegraph-error">
      <p>火焰图加载失败</p>
      <p>${error.message}</p>
      <p style="margin-top: 16px; color: #666;">路径: ${svgPath}</p>
    </div>`
  }
}

// 清理火焰图
const clearFlameGraph = () => {
  flameGraphSvgData.value = ''
}

// 重新生成火焰图（切换类型）
const regenerateFlameGraph = async () => {
  if (!detailProfile.value) return
  clearFlameGraph()
  await generateFlameGraphFromProfile(detailProfile.value.id)
}

// 显示数据库采集对话框
const showCPUDialog = () => {
  cpuForm.name = `cpu-${Date.now()}`
  cpuForm.duration = 30
  cpuDialogVisible.value = true
}

// 数据库 CPU 采集
const startCPUProfile = async () => {
  if (!cpuForm.name) {
    ElMessage.warning('请输入采集名称')
    return
  }

  cpuSubmitting.value = true
  try {
    const res = await api.startCPUProfile({
      name: cpuForm.name,
      duration: cpuForm.duration
    })
    if (res.success) {
      ElMessage.success(`CPU 采集已启动，ID: ${res.profile_id}`)
      cpuDialogVisible.value = false
      loadProfiles()
      ElMessage.info(`采集将在 ${cpuForm.duration} 秒后完成，请稍后刷新查看`)
    }
  } catch (error) {
    ElMessage.error('启动 CPU 采集失败')
  } finally {
    cpuSubmitting.value = false
  }
}

// 查看采集详情
const viewProfile = async (profile) => {
  detailProfile.value = profile
  detailDialogVisible.value = true
  activeTab.value = 'info'
  clearFlameGraph()
  analysisReport.value = null
  topFunctions.value = []
  pprofData.value = null
}

// 分析采集
const analyzeProfile = async (profile) => {
  detailProfile.value = profile
  detailDialogVisible.value = true
  activeTab.value = 'analysis'
  clearFlameGraph()
  analysisReport.value = null
  topFunctions.value = []
  pprofData.value = null

  // 获取 Top 函数
  await loadTopFunctions(profile)

  // 生成分析报告
  await loadAnalysisReport(profile)

  // 同时加载 pprof 数据
  await loadPprofData(profile)
}

// 加载 pprof 数据
const loadPprofData = async (profile) => {
  pprofLoading.value = true
  try {
    const res = await api.analyzeProfile(profile.id)
    if (res.success && res.report) {
      pprofData.value = {
        top_functions: res.report.top_functions || [],
        summary: res.report.summary,
        issues: res.report.issues
      }
    }
  } catch (error) {
    console.error('加载 pprof 数据失败:', error)
  } finally {
    pprofLoading.value = false
  }
}

// 切换 pprof 视图
const changePprofView = async (view) => {
  if (!detailProfile.value) return
  pprofView.value = view

  if (view === 'top' && !pprofData.value) {
    await loadPprofData(detailProfile.value)
  }
}

// 刷新 pprof 视图
const refreshPprofView = async () => {
  if (!detailProfile.value) return
  await loadPprofData(detailProfile.value)
}

// 显示函数详情
const showFunctionDetail = (func) => {
  // 构建函数详情 HTML
  const flatPercent = func.flat_percent ? ` (${func.flat_percent.toFixed(2)}%)` : ''
  const cumPercent = func.cum_percent ? ` (${func.cum_percent.toFixed(2)}%)` : ''
  const callCountInfo = func.call_count !== undefined ? `\n调用次数: ${func.call_count}` : ''

  ElMessageBox.alert({
    title: '函数详情',
    message: `
      <div style="line-height: 1.8;">
        <p><strong>函数名:</strong></p>
        <code style="background: #f5f5f5; padding: 4px 8px; border-radius: 4px; word-break: break-all; display: block;">${func.name}</code>
        <p style="margin-top: 16px;"><strong>自身采样:</strong> ${formatValue(func.flat)}${flatPercent}</p>
        <p><strong>累计采样:</strong> ${formatValue(func.cum)}${cumPercent}</p>
        ${callCountInfo ? `<p><strong>${callCountInfo}</strong></p>` : ''}
      </div>
    `,
    dangerouslyUseHTMLString: true,
    confirmButtonText: '关闭'
  })
}

// 加载 Top 函数
const loadTopFunctions = async (profile) => {
  topFunctionsLoading.value = true
  try {
    // 从 API 获取分析报告，提取 Top 函数
    const res = await api.analyzeProfile(profile.id)
    if (res.success && res.report?.top_functions) {
      topFunctions.value = res.report.top_functions
    }
  } catch (error) {
    console.error('加载 Top 函数失败:', error)
  } finally {
    topFunctionsLoading.value = false
  }
}

// 加载分析报告
const loadAnalysisReport = async (profile) => {
  analysisLoading.value = true
  try {
    const res = await api.analyzeProfile(profile.id)
    if (res.success) {
      analysisReport.value = res.report
    }
  } catch (error) {
    console.error('分析失败:', error)
  } finally {
    analysisLoading.value = false
  }
}

// 删除采集
const deleteProfile = async (profile) => {
  try {
    await ElMessageBox.confirm(`确定要删除采集 "${profile.name}" 吗？此操作不可恢复！`, '确认删除', {
      type: 'warning'
    })

    await api.deleteProfile(profile.id)
    ElMessage.success('删除成功')
    loadProfiles()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 清理旧采集
const cleanupProfiles = async () => {
  cleanupSubmitting.value = true
  try {
    const res = await api.cleanupProfiles(cleanupForm.days)
    if (res.success) {
      ElMessage.success(`已清理 ${res.deleted_count} 个旧采集`)
      cleanupDialogVisible.value = false
      loadProfiles()
    }
  } catch (error) {
    ElMessage.error('清理失败')
  } finally {
    cleanupSubmitting.value = false
  }
}

// 显示清理对话框
const showCleanupDialog = () => {
  cleanupForm.days = 7
  cleanupDialogVisible.value = true
}

// 新窗口打开火焰图
const openFlameGraphNewTab = () => {
  if (detailProfile.value) {
    const svgUrl = api.getFlameGraphUrl(detailProfile.value.id)
    window.open(svgUrl, '_blank')
  }
}

// 下载 profile 文件
const downloadProfileFile = () => {
  if (!detailProfile.value) return

  const url = api.downloadProfile(detailProfile.value.id)
  const a = document.createElement('a')
  a.href = url
  a.download = `${detailProfile.value.type}-${detailProfile.value.id}.prof`
  a.click()
}

// 使用 go tool pprof 分析
const openWithPprofTool = () => {
  const url = window.location.origin
  ElMessageBox.alert({
    title: '使用 go tool pprof 分析',
    message: `
      <div style="line-height: 2;">
        <p><strong>1. 安装 Go 工具链:</strong></p>
        <code style="background: #f5f5f5; padding: 4px 8px; border-radius: 4px; display: block; white-space: pre-wrap;">go install golang.org/x/perf/cmd@latest</code>

        <p style="margin-top: 16px;"><strong>2. CPU 分析（30秒）:</strong></p>
        <code style="background: #f5f5f5; padding: 4px 8px; border-radius: 4px; display: block; white-space: pre-wrap;">go tool pprof ${url}/debug/pprof/profile?seconds=30</code>

        <p style="margin-top: 16px;"><strong>3. Web 界面分析:</strong></p>
        <code style="background: #f5f5f5; padding: 4px 8px; border-radius: 4px; display: block; white-space: pre-wrap;">go tool pprof -http=:8080 ${url}/debug/pprof/profile?seconds=30</code>

        <p style="margin-top: 16px;"><strong>4. Top 函数:</strong></p>
        <code style="background: #f5f5f5; padding: 4px 8px; border-radius: 4px; display: block; white-space: pre-wrap;">go tool pprof -top -nodecount=10 ${url}/debug/pprof/heap</code>

        <p style="margin-top: 16px;"><strong>5. 生成火焰图:</strong></p>
        <code style="background: #f5f5f5; padding: 4px 8px; border-radius: 4px; display: block; white-space: pre-wrap;">go tool pprof -raw -output=svg ${url}/debug/pprof/profile?seconds=30 > flamegraph.svg</code>
      </div>
    `,
    dangerouslyUseHTMLString: true,
    confirmButtonText: '关闭'
  })
}

// 格式化文件大小
const formatFileSize = (bytes) => {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

// 格式化时间
const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN')
}

// 格式化数值
const formatValue = (value) => {
  if (!value) return '0'
  if (value < 1024) return value + ' B'
  if (value < 1024 * 1024) return (value / 1024).toFixed(2) + ' KB'
  if (value < 1024 * 1024 * 1024) return (value / (1024 * 1024)).toFixed(2) + ' MB'
  return (value / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

// 获取进度条颜色
const getPercentageColor = (percentage) => {
  if (percentage >= 20) return 'danger'
  if (percentage >= 10) return 'warning'
  return 'success'
}

// 获取问题类型
const getIssueType = (severity) => {
  switch (severity) {
    case 'critical': return 'danger'
    case 'high': return 'warning'
    case 'medium': return 'primary'
    default: return 'info'
  }
}

// 获取问题标签类型
const getIssueTagType = (severity) => {
  switch (severity) {
    case 'critical': return 'danger'
    case 'high': return 'warning'
    case 'medium': return 'primary'
    default: return 'info'
  }
}

// 复制到剪贴板
const copyToClipboard = (text) => {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制到剪贴板')
}

// 关闭详情对话框
const closeDetailDialog = () => {
  clearFlameGraph()
  analysisReport.value = null
  topFunctions.value = []
  pprofData.value = null
}

// 详情标题
const detailTitle = computed(() => {
  if (!detailProfile.value) return ''
  const typeMap = {
    cpu: 'CPU',
    memory: '内存',
    goroutine: 'Goroutine'
  }
  return `${typeMap[detailProfile.value.type] || ''} 采集详情 - ${detailProfile.value.name}`
})

onMounted(() => {
  loadProfiles()
})
</script>

<style scoped>
.profiling-page {
  padding: 20px;
}

.header-card {
  margin-bottom: 20px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-section h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
}

.subtitle {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.actions {
  display: flex;
  gap: 12px;
}

.capturing-card {
  margin-bottom: 20px;
}

.capturing-content {
  padding: 10px 0;
}

.capturing-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  text-align: center;
}

.stat-cpu :deep(.el-statistic__number) {
  color: #409eff;
}

.stat-memory :deep(.el-statistic__number) {
  color: #67c23a;
}

.stat-goroutine :deep(.el-statistic__number) {
  color: #e6a23c;
}

.table-card {
  min-height: 400px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-controls {
  display: flex;
  gap: 12px;
}

.hint {
  margin-top: 8px;
  color: #999;
  font-size: 12px;
}

.profile-detail {
  padding: 10px 0;
}

.detail-tabs {
  min-height: 400px;
}

.download-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #eee;
}

.flamegraph-section {
  padding: 16px 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header h3 {
  margin: 0;
  font-size: 16px;
}

.flamegraph-container {
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: auto;
  min-height: 300px;
  background: #fff;
  padding: 10px;
}

.flamegraph-loading {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px;
  color: #999;
}

.flamegraph-placeholder {
  padding: 40px;
}

.flamegraph-svg :deep(svg) {
  display: block;
  max-width: 100%;
}

.flamegraph-legend {
  margin-top: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-right: 20px;
}

.legend-color {
  width: 20px;
  height: 12px;
  border-radius: 2px;
}

.legend-color.warm {
  background: linear-gradient(to right, #ff6b6b, #ffd93d);
}

.legend-color.cool {
  background: linear-gradient(to right, #4facfe, #00f2fe);
}

.legend-hint {
  margin: 8px 0 0 0;
  font-size: 12px;
  color: #666;
}

.flamegraph-error {
  padding: 20px;
  text-align: center;
}

.flamegraph-error code {
  display: block;
  background: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  margin-top: 10px;
  text-align: left;
}

.summary-card,
.issues-card,
.top-functions-section {
  margin-bottom: 16px;
}

.recommendation {
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 14px;
}

.issue-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.issue-desc {
  margin: 8px 0;
  color: #666;
}

.issue-location {
  margin: 8px 0;
  font-size: 12px;
}

.issue-suggestion {
  margin-top: 8px;
}

:deep(.el-timeline-item__wrapper) {
  padding-left: 20px;
}

.loading-section {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px;
  color: #999;
}

/* ========== pprof Web 视图样式 ========== */
.pprofweb-section {
  padding: 16px 0;
}

.pprof-top-view {
  padding: 10px 0;
}

.pprof-graph-view,
.pprof-source-view,
.pprof-disasm-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  padding: 40px;
}

.graph-placeholder,
.source-placeholder,
.disasm-placeholder {
  text-align: center;
  color: #999;
}

.graph-placeholder .el-icon,
.source-placeholder .el-icon,
.disasm-placeholder .el-icon {
  font-size: 64px;
  margin-bottom: 16px;
  color: #ddd;
}

.graph-placeholder p,
.source-placeholder p,
.disasm-placeholder p {
  margin: 8px 0;
  font-size: 16px;
}

.hint {
  color: #999;
  font-size: 14px;
}
</style>
