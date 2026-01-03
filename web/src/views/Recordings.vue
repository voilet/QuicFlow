<template>
  <div class="recordings-container">
    <!-- 统计卡片 -->
    <div class="stats-row" v-if="stats">
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-icon video">
            <el-icon :size="30"><VideoCamera /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total_recordings }}</div>
            <div class="stat-label">总录像数</div>
          </div>
        </div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-icon storage">
            <el-icon :size="30"><FolderOpened /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatSize(stats.total_size_bytes) }}</div>
            <div class="stat-label">存储大小</div>
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
          <el-button type="primary" @click="fetchRecordings">
            <el-icon><Search /></el-icon>
            查询
          </el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 录像列表 -->
    <el-card class="recordings-card" shadow="hover">
      <template #header>
        <div class="card-header-title">
          <el-icon><VideoCamera /></el-icon>
          <span>录像列表</span>
        </div>
      </template>
      <el-table
        :data="recordings"
        v-loading="loading"
        stripe
        border
        style="width: 100%"
        :default-sort="{ prop: 'created_at', order: 'descending' }"
        table-layout="auto"
      >
        <el-table-column prop="created_at" label="录制时间" min-width="120">
          <template #default="scope">
            {{ formatTime(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="client_id" label="客户端 ID" min-width="120" />
        <el-table-column prop="username" label="用户名" min-width="100" />
        <el-table-column prop="duration" label="时长" min-width="80">
          <template #default="scope">
            {{ formatDuration(scope.row.duration) }}
          </template>
        </el-table-column>
        <el-table-column label="终端大小" min-width="100">
          <template #default="scope">
            {{ scope.row.width }}x{{ scope.row.height }}
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" min-width="100">
          <template #default="scope">
            {{ formatSize(scope.row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="200" fixed="right">
          <template #default="scope">
            <el-button type="primary" size="small" @click="playRecording(scope.row)">
              <el-icon><VideoPlay /></el-icon>
              播放
            </el-button>
            <el-button size="small" @click="downloadRecording(scope.row)">
              <el-icon><Download /></el-icon>
            </el-button>
            <el-popconfirm
              title="确定要删除这个录像吗？"
              @confirm="deleteRecording(scope.row)"
            >
              <template #reference>
                <el-button type="danger" size="small">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        class="pagination"
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </el-card>

    <!-- 播放器对话框 -->
    <el-dialog
      v-model="playerVisible"
      :title="playerTitle"
      width="95%"
      top="2vh"
      :close-on-click-modal="false"
      :fullscreen="isFullscreen"
      @close="stopPlayback"
      class="player-dialog"
    >
      <div class="player-container" :class="{ 'fullscreen': isFullscreen }">
        <!-- 信息栏 -->
        <div class="player-info" v-if="currentRecording">
          <div class="info-item">
            <span class="info-label">客户端:</span>
            <span class="info-value">{{ currentRecording.client_id }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">用户:</span>
            <span class="info-value">{{ currentRecording.username || 'root' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">时间:</span>
            <span class="info-value">{{ formatTime(currentRecording.created_at) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">终端:</span>
            <span class="info-value">{{ currentRecording.width }}x{{ currentRecording.height }}</span>
          </div>
        </div>

        <!-- 终端显示区域 -->
        <div class="terminal-wrapper" ref="terminalWrapperRef">
          <div v-if="!isLoaded" class="loading-overlay">
            <el-icon class="loading-icon"><Loading /></el-icon>
            <span>加载录像中...</span>
          </div>
          <div ref="terminalRef" class="terminal-display"></div>
        </div>

        <!-- 进度条 -->
        <div class="progress-container">
          <span class="time-display">{{ formatDuration(currentTime) }}</span>
          <el-slider
            v-model="progress"
            :max="100"
            :step="0.1"
            :disabled="!isLoaded"
            :show-tooltip="false"
            @change="seekTo"
            class="progress-slider"
          />
          <span class="time-display">{{ formatDuration(totalDuration) }}</span>
        </div>

        <!-- 控制栏 -->
        <div class="player-controls">
          <div class="controls-left">
            <el-button-group>
              <el-button @click="restart" :disabled="!isLoaded" size="default">
                <el-icon><RefreshLeft /></el-icon>
              </el-button>
              <el-button @click="skipBackward" :disabled="!isLoaded" size="default">
                <el-icon><DArrowLeft /></el-icon>
              </el-button>
              <el-button
                @click="togglePlayPause"
                :disabled="!isLoaded"
                type="primary"
                size="default"
                class="play-btn"
              >
                <el-icon :size="20">
                  <component :is="isPlaying ? 'VideoPause' : 'VideoPlay'" />
                </el-icon>
              </el-button>
              <el-button @click="skipForward" :disabled="!isLoaded" size="default">
                <el-icon><DArrowRight /></el-icon>
              </el-button>
            </el-button-group>
          </div>

          <div class="controls-center">
            <div class="speed-control">
              <span class="speed-label">速度</span>
              <el-button-group>
                <el-button
                  v-for="speed in speedOptions"
                  :key="speed"
                  :type="playbackSpeed === speed ? 'primary' : 'default'"
                  size="small"
                  @click="setSpeed(speed)"
                >
                  {{ speed }}x
                </el-button>
              </el-button-group>
            </div>
          </div>

          <div class="controls-right">
            <el-tooltip content="全屏" placement="top">
              <el-button @click="toggleFullscreen" size="default">
                <el-icon><FullScreen /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <!-- 事件数信息 -->
        <div class="events-info" v-if="isLoaded">
          <span>事件: {{ currentEventIndex }} / {{ events.length }}</span>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, nextTick, onUnmounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Search, Download, Delete, VideoPlay, VideoPause,
  RefreshLeft, DArrowLeft, DArrowRight, FullScreen, Loading
} from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import api, { request } from '../api'

const loading = ref(false)
const recordings = ref([])
const stats = ref(null)
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const dateRange = ref(null)

const playerVisible = ref(false)
const currentRecording = ref(null)
const isLoaded = ref(false)
const isPlaying = ref(false)
const playbackSpeed = ref(1)
const currentTime = ref(0)
const totalDuration = ref(0)
const progress = ref(0)
const terminalRef = ref(null)
const terminalWrapperRef = ref(null)
const isFullscreen = ref(false)
const currentEventIndex = ref(0)

const speedOptions = [0.5, 1, 2, 4, 8]

let terminal = null
let fitAddon = null
let events = []
let playbackTimer = null

const playerTitle = computed(() => {
  if (!currentRecording.value) return '录像回放'
  return `录像回放 - ${currentRecording.value.client_id}`
})

const filter = reactive({
  client_id: '',
  username: '',
  start_time: '',
  end_time: ''
})

const fetchRecordings = async () => {
  loading.value = true
  try {
    const params = {
      ...filter,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null) {
        delete params[key]
      }
    })

    const res = await api.getRecordings(params)
    if (res.success) {
      recordings.value = res.recordings || []
      total.value = res.count || 0
    }
  } catch (err) {
    ElMessage.error('获取录像列表失败: ' + err.message)
  } finally {
    loading.value = false
  }
}

const fetchStats = async () => {
  try {
    const res = await api.getRecordingStats()
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
  filter.start_time = ''
  filter.end_time = ''
  dateRange.value = null
  currentPage.value = 1
  fetchRecordings()
}

const handleSizeChange = () => {
  currentPage.value = 1
  fetchRecordings()
}

const handlePageChange = () => {
  fetchRecordings()
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const formatDuration = (seconds) => {
  if (!seconds || seconds < 0) return '0:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

const formatSize = (bytes) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024
    i++
  }
  return `${bytes.toFixed(1)} ${units[i]}`
}

const playRecording = async (recording) => {
  currentRecording.value = recording
  playerVisible.value = true
  isLoaded.value = false
  isPlaying.value = false
  currentTime.value = 0
  progress.value = 0
  events = []
  currentEventIndex.value = 0

  await nextTick()

  // Initialize terminal
  if (terminal) {
    terminal.dispose()
  }

  terminal = new Terminal({
    cols: recording.width || 80,
    rows: recording.height || 24,
    theme: {
      background: '#0d1117',
      foreground: '#c9d1d9',
      cursor: '#58a6ff',
      cursorAccent: '#0d1117',
      selection: 'rgba(56, 139, 253, 0.4)',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d4dd',
      brightWhite: '#f0f6fc',
    },
    fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", Consolas, Monaco, "Courier New", monospace',
    fontSize: 14,
    lineHeight: 1.2,
    cursorBlink: false,
    cursorStyle: 'block',
    scrollback: 10000,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(terminalRef.value)

  await nextTick()
  fitAddon.fit()

  // Load recording
  try {
    const res = await request.get(`/recordings/${recording.id}/download`, {
      responseType: 'text'
    })

    const lines = res.trim().split('\n')
    if (lines.length === 0) {
      ElMessage.error('录像文件为空')
      return
    }

    // Parse header
    const header = JSON.parse(lines[0])
    totalDuration.value = header.duration || 0

    // Parse events
    for (let i = 1; i < lines.length; i++) {
      try {
        const event = JSON.parse(lines[i])
        if (Array.isArray(event) && event.length >= 3) {
          events.push({
            time: event[0],
            type: event[1],
            data: event[2]
          })
        }
      } catch (e) {
        // Skip invalid lines
      }
    }

    // Calculate total duration from events if not in header
    if (events.length > 0 && !totalDuration.value) {
      totalDuration.value = events[events.length - 1].time
    }

    isLoaded.value = true
    ElMessage.success(`已加载 ${events.length} 个事件`)

    // Auto play
    startPlayback()

  } catch (err) {
    ElMessage.error('加载录像失败: ' + err.message)
  }
}

const startPlayback = () => {
  if (!isLoaded.value || events.length === 0) return

  isPlaying.value = true
  scheduleNextEvent()
}

const scheduleNextEvent = () => {
  if (!isPlaying.value || currentEventIndex.value >= events.length) {
    isPlaying.value = false
    return
  }

  const event = events[currentEventIndex.value]
  const prevTime = currentEventIndex.value > 0 ? events[currentEventIndex.value - 1].time : 0
  let delay = ((event.time - prevTime) * 1000) / playbackSpeed.value

  // Cap very long delays
  if (delay > 2000) {
    delay = 100
  }

  playbackTimer = setTimeout(() => {
    processEvent(event)
    currentEventIndex.value++
    currentTime.value = event.time
    progress.value = totalDuration.value > 0 ? (event.time / totalDuration.value) * 100 : 0

    scheduleNextEvent()
  }, Math.max(0, delay))
}

const processEvent = (event) => {
  if (!terminal) return

  switch (event.type) {
    case 'o': // output
      terminal.write(event.data)
      break
    case 'i': // input (optional display)
      break
    case 'r': // resize
      try {
        const size = JSON.parse(event.data)
        terminal.resize(size.cols, size.rows)
      } catch (e) {}
      break
  }
}

const togglePlayPause = () => {
  if (isPlaying.value) {
    pausePlayback()
  } else {
    resumePlayback()
  }
}

const pausePlayback = () => {
  isPlaying.value = false
  if (playbackTimer) {
    clearTimeout(playbackTimer)
    playbackTimer = null
  }
}

const resumePlayback = () => {
  if (currentEventIndex.value >= events.length) {
    restart()
  } else {
    startPlayback()
  }
}

const restart = () => {
  pausePlayback()
  currentEventIndex.value = 0
  currentTime.value = 0
  progress.value = 0
  if (terminal) {
    terminal.clear()
    terminal.reset()
  }
  startPlayback()
}

const seekTo = (value) => {
  pausePlayback()

  const targetTime = (value / 100) * totalDuration.value

  // Reset terminal
  if (terminal) {
    terminal.clear()
    terminal.reset()
  }

  // Find the event index for target time and replay all events up to that point
  currentEventIndex.value = 0
  for (let i = 0; i < events.length; i++) {
    if (events[i].time <= targetTime) {
      processEvent(events[i])
      currentEventIndex.value = i + 1
    } else {
      break
    }
  }

  currentTime.value = targetTime
}

const skipForward = () => {
  const newTime = Math.min(currentTime.value + 10, totalDuration.value)
  const newProgress = totalDuration.value > 0 ? (newTime / totalDuration.value) * 100 : 0
  seekTo(newProgress)
  if (isPlaying.value) {
    startPlayback()
  }
}

const skipBackward = () => {
  const newTime = Math.max(currentTime.value - 10, 0)
  const newProgress = totalDuration.value > 0 ? (newTime / totalDuration.value) * 100 : 0
  seekTo(newProgress)
  if (isPlaying.value) {
    startPlayback()
  }
}

const setSpeed = (speed) => {
  playbackSpeed.value = speed
  if (isPlaying.value) {
    pausePlayback()
    resumePlayback()
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  nextTick(() => {
    if (fitAddon) {
      fitAddon.fit()
    }
  })
}

const stopPlayback = () => {
  pausePlayback()
  isFullscreen.value = false
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  events = []
  currentEventIndex.value = 0
}

const downloadRecording = (recording) => {
  window.open(`/api/recordings/${recording.id}/download`, '_blank')
}

const deleteRecording = async (recording) => {
  try {
    const res = await request.delete(`/recordings/${recording.id}`)
    if (res.success) {
      ElMessage.success('删除成功')
      fetchRecordings()
      fetchStats()
    }
  } catch (err) {
    ElMessage.error('删除失败: ' + err.message)
  }
}

// Handle window resize
const handleResize = () => {
  if (fitAddon && playerVisible.value) {
    fitAddon.fit()
  }
}

onMounted(() => {
  fetchRecordings()
  fetchStats()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  stopPlayback()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.recordings-container {
  padding: 0;
  position: relative;
}

/* 统计卡片 */
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s ease;
  background: var(--tech-bg-card);
  backdrop-filter: blur(20px);
  border: 1px solid var(--tech-border-light);
  position: relative;
  overflow: hidden;
}

[data-theme="dark"] .stat-card {
  border: 1px solid var(--tech-border);
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
    rgba(0, 102, 255, 0.1),
    transparent
  );
  transition: left 0.5s ease;
}

[data-theme="dark"] .stat-card::before {
  background: linear-gradient(
    90deg,
    transparent,
    rgba(0, 255, 255, 0.1),
    transparent
  );
}

.stat-card:hover {
  transform: translateY(-4px);
  border-color: var(--tech-border-active);
  box-shadow: var(--tech-shadow-md);
}

[data-theme="dark"] .stat-card:hover {
  box-shadow: 
    0 0 20px rgba(0, 255, 255, 0.3),
    inset 0 0 20px rgba(0, 255, 255, 0.05);
}

.stat-card:hover::before {
  left: 100%;
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
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  transition: all 0.3s ease;
}

.stat-icon.video {
  background: linear-gradient(135deg, rgba(0, 102, 255, 0.15) 0%, rgba(0, 128, 255, 0.15) 100%);
  border: 1px solid var(--tech-primary);
  color: var(--tech-primary);
  box-shadow: 0 0 10px rgba(0, 102, 255, 0.2);
}

[data-theme="dark"] .stat-icon.video {
  background: linear-gradient(135deg, rgba(0, 255, 255, 0.2) 0%, rgba(0, 128, 255, 0.2) 100%);
  box-shadow: 0 0 15px rgba(0, 255, 255, 0.3);
}

.stat-icon.storage {
  background: linear-gradient(135deg, rgba(0, 170, 68, 0.15) 0%, rgba(0, 200, 0, 0.15) 100%);
  border: 1px solid var(--tech-secondary);
  color: var(--tech-secondary);
  box-shadow: 0 0 10px rgba(0, 170, 68, 0.2);
}

[data-theme="dark"] .stat-icon.storage {
  background: linear-gradient(135deg, rgba(0, 255, 0, 0.2) 0%, rgba(0, 200, 0, 0.2) 100%);
  box-shadow: 0 0 15px rgba(0, 255, 0, 0.3);
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--tech-text-primary);
  margin-bottom: 6px;
  font-family: var(--tech-font-heading);
  text-shadow: 0 0 5px rgba(0, 102, 255, 0.2);
}

[data-theme="dark"] .stat-value {
  text-shadow: 0 0 10px rgba(0, 255, 255, 0.3);
}

.stat-label {
  font-size: 14px;
  color: var(--tech-text-secondary);
  font-weight: 500;
}

.filter-card {
  margin-bottom: 24px;
  background: var(--tech-bg-card);
  backdrop-filter: blur(20px);
  border: 1px solid var(--tech-border-light);
  transition: all 0.3s ease;
}

[data-theme="dark"] .filter-card {
  border: 1px solid var(--tech-border);
}

.filter-card:hover {
  border-color: var(--tech-border-active);
  box-shadow: var(--tech-shadow-md);
}

.card-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--tech-text-primary);
  font-family: var(--tech-font-heading);
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

.recordings-card {
  margin-bottom: 24px;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  transition: all 0.3s ease;
}

.recordings-card:hover {
  box-shadow: var(--tech-shadow-md);
}

/* 表格美化 */
.recordings-card :deep(.el-table) {
  background: transparent;
  color: var(--tech-text-primary);
  border: none;
  font-size: 14px;
}

.recordings-card :deep(.el-table__header-wrapper) {
  background: transparent;
}

.recordings-card :deep(.el-table__header) {
  background: transparent;
}

.recordings-card :deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  font-weight: 600;
  padding: 12px;
  font-size: 14px;
  transition: all 0.3s ease;
}

.recordings-card :deep(.el-table th:hover) {
  background-color: var(--tech-bg-tertiary);
  color: var(--tech-text-primary);
}

.recordings-card :deep(.el-table td) {
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  padding: 12px;
  transition: all 0.2s ease;
}

.recordings-card :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: var(--tech-bg-tertiary);
}

.recordings-card :deep(.el-table__row) {
  transition: all 0.2s ease;
}

.recordings-card :deep(.el-table__row:hover) {
  background-color: var(--tech-bg-tertiary);
}

.recordings-card :deep(.el-table__row:hover td) {
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
}

.recordings-card :deep(.el-table__body tr) {
  border-bottom: 1px solid var(--tech-border-light);
}

[data-theme="dark"] .recordings-card :deep(.el-table__body tr) {
  border-bottom: 1px solid var(--tech-border);
}

.recordings-card :deep(.el-table__body tr:last-child) {
  border-bottom: none;
}

/* 操作按钮美化 */
.recordings-card :deep(.el-table .el-button) {
  border-radius: 6px;
  font-weight: 500;
  transition: all 0.3s ease;
  margin: 0 2px;
}

.recordings-card :deep(.el-table .el-button--primary) {
  background-color: var(--tech-primary);
  border-color: var(--tech-primary);
  color: #ffffff;
}

.recordings-card :deep(.el-table .el-button--primary:hover) {
  background-color: var(--tech-primary-light);
  border-color: var(--tech-primary-light);
}

.recordings-card :deep(.el-table .el-button:not(.el-button--primary):not(.el-button--danger)) {
  background-color: var(--tech-bg-secondary);
  border-color: var(--tech-border);
  color: var(--tech-text-secondary);
}

.recordings-card :deep(.el-table .el-button:not(.el-button--primary):not(.el-button--danger):hover) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
}

.recordings-card :deep(.el-table .el-button--danger) {
  background-color: var(--tech-danger);
  border-color: var(--tech-danger);
  color: #ffffff;
}

.recordings-card :deep(.el-table .el-button--danger:hover) {
  background-color: #F78989;
  border-color: #F78989;
}

/* 表格单元格内容美化 */
.recordings-card :deep(.el-table .cell) {
  padding: 0;
  line-height: 1.6;
}

.recordings-card :deep(.el-table td .cell) {
  color: var(--tech-text-primary);
}

.recordings-card :deep(.el-table td .cell code) {
  background: var(--tech-bg-secondary);
  border: 1px solid var(--tech-border-light);
  border-radius: 4px;
  padding: 2px 6px;
  font-family: var(--tech-font-mono);
  font-size: 12px;
  color: var(--tech-primary);
}

[data-theme="dark"] .recordings-card :deep(.el-table td .cell code) {
  background: rgba(0, 0, 0, 0.4);
  border-color: rgba(0, 255, 255, 0.2);
  color: #00FFFF;
}

/* 分页美化 */
.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
  padding: 16px 0;
}

.pagination :deep(.el-pagination) {
  color: var(--tech-text-primary);
}

.pagination :deep(.el-pagination__total) {
  color: var(--tech-text-secondary);
  font-weight: 500;
}

.pagination :deep(.el-pagination button) {
  background: var(--tech-bg-card);
  border-color: var(--tech-border-light);
  color: var(--tech-text-primary);
  transition: all 0.3s ease;
}

[data-theme="dark"] .pagination :deep(.el-pagination button) {
  background: rgba(30, 30, 30, 0.8);
  border-color: var(--tech-border);
}

.pagination :deep(.el-pagination button:hover) {
  background: var(--tech-bg-glass);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
}

.pagination :deep(.el-pagination button.is-disabled) {
  background: var(--tech-bg-secondary);
  border-color: var(--tech-border-light);
  color: var(--tech-text-muted);
  opacity: 0.5;
}

.pagination :deep(.el-pagination .el-pager li) {
  background: var(--tech-bg-card);
  border-color: var(--tech-border-light);
  color: var(--tech-text-primary);
  transition: all 0.3s ease;
}

[data-theme="dark"] .pagination :deep(.el-pagination .el-pager li) {
  background: rgba(30, 30, 30, 0.8);
  border-color: var(--tech-border);
}

.pagination :deep(.el-pagination .el-pager li:hover) {
  background: var(--tech-bg-glass);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
}

.pagination :deep(.el-pagination .el-pager li.is-active) {
  background-color: var(--tech-primary);
  border-color: var(--tech-primary);
  color: #ffffff;
  font-weight: 600;
}

.pagination :deep(.el-pagination .el-select) {
  border-color: var(--tech-border-light);
}

[data-theme="dark"] .pagination :deep(.el-pagination .el-select) {
  border-color: var(--tech-border);
}

.pagination :deep(.el-pagination .el-select:hover) {
  border-color: var(--tech-primary);
}

/* 空状态 */
.recordings-card :deep(.el-table__empty-block) {
  background: transparent;
  color: var(--tech-text-secondary);
  padding: 40px 0;
}

.recordings-card :deep(.el-table__empty-text) {
  color: var(--tech-text-muted);
  font-size: 14px;
}

/* Loading 状态 */
.recordings-card :deep(.el-loading-mask) {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
}

[data-theme="dark"] .recordings-card :deep(.el-loading-mask) {
  background: rgba(13, 13, 13, 0.8);
}

.recordings-card :deep(.el-loading-spinner .el-loading-text) {
  color: var(--tech-primary);
  font-weight: 500;
}

.recordings-card :deep(.el-loading-spinner .circular) {
  color: var(--tech-primary);
}

/* Player Styles */
.player-dialog :deep(.el-dialog) {
  background: var(--tech-bg-card);
  backdrop-filter: blur(20px);
  border: 1px solid var(--tech-border-light);
}

[data-theme="dark"] .player-dialog :deep(.el-dialog) {
  background: rgba(13, 13, 13, 0.95);
  border: 1px solid var(--tech-border);
}

.player-dialog :deep(.el-dialog__header) {
  background: rgba(0, 102, 255, 0.05);
  border-bottom: 1px solid var(--tech-border-light);
  padding: 16px 20px;
}

[data-theme="dark"] .player-dialog :deep(.el-dialog__header) {
  background: rgba(0, 255, 255, 0.05);
  border-bottom: 1px solid var(--tech-border);
}

.player-dialog :deep(.el-dialog__title) {
  color: var(--tech-text-primary);
  font-family: var(--tech-font-heading);
  font-weight: 600;
}

.player-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.player-container {
  display: flex;
  flex-direction: column;
  background: var(--tech-bg-primary);
  padding: 20px;
  min-height: 500px;
  transition: background-color 0.3s ease;
}

[data-theme="dark"] .player-container {
  background: #0d1117;
}

.player-container.fullscreen {
  min-height: calc(100vh - 100px);
}

.player-info {
  display: flex;
  gap: 24px;
  padding: 12px 16px;
  background: var(--tech-bg-secondary);
  border-radius: 8px;
  margin-bottom: 16px;
  border: 1px solid var(--tech-border-light);
  transition: all 0.3s ease;
}

[data-theme="dark"] .player-info {
  background: rgba(22, 27, 34, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-label {
  color: var(--tech-text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.info-value {
  color: var(--tech-text-primary);
  font-size: 14px;
  font-weight: 600;
  font-family: var(--tech-font-heading);
}

.terminal-wrapper {
  flex: 1;
  background: var(--tech-bg-primary);
  border-radius: 8px;
  border: 1px solid var(--tech-border-light);
  padding: 16px;
  position: relative;
  min-height: 400px;
  overflow: hidden;
  transition: all 0.3s ease;
}

[data-theme="dark"] .terminal-wrapper {
  background: #0d1117;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.fullscreen .terminal-wrapper {
  min-height: calc(100vh - 280px);
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(10px);
  z-index: 10;
  color: var(--tech-text-primary);
  gap: 16px;
  border-radius: 8px;
}

[data-theme="dark"] .loading-overlay {
  background: rgba(13, 17, 23, 0.9);
}

.loading-icon {
  font-size: 40px;
  color: var(--tech-primary);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.terminal-display {
  width: 100%;
  height: 100%;
}

.progress-container {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 0;
  border-top: 1px solid var(--tech-border-light);
  border-bottom: 1px solid var(--tech-border-light);
  margin: 16px 0;
}

[data-theme="dark"] .progress-container {
  border-color: rgba(255, 255, 255, 0.1);
}

.time-display {
  color: var(--tech-text-secondary);
  font-size: 13px;
  font-family: var(--tech-font-mono);
  min-width: 60px;
  font-weight: 600;
}

.progress-slider {
  flex: 1;
}

.progress-slider :deep(.el-slider__runway) {
  background-color: var(--tech-bg-secondary);
  height: 6px;
  border-radius: 3px;
  transition: background-color 0.3s ease;
}

[data-theme="dark"] .progress-slider :deep(.el-slider__runway) {
  background-color: rgba(255, 255, 255, 0.1);
}

.progress-slider :deep(.el-slider__bar) {
  background: linear-gradient(90deg, var(--tech-primary) 0%, var(--tech-primary-dark) 100%);
  height: 6px;
  border-radius: 3px;
}

[data-theme="dark"] .progress-slider :deep(.el-slider__bar) {
  background: linear-gradient(90deg, #00FFFF 0%, #0080FF 100%);
}

.progress-slider :deep(.el-slider__button-wrapper) {
  top: -15px;
}

.progress-slider :deep(.el-slider__button) {
  width: 16px;
  height: 16px;
  border: 3px solid var(--tech-primary);
  background: var(--tech-bg-primary);
  box-shadow: 0 0 10px rgba(0, 102, 255, 0.4);
  transition: all 0.3s ease;
}

[data-theme="dark"] .progress-slider :deep(.el-slider__button) {
  border-color: #00FFFF;
  background: #0d1117;
  box-shadow: 0 0 10px rgba(0, 255, 255, 0.5);
}

.progress-slider :deep(.el-slider__button:hover) {
  transform: scale(1.2);
  box-shadow: 0 0 15px rgba(0, 102, 255, 0.6);
}

[data-theme="dark"] .progress-slider :deep(.el-slider__button:hover) {
  box-shadow: 0 0 20px rgba(0, 255, 255, 0.7);
}

.player-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0;
  border-top: 1px solid var(--tech-border-light);
  margin-top: 16px;
}

[data-theme="dark"] .player-controls {
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.controls-left,
.controls-center,
.controls-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.controls-left :deep(.el-button-group) {
  display: flex;
  gap: 4px;
}

.controls-left :deep(.el-button) {
  background: var(--tech-bg-card);
  border-color: var(--tech-border-light);
  color: var(--tech-text-primary);
  transition: all 0.3s ease;
}

[data-theme="dark"] .controls-left :deep(.el-button) {
  background: rgba(33, 38, 45, 0.8);
  border-color: rgba(255, 255, 255, 0.1);
}

.controls-left :deep(.el-button:hover) {
  background: var(--tech-bg-glass);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
  transform: translateY(-2px);
  box-shadow: var(--tech-shadow-glow);
}

.controls-left :deep(.el-button.is-disabled) {
  background: var(--tech-bg-secondary);
  border-color: var(--tech-border-light);
  color: var(--tech-text-muted);
  opacity: 0.5;
  cursor: not-allowed;
}

.play-btn {
  padding: 10px 24px !important;
  background: var(--tech-primary) !important;
  border-color: var(--tech-primary) !important;
  color: #ffffff !important;
  font-weight: 600;
  box-shadow: 0 4px 12px rgba(0, 102, 255, 0.3);
}

[data-theme="dark"] .play-btn {
  background: rgba(0, 255, 255, 0.2) !important;
  border-color: #00FFFF !important;
  box-shadow: 0 4px 12px rgba(0, 255, 255, 0.3);
}

.play-btn:hover {
  transform: translateY(-2px) !important;
  box-shadow: 0 6px 20px rgba(0, 102, 255, 0.4) !important;
}

[data-theme="dark"] .play-btn:hover {
  box-shadow: 0 6px 20px rgba(0, 255, 255, 0.5) !important;
}

.speed-control {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: var(--tech-bg-card);
  border-radius: 8px;
  border: 1px solid var(--tech-border-light);
}

[data-theme="dark"] .speed-control {
  background: rgba(33, 38, 45, 0.8);
  border-color: rgba(255, 255, 255, 0.1);
}

.speed-label {
  color: var(--tech-text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.speed-control :deep(.el-button-group) {
  display: flex;
  gap: 4px;
}

.speed-control :deep(.el-button-group .el-button) {
  background: transparent;
  border-color: var(--tech-border-light);
  color: var(--tech-text-secondary);
  padding: 6px 12px;
  transition: all 0.3s ease;
}

[data-theme="dark"] .speed-control :deep(.el-button-group .el-button) {
  border-color: rgba(255, 255, 255, 0.1);
}

.speed-control :deep(.el-button-group .el-button:hover) {
  background: var(--tech-bg-glass);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
}

.speed-control :deep(.el-button-group .el-button--primary) {
  background: var(--tech-primary);
  border-color: var(--tech-primary);
  color: #ffffff;
  font-weight: 600;
}

[data-theme="dark"] .speed-control :deep(.el-button-group .el-button--primary) {
  background: rgba(0, 255, 255, 0.2);
  border-color: #00FFFF;
}

.controls-right :deep(.el-button) {
  background: var(--tech-bg-card);
  border-color: var(--tech-border-light);
  color: var(--tech-text-primary);
  transition: all 0.3s ease;
}

[data-theme="dark"] .controls-right :deep(.el-button) {
  background: rgba(33, 38, 45, 0.8);
  border-color: rgba(255, 255, 255, 0.1);
}

.controls-right :deep(.el-button:hover) {
  background: var(--tech-bg-glass);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
  transform: translateY(-2px);
  box-shadow: var(--tech-shadow-glow);
}

.events-info {
  text-align: center;
  padding: 12px;
  color: var(--tech-text-secondary);
  font-size: 12px;
  font-family: var(--tech-font-mono);
  background: var(--tech-bg-secondary);
  border-radius: 6px;
  margin-top: 12px;
  border: 1px solid var(--tech-border-light);
}

[data-theme="dark"] .events-info {
  background: rgba(22, 27, 34, 0.6);
  border-color: rgba(255, 255, 255, 0.05);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .filter-form {
    flex-direction: column;
    align-items: stretch;
  }
  
  .player-info {
    flex-wrap: wrap;
    gap: 12px;
  }
  
  .player-controls {
    flex-wrap: wrap;
    gap: 12px;
  }
  
  .controls-left,
  .controls-center,
  .controls-right {
    flex: 1 1 100%;
    justify-content: center;
  }
}
</style>
