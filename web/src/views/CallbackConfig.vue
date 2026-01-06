<template>
  <div class="callback-config-page">
    <el-page-header @back="goBack" title="返回发布管理">
      <template #content>
        <span class="page-title">回调配置管理</span>
      </template>
      <template #extra>
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          新建配置
        </el-button>
      </template>
    </el-page-header>

    <!-- 项目选择 -->
    <el-card class="project-selector" shadow="never">
      <el-select
        v-model="selectedProjectId"
        placeholder="请选择项目"
        @change="loadConfigs"
        style="width: 300px"
      >
        <el-option
          v-for="project in projects"
          :key="project.id"
          :label="project.name"
          :value="project.id"
        />
      </el-select>
    </el-card>

    <!-- 配置列表 -->
    <el-card v-loading="loading" shadow="never">
      <el-empty v-if="!selectedProjectId" description="请先选择项目" />
      <el-empty v-else-if="configs.length === 0" description="暂无回调配置">
        <el-button type="primary" @click="openCreateDialog">创建配置</el-button>
      </el-empty>
      <div v-else class="config-list">
        <div
          v-for="config in configs"
          :key="config.id"
          class="config-item"
          :class="{ 'config-item-disabled': !config.enabled }"
        >
          <div class="config-header">
            <div class="config-title">
              <el-icon class="config-icon" :class="getChannelIconClass(config)">
                <component :is="getChannelIcon(config)" />
              </el-icon>
              <span class="config-name">{{ config.name }}</span>
              <el-tag :type="config.enabled ? 'success' : 'info'" size="small">
                {{ config.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </div>
            <div class="config-actions">
              <el-switch
                v-model="config.enabled"
                @change="toggleEnabled(config)"
                :loading="config._toggling"
              />
              <el-button text :icon="Edit" @click="openEditDialog(config)">编辑</el-button>
              <el-button text :icon="Delete" type="danger" @click="handleDelete(config)">删除</el-button>
            </div>
          </div>

          <!-- 渠道信息 -->
          <div class="config-channels">
            <el-tag
              v-for="channel in config.channels"
              :key="channel.type"
              :type="channel.enabled ? '' : 'info'"
              size="small"
              class="channel-tag"
            >
              {{ getChannelTypeName(channel.type) }}
            </el-tag>
          </div>

          <!-- 事件类型 -->
          <div class="config-events">
            <span class="events-label">触发事件:</span>
            <el-tag
              v-for="event in config.events"
              :key="event"
              size="small"
              effect="plain"
            >
              {{ getEventTypeName(event) }}
            </el-tag>
          </div>

          <!-- 测试按钮 -->
          <div class="config-footer">
            <el-button
              size="small"
              :icon="Connection"
              @click="openTestDialog(config)"
              :disabled="!config.enabled"
            >
              测试回调
            </el-button>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑回调配置' : '创建回调配置'"
      width="700px"
      @close="resetForm"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入配置名称" />
        </el-form-item>

        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" />
        </el-form-item>

        <!-- 渠道配置 -->
        <el-form-item label="回调渠道">
          <div class="channels-config">
            <el-checkbox-group v-model="selectedChannels">
              <el-checkbox label="feishu">飞书</el-checkbox>
              <el-checkbox label="dingtalk">钉钉</el-checkbox>
              <el-checkbox label="wechat">企业微信</el-checkbox>
              <el-checkbox label="custom">自定义接口</el-checkbox>
            </el-checkbox-group>
          </div>
        </el-form-item>

        <!-- 飞书配置 -->
        <template v-if="selectedChannels.includes('feishu')">
          <el-divider content-position="left">飞书配置</el-divider>
          <el-form-item label="Webhook URL" prop="feishu_webhook">
            <el-input v-model="form.feishu.webhook_url" placeholder="请输入飞书 Webhook URL" />
          </el-form-item>
          <el-form-item label="签名密钥">
            <el-input v-model="form.feishu.sign_secret" type="password" placeholder="选填，用于验证签名" show-password />
          </el-form-item>
        </template>

        <!-- 钉钉配置 -->
        <template v-if="selectedChannels.includes('dingtalk')">
          <el-divider content-position="left">钉钉配置</el-divider>
          <el-form-item label="Webhook URL" prop="dingtalk_webhook">
            <el-input v-model="form.dingtalk.webhook_url" placeholder="请输入钉钉 Webhook URL" />
          </el-form-item>
          <el-form-item label="签名密钥">
            <el-input v-model="form.dingtalk.sign_secret" type="password" placeholder="选填，用于验证签名" show-password />
          </el-form-item>
        </template>

        <!-- 企业微信配置 -->
        <template v-if="selectedChannels.includes('wechat')">
          <el-divider content-position="left">企业微信配置</el-divider>
          <el-form-item label="企业 ID" prop="wechat_corp_id">
            <el-input v-model="form.wechat.corp_id" placeholder="请输入企业 ID" />
          </el-form-item>
          <el-form-item label="应用 ID" prop="wechat_agent_id">
            <el-input-number v-model="form.wechat.agent_id" :min="0" placeholder="请输入应用 ID" />
          </el-form-item>
          <el-form-item label="应用密钥" prop="wechat_secret">
            <el-input v-model="form.wechat.secret" type="password" placeholder="请输入应用密钥" show-password />
          </el-form-item>
          <el-form-item label="接收用户">
            <el-input v-model="form.wechat.to_user" placeholder="默认 @all，可指定用户 ID 或部门" />
          </el-form-item>
        </template>

        <!-- 自定义接口配置 -->
        <template v-if="selectedChannels.includes('custom')">
          <el-divider content-position="left">自定义接口配置</el-divider>
          <el-form-item label="回调 URL" prop="custom_url">
            <el-input v-model="form.custom.url" placeholder="请输入回调 URL" />
          </el-form-item>
          <el-form-item label="请求方法">
            <el-select v-model="form.custom.method" style="width: 120px">
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
            </el-select>
          </el-form-item>
          <el-form-item label="超时时间(秒)">
            <el-input-number v-model="form.custom.timeout" :min="1" :max="300" />
          </el-form-item>
          <el-form-item label="重试次数">
            <el-input-number v-model="form.custom.retry_count" :min="0" :max="10" />
          </el-form-item>
          <el-form-item label="重试间隔(秒)">
            <el-input-number v-model="form.custom.retry_interval" :min="1" :max="60" />
          </el-form-item>
          <el-form-item label="请求头">
            <el-input
              v-model="customHeadersText"
              type="textarea"
              :rows="3"
              placeholder='每行一个，格式：Key: Value&#10;例如：Authorization: Bearer token'
            />
          </el-form-item>
          <el-form-item label="消息模板">
            <div class="template-editor-wrapper">
              <el-input
                v-model="form.custom.msg_template"
                type="textarea"
                :rows="6"
                placeholder="留空使用默认 JSON 格式，支持模板变量和条件渲染"
              />
              <div class="template-actions">
                <el-button size="small" @click="openTemplateHelp">
                  <el-icon><QuestionFilled /></el-icon>
                  模板语法
                </el-button>
                <el-button size="small" type="primary" @click="openTemplatePreview">
                  <el-icon><View /></el-icon>
                  预览模板
                </el-button>
              </div>
            </div>
          </el-form-item>
        </template>

        <!-- 事件类型 -->
        <el-divider content-position="left">触发事件</el-divider>
        <el-form-item label="事件类型" prop="events">
          <el-checkbox-group v-model="form.events">
            <el-checkbox label="canary_started">金丝雀开始</el-checkbox>
            <el-checkbox label="canary_completed">金丝雀完成</el-checkbox>
            <el-checkbox label="full_completed">全量完成</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ isEdit ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 测试对话框 -->
    <el-dialog
      v-model="testDialogVisible"
      title="测试回调"
      width="500px"
    >
      <el-form label-width="100px">
        <el-form-item label="选择渠道">
          <el-select v-model="testChannelType" placeholder="请选择要测试的渠道">
            <el-option
              v-for="channel in currentConfig?.channels"
              :key="channel.type"
              :label="getChannelTypeName(channel.type)"
              :value="channel.type"
              :disabled="!channel.enabled"
            />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleTest" :loading="testing">
          发送测试
        </el-button>
      </template>
    </el-dialog>

    <!-- 模板预览对话框 -->
    <el-dialog
      v-model="templatePreviewVisible"
      title="模板预览"
      width="900px"
      @open="loadTemplateData"
    >
      <div class="template-preview-container">
        <el-row :gutter="20">
          <!-- 左侧：模板编辑 -->
          <el-col :span="12">
            <div class="template-section">
              <div class="section-header">
                <span class="section-title">模板编辑</span>
                <el-dropdown @command="applyDefaultTemplate">
                  <el-button size="small">
                    使用默认模板 <el-icon><ArrowDown /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="simple">简单模板</el-dropdown-item>
                      <el-dropdown-item command="detailed">详细模板</el-dropdown-item>
                      <el-dropdown-item command="json">JSON 模板</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
              <el-input
                v-model="previewTemplate"
                type="textarea"
                :rows="12"
                placeholder="输入模板内容"
              />
              <div class="template-btn-group">
                <el-button type="primary" @click="handlePreview" :loading="previewing">
                  <el-icon><Refresh /></el-icon>
                  预览渲染结果
                </el-button>
                <el-button @click="handleValidate" :loading="validating">
                  <el-icon><CircleCheck /></el-icon>
                  验证语法
                </el-button>
              </div>
            </div>
          </el-col>

          <!-- 右侧：预览结果 -->
          <el-col :span="12">
            <div class="template-section">
              <div class="section-header">
                <span class="section-title">渲染结果</span>
              </div>
              <div class="preview-result">
                <el-alert
                  v-if="previewError"
                  :title="previewError"
                  type="error"
                  show-icon
                  :closable="false"
                />
                <pre v-else-if="previewResult" class="rendered-preview">{{ previewResult }}</pre>
                <el-empty v-else description="点击预览按钮查看渲染结果" :image-size="100" />
              </div>
              <div v-if="validationResult" class="validation-result">
                <el-alert
                  :title="validationResult.valid ? '模板语法正确' : '模板语法错误'"
                  :type="validationResult.valid ? 'success' : 'error'"
                  show-icon
                  :closable="false"
                >
                  <template #default v-if="validationResult.errors?.length">
                    <ul class="validation-errors">
                      <li v-for="(err, idx) in validationResult.errors" :key="idx">{{ err }}</li>
                    </ul>
                  </template>
                  <template #default v-else-if="validationResult.warnings?.length">
                    <ul class="validation-warnings">
                      <li v-for="(warn, idx) in validationResult.warnings" :key="idx">{{ warn }}</li>
                    </ul>
                  </template>
                </el-alert>
              </div>
            </div>
          </el-col>
        </el-row>

        <!-- 变量参考 -->
        <el-collapse v-model="activeCollapse" class="variables-collapse">
          <el-collapse-item title="可用变量参考" name="variables">
            <el-table :data="templateVariables" size="small" max-height="250">
              <el-table-column prop="name" label="变量名" width="180">
                <template #default="{ row }">
                  <code class="var-name" v-text="`{{${row.name}}}`"></code>
                </template>
              </el-table-column>
              <el-table-column prop="description" label="说明" />
              <el-table-column prop="example" label="示例" width="150" />
            </el-table>
          </el-collapse-item>
          <el-collapse-item title="条件与循环语法" name="syntax">
            <div class="syntax-examples">
              <div v-for="example in templateExamples" :key="example.name" class="syntax-item">
                <div class="syntax-name">{{ example.name }}</div>
                <div class="syntax-desc">{{ example.description }}</div>
                <code class="syntax-code">{{ example.template }}</code>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>

      <template #footer>
        <el-button @click="templatePreviewVisible = false">关闭</el-button>
        <el-button type="primary" @click="applyTemplate">应用到配置</el-button>
      </template>
    </el-dialog>

    <!-- 模板语法帮助对话框 -->
    <el-dialog
      v-model="templateHelpVisible"
      title="模板语法说明"
      width="700px"
    >
      <div class="template-help">
        <h4>基本变量</h4>
        <p>使用 <code v-text="'{{变量名}}'"></code> 语法引用变量，例如：<code v-text="'{{project_name}}'"></code></p>

        <h4>条件渲染</h4>
        <pre><code v-text="'{{#if is_success}}部署成功{{else}}部署失败{{/if}}'"></code></pre>

        <h4>循环渲染</h4>
        <pre><code v-text="'{{#each hosts}}主机: {{this}}\\n{{/each}}'"></code></pre>

        <h4>常用变量</h4>
        <ul>
          <li><code>project_name</code> - 项目名称</li>
          <li><code>version_name</code> - 版本号</li>
          <li><code>environment</code> - 环境名称</li>
          <li><code>task_status</code> - 任务状态</li>
          <li><code>total_count</code> - 总主机数</li>
          <li><code>success_rate</code> - 成功率</li>
          <li><code>is_success</code> - 是否成功</li>
          <li><code>is_failed</code> - 是否失败</li>
        </ul>
      </div>
      <template #footer>
        <el-button type="primary" @click="templateHelpVisible = false">知道了</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Edit, Delete, Connection,
  ChatDotRound, Notification, Message, Link,
  QuestionFilled, View, ArrowDown, Refresh, CircleCheck
} from '@element-plus/icons-vue'
import api from '@/api'

const router = useRouter()

// 数据状态
const loading = ref(false)
const projects = ref([])
const selectedProjectId = ref('')
const configs = ref([])

// 对话框状态
const dialogVisible = ref(false)
const testDialogVisible = ref(false)
const templatePreviewVisible = ref(false)
const templateHelpVisible = ref(false)
const isEdit = ref(false)
const currentConfig = ref(null)
const submitting = ref(false)
const testing = ref(false)
const previewing = ref(false)
const validating = ref(false)
const formRef = ref(null)
const testChannelType = ref('')

// 模板预览相关状态
const previewTemplate = ref('')
const previewResult = ref('')
const previewError = ref('')
const validationResult = ref(null)
const templateVariables = ref([])
const templateExamples = ref([])
const defaultTemplates = ref({})
const activeCollapse = ref(['variables'])

// 表单数据
const selectedChannels = ref([])
const form = reactive({
  name: '',
  enabled: true,
  events: [],
  feishu: {
    webhook_url: '',
    sign_secret: ''
  },
  dingtalk: {
    webhook_url: '',
    sign_secret: ''
  },
  wechat: {
    corp_id: '',
    agent_id: null,
    secret: '',
    to_user: ''
  },
  custom: {
    url: '',
    method: 'POST',
    timeout: 30,
    retry_count: 3,
    retry_interval: 5,
    headers: {},
    msg_template: ''
  }
})

const customHeadersText = computed({
  get: () => {
    if (!form.custom.headers) return ''
    return Object.entries(form.custom.headers)
      .map(([k, v]) => `${k}: ${v}`)
      .join('\n')
  },
  set: (val) => {
    const headers = {}
    val.split('\n').forEach(line => {
      const idx = line.indexOf(':')
      if (idx > 0) {
        const key = line.slice(0, idx).trim()
        const value = line.slice(idx + 1).trim()
        if (key) {
          headers[key] = value
        }
      }
    })
    form.custom.headers = headers
  }
})

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入配置名称', trigger: 'blur' }
  ],
  events: [
    { type: 'array', min: 1, message: '请至少选择一个事件类型', trigger: 'change' }
  ]
}

// 渠道类型名称映射
const channelTypeNames = {
  feishu: '飞书',
  dingtalk: '钉钉',
  wechat: '企业微信',
  custom: '自定义'
}

// 事件类型名称映射
const eventTypeNames = {
  canary_started: '金丝雀开始',
  canary_completed: '金丝雀完成',
  full_completed: '全量完成'
}

// 获取渠道图标
const getChannelIcon = (config) => {
  if (!config.channels || config.channels.length === 0) return Message
  const firstEnabled = config.channels.find(c => c.enabled)
  if (!firstEnabled) return Message
  switch (firstEnabled.type) {
    case 'feishu': return ChatDotRound
    case 'dingtalk': return Notification
    case 'wechat': return Message
    case 'custom': return Link
    default: return Message
  }
}

// 获取渠道图标样式类
const getChannelIconClass = (config) => {
  if (!config.channels || config.channels.length === 0) return ''
  const firstEnabled = config.channels.find(c => c.enabled)
  if (!firstEnabled) return ''
  return `channel-icon-${firstEnabled.type}`
}

// 获取渠道类型名称
const getChannelTypeName = (type) => channelTypeNames[type] || type

// 获取事件类型名称
const getEventTypeName = (event) => eventTypeNames[event] || event

// 返回上一页
const goBack = () => {
  router.push('/release')
}

// 加载项目列表
const loadProjects = async () => {
  try {
    const res = await api.getProjects()
    if (res.success) {
      projects.value = res.projects || []
    }
  } catch (error) {
    console.error('Failed to load projects:', error)
  }
}

// 加载回调配置
const loadConfigs = async () => {
  if (!selectedProjectId.value) return
  loading.value = true
  try {
    const res = await api.getCallbackConfigs(selectedProjectId.value)
    if (res.success) {
      configs.value = res.data || []
    } else {
      configs.value = []
    }
  } catch (error) {
    console.error('Failed to load callback configs:', error)
    configs.value = []
  } finally {
    loading.value = false
  }
}

// 打开创建对话框
const openCreateDialog = () => {
  if (!selectedProjectId.value) {
    ElMessage.warning('请先选择项目')
    return
  }
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

// 打开编辑对话框
const openEditDialog = (config) => {
  isEdit.value = true
  currentConfig.value = config
  resetForm()

  // 填充表单
  form.name = config.name
  form.enabled = config.enabled
  form.events = [...config.events]

  // 填充渠道配置
  selectedChannels.value = []
  config.channels.forEach(channel => {
    if (channel.enabled) {
      selectedChannels.value.push(channel.type)
    }
    // 解析渠道配置
    if (channel.config) {
      switch (channel.type) {
        case 'feishu':
          if (channel.config.webhook_url) form.feishu.webhook_url = channel.config.webhook_url
          if (channel.config.sign_secret) form.feishu.sign_secret = channel.config.sign_secret
          break
        case 'dingtalk':
          if (channel.config.webhook_url) form.dingtalk.webhook_url = channel.config.webhook_url
          if (channel.config.sign_secret) form.dingtalk.sign_secret = channel.config.sign_secret
          break
        case 'wechat':
          if (channel.config.corp_id) form.wechat.corp_id = channel.config.corp_id
          if (channel.config.agent_id) form.wechat.agent_id = channel.config.agent_id
          if (channel.config.secret) form.wechat.secret = channel.config.secret
          if (channel.config.to_user) form.wechat.to_user = channel.config.to_user
          break
        case 'custom':
          if (channel.config.url) form.custom.url = channel.config.url
          if (channel.config.method) form.custom.method = channel.config.method
          if (channel.config.timeout) form.custom.timeout = channel.config.timeout
          if (channel.config.retry_count) form.custom.retry_count = channel.config.retry_count
          if (channel.config.retry_interval) form.custom.retry_interval = channel.config.retry_interval
          if (channel.config.headers) form.custom.headers = channel.config.headers
          if (channel.config.msg_template) form.custom.msg_template = channel.config.msg_template
          break
      }
    }
  })

  dialogVisible.value = true
}

// 重置表单
const resetForm = () => {
  form.name = ''
  form.enabled = true
  form.events = []
  form.feishu = { webhook_url: '', sign_secret: '' }
  form.dingtalk = { webhook_url: '', sign_secret: '' }
  form.wechat = { corp_id: '', agent_id: null, secret: '', to_user: '' }
  form.custom = {
    url: '',
    method: 'POST',
    timeout: 30,
    retry_count: 3,
    retry_interval: 5,
    headers: {},
    msg_template: ''
  }
  selectedChannels.value = []
  formRef.value?.clearValidate()
}

// 构建提交数据
const buildSubmitData = () => {
  const channels = []

  if (selectedChannels.value.includes('feishu')) {
    channels.push({
      type: 'feishu',
      enabled: true,
      config: { ...form.feishu }
    })
  }
  if (selectedChannels.value.includes('dingtalk')) {
    channels.push({
      type: 'dingtalk',
      enabled: true,
      config: { ...form.dingtalk }
    })
  }
  if (selectedChannels.value.includes('wechat')) {
    channels.push({
      type: 'wechat',
      enabled: true,
      config: { ...form.wechat }
    })
  }
  if (selectedChannels.value.includes('custom')) {
    channels.push({
      type: 'custom',
      enabled: true,
      config: { ...form.custom }
    })
  }

  return {
    name: form.name,
    enabled: form.enabled,
    channels: channels,
    events: form.events
  }
}

// 提交表单
const handleSubmit = async () => {
  await formRef.value?.validate()

  if (selectedChannels.value.length === 0) {
    ElMessage.warning('请至少选择一个回调渠道')
    return
  }

  submitting.value = true
  try {
    const data = buildSubmitData()

    if (isEdit.value) {
      await api.updateCallbackConfig(currentConfig.value.id, data)
      ElMessage.success('更新成功')
    } else {
      await api.createCallbackConfig(selectedProjectId.value, data)
      ElMessage.success('创建成功')
    }

    dialogVisible.value = false
    loadConfigs()
  } catch (error) {
    ElMessage.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 切换启用状态
const toggleEnabled = async (config) => {
  config._toggling = true
  try {
    await api.updateCallbackConfig(config.id, {
      ...config,
      channels: config.channels,
      events: config.events,
      enabled: !config.enabled
    })
    config.enabled = !config.enabled
    ElMessage.success('状态已更新')
  } catch (error) {
    ElMessage.error('更新失败')
  } finally {
    config._toggling = false
  }
}

// 删除配置
const handleDelete = async (config) => {
  try {
    await ElMessageBox.confirm(`确定要删除配置 "${config.name}" 吗？`, '确认删除', {
      type: 'warning'
    })
    await api.deleteCallbackConfig(config.id)
    ElMessage.success('删除成功')
    loadConfigs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 打开测试对话框
const openTestDialog = (config) => {
  currentConfig.value = config
  const enabledChannels = config.channels?.filter(c => c.enabled) || []
  if (enabledChannels.length > 0) {
    testChannelType.value = enabledChannels[0].type
  } else {
    testChannelType.value = ''
  }
  testDialogVisible.value = true
}

// 执行测试
const handleTest = async () => {
  if (!testChannelType.value) {
    ElMessage.warning('请选择要测试的渠道')
    return
  }

  testing.value = true
  try {
    const res = await api.testCallbackConfig(currentConfig.value.id, testChannelType.value)
    if (res.success) {
      ElMessage.success('测试回调发送成功')
    } else {
      ElMessage.error(res.message || res.error || '测试失败')
    }
  } catch (error) {
    ElMessage.error('测试失败: ' + (error.message || error))
  } finally {
    testing.value = false
  }
}

// ==================== 模板预览功能 ====================

// 打开模板帮助
const openTemplateHelp = () => {
  templateHelpVisible.value = true
}

// 打开模板预览
const openTemplatePreview = () => {
  previewTemplate.value = form.custom.msg_template || ''
  previewResult.value = ''
  previewError.value = ''
  validationResult.value = null
  templatePreviewVisible.value = true
}

// 加载模板数据（变量列表和默认模板）
const loadTemplateData = async () => {
  try {
    // 并行加载变量列表和默认模板
    const [varsRes, defaultsRes] = await Promise.all([
      api.getCallbackTemplateVariables(),
      api.getDefaultCallbackTemplates()
    ])

    if (varsRes.success && varsRes.data) {
      templateVariables.value = varsRes.data.variables || []
      templateExamples.value = varsRes.data.examples || []
    }

    if (defaultsRes.success && defaultsRes.data) {
      defaultTemplates.value = defaultsRes.data
    }
  } catch (error) {
    console.error('Failed to load template data:', error)
  }
}

// 应用默认模板
const applyDefaultTemplate = (type) => {
  if (defaultTemplates.value[type]) {
    previewTemplate.value = defaultTemplates.value[type]
    previewResult.value = ''
    previewError.value = ''
    validationResult.value = null
  }
}

// 预览模板
const handlePreview = async () => {
  if (!previewTemplate.value.trim()) {
    ElMessage.warning('请输入模板内容')
    return
  }

  previewing.value = true
  previewError.value = ''

  try {
    const res = await api.previewCallbackTemplate(previewTemplate.value)
    if (res.success) {
      previewResult.value = res.data.rendered
      validationResult.value = res.data.validation
    } else {
      previewError.value = res.error || '预览失败'
      if (res.data?.validation) {
        validationResult.value = res.data.validation
      }
    }
  } catch (error) {
    previewError.value = error.message || '预览请求失败'
  } finally {
    previewing.value = false
  }
}

// 验证模板
const handleValidate = async () => {
  if (!previewTemplate.value.trim()) {
    ElMessage.warning('请输入模板内容')
    return
  }

  validating.value = true
  try {
    const res = await api.validateCallbackTemplate(previewTemplate.value)
    if (res.success) {
      validationResult.value = res.data
      if (res.data.valid) {
        ElMessage.success('模板语法正确')
      } else {
        ElMessage.error('模板语法错误')
      }
    }
  } catch (error) {
    ElMessage.error('验证请求失败: ' + error.message)
  } finally {
    validating.value = false
  }
}

// 应用模板到配置
const applyTemplate = () => {
  form.custom.msg_template = previewTemplate.value
  templatePreviewVisible.value = false
  ElMessage.success('模板已应用到配置')
}

onMounted(() => {
  loadProjects()
  // 从路由参数获取项目 ID
  const projectId = router.currentRoute.value.query.project_id
  if (projectId) {
    selectedProjectId.value = projectId
    loadConfigs()
  }
})
</script>

<style scoped>
.callback-config-page {
  padding: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
}

.project-selector {
  margin: 20px 0;
}

.config-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.config-item {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s;
}

.config-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.config-item-disabled {
  opacity: 0.6;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.config-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.config-icon {
  font-size: 20px;
}

.config-icon-feishu {
  color: #00d6b9;
}

.config-icon-dingtalk {
  color: #0089ff;
}

.config-icon-wechat {
  color: #07c160;
}

.config-icon-custom {
  color: #909399;
}

.config-name {
  font-size: 16px;
  font-weight: 500;
}

.config-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.config-channels {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.channel-tag {
  margin: 0;
}

.config-events {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.events-label {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.config-footer {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.channels-config {
  width: 100%;
}

.el-checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

:deep(.el-divider__text) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

/* 模板编辑器样式 */
.template-editor-wrapper {
  width: 100%;
}

.template-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

/* 模板预览对话框样式 */
.template-preview-container {
  min-height: 400px;
}

.template-section {
  height: 100%;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.section-title {
  font-weight: 600;
  font-size: 14px;
}

.template-btn-group {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.preview-result {
  min-height: 260px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 12px;
  background: var(--el-fill-color-lighter);
}

.rendered-preview {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.5;
}

.validation-result {
  margin-top: 12px;
}

.validation-errors,
.validation-warnings {
  margin: 8px 0 0 0;
  padding-left: 20px;
}

.validation-errors li {
  color: var(--el-color-danger);
}

.validation-warnings li {
  color: var(--el-color-warning);
}

.variables-collapse {
  margin-top: 20px;
}

.var-name {
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.syntax-examples {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.syntax-item {
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.syntax-name {
  font-weight: 600;
  margin-bottom: 4px;
}

.syntax-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.syntax-code {
  display: block;
  background: var(--el-fill-color);
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
  white-space: pre-wrap;
}

/* 模板帮助对话框样式 */
.template-help h4 {
  margin: 16px 0 8px 0;
}

.template-help h4:first-child {
  margin-top: 0;
}

.template-help pre {
  background: var(--el-fill-color-lighter);
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
}

.template-help code {
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}

.template-help ul {
  padding-left: 20px;
}

.template-help li {
  margin: 4px 0;
}
</style>
