<template>
  <div class="docker-network-config">
    <!-- 网络模式 -->
    <div class="config-section">
      <div class="section-title">网络模式</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="网络模式">
            <el-select v-model="config.network_mode" placeholder="默认 (bridge)" style="width: 100%" @change="emitChange">
              <el-option label="bridge (默认桥接)" value="bridge" />
              <el-option label="host (使用主机网络)" value="host" />
              <el-option label="none (无网络)" value="none" />
              <el-option label="container:name (共享容器网络)" value="container:" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12" v-if="config.network_mode === 'container:'">
          <el-form-item label="共享容器名称">
            <el-input
              v-model="containerNameForNetwork"
              placeholder="容器名称或ID"
              @input="updateNetworkMode"
            />
          </el-form-item>
        </el-col>
      </el-row>

      <!-- 加入的网络 -->
      <el-form-item label="加入网络">
        <el-select
          v-model="config.networks"
          multiple
          filterable
          allow-create
          placeholder="选择或输入网络名称"
          style="width: 100%"
          @change="emitChange"
        >
          <el-option label="bridge" value="bridge" />
          <el-option label="host" value="host" />
        </el-select>
      </el-form-item>
    </div>

    <!-- 主机名和域名 -->
    <div class="config-section">
      <div class="section-title">主机名配置</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="主机名">
            <el-input v-model="config.hostname" placeholder="container-hostname" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="域名">
            <el-input v-model="config.domainname" placeholder="example.com" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="MAC 地址">
            <el-input v-model="config.mac_address" placeholder="02:42:ac:11:00:02" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- DNS 配置 -->
    <div class="config-section">
      <div class="section-title">DNS 配置</div>
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="DNS 服务器">
            <el-select
              v-model="config.dns"
              multiple
              filterable
              allow-create
              placeholder="如 8.8.8.8"
              style="width: 100%"
              @change="emitChange"
            >
              <el-option label="8.8.8.8 (Google)" value="8.8.8.8" />
              <el-option label="8.8.4.4 (Google)" value="8.8.4.4" />
              <el-option label="114.114.114.114 (国内)" value="114.114.114.114" />
              <el-option label="223.5.5.5 (阿里)" value="223.5.5.5" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="DNS 搜索域">
            <el-select
              v-model="config.dns_search"
              multiple
              filterable
              allow-create
              placeholder="如 example.com"
              style="width: 100%"
              @change="emitChange"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="DNS 选项">
            <el-select
              v-model="config.dns_opt"
              multiple
              filterable
              allow-create
              placeholder="如 ndots:5"
              style="width: 100%"
              @change="emitChange"
            />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 额外主机 -->
    <div class="config-section">
      <div class="section-title">额外主机 (extra_hosts)</div>
      <div class="config-header">
        <el-button type="primary" size="small" @click="addExtraHost">
          <el-icon><Plus /></el-icon>
          添加主机
        </el-button>
      </div>

      <el-table :data="extraHostsList" size="small" border v-if="extraHostsList.length > 0">
        <el-table-column label="主机名" min-width="180">
          <template #default="{ row, $index }">
            <el-input
              v-model="row.hostname"
              size="small"
              placeholder="hostname"
              @input="updateExtraHosts"
            />
          </template>
        </el-table-column>
        <el-table-column label="IP 地址" min-width="180">
          <template #default="{ row }">
            <el-input
              v-model="row.ip"
              size="small"
              placeholder="192.168.1.1"
              @input="updateExtraHosts"
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
              @click="removeExtraHost($index)"
            />
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- IP 地址配置 -->
    <div class="config-section">
      <div class="section-title">IP 地址配置</div>
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="IPv4 地址">
            <el-input v-model="config.ipv4_address" placeholder="172.20.0.10" @input="emitChange" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="IPv6 地址">
            <el-input v-model="config.ipv6_address" placeholder="2001:db8::1" @input="emitChange" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <!-- 容器链接 -->
    <div class="config-section">
      <div class="section-title">容器链接 (links)</div>
      <el-select
        v-model="config.links"
        multiple
        filterable
        allow-create
        placeholder="container:alias 格式"
        style="width: 100%"
        @change="emitChange"
      />
      <div class="form-tip">格式: container_name:alias，用于让容器通过别名访问其他容器</div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, watch, computed } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const config = reactive({
  network_mode: '',
  networks: [],
  hostname: '',
  domainname: '',
  mac_address: '',
  dns: [],
  dns_search: [],
  dns_opt: [],
  extra_hosts: [],
  ipv4_address: '',
  ipv6_address: '',
  links: []
})

const containerNameForNetwork = ref('')
const extraHostsList = ref([])

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    Object.assign(config, {
      network_mode: newVal.network_mode || '',
      networks: newVal.networks || [],
      hostname: newVal.hostname || '',
      domainname: newVal.domainname || '',
      mac_address: newVal.mac_address || '',
      dns: newVal.dns || [],
      dns_search: newVal.dns_search || [],
      dns_opt: newVal.dns_opt || [],
      extra_hosts: newVal.extra_hosts || [],
      ipv4_address: newVal.ipv4_address || '',
      ipv6_address: newVal.ipv6_address || '',
      links: newVal.links || []
    })

    // 解析 extra_hosts 到列表
    extraHostsList.value = (newVal.extra_hosts || []).map(h => {
      const [hostname, ip] = h.split(':')
      return { hostname, ip }
    })

    // 解析容器网络模式
    if (newVal.network_mode?.startsWith('container:')) {
      containerNameForNetwork.value = newVal.network_mode.substring(10)
    }
  }
}, { immediate: true, deep: true })

function emitChange() {
  emit('update:modelValue', { ...config })
}

function updateNetworkMode() {
  if (containerNameForNetwork.value) {
    config.network_mode = `container:${containerNameForNetwork.value}`
  } else {
    config.network_mode = 'container:'
  }
  emitChange()
}

function addExtraHost() {
  extraHostsList.value.push({ hostname: '', ip: '' })
  updateExtraHosts()
}

function removeExtraHost(index) {
  extraHostsList.value.splice(index, 1)
  updateExtraHosts()
}

function updateExtraHosts() {
  config.extra_hosts = extraHostsList.value
    .filter(h => h.hostname && h.ip)
    .map(h => `${h.hostname}:${h.ip}`)
  emitChange()
}
</script>

<style scoped>
.docker-network-config {
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
  margin-top: 8px;
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
