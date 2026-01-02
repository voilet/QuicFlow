<template>
  <div class="docker-log-config">
    <!-- 日志驱动 -->
    <div class="config-section">
      <div class="section-title">日志配置</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="日志驱动">
            <el-select v-model="config.log_driver" placeholder="默认 (json-file)" style="width: 100%" @change="emitChange">
              <el-option label="json-file (默认)" value="" />
              <el-option label="json-file" value="json-file" />
              <el-option label="syslog" value="syslog" />
              <el-option label="journald" value="journald" />
              <el-option label="gelf (Graylog)" value="gelf" />
              <el-option label="fluentd" value="fluentd" />
              <el-option label="awslogs (AWS CloudWatch)" value="awslogs" />
              <el-option label="splunk" value="splunk" />
              <el-option label="none (禁用日志)" value="none" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- json-file 驱动选项 -->
    <div class="config-section" v-if="!config.log_driver || config.log_driver === 'json-file'">
      <div class="section-title">JSON 日志选项</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="最大文件大小">
            <el-input v-model="logOpts['max-size']" placeholder="10m" @input="emitChange">
              <template #append>
                <el-tooltip content="如: 10m, 100m, 1g">
                  <el-icon><QuestionFilled /></el-icon>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="最大文件数">
            <el-input-number
              v-model.number="logOptsMaxFile"
              :min="1"
              :max="100"
              controls-position="right"
              @change="updateMaxFile"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="压缩日志">
            <el-select v-model="logOpts['compress']" @change="emitChange">
              <el-option label="启用" value="true" />
              <el-option label="禁用" value="false" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- syslog 驱动选项 -->
    <div class="config-section" v-if="config.log_driver === 'syslog'">
      <div class="section-title">Syslog 选项</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="Syslog 地址">
            <el-input v-model="logOpts['syslog-address']" placeholder="udp://localhost:514" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="Syslog Facility">
            <el-select v-model="logOpts['syslog-facility']" placeholder="daemon" @change="emitChange">
              <el-option label="daemon" value="daemon" />
              <el-option label="user" value="user" />
              <el-option label="local0" value="local0" />
              <el-option label="local1" value="local1" />
              <el-option label="local2" value="local2" />
              <el-option label="local3" value="local3" />
              <el-option label="local4" value="local4" />
              <el-option label="local5" value="local5" />
              <el-option label="local6" value="local6" />
              <el-option label="local7" value="local7" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="Syslog Tag">
            <el-input v-model="logOpts['tag']" placeholder="{{.Name}}" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- fluentd 驱动选项 -->
    <div class="config-section" v-if="config.log_driver === 'fluentd'">
      <div class="section-title">Fluentd 选项</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="Fluentd 地址">
            <el-input v-model="logOpts['fluentd-address']" placeholder="localhost:24224" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="异步模式">
            <el-select v-model="logOpts['fluentd-async']" @change="emitChange">
              <el-option label="启用" value="true" />
              <el-option label="禁用" value="false" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="Tag">
            <el-input v-model="logOpts['tag']" placeholder="docker.{{.Name}}" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 自定义日志选项 -->
    <div class="config-section">
      <div class="section-title">自定义日志选项</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addLogOpt">
          <el-icon><Plus /></el-icon>
          添加选项
        </el-button>
      </div>

      <el-table :data="customLogOpts" size="small" border v-if="customLogOpts.length > 0">
        <el-table-column label="选项名" min-width="200">
          <template #default="{ row }">
            <el-input
              v-model="row.key"
              size="small"
              placeholder="选项名"
              @input="updateCustomLogOpts"
            />
          </template>
        </el-table-column>
        <el-table-column label="值" min-width="200">
          <template #default="{ row }">
            <el-input
              v-model="row.value"
              size="small"
              placeholder="值"
              @input="updateCustomLogOpts"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template #default="{ $index }">
            <el-button
              type="danger"
              size="small"
              :icon="Delete"
              circle
              @click="removeLogOpt($index)"
            />
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, watch, computed } from 'vue'
import { Plus, Delete, QuestionFilled } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  log_driver: '',
  log_opts: {}
})

const logOpts = reactive({})
const logOptsMaxFile = ref(3)
const customLogOpts = ref([])

// 预定义的日志选项 key
const predefinedLogOptKeys = [
  'max-size', 'max-file', 'compress',
  'syslog-address', 'syslog-facility', 'tag',
  'fluentd-address', 'fluentd-async'
]

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    config.log_driver = newVal.log_driver || ''
    config.log_opts = newVal.log_opts || {}

    // 填充预定义选项
    Object.keys(config.log_opts).forEach(key => {
      if (predefinedLogOptKeys.includes(key)) {
        logOpts[key] = config.log_opts[key]
      }
    })

    // max-file 特殊处理
    if (config.log_opts['max-file']) {
      logOptsMaxFile.value = parseInt(config.log_opts['max-file']) || 3
    }

    // 自定义选项
    customLogOpts.value = Object.entries(config.log_opts)
      .filter(([key]) => !predefinedLogOptKeys.includes(key))
      .map(([key, value]) => ({ key, value }))
  }
}, { immediate: true, deep: true })

function emitChange() {
  // 合并所有日志选项
  const allOpts = { ...logOpts }

  // 添加自定义选项
  customLogOpts.value.forEach(opt => {
    if (opt.key) {
      allOpts[opt.key] = opt.value
    }
  })

  // 清理空值
  Object.keys(allOpts).forEach(key => {
    if (!allOpts[key]) {
      delete allOpts[key]
    }
  })

  config.log_opts = allOpts
  emit('update:modelValue', { ...config })
}

function updateMaxFile() {
  logOpts['max-file'] = String(logOptsMaxFile.value)
  emitChange()
}

function addLogOpt() {
  customLogOpts.value.push({ key: '', value: '' })
}

function removeLogOpt(index) {
  customLogOpts.value.splice(index, 1)
  updateCustomLogOpts()
}

function updateCustomLogOpts() {
  emitChange()
}
</script>

<style scoped>
.docker-log-config {
  padding: 12px 0;
}

.config-section {
  margin-bottom: 24px;
  padding: 16px;
  background: var(--tech-bg-tertiary);
  border-radius: 4px;
  border: 1px solid var(--tech-border);
}

.section-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--tech-text-primary);
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--tech-border);
}

.config-header {
  margin-bottom: 12px;
}

:deep(.el-table) {
  background: transparent;
}

:deep(.el-table th) {
  background-color: var(--tech-bg-card);
  color: var(--tech-text-primary);
  font-weight: 600;
}

:deep(.el-table td) {
  background-color: var(--tech-bg-card);
}
</style>
