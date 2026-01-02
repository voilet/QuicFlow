<template>
  <div class="docker-volumes-config">
    <div class="config-header">
      <span class="title">卷挂载</span>
      <el-button type="primary" size="small" @click="addVolume">
        <el-icon><Plus /></el-icon>
        添加卷
      </el-button>
    </div>

    <el-table :data="modelValue" size="small" border v-if="modelValue?.length > 0">
      <el-table-column label="类型" width="100">
        <template #default="{ row }">
          <el-select v-model="row.type" size="small" @change="emitChange">
            <el-option label="绑定挂载" value="bind" />
            <el-option label="卷" value="volume" />
            <el-option label="tmpfs" value="tmpfs" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="主机路径/卷名" min-width="180">
        <template #default="{ row }">
          <el-input
            v-model="row.host_path"
            size="small"
            :placeholder="row.type === 'volume' ? '卷名称' : '/host/path'"
            @input="emitChange"
          />
        </template>
      </el-table-column>
      <el-table-column label="容器路径" min-width="180">
        <template #default="{ row }">
          <el-input
            v-model="row.container_path"
            size="small"
            placeholder="/container/path"
            @input="emitChange"
          />
        </template>
      </el-table-column>
      <el-table-column label="只读" width="80" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.read_only" @change="emitChange" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="80" align="center">
        <template #default="{ $index }">
          <el-button
            type="danger"
            size="small"
            :icon="Delete"
            circle
            @click="removeVolume($index)"
          />
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-else description="暂无卷挂载" :image-size="60">
      <el-button type="primary" size="small" @click="addVolume">添加卷挂载</el-button>
    </el-empty>

    <!-- tmpfs 挂载 -->
    <template v-if="showTmpfs">
      <div class="config-header mt-16">
        <span class="title">Tmpfs 挂载</span>
        <el-button type="primary" size="small" @click="addTmpfs">
          <el-icon><Plus /></el-icon>
          添加 Tmpfs
        </el-button>
      </div>

      <el-table :data="tmpfsMounts" size="small" border v-if="tmpfsMounts?.length > 0">
        <el-table-column label="容器路径" min-width="200">
          <template #default="{ row }">
            <el-input
              v-model="row.container_path"
              size="small"
              placeholder="/run/secrets"
              @input="emitTmpfsChange"
            />
          </template>
        </el-table-column>
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            <el-input
              v-model="row.size"
              size="small"
              placeholder="64m"
              @input="emitTmpfsChange"
            />
          </template>
        </el-table-column>
        <el-table-column label="权限" width="100">
          <template #default="{ row }">
            <el-input-number
              v-model="row.mode"
              size="small"
              :min="0"
              :max="7777"
              controls-position="right"
              @change="emitTmpfsChange"
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
              @click="removeTmpfs($index)"
            />
          </template>
        </el-table-column>
      </el-table>
    </template>
  </div>
</template>

<script setup>
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  },
  tmpfsMounts: {
    type: Array,
    default: () => []
  },
  showTmpfs: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['update:modelValue', 'update:tmpfsMounts'])

function addVolume() {
  const volumes = [...(props.modelValue || [])]
  volumes.push({
    type: 'bind',
    host_path: '',
    container_path: '',
    read_only: false
  })
  emit('update:modelValue', volumes)
}

function removeVolume(index) {
  const volumes = [...props.modelValue]
  volumes.splice(index, 1)
  emit('update:modelValue', volumes)
}

function emitChange() {
  emit('update:modelValue', [...props.modelValue])
}

function addTmpfs() {
  const mounts = [...(props.tmpfsMounts || [])]
  mounts.push({
    container_path: '',
    size: '64m',
    mode: 1777
  })
  emit('update:tmpfsMounts', mounts)
}

function removeTmpfs(index) {
  const mounts = [...props.tmpfsMounts]
  mounts.splice(index, 1)
  emit('update:tmpfsMounts', mounts)
}

function emitTmpfsChange() {
  emit('update:tmpfsMounts', [...props.tmpfsMounts])
}
</script>

<style scoped>
.docker-volumes-config {
  padding: 12px 0;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.config-header.mt-16 {
  margin-top: 16px;
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
