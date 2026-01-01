<template>
  <div class="recordings-container">
    <div class="header">
      <h2>会话录像</h2>
      <div class="stats" v-if="stats">
        <el-tag type="info">总录像: {{ stats.total_recordings }}</el-tag>
        <el-tag type="success">存储大小: {{ formatSize(stats.total_size_bytes) }}</el-tag>
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
    <el-card class="recordings-card">
      <el-table
        :data="recordings"
        v-loading="loading"
        stripe
        border
        style="width: 100%"
        :default-sort="{ prop: 'created_at', order: 'descending' }"
      >
        <el-table-column prop="created_at" label="录制时间" width="180">
          <template #default="scope">
            {{ formatTime(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="client_id" label="客户端 ID" width="150" />
        <el-table-column prop="username" label="用户名" width="100" />
        <el-table-column prop="duration" label="时长" width="100">
          <template #default="scope">
            {{ formatDuration(scope.row.duration) }}
          </template>
        </el-table-column>
        <el-table-column label="终端大小" width="100">
          <template #default="scope">
            {{ scope.row.width }}x{{ scope.row.height }}
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" width="100">
          <template #default="scope">
            {{ formatSize(scope.row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
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

.recordings-card {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* Player Styles */
.player-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.player-container {
  display: flex;
  flex-direction: column;
  background: #0d1117;
  padding: 16px;
  min-height: 500px;
}

.player-container.fullscreen {
  min-height: calc(100vh - 100px);
}

.player-info {
  display: flex;
  gap: 24px;
  padding: 8px 12px;
  background: #161b22;
  border-radius: 6px;
  margin-bottom: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.info-label {
  color: #8b949e;
  font-size: 12px;
}

.info-value {
  color: #c9d1d9;
  font-size: 13px;
  font-weight: 500;
}

.terminal-wrapper {
  flex: 1;
  background: #0d1117;
  border-radius: 8px;
  border: 1px solid #30363d;
  padding: 12px;
  position: relative;
  min-height: 400px;
  overflow: hidden;
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
  background: rgba(13, 17, 23, 0.9);
  z-index: 10;
  color: #c9d1d9;
  gap: 12px;
}

.loading-icon {
  font-size: 32px;
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
  gap: 12px;
  padding: 12px 0;
}

.time-display {
  color: #8b949e;
  font-size: 13px;
  font-family: 'JetBrains Mono', monospace;
  min-width: 50px;
}

.progress-slider {
  flex: 1;
}

.progress-slider :deep(.el-slider__runway) {
  background-color: #30363d;
  height: 4px;
}

.progress-slider :deep(.el-slider__bar) {
  background-color: #58a6ff;
  height: 4px;
}

.progress-slider :deep(.el-slider__button-wrapper) {
  top: -15px;
}

.progress-slider :deep(.el-slider__button) {
  width: 14px;
  height: 14px;
  border: 2px solid #58a6ff;
  background: #0d1117;
}

.player-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-top: 1px solid #30363d;
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
}

.controls-left :deep(.el-button) {
  background: #21262d;
  border-color: #30363d;
  color: #c9d1d9;
}

.controls-left :deep(.el-button:hover) {
  background: #30363d;
  border-color: #8b949e;
  color: #f0f6fc;
}

.controls-left :deep(.el-button.is-disabled) {
  background: #161b22;
  border-color: #21262d;
  color: #484f58;
}

.play-btn {
  padding: 8px 20px !important;
}

.speed-control {
  display: flex;
  align-items: center;
  gap: 8px;
}

.speed-label {
  color: #8b949e;
  font-size: 13px;
}

.speed-control :deep(.el-button-group .el-button) {
  background: #21262d;
  border-color: #30363d;
  color: #8b949e;
  padding: 5px 10px;
}

.speed-control :deep(.el-button-group .el-button:hover) {
  background: #30363d;
  color: #c9d1d9;
}

.speed-control :deep(.el-button-group .el-button--primary) {
  background: #238636;
  border-color: #238636;
  color: #ffffff;
}

.controls-right :deep(.el-button) {
  background: #21262d;
  border-color: #30363d;
  color: #c9d1d9;
}

.controls-right :deep(.el-button:hover) {
  background: #30363d;
  border-color: #8b949e;
  color: #f0f6fc;
}

.events-info {
  text-align: center;
  padding: 8px;
  color: #6e7681;
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
}
</style>
