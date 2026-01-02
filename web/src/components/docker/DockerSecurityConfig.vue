<template>
  <div class="docker-security-config">
    <!-- 基础安全选项 -->
    <div class="config-section">
      <div class="section-title">基础安全</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="特权模式">
            <el-switch v-model="config.privileged" @change="emitChange" />
            <div class="form-tip warning" v-if="config.privileged">
              特权模式会授予容器几乎全部主机权限，请谨慎使用
            </div>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="只读根文件系统">
            <el-switch v-model="config.read_only_rootfs" @change="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="禁止新权限">
            <el-switch v-model="config.no_new_privileges" @change="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- Capabilities -->
    <div class="config-section">
      <div class="section-title">Capabilities 配置</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="添加 Capabilities">
            <el-select
              v-model="config.cap_add"
              multiple
              filterable
              allow-create
              placeholder="选择或输入要添加的 Capabilities"
              style="width: 100%"
              @change="emitChange"
            >
              <el-option v-for="cap in commonCapabilities" :key="cap" :label="cap" :value="cap" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="移除 Capabilities">
            <el-select
              v-model="config.cap_drop"
              multiple
              filterable
              allow-create
              placeholder="选择或输入要移除的 Capabilities"
              style="width: 100%"
              @change="emitChange"
            >
              <el-option label="ALL (移除全部)" value="ALL" />
              <el-option v-for="cap in commonCapabilities" :key="cap" :label="cap" :value="cap" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 安全选项 -->
    <div class="config-section">
      <div class="section-title">安全选项 (security_opt)</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addSecurityOpt">
          <el-icon><Plus /></el-icon>
          添加安全选项
        </el-button>
      </div>

      <div class="security-opts-list" v-if="config.security_opt?.length > 0">
        <div v-for="(opt, index) in config.security_opt" :key="index" class="security-opt-item">
          <el-input
            v-model="config.security_opt[index]"
            size="small"
            placeholder="如: no-new-privileges:true, apparmor:unconfined"
            @input="emitChange"
          />
          <el-button
            type="danger"
            size="small"
            :icon="Delete"
            circle
            @click="removeSecurityOpt(index)"
          />
        </div>
      </div>

      <div class="common-opts">
        <span class="tip-label">常用选项:</span>
        <el-tag
          v-for="opt in commonSecurityOpts"
          :key="opt"
          size="small"
          class="opt-tag"
          @click="addCommonSecurityOpt(opt)"
        >
          {{ opt }}
        </el-tag>
      </div>
    </div>

    <!-- 用户命名空间 -->
    <div class="config-section">
      <div class="section-title">命名空间配置</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="用户命名空间模式">
            <el-select v-model="config.userns_mode" placeholder="默认" style="width: 100%" @change="emitChange">
              <el-option label="默认" value="" />
              <el-option label="host (使用主机用户命名空间)" value="host" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  privileged: false,
  read_only_rootfs: false,
  no_new_privileges: false,
  cap_add: [],
  cap_drop: [],
  security_opt: [],
  userns_mode: ''
})

// 常用 Capabilities
const commonCapabilities = [
  'NET_ADMIN',
  'NET_RAW',
  'SYS_ADMIN',
  'SYS_PTRACE',
  'SYS_TIME',
  'SYS_RESOURCE',
  'MKNOD',
  'AUDIT_WRITE',
  'SETFCAP',
  'IPC_LOCK',
  'NET_BIND_SERVICE'
]

// 常用安全选项
const commonSecurityOpts = [
  'no-new-privileges:true',
  'apparmor:unconfined',
  'seccomp:unconfined',
  'label:disable'
]

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    Object.assign(config, {
      privileged: newVal.privileged || false,
      read_only_rootfs: newVal.read_only_rootfs || false,
      no_new_privileges: newVal.no_new_privileges || false,
      cap_add: newVal.cap_add || [],
      cap_drop: newVal.cap_drop || [],
      security_opt: newVal.security_opt || [],
      userns_mode: newVal.userns_mode || ''
    })
  }
}, { immediate: true, deep: true })

function emitChange() {
  emit('update:modelValue', { ...config })
}

function addSecurityOpt() {
  config.security_opt.push('')
  emitChange()
}

function removeSecurityOpt(index) {
  config.security_opt.splice(index, 1)
  emitChange()
}

function addCommonSecurityOpt(opt) {
  if (!config.security_opt.includes(opt)) {
    config.security_opt.push(opt)
    emitChange()
  }
}
</script>

<style scoped>
.docker-security-config {
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

.security-opts-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.security-opt-item {
  display: flex;
  gap: 8px;
  align-items: center;
}

.security-opt-item .el-input {
  flex: 1;
}

.common-opts {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 12px;
}

.tip-label {
  font-size: 12px;
  color: var(--tech-text-muted);
}

.opt-tag {
  cursor: pointer;
  transition: all 0.2s;
}

.opt-tag:hover {
  background-color: var(--tech-primary);
  color: #fff;
  border-color: var(--tech-primary);
}
</style>
