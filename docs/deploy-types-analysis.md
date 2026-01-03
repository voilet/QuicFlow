# 三种部署类型特性分析

## 概述

系统支持四种部署类型，每种类型有不同的特性、Shell命令和操作映射：

| 类型 | 部署载体 | 核心工具 | 版本标识 |
|-----|---------|---------|---------|
| Script | 脚本 | bash/sh | 版本号 |
| Container | Docker容器 | docker | 镜像tag |
| Kubernetes | K8s资源 | kubectl | 镜像tag+YAML |
| GitPull | Git仓库 | git | tag/branch/commit |

---

## 1. Docker 容器部署 (container)

### 1.1 核心特性
- 以 Docker 镜像为部署单元
- 支持完整的 Docker 配置（端口、卷、网络、资源限制等）
- 原子性部署，便于回滚

### 1.2 操作类型映射

| 操作 | Docker 命令序列 | 说明 |
|-----|----------------|------|
| install | docker pull → docker run | 首次部署，创建新容器 |
| update | docker pull → docker stop → docker rm → docker run | 更新部署，替换容器 |
| rollback | docker stop → docker rm → docker run (旧镜像) | 回滚到指定版本 |
| uninstall | docker stop → docker rm | 完全卸载 |

### 1.3 Shell 命令生成

```bash
# 安装/更新
docker pull ${IMAGE}
docker stop ${CONTAINER_NAME} 2>/dev/null || true
docker rm ${CONTAINER_NAME} 2>/dev/null || true
docker run -d \
  --name ${CONTAINER_NAME} \
  --restart unless-stopped \
  -p 8080:80 \
  -v /data:/app/data \
  --memory 512m \
  --cpus 0.5 \
  ${IMAGE}

# 卸载
docker stop ${CONTAINER_NAME}
docker rm ${CONTAINER_NAME}
```

### 1.4 版本管理
- 版本 = 镜像 tag（如 v1.0.0、latest）
- 可通过 Registry API 或 docker images 获取版本列表
- 回滚需要保留旧镜像或重新拉取

---

## 2. Kubernetes 部署 (kubernetes)

### 2.1 核心特性
- 以 K8s 资源（Deployment、Service等）为部署单元
- 支持声明式 YAML 配置
- 内置滚动更新和回滚机制

### 2.2 操作类型映射

| 操作 | kubectl 命令 | 说明 |
|-----|-------------|------|
| install | kubectl apply -f | 创建资源 |
| update | kubectl apply -f / kubectl set image | 更新资源 |
| rollback | kubectl rollout undo | 回滚到上一版本 |
| uninstall | kubectl delete -f | 删除资源 |

### 2.3 Shell 命令生成

```bash
# 安装/更新 (YAML 方式)
kubectl apply -f deployment.yaml -n ${NAMESPACE}

# 更新 (仅镜像)
kubectl set image deployment/${DEPLOYMENT_NAME} \
  ${CONTAINER_NAME}=${NEW_IMAGE} -n ${NAMESPACE}

# 等待部署完成
kubectl rollout status deployment/${DEPLOYMENT_NAME} -n ${NAMESPACE} --timeout=300s

# 回滚到上一版本
kubectl rollout undo deployment/${DEPLOYMENT_NAME} -n ${NAMESPACE}

# 回滚到指定版本
kubectl rollout undo deployment/${DEPLOYMENT_NAME} --to-revision=${REVISION} -n ${NAMESPACE}

# 查看回滚历史
kubectl rollout history deployment/${DEPLOYMENT_NAME} -n ${NAMESPACE}

# 卸载
kubectl delete -f deployment.yaml -n ${NAMESPACE}
```

### 2.4 版本管理
- 版本 = 镜像 tag + Revision 号
- K8s 自动保留部署历史（默认10个）
- 可通过 `kubectl rollout history` 查看版本

### 2.5 特殊处理
- 需要处理 kubeconfig 认证
- 支持多种更新策略（RollingUpdate、Recreate）
- 需要等待 Pod Ready

---

## 3. Git 拉取部署 (gitpull)

### 3.1 核心特性
- 以 Git 仓库代码为部署单元
- 支持 tag/branch/commit 指定版本
- 支持 pre/post 部署脚本

### 3.2 操作类型映射

| 操作 | Git 命令序列 | 说明 |
|-----|-------------|------|
| install | git clone → post_script | 首次克隆 |
| update | git pull/checkout → post_script | 更新代码 |
| rollback | git checkout ${OLD_VERSION} → post_script | 回滚版本 |
| uninstall | pre_script → rm -rf | 清理目录 |

### 3.3 Shell 命令生成

```bash
# 安装（首次克隆）
cd ${WORK_DIR}
git clone ${REPO_URL} . --depth ${DEPTH}
git checkout ${TAG_OR_BRANCH}
${PRE_SCRIPT}
${POST_SCRIPT}

# 更新
cd ${WORK_DIR}
git fetch --all
git checkout ${TAG_OR_BRANCH}
git pull origin ${BRANCH}  # 如果是分支
${PRE_SCRIPT}
${POST_SCRIPT}

# 回滚
cd ${WORK_DIR}
git checkout ${OLD_TAG_OR_COMMIT}
${POST_SCRIPT}

# 卸载
cd ${WORK_DIR}
${UNINSTALL_SCRIPT}
rm -rf ${WORK_DIR}
```

### 3.4 版本管理
- 版本 = Git tag/branch/commit SHA
- 可通过 `git tag` / `git branch` 获取版本列表
- 支持语义化版本排序

### 3.5 特殊处理
- 需要处理 Git 认证（SSH/Token/Basic）
- 支持备份旧代码
- 支持子模块处理

---

## 4. 部署任务处理器设计

### 4.1 统一接口

```go
type DeployExecutor interface {
    // 准备部署（检查环境、拉取资源等）
    Prepare(ctx context.Context, req *DeployRequest) error

    // 执行安装
    Install(ctx context.Context, req *DeployRequest) (*DeployResult, error)

    // 执行更新
    Update(ctx context.Context, req *DeployRequest) (*DeployResult, error)

    // 执行回滚
    Rollback(ctx context.Context, req *DeployRequest) (*DeployResult, error)

    // 执行卸载
    Uninstall(ctx context.Context, req *DeployRequest) (*DeployResult, error)

    // 检查状态
    CheckStatus(ctx context.Context, req *StatusRequest) (*StatusResult, error)

    // 获取版本列表
    ListVersions(ctx context.Context, req *VersionsRequest) (*VersionsResult, error)
}
```

### 4.2 操作类型自动判断

```go
func DetermineOperation(currentStatus InstallStatus, requestedOp OperationType) OperationType {
    if requestedOp != OperationTypeDeploy {
        return requestedOp  // 明确指定的操作
    }

    // deploy 类型自动判断
    switch currentStatus {
    case InstallStatusInstalled:
        return OperationTypeUpdate
    case InstallStatusUninstalled, InstallStatusUnknown:
        return OperationTypeInstall
    default:
        return OperationTypeInstall
    }
}
```

### 4.3 失败处理策略

| 策略 | 描述 | 适用场景 |
|-----|------|---------|
| continue | 继续执行其他目标 | 非关键服务 |
| pause | 暂停等待人工干预 | 需要确认时 |
| abort | 中止整个任务 | 关键服务 |
| rollback | 自动回滚已完成的目标 | 保证一致性 |

---

## 5. 版本对比

| 维度 | Docker | Kubernetes | GitPull |
|-----|--------|------------|---------|
| 部署速度 | 快（秒级） | 中（分钟级） | 中（取决于代码量） |
| 回滚速度 | 快 | 快（内置） | 快 |
| 资源占用 | 低 | 高 | 低 |
| 隔离性 | 高 | 高 | 低 |
| 复杂度 | 中 | 高 | 低 |
| 适用场景 | 微服务 | 大规模部署 | 传统应用 |

---

## 6. 最佳实践

### 6.1 Docker 部署
1. 使用明确的镜像 tag，避免 latest
2. 配置健康检查
3. 设置资源限制
4. 使用 volume 持久化数据

### 6.2 Kubernetes 部署
1. 使用 Deployment 而非 Pod
2. 配置 readinessProbe 和 livenessProbe
3. 设置 Resource Requests/Limits
4. 使用 ConfigMap/Secret 管理配置

### 6.3 GitPull 部署
1. 使用 tag 而非 branch 部署生产
2. 配置部署前备份
3. 使用 post_script 重启服务
4. 考虑使用 shallow clone 加速
