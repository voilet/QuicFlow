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
                  <div class="stat-value">{{ clients.length }}</div>
                  <div class="stat-label">在线客户端</div>
                </div>
              </el-col>
            </el-row>
          </div>

          <!-- 最近部署日志 -->
          <div class="recent-logs" v-if="globalRecentLogs.length > 0">
            <h4>最近部署记录</h4>
            <el-table :data="globalRecentLogs" size="small" stripe max-height="300">
              <el-table-column prop="project_name" label="项目" width="120" show-overflow-tooltip />
              <el-table-column prop="client_id" label="客户端" width="150" show-overflow-tooltip />
              <el-table-column prop="version" label="版本" width="100">
                <template #default="{ row }">
                  <el-tag size="small">{{ row.version }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="operation" label="操作" width="80">
                <template #default="{ row }">
                  <el-tag size="small" :type="getOperationTag(row.operation)">
                    {{ getOperationLabel(row.operation) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="结果" width="80">
                <template #default="{ row }">
                  <el-tag size="small" :type="getLogStatusTag(row.status)">
                    {{ getLogStatusLabel(row.status) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="started_at" label="时间" width="160">
                <template #default="{ row }">
                  {{ formatTime(row.started_at) }}
                </template>
              </el-table-column>
            </el-table>
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
                  <el-radio-group v-model="taskStatusFilter" size="small">
                    <el-radio-button value="">全部</el-radio-button>
                    <el-radio-button value="pending">待执行</el-radio-button>
                    <el-radio-button value="running">执行中</el-radio-button>
                    <el-radio-button value="canary">金丝雀中</el-radio-button>
                    <el-radio-button value="completed">已完成</el-radio-button>
                  </el-radio-group>
                </div>

                <el-table :data="filteredTasks" v-loading="loadingTasks" stripe>
                  <el-table-column prop="version" label="版本" width="100">
                    <template #default="{ row }">
                      <el-tag size="small">{{ row.version }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="operation" label="操作" width="80">
                    <template #default="{ row }">
                      <el-tag size="small" :type="getOperationTag(row.operation)">
                        {{ getOperationLabel(row.operation) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="status" label="状态" width="100">
                    <template #default="{ row }">
                      <el-tag size="small" :type="getTaskStatusTag(row.status)">
                        {{ getTaskStatusLabel(row.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="进度" width="180">
                    <template #default="{ row }">
                      <div class="progress-cell">
                        <el-progress
                          :percentage="getTaskProgress(row)"
                          :status="getProgressStatus(row.status)"
                          :stroke-width="6"
                        />
                        <span class="progress-text">{{ row.success_count || 0 }}/{{ row.total_count || 0 }}</span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column prop="schedule_type" label="执行方式" width="100">
                    <template #default="{ row }">
                      {{ row.schedule_type === 'immediate' ? '立即执行' : '定时执行' }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="canary_enabled" label="金丝雀" width="80">
                    <template #default="{ row }">
                      <el-tag v-if="row.canary_enabled" size="small" type="warning">
                        {{ row.canary_percent }}%
                      </el-tag>
                      <span v-else>-</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="200" fixed="right">
                    <template #default="{ row }">
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
                    </template>
                  </el-table-column>
                </el-table>
              </el-tab-pane>

              <!-- 部署记录 -->
              <el-tab-pane label="部署日志" name="history">
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
                <el-table :data="deployLogs" v-loading="loadingLogs" stripe>
                  <el-table-column prop="client_id" label="客户端" width="150" show-overflow-tooltip />
                  <el-table-column prop="version" label="版本" width="100">
                    <template #default="{ row }">
                      <el-tag size="small">{{ row.version }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="operation" label="操作" width="80">
                    <template #default="{ row }">
                      <el-tag size="small" :type="getOperationTag(row.operation)">
                        {{ getOperationLabel(row.operation) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="status" label="结果" width="80">
                    <template #default="{ row }">
                      <el-tag size="small" :type="getLogStatusTag(row.status)">
                        {{ getLogStatusLabel(row.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="is_canary" label="金丝雀" width="70">
                    <template #default="{ row }">
                      <el-tag v-if="row.is_canary" size="small" type="warning">是</el-tag>
                      <span v-else>-</span>
                    </template>
                  </el-table-column>
                  <el-table-column prop="duration" label="耗时" width="80">
                    <template #default="{ row }">
                      {{ formatDuration(row.duration) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="started_at" label="执行时间" width="180">
                    <template #default="{ row }">
                      {{ formatTime(row.started_at) }}
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ row }">
                      <el-button size="small" @click="viewLog(row)">详情</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </el-tab-pane>
            </el-tabs>
          </el-card>
        </template>
      </el-col>
    </el-row>

    <!-- 新建/编辑项目对话框 -->
    <el-dialog v-model="projectDialogVisible" :title="editingProject ? '编辑项目' : '新建项目'" width="600px">
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
            <el-input v-model="projectForm.gitpull_config.pre_script" type="textarea" rows="3" placeholder="#!/bin/bash&#10;# 拉取代码前执行的脚本（可选）" />
          </el-form-item>
          <el-form-item label="部署后脚本">
            <el-input v-model="projectForm.gitpull_config.post_script" type="textarea" rows="3" placeholder="#!/bin/bash&#10;# 拉取代码后执行的脚本（可选）" />
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
        </template>

        <!-- Kubernetes 配置 -->
        <template v-if="projectForm.type === 'kubernetes'">
          <el-divider content-position="left">Kubernetes 配置</el-divider>
          <el-form-item label="命名空间">
            <el-input v-model="projectForm.k8s_config.namespace" placeholder="default" />
          </el-form-item>
          <el-form-item label="部署名称">
            <el-input v-model="projectForm.k8s_config.deployment_name" placeholder="my-app" />
          </el-form-item>
          <el-form-item label="镜像地址">
            <el-input v-model="projectForm.k8s_config.image" placeholder="nginx:latest" />
          </el-form-item>
          <el-form-item label="副本数">
            <el-input-number v-model="projectForm.k8s_config.replicas" :min="1" :max="100" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="projectDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveProject" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

    <!-- 新建版本对话框 -->
    <el-dialog v-model="versionDialogVisible" title="新建版本" width="800px">
      <el-form :model="versionForm" :rules="versionRules" ref="versionFormRef" label-width="100px">
        <el-form-item label="版本号" prop="version">
          <el-input v-model="versionForm.version" placeholder="如: 1.0.0" />
        </el-form-item>
        <el-form-item label="版本说明">
          <el-input v-model="versionForm.description" type="textarea" rows="2" placeholder="版本更新说明" />
        </el-form-item>

        <!-- Git 拉取项目：从仓库选择版本 -->
        <template v-if="selectedProject?.type === 'gitpull'">
          <el-divider content-position="left">Git 版本选择</el-divider>

          <el-form-item label="选择客户端">
            <el-select v-model="gitVersionForm.client_id" placeholder="选择一个客户端来查询 Git 仓库" @change="onGitClientChange" style="width: 300px">
              <el-option v-for="c in clients" :key="c.client_id" :label="c.client_id" :value="c.client_id" />
            </el-select>
            <el-button type="primary" :loading="loadingGitVersions" @click="fetchGitVersions" :disabled="!gitVersionForm.client_id" class="ml-2">
              获取版本
            </el-button>
          </el-form-item>

          <el-form-item label="版本类型">
            <el-radio-group v-model="gitVersionForm.version_type">
              <el-radio value="tag">Tag</el-radio>
              <el-radio value="branch">分支</el-radio>
              <el-radio value="commit">Commit</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item v-if="gitVersionForm.version_type === 'tag'" label="选择 Tag">
            <el-select v-model="gitVersionForm.selected_tag" placeholder="选择 Tag" style="width: 100%" filterable>
              <el-option v-for="tag in gitVersions.tags" :key="tag.name" :label="`${tag.name} - ${tag.message || tag.commit?.substring(0, 7)}`" :value="tag.name" />
            </el-select>
            <div class="form-tip" v-if="gitVersions.tags?.length === 0">暂无 Tag，请先获取版本</div>
          </el-form-item>

          <el-form-item v-if="gitVersionForm.version_type === 'branch'" label="选择分支">
            <el-select v-model="gitVersionForm.selected_branch" placeholder="选择分支" style="width: 100%" filterable>
              <el-option v-for="branch in gitVersions.branches" :key="branch.name" :label="`${branch.name}${branch.is_default ? ' (默认)' : ''}`" :value="branch.name" />
            </el-select>
          </el-form-item>

          <el-form-item v-if="gitVersionForm.version_type === 'commit'" label="选择 Commit">
            <el-select v-model="gitVersionForm.selected_commit" placeholder="选择 Commit" style="width: 100%" filterable>
              <el-option v-for="commit in gitVersions.recent_commits" :key="commit.hash" :label="`${commit.hash} - ${commit.message}`" :value="commit.full_hash || commit.hash" />
            </el-select>
          </el-form-item>

          <el-form-item label="当前信息" v-if="gitVersions.current_branch">
            <el-tag>{{ gitVersions.current_branch }}</el-tag>
            <span class="ml-2 text-gray">{{ gitVersions.current_commit?.substring(0, 7) }}</span>
          </el-form-item>

          <el-divider content-position="left">部署后脚本（可选）</el-divider>
          <el-form-item label="部署脚本">
            <CodeEditor
              v-model="versionForm.install_script"
              language="shell"
              height="200px"
              placeholder="#!/bin/bash&#10;# Git 拉取后执行的脚本（可选）"
            />
          </el-form-item>
        </template>

        <!-- 容器项目：镜像版本 -->
        <template v-else-if="selectedProject?.type === 'container'">
          <el-divider content-position="left">容器镜像配置</el-divider>

          <el-form-item label="镜像地址" prop="container_image">
            <el-input v-model="versionForm.container_image" placeholder="nginx:1.25.0 或 registry.example.com/app:v1.0.0" />
          </el-form-item>

          <el-form-item label="环境变量">
            <el-input v-model="versionForm.container_env" type="textarea" rows="3" placeholder="KEY1=value1&#10;KEY2=value2" />
          </el-form-item>

          <el-divider content-position="left">部署脚本（可选）</el-divider>
          <el-form-item label="部署前脚本">
            <CodeEditor
              v-model="versionForm.install_script"
              language="shell"
              height="150px"
              placeholder="#!/bin/bash&#10;# 容器部署前执行的脚本（可选）"
            />
          </el-form-item>
        </template>

        <!-- Kubernetes 项目 -->
        <template v-else-if="selectedProject?.type === 'kubernetes'">
          <el-divider content-position="left">Kubernetes 部署配置</el-divider>

          <el-form-item label="镜像地址">
            <el-input v-model="versionForm.container_image" placeholder="nginx:1.25.0" />
          </el-form-item>

          <el-form-item label="副本数">
            <el-input-number v-model="versionForm.replicas" :min="1" :max="100" />
          </el-form-item>

          <el-form-item label="YAML 配置">
            <CodeEditor
              v-model="versionForm.k8s_yaml"
              language="yaml"
              height="250px"
              placeholder="# Kubernetes Deployment YAML（可选覆盖）"
            />
          </el-form-item>
        </template>

        <!-- 脚本项目：传统脚本编辑 -->
        <template v-else>
          <el-form-item label="工作目录">
            <el-input v-model="versionForm.work_dir" placeholder="/opt/app" />
          </el-form-item>

          <el-divider content-position="left">部署脚本</el-divider>

          <el-tabs v-model="scriptTab" type="border-card">
            <el-tab-pane label="安装脚本" name="install">
              <CodeEditor
                v-model="versionForm.install_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 首次安装时执行的脚本"
              />
            </el-tab-pane>
            <el-tab-pane label="升级脚本" name="update">
              <CodeEditor
                v-model="versionForm.update_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 升级时执行的脚本"
              />
            </el-tab-pane>
            <el-tab-pane label="回滚脚本" name="rollback">
              <CodeEditor
                v-model="versionForm.rollback_script"
                language="shell"
                height="250px"
                placeholder="#!/bin/bash&#10;# 回滚时执行的脚本"
              />
            </el-tab-pane>
            <el-tab-pane label="卸载脚本" name="uninstall">
              <CodeEditor
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
        <el-button @click="versionDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveVersion" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>

    <!-- 创建部署任务对话框 -->
    <el-dialog v-model="taskDialogVisible" title="创建部署任务" width="700px">
      <el-form :model="taskForm" :rules="taskRules" ref="taskFormRef" label-width="120px">
        <el-form-item label="部署版本">
          <el-tag>{{ selectedVersion?.version }}</el-tag>
        </el-form-item>

        <el-form-item label="操作类型" prop="operation">
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
        </el-form-item>

        <el-form-item label="目标客户端" prop="client_ids">
          <el-select
            v-model="taskForm.client_ids"
            multiple
            filterable
            placeholder="选择目标客户端"
            style="width: 100%"
          >
            <el-option
              v-for="client in availableClients"
              :key="client.client_id"
              :label="getClientLabel(client)"
              :value="client.client_id"
            />
          </el-select>
          <div class="form-tip">
            <el-button link type="primary" @click="selectAllClients">全选</el-button>
            <el-button link @click="taskForm.client_ids = []">清空</el-button>
            <span class="ml-2">已选 {{ taskForm.client_ids.length }} 个</span>
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
        <el-button @click="taskDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createTask" :loading="submitting">创建任务</el-button>
      </template>
    </el-dialog>

    <!-- 任务详情抽屉 -->
    <el-drawer v-model="taskDetailVisible" title="任务详情" size="60%">
      <template v-if="selectedTask">
        <el-descriptions :column="3" border>
          <el-descriptions-item label="版本">{{ selectedTask.version }}</el-descriptions-item>
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
          <el-table-column prop="client_id" label="客户端" width="200" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="getResultStatusTag(row.status)">
                {{ getResultStatusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="is_canary" label="金丝雀" width="80">
            <template #default="{ row }">
              <el-tag v-if="row.is_canary" size="small" type="warning">是</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="started_at" label="开始时间" width="160">
            <template #default="{ row }">
              {{ formatTime(row.started_at) }}
            </template>
          </el-table-column>
          <el-table-column prop="duration" label="耗时" width="80">
            <template #default="{ row }">
              {{ formatDuration(row.duration) }}
            </template>
          </el-table-column>
          <el-table-column prop="error" label="错误信息">
            <template #default="{ row }">
              <span class="error-text">{{ row.error || '-' }}</span>
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
          <el-descriptions-item label="工作目录">{{ selectedVersionDetail.work_dir || '/opt/app' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(selectedVersionDetail.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="版本说明" :span="2">
            {{ selectedVersionDetail.description || '无' }}
          </el-descriptions-item>
        </el-descriptions>

        <el-tabs class="script-tabs">
          <el-tab-pane label="安装脚本">
            <CodeEditor
              :model-value="selectedVersionDetail.install_script || '# 未配置'"
              language="shell"
              height="300px"
              :read-only="true"
            />
          </el-tab-pane>
          <el-tab-pane label="升级脚本">
            <CodeEditor
              :model-value="selectedVersionDetail.update_script || '# 未配置'"
              language="shell"
              height="300px"
              :read-only="true"
            />
          </el-tab-pane>
          <el-tab-pane label="回滚脚本">
            <CodeEditor
              :model-value="selectedVersionDetail.rollback_script || '# 未配置'"
              language="shell"
              height="300px"
              :read-only="true"
            />
          </el-tab-pane>
          <el-tab-pane label="卸载脚本">
            <CodeEditor
              :model-value="selectedVersionDetail.uninstall_script || '# 未配置'"
              language="shell"
              height="300px"
              :read-only="true"
            />
          </el-tab-pane>
        </el-tabs>
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Refresh, MoreFilled, Download, Upload, RefreshLeft, Delete
} from '@element-plus/icons-vue'
import api from '@/api'
import CodeEditor from '@/components/CodeEditor.vue'

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
const clients = ref([])

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
  client_id: '',
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
const editingProject = ref(null)

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
    restart_policy: 'unless-stopped'
  },
  // Kubernetes 配置
  k8s_config: {
    namespace: 'default',
    deployment_name: '',
    image: '',
    replicas: 1
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
  // 容器/K8s 相关
  container_image: '',
  container_env: '',
  replicas: 1,
  k8s_yaml: '',
  // Git 相关
  git_ref: '',       // tag/branch/commit 值
  git_ref_type: ''   // tag/branch/commit 类型
})

const versionRules = {
  version: [{ required: true, message: '请输入版本号', trigger: 'blur' }]
  // install_script 不再是必须的，因为 Git/容器/K8s 项目不需要
}

const taskForm = reactive({
  operation: 'install',
  client_ids: [],
  schedule_type: 'immediate',
  schedule_time: null,
  canary_enabled: true,
  canary_percent: 10,
  canary_duration: 30,
  canary_auto_promote: false,
  failure_strategy: 'continue',
  auto_rollback: true
})

const taskRules = {
  operation: [{ required: true, message: '请选择操作类型', trigger: 'change' }],
  client_ids: [{ required: true, type: 'array', min: 1, message: '请选择目标客户端', trigger: 'change' }]
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

const availableClients = computed(() => {
  // 根据操作类型筛选客户端
  // install: 未安装的客户端
  // update/rollback/uninstall: 已安装的客户端
  return clients.value
})

// ==================== 数据加载 ====================
async function loadData() {
  loading.value = true
  try {
    await Promise.all([loadProjects(), loadClients(), loadGlobalStats()])
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

async function loadClients() {
  try {
    const res = await api.getClients()
    clients.value = res.clients || []
  } catch (e) {
    clients.value = []
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
    restart_policy: 'unless-stopped'
  }
  // 重置 K8s 配置
  projectForm.k8s_config = {
    namespace: 'default',
    deployment_name: '',
    image: '',
    replicas: 1
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
  // Git 字段
  versionForm.git_ref = ''
  versionForm.git_ref_type = ''
  // 重置 Git 版本表单
  gitVersionForm.client_id = ''
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

// 获取 Git 仓库版本信息
async function fetchGitVersions() {
  if (!gitVersionForm.client_id || !selectedProject.value) return

  loadingGitVersions.value = true
  try {
    const res = await api.getGitVersions({
      client_id: gitVersionForm.client_id,
      project_id: selectedProject.value.id,
      repo_url: selectedProject.value.gitpull_config?.repo_url || '',
      work_dir: selectedProject.value.gitpull_config?.work_dir || '',
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
    } else {
      ElMessage.error(res.error || '获取 Git 版本失败')
    }
  } catch (e) {
    ElMessage.error(e.message || '获取 Git 版本失败')
  } finally {
    loadingGitVersions.value = false
  }
}

// Git 客户端变更时清空已选版本
function onGitClientChange() {
  gitVersionForm.selected_tag = ''
  gitVersionForm.selected_branch = ''
  gitVersionForm.selected_commit = ''
  gitVersions.tags = []
  gitVersions.branches = []
  gitVersions.recent_commits = []
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
    const res = await api.createVersion(selectedProject.value.id, versionForm)
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
  taskForm.operation = 'install'
  taskForm.client_ids = []
  taskForm.schedule_type = 'immediate'
  taskForm.schedule_time = null
  taskForm.canary_enabled = true
  taskForm.canary_percent = 10
  taskForm.canary_duration = 30
  taskForm.canary_auto_promote = false
  taskForm.failure_strategy = 'continue'
  taskForm.auto_rollback = true
  taskDialogVisible.value = true
}

function selectAllClients() {
  taskForm.client_ids = clients.value.map(c => c.client_id)
}

function getClientLabel(client) {
  // 显示客户端ID和当前安装版本
  const version = client.installed_version
  return version ? `${client.client_id} (当前: ${version})` : client.client_id
}

async function createTask() {
  const valid = await taskFormRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const taskData = {
      project_id: selectedProject.value.id,
      version_id: selectedVersion.value.id,
      ...taskForm
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

// ==================== 工具函数 ====================
function getProjectTypeTag(type) {
  const map = { script: '', container: 'success', gitpull: 'info' }
  return map[type] || ''
}

function getProjectTypeLabel(type) {
  const map = { script: '脚本', container: '容器', gitpull: 'Git' }
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
  const map = { install: 'success', update: 'primary', rollback: 'warning', uninstall: 'danger' }
  return map[op] || ''
}

function getOperationLabel(op) {
  const map = { install: '安装', update: '升级', rollback: '回滚', uninstall: '卸载' }
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
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}

.project-card {
  height: calc(100vh - 160px);
  overflow: hidden;
}

.project-card :deep(.el-card__body) {
  padding: 0;
  height: calc(100% - 50px);
  overflow-y: auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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
  padding: 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.project-item:hover {
  background: #f5f7fa;
}

.project-item.active {
  background: #ecf5ff;
}

.project-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.project-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.version-count {
  font-size: 12px;
  color: #909399;
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

.recent-logs h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #303133;
}

.mt-20 {
  margin-top: 20px;
}

.info-card {
  margin-bottom: 16px;
}

.project-desc {
  margin: 0;
  color: #909399;
}

.main-card {
  height: calc(100vh - 280px);
  overflow: hidden;
}

.main-card :deep(.el-card__body) {
  height: 100%;
  overflow: auto;
}

.tab-header {
  margin-bottom: 16px;
}

.progress-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-text {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

.form-tip {
  font-size: 12px;
  color: #909399;
}

.ml-2 {
  margin-left: 8px;
}

.text-gray {
  color: #909399;
}

.task-progress {
  margin: 20px 0;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
}

.task-progress h4 {
  margin: 0 0 12px 0;
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
  color: #67c23a;
}

.progress-stats .failed {
  color: #f56c6c;
}

.progress-stats .pending {
  color: #909399;
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
  background: #f5f7fa;
  padding: 16px;
  border-radius: 8px;
  text-align: center;
}

.stat-card.success {
  background: #f0f9eb;
}

.stat-card.danger {
  background: #fef0f0;
}

.stat-card.primary {
  background: #ecf5ff;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}

.stat-card.success .stat-value {
  color: #67c23a;
}

.stat-card.danger .stat-value {
  color: #f56c6c;
}

.stat-card.primary .stat-value {
  color: #409eff;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

.log-section {
  margin-top: 20px;
}

.log-section h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #303133;
}

.log-section.error h4 {
  color: #f56c6c;
}
</style>
