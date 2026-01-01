<template>
  <div class="setup-container">
    <div class="setup-card">
      <div class="setup-header">
        <div class="logo">
          <el-icon :size="48" color="#409eff"><Coin /></el-icon>
        </div>
        <h1>QUIC Flow 初始化向导</h1>
        <p class="subtitle">配置数据库连接以启用发布管理功能</p>
      </div>

      <!-- 步骤指示器 -->
      <el-steps :active="currentStep" finish-status="success" align-center class="setup-steps">
        <el-step title="数据库配置" description="填写连接信息" />
        <el-step title="连接测试" description="验证数据库连接" />
        <el-step title="初始化" description="创建数据表" />
        <el-step title="完成" description="开始使用" />
      </el-steps>

      <!-- 步骤 1: 数据库配置 -->
      <div v-show="currentStep === 0" class="step-content">
        <el-form
          ref="configFormRef"
          :model="dbConfig"
          :rules="configRules"
          label-width="120px"
          class="config-form"
        >
          <el-form-item label="数据库类型">
            <el-tag type="primary">PostgreSQL</el-tag>
          </el-form-item>

          <el-form-item label="主机地址" prop="host">
            <el-input v-model="dbConfig.host" placeholder="localhost" />
          </el-form-item>

          <el-form-item label="端口" prop="port">
            <el-input-number v-model="dbConfig.port" :min="1" :max="65535" />
          </el-form-item>

          <el-form-item label="用户名" prop="user">
            <el-input v-model="dbConfig.user" placeholder="postgres" />
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              v-model="dbConfig.password"
              type="password"
              placeholder="输入密码"
              show-password
            />
          </el-form-item>

          <el-form-item label="数据库名" prop="dbname">
            <el-input v-model="dbConfig.dbname" placeholder="quic_release" />
          </el-form-item>

          <el-form-item label="SSL 模式" prop="sslmode">
            <el-select v-model="dbConfig.sslmode" style="width: 100%">
              <el-option label="禁用 (disable)" value="disable" />
              <el-option label="要求 (require)" value="require" />
              <el-option label="验证 CA (verify-ca)" value="verify-ca" />
              <el-option label="完全验证 (verify-full)" value="verify-full" />
            </el-select>
          </el-form-item>

          <el-form-item label="自动迁移">
            <el-switch v-model="dbConfig.auto_migrate" />
            <span class="form-tip">启用后将自动创建和更新数据表结构</span>
          </el-form-item>
        </el-form>

        <div class="step-actions">
          <el-button type="primary" @click="goToStep(1)" :loading="testing">
            下一步
            <el-icon class="el-icon--right"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 步骤 2: 连接测试 -->
      <div v-show="currentStep === 1" class="step-content">
        <div class="test-section">
          <div class="test-info">
            <el-descriptions :column="1" border>
              <el-descriptions-item label="主机">{{ dbConfig.host }}:{{ dbConfig.port }}</el-descriptions-item>
              <el-descriptions-item label="用户名">{{ dbConfig.user }}</el-descriptions-item>
              <el-descriptions-item label="数据库">{{ dbConfig.dbname }}</el-descriptions-item>
              <el-descriptions-item label="SSL 模式">{{ dbConfig.sslmode }}</el-descriptions-item>
            </el-descriptions>
          </div>

          <div class="test-result" v-if="testResult !== null">
            <el-result
              v-if="testResult.success"
              icon="success"
              title="连接成功"
              sub-title="数据库连接测试通过，可以继续下一步"
            />
            <el-result
              v-else
              icon="error"
              title="连接失败"
              :sub-title="testResult.error"
            />
          </div>

          <div class="test-actions">
            <el-button @click="testConnection" :loading="testing" type="primary">
              <el-icon><Connection /></el-icon>
              测试连接
            </el-button>
          </div>
        </div>

        <div class="step-actions">
          <el-button @click="goToStep(0)">
            <el-icon class="el-icon--left"><ArrowLeft /></el-icon>
            上一步
          </el-button>
          <el-button
            type="primary"
            @click="goToStep(2)"
            :disabled="!testResult?.success"
          >
            下一步
            <el-icon class="el-icon--right"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 步骤 3: 初始化数据库 -->
      <div v-show="currentStep === 2" class="step-content">
        <div class="init-section">
          <div class="init-info">
            <el-alert
              title="即将创建以下数据表"
              type="info"
              :closable="false"
              show-icon
            >
              <div class="table-list">
                <el-tag v-for="table in tables" :key="table" class="table-tag">
                  {{ table }}
                </el-tag>
              </div>
            </el-alert>
          </div>

          <div class="init-progress" v-if="initializing">
            <el-progress
              :percentage="initProgress"
              :status="initStatus"
              :stroke-width="20"
              striped
              striped-flow
            />
            <p class="init-message">{{ initMessage }}</p>
          </div>

          <div class="init-result" v-if="initResult !== null">
            <el-result
              v-if="initResult.success"
              icon="success"
              title="初始化成功"
              sub-title="数据库表结构已创建完成"
            />
            <el-result
              v-else
              icon="error"
              title="初始化失败"
              :sub-title="initResult.error"
            />
          </div>

          <div class="init-actions" v-if="!initializing && !initResult?.success">
            <el-button type="primary" @click="initializeDatabase" size="large">
              <el-icon><SetUp /></el-icon>
              开始初始化
            </el-button>
          </div>
        </div>

        <div class="step-actions">
          <el-button @click="goToStep(1)" :disabled="initializing">
            <el-icon class="el-icon--left"><ArrowLeft /></el-icon>
            上一步
          </el-button>
          <el-button
            type="primary"
            @click="goToStep(3)"
            :disabled="!initResult?.success"
          >
            下一步
            <el-icon class="el-icon--right"><ArrowRight /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 步骤 4: 完成 -->
      <div v-show="currentStep === 3" class="step-content">
        <div class="complete-section">
          <el-result
            icon="success"
            title="配置完成"
            sub-title="数据库已成功初始化，您现在可以开始使用发布管理功能"
          >
            <template #extra>
              <div class="complete-actions">
                <el-button type="primary" size="large" @click="goToMain">
                  <el-icon><House /></el-icon>
                  进入系统
                </el-button>
                <el-button size="large" @click="goToRelease">
                  <el-icon><Upload /></el-icon>
                  发布管理
                </el-button>
              </div>
            </template>
          </el-result>

          <div class="config-saved">
            <el-alert
              title="配置已保存"
              type="success"
              :closable="false"
              show-icon
            >
              数据库配置已保存到配置文件，下次启动将自动连接。
            </el-alert>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  Coin, ArrowRight, ArrowLeft, Connection, SetUp, House, Upload
} from '@element-plus/icons-vue'
import { request } from '@/api'

const router = useRouter()

// 步骤状态
const currentStep = ref(0)
const configFormRef = ref(null)

// 数据库配置
const dbConfig = reactive({
  host: 'localhost',
  port: 5432,
  user: 'postgres',
  password: '',
  dbname: 'quic_release',
  sslmode: 'disable',
  auto_migrate: true
})

// 表单验证规则
const configRules = {
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  user: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  dbname: [{ required: true, message: '请输入数据库名', trigger: 'blur' }]
}

// 测试状态
const testing = ref(false)
const testResult = ref(null)

// 初始化状态
const initializing = ref(false)
const initProgress = ref(0)
const initStatus = ref('')
const initMessage = ref('')
const initResult = ref(null)

// 表列表
const tables = [
  'projects (项目)',
  'environments (环境)',
  'targets (目标)',
  'variables (变量)',
  'pipelines (流水线)',
  'releases (发布记录)',
  'target_installations (安装状态)',
  'release_status_reports (状态上报)',
  'release_approvals (审批记录)',
  'release_service_dependencies (服务依赖)'
]

// 跳转到指定步骤
async function goToStep(step) {
  if (step === 1 && currentStep.value === 0) {
    // 验证表单
    const valid = await configFormRef.value?.validate().catch(() => false)
    if (!valid) return

    // 自动测试连接
    await testConnection()
    if (testResult.value?.success) {
      currentStep.value = step
    }
  } else {
    currentStep.value = step
  }
}

// 测试数据库连接
async function testConnection() {
  testing.value = true
  testResult.value = null

  try {
    const res = await request.post('/setup/test-connection', dbConfig)
    testResult.value = res
    if (res.connected) {
      ElMessage.success('数据库连接成功')
    } else {
      ElMessage.error(res.error || '连接失败')
    }
  } catch (e) {
    testResult.value = { success: false, error: e.message }
    ElMessage.error('连接测试失败: ' + e.message)
  } finally {
    testing.value = false
  }
}

// 初始化数据库
async function initializeDatabase() {
  initializing.value = true
  initProgress.value = 0
  initStatus.value = ''
  initMessage.value = '正在连接数据库...'
  initResult.value = null

  // 模拟进度
  const progressInterval = setInterval(() => {
    if (initProgress.value < 90) {
      initProgress.value += 10
      if (initProgress.value === 30) {
        initMessage.value = '正在创建表结构...'
      } else if (initProgress.value === 60) {
        initMessage.value = '正在创建索引...'
      } else if (initProgress.value === 90) {
        initMessage.value = '正在保存配置...'
      }
    }
  }, 300)

  try {
    const res = await request.post('/setup/initialize', dbConfig)
    clearInterval(progressInterval)

    if (res.success) {
      initProgress.value = 100
      initStatus.value = 'success'
      initMessage.value = '初始化完成!'
      initResult.value = res
      ElMessage.success('数据库初始化成功')

      // 自动跳转到完成步骤
      setTimeout(() => {
        currentStep.value = 3
      }, 1000)
    } else {
      initProgress.value = 100
      initStatus.value = 'exception'
      initMessage.value = res.error
      initResult.value = res
      ElMessage.error('初始化失败: ' + res.error)
    }
  } catch (e) {
    clearInterval(progressInterval)
    initProgress.value = 100
    initStatus.value = 'exception'
    initMessage.value = e.message
    initResult.value = { success: false, error: e.message }
    ElMessage.error('初始化失败: ' + e.message)
  } finally {
    initializing.value = false
  }
}

// 进入主页
function goToMain() {
  router.push('/')
}

// 进入发布管理
function goToRelease() {
  router.push('/release')
}
</script>

<style scoped>
.setup-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
}

.setup-card {
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  width: 100%;
  max-width: 800px;
  padding: 40px;
}

.setup-header {
  text-align: center;
  margin-bottom: 40px;
}

.logo {
  margin-bottom: 16px;
}

.setup-header h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  color: #303133;
}

.subtitle {
  margin: 0;
  color: #909399;
  font-size: 14px;
}

.setup-steps {
  margin-bottom: 40px;
}

.step-content {
  min-height: 300px;
}

.config-form {
  max-width: 500px;
  margin: 0 auto;
}

.form-tip {
  margin-left: 12px;
  font-size: 12px;
  color: #909399;
}

.step-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}

.test-section,
.init-section,
.complete-section {
  max-width: 500px;
  margin: 0 auto;
}

.test-info {
  margin-bottom: 24px;
}

.test-result {
  margin: 24px 0;
}

.test-actions {
  text-align: center;
}

.table-list {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.table-tag {
  margin: 0;
}

.init-progress {
  margin: 32px 0;
  text-align: center;
}

.init-message {
  margin-top: 12px;
  color: #606266;
}

.init-result {
  margin: 24px 0;
}

.init-actions {
  text-align: center;
  margin: 32px 0;
}

.complete-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.config-saved {
  margin-top: 24px;
}
</style>
