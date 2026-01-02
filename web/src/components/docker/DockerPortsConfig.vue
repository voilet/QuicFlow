<template>
  <div class="docker-ports-config">
    <div class="config-header">
      <span class="title">端口映射</span>
      <el-button type="primary" size="small" @click="addPort">
        <el-icon><Plus /></el-icon>
        添加端口
      </el-button>
    </div>

    <el-table :data="modelValue" size="small" border v-if="modelValue?.length > 0">
      <el-table-column label="主机端口" min-width="120">
        <template #default="{ row, $index }">
          <el-input-number
            v-model="row.host_port"
            :min="1"
            :max="65535"
            size="small"
            controls-position="right"
            @change="emitChange"
          />
        </template>
      </el-table-column>
      <el-table-column label="容器端口" min-width="120">
        <template #default="{ row }">
          <el-input-number
            v-model="row.container_port"
            :min="1"
            :max="65535"
            size="small"
            controls-position="right"
            @change="emitChange"
          />
        </template>
      </el-table-column>
      <el-table-column label="协议" width="100">
        <template #default="{ row }">
          <el-select v-model="row.protocol" size="small" @change="emitChange">
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="绑定IP" min-width="140">
        <template #default="{ row }">
          <el-input
            v-model="row.host_ip"
            size="small"
            placeholder="0.0.0.0"
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
            @click="removePort($index)"
          />
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-else description="暂无端口映射" :image-size="60">
      <el-button type="primary" size="small" @click="addPort">添加端口映射</el-button>
    </el-empty>
  </div>
</template>

<script setup>
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

function addPort() {
  const ports = [...(props.modelValue || [])]
  ports.push({
    host_port: 8080,
    container_port: 80,
    protocol: 'tcp',
    host_ip: ''
  })
  emit('update:modelValue', ports)
}

function removePort(index) {
  const ports = [...props.modelValue]
  ports.splice(index, 1)
  emit('update:modelValue', ports)
}

function emitChange() {
  emit('update:modelValue', [...props.modelValue])
}
</script>

<style scoped>
.docker-ports-config {
  padding: 12px 0;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.title {
  font-weight: 600;
  font-size: 14px;
  color: var(--tech-text-primary);
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
</style>
