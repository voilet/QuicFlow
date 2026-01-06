<template>
  <div class="release-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <h2>发布管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateProject">
          <el-icon><Plus /></el-icon>
          新建项目
        </el-button>
        <el-button @click="loadData" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 主要内容 -->
    <el-row :gutter="20">
      <!-- 左侧：项目列表 -->
      <el-col :span="6">
        <el-card shadow="never" class="project-card">
          <template #header>
            <div class="card-header">
              <span>项目列表</span>
            </div>
          </template>

          <div class="project-list" v-loading="loading">
            <div
              v-for="project in projects"
              :key="project.id"
              class="project-item"
              :class="{ active: selectedProject?.id === project.id }"
              @click="selectProject(project)"
            >
              <div class="project-info">
                <div class="project-name">{{ project.name }}</div>
                <div class="project-meta">
                  <el-tag size="small" :type="getProjectTypeTag(project.type)">
                    {{ getProjectTypeLabel(project.type) }}
                  </el-tag>
                  <span class="version-count">{{ project.version_count || 0 }} 个版本</span>
                </div>
              </div>
              <el-dropdown @command="handleProjectAction($event, project)" @click.stop>
                <el-icon class="more-icon"><MoreFilled /></el-icon>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="edit">编辑</el-dropdown-item>
                    <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>

            <el-empty v-if="projects.length === 0" description="暂无项目" :image-size="60" />
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：项目详情 -->
      <el-col :span="18">
        <!-- 未选择项目：显示全局统计 -->
        <el-card v-if="!selectedProject" shadow="never" class="overview-card">
          <template #header>
            <div class="card-header">
              <span>发布总览</span>
            </div>
          </template>

          <!-- 全局统计卡片 -->
          <div class="global-stats" v-loading="loadingGlobalStats">
            <el-row :gutter="20">
              <el-col :span="6">
                <div class="stat-card large">
                  <div class="stat-value">{{ globalDeployStats?.total_count || 0 }}</div>
                  <div class="stat-label">总部署次数</div>
                </div>
              </el-col>
              <el-col :span="6">
                <div class="stat-card large success">
                  <div class="stat-value">{{ globalDeployStats?.success_count || 0 }}</div>
                  <div class="stat-label">成功次数</div>
                </div>
              </el-col>
              <el-col :span="6">
                <div class="stat-card large danger">
                  <div class="stat-value">{{ globalDeployStats?.failed_count || 0 }}</div>
                  <div class="stat-label">失败次数</div>
                </div>
              </el-col>
              <el-col :span="6">
                <div class="stat-card large primary">
                  <div class="stat-value">{{ globalDeployStats?.success_rate?.toFixed(1) || 0 }}%</div>
                  <div class="stat-label">成功率</div>
                </div>
              </el-col>
            </el-row>

            <!-- 额外统计信息 -->
            <el-row :gutter="20" class="mt-20">
              <el-col :span="8">
                <div class="stat-card">
                  <div class="stat-value">{{ projects.length }}</div>
                  <div class="stat-label">项目数量</div>
                </div>
              </el-col>
              <el-col :span="8">
                <div class="stat-card">
                  <div class="stat-value">{{ globalDeployStats?.running_count || 0 }}</div>
                  <div class="stat-label">执行中任务</div>
                </div>
              </el-col>
              <el-col :span="8">
                <div class="stat-card">
                  <div class="stat-value">{{ globalDeployStats?.version_count || 0 }}</div>
                  <div class="stat-label">总版本数</div>
                </div>
              </el-col>
            </el-row>
          </div>

          <!-- 最近部署日志 -->
          <div class="recent-logs" v-if="globalRecentLogs.length > 0">
            <div class="recent-logs-header">
              <h4>最近部署记录</h4>
              <el-tag type="info" size="small">{{ globalRecentLogs.length }} 条记录</el-tag>
            </div>
            <div class="recent-logs-table-wrapper">
              <el-table 
                :data="globalRecentLogs" 
                size="default" 
                stripe 
                max-height="300"
                class="recent-logs-table"
                table-layout="auto"
              >
                <el-table-column prop="project_name" label="项目" min-width="120" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span class="table-cell-text">{{ row.project_name }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="client_id" label="客户端" min-width="120" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span class="table-cell-text">{{ row.client_id }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="version" label="版本" min-width="100">
                  <template #default="{ row }">
                    <el-tag size="small" effect="plain" type="primary">{{ row.version }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="operation" label="操作" min-width="90">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getOperationTag(row.operation)" effect="plain">
                      {{ getOperationLabel(row.operation) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="status" label="结果" min-width="90">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getLogStatusTag(row.status)" effect="plain">
                      {{ getLogStatusLabel(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="started_at" label="时间" min-width="160">
                  <template #default="{ row }">
                    <span class="table-cell-time">{{ formatTime(row.started_at) }}</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>

          <el-empty v-else-if="!loadingGlobalStats" description="暂无部署记录，请选择项目开始部署" :image-size="80" />
        </el-card>

        <!-- 已选择项目 -->
        <template v-else>
          <!-- 项目信息 -->
          <el-card shadow="never" class="info-card">
            <template #header>
              <div class="card-header">
                <div class="project-title">
                  <span>{{ selectedProject.name }}</span>
                  <el-tag :type="getProjectTypeTag(selectedProject.type)" class="ml-2">
                    {{ getProjectTypeLabel(selectedProject.type) }}
                  </el-tag>
                </div>
                <el-button type="primary" size="small" @click="showCreateVersion">
                  <el-icon><Plus /></el-icon>
                  新建版本
                </el-button>
              </div>
            </template>
            <p class="project-desc">{{ selectedProject.description || '暂无描述' }}</p>
          </el-card>

          <!-- 选项卡 -->
          <el-card shadow="never" class="main-card">
            <el-tabs v-model="activeTab">
              <!-- 版本管理 -->
              <el-tab-pane label="版本管理" name="versions">
                <el-table :data="versions" v-loading="loadingVersions" stripe>
                  <el-table-column prop="version" label="版本号" width="120">
                    <template #default="{ row }">
                      <el-tag>{{ row.version }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="status" label="状态" width="100">
                    <template #default="{ row }">
                      <el-tag size="small" :type="getVersionStatusTag(row.status)">
                        {{ getVersionStatusLabel(row.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="deploy_count" label="部署数" width="80" />
                  <el-table-column prop="created_at" label="创建时间" width="180">
                    <template #default="{ row }">
                      {{ formatTime(row.created_at) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="description" label="说明" show-overflow-tooltip />
                  <el-table-column label="操作" width="200" fixed="right">
                    <template #default="{ row }">
                      <el-button type="primary" size="small" @click="showCreateTask(row)">
                        部署
                      </el-button>
                      <el-button size="small" @click="viewVersion(row)">详情</el-button>
                      <el-button size="small" type="danger" @click="deleteVersion(row)">删除</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </el-tab-pane>

              <!-- 部署任务 -->
              <el-tab-pane label="部署任务" name="tasks">
                <div class="tab-header">
                  <el-radio-group v-model="taskStatusFilter" size="default" class="task-filter-group">
                    <el-radio-button value="">全部</el-radio-button>
                    <el-radio-button value="pending">待执行</el-radio-button>
                    <el-radio-button value="running">执行中</el-radio-button>
                    <el-radio-button value="canary">金丝雀中</el-radio-button>
                    <el-radio-button value="completed">已完成</el-radio-button>
                  </el-radio-group>
                  <div class="tab-header-right">
                    <el-tag type="info" size="small" class="task-count-tag">
                      {{ filteredTasks.length }} 个任务
                    </el-tag>
                    <el-button size="small" @click="loadTasks" :loading="loadingTasks">
                      <el-icon><Refresh /></el-icon>
                      刷新
                    </el-button>
                  </div>
                </div>

                <div class="tasks-table-wrapper">
                  <el-table 
                    :data="filteredTasks" 
                    v-loading="loadingTasks" 
                    stripe
                    class="tasks-table"
                    table-layout="auto"
                  >
                    <el-table-column prop="version" label="版本" min-width="100">
                      <template #default="{ row }">
                        <el-tag size="small" effect="plain" type="primary">{{ row.version }}</el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="operation" label="操作" min-width="90">
                      <template #default="{ row }">
                        <el-tag size="small" :type="getOperationTag(row.operation)" effect="plain">
                          {{ getOperationLabel(row.operation) }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="status" label="状态" min-width="100">
                      <template #default="{ row }">
                        <el-tag size="small" :type="getTaskStatusTag(row.status)" effect="plain">
                          {{ getTaskStatusLabel(row.status) }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column label="进度" min-width="200">
                      <template #default="{ row }">
                        <div class="progress-cell">
                          <el-progress
                            :percentage="getTaskProgress(row)"
                            :status="getProgressStatus(row.status)"
                            :stroke-width="8"
                            class="task-progress-bar"
                          />
                          <span class="progress-text">{{ row.success_count || 0 }}/{{ row.total_count || 0 }}</span>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column prop="schedule_type" label="执行方式" min-width="110">
                      <template #default="{ row }">
                        <span class="table-cell-text">
                          {{ row.schedule_type === 'immediate' ? '立即执行' : '定时执行' }}
                        </span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="canary_enabled" label="金丝雀" min-width="90">
                      <template #default="{ row }">
                        <el-tag v-if="row.canary_enabled" size="small" type="warning" effect="plain">
                          {{ row.canary_percent }}%
                        </el-tag>
                        <span v-else class="table-cell-empty">-</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" min-width="280" fixed="right">
                      <template #default="{ row }">
                        <div class="task-actions">
                          <template v-if="row.status === 'pending'">
                            <el-button type="primary" size="small" @click="startTask(row)">开始</el-button>
                            <el-button size="small" @click="cancelTask(row)">取消</el-button>
                          </template>
                          <template v-else-if="row.status === 'canary'">
                            <el-button type="success" size="small" @click="promoteTask(row)">全量发布</el-button>
                            <el-button type="danger" size="small" @click="rollbackTask(row)">回滚</el-button>
                          </template>
                          <template v-else-if="row.status === 'running'">
                            <el-button type="warning" size="small" @click="pauseTask(row)">暂停</el-button>
                          </template>
                          <el-button size="small" @click="viewTask(row)">详情</el-button>
                          <!-- 实时部署日志按钮（所有部署类型都可用） -->
                          <el-button
                            size="small"
                            :type="row.status === 'running' || row.status === 'canary' ? 'primary' : 'info'"
                            @click="showDeployLogs(row)"
                          >
                            <el-icon><Document /></el-icon>
                            {{ row.status === 'running' || row.status === 'canary' ? '实时日志' : '执行日志' }}
                          </el-button>
                          <!-- 容器日志按钮（仅容器项目） -->
                          <el-button
                            v-if="selectedProject?.type === 'container' && (row.status === 'completed' || row.status === 'failed')"
                            size="small"
                            type="success"
                            @click="showTaskLogsDialog(row)"
                          >
                            <el-icon><Document /></el-icon>
                            容器日志
                          </el-button>
                        </div>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </el-tab-pane>

              <!-- 部署记录 -->
              <el-tab-pane label="部署日志" name="history">
                <!-- 头部操作栏 -->
                <div class="tab-header" style="margin-bottom: 16px;">
                  <div class="tab-header-left">
                    <el-tag type="info" size="small">{{ deployLogs.length }} 条记录</el-tag>
                  </div>
                  <div class="tab-header-right">
                    <el-button size="small" @click="loadDeployLogs(); loadDeployStats()" :loading="loadingLogs">
                      <el-icon><Refresh /></el-icon>
                      刷新
                    </el-button>
                  </div>
                </div>

                <!-- 统计信息 -->
                <div class="stats-summary" v-if="deployStats">
                  <el-row :gutter="20">
                    <el-col :span="6">
                      <div class="stat-card">
                        <div class="stat-value">{{ deployStats.total_count }}</div>
                        <div class="stat-label">总部署次数</div>
                      </div>
                    </el-col>
                    <el-col :span="6">
                      <div class="stat-card success">
                        <div class="stat-value">{{ deployStats.success_count }}</div>
                        <div class="stat-label">成功次数</div>
                      </div>
                    </el-col>
                    <el-col :span="6">
                      <div class="stat-card danger">
                        <div class="stat-value">{{ deployStats.failed_count }}</div>
                        <div class="stat-label">失败次数</div>
                      </div>
                    </el-col>
                    <el-col :span="6">
                      <div class="stat-card primary">
                        <div class="stat-value">{{ deployStats.success_rate?.toFixed(1) || 0 }}%</div>
                        <div class="stat-label">成功率</div>
                      </div>
                    </el-col>
                  </el-row>
                </div>

                <!-- 日志列表 -->
                <div class="deploy-logs-table-wrapper">
                  <el-table 
                    :data="deployLogs" 
                    v-loading="loadingLogs" 
                    stripe
                    class="deploy-logs-table"
                    table-layout="auto"
                  >
                    <el-table-column prop="client_id" label="客户端" min-width="120" show-overflow-tooltip>
                      <template #default="{ row }">
                        <span class="table-cell-text">{{ row.client_id }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="version" label="版本" min-width="100">
                      <template #default="{ row }">
                        <el-tag size="small" effect="plain" type="primary">{{ row.version }}</el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="operation" label="操作" min-width="90">
                      <template #default="{ row }">
                        <el-tag size="small" :type="getOperationTag(row.operation)" effect="plain">
                          {{ getOperationLabel(row.operation) }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="status" label="结果" min-width="90">
                      <template #default="{ row }">
                        <el-tag size="small" :type="getLogStatusTag(row.status)" effect="plain">
                          {{ getLogStatusLabel(row.status) }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="is_canary" label="金丝雀" min-width="80">
                      <template #default="{ row }">
                        <el-tag v-if="row.is_canary" size="small" type="warning" effect="plain">是</el-tag>
                        <span v-else class="table-cell-empty">-</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="duration" label="耗时" min-width="90">
                      <template #default="{ row }">
                        <span class="table-cell-time">{{ formatDuration(row.duration) }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="started_at" label="执行时间" min-width="160">
                      <template #default="{ row }">
                        <span class="table-cell-time">{{ formatTime(row.started_at) }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" min-width="100" fixed="right">
                      <template #default="{ row }">
                        <el-button size="small" @click="viewLog(row)">详情</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </el-tab-pane>
            </el-tabs>
          </el-card>
        </template>
      </el-col>
    </el-row>

    <!-- 新建/编辑项目对话框 -->
    <el-dialog v-model="projectDialogVisible" :title="editingProject ? '编辑项目' : '新建项目'" width="600px" destroy-on-close>
      <el-form :model="projectForm" :rules="projectRules" ref="projectFormRef" label-width="100px">
        <el-form-item label="项目名称" prop="name">
          <el-input v-model="projectForm.name" placeholder="输入项目名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="projectForm.description" type="textarea" rows="2" placeholder="项目描述（可选）" />
        </el-form-item>
        <el-form-item label="部署类型" prop="type">
          <el-radio-group v-model="projectForm.type">
            <el-radio value="script">脚本部署</el-radio>
            <el-radio value="container">容器部署</el-radio>
            <el-radio value="gitpull">Git 拉取</el-radio>
            <el-radio value="kubernetes">Kubernetes</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- Git 拉取配置 -->
        <template v-if="projectForm.type === 'gitpull'">
          <el-divider content-position="left">Git 仓库配置</el-divider>
          <el-form-item label="仓库地址" prop="gitpull_config.repo_url">
            <el-input v-model="projectForm.gitpull_config.repo_url" placeholder="https://github.com/user/repo.git" />
          </el-form-item>
          <el-form-item label="默认分支">
            <el-input v-model="projectForm.gitpull_config.branch" placeholder="main" />
          </el-form-item>
          <el-form-item label="工作目录">
            <el-input v-model="projectForm.gitpull_config.work_dir" placeholder="/opt/app" />
          </el-form-item>
          <el-form-item label="认证方式">
            <el-select v-model="projectForm.gitpull_config.auth_type" placeholder="选择认证方式">
              <el-option label="无需认证" value="none" />
              <el-option label="SSH Key" value="ssh" />
              <el-option label="Access Token" value="token" />
              <el-option label="用户名密码" value="basic" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="projectForm.gitpull_config.auth_type === 'token'" label="Access Token">
            <el-input v-model="projectForm.gitpull_config.token" type="password" show-password placeholder="输入 Token" />
          </el-form-item>
          <el-form-item v-if="projectForm.gitpull_config.auth_type === 'basic'" label="用户名">
            <el-input v-model="projectForm.gitpull_config.username" placeholder="用户名" />
          </el-form-item>
          <el-form-item v-if="projectForm.gitpull_config.auth_type === 'basic'" label="密码">
            <el-input v-model="projectForm.gitpull_config.password" type="password" show-password placeholder="密码" />
          </el-form-item>
          <el-form-item label="部署前脚本">
            <CodeEditor
              :key="`project-pre-script-${projectDialogVisible}`"
              v-model="projectForm.gitpull_config.pre_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 拉取代码前执行的脚本（可选）"
            />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <CodeEditor
              :key="`project-post-script-${projectDialogVisible}`"
              v-model="projectForm.gitpull_config.post_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 拉取代码后执行的脚本（可选）"
            />
          </el-form-item>
        </template>

        <!-- 容器部署配置 -->
        <template v-if="projectForm.type === 'container'">
          <el-divider content-position="left">容器配置</el-divider>
          <el-form-item label="镜像地址" prop="container_config.image">
            <el-input v-model="projectForm.container_config.image" placeholder="nginx:latest" />
          </el-form-item>
          <el-form-item label="容器名称">
            <el-input v-model="projectForm.container_config.container_name" placeholder="my-app" />
          </el-form-item>
          <el-form-item label="重启策略">
            <el-select v-model="projectForm.container_config.restart_policy">
              <el-option label="不重启" value="no" />
              <el-option label="总是重启" value="always" />
              <el-option label="失败时重启" value="on-failure" />
              <el-option label="除非停止" value="unless-stopped" />
            </el-select>
          </el-form-item>
          <el-form-item label="高级配置">
            <el-button type="primary" @click="dockerConfigDialogVisible = true">
              <el-icon><Setting /></el-icon>
              配置端口、存储、网络、资源限制等
            </el-button>
            <div class="form-tip" v-if="hasAdvancedDockerConfig">
              已配置:
              <el-tag v-if="projectForm.container_config.ports?.length" size="small" class="config-tag">{{ projectForm.container_config.ports.length }} 个端口</el-tag>
              <el-tag v-if="projectForm.container_config.volumes?.length" size="small" class="config-tag">{{ projectForm.container_config.volumes.length }} 个卷</el-tag>
              <el-tag v-if="projectForm.container_config.memory_limit" size="small" class="config-tag">内存限制</el-tag>
              <el-tag v-if="projectForm.container_config.cpu_limit" size="small" class="config-tag">CPU限制</el-tag>
              <el-tag v-if="projectForm.container_config.privileged" size="small" type="warning" class="config-tag">特权模式</el-tag>
            </div>
          </el-form-item>
          <el-divider content-position="left">部署脚本（默认，可被版本覆盖）</el-divider>
          <el-form-item label="部署前脚本">
            <CodeEditor
              :key="`container-project-pre-script-${projectDialogVisible}`"
              v-model="projectForm.container_config.pre_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 容器部署前执行的脚本（可选）"
            />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <CodeEditor
              :key="`container-project-post-script-${projectDialogVisible}`"
              v-model="projectForm.container_config.post_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 容器部署后执行的脚本（可选）"
            />
          </el-form-item>
        </template>

        <!-- Kubernetes 配置 -->
        <template v-if="projectForm.type === 'kubernetes'">
          <el-divider content-position="left">Kubernetes 基础配置</el-divider>
          <el-form-item label="命名空间">
            <el-input v-model="projectForm.k8s_config.namespace" placeholder="default" />
          </el-form-item>
          <el-form-item label="资源类型">
            <el-select v-model="projectForm.k8s_config.resource_type" placeholder="选择资源类型">
              <el-option label="Deployment" value="deployment" />
              <el-option label="StatefulSet" value="statefulset" />
              <el-option label="DaemonSet" value="daemonset" />
            </el-select>
          </el-form-item>
          <el-form-item label="资源名称">
            <el-input v-model="projectForm.k8s_config.resource_name" placeholder="my-app" />
          </el-form-item>
          <el-form-item label="容器名称">
            <el-input v-model="projectForm.k8s_config.container_name" placeholder="留空则使用资源名称" />
          </el-form-item>

          <el-divider content-position="left">镜像配置</el-divider>
          <el-form-item label="默认镜像">
            <el-input v-model="projectForm.k8s_config.image" placeholder="nginx:latest" />
          </el-form-item>
          <el-form-item label="镜像仓库">
            <el-input v-model="projectForm.k8s_config.registry" placeholder="registry.example.com（可选）" />
          </el-form-item>
          <el-form-item label="仓库用户" v-if="projectForm.k8s_config.registry">
            <el-input v-model="projectForm.k8s_config.registry_user" placeholder="用户名" />
          </el-form-item>
          <el-form-item label="仓库密码" v-if="projectForm.k8s_config.registry">
            <el-input v-model="projectForm.k8s_config.registry_pass" type="password" show-password placeholder="密码" />
          </el-form-item>
          <el-form-item label="镜像拉取策略">
            <el-select v-model="projectForm.k8s_config.image_pull_policy">
              <el-option label="IfNotPresent（推荐）" value="IfNotPresent" />
              <el-option label="Always" value="Always" />
              <el-option label="Never" value="Never" />
            </el-select>
          </el-form-item>
          <el-form-item label="ImagePullSecret" v-if="projectForm.k8s_config.registry">
            <el-input v-model="projectForm.k8s_config.image_pull_secret" placeholder="自动创建或使用已有 Secret 名称" />
          </el-form-item>

          <el-divider content-position="left">副本与更新策略</el-divider>
          <el-form-item label="默认副本数">
            <el-input-number v-model="projectForm.k8s_config.replicas" :min="1" :max="100" />
          </el-form-item>
          <el-form-item label="更新策略">
            <el-select v-model="projectForm.k8s_config.update_strategy">
              <el-option label="RollingUpdate（滚动更新）" value="RollingUpdate" />
              <el-option label="Recreate（重建）" value="Recreate" />
            </el-select>
          </el-form-item>
          <el-form-item label="最大不可用" v-if="projectForm.k8s_config.update_strategy === 'RollingUpdate'">
            <el-input v-model="projectForm.k8s_config.max_unavailable" placeholder="25% 或 1" style="width: 150px" />
          </el-form-item>
          <el-form-item label="最大超出" v-if="projectForm.k8s_config.update_strategy === 'RollingUpdate'">
            <el-input v-model="projectForm.k8s_config.max_surge" placeholder="25% 或 1" style="width: 150px" />
          </el-form-item>
          <el-form-item label="部署超时">
            <el-input-number v-model="projectForm.k8s_config.rollout_timeout" :min="60" :max="3600" />
            <span class="form-tip ml-2">秒</span>
          </el-form-item>

          <el-divider content-position="left">Service 配置（可选）</el-divider>
          <el-form-item label="Service 类型">
            <el-select v-model="projectForm.k8s_config.service_type" placeholder="不创建 Service">
              <el-option label="不创建 Service" value="" />
              <el-option label="ClusterIP" value="ClusterIP" />
              <el-option label="NodePort" value="NodePort" />
              <el-option label="LoadBalancer" value="LoadBalancer" />
            </el-select>
          </el-form-item>

          <el-divider content-position="left">Kubeconfig（可选）</el-divider>
          <el-form-item label="Kubeconfig">
            <el-input v-model="projectForm.k8s_config.kubeconfig" placeholder="留空使用默认配置" />
            <div class="form-tip">指定 kubeconfig 文件路径，留空使用客户端默认配置</div>
          </el-form-item>
          <el-form-item label="Context" v-if="projectForm.k8s_config.kubeconfig">
            <el-input v-model="projectForm.k8s_config.kube_context" placeholder="留空使用当前 context" />
          </el-form-item>

          <el-divider content-position="left">部署脚本（默认，可被版本覆盖）</el-divider>
          <el-form-item label="部署前脚本">
            <CodeEditor
              :key="`k8s-project-pre-script-${projectDialogVisible}`"
              v-model="projectForm.k8s_config.pre_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# K8s 部署前执行的脚本（可选）"
            />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <CodeEditor
              :key="`k8s-project-post-script-${projectDialogVisible}`"
              v-model="projectForm.k8s_config.post_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# K8s 部署后执行的脚本（可选）"
            />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="projectDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveProject" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

    <!-- 新建版本对话框 -->
    <el-dialog 
      v-model="versionDialogVisible" 
      :title="`新建版本 - ${selectedProject?.name || ''}`" 
      width="900px" 
      destroy-on-close
      class="version-dialog"
    >
      <el-form :model="versionForm" :rules="versionRules" ref="versionFormRef" label-width="120px" class="version-form">
        <!-- 基础信息 -->
        <el-form-item label="版本号" prop="version">
          <el-input 
            v-model="versionForm.version" 
            placeholder="如: 1.0.0" 
            clearable
            style="max-width: 300px;"
          />
          <!-- <div class="form-tip">遵循语义化版本规范 (SemVer)</div> -->
        </el-form-item>
        <el-form-item label="版本说明">
          <el-input 
            v-model="versionForm.description" 
            type="textarea" 
            :rows="2" 
            placeholder="版本更新说明（可选）"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>

        <!-- Git 拉取项目：从仓库选择版本 -->
        <template v-if="selectedProject?.type === 'gitpull'">
          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><DocumentCopy /></el-icon>
              Git 版本选择
            </span>
          </el-divider>

          <el-form-item label="获取版本">
            <div class="flex-items-center">
              <el-button type="primary" :loading="loadingGitVersions" @click="fetchGitVersions">
                <el-icon><Refresh /></el-icon>
                从仓库获取版本
              </el-button>
              <span class="form-tip ml-2" v-if="gitVersions.tags?.length > 0">
                已获取 {{ gitVersions.tags.length }} 个 Tag, {{ gitVersions.branches?.length || 0 }} 个分支
              </span>
            </div>
          </el-form-item>

          <el-form-item label="版本类型">
            <el-radio-group v-model="gitVersionForm.version_type">
              <el-radio value="tag">Tag</el-radio>
              <el-radio value="branch">分支</el-radio>
              <el-radio value="commit">Commit</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item v-if="gitVersionForm.version_type === 'tag'" label="选择 Tag">
            <el-select 
              v-model="gitVersionForm.selected_tag" 
              placeholder="选择 Tag" 
              style="width: 100%" 
              filterable 
              @change="onTagSelected"
            >
              <el-option 
                v-for="tag in gitVersions.tags" 
                :key="tag.name" 
                :label="`${tag.name} - ${tag.message || tag.commit?.substring(0, 7)}`" 
                :value="tag.name" 
              />
            </el-select>
            <div class="form-tip" v-if="gitVersions.tags?.length === 0">暂无 Tag，请先获取版本</div>
          </el-form-item>

          <el-form-item v-if="gitVersionForm.version_type === 'branch'" label="选择分支">
            <el-select 
              v-model="gitVersionForm.selected_branch" 
              placeholder="选择分支" 
              style="width: 100%" 
              filterable
            >
              <el-option 
                v-for="branch in gitVersions.branches" 
                :key="branch.name" 
                :label="`${branch.name}${branch.is_default ? ' (默认)' : ''}`" 
                :value="branch.name" 
              />
            </el-select>
          </el-form-item>

          <el-form-item v-if="gitVersionForm.version_type === 'commit'" label="选择 Commit">
            <el-select 
              v-model="gitVersionForm.selected_commit" 
              placeholder="选择 Commit" 
              style="width: 100%" 
              filterable
            >
              <el-option 
                v-for="commit in gitVersions.recent_commits" 
                :key="commit.hash" 
                :label="`${commit.hash} - ${commit.message}`" 
                :value="commit.full_hash || commit.hash" 
              />
            </el-select>
          </el-form-item>

          <el-form-item label="当前信息" v-if="gitVersions.current_branch">
            <div class="flex-items-center">
              <el-tag type="info">{{ gitVersions.current_branch }}</el-tag>
              <span class="ml-2 text-gray">{{ gitVersions.current_commit?.substring(0, 7) }}</span>
            </div>
          </el-form-item>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Document /></el-icon>
              部署脚本（可选）
            </span>
          </el-divider>
          <el-form-item label="部署前脚本">
            <CodeEditor
              :key="`git-pre-${versionDialogVisible}`"
              v-model="versionForm.pre_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# Git 拉取前执行的脚本（可选）&#10;# 例如：停止服务、备份数据等"
            />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <CodeEditor
              :key="`git-post-${versionDialogVisible}`"
              v-model="versionForm.post_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# Git 拉取后执行的脚本（可选）&#10;# 例如：编译、重启服务、清理缓存等"
            />
          </el-form-item>
        </template>

        <!-- 容器项目：镜像版本 -->
        <template v-else-if="selectedProject?.type === 'container'">
          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Box /></el-icon>
              容器镜像配置
            </span>
          </el-divider>

          <el-form-item label="镜像地址" prop="container_image">
            <el-input 
              v-model="versionForm.container_image" 
              placeholder="nginx:1.25.0 或 registry.example.com/app:v1.0.0" 
              clearable
            />
            <div class="form-tip" v-if="selectedProject?.container_config?.image">
              <el-icon><InfoFilled /></el-icon>
              项目默认镜像: <code>{{ selectedProject.container_config.image }}</code>
            </div>
          </el-form-item>

          <el-form-item label="环境变量">
            <CodeEditor
              :key="`container-env-${versionDialogVisible}`"
              v-model="versionForm.container_env"
              language="properties"
              height="120px"
              placeholder="KEY1=value1&#10;KEY2=value2&#10;（增量追加到项目配置）"
            />
          </el-form-item>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Setting /></el-icon>
              资源限制（可选，覆盖项目默认值）
            </span>
          </el-divider>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="内存限制">
                <el-input 
                  v-model="versionForm.deploy_config.resources.memory_limit" 
                  placeholder="512Mi" 
                  clearable
                />
                <div class="form-tip" v-if="selectedProject?.container_config?.memory_limit">
                  <el-icon><InfoFilled /></el-icon>
                  项目默认: <code>{{ selectedProject.container_config.memory_limit }}</code>
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="CPU 限制">
                <el-input 
                  v-model="versionForm.deploy_config.resources.cpu_limit" 
                  placeholder="500m" 
                  clearable
                />
                <div class="form-tip" v-if="selectedProject?.container_config?.cpu_limit">
                  <el-icon><InfoFilled /></el-icon>
                  项目默认: <code>{{ selectedProject.container_config.cpu_limit }}</code>
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Document /></el-icon>
              部署脚本（可选）
            </span>
          </el-divider>
          <el-form-item label="部署前脚本">
            <CodeEditor
              :key="`container-pre-${versionDialogVisible}`"
              v-model="versionForm.deploy_config.pre_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 容器部署前执行的脚本（可选）"
            />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <CodeEditor
              :key="`container-post-${versionDialogVisible}`"
              v-model="versionForm.deploy_config.post_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 容器部署后执行的脚本（可选）"
            />
          </el-form-item>
        </template>

        <!-- Kubernetes 项目 -->
        <template v-else-if="selectedProject?.type === 'kubernetes'">
          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Grid /></el-icon>
              Kubernetes 部署配置
            </span>
          </el-divider>

          <el-form-item label="镜像版本" prop="container_image">
            <el-input 
              v-model="versionForm.container_image" 
              placeholder="nginx:1.25.0（留空使用项目默认镜像）" 
              clearable
            />
            <div class="form-tip" v-if="selectedProject?.k8s_config?.image">
              <el-icon><InfoFilled /></el-icon>
              项目默认镜像: <code>{{ selectedProject.k8s_config.image }}</code>
            </div>
          </el-form-item>

          <el-form-item label="副本数">
            <div class="flex-items-center">
              <el-input-number 
                v-model="versionForm.deploy_config.replicas" 
                :min="1" 
                :max="100" 
                style="width: 150px;"
              />
              <span class="form-tip ml-2">
                留空使用项目默认值
                <template v-if="selectedProject?.k8s_config?.replicas">
                  ({{ selectedProject.k8s_config.replicas }})
                </template>
              </span>
            </div>
          </el-form-item>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Setting /></el-icon>
              资源限制（可选，覆盖项目默认值）
            </span>
          </el-divider>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="CPU Request">
                <el-input 
                  v-model="versionForm.deploy_config.resources.cpu_request" 
                  placeholder="100m" 
                  clearable
                />
                <div class="form-tip" v-if="selectedProject?.k8s_config?.cpu_request">
                  <el-icon><InfoFilled /></el-icon>
                  项目默认: <code>{{ selectedProject.k8s_config.cpu_request }}</code>
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="CPU Limit">
                <el-input 
                  v-model="versionForm.deploy_config.resources.cpu_limit" 
                  placeholder="500m" 
                  clearable
                />
                <div class="form-tip" v-if="selectedProject?.k8s_config?.cpu_limit">
                  <el-icon><InfoFilled /></el-icon>
                  项目默认: <code>{{ selectedProject.k8s_config.cpu_limit }}</code>
                </div>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="Memory Request">
                <el-input 
                  v-model="versionForm.deploy_config.resources.memory_request" 
                  placeholder="128Mi" 
                  clearable
                />
                <div class="form-tip" v-if="selectedProject?.k8s_config?.memory_request">
                  <el-icon><InfoFilled /></el-icon>
                  项目默认: <code>{{ selectedProject.k8s_config.memory_request }}</code>
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Memory Limit">
                <el-input 
                  v-model="versionForm.deploy_config.resources.memory_limit" 
                  placeholder="512Mi" 
                  clearable
                />
                <div class="form-tip" v-if="selectedProject?.k8s_config?.memory_limit">
                  <el-icon><InfoFilled /></el-icon>
                  项目默认: <code>{{ selectedProject.k8s_config.memory_limit }}</code>
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Key /></el-icon>
              环境变量（可选）
            </span>
          </el-divider>
          <el-form-item label="环境变量">
            <CodeEditor
              :key="`k8s-env-${versionDialogVisible}`"
              v-model="versionForm.container_env"
              language="properties"
              height="120px"
              placeholder="KEY1=value1&#10;KEY2=value2&#10;（增量追加到项目配置）"
            />
          </el-form-item>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><DocumentCopy /></el-icon>
              YAML 配置（可选）
            </span>
          </el-divider>
          <el-form-item>
            <div class="form-tip mb-8">
              <el-icon><InfoFilled /></el-icon>
              可选择覆盖默认配置，或提供完整的 Kubernetes YAML 资源定义
            </div>
            <CodeEditor
              :key="`k8s-yaml-${versionDialogVisible}`"
              v-model="versionForm.deploy_config.k8s_yaml_full"
              language="yaml"
              height="250px"
              placeholder="# Kubernetes Deployment YAML（可选覆盖）&#10;# 留空则使用项目配置自动生成"
            />
          </el-form-item>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Document /></el-icon>
              部署脚本（可选）
            </span>
          </el-divider>
          <el-form-item label="部署前脚本">
            <CodeEditor
              :key="`k8s-pre-${versionDialogVisible}`"
              v-model="versionForm.deploy_config.pre_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# K8s 部署前执行的脚本（可选）"
            />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <CodeEditor
              :key="`k8s-post-${versionDialogVisible}`"
              v-model="versionForm.deploy_config.post_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# K8s 部署后执行的脚本（可选）"
            />
          </el-form-item>
        </template>

        <!-- 脚本项目：传统脚本编辑 -->
        <template v-else>
          <el-form-item label="工作目录">
            <el-input 
              v-model="versionForm.work_dir" 
              placeholder="/opt/app" 
              clearable
            />
            <div class="form-tip">脚本执行的工作目录路径</div>
          </el-form-item>

          <el-divider content-position="left">
            <span style="display: flex; align-items: center; gap: 6px;">
              <el-icon><Document /></el-icon>
              部署脚本
            </span>
          </el-divider>

          <el-tabs v-model="scriptTab" type="border-card" class="script-tabs">
            <el-tab-pane label="安装脚本" name="install">
              <CodeEditor
                :key="`script-install-${versionDialogVisible}`"
                v-model="versionForm.install_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 首次安装时执行的脚本"
              />
            </el-tab-pane>
            <el-tab-pane label="升级脚本" name="update">
              <CodeEditor
                :key="`script-update-${versionDialogVisible}`"
                v-model="versionForm.update_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 升级时执行的脚本"
              />
            </el-tab-pane>
            <el-tab-pane label="回滚脚本" name="rollback">
              <CodeEditor
                :key="`script-rollback-${versionDialogVisible}`"
                v-model="versionForm.rollback_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 回滚时执行的脚本"
              />
            </el-tab-pane>
            <el-tab-pane label="卸载脚本" name="uninstall">
              <CodeEditor
                :key="`script-uninstall-${versionDialogVisible}`"
                v-model="versionForm.uninstall_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 卸载时执行的脚本"
              />
            </el-tab-pane>
          </el-tabs>
        </template>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="versionDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveVersion" :loading="submitting">
            <el-icon v-if="!submitting"><Check /></el-icon>
            保存版本
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 创建部署任务对话框 -->
    <el-dialog v-model="taskDialogVisible" title="创建部署任务" width="750px">
      <el-form :model="taskForm" :rules="taskRules" ref="taskFormRef" label-width="120px">
        <el-form-item label="部署版本">
          <el-tag>{{ selectedVersion?.version }}</el-tag>
          <el-tag type="info" class="ml-2">{{ getProjectTypeLabel(selectedProject?.type) }}</el-tag>
        </el-form-item>

        <el-form-item label="操作类型" prop="operation">
          <!-- Docker 容器操作 -->
          <template v-if="selectedProject?.type === 'container'">
            <el-radio-group v-model="taskForm.operation">
              <el-radio value="deploy">
                <el-icon><Promotion /></el-icon> 部署
                <span class="op-desc">（自动判断新建/更新）</span>
              </el-radio>
              <el-radio value="install">
                <el-icon><Download /></el-icon> 新建容器
              </el-radio>
              <el-radio value="update">
                <el-icon><Upload /></el-icon> 更新镜像
              </el-radio>
              <el-radio value="rollback">
                <el-icon><RefreshLeft /></el-icon> 回滚
              </el-radio>
              <el-radio value="uninstall">
                <el-icon><Delete /></el-icon> 删除容器
              </el-radio>
            </el-radio-group>
          </template>

          <!-- Kubernetes 操作 -->
          <template v-else-if="selectedProject?.type === 'kubernetes'">
            <el-radio-group v-model="taskForm.operation">
              <el-radio value="deploy">
                <el-icon><Promotion /></el-icon> 部署
                <span class="op-desc">（自动判断新建/更新）</span>
              </el-radio>
              <el-radio value="install">
                <el-icon><Download /></el-icon> 创建资源
              </el-radio>
              <el-radio value="update">
                <el-icon><Upload /></el-icon> 滚动更新
              </el-radio>
              <el-radio value="rollback">
                <el-icon><RefreshLeft /></el-icon> 回滚版本
              </el-radio>
              <el-radio value="uninstall">
                <el-icon><Delete /></el-icon> 删除资源
              </el-radio>
            </el-radio-group>
          </template>

          <!-- Git Pull 操作 -->
          <template v-else-if="selectedProject?.type === 'gitpull'">
            <el-radio-group v-model="taskForm.operation">
              <el-radio value="deploy">
                <el-icon><Promotion /></el-icon> 部署
                <span class="op-desc">（自动判断克隆/拉取）</span>
              </el-radio>
              <el-radio value="install">
                <el-icon><Download /></el-icon> 克隆仓库
              </el-radio>
              <el-radio value="update">
                <el-icon><Upload /></el-icon> 拉取更新
              </el-radio>
              <el-radio value="rollback">
                <el-icon><RefreshLeft /></el-icon> 回滚
              </el-radio>
            </el-radio-group>
          </template>

          <!-- 脚本操作（默认） -->
          <template v-else>
            <el-radio-group v-model="taskForm.operation">
              <el-radio value="install">
                <el-icon><Download /></el-icon> 安装
              </el-radio>
              <el-radio value="update">
                <el-icon><Upload /></el-icon> 升级
              </el-radio>
              <el-radio value="rollback">
                <el-icon><RefreshLeft /></el-icon> 回滚
              </el-radio>
              <el-radio value="uninstall">
                <el-icon><Delete /></el-icon> 卸载
              </el-radio>
            </el-radio-group>
          </template>
        </el-form-item>

        <!-- Docker 容器特定配置 -->
        <template v-if="selectedProject?.type === 'container'">
          <el-divider content-position="left">容器配置覆盖（临时覆盖，仅本次任务生效）</el-divider>

          <el-form-item label="镜像覆盖" v-if="taskForm.operation !== 'uninstall'">
            <el-input v-model="taskForm.override_config.image" :placeholder="selectedVersion?.container_image || selectedProject?.container_config?.image || 'nginx:latest'" />
            <div class="form-tip">留空则使用版本配置的镜像（临时测试未发布的镜像时使用）</div>
          </el-form-item>

          <el-form-item label="追加环境变量" v-if="taskForm.operation !== 'uninstall'">
            <CodeEditor
              v-if="taskDialogVisible"
              v-model="taskForm.container_env"
              language="properties"
              height="100px"
              placeholder="KEY=value（每行一个，追加到版本环境变量之上）"
            />
          </el-form-item>

          <el-form-item label="资源覆盖" v-if="taskForm.operation !== 'uninstall'">
            <el-row :gutter="20">
              <el-col :span="12">
                <el-input v-model="taskForm.override_config.resources.memory_limit" placeholder="内存限制（如 512Mi）">
                  <template #prepend>内存</template>
                </el-input>
              </el-col>
              <el-col :span="12">
                <el-input v-model="taskForm.override_config.resources.cpu_limit" placeholder="CPU 限制（如 500m）">
                  <template #prepend>CPU</template>
                </el-input>
              </el-col>
            </el-row>
            <div class="form-tip">紧急扩容或金丝雀使用不同资源时填写</div>
          </el-form-item>

          <el-form-item label="拉取策略" v-if="taskForm.operation !== 'uninstall'">
            <el-select v-model="taskForm.image_pull_policy">
              <el-option label="始终拉取 (Always)" value="always" />
              <el-option label="不存在时拉取 (IfNotPresent)" value="ifnotpresent" />
              <el-option label="从不拉取 (Never)" value="never" />
            </el-select>
          </el-form-item>
        </template>

        <!-- Kubernetes 特定配置 -->
        <template v-if="selectedProject?.type === 'kubernetes'">
          <el-divider content-position="left">Kubernetes 配置覆盖（临时覆盖，仅本次任务生效）</el-divider>

          <el-form-item label="镜像覆盖" v-if="taskForm.operation !== 'uninstall' && taskForm.operation !== 'rollback'">
            <el-input v-model="taskForm.override_config.image" :placeholder="selectedVersion?.container_image || selectedProject?.k8s_config?.image || 'nginx:latest'" />
            <div class="form-tip">留空则使用版本配置的镜像（临时测试未发布的镜像时使用）</div>
          </el-form-item>

          <el-form-item label="副本数覆盖" v-if="taskForm.operation === 'install' || taskForm.operation === 'deploy'">
            <el-input-number v-model="taskForm.override_config.replicas" :min="1" :max="100" />
            <span class="form-tip ml-2">留空使用版本配置（紧急扩缩容时填写）</span>
          </el-form-item>

          <el-form-item label="资源覆盖" v-if="taskForm.operation !== 'uninstall' && taskForm.operation !== 'rollback'">
            <el-row :gutter="20" class="mb-8">
              <el-col :span="12">
                <el-input v-model="taskForm.override_config.resources.cpu_request" placeholder="CPU Request（如 100m）">
                  <template #prepend>CPU Req</template>
                </el-input>
              </el-col>
              <el-col :span="12">
                <el-input v-model="taskForm.override_config.resources.cpu_limit" placeholder="CPU Limit（如 500m）">
                  <template #prepend>CPU Lim</template>
                </el-input>
              </el-col>
            </el-row>
            <el-row :gutter="20">
              <el-col :span="12">
                <el-input v-model="taskForm.override_config.resources.memory_request" placeholder="Memory Request（如 128Mi）">
                  <template #prepend>Mem Req</template>
                </el-input>
              </el-col>
              <el-col :span="12">
                <el-input v-model="taskForm.override_config.resources.memory_limit" placeholder="Memory Limit（如 512Mi）">
                  <template #prepend>Mem Lim</template>
                </el-input>
              </el-col>
            </el-row>
            <div class="form-tip">紧急扩容或金丝雀使用不同资源时填写</div>
          </el-form-item>

          <el-form-item label="追加环境变量" v-if="taskForm.operation !== 'uninstall' && taskForm.operation !== 'rollback'">
            <CodeEditor
              v-if="taskDialogVisible"
              v-model="taskForm.container_env"
              language="properties"
              height="100px"
              placeholder="KEY=value（每行一个，追加到版本环境变量之上）"
            />
          </el-form-item>

          <el-form-item label="回滚到版本" v-if="taskForm.operation === 'rollback'">
            <el-input-number v-model="taskForm.k8s_revision" :min="0" placeholder="0 表示上一个版本" />
            <div class="form-tip">输入 0 或留空表示回滚到上一个版本</div>
          </el-form-item>

          <el-form-item label="等待超时" v-if="taskForm.operation !== 'uninstall'">
            <el-input-number v-model="taskForm.k8s_timeout" :min="60" :max="3600" />
            <span class="form-tip ml-2">秒（等待 Pod 就绪）</span>
          </el-form-item>
        </template>

        <!-- Git Pull 特定配置 -->
        <template v-if="selectedProject?.type === 'gitpull'">
          <el-divider content-position="left">Git 配置</el-divider>

          <el-form-item label="目标版本" v-if="taskForm.operation !== 'rollback'">
            <el-input v-model="taskForm.git_ref" :placeholder="selectedVersion?.git_ref || 'main'" />
            <div class="form-tip">Tag、分支名或 Commit SHA（留空使用版本默认值）</div>
          </el-form-item>

          <el-form-item label="回滚到" v-if="taskForm.operation === 'rollback'">
            <el-input v-model="taskForm.git_rollback_ref" placeholder="输入要回滚到的 Tag、分支或 Commit" />
          </el-form-item>

          <el-form-item label="部署后脚本">
            <el-switch v-model="taskForm.run_post_script" />
            <span class="form-tip ml-2">执行项目配置的部署后脚本</span>
          </el-form-item>

          <el-form-item label="强制覆盖">
            <el-switch v-model="taskForm.git_force" />
            <span class="form-tip ml-2">丢弃本地修改，强制同步远程版本</span>
          </el-form-item>
        </template>

        <el-divider content-position="left">目标选择</el-divider>

        <el-form-item label="目标客户端" prop="client_ids_text">
          <el-input
            v-model="taskForm.client_ids_text"
            type="textarea"
            :rows="6"
            placeholder="请输入客户端ID，每行一个&#10;例如：&#10;client-001&#10;client-002&#10;client-003"
            style="font-family: monospace;"
          />
          <div class="form-tip">
            <el-button link type="primary" @click="selectAllClients">填充所有客户端</el-button>
            <el-button link @click="taskForm.client_ids_text = ''">清空</el-button>
            <span class="ml-2">已输入 {{ getClientIdsCount() }} 个客户端ID</span>
          </div>
        </el-form-item>

        <el-divider content-position="left">执行计划</el-divider>

        <el-form-item label="执行方式">
          <el-radio-group v-model="taskForm.schedule_type">
            <el-radio value="immediate">立即执行</el-radio>
            <el-radio value="scheduled">定时执行</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="taskForm.schedule_type === 'scheduled'" label="执行时间">
          <el-date-picker
            v-model="taskForm.schedule_time"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            :shortcuts="scheduleShortcuts"
          />
        </el-form-item>

        <el-divider content-position="left">金丝雀发布</el-divider>

        <el-form-item label="启用金丝雀">
          <el-switch v-model="taskForm.canary_enabled" />
          <span class="form-tip ml-2">先部署部分节点，观察无问题后再全量发布</span>
        </el-form-item>

        <template v-if="taskForm.canary_enabled">
          <el-form-item label="金丝雀比例">
            <el-slider
              v-model="taskForm.canary_percent"
              :min="1"
              :max="50"
              :marks="{ 10: '10%', 20: '20%', 30: '30%', 50: '50%' }"
              show-input
            />
          </el-form-item>

          <el-form-item label="观察时间">
            <el-input-number v-model="taskForm.canary_duration" :min="1" :max="1440" />
            <span class="form-tip ml-2">分钟（观察期内可手动决定是否全量）</span>
          </el-form-item>

          <el-form-item label="自动全量">
            <el-switch v-model="taskForm.canary_auto_promote" />
            <span class="form-tip ml-2">观察期结束后自动全量发布（否则需手动确认）</span>
          </el-form-item>
        </template>

        <el-divider content-position="left">回调通知</el-divider>

        <el-form-item label="启用回调">
          <el-switch v-model="taskForm.callback_enabled" />
          <span class="form-tip ml-2">发布完成后发送通知（飞书/钉钉/企微等）</span>
        </el-form-item>

        <template v-if="taskForm.callback_enabled">
          <el-form-item label="回调配置">
            <el-select
              v-model="taskForm.callback_config_id"
              placeholder="选择回调配置"
              clearable
              :loading="loadingCallbackConfigs"
              style="width: 100%"
            >
              <el-option
                v-for="cfg in projectCallbackConfigs"
                :key="cfg.id"
                :label="cfg.name"
                :value="cfg.id"
              >
                <div class="callback-option">
                  <span>{{ cfg.name }}</span>
                  <el-tag size="small" type="info" class="ml-2">
                    {{ getChannelLabels(cfg.channels) }}
                  </el-tag>
                </div>
              </el-option>
            </el-select>
            <div class="form-tip">
              <span v-if="projectCallbackConfigs.length === 0">
                暂无回调配置，
                <el-button link type="primary" @click="goToCallbackConfig">去配置</el-button>
              </span>
              <span v-else>
                选择要使用的回调配置，发布事件将推送到配置的渠道
              </span>
            </div>
          </el-form-item>

          <el-form-item label="通知事件" v-if="taskForm.callback_config_id">
            <el-checkbox-group v-model="taskForm.callback_events">
              <el-checkbox value="canary_started" v-if="taskForm.canary_enabled">
                <el-icon><VideoPlay /></el-icon> 金丝雀开始
              </el-checkbox>
              <el-checkbox value="canary_completed" v-if="taskForm.canary_enabled">
                <el-icon><Check /></el-icon> 金丝雀完成
              </el-checkbox>
              <el-checkbox value="full_completed">
                <el-icon><SuccessFilled /></el-icon> 全量完成
              </el-checkbox>
            </el-checkbox-group>
            <div class="form-tip">选择需要推送通知的发布事件</div>
          </el-form-item>
        </template>

        <el-divider content-position="left">失败处理</el-divider>

        <el-form-item label="失败策略">
          <el-radio-group v-model="taskForm.failure_strategy">
            <el-radio value="continue">继续执行其他节点</el-radio>
            <el-radio value="pause">暂停等待处理</el-radio>
            <el-radio value="abort">立即终止任务</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="taskForm.operation === 'update'" label="升级失败回滚">
          <el-switch v-model="taskForm.auto_rollback" />
          <span class="form-tip ml-2">升级失败时自动回滚到上一个版本</span>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="task-dialog-footer">
          <el-button
            v-if="selectedProject?.type === 'container' || selectedProject?.type === 'kubernetes'"
            @click="showConfigPreview"
          >
            <el-icon><View /></el-icon>
            预览最终配置
          </el-button>
          <div class="footer-actions">
            <el-button @click="taskDialogVisible = false">取消</el-button>
            <el-button type="primary" @click="createTask" :loading="submitting">创建任务</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- 配置预览对话框 -->
    <el-dialog v-model="configPreviewVisible" title="最终配置预览（三层合并结果）" width="70%" destroy-on-close>
      <el-alert
        type="info"
        :closable="false"
        class="mb-16"
        title="配置合并规则：项目配置 → 版本配置 → 任务覆盖"
        description="以下是三层配置合并后的最终结果。任务覆盖配置仅对本次部署生效。"
      />

      <el-tabs v-model="configPreviewTab">
        <el-tab-pane label="基础配置" name="basic">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="镜像">
              <el-tag type="primary">{{ mergedConfigPreview.image || '未配置' }}</el-tag>
              <el-tag
                v-if="configPreviewSource.image"
                size="small"
                :type="configPreviewSource.image === 'task' ? 'warning' : 'info'"
                class="ml-2"
              >
                来自{{ configPreviewSource.image === 'task' ? '任务覆盖' : configPreviewSource.image === 'version' ? '版本' : '项目' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="容器名" v-if="selectedProject?.type === 'container'">
              {{ mergedConfigPreview.containerName || '默认' }}
            </el-descriptions-item>
            <el-descriptions-item label="副本数" v-if="selectedProject?.type === 'kubernetes'">
              {{ mergedConfigPreview.replicas || 1 }}
              <el-tag
                v-if="configPreviewSource.replicas"
                size="small"
                :type="configPreviewSource.replicas === 'task' ? 'warning' : 'info'"
                class="ml-2"
              >
                来自{{ configPreviewSource.replicas === 'task' ? '任务覆盖' : configPreviewSource.replicas === 'version' ? '版本' : '项目' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="重启策略" v-if="selectedProject?.type === 'container'">
              {{ mergedConfigPreview.restartPolicy || 'unless-stopped' }}
            </el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="资源限制" name="resources">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="CPU Request" v-if="selectedProject?.type === 'kubernetes'">
              {{ mergedConfigPreview.cpuRequest || '未限制' }}
            </el-descriptions-item>
            <el-descriptions-item label="CPU Limit">
              {{ mergedConfigPreview.cpuLimit || '未限制' }}
              <el-tag
                v-if="configPreviewSource.cpuLimit"
                size="small"
                :type="configPreviewSource.cpuLimit === 'task' ? 'warning' : 'info'"
                class="ml-2"
              >
                来自{{ configPreviewSource.cpuLimit === 'task' ? '任务覆盖' : configPreviewSource.cpuLimit === 'version' ? '版本' : '项目' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Memory Request" v-if="selectedProject?.type === 'kubernetes'">
              {{ mergedConfigPreview.memoryRequest || '未限制' }}
            </el-descriptions-item>
            <el-descriptions-item label="Memory Limit">
              {{ mergedConfigPreview.memoryLimit || '未限制' }}
              <el-tag
                v-if="configPreviewSource.memoryLimit"
                size="small"
                :type="configPreviewSource.memoryLimit === 'task' ? 'warning' : 'info'"
                class="ml-2"
              >
                来自{{ configPreviewSource.memoryLimit === 'task' ? '任务覆盖' : configPreviewSource.memoryLimit === 'version' ? '版本' : '项目' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="环境变量" name="environment">
          <div v-if="Object.keys(mergedConfigPreview.environment || {}).length > 0">
            <el-table :data="environmentPreviewData" size="small" border>
              <el-table-column prop="key" label="变量名" width="200" />
              <el-table-column prop="value" label="值" />
              <el-table-column prop="source" label="来源" width="100">
                <template #default="{ row }">
                  <el-tag
                    size="small"
                    :type="row.source === 'task' ? 'warning' : row.source === 'version' ? 'info' : ''"
                  >
                    {{ row.source === 'task' ? '任务' : row.source === 'version' ? '版本' : '项目' }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </div>
          <el-empty v-else description="未配置环境变量" :image-size="60" />
        </el-tab-pane>

        <el-tab-pane label="网络/存储" name="network" v-if="selectedProject?.type === 'container'">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="网络模式">
              {{ selectedProject?.container_config?.network_mode || 'bridge' }}
            </el-descriptions-item>
            <el-descriptions-item label="端口映射">
              <template v-if="selectedProject?.container_config?.ports?.length">
                <el-tag v-for="port in selectedProject.container_config.ports" :key="port.host + ':' + port.container" size="small" class="mr-2">
                  {{ port.host }}:{{ port.container }}
                </el-tag>
              </template>
              <span v-else>未配置</span>
            </el-descriptions-item>
            <el-descriptions-item label="存储卷" :span="2">
              <template v-if="selectedProject?.container_config?.volumes?.length">
                <div v-for="vol in selectedProject.container_config.volumes" :key="vol.host" class="volume-item">
                  {{ vol.host }} → {{ vol.container }} ({{ vol.mode || 'rw' }})
                </div>
              </template>
              <span v-else>未配置</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <!-- 任务详情抽屉 -->
    <el-drawer v-model="taskDetailVisible" title="任务详情" size="60%">
      <template v-if="selectedTask">
        <el-descriptions :column="3" border>
          <el-descriptions-item label="版本">{{ selectedTask.version }}</el-descriptions-item>
          <el-descriptions-item label="部署类型">
            <el-tag :type="getProjectTypeTag(selectedTask.deploy_type || selectedProject?.type)">
              {{ getProjectTypeLabel(selectedTask.deploy_type || selectedProject?.type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="操作">
            <el-tag :type="getOperationTag(selectedTask.operation)">
              {{ getOperationLabel(selectedTask.operation) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getTaskStatusTag(selectedTask.status)">
              {{ getTaskStatusLabel(selectedTask.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="执行方式">
            {{ selectedTask.schedule_type === 'immediate' ? '立即执行' : '定时执行' }}
          </el-descriptions-item>
          <el-descriptions-item label="金丝雀">
            {{ selectedTask.canary_enabled ? `${selectedTask.canary_percent}%` : '未启用' }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(selectedTask.created_at) }}</el-descriptions-item>
        </el-descriptions>

        <!-- Docker 容器任务详情 -->
        <template v-if="(selectedTask.deploy_type || selectedProject?.type) === 'container'">
          <el-descriptions :column="2" border class="mt-16" title="容器配置">
            <el-descriptions-item label="镜像">
              <el-tag type="info">{{ selectedTask.container_image || selectedTask.image || '默认' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="容器名">{{ selectedTask.container_name || '默认' }}</el-descriptions-item>
            <el-descriptions-item label="拉取策略">{{ selectedTask.image_pull_policy || 'ifnotpresent' }}</el-descriptions-item>
            <el-descriptions-item label="镜像已拉取">
              <el-tag v-if="selectedTask.image_pulled" size="small" type="success">是</el-tag>
              <el-tag v-else size="small" type="info">否</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </template>

        <!-- Kubernetes 任务详情 -->
        <template v-if="(selectedTask.deploy_type || selectedProject?.type) === 'kubernetes'">
          <el-descriptions :column="2" border class="mt-16" title="Kubernetes 配置">
            <el-descriptions-item label="命名空间">{{ selectedTask.namespace || 'default' }}</el-descriptions-item>
            <el-descriptions-item label="资源名">{{ selectedTask.resource_name || '默认' }}</el-descriptions-item>
            <el-descriptions-item label="镜像">
              <el-tag type="info">{{ selectedTask.k8s_image || selectedTask.image || '默认' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="副本数">
              {{ selectedTask.replicas || '默认' }} / {{ selectedTask.ready_replicas || 0 }} 就绪
            </el-descriptions-item>
            <el-descriptions-item label="版本号">{{ selectedTask.revision || '-' }}</el-descriptions-item>
            <el-descriptions-item label="滚动状态">
              <el-tag :type="selectedTask.rollout_status === 'complete' ? 'success' : 'warning'" size="small">
                {{ selectedTask.rollout_status || '未知' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </template>

        <!-- Git Pull 任务详情 -->
        <template v-if="(selectedTask.deploy_type || selectedProject?.type) === 'gitpull'">
          <el-descriptions :column="2" border class="mt-16" title="Git 配置">
            <el-descriptions-item label="Git Ref">
              <el-tag type="info">{{ selectedTask.git_ref || selectedTask.branch || '默认' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="当前 Commit">{{ selectedTask.commit?.substring(0, 8) || '-' }}</el-descriptions-item>
            <el-descriptions-item label="执行部署脚本">
              <el-tag v-if="selectedTask.run_post_script" size="small" type="success">是</el-tag>
              <el-tag v-else size="small" type="info">否</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="备份">
              <el-tag v-if="selectedTask.backed_up" size="small" type="success">已备份</el-tag>
              <el-tag v-else size="small" type="info">未备份</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </template>

        <div class="task-progress" v-if="selectedTask.status === 'running' || selectedTask.status === 'canary'">
          <h4>执行进度</h4>
          <el-progress
            :percentage="getTaskProgress(selectedTask)"
            :status="getProgressStatus(selectedTask.status)"
            :stroke-width="16"
          />
          <div class="progress-stats">
            <span class="stat success">成功: {{ selectedTask.success_count || 0 }}</span>
            <span class="stat failed">失败: {{ selectedTask.failed_count || 0 }}</span>
            <span class="stat pending">待执行: {{ selectedTask.pending_count || 0 }}</span>
          </div>
        </div>

        <h4 style="margin: 20px 0 10px">目标执行结果</h4>
        <el-table :data="selectedTask.results || []" size="small" border max-height="400">
          <el-table-column prop="client_id" label="客户端" width="180" />
          <el-table-column prop="status" label="状态" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="getResultStatusTag(row.status)">
                {{ getResultStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="is_canary" label="金丝雀" width="70">
            <template #default="{ row }">
              <el-tag v-if="row.is_canary" size="small" type="warning">是</el-tag>
            </template>
          </el-table-column>
          <!-- Docker/K8s 特有列 -->
          <el-table-column v-if="(selectedTask.deploy_type || selectedProject?.type) === 'container'" prop="container_id" label="容器ID" width="100">
            <template #default="{ row }">
              {{ row.container_id?.substring(0, 12) || '-' }}
            </template>
          </el-table-column>
          <el-table-column v-if="(selectedTask.deploy_type || selectedProject?.type) === 'kubernetes'" prop="ready_replicas" label="就绪" width="60">
            <template #default="{ row }">
              {{ row.ready_replicas || 0 }}/{{ row.replicas || 0 }}
            </template>
          </el-table-column>
          <!-- Git 特有列 -->
          <el-table-column v-if="(selectedTask.deploy_type || selectedProject?.type) === 'gitpull'" prop="commit" label="Commit" width="80">
            <template #default="{ row }">
              {{ row.commit?.substring(0, 7) || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="started_at" label="开始时间" width="150">
            <template #default="{ row }">
              {{ formatTime(row.started_at) }}
            </template>
          </el-table-column>
          <el-table-column prop="duration" label="耗时" width="70">
            <template #default="{ row }">
              {{ formatDuration(row.duration) }}
            </template>
          </el-table-column>
          <el-table-column prop="error" label="错误信息" min-width="150">
            <template #default="{ row }">
              <span class="error-text">{{ row.error || '-' }}</span>
            </template>
          </el-table-column>
          <!-- 容器日志操作 -->
          <el-table-column v-if="(selectedTask.deploy_type || selectedProject?.type) === 'container'" label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button
                size="small"
                type="primary"
                link
                @click="showContainerLogs(row)"
                :disabled="!row.container_id && !selectedTask.container_name"
              >
                <el-icon><Document /></el-icon>
                日志
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <!-- 版本详情抽屉 -->
    <el-drawer v-model="versionDetailVisible" title="版本详情" size="50%">
      <template v-if="selectedVersionDetail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="版本号">{{ selectedVersionDetail.version }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getVersionStatusTag(selectedVersionDetail.status)">
              {{ getVersionStatusLabel(selectedVersionDetail.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="部署类型">
            <el-tag :type="getProjectTypeTag(selectedProject?.type)">
              {{ getProjectTypeLabel(selectedProject?.type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(selectedVersionDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="版本说明" :span="2">
            {{ selectedVersionDetail.description || '无' }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- Docker 容器版本详情 -->
        <template v-if="selectedProject?.type === 'container'">
          <el-descriptions :column="2" border class="mt-16" title="容器配置">
            <el-descriptions-item label="镜像">
              <el-tag type="info">{{ selectedVersionDetail.container_image || selectedProject?.container_config?.image || '未配置' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="容器名">{{ selectedProject?.container_config?.container_name || '默认' }}</el-descriptions-item>
            <el-descriptions-item label="重启策略">{{ selectedProject?.container_config?.restart_policy || 'unless-stopped' }}</el-descriptions-item>
            <el-descriptions-item label="网络模式">{{ selectedProject?.container_config?.network_mode || 'bridge' }}</el-descriptions-item>
          </el-descriptions>
          <div v-if="selectedVersionDetail.container_env" class="version-config-section">
            <h4>环境变量</h4>
            <CodeEditor
              :model-value="selectedVersionDetail.container_env || '# 未配置'"
              language="shell"
              height="100px"
              :read-only="true"
            />
          </div>
        </template>

        <!-- Kubernetes 版本详情 -->
        <template v-else-if="selectedProject?.type === 'kubernetes'">
          <el-descriptions :column="2" border class="mt-16" title="Kubernetes 配置">
            <el-descriptions-item label="镜像">
              <el-tag type="info">{{ selectedVersionDetail.container_image || selectedProject?.k8s_config?.image || '未配置' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="命名空间">{{ selectedProject?.k8s_config?.namespace || 'default' }}</el-descriptions-item>
            <el-descriptions-item label="资源名">{{ selectedProject?.k8s_config?.resource_name || selectedProject?.name }}</el-descriptions-item>
            <el-descriptions-item label="副本数">{{ selectedVersionDetail.replicas || selectedProject?.k8s_config?.replicas || 1 }}</el-descriptions-item>
          </el-descriptions>
          <el-descriptions :column="4" border class="mt-16" title="资源限制" v-if="selectedVersionDetail.cpu_request || selectedVersionDetail.cpu_limit || selectedVersionDetail.memory_request || selectedVersionDetail.memory_limit">
            <el-descriptions-item label="CPU Request">{{ selectedVersionDetail.cpu_request || '-' }}</el-descriptions-item>
            <el-descriptions-item label="CPU Limit">{{ selectedVersionDetail.cpu_limit || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Memory Request">{{ selectedVersionDetail.memory_request || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Memory Limit">{{ selectedVersionDetail.memory_limit || '-' }}</el-descriptions-item>
          </el-descriptions>
          <div v-if="selectedVersionDetail.container_env" class="version-config-section">
            <h4>环境变量</h4>
            <CodeEditor
              :model-value="selectedVersionDetail.container_env || '# 未配置'"
              language="shell"
              height="100px"
              :read-only="true"
            />
          </div>
          <div v-if="selectedVersionDetail.k8s_yaml" class="version-config-section">
            <h4>YAML 配置</h4>
            <CodeEditor
              :model-value="selectedVersionDetail.k8s_yaml || '# 未配置'"
              language="yaml"
              height="250px"
              :read-only="true"
            />
          </div>
        </template>

        <!-- Git Pull 版本详情 -->
        <template v-else-if="selectedProject?.type === 'gitpull'">
          <el-descriptions :column="2" border class="mt-16" title="Git 配置">
            <el-descriptions-item label="仓库地址">{{ selectedProject?.gitpull_config?.repo_url || '未配置' }}</el-descriptions-item>
            <el-descriptions-item label="目标 Ref">
              <el-tag type="info">{{ selectedVersionDetail.git_ref || selectedProject?.gitpull_config?.branch || 'main' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Ref 类型">{{ selectedVersionDetail.git_ref_type || 'branch' }}</el-descriptions-item>
            <el-descriptions-item label="工作目录">{{ selectedProject?.gitpull_config?.work_dir || '/opt/app' }}</el-descriptions-item>
          </el-descriptions>
          <div v-if="selectedVersionDetail.pre_script" class="version-config-section">
            <h4>部署前脚本</h4>
            <CodeEditor
              :model-value="selectedVersionDetail.pre_script || '# 未配置'"
              language="shell"
              height="150px"
              :read-only="true"
            />
          </div>
          <div v-if="selectedVersionDetail.post_script" class="version-config-section">
            <h4>部署后脚本</h4>
            <CodeEditor
              :model-value="selectedVersionDetail.post_script || '# 未配置'"
              language="shell"
              height="150px"
              :read-only="true"
            />
          </div>
          <div v-if="!selectedVersionDetail.pre_script && !selectedVersionDetail.post_script && selectedVersionDetail.install_script" class="version-config-section">
            <h4>部署脚本</h4>
            <CodeEditor
              :model-value="selectedVersionDetail.install_script || '# 未配置'"
              language="shell"
              height="200px"
              :read-only="true"
            />
          </div>
        </template>

        <!-- 脚本版本详情 -->
        <template v-else>
          <el-descriptions :column="2" border class="mt-16">
            <el-descriptions-item label="工作目录">{{ selectedVersionDetail.work_dir || '/opt/app' }}</el-descriptions-item>
            <el-descriptions-item label="部署次数">{{ selectedVersionDetail.deploy_count || 0 }}</el-descriptions-item>
          </el-descriptions>
          <el-tabs class="script-tabs">
            <el-tab-pane label="安装脚本">
              <CodeEditor
                :model-value="selectedVersionDetail.install_script || '# 未配置'"
                language="shell"
                height="250px"
                :read-only="true"
              />
            </el-tab-pane>
            <el-tab-pane label="升级脚本">
              <CodeEditor
                :model-value="selectedVersionDetail.update_script || '# 未配置'"
                language="shell"
                height="250px"
                :read-only="true"
              />
            </el-tab-pane>
            <el-tab-pane label="回滚脚本">
              <CodeEditor
                :model-value="selectedVersionDetail.rollback_script || '# 未配置'"
                language="shell"
                height="250px"
                :read-only="true"
              />
            </el-tab-pane>
            <el-tab-pane label="卸载脚本">
              <CodeEditor
                :model-value="selectedVersionDetail.uninstall_script || '# 未配置'"
                language="shell"
                height="250px"
                :read-only="true"
              />
            </el-tab-pane>
          </el-tabs>
        </template>
      </template>
    </el-drawer>

    <!-- 日志详情抽屉 -->
    <el-drawer v-model="logDetailVisible" title="部署日志详情" size="50%">
      <template v-if="selectedLog">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="客户端">{{ selectedLog.client_id }}</el-descriptions-item>
          <el-descriptions-item label="版本">
            <el-tag size="small">{{ selectedLog.version }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="操作">
            <el-tag :type="getOperationTag(selectedLog.operation)">
              {{ getOperationLabel(selectedLog.operation) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getLogStatusTag(selectedLog.status)">
              {{ getLogStatusLabel(selectedLog.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="金丝雀">
            {{ selectedLog.is_canary ? '是' : '否' }}
          </el-descriptions-item>
          <el-descriptions-item label="耗时">
            {{ formatDuration(selectedLog.duration) }}
          </el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ formatTime(selectedLog.started_at) }}</el-descriptions-item>
          <el-descriptions-item label="结束时间">{{ formatTime(selectedLog.finished_at) }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="selectedLog.output" class="log-section">
          <h4>执行输出</h4>
          <CodeEditor
            :model-value="selectedLog.output"
            language="plaintext"
            height="200px"
            :read-only="true"
          />
        </div>

        <div v-if="selectedLog.error" class="log-section error">
          <h4>错误信息</h4>
          <CodeEditor
            :model-value="selectedLog.error"
            language="plaintext"
            height="150px"
            :read-only="true"
          />
        </div>
      </template>
    </el-drawer>

    <!-- Docker 配置对话框 -->
    <DockerConfigDialog
      v-model="dockerConfigDialogVisible"
      :initial-config="projectForm.container_config"
      @save="handleDockerConfigSave"
    />

    <!-- 容器日志对话框 -->
    <el-dialog
      v-model="containerLogsDialogVisible"
      title="容器日志"
      width="80%"
      destroy-on-close
      @close="closeContainerLogs"
    >
      <div class="container-logs-header">
        <div class="logs-info">
          <el-tag type="info" size="small">{{ currentLogsClient }}</el-tag>
          <el-tag type="success" size="small" class="ml-2">{{ currentLogsContainer }}</el-tag>
        </div>
        <div class="logs-actions">
          <el-switch v-model="logsAutoScroll" active-text="自动滚动" />
          <el-button size="small" @click="clearLogs">
            <el-icon><Delete /></el-icon>
            清空
          </el-button>
          <el-button
            size="small"
            :type="logsStreaming ? 'danger' : 'primary'"
            @click="toggleLogsStream"
          >
            <el-icon><component :is="logsStreaming ? 'VideoPause' : 'VideoPlay'" /></el-icon>
            {{ logsStreaming ? '停止' : '开始' }}
          </el-button>
        </div>
      </div>
      <div class="container-logs-content" ref="logsContainerRef">
        <pre class="logs-pre" v-if="containerLogs">{{ containerLogs }}</pre>
        <el-empty v-else description="暂无日志" :image-size="60" />
      </div>
      <div class="container-logs-footer" v-if="logsError">
        <el-alert :title="logsError" type="error" :closable="false" show-icon />
      </div>
    </el-dialog>

    <!-- 选择客户端查看日志对话框 -->
    <el-dialog
      v-model="selectClientForLogsVisible"
      title="选择客户端查看日志"
      width="500px"
      destroy-on-close
    >
      <div class="select-client-list">
        <div
          v-for="client in logsTaskClients"
          :key="client.client_id"
          class="select-client-item"
          @click="selectClientForLogs(client)"
        >
          <div class="client-info">
            <span class="client-id">{{ client.client_id }}</span>
            <el-tag
              size="small"
              :type="client.status === 'success' ? 'success' : client.status === 'failed' ? 'danger' : 'info'"
            >
              {{ getResultStatusLabel(client.status) }}
            </el-tag>
          </div>
          <div class="client-container" v-if="client.container_id">
            容器: {{ client.container_id.substring(0, 12) }}
          </div>
        </div>
        <el-empty v-if="logsTaskClients.length === 0" description="暂无客户端" :image-size="60" />
      </div>
    </el-dialog>

    <!-- 实时部署日志对话框 -->
    <el-dialog
      v-model="deployLogsDialogVisible"
      title="实时部署日志"
      width="80%"
      destroy-on-close
      @close="closeDeployLogs"
    >
      <div class="deploy-logs-header">
        <div class="logs-info">
          <el-tag type="info" size="small">{{ deployLogsTask?.version }}</el-tag>
          <el-tag :type="getTaskStatusTag(deployLogsTask?.status)" size="small" class="ml-2">
            {{ getTaskStatusLabel(deployLogsTask?.status) }}
          </el-tag>
          <span class="progress-mini ml-2" v-if="deployLogsTask">
            成功: {{ deployLogsTask.success_count || 0 }} /
            失败: {{ deployLogsTask.failed_count || 0 }} /
            待执行: {{ deployLogsTask.pending_count || 0 }}
          </span>
        </div>
        <div class="logs-actions">
          <el-switch v-model="deployLogsAutoScroll" active-text="自动滚动" />
          <el-button size="small" @click="clearDeployLogs">
            <el-icon><Delete /></el-icon>
            清空
          </el-button>
        </div>
      </div>
      <div class="deploy-logs-content" ref="deployLogsContainerRef">
        <div
          v-for="(log, index) in deployTaskLogs"
          :key="index"
          class="deploy-log-item"
          :class="log.level"
        >
          <span class="log-time">{{ formatLogTime(log.timestamp) }}</span>
          <el-tag :type="getLogLevelTag(log.level)" size="small" class="log-level">
            {{ log.level.toUpperCase() }}
          </el-tag>
          <span class="log-client" v-if="log.client_id">[{{ log.client_id }}]</span>
          <span class="log-stage" v-if="log.stage">[{{ log.stage }}]</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
        <el-empty v-if="deployTaskLogs.length === 0" description="暂无日志，任务开始后将显示实时日志" :image-size="60" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Refresh, MoreFilled, Download, Upload, RefreshLeft, Delete, Setting, Promotion,
  Document, VideoPlay, VideoPause, View, DocumentCopy, Box, Grid, Key, InfoFilled, Check, SuccessFilled
} from '@element-plus/icons-vue'
import api from '@/api'
import CodeEditor from '@/components/CodeEditor.vue'
import { DockerConfigDialog } from '@/components/docker'

const router = useRouter()

// ==================== 状态 ====================
const loading = ref(false)
const loadingVersions = ref(false)
const loadingTasks = ref(false)
const loadingLogs = ref(false)
const loadingGlobalStats = ref(false)
const submitting = ref(false)

const projects = ref([])
const versions = ref([])
const tasks = ref([])
const deployLogs = ref([])
const deployStats = ref(null)
const globalDeployStats = ref(null)
const globalRecentLogs = ref([])
// 已移除 clients 变量，客户端列表改为按需从项目安装信息获取

const selectedProject = ref(null)
const selectedVersion = ref(null)
const selectedTask = ref(null)
const selectedVersionDetail = ref(null)
const selectedLog = ref(null)

const activeTab = ref('versions')
const taskStatusFilter = ref('')
const scriptTab = ref('install')

// Git 版本相关状态
const loadingGitVersions = ref(false)
const gitVersionForm = reactive({
  version_type: 'tag',
  selected_tag: '',
  selected_branch: '',
  selected_commit: ''
})
const gitVersions = reactive({
  tags: [],
  branches: [],
  recent_commits: [],
  current_commit: '',
  current_branch: '',
  default_branch: ''
})

// 对话框状态
const projectDialogVisible = ref(false)
const versionDialogVisible = ref(false)
const taskDialogVisible = ref(false)
const taskDetailVisible = ref(false)
const versionDetailVisible = ref(false)
const logDetailVisible = ref(false)
const dockerConfigDialogVisible = ref(false)
const containerLogsDialogVisible = ref(false)
const editingProject = ref(null)

// 容器日志状态
const containerLogs = ref('')
const logsStreaming = ref(false)
const logsAutoScroll = ref(true)
const logsError = ref('')
const currentLogsClient = ref('')
const currentLogsContainer = ref('')
const logsContainerRef = ref(null)
const selectClientForLogsVisible = ref(false)
const logsTaskClients = ref([])
const currentLogsTask = ref(null)
let logsStreamHandle = null

// 实时部署日志状态
const deployLogsDialogVisible = ref(false)
const deployTaskLogs = ref([])
const deployLogsStreaming = ref(false)
const deployLogsAutoScroll = ref(true)
const deployLogsTask = ref(null)
const deployLogsContainerRef = ref(null)
let deployLogsStreamHandle = null

// 配置预览状态
const configPreviewVisible = ref(false)
const configPreviewTab = ref('basic')

// ==================== 表单 ====================
const projectFormRef = ref(null)
const versionFormRef = ref(null)
const taskFormRef = ref(null)

const projectForm = reactive({
  name: '',
  description: '',
  type: 'script',
  // Git 拉取配置
  gitpull_config: {
    repo_url: '',
    branch: 'main',
    work_dir: '',
    auth_type: 'none',
    ssh_key: '',
    token: '',
    username: '',
    password: '',
    pre_script: '',
    post_script: ''
  },
  // 容器配置
  container_config: {
    image: '',
    container_name: '',
    restart_policy: 'unless-stopped',
    pre_script: '',
    post_script: ''
  },
  // Kubernetes 配置
  k8s_config: {
    namespace: 'default',
    resource_type: 'deployment',
    resource_name: '',
    container_name: '',
    image: '',
    registry: '',
    registry_user: '',
    registry_pass: '',
    image_pull_policy: 'IfNotPresent',
    image_pull_secret: '',
    replicas: 1,
    update_strategy: 'RollingUpdate',
    max_unavailable: '25%',
    max_surge: '25%',
    rollout_timeout: 300,
    service_type: '',
    kubeconfig: '',
    kube_context: '',
    pre_script: '',
    post_script: ''
  }
})

const projectRules = {
  name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择部署类型', trigger: 'change' }]
}

const versionForm = reactive({
  version: '',
  description: '',
  work_dir: '/opt/app',
  install_script: '',
  update_script: '',
  rollback_script: '',
  uninstall_script: '',
  // Git 相关
  git_ref: '',       // tag/branch/commit 值
  git_ref_type: '',  // tag/branch/commit 类型
  // 向后兼容字段（将迁移到 deploy_config）
  container_image: '',
  container_env: '',
  replicas: 1,
  k8s_yaml: '',
  // 新版部署配置（三层配置架构）
  deploy_config: {
    // 镜像配置
    image: '',
    // 环境变量（增量合并到项目配置）
    environment: {},
    // 资源限制覆盖
    resources: {
      cpu_request: '',
      cpu_limit: '',
      memory_request: '',
      memory_limit: ''
    },
    // K8s 副本数覆盖
    replicas: null,
    // 启动命令覆盖
    command: [],
    entrypoint: [],
    working_dir: '',
    // 健康检查覆盖
    health_check: null,
    // 部署脚本
    pre_script: '',
    post_script: '',
    // K8s YAML 覆盖
    k8s_yaml_patch: '',
    k8s_yaml_full: ''
  }
})

const versionRules = {
  version: [{ required: true, message: '请输入版本号', trigger: 'blur' }]
  // install_script 不再是必须的，因为 Git/容器/K8s 项目不需要
}

// 项目回调配置列表
const projectCallbackConfigs = ref([])
const loadingCallbackConfigs = ref(false)

const taskForm = reactive({
  operation: 'deploy',
  client_ids: [],
  client_ids_text: '', // 客户端ID文本输入（每行一个）
  schedule_type: 'immediate',
  schedule_time: null,
  canary_enabled: true,
  canary_percent: 10,
  canary_duration: 30,
  canary_auto_promote: false,
  failure_strategy: 'continue',
  auto_rollback: true,
  // 回调通知配置
  callback_enabled: true,
  callback_config_id: '', // 选择的回调配置ID
  callback_events: ['canary_completed', 'full_completed'], // 启用的事件类型
  // 向后兼容字段
  container_image: '',
  container_env: '',
  image_pull_policy: 'ifnotpresent',
  k8s_image: '',
  k8s_replicas: null,
  k8s_revision: 0,
  k8s_timeout: 300,
  git_ref: '',
  git_rollback_ref: '',
  run_post_script: true,
  git_force: false,
  // 新版任务覆盖配置（三层配置架构）
  override_config: {
    // 镜像覆盖（临时测试未发布的镜像）
    image: '',
    // 环境变量追加（追加到版本环境变量之上）
    environment_add: {},
    // 资源覆盖（紧急扩容或金丝雀使用不同资源）
    resources: {
      cpu_request: '',
      cpu_limit: '',
      memory_request: '',
      memory_limit: ''
    },
    // 副本数覆盖（紧急扩缩容）
    replicas: null,
    // 启动命令覆盖（调试用）
    command: []
  }
})

const taskRules = {
  operation: [{ required: true, message: '请选择操作类型', trigger: 'change' }],
  client_ids_text: [
    { required: true, message: '请输入目标客户端ID', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value || !value.trim()) {
          callback(new Error('请输入至少一个客户端ID'))
          return
        }
        const ids = value.split('\n').map(line => line.trim()).filter(line => line.length > 0)
        if (ids.length === 0) {
          callback(new Error('请输入至少一个客户端ID'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

// 定时执行快捷选项
const scheduleShortcuts = [
  {
    text: '今晚 22:00',
    value: () => {
      const start = new Date()
      start.setHours(22, 0, 0, 0)
      const end = new Date(start)
      end.setHours(23, 59)
      return [start, end]
    }
  },
  {
    text: '明天凌晨',
    value: () => {
      const start = new Date()
      start.setDate(start.getDate() + 1)
      start.setHours(2, 0, 0, 0)
      const end = new Date(start)
      end.setHours(5, 0)
      return [start, end]
    }
  },
  {
    text: '周末',
    value: () => {
      const start = new Date()
      const day = start.getDay()
      const daysUntilSat = (6 - day + 7) % 7 || 7
      start.setDate(start.getDate() + daysUntilSat)
      start.setHours(10, 0, 0, 0)
      const end = new Date(start)
      end.setHours(18, 0)
      return [start, end]
    }
  }
]

// ==================== 计算属性 ====================
const filteredTasks = computed(() => {
  if (!taskStatusFilter.value) return tasks.value
  return tasks.value.filter(t => t.status === taskStatusFilter.value)
})

// 已移除 availableClients 计算属性，客户端通过多行文本输入，不再需要下拉列表

// 判断是否有高级 Docker 配置
const hasAdvancedDockerConfig = computed(() => {
  const cfg = projectForm.container_config
  return cfg.ports?.length > 0 ||
         cfg.volumes?.length > 0 ||
         cfg.memory_limit ||
         cfg.cpu_limit ||
         cfg.privileged ||
         cfg.networks?.length > 0 ||
         cfg.devices?.length > 0
})

// 辅助函数：解析环境变量字符串（优化性能）
function parseEnvString(envStr) {
  if (!envStr || typeof envStr !== 'string') return {}
  const env = {}
  // 使用正则表达式一次性匹配，避免多次字符串操作
  const lines = envStr.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line || line.startsWith('#')) continue
    const idx = line.indexOf('=')
    if (idx > 0) {
      const key = line.substring(0, idx).trim()
      const value = line.substring(idx + 1).trim()
      if (key) {
        env[key] = value
      }
    }
  }
  return env
}

// 配置预览 - 合并后的最终配置
const mergedConfigPreview = computed(() => {
  const project = selectedProject.value
  const version = selectedVersion.value
  
  // 早期返回，避免不必要的计算
  if (!project) return {}

  // 使用局部变量访问 taskForm，减少响应式追踪
  const overrideConfig = taskForm.override_config
  const taskContainerEnv = taskForm.container_env

  const result = {
    image: '',
    containerName: '',
    replicas: 1,
    restartPolicy: '',
    cpuRequest: '',
    cpuLimit: '',
    memoryRequest: '',
    memoryLimit: '',
    environment: {}
  }

  if (project.type === 'container') {
    // 项目级配置
    result.image = project.container_config?.image || ''
    result.containerName = project.container_config?.container_name || ''
    result.restartPolicy = project.container_config?.restart_policy || 'unless-stopped'
    result.cpuLimit = project.container_config?.cpu_limit || ''
    result.memoryLimit = project.container_config?.memory_limit || ''
    // 合并项目环境变量
    if (project.container_config?.environment) {
      Object.assign(result.environment, project.container_config.environment)
    }

    // 版本级覆盖
    if (version?.container_image) result.image = version.container_image
    if (version?.deploy_config?.resources?.cpu_limit) result.cpuLimit = version.deploy_config.resources.cpu_limit
    if (version?.deploy_config?.resources?.memory_limit) result.memoryLimit = version.deploy_config.resources.memory_limit

    // 任务级覆盖
    if (overrideConfig?.image) result.image = overrideConfig.image
    if (overrideConfig?.resources?.cpu_limit) result.cpuLimit = overrideConfig.resources.cpu_limit
    if (overrideConfig?.resources?.memory_limit) result.memoryLimit = overrideConfig.resources.memory_limit
  } else if (project.type === 'kubernetes') {
    // 项目级配置
    result.image = project.k8s_config?.image || ''
    result.replicas = project.k8s_config?.replicas || 1
    result.cpuRequest = project.k8s_config?.cpu_request || ''
    result.cpuLimit = project.k8s_config?.cpu_limit || ''
    result.memoryRequest = project.k8s_config?.memory_request || ''
    result.memoryLimit = project.k8s_config?.memory_limit || ''
    // 合并项目环境变量
    if (project.k8s_config?.environment) {
      Object.assign(result.environment, project.k8s_config.environment)
    }

    // 版本级覆盖
    if (version?.container_image) result.image = version.container_image
    if (version?.deploy_config?.replicas) result.replicas = version.deploy_config.replicas
    if (version?.deploy_config?.resources?.cpu_request) result.cpuRequest = version.deploy_config.resources.cpu_request
    if (version?.deploy_config?.resources?.cpu_limit) result.cpuLimit = version.deploy_config.resources.cpu_limit
    if (version?.deploy_config?.resources?.memory_request) result.memoryRequest = version.deploy_config.resources.memory_request
    if (version?.deploy_config?.resources?.memory_limit) result.memoryLimit = version.deploy_config.resources.memory_limit

    // 任务级覆盖
    if (overrideConfig?.image) result.image = overrideConfig.image
    if (overrideConfig?.replicas) result.replicas = overrideConfig.replicas
    if (overrideConfig?.resources?.cpu_request) result.cpuRequest = overrideConfig.resources.cpu_request
    if (overrideConfig?.resources?.cpu_limit) result.cpuLimit = overrideConfig.resources.cpu_limit
    if (overrideConfig?.resources?.memory_request) result.memoryRequest = overrideConfig.resources.memory_request
    if (overrideConfig?.resources?.memory_limit) result.memoryLimit = overrideConfig.resources.memory_limit
  }

  // 合并版本环境变量（优化：使用辅助函数）
  if (version?.container_env) {
    Object.assign(result.environment, parseEnvString(version.container_env))
  }

  // 合并任务环境变量（优化：使用辅助函数）
  if (taskContainerEnv) {
    Object.assign(result.environment, parseEnvString(taskContainerEnv))
  }

  return result
})

// 配置来源追踪
const configPreviewSource = computed(() => {
  const project = selectedProject.value
  const version = selectedVersion.value
  
  // 早期返回
  if (!project) return {}

  // 使用局部变量访问 taskForm，减少响应式追踪
  const overrideConfig = taskForm.override_config

  const source = {}

  // 镜像来源
  if (overrideConfig?.image) {
    source.image = 'task'
  } else if (version?.container_image) {
    source.image = 'version'
  } else {
    source.image = 'project'
  }

  // 副本数来源 (K8s)
  if (project.type === 'kubernetes') {
    if (overrideConfig?.replicas) {
      source.replicas = 'task'
    } else if (version?.deploy_config?.replicas) {
      source.replicas = 'version'
    } else {
      source.replicas = 'project'
    }
  }

  // CPU Limit 来源
  if (overrideConfig?.resources?.cpu_limit) {
    source.cpuLimit = 'task'
  } else if (version?.deploy_config?.resources?.cpu_limit) {
    source.cpuLimit = 'version'
  } else if (project.container_config?.cpu_limit || project.k8s_config?.cpu_limit) {
    source.cpuLimit = 'project'
  }

  // Memory Limit 来源
  if (overrideConfig?.resources?.memory_limit) {
    source.memoryLimit = 'task'
  } else if (version?.deploy_config?.resources?.memory_limit) {
    source.memoryLimit = 'version'
  } else if (project.container_config?.memory_limit || project.k8s_config?.memory_limit) {
    source.memoryLimit = 'project'
  }

  return source
})

// 环境变量预览数据（带来源标记）
const environmentPreviewData = computed(() => {
  const project = selectedProject.value
  const version = selectedVersion.value
  
  // 早期返回
  if (!project) return []

  // 使用局部变量访问 taskForm，减少响应式追踪
  const taskContainerEnv = taskForm.container_env

  const envSources = {}

  // 收集项目环境变量
  const projectEnv = project.type === 'container'
    ? project.container_config?.environment
    : project.k8s_config?.environment
  if (projectEnv) {
    for (const [key, value] of Object.entries(projectEnv)) {
      envSources[key] = { value, source: 'project' }
    }
  }

  // 收集版本环境变量（优化：使用辅助函数）
  if (version?.container_env) {
    const versionEnv = parseEnvString(version.container_env)
    for (const [key, value] of Object.entries(versionEnv)) {
      envSources[key] = { value, source: 'version' }
    }
  }

  // 收集任务环境变量（优化：使用辅助函数）
  if (taskContainerEnv) {
    const taskEnv = parseEnvString(taskContainerEnv)
    for (const [key, value] of Object.entries(taskEnv)) {
      envSources[key] = { value, source: 'task' }
    }
  }

  // 转换为数组并排序
  const result = Object.entries(envSources).map(([key, data]) => ({
    key,
    value: data.value,
    source: data.source
  }))

  return result.sort((a, b) => a.key.localeCompare(b.key))
})

// ==================== 数据加载 ====================
async function loadData() {
  loading.value = true
  try {
    // 只加载项目列表和全局统计，客户端ID通过文本框输入
    await Promise.all([loadProjects(), loadGlobalStats()])
  } finally {
    loading.value = false
  }
}

async function loadProjects() {
  try {
    const res = await api.getProjects()
    if (res.success) {
      projects.value = res.projects || []
    }
  } catch (e) {
    projects.value = []
  }
}

async function loadGlobalStats() {
  loadingGlobalStats.value = true
  try {
    // 获取全局部署统计
    const statsRes = await api.getDeployStats({ days: 30 })
    if (statsRes.success !== false) {
      globalDeployStats.value = {
        ...statsRes.stats,
        running_count: statsRes.running_count || 0
      }
    }

    // 获取最近部署日志
    const logsRes = await api.getDeployLogs({ limit: 10 })
    if (logsRes.success !== false) {
      globalRecentLogs.value = logsRes.logs || []
    }
  } catch (e) {
    globalDeployStats.value = null
    globalRecentLogs.value = []
  } finally {
    loadingGlobalStats.value = false
  }
}

async function loadVersions() {
  if (!selectedProject.value) return
  loadingVersions.value = true
  try {
    const res = await api.getVersions(selectedProject.value.id)
    if (res.success) {
      versions.value = res.versions || []
    }
  } catch (e) {
    versions.value = []
  } finally {
    loadingVersions.value = false
  }
}

async function loadTasks() {
  if (!selectedProject.value) return
  loadingTasks.value = true
  try {
    const res = await api.getDeployTasks(selectedProject.value.id)
    if (res.success) {
      tasks.value = res.tasks || []
    }
  } catch (e) {
    tasks.value = []
  } finally {
    loadingTasks.value = false
  }
}

async function loadDeployLogs() {
  if (!selectedProject.value) return
  loadingLogs.value = true
  try {
    const res = await api.getProjectDeployLogs(selectedProject.value.id, { limit: 50 })
    if (res.success) {
      deployLogs.value = res.logs || []
    }
  } catch (e) {
    deployLogs.value = []
  } finally {
    loadingLogs.value = false
  }
}

async function loadDeployStats() {
  if (!selectedProject.value) return
  try {
    const res = await api.getProjectDeployStats(selectedProject.value.id, { days: 30 })
    if (res.success) {
      deployStats.value = res.stats
    }
  } catch (e) {
    deployStats.value = null
  }
}

function selectProject(project) {
  selectedProject.value = project
  activeTab.value = 'versions'
  loadVersions()
  loadTasks()
  loadDeployLogs()
  loadDeployStats()
}

// ==================== 项目管理 ====================
function showCreateProject() {
  editingProject.value = null
  projectForm.name = ''
  projectForm.description = ''
  projectForm.type = 'script'
  // 重置 Git 配置
  projectForm.gitpull_config = {
    repo_url: '',
    branch: 'main',
    work_dir: '',
    auth_type: 'none',
    ssh_key: '',
    token: '',
    username: '',
    password: '',
    pre_script: '',
    post_script: ''
  }
  // 重置容器配置
  projectForm.container_config = {
    image: '',
    container_name: '',
    restart_policy: 'unless-stopped',
    pre_script: '',
    post_script: ''
  }
  // 重置 K8s 配置
  projectForm.k8s_config = {
    namespace: 'default',
    resource_type: 'deployment',
    resource_name: '',
    container_name: '',
    image: '',
    registry: '',
    registry_user: '',
    registry_pass: '',
    image_pull_policy: 'IfNotPresent',
    image_pull_secret: '',
    replicas: 1,
    update_strategy: 'RollingUpdate',
    max_unavailable: '25%',
    max_surge: '25%',
    rollout_timeout: 300,
    service_type: '',
    kubeconfig: '',
    kube_context: '',
    pre_script: '',
    post_script: ''
  }
  projectDialogVisible.value = true
}

async function saveProject() {
  const valid = await projectFormRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    let res
    if (editingProject.value) {
      res = await api.updateProject(editingProject.value.id, projectForm)
    } else {
      res = await api.createProject(projectForm)
    }
    if (res.success) {
      ElMessage.success('保存成功')
      projectDialogVisible.value = false
      await loadProjects()
    } else {
      ElMessage.error(res.error || '保存失败')
    }
  } finally {
    submitting.value = false
  }
}

async function handleProjectAction(action, project) {
  if (action === 'edit') {
    editingProject.value = project
    projectForm.name = project.name
    projectForm.description = project.description || ''
    projectForm.type = project.type
    // 加载 Git 配置
    if (project.gitpull_config) {
      Object.assign(projectForm.gitpull_config, project.gitpull_config)
    }
    // 加载容器配置
    if (project.container_config) {
      Object.assign(projectForm.container_config, project.container_config)
    }
    // 加载 K8s 配置
    if (project.k8s_config) {
      Object.assign(projectForm.k8s_config, project.k8s_config)
    }
    projectDialogVisible.value = true
  } else if (action === 'delete') {
    try {
      await ElMessageBox.confirm('确定要删除此项目吗？相关版本和部署记录都会被删除', '确认')
      const res = await api.deleteProject(project.id)
      if (res.success) {
        ElMessage.success('删除成功')
        if (selectedProject.value?.id === project.id) {
          selectedProject.value = null
        }
        await loadProjects()
      }
    } catch (e) {
      if (e !== 'cancel') ElMessage.error(e.message)
    }
  }
}

// 处理 Docker 配置保存
function handleDockerConfigSave(dockerConfig) {
  // 合并 Docker 配置到 container_config
  Object.assign(projectForm.container_config, dockerConfig)
  ElMessage.success('Docker 配置已更新')
}

// ==================== 版本管理 ====================
function showCreateVersion() {
  versionForm.version = ''
  versionForm.description = ''
  versionForm.work_dir = '/opt/app'
  versionForm.install_script = ''
  versionForm.update_script = ''
  versionForm.rollback_script = ''
  versionForm.uninstall_script = ''
  // 容器/K8s 字段
  versionForm.container_image = ''
  versionForm.container_env = ''
  versionForm.replicas = 1
  versionForm.k8s_yaml = ''
  // K8s 资源限制
  versionForm.cpu_request = ''
  versionForm.cpu_limit = ''
  versionForm.memory_request = ''
  versionForm.memory_limit = ''
  // Git 字段
  versionForm.git_ref = ''
  versionForm.git_ref_type = ''
  versionForm.pre_script = ''
  versionForm.post_script = ''
  // 重置 Git 版本表单
  gitVersionForm.version_type = 'tag'
  gitVersionForm.selected_tag = ''
  gitVersionForm.selected_branch = ''
  gitVersionForm.selected_commit = ''
  // 重置 Git 版本数据
  gitVersions.tags = []
  gitVersions.branches = []
  gitVersions.recent_commits = []
  gitVersions.current_commit = ''
  gitVersions.current_branch = ''
  gitVersions.default_branch = ''

  versionDialogVisible.value = true
}

// 获取 Git 仓库版本信息（Server 端直接执行，不需要选择 Client）
async function fetchGitVersions() {
  if (!selectedProject.value) return

  loadingGitVersions.value = true
  try {
    const res = await api.getGitVersions({
      project_id: selectedProject.value.id,
      repo_url: selectedProject.value.gitpull_config?.repo_url || '',
      auth_type: selectedProject.value.gitpull_config?.auth_type || 'none',
      token: selectedProject.value.gitpull_config?.token || '',
      username: selectedProject.value.gitpull_config?.username || '',
      password: selectedProject.value.gitpull_config?.password || '',
      max_tags: 30,
      max_commits: 20,
      include_branches: true
    })

    if (res.success !== false) {
      gitVersions.tags = res.tags || []
      gitVersions.branches = res.branches || []
      gitVersions.recent_commits = res.recent_commits || []
      gitVersions.current_commit = res.current_commit || ''
      gitVersions.current_branch = res.current_branch || ''
      gitVersions.default_branch = res.default_branch || ''
      ElMessage.success(`获取成功: ${gitVersions.tags.length} 个 Tag, ${gitVersions.branches.length} 个分支`)

      // 自动选择第一个 tag 并填充版本信息
      if (gitVersions.tags.length > 0) {
        const firstTag = gitVersions.tags[0]
        gitVersionForm.version_type = 'tag'
        gitVersionForm.selected_tag = firstTag.name
        // 将 tag 名称填充到版本号
        versionForm.version = firstTag.name
        // 将 tag 的提交信息填充到版本说明
        versionForm.description = firstTag.message || `${firstTag.name} - ${firstTag.commit?.substring(0, 7) || ''}`
      }
    } else {
      ElMessage.error(res.error || '获取 Git 版本失败')
    }
  } catch (e) {
    ElMessage.error(e.message || '获取 Git 版本失败')
  } finally {
    loadingGitVersions.value = false
  }
}

// 当选择 tag 时自动填充版本号和说明
function onTagSelected(tagName) {
  const tag = gitVersions.tags.find(t => t.name === tagName)
  if (tag) {
    versionForm.version = tag.name
    versionForm.description = tag.message || `${tag.name} - ${tag.commit?.substring(0, 7) || ''}`
  }
}

async function saveVersion() {
  const valid = await versionFormRef.value?.validate().catch(() => false)
  if (!valid) return

  // 根据项目类型设置 Git ref
  if (selectedProject.value?.type === 'gitpull') {
    switch (gitVersionForm.version_type) {
      case 'tag':
        versionForm.git_ref = gitVersionForm.selected_tag
        versionForm.git_ref_type = 'tag'
        break
      case 'branch':
        versionForm.git_ref = gitVersionForm.selected_branch
        versionForm.git_ref_type = 'branch'
        break
      case 'commit':
        versionForm.git_ref = gitVersionForm.selected_commit
        versionForm.git_ref_type = 'commit'
        break
    }
  }

  submitting.value = true
  try {
    // 构建请求数据
    const requestData = {
      version: versionForm.version,
      description: versionForm.description,
      work_dir: versionForm.work_dir,
      install_script: versionForm.install_script,
      update_script: versionForm.update_script,
      rollback_script: versionForm.rollback_script,
      uninstall_script: versionForm.uninstall_script,
      git_ref: versionForm.git_ref,
      git_ref_type: versionForm.git_ref_type,
      // 向后兼容：同时发送旧字段
      container_image: versionForm.container_image || versionForm.deploy_config.image,
      container_env: versionForm.container_env,
      replicas: versionForm.replicas,
      k8s_yaml: versionForm.k8s_yaml || versionForm.deploy_config.k8s_yaml_full
    }

    // 构建新版 deploy_config（仅当有值时才发送）
    const projectType = selectedProject.value?.type
    if (projectType === 'container' || projectType === 'kubernetes') {
      const deployConfig = {}

      // 镜像
      const image = versionForm.deploy_config.image || versionForm.container_image
      if (image) deployConfig.image = image

      // 环境变量（从文本格式转换为 map）
      const envText = versionForm.container_env
      if (envText) {
        const envMap = {}
        envText.split('\n').forEach(line => {
          const idx = line.indexOf('=')
          if (idx > 0) {
            envMap[line.substring(0, idx).trim()] = line.substring(idx + 1).trim()
          }
        })
        if (Object.keys(envMap).length > 0) {
          deployConfig.environment = envMap
        }
      }

      // 资源限制
      const resources = versionForm.deploy_config.resources
      if (resources.cpu_request || resources.cpu_limit || resources.memory_request || resources.memory_limit) {
        deployConfig.resources = {
          cpu_request: resources.cpu_request || undefined,
          cpu_limit: resources.cpu_limit || undefined,
          memory_request: resources.memory_request || undefined,
          memory_limit: resources.memory_limit || undefined
        }
      }

      // K8s 副本数
      if (projectType === 'kubernetes' && versionForm.replicas > 0) {
        deployConfig.replicas = versionForm.replicas
      }

      // 工作目录
      if (versionForm.deploy_config.working_dir) {
        deployConfig.working_dir = versionForm.deploy_config.working_dir
      }

      // 部署脚本
      if (versionForm.deploy_config.pre_script) {
        deployConfig.pre_script = versionForm.deploy_config.pre_script
      }
      if (versionForm.deploy_config.post_script) {
        deployConfig.post_script = versionForm.deploy_config.post_script
      }

      // K8s YAML
      if (projectType === 'kubernetes') {
        if (versionForm.deploy_config.k8s_yaml_full || versionForm.k8s_yaml) {
          deployConfig.k8s_yaml_full = versionForm.deploy_config.k8s_yaml_full || versionForm.k8s_yaml
        }
      }

      // 只有当有配置时才添加 deploy_config
      if (Object.keys(deployConfig).length > 0) {
        requestData.deploy_config = deployConfig
      }
    }

    const res = await api.createVersion(selectedProject.value.id, requestData)
    if (res.success) {
      ElMessage.success('版本创建成功')
      versionDialogVisible.value = false
      await loadVersions()
    } else {
      ElMessage.error(res.error || '创建失败')
    }
  } finally {
    submitting.value = false
  }
}

function viewVersion(version) {
  selectedVersionDetail.value = version
  versionDetailVisible.value = true
}

async function deleteVersion(version) {
  try {
    await ElMessageBox.confirm('确定要删除此版本吗？', '确认')
    const res = await api.deleteVersion(version.id)
    if (res.success) {
      ElMessage.success('删除成功')
      await loadVersions()
    } else {
      ElMessage.error(res.error || '删除失败')
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

// ==================== 部署任务 ====================
function showCreateTask(version) {
  selectedVersion.value = version
  // 根据项目类型设置默认操作
  const projectType = selectedProject.value?.type
  if (projectType === 'container' || projectType === 'kubernetes' || projectType === 'gitpull') {
    taskForm.operation = 'deploy'
  } else {
    taskForm.operation = 'install'
  }
  taskForm.client_ids = []
  taskForm.client_ids_text = ''
  taskForm.schedule_type = 'immediate'
  taskForm.schedule_time = null
  taskForm.canary_enabled = true
  taskForm.canary_percent = 10
  taskForm.canary_duration = 30
  taskForm.canary_auto_promote = false
  taskForm.failure_strategy = 'continue'
  taskForm.auto_rollback = true
  // 重置 Docker 配置
  taskForm.container_image = ''
  taskForm.container_env = ''
  taskForm.image_pull_policy = 'ifnotpresent'
  // 重置 K8s 配置
  taskForm.k8s_image = ''
  taskForm.k8s_replicas = null
  taskForm.k8s_revision = 0
  taskForm.k8s_timeout = 300
  // 重置 Git 配置
  taskForm.git_ref = ''
  taskForm.git_rollback_ref = ''
  taskForm.run_post_script = true
  taskForm.git_force = false
  // 重置 override_config（新版三层配置）
  // 使用 Object.assign 避免替换整个对象，减少响应式更新
  Object.assign(taskForm.override_config, {
    image: '',
    environment_add: {},
    resources: {
      cpu_request: '',
      cpu_limit: '',
      memory_request: '',
      memory_limit: ''
    },
    replicas: null,
    command: []
  })
  // 重置回调配置
  taskForm.callback_enabled = true
  taskForm.callback_config_id = ''
  taskForm.callback_events = ['canary_completed', 'full_completed']
  // 加载项目回调配置
  loadProjectCallbackConfigs()
  // 客户端ID通过多行文本框输入，不再自动加载客户端列表

  // 使用 nextTick 延迟对话框显示，避免在大量响应式更新时立即打开
  // 这样可以确保所有计算属性都已完成更新，避免卡顿
  nextTick(() => {
    taskDialogVisible.value = true
  })
}

// 加载项目回调配置
async function loadProjectCallbackConfigs() {
  if (!selectedProject.value?.id) {
    projectCallbackConfigs.value = []
    return
  }
  loadingCallbackConfigs.value = true
  try {
    const res = await api.getCallbackConfigs(selectedProject.value.id)
    if (res.success) {
      projectCallbackConfigs.value = res.data || []
      // 如果有配置，默认选择第一个
      if (projectCallbackConfigs.value.length > 0 && !taskForm.callback_config_id) {
        taskForm.callback_config_id = projectCallbackConfigs.value[0].id
      }
    } else {
      projectCallbackConfigs.value = []
    }
  } catch (e) {
    projectCallbackConfigs.value = []
  } finally {
    loadingCallbackConfigs.value = false
  }
}

// 获取渠道标签显示
function getChannelLabels(channels) {
  if (!channels || channels.length === 0) return '无渠道'
  const typeMap = {
    feishu: '飞书',
    dingtalk: '钉钉',
    wechat: '企微',
    custom: '自定义'
  }
  return channels
    .filter(ch => ch.enabled)
    .map(ch => typeMap[ch.type] || ch.type)
    .join('/')
}

// 跳转到回调配置页面
function goToCallbackConfig() {
  if (selectedProject.value?.id) {
    router.push(`/release/callbacks?project_id=${selectedProject.value.id}`)
  } else {
    router.push('/release/callbacks')
  }
}

async function selectAllClients() {
  // 从项目安装信息中获取所有已安装客户端
  if (!selectedProject.value?.id) {
    ElMessage.warning('请先选择项目')
    return
  }

  try {
    const res = await api.getProjectInstallations(selectedProject.value.id)
    if (res.success && res.installations && res.installations.length > 0) {
      const clientIds = res.installations.map(i => i.client_id).filter(id => id)
      taskForm.client_ids_text = [...new Set(clientIds)].join('\n')
      ElMessage.success(`已填充 ${clientIds.length} 个客户端`)
    } else {
      ElMessage.info('该项目暂无已安装的客户端')
    }
  } catch (e) {
    ElMessage.warning('获取客户端列表失败，请手动输入')
  }
}

// 从文本输入解析客户端ID数组
function parseClientIds() {
  if (!taskForm.client_ids_text || !taskForm.client_ids_text.trim()) {
    return []
  }
  return taskForm.client_ids_text
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
}

// 获取客户端ID数量
function getClientIdsCount() {
  return parseClientIds().length
}

async function createTask() {
  const valid = await taskFormRef.value?.validate().catch(() => false)
  if (!valid) return

  // 从文本输入解析客户端ID数组
  const clientIds = parseClientIds()
  if (clientIds.length === 0) {
    ElMessage.warning('请输入至少一个客户端ID')
    return
  }

  submitting.value = true
  try {
    const projectType = selectedProject.value?.type

    // 构建基础任务数据
    const taskData = {
      project_id: selectedProject.value.id,
      version_id: selectedVersion.value.id,
      operation: taskForm.operation,
      client_ids: clientIds,
      schedule_type: taskForm.schedule_type,
      schedule_from: taskForm.schedule_time?.[0],
      schedule_to: taskForm.schedule_time?.[1],
      canary_enabled: taskForm.canary_enabled,
      canary_percent: taskForm.canary_percent,
      canary_duration: taskForm.canary_duration,
      canary_auto_promote: taskForm.canary_auto_promote,
      failure_strategy: taskForm.failure_strategy,
      auto_rollback: taskForm.auto_rollback
    }

    // 添加回调配置
    if (taskForm.callback_enabled && taskForm.callback_config_id) {
      taskData.callback_config_id = taskForm.callback_config_id
      taskData.callback_events = taskForm.callback_events
    }

    // 向后兼容：添加旧字段
    if (projectType === 'container') {
      taskData.container_image = taskForm.container_image || taskForm.override_config.image
      taskData.container_env = taskForm.container_env
      taskData.image_pull_policy = taskForm.image_pull_policy
    } else if (projectType === 'kubernetes') {
      taskData.k8s_image = taskForm.k8s_image || taskForm.override_config.image
      taskData.k8s_replicas = taskForm.k8s_replicas || taskForm.override_config.replicas
      taskData.k8s_revision = taskForm.k8s_revision
      taskData.k8s_timeout = taskForm.k8s_timeout
    } else if (projectType === 'gitpull') {
      taskData.git_ref = taskForm.git_ref
      taskData.git_rollback_ref = taskForm.git_rollback_ref
      taskData.run_post_script = taskForm.run_post_script
      taskData.git_force = taskForm.git_force
    }

    // 构建新版 override_config（三层配置架构）
    if (projectType === 'container' || projectType === 'kubernetes') {
      const overrideConfig = {}

      // 镜像覆盖
      const image = taskForm.override_config.image ||
                    (projectType === 'container' ? taskForm.container_image : taskForm.k8s_image)
      if (image) overrideConfig.image = image

      // 环境变量追加（从文本格式转换）
      const envText = taskForm.container_env
      if (envText) {
        const envMap = {}
        envText.split('\n').forEach(line => {
          const idx = line.indexOf('=')
          if (idx > 0) {
            envMap[line.substring(0, idx).trim()] = line.substring(idx + 1).trim()
          }
        })
        if (Object.keys(envMap).length > 0) {
          overrideConfig.environment_add = envMap
        }
      }

      // 资源覆盖
      const resources = taskForm.override_config.resources
      if (resources.cpu_request || resources.cpu_limit || resources.memory_request || resources.memory_limit) {
        overrideConfig.resources = {
          cpu_request: resources.cpu_request || undefined,
          cpu_limit: resources.cpu_limit || undefined,
          memory_request: resources.memory_request || undefined,
          memory_limit: resources.memory_limit || undefined
        }
      }

      // 副本数覆盖
      const replicas = taskForm.override_config.replicas ||
                       (projectType === 'kubernetes' ? taskForm.k8s_replicas : null)
      if (replicas && replicas > 0) {
        overrideConfig.replicas = replicas
      }

      // 只有当有配置时才添加 override_config
      if (Object.keys(overrideConfig).length > 0) {
        taskData.override_config = overrideConfig
      }
    }

    const res = await api.createDeployTask(taskData)
    if (res.success) {
      ElMessage.success('部署任务创建成功')
      taskDialogVisible.value = false
      activeTab.value = 'tasks'
      await loadTasks()
    } else {
      ElMessage.error(res.error || '创建失败')
    }
  } finally {
    submitting.value = false
  }
}

async function startTask(task) {
  try {
    const res = await api.startDeployTask(task.id)
    if (res.success) {
      ElMessage.success('任务已开始')
      await loadTasks()
    } else {
      ElMessage.error(res.error || '启动失败')
    }
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function cancelTask(task) {
  try {
    await ElMessageBox.confirm('确定要取消此任务吗？', '确认')
    const res = await api.cancelDeployTask(task.id)
    if (res.success) {
      ElMessage.success('任务已取消')
      await loadTasks()
    } else {
      ElMessage.error(res.error || '取消失败')
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

async function pauseTask(task) {
  try {
    const res = await api.pauseDeployTask(task.id)
    if (res.success) {
      ElMessage.success('任务已暂停')
      await loadTasks()
    } else {
      ElMessage.error(res.error || '暂停失败')
    }
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function promoteTask(task) {
  try {
    await ElMessageBox.confirm('确定要全量发布吗？将对剩余节点执行部署', '确认全量发布')
    const res = await api.promoteDeployTask(task.id)
    if (res.success) {
      ElMessage.success('已开始全量发布')
      await loadTasks()
    } else {
      ElMessage.error(res.error || '全量发布失败')
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

async function rollbackTask(task) {
  try {
    await ElMessageBox.confirm('确定要回滚吗？将把已部署的金丝雀节点回滚到之前版本', '确认回滚')
    const res = await api.rollbackDeployTask(task.id)
    if (res.success) {
      ElMessage.success('已开始回滚')
      await loadTasks()
    } else {
      ElMessage.error(res.error || '回滚失败')
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.message)
  }
}

function viewTask(task) {
  selectedTask.value = task
  taskDetailVisible.value = true
}

function viewLog(log) {
  selectedLog.value = log
  logDetailVisible.value = true
}

// ==================== 容器日志 ====================

// 从任务列表打开日志选择对话框
function showTaskLogsDialog(task) {
  currentLogsTask.value = task

  // 获取任务的客户端列表
  const clients = task.results || []
  if (clients.length === 0) {
    // 如果任务还没有执行结果，使用任务的 client_ids
    const clientIds = task.client_ids || []
    logsTaskClients.value = clientIds.map(id => ({
      client_id: id,
      status: 'pending',
      container_id: ''
    }))
  } else {
    logsTaskClients.value = clients
  }

  // 如果只有一个客户端，直接打开日志
  if (logsTaskClients.value.length === 1) {
    selectClientForLogs(logsTaskClients.value[0])
    return
  }

  // 显示客户端选择对话框
  selectClientForLogsVisible.value = true
}

// 选择客户端查看日志
function selectClientForLogs(client) {
  selectClientForLogsVisible.value = false

  const containerId = client.container_id || ''
  const containerName = currentLogsTask.value?.container_name || selectedProject.value?.container_config?.container_name || ''

  if (!containerId && !containerName) {
    ElMessage.warning('无法获取容器标识')
    return
  }

  currentLogsClient.value = client.client_id
  currentLogsContainer.value = containerId ? containerId.substring(0, 12) : containerName
  containerLogs.value = ''
  logsError.value = ''
  logsStreaming.value = false
  containerLogsDialogVisible.value = true

  // 自动开始流式获取
  startLogsStream(client.client_id, containerId, containerName)
}

function showContainerLogs(row) {
  // 确定容器标识
  const containerId = row.container_id || ''
  const containerName = selectedTask.value?.container_name || selectedProject.value?.container_config?.container_name || ''

  if (!containerId && !containerName) {
    ElMessage.warning('无法获取容器标识')
    return
  }

  currentLogsClient.value = row.client_id
  currentLogsContainer.value = containerId ? containerId.substring(0, 12) : containerName
  containerLogs.value = ''
  logsError.value = ''
  logsStreaming.value = false
  containerLogsDialogVisible.value = true

  // 自动开始流式获取
  startLogsStream(row.client_id, containerId, containerName)
}

function startLogsStream(clientId, containerId, containerName) {
  if (logsStreamHandle) {
    logsStreamHandle.close()
    logsStreamHandle = null
  }

  logsStreaming.value = true
  logsError.value = ''

  logsStreamHandle = api.streamContainerLogs(
    {
      client_id: clientId,
      container_id: containerId,
      container_name: containerName,
      tail: 200,
      timestamps: true
    },
    // onLogs
    (event) => {
      if (event.logs) {
        containerLogs.value = event.logs
        // 自动滚动到底部
        if (logsAutoScroll.value && logsContainerRef.value) {
          setTimeout(() => {
            logsContainerRef.value.scrollTop = logsContainerRef.value.scrollHeight
          }, 50)
        }
      }
    },
    // onStart
    (event) => {
      console.log('Container logs stream started', event)
    },
    // onError
    (event) => {
      logsError.value = event.error || '获取日志失败'
      logsStreaming.value = false
    }
  )
}

function stopLogsStream() {
  if (logsStreamHandle) {
    logsStreamHandle.close()
    logsStreamHandle = null
  }
  logsStreaming.value = false
}

function toggleLogsStream() {
  if (logsStreaming.value) {
    stopLogsStream()
  } else {
    // 重新开始
    const containerId = selectedTask.value?.results?.find(r => r.client_id === currentLogsClient.value)?.container_id || ''
    const containerName = selectedTask.value?.container_name || selectedProject.value?.container_config?.container_name || ''
    startLogsStream(currentLogsClient.value, containerId, containerName)
  }
}

function clearLogs() {
  containerLogs.value = ''
}

function closeContainerLogs() {
  stopLogsStream()
  containerLogs.value = ''
  logsError.value = ''
}

// ==================== 配置预览 ====================

function showConfigPreview() {
  configPreviewTab.value = 'basic'
  configPreviewVisible.value = true
}

// ==================== 实时部署日志 ====================

function showDeployLogs(task) {
  deployLogsTask.value = task
  deployTaskLogs.value = []
  deployLogsDialogVisible.value = true

  // 如果任务还在运行，开始 SSE 流
  if (task.status === 'running' || task.status === 'canary' || task.status === 'pending') {
    startDeployLogsStream(task.id)
  } else {
    // 任务已完成，获取历史日志
    loadDeployTaskLogs(task.id)
  }
}

function startDeployLogsStream(taskId) {
  if (deployLogsStreamHandle) {
    deployLogsStreamHandle.close()
    deployLogsStreamHandle = null
  }

  deployLogsStreaming.value = true

  deployLogsStreamHandle = api.streamDeployTaskLogs(
    taskId,
    {},
    // onLog
    (log) => {
      deployTaskLogs.value.push(log)
      // 自动滚动到底部
      if (deployLogsAutoScroll.value && deployLogsContainerRef.value) {
        setTimeout(() => {
          deployLogsContainerRef.value.scrollTop = deployLogsContainerRef.value.scrollHeight
        }, 50)
      }
    },
    // onStatus
    (status) => {
      if (deployLogsTask.value) {
        deployLogsTask.value.status = status.task_status
        deployLogsTask.value.success_count = status.success_count
        deployLogsTask.value.failed_count = status.failed_count
        deployLogsTask.value.pending_count = status.pending_count
      }
    },
    // onDone
    (data) => {
      deployLogsStreaming.value = false
      if (deployLogsTask.value) {
        deployLogsTask.value.status = data.task_status
      }
    },
    // onError
    (error) => {
      console.error('Deploy logs stream error:', error)
      deployLogsStreaming.value = false
    }
  )
}

async function loadDeployTaskLogs(taskId) {
  try {
    const res = await api.getDeployTaskLogs(taskId)
    if (res.success) {
      deployTaskLogs.value = res.logs || []
    }
  } catch (e) {
    console.error('Failed to load deploy task logs:', e)
  }
}

function clearDeployLogs() {
  deployTaskLogs.value = []
}

function closeDeployLogs() {
  if (deployLogsStreamHandle) {
    deployLogsStreamHandle.close()
    deployLogsStreamHandle = null
  }
  deployLogsStreaming.value = false
  deployTaskLogs.value = []
}

function formatLogTime(timestamp) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', { hour12: false })
}

function getLogLevelTag(level) {
  switch (level) {
    case 'error': return 'danger'
    case 'warn': return 'warning'
    case 'info': return 'info'
    case 'debug': return ''
    default: return 'info'
  }
}

// ==================== 工具函数 ====================
function getProjectTypeTag(type) {
  const map = { script: '', container: 'success', gitpull: 'info', kubernetes: 'warning' }
  return map[type] || ''
}

function getProjectTypeLabel(type) {
  const map = { script: '脚本', container: '容器', gitpull: 'Git', kubernetes: 'K8s' }
  return map[type] || type
}

function getVersionStatusTag(status) {
  const map = { draft: 'info', active: 'success', deprecated: 'warning' }
  return map[status] || ''
}

function getVersionStatusLabel(status) {
  const map = { draft: '草稿', active: '已发布', deprecated: '已废弃' }
  return map[status] || status
}

function getOperationTag(op) {
  const map = { deploy: 'primary', install: 'success', update: 'primary', rollback: 'warning', uninstall: 'danger' }
  return map[op] || ''
}

function getOperationLabel(op) {
  const map = { deploy: '部署', install: '安装', update: '升级', rollback: '回滚', uninstall: '卸载' }
  return map[op] || op
}

function getTaskStatusTag(status) {
  const map = {
    pending: 'info',
    scheduled: 'info',
    running: '',
    canary: 'warning',
    paused: 'warning',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return map[status] || ''
}

function getTaskStatusLabel(status) {
  const map = {
    pending: '待执行',
    scheduled: '已计划',
    running: '执行中',
    canary: '金丝雀中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return map[status] || status
}

function getResultStatusTag(status) {
  const map = { pending: 'info', running: '', success: 'success', failed: 'danger', skipped: 'info' }
  return map[status] || ''
}

function getResultStatusLabel(status) {
  const map = { pending: '待执行', running: '执行中', success: '成功', failed: '失败', skipped: '跳过' }
  return map[status] || status
}

function getLogStatusTag(status) {
  const map = { success: 'success', failed: 'danger', skipped: 'info', rollback: 'warning' }
  return map[status] || ''
}

function getLogStatusLabel(status) {
  const map = { success: '成功', failed: '失败', skipped: '跳过', rollback: '已回滚' }
  return map[status] || status
}

function getTaskProgress(task) {
  if (!task.total_count) return 0
  return Math.round(((task.success_count || 0) + (task.failed_count || 0)) / task.total_count * 100)
}

function getProgressStatus(status) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'exception'
  return null
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

function formatDuration(seconds) {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

// ==================== 初始化 ====================
onMounted(() => {
  loadData()
})
</script>

<style scoped>
.release-page {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 0 4px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.header-actions {
  display: flex;
  gap: 12px;
}

.project-card {
  height: calc(100vh - 160px);
  overflow: hidden;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
}

.project-card :deep(.el-card__body) {
  padding: 0;
  height: calc(100% - 50px);
  overflow-y: auto;
}

.project-card :deep(.el-card__header) {
  background: var(--tech-bg-tertiary);
  border-bottom: 1px solid var(--tech-border);
  padding: 15px 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.project-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.project-list {
  padding: 8px;
}

.project-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  margin: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.project-item:hover {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
}

.project-item.active {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-primary);
  border-left: 3px solid var(--tech-primary);
}

.project-name {
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--tech-text-primary);
  font-size: 14px;
}

.project-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.version-count {
  font-size: 12px;
  color: var(--tech-text-muted);
}

.more-icon {
  opacity: 0;
  cursor: pointer;
}

.project-item:hover .more-icon {
  opacity: 1;
}

.empty-card {
  height: calc(100vh - 160px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.overview-card {
  height: calc(100vh - 160px);
  overflow: auto;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
}

.overview-card :deep(.el-card__header) {
  background: var(--tech-bg-tertiary);
  border-bottom: 1px solid var(--tech-border);
  padding: 15px 20px;
}

.overview-card :deep(.el-card__body) {
  padding: 20px;
}

.global-stats {
  margin-bottom: 20px;
}

.stat-card.large {
  padding: 24px;
}

.stat-card.large .stat-value {
  font-size: 36px;
}

.stat-card.large .stat-label {
  font-size: 14px;
  margin-top: 8px;
}

.recent-logs {
  margin-top: 24px;
}

.recent-logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.recent-logs-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.recent-logs-table-wrapper {
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  overflow: hidden;
}

.recent-logs-table {
  background: transparent;
}

.recent-logs-table :deep(.el-table__header-wrapper) {
  background: transparent;
}

.recent-logs-table :deep(.el-table__header) {
  background: transparent;
}

.recent-logs-table :deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  font-weight: 600;
  padding: 12px;
  font-size: 14px;
  transition: all 0.3s ease;
}

.recent-logs-table :deep(.el-table th:hover) {
  background-color: var(--tech-bg-tertiary);
  color: var(--tech-text-primary);
}

.recent-logs-table :deep(.el-table td) {
  border-color: var(--tech-border);
  padding: 12px;
  transition: all 0.2s ease;
}

.recent-logs-table :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: var(--tech-bg-tertiary);
}

.recent-logs-table :deep(.el-table__row) {
  transition: all 0.2s ease;
}

.recent-logs-table :deep(.el-table__row:hover) {
  background-color: var(--tech-bg-tertiary);
}

.recent-logs-table :deep(.el-table__row:hover td) {
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
}

.table-cell-text {
  color: var(--tech-text-primary);
  font-size: 14px;
}

.table-cell-time {
  color: var(--tech-text-secondary);
  font-size: 13px;
  font-family: var(--tech-font-mono);
}

.recent-logs-table :deep(.el-tag) {
  border-radius: 4px;
  font-weight: 500;
  border-width: 1px;
}

.mt-20 {
  margin-top: 20px;
}

.mb-8 {
  margin-bottom: 8px;
}

.info-card {
  margin-bottom: 16px;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
}

.info-card :deep(.el-card__header) {
  background: var(--tech-bg-tertiary);
  border-bottom: 1px solid var(--tech-border);
  padding: 15px 20px;
}

.project-desc {
  margin: 0;
  color: var(--tech-text-secondary);
  font-size: 14px;
}

.main-card {
  height: calc(100vh - 280px);
  overflow: hidden;
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
}

.main-card :deep(.el-card__body) {
  height: 100%;
  overflow: auto;
  padding: 20px;
}

.main-card :deep(.el-tabs__header) {
  margin: 0 0 16px 0;
  background: transparent;
}

.main-card :deep(.el-tabs__nav-wrap::after) {
  background-color: var(--tech-border);
}

.tab-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 0 4px;
}

.task-filter-group {
  flex: 1;
}

.task-count-tag {
  margin-left: 12px;
}

.tasks-table-wrapper {
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  overflow: hidden;
}

.tasks-table {
  background: transparent;
}

.tasks-table :deep(.el-table__header-wrapper) {
  background: transparent;
}

.tasks-table :deep(.el-table__header) {
  background: transparent;
}

.tasks-table :deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  font-weight: 600;
  padding: 12px;
  font-size: 14px;
  transition: all 0.3s ease;
}

.tasks-table :deep(.el-table th:hover) {
  background-color: var(--tech-bg-tertiary);
  color: var(--tech-text-primary);
}

.tasks-table :deep(.el-table td) {
  border-color: var(--tech-border);
  padding: 12px;
  transition: all 0.2s ease;
}

.tasks-table :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: var(--tech-bg-tertiary);
}

.tasks-table :deep(.el-table__row) {
  transition: all 0.2s ease;
}

.tasks-table :deep(.el-table__row:hover) {
  background-color: var(--tech-bg-tertiary);
}

.tasks-table :deep(.el-table__row:hover td) {
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
}

.tasks-table :deep(.el-tag) {
  border-radius: 4px;
  font-weight: 500;
  border-width: 1px;
}

.table-cell-text {
  color: var(--tech-text-primary);
  font-size: 14px;
}

.table-cell-empty {
  color: var(--tech-text-muted);
  font-size: 14px;
}

.task-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.task-actions .el-button {
  margin: 0;
}

.task-progress-bar {
  flex: 1;
}

.task-progress-bar :deep(.el-progress-bar__outer) {
  background-color: var(--tech-bg-tertiary);
}

.task-progress-bar :deep(.el-progress-bar__inner) {
  transition: width 0.3s ease;
}

.deploy-logs-table-wrapper {
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  overflow: hidden;
}

.deploy-logs-table {
  background: transparent;
}

.deploy-logs-table :deep(.el-table__header-wrapper) {
  background: transparent;
}

.deploy-logs-table :deep(.el-table__header) {
  background: transparent;
}

.deploy-logs-table :deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  font-weight: 600;
  padding: 12px;
  font-size: 14px;
  transition: all 0.3s ease;
}

.deploy-logs-table :deep(.el-table th:hover) {
  background-color: var(--tech-bg-tertiary);
  color: var(--tech-text-primary);
}

.deploy-logs-table :deep(.el-table td) {
  border-color: var(--tech-border);
  padding: 12px;
  transition: all 0.2s ease;
}

.deploy-logs-table :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: var(--tech-bg-tertiary);
}

.deploy-logs-table :deep(.el-table__row) {
  transition: all 0.2s ease;
}

.deploy-logs-table :deep(.el-table__row:hover) {
  background-color: var(--tech-bg-tertiary);
}

.deploy-logs-table :deep(.el-table__row:hover td) {
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
}

.deploy-logs-table :deep(.el-tag) {
  border-radius: 4px;
  font-weight: 500;
  border-width: 1px;
}

.deploy-logs-table :deep(.el-button) {
  border-radius: 4px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.deploy-logs-table :deep(.el-button:not(.el-button--primary):not(.el-button--danger)) {
  background-color: var(--tech-bg-secondary);
  border-color: var(--tech-border);
  color: var(--tech-text-secondary);
}

.deploy-logs-table :deep(.el-button:not(.el-button--primary):not(.el-button--danger):hover) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-primary);
  color: var(--tech-primary);
}

.progress-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-text {
  font-size: 12px;
  color: var(--tech-text-muted);
  white-space: nowrap;
}

.form-tip {
  font-size: 12px;
  color: var(--tech-text-muted);
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.form-tip code {
  background: var(--tech-bg-tertiary);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: var(--tech-font-mono);
  font-size: 11px;
  color: var(--tech-primary);
}

.ml-2 {
  margin-left: 8px;
}

.text-gray {
  color: #909399;
}

/* 版本对话框样式优化 */
.version-dialog :deep(.el-dialog__header) {
  padding: 20px 20px 16px;
  border-bottom: 1px solid var(--tech-border);
}

.version-dialog :deep(.el-dialog__body) {
  padding: 20px;
  max-height: calc(90vh - 120px);
  overflow-y: auto;
}

.version-form {
  padding: 0;
}

.version-form .el-form-item {
  margin-bottom: 20px;
}

.version-form .el-divider {
  margin: 24px 0 20px;
}

.version-form .el-divider__text {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.flex-items-center {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--tech-border);
}

.version-dialog :deep(.el-divider) {
  border-color: var(--tech-border);
}

.version-dialog :deep(.el-divider__text) {
  background: var(--tech-bg-card);
  padding: 0 12px;
}

.script-tabs {
  margin-top: 0;
}

.script-tabs :deep(.el-tabs__content) {
  padding: 16px;
}

.task-progress {
  margin: 20px 0;
  padding: 16px;
  background: var(--tech-bg-tertiary);
  border: 1px solid var(--tech-border);
  border-radius: 4px;
}

.task-progress h4 {
  margin: 0 0 12px 0;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.progress-stats {
  display: flex;
  gap: 20px;
  margin-top: 12px;
}

.progress-stats .stat {
  font-size: 14px;
}

.progress-stats .success {
  color: var(--tech-secondary);
}

.progress-stats .failed {
  color: var(--tech-danger);
}

.progress-stats .pending {
  color: var(--tech-text-muted);
}

.script-tabs {
  margin-top: 20px;
}

.script-content {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
}

.error-text {
  color: #f56c6c;
  font-size: 12px;
}

.stats-summary {
  margin-bottom: 20px;
}

.stat-card {
  background: var(--tech-bg-card);
  border: 1px solid var(--tech-border);
  padding: 20px;
  border-radius: 4px;
  text-align: center;
  transition: all 0.3s ease;
  cursor: pointer;
}

.stat-card:hover {
  box-shadow: var(--tech-shadow-md);
  transform: translateY(-2px);
}

.stat-card.success {
  background-color: rgba(103, 194, 58, 0.05);
  border-color: rgba(103, 194, 58, 0.2);
}

.stat-card.danger {
  background-color: rgba(245, 108, 108, 0.05);
  border-color: rgba(245, 108, 108, 0.2);
}

.stat-card.primary {
  background-color: rgba(64, 158, 255, 0.05);
  border-color: rgba(64, 158, 255, 0.2);
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--tech-text-primary);
  margin-bottom: 8px;
}

.stat-card.large .stat-value {
  font-size: 36px;
}

.stat-card.success .stat-value {
  color: var(--tech-secondary);
}

.stat-card.danger .stat-value {
  color: var(--tech-danger);
}

.stat-card.primary .stat-value {
  color: var(--tech-primary);
}

.stat-label {
  font-size: 13px;
  color: var(--tech-text-secondary);
  margin-top: 4px;
  font-weight: 500;
}

.stat-card.large .stat-label {
  font-size: 14px;
  margin-top: 8px;
}

.log-section {
  margin-top: 20px;
}

.log-section h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

.log-section.error h4 {
  color: var(--tech-danger);
}

/* 表格样式美化 */
.release-page :deep(.el-table) {
  background: transparent;
  color: var(--tech-text-primary);
}

.release-page :deep(.el-table th) {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-border);
  color: var(--tech-text-primary);
  font-weight: 600;
}

.release-page :deep(.el-table td) {
  border-color: var(--tech-border);
}

.release-page :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background-color: var(--tech-bg-tertiary);
}

.release-page :deep(.el-table__row:hover) {
  background-color: var(--tech-bg-tertiary);
}

/* 按钮样式 */
.release-page :deep(.el-button--primary) {
  background-color: var(--tech-primary);
  border-color: var(--tech-primary);
  color: #ffffff;
}

.release-page :deep(.el-button--primary:hover) {
  background-color: var(--tech-primary-light);
  border-color: var(--tech-primary-light);
}

/* Docker 配置标签样式 */
.config-tag {
  margin-left: 4px;
  margin-bottom: 4px;
}

.form-tip .config-tag:first-child {
  margin-left: 8px;
}

/* 操作类型描述 */
.op-desc {
  font-size: 12px;
  color: var(--tech-text-muted);
  margin-left: 4px;
}

.el-radio-group .el-radio {
  margin-bottom: 8px;
}

.mt-16 {
  margin-top: 16px;
}

.version-config-section {
  margin-top: 16px;
}

.version-config-section h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--tech-text-primary);
}

/* 容器日志对话框样式 */
.container-logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--tech-border);
}

.logs-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logs-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.container-logs-content {
  height: 500px;
  overflow-y: auto;
  background: #1e1e1e;
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  padding: 0;
}

.logs-pre {
  margin: 0;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}

.container-logs-footer {
  margin-top: 12px;
}

.container-logs-content::-webkit-scrollbar {
  width: 8px;
}

.container-logs-content::-webkit-scrollbar-track {
  background: #2d2d2d;
}

.container-logs-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.container-logs-content::-webkit-scrollbar-thumb:hover {
  background: #666;
}

/* 实时部署日志对话框样式 */
.deploy-logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--tech-border);
}

.progress-mini {
  font-size: 12px;
  color: var(--tech-text-muted);
  font-family: var(--tech-font-mono);
}

.deploy-logs-content {
  height: 500px;
  overflow-y: auto;
  background: #1e1e1e;
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  padding: 12px 16px;
}

.deploy-log-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 4px 0;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: #d4d4d4;
}

.deploy-log-item.error {
  color: #f56c6c;
}

.deploy-log-item.warn {
  color: #e6a23c;
}

.deploy-log-item.info {
  color: #67c23a;
}

.deploy-log-item.debug {
  color: #909399;
}

.deploy-log-item .log-time {
  color: #6a9955;
  flex-shrink: 0;
  min-width: 70px;
}

.deploy-log-item .log-level {
  flex-shrink: 0;
  font-size: 11px;
}

.deploy-log-item .log-client {
  color: #4fc1ff;
  flex-shrink: 0;
}

.deploy-log-item .log-stage {
  color: #c586c0;
  flex-shrink: 0;
}

.deploy-log-item .log-message {
  flex: 1;
  word-break: break-all;
  white-space: pre-wrap;
}

.deploy-logs-content::-webkit-scrollbar {
  width: 8px;
}

.deploy-logs-content::-webkit-scrollbar-track {
  background: #2d2d2d;
}

.deploy-logs-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.deploy-logs-content::-webkit-scrollbar-thumb:hover {
  background: #666;
}

/* 客户端选择列表样式 */
.select-client-list {
  max-height: 400px;
  overflow-y: auto;
}

.select-client-item {
  padding: 12px 16px;
  border: 1px solid var(--tech-border);
  border-radius: 4px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.select-client-item:hover {
  background-color: var(--tech-bg-tertiary);
  border-color: var(--tech-primary);
}

.select-client-item:last-child {
  margin-bottom: 0;
}

.select-client-item .client-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.select-client-item .client-id {
  font-weight: 600;
  color: var(--tech-text-primary);
}

.select-client-item .client-container {
  margin-top: 4px;
  font-size: 12px;
  color: var(--tech-text-muted);
  font-family: var(--tech-font-mono);
}

/* 任务对话框底部样式 */
.task-dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.task-dialog-footer .footer-actions {
  display: flex;
  gap: 12px;
}

/* 配置预览样式 */
.mb-16 {
  margin-bottom: 16px;
}

.mr-2 {
  margin-right: 8px;
}

.volume-item {
  padding: 4px 0;
  font-size: 13px;
  color: var(--tech-text-secondary);
  font-family: var(--tech-font-mono);
}

.volume-item + .volume-item {
  border-top: 1px dashed var(--tech-border);
}
</style>
