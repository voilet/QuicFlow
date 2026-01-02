<template>
  <el-dialog
    v-model="visible"
    title="Docker 容器配置"
    width="900px"
    :close-on-click-modal="false"
    destroy-on-close
    class="docker-config-dialog"
  >
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 基础配置 -->
      <el-tab-pane label="基础配置" name="basic">
        <el-form :model="config" label-width="120px" class="config-form">
          <el-divider content-position="left">镜像配置</el-divider>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="镜像地址" required>
                <el-input v-model="config.image" placeholder="nginx:latest" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="镜像拉取策略">
                <el-select v-model="config.image_pull_policy" placeholder="默认" style="width: 100%">
                  <el-option label="总是拉取 (always)" value="always" />
                  <el-option label="不存在时拉取 (ifnotpresent)" value="ifnotpresent" />
                  <el-option label="从不拉取 (never)" value="never" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="镜像仓库">
                <el-input v-model="config.registry" placeholder="registry.example.com" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="目标平台">
                <el-select v-model="config.platform" placeholder="默认" style="width: 100%">
                  <el-option label="默认 (自动检测)" value="" />
                  <el-option label="linux/amd64" value="linux/amd64" />
                  <el-option label="linux/arm64" value="linux/arm64" />
                  <el-option label="linux/arm/v7" value="linux/arm/v7" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="仓库用户名">
                <el-input v-model="config.registry_user" placeholder="用户名" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="仓库密码">
                <el-input v-model="config.registry_pass" type="password" show-password placeholder="密码" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-divider content-position="left">容器基础配置</el-divider>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="容器名称">
                <el-input v-model="config.container_name" placeholder="my-container" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="运行用户">
                <el-input v-model="config.user" placeholder="1000:1000 或 username" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="工作目录">
                <el-input v-model="config.working_dir" placeholder="/app" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="附加用户组">
                <el-select
                  v-model="config.group_add"
                  multiple
                  filterable
                  allow-create
                  placeholder="输入用户组"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-divider content-position="left">启动命令</el-divider>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="Entrypoint">
                <el-input v-model="entrypointStr" placeholder="/entrypoint.sh" @blur="parseEntrypoint" />
                <div class="form-tip">多个参数用空格分隔</div>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Command">
                <el-input v-model="commandStr" placeholder="--config /etc/config.yaml" @blur="parseCommand" />
                <div class="form-tip">多个参数用空格分隔</div>
              </el-form-item>
            </el-col>
          </el-row>

          <el-divider content-position="left">环境变量</el-divider>
          <div class="env-config">
            <div class="config-header">
              <el-button type="primary" size="small" @click="addEnv">
                <el-icon><Plus /></el-icon>
                添加环境变量
              </el-button>
            </div>
            <el-table :data="envList" size="small" border v-if="envList.length > 0" max-height="200">
              <el-table-column label="变量名" min-width="150">
                <template #default="{ row }">
                  <el-input v-model="row.key" size="small" placeholder="KEY" @input="updateEnv" />
                </template>
              </el-table-column>
              <el-table-column label="值" min-width="200">
                <template #default="{ row }">
                  <el-input v-model="row.value" size="small" placeholder="value" @input="updateEnv" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="80" align="center">
                <template #default="{ $index }">
                  <el-button type="danger" size="small" :icon="Delete" circle @click="removeEnv($index)" />
                </template>
              </el-table-column>
            </el-table>
          </div>

          <el-divider content-position="left">标签</el-divider>
          <div class="labels-config">
            <div class="config-header">
              <el-button type="primary" size="small" @click="addLabel">
                <el-icon><Plus /></el-icon>
                添加标签
              </el-button>
            </div>
            <el-table :data="labelsList" size="small" border v-if="labelsList.length > 0" max-height="150">
              <el-table-column label="标签名" min-width="150">
                <template #default="{ row }">
                  <el-input v-model="row.key" size="small" placeholder="com.example.key" @input="updateLabels" />
                </template>
              </el-table-column>
              <el-table-column label="值" min-width="200">
                <template #default="{ row }">
                  <el-input v-model="row.value" size="small" placeholder="value" @input="updateLabels" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="80" align="center">
                <template #default="{ $index }">
                  <el-button type="danger" size="small" :icon="Delete" circle @click="removeLabel($index)" />
                </template>
              </el-table-column>
            </el-table>
          </div>

          <el-divider content-position="left">重启策略</el-divider>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="重启策略">
                <el-select v-model="config.restart_policy" style="width: 100%">
                  <el-option label="不重启 (no)" value="no" />
                  <el-option label="总是重启 (always)" value="always" />
                  <el-option label="失败时重启 (on-failure)" value="on-failure" />
                  <el-option label="除非手动停止 (unless-stopped)" value="unless-stopped" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12" v-if="config.restart_policy === 'on-failure'">
              <el-form-item label="最大重试次数">
                <el-input-number v-model="config.restart_max_retries" :min="0" controls-position="right" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="停止超时">
                <el-input-number v-model="config.stop_timeout" :min="0" controls-position="right">
                  <template #append>秒</template>
                </el-input-number>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="自动删除">
                <el-switch v-model="config.auto_remove" />
                <div class="form-tip">退出时自动删除容器</div>
              </el-form-item>
            </el-col>
          </el-row>

          <el-divider content-position="left">部署策略</el-divider>
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form-item label="移除旧容器">
                <el-switch v-model="config.remove_old" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="先拉取再停止">
                <el-switch v-model="config.pull_before_stop" />
                <div class="form-tip">减少服务中断时间</div>
              </el-form-item>
            </el-col>
            <el-col :span="8" v-if="config.remove_old">
              <el-form-item label="保留旧容器数">
                <el-input-number v-model="config.keep_old_count" :min="0" controls-position="right" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </el-tab-pane>

      <!-- 端口映射 -->
      <el-tab-pane label="端口映射" name="ports">
        <DockerPortsConfig v-model="config.ports" />
      </el-tab-pane>

      <!-- 存储配置 -->
      <el-tab-pane label="存储配置" name="volumes">
        <DockerVolumesConfig
          v-model="config.volumes"
          v-model:tmpfs-mounts="config.tmpfs_mounts"
        />
      </el-tab-pane>

      <!-- 网络配置 -->
      <el-tab-pane label="网络配置" name="network">
        <DockerNetworkConfig v-model="networkConfig" @update:modelValue="updateNetworkConfig" />
      </el-tab-pane>

      <!-- 安全配置 -->
      <el-tab-pane label="安全配置" name="security">
        <DockerSecurityConfig v-model="securityConfig" @update:modelValue="updateSecurityConfig" />
      </el-tab-pane>

      <!-- 资源限制 -->
      <el-tab-pane label="资源限制" name="resources">
        <DockerResourceConfig v-model="resourceConfig" @update:modelValue="updateResourceConfig" />
      </el-tab-pane>

      <!-- 设备配置 -->
      <el-tab-pane label="设备配置" name="devices">
        <DockerDeviceConfig v-model="deviceConfig" @update:modelValue="updateDeviceConfig" />
      </el-tab-pane>

      <!-- 日志配置 -->
      <el-tab-pane label="日志配置" name="logs">
        <DockerLogConfig v-model="logConfig" @update:modelValue="updateLogConfig" />
      </el-tab-pane>

      <!-- 运行时配置 -->
      <el-tab-pane label="运行时" name="runtime">
        <DockerRuntimeConfig v-model="runtimeConfig" @update:modelValue="updateRuntimeConfig" />
      </el-tab-pane>

      <!-- 健康检查 -->
      <el-tab-pane label="健康检查" name="healthcheck">
        <el-form :model="config.health_check" label-width="120px" class="config-form">
          <el-form-item label="启用健康检查">
            <el-switch v-model="healthCheckEnabled" />
          </el-form-item>

          <template v-if="healthCheckEnabled">
            <el-form-item label="检查命令">
              <el-input
                v-model="healthCheckCommand"
                placeholder="CMD curl -f http://localhost/ || exit 1"
                @blur="parseHealthCheckCommand"
              />
              <div class="form-tip">如: CMD curl -f http://localhost/ || exit 1</div>
            </el-form-item>
            <el-row :gutter="20">
              <el-col :span="6">
                <el-form-item label="检查间隔">
                  <el-input-number v-model="config.health_check.interval" :min="1" controls-position="right" />
                  <div class="form-tip">秒</div>
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="超时时间">
                  <el-input-number v-model="config.health_check.timeout" :min="1" controls-position="right" />
                  <div class="form-tip">秒</div>
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="重试次数">
                  <el-input-number v-model="config.health_check.retries" :min="1" controls-position="right" />
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="启动等待期">
                  <el-input-number v-model="config.health_check.start_period" :min="0" controls-position="right" />
                  <div class="form-tip">秒</div>
                </el-form-item>
              </el-col>
            </el-row>
          </template>
        </el-form>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'
import DockerPortsConfig from './DockerPortsConfig.vue'
import DockerVolumesConfig from './DockerVolumesConfig.vue'
import DockerNetworkConfig from './DockerNetworkConfig.vue'
import DockerSecurityConfig from './DockerSecurityConfig.vue'
import DockerResourceConfig from './DockerResourceConfig.vue'
import DockerDeviceConfig from './DockerDeviceConfig.vue'
import DockerLogConfig from './DockerLogConfig.vue'
import DockerRuntimeConfig from './DockerRuntimeConfig.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  initialConfig: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue', 'save'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const activeTab = ref('basic')

// 主配置对象
const config = reactive({
  // 镜像配置
  image: '',
  registry: '',
  registry_user: '',
  registry_pass: '',
  image_pull_policy: '',
  platform: '',

  // 容器基础配置
  container_name: '',
  hostname: '',
  domainname: '',
  user: '',
  group_add: [],
  working_dir: '',
  environment: {},
  labels: {},
  command: [],
  entrypoint: [],

  // 端口配置
  ports: [],
  expose_ports: [],

  // 网络配置
  network_mode: '',
  networks: [],
  dns: [],
  dns_search: [],
  dns_opt: [],
  extra_hosts: [],
  mac_address: '',
  ipv4_address: '',
  ipv6_address: '',
  links: [],

  // 存储配置
  volumes: [],
  tmpfs_mounts: [],
  volume_driver: '',
  storage_opts: {},

  // 安全配置
  privileged: false,
  cap_add: [],
  cap_drop: [],
  security_opt: [],
  read_only_rootfs: false,
  no_new_privileges: false,
  userns_mode: '',

  // 设备配置
  devices: [],
  gpus: '',
  device_cgroup_rules: [],

  // 资源限制
  memory_limit: '',
  memory_reserve: '',
  memory_swap: '',
  memory_swappiness: null,
  cpu_limit: '',
  cpu_shares: null,
  cpuset_cpus: '',
  cpuset_mems: '',
  pids_limit: null,
  ulimits: [],
  oom_kill_disable: false,
  oom_score_adj: 0,
  shm_size: '',

  // 运行时配置
  runtime: '',
  init: false,
  pid_mode: '',
  ipc_mode: '',
  uts_mode: '',
  cgroup_parent: '',
  sysctls: {},
  stop_signal: '',
  tty: false,
  stdin_open: false,

  // 日志配置
  log_driver: '',
  log_opts: {},

  // 健康检查
  health_check: {
    command: [],
    interval: 30,
    timeout: 30,
    retries: 3,
    start_period: 0
  },

  // 重启策略
  restart_policy: 'unless-stopped',
  restart_max_retries: 0,

  // 部署策略
  stop_timeout: 10,
  remove_old: true,
  keep_old_count: 0,
  pull_before_stop: true,
  auto_remove: false
})

// 辅助状态
const envList = ref([])
const labelsList = ref([])
const entrypointStr = ref('')
const commandStr = ref('')
const healthCheckEnabled = ref(false)
const healthCheckCommand = ref('')

// 子组件配置对象
const networkConfig = ref({})
const securityConfig = ref({})
const resourceConfig = ref({})
const deviceConfig = ref({})
const logConfig = ref({})
const runtimeConfig = ref({})

// 监听初始配置
watch(() => props.initialConfig, (newVal) => {
  if (newVal && Object.keys(newVal).length > 0) {
    Object.assign(config, newVal)

    // 转换环境变量为列表
    envList.value = Object.entries(config.environment || {}).map(([key, value]) => ({ key, value }))

    // 转换标签为列表
    labelsList.value = Object.entries(config.labels || {}).map(([key, value]) => ({ key, value }))

    // 转换命令为字符串
    entrypointStr.value = (config.entrypoint || []).join(' ')
    commandStr.value = (config.command || []).join(' ')

    // 健康检查
    if (config.health_check?.command?.length > 0) {
      healthCheckEnabled.value = true
      healthCheckCommand.value = config.health_check.command.join(' ')
    }

    // 更新子组件配置
    updateSubConfigs()
  }
}, { immediate: true, deep: true })

function updateSubConfigs() {
  networkConfig.value = {
    network_mode: config.network_mode,
    networks: config.networks,
    hostname: config.hostname,
    domainname: config.domainname,
    mac_address: config.mac_address,
    dns: config.dns,
    dns_search: config.dns_search,
    dns_opt: config.dns_opt,
    extra_hosts: config.extra_hosts,
    ipv4_address: config.ipv4_address,
    ipv6_address: config.ipv6_address,
    links: config.links
  }

  securityConfig.value = {
    privileged: config.privileged,
    read_only_rootfs: config.read_only_rootfs,
    no_new_privileges: config.no_new_privileges,
    cap_add: config.cap_add,
    cap_drop: config.cap_drop,
    security_opt: config.security_opt,
    userns_mode: config.userns_mode
  }

  resourceConfig.value = {
    memory_limit: config.memory_limit,
    memory_reserve: config.memory_reserve,
    memory_swap: config.memory_swap,
    memory_swappiness: config.memory_swappiness,
    cpu_limit: config.cpu_limit,
    cpu_shares: config.cpu_shares,
    cpuset_cpus: config.cpuset_cpus,
    cpuset_mems: config.cpuset_mems,
    pids_limit: config.pids_limit,
    shm_size: config.shm_size,
    oom_kill_disable: config.oom_kill_disable,
    oom_score_adj: config.oom_score_adj,
    ulimits: config.ulimits
  }

  deviceConfig.value = {
    devices: config.devices,
    gpus: config.gpus,
    device_cgroup_rules: config.device_cgroup_rules
  }

  logConfig.value = {
    log_driver: config.log_driver,
    log_opts: config.log_opts
  }

  runtimeConfig.value = {
    runtime: config.runtime,
    init: config.init,
    tty: config.tty,
    stdin_open: config.stdin_open,
    stop_signal: config.stop_signal,
    pid_mode: config.pid_mode,
    ipc_mode: config.ipc_mode,
    uts_mode: config.uts_mode,
    cgroup_parent: config.cgroup_parent,
    sysctls: config.sysctls
  }
}

// 更新子配置到主配置
function updateNetworkConfig(val) {
  Object.assign(config, val)
}

function updateSecurityConfig(val) {
  Object.assign(config, val)
}

function updateResourceConfig(val) {
  Object.assign(config, val)
}

function updateDeviceConfig(val) {
  Object.assign(config, val)
}

function updateLogConfig(val) {
  Object.assign(config, val)
}

function updateRuntimeConfig(val) {
  Object.assign(config, val)
}

// 环境变量操作
function addEnv() {
  envList.value.push({ key: '', value: '' })
  updateEnv()
}

function removeEnv(index) {
  envList.value.splice(index, 1)
  updateEnv()
}

function updateEnv() {
  const env = {}
  envList.value.forEach(item => {
    if (item.key) {
      env[item.key] = item.value
    }
  })
  config.environment = env
}

// 标签操作
function addLabel() {
  labelsList.value.push({ key: '', value: '' })
  updateLabels()
}

function removeLabel(index) {
  labelsList.value.splice(index, 1)
  updateLabels()
}

function updateLabels() {
  const labels = {}
  labelsList.value.forEach(item => {
    if (item.key) {
      labels[item.key] = item.value
    }
  })
  config.labels = labels
}

// 解析命令
function parseEntrypoint() {
  config.entrypoint = entrypointStr.value ? entrypointStr.value.split(/\s+/).filter(Boolean) : []
}

function parseCommand() {
  config.command = commandStr.value ? commandStr.value.split(/\s+/).filter(Boolean) : []
}

function parseHealthCheckCommand() {
  if (healthCheckCommand.value) {
    // 简单解析，移除 CMD 前缀
    let cmd = healthCheckCommand.value.trim()
    if (cmd.startsWith('CMD ')) {
      cmd = cmd.substring(4)
    }
    config.health_check.command = ['CMD-SHELL', cmd]
  } else {
    config.health_check.command = []
  }
}

// 保存
function handleSave() {
  // 确保所有数据都已更新
  parseEntrypoint()
  parseCommand()
  if (healthCheckEnabled.value) {
    parseHealthCheckCommand()
  } else {
    config.health_check = null
  }

  // 清理空值
  const cleanConfig = { ...config }
  Object.keys(cleanConfig).forEach(key => {
    const val = cleanConfig[key]
    if (val === '' || val === null || (Array.isArray(val) && val.length === 0) ||
        (typeof val === 'object' && val !== null && !Array.isArray(val) && Object.keys(val).length === 0)) {
      delete cleanConfig[key]
    }
  })

  emit('save', cleanConfig)
  visible.value = false
}
</script>

<style scoped>
.docker-config-dialog :deep(.el-dialog__body) {
  padding: 0 20px 20px;
  max-height: 70vh;
  overflow-y: auto;
}

.docker-config-dialog :deep(.el-tabs__content) {
  padding: 16px;
  max-height: 55vh;
  overflow-y: auto;
}

.config-form {
  padding: 8px 0;
}

.config-header {
  margin-bottom: 12px;
}

.form-tip {
  font-size: 12px;
  color: var(--tech-text-muted);
  margin-top: 4px;
}

.env-config,
.labels-config {
  padding: 0 12px;
}

:deep(.el-table) {
  background: transparent;
}

:deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  color: var(--tech-text-primary);
  font-weight: 600;
}

:deep(.el-table td) {
  background-color: var(--tech-bg-card);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
