<template>
  <div class="docker-runtime-config">
    <!-- 运行时配置 -->
    <div class="config-section">
      <div class="section-title">运行时</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="容器运行时">
            <el-select v-model="config.runtime" placeholder="默认 (runc)" style="width: 100%" @change="emitChange">
              <el-option label="runc (默认)" value="" />
              <el-option label="nvidia (GPU 支持)" value="nvidia" />
              <el-option label="runsc (gVisor)" value="runsc" />
              <el-option label="kata (Kata Containers)" value="kata" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="使用 Init 进程">
            <el-switch v-model="config.init" @change="emitChange" />
            <div class="form-tip">使用 tini 作为 init 进程处理僵尸进程</div>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="分配 TTY">
            <el-switch v-model="config.tty" @change="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="保持 STDIN 开启">
            <el-switch v-model="config.stdin_open" @change="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="停止信号">
            <el-select v-model="config.stop_signal" placeholder="SIGTERM" style="width: 100%" @change="emitChange">
              <el-option label="SIGTERM (默认)" value="" />
              <el-option label="SIGINT" value="SIGINT" />
              <el-option label="SIGQUIT" value="SIGQUIT" />
              <el-option label="SIGKILL" value="SIGKILL" />
              <el-option label="SIGUSR1" value="SIGUSR1" />
              <el-option label="SIGUSR2" value="SIGUSR2" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 命名空间配置 -->
    <div class="config-section">
      <div class="section-title">命名空间模式</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="PID 命名空间">
            <el-select v-model="config.pid_mode" placeholder="默认" style="width: 100%" @change="emitChange">
              <el-option label="默认 (隔离)" value="" />
              <el-option label="host (共享主机 PID)" value="host" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="IPC 命名空间">
            <el-select v-model="config.ipc_mode" placeholder="默认" style="width: 100%" @change="emitChange">
              <el-option label="默认 (隔离)" value="" />
              <el-option label="host (共享主机 IPC)" value="host" />
              <el-option label="shareable (可共享)" value="shareable" />
              <el-option label="private (私有)" value="private" />
              <el-option label="none (无)" value="none" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="UTS 命名空间">
            <el-select v-model="config.uts_mode" placeholder="默认" style="width: 100%" @change="emitChange">
              <el-option label="默认 (隔离)" value="" />
              <el-option label="host (共享主机 UTS)" value="host" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- Cgroup 配置 -->
    <div class="config-section">
      <div class="section-title">Cgroup 配置</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="父 Cgroup">
            <el-input v-model="config.cgroup_parent" placeholder="/docker" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 内核参数 (sysctls) -->
    <div class="config-section">
      <div class="section-title">内核参数 (sysctls)</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addSysctl">
          <el-icon><Plus /></el-icon>
          添加参数
        </el-button>
      </div>

      <el-table :data="sysctlsList" size="small" border v-if="sysctlsList.length > 0">
        <el-table-column label="参数名" min-width="250">
          <template #default="{ row }">
            <el-input
              v-model="row.key"
              size="small"
              placeholder="net.core.somaxconn"
              @input="updateSysctls"
            />
          </template>
        </el-table-column>
        <el-table-column label="值" min-width="150">
          <template #default="{ row }">
            <el-input
              v-model="row.value"
              size="small"
              placeholder="65535"
              @input="updateSysctls"
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
              @click="removeSysctl($index)"
            />
          </template>
        </el-table-column>
      </el-table>

      <div class="common-presets">
        <span class="tip-label">常用参数:</span>
        <el-button size="small" @click="addPresetSysctl('net.core.somaxconn', '65535')">
          somaxconn: 65535
        </el-button>
        <el-button size="small" @click="addPresetSysctl('net.ipv4.tcp_syncookies', '1')">
          tcp_syncookies: 1
        </el-button>
        <el-button size="small" @click="addPresetSysctl('vm.overcommit_memory', '1')">
          overcommit: 1
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  runtime: '',
  init: false,
  tty: false,
  stdin_open: false,
  stop_signal: '',
  pid_mode: '',
  ipc_mode: '',
  uts_mode: '',
  cgroup_parent: '',
  sysctls: {}
})

const sysctlsList = ref([])

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    Object.assign(config, {
      runtime: newVal.runtime || '',
      init: newVal.init || false,
      tty: newVal.tty || false,
      stdin_open: newVal.stdin_open || false,
      stop_signal: newVal.stop_signal || '',
      pid_mode: newVal.pid_mode || '',
      ipc_mode: newVal.ipc_mode || '',
      uts_mode: newVal.uts_mode || '',
      cgroup_parent: newVal.cgroup_parent || '',
      sysctls: newVal.sysctls || {}
    })

    // 转换 sysctls 对象为列表
    sysctlsList.value = Object.entries(newVal.sysctls || {}).map(([key, value]) => ({
      key,
      value
    }))
  }
}, { immediate: true, deep: true })

function emitChange() {
  emit('update:modelValue', { ...config })
}

function addSysctl() {
  sysctlsList.value.push({ key: '', value: '' })
  updateSysctls()
}

function removeSysctl(index) {
  sysctlsList.value.splice(index, 1)
  updateSysctls()
}

function updateSysctls() {
  const sysctls = {}
  sysctlsList.value.forEach(item => {
    if (item.key) {
      sysctls[item.key] = item.value
    }
  })
  config.sysctls = sysctls
  emitChange()
}

function addPresetSysctl(key, value) {
  const exists = sysctlsList.value.find(s => s.key === key)
  if (exists) {
    exists.value = value
  } else {
    sysctlsList.value.push({ key, value })
  }
  updateSysctls()
}
</script>

<style scoped>
.docker-runtime-config {
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
