<template>
  <div class="docker-device-config">
    <!-- 设备映射 -->
    <div class="config-section">
      <div class="section-title">设备映射</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addDevice">
          <el-icon><Plus /></el-icon>
          添加设备
        </el-button>
      </div>

      <el-table :data="config.devices" size="small" border v-if="config.devices?.length > 0">
        <el-table-column label="主机设备路径" min-width="180">
          <template #default="{ row }">
            <el-input
              v-model="row.host_path"
              size="small"
              placeholder="/dev/sda"
              @input="emitChange"
            />
          </template>
        </el-table-column>
        <el-table-column label="容器设备路径" min-width="180">
          <template #default="{ row }">
            <el-input
              v-model="row.container_path"
              size="small"
              placeholder="/dev/sda (留空则同主机路径)"
              @input="emitChange"
            />
          </template>
        </el-table-column>
        <el-table-column label="权限" width="100">
          <template #default="{ row }">
            <el-input
              v-model="row.permissions"
              size="small"
              placeholder="rwm"
              @input="emitChange"
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
              @click="removeDevice($index)"
            />
          </template>
        </el-table-column>
      </el-table>

      <div class="common-presets">
        <span class="tip-label">常用设备:</span>
        <el-button size="small" @click="addPresetDevice('/dev/fuse', '/dev/fuse', 'rwm')">
          FUSE
        </el-button>
        <el-button size="small" @click="addPresetDevice('/dev/net/tun', '/dev/net/tun', 'rwm')">
          TUN
        </el-button>
        <el-button size="small" @click="addPresetDevice('/dev/kvm', '/dev/kvm', 'rwm')">
          KVM
        </el-button>
      </div>
    </div>

    <!-- GPU 配置 -->
    <div class="config-section">
      <div class="section-title">GPU 配置</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="GPU 访问">
            <el-select v-model="config.gpus" placeholder="不使用 GPU" style="width: 100%" @change="emitChange">
              <el-option label="不使用 GPU" value="" />
              <el-option label="all (所有 GPU)" value="all" />
              <el-option label="device=0 (第一个 GPU)" value="device=0" />
              <el-option label="device=0,1 (前两个 GPU)" value="device=0,1" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="自定义 GPU ID">
            <el-input
              v-model="customGpuId"
              placeholder="如: device=GPU-xxxxx"
              :disabled="config.gpus === 'all'"
              @input="updateCustomGpu"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <div class="form-tip">
        使用 GPU 需要安装 NVIDIA Container Toolkit 并设置运行时为 nvidia
      </div>
    </div>

    <!-- 设备 Cgroup 规则 -->
    <div class="config-section">
      <div class="section-title">设备 Cgroup 规则</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addCgroupRule">
          <el-icon><Plus /></el-icon>
          添加规则
        </el-button>
      </div>

      <el-table :data="config.device_cgroup_rules" size="small" border v-if="config.device_cgroup_rules?.length > 0">
        <el-table-column label="规则" min-width="300">
          <template #default="{ row, $index }">
            <el-input
              v-model="config.device_cgroup_rules[$index]"
              size="small"
              placeholder="c 1:3 mr (字符设备 1:3 可读)"
              @input="emitChange"
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
              @click="removeCgroupRule($index)"
            />
          </template>
        </el-table-column>
      </el-table>

      <div class="form-tip mt-12">
        规则格式: [类型] [主:次] [权限]<br>
        类型: a=所有, b=块设备, c=字符设备<br>
        权限: r=读, w=写, m=创建
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
  devices: [],
  gpus: '',
  device_cgroup_rules: []
})

const customGpuId = ref('')

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    config.devices = newVal.devices || []
    config.gpus = newVal.gpus || ''
    config.device_cgroup_rules = newVal.device_cgroup_rules || []

    // 如果 gpus 不是预设值，则是自定义
    if (config.gpus && !['all', 'device=0', 'device=0,1'].includes(config.gpus)) {
      customGpuId.value = config.gpus
    }
  }
}, { immediate: true, deep: true })

function emitChange() {
  emit('update:modelValue', { ...config })
}

function addDevice() {
  config.devices.push({
    host_path: '',
    container_path: '',
    permissions: 'rwm'
  })
  emitChange()
}

function removeDevice(index) {
  config.devices.splice(index, 1)
  emitChange()
}

function addPresetDevice(hostPath, containerPath, permissions) {
  const exists = config.devices.find(d => d.host_path === hostPath)
  if (!exists) {
    config.devices.push({
      host_path: hostPath,
      container_path: containerPath,
      permissions
    })
    emitChange()
  }
}

function updateCustomGpu() {
  if (customGpuId.value) {
    config.gpus = customGpuId.value
    emitChange()
  }
}

function addCgroupRule() {
  config.device_cgroup_rules.push('')
  emitChange()
}

function removeCgroupRule(index) {
  config.device_cgroup_rules.splice(index, 1)
  emitChange()
}
</script>

<style scoped>
.docker-device-config {
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
  line-height: 1.6;
}

.form-tip.mt-12 {
  margin-top: 12px;
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
