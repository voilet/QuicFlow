<template>
  <div class="docker-resource-config">
    <!-- 内存配置 -->
    <div class="config-section">
      <div class="section-title">内存配置</div>
      <el-row :gutter="20">
        <el-col :span="6">
          <el-form-item label="内存限制">
            <el-input v-model="config.memory_limit" placeholder="512m, 1g" @input="emitChange">
              <template #append>
                <el-tooltip content="如: 512m, 1g, 2g">
                  <el-icon><QuestionFilled /></el-icon>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="6">
          <el-form-item label="内存预留">
            <el-input v-model="config.memory_reserve" placeholder="256m" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="6">
          <el-form-item label="内存+交换限制">
            <el-input v-model="config.memory_swap" placeholder="-1 无限制" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="6">
          <el-form-item label="交换倾向 (0-100)">
            <el-input-number
              v-model="config.memory_swappiness"
              :min="0"
              :max="100"
              controls-position="right"
              @change="emitChange"
            />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- CPU 配置 -->
    <div class="config-section">
      <div class="section-title">CPU 配置</div>
      <el-row :gutter="20">
        <el-col :span="6">
          <el-form-item label="CPU 限制">
            <el-input v-model="config.cpu_limit" placeholder="0.5, 1, 2" @input="emitChange">
              <template #append>核心数</template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="6">
          <el-form-item label="CPU 权重">
            <el-input-number
              v-model="config.cpu_shares"
              :min="0"
              :max="10240"
              :step="128"
              controls-position="right"
              placeholder="1024"
              @change="emitChange"
            />
            <div class="form-tip">默认 1024，相对权重值</div>
          </el-form-item>
        </el-col>
        <el-col :span="6">
          <el-form-item label="CPU 绑定">
            <el-input v-model="config.cpuset_cpus" placeholder="0,1 或 0-3" @input="emitChange" />
            <div class="form-tip">绑定到特定 CPU 核心</div>
          </el-form-item>
        </el-col>
        <el-col :span="6">
          <el-form-item label="内存节点绑定">
            <el-input v-model="config.cpuset_mems" placeholder="0,1" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 进程限制 -->
    <div class="config-section">
      <div class="section-title">进程限制</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="进程数限制">
            <el-input-number
              v-model="config.pids_limit"
              :min="-1"
              controls-position="right"
              placeholder="-1 无限制"
              @change="emitChange"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="/dev/shm 大小">
            <el-input v-model="config.shm_size" placeholder="64m" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- OOM 配置 -->
    <div class="config-section">
      <div class="section-title">OOM 配置</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="禁用 OOM Killer">
            <el-switch v-model="config.oom_kill_disable" @change="emitChange" />
            <div class="form-tip warning" v-if="config.oom_kill_disable">
              禁用 OOM Killer 可能导致系统不稳定
            </div>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="OOM 分数调整">
            <el-input-number
              v-model="config.oom_score_adj"
              :min="-1000"
              :max="1000"
              :step="100"
              controls-position="right"
              @change="emitChange"
            />
            <div class="form-tip">-1000 到 1000，值越小越不容易被 kill</div>
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- Ulimits 配置 -->
    <div class="config-section">
      <div class="section-title">Ulimits 资源限制</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addUlimit">
          <el-icon><Plus /></el-icon>
          添加 Ulimit
        </el-button>
      </div>

      <el-table :data="config.ulimits" size="small" border v-if="config.ulimits?.length > 0">
        <el-table-column label="资源名称" min-width="150">
          <template #default="{ row }">
            <el-select v-model="row.name" size="small" filterable allow-create @change="emitChange">
              <el-option label="nofile (文件描述符)" value="nofile" />
              <el-option label="nproc (进程数)" value="nproc" />
              <el-option label="core (core dump 大小)" value="core" />
              <el-option label="memlock (锁定内存)" value="memlock" />
              <el-option label="stack (栈大小)" value="stack" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="软限制" width="140">
          <template #default="{ row }">
            <el-input-number
              v-model="row.soft"
              size="small"
              :min="-1"
              controls-position="right"
              @change="emitChange"
            />
          </template>
        </el-table-column>
        <el-table-column label="硬限制" width="140">
          <template #default="{ row }">
            <el-input-number
              v-model="row.hard"
              size="small"
              :min="-1"
              controls-position="right"
              @change="emitChange"
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
              @click="removeUlimit($index)"
            />
          </template>
        </el-table-column>
      </el-table>

      <div class="common-presets">
        <span class="tip-label">快速添加:</span>
        <el-button size="small" @click="addPresetUlimit('nofile', 65536, 65536)">
          nofile: 65536
        </el-button>
        <el-button size="small" @click="addPresetUlimit('nproc', 4096, 4096)">
          nproc: 4096
        </el-button>
        <el-button size="small" @click="addPresetUlimit('core', -1, -1)">
          core: unlimited
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch } from 'vue'
import { Plus, Delete, QuestionFilled } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  memory_limit: '',
  memory_reserve: '',
  memory_swap: '',
  memory_swappiness: null,
  cpu_limit: '',
  cpu_shares: null,
  cpuset_cpus: '',
  cpuset_mems: '',
  pids_limit: null,
  shm_size: '',
  oom_kill_disable: false,
  oom_score_adj: 0,
  ulimits: []
})

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    Object.assign(config, {
      memory_limit: newVal.memory_limit || '',
      memory_reserve: newVal.memory_reserve || '',
      memory_swap: newVal.memory_swap || '',
      memory_swappiness: newVal.memory_swappiness ?? null,
      cpu_limit: newVal.cpu_limit || '',
      cpu_shares: newVal.cpu_shares ?? null,
      cpuset_cpus: newVal.cpuset_cpus || '',
      cpuset_mems: newVal.cpuset_mems || '',
      pids_limit: newVal.pids_limit ?? null,
      shm_size: newVal.shm_size || '',
      oom_kill_disable: newVal.oom_kill_disable || false,
      oom_score_adj: newVal.oom_score_adj || 0,
      ulimits: newVal.ulimits || []
    })
  }
}, { immediate: true, deep: true })

function emitChange() {
  emit('update:modelValue', { ...config })
}

function addUlimit() {
  config.ulimits.push({ name: 'nofile', soft: 65536, hard: 65536 })
  emitChange()
}

function removeUlimit(index) {
  config.ulimits.splice(index, 1)
  emitChange()
}

function addPresetUlimit(name, soft, hard) {
  // 检查是否已存在
  const exists = config.ulimits.find(u => u.name === name)
  if (exists) {
    exists.soft = soft
    exists.hard = hard
  } else {
    config.ulimits.push({ name, soft, hard })
  }
  emitChange()
}
</script>

<style scoped>
.docker-resource-config {
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

.form-tip {
  font-size: 12px;
  color: var(--tech-text-muted);
  margin-top: 4px;
}

.form-tip.warning {
  color: var(--tech-warning);
}

.common-presets {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.tip-label {
  font-size: 12px;
  color: var(--tech-text-muted);
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
