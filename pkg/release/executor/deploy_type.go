package executor

import (
	"github.com/voilet/quic-flow/pkg/release/models"
)

// DeployTypeHandler 部署类型处理器
// 定义不同部署类型的操作映射和特性
type DeployTypeHandler struct {
	DeployType models.DeployType
}

// OperationMapping 操作映射
type OperationMapping struct {
	// 操作名称
	Operation models.OperationType
	// 命令模板（用于生成 Shell 命令）
	CommandTemplate string
	// 默认超时（秒）
	DefaultTimeout int
	// 是否需要前置检查
	RequirePreCheck bool
	// 是否支持回滚
	CanRollback bool
	// 描述
	Description string
}

// DeployTypeConfig 部署类型配置
type DeployTypeConfig struct {
	// 部署类型
	Type models.DeployType
	// 版本来源
	VersionSource string // image_tag, git_ref, version_number
	// 核心工具
	CoreTool string // docker, kubectl, git
	// 支持的操作
	Operations []OperationMapping
	// 是否支持健康检查
	SupportsHealthCheck bool
	// 是否支持原子回滚
	SupportsAtomicRollback bool
	// 状态检查命令模板
	StatusCheckTemplate string
}

// GetDeployTypeConfig 获取部署类型配置
func GetDeployTypeConfig(deployType models.DeployType) *DeployTypeConfig {
	switch deployType {
	case models.DeployTypeContainer:
		return getContainerConfig()
	case models.DeployTypeKubernetes:
		return getKubernetesConfig()
	case models.DeployTypeGitPull:
		return getGitPullConfig()
	case models.DeployTypeScript:
		return getScriptConfig()
	default:
		return nil
	}
}

// getContainerConfig Docker 容器部署配置
func getContainerConfig() *DeployTypeConfig {
	return &DeployTypeConfig{
		Type:          models.DeployTypeContainer,
		VersionSource: "image_tag",
		CoreTool:      "docker",
		Operations: []OperationMapping{
			{
				Operation:       models.OperationTypeInstall,
				CommandTemplate: "docker pull ${IMAGE} && docker run -d --name ${CONTAINER_NAME} ${OPTIONS} ${IMAGE}",
				DefaultTimeout:  600,
				RequirePreCheck: true,
				CanRollback:     true,
				Description:     "拉取镜像并创建容器",
			},
			{
				Operation:       models.OperationTypeUpdate,
				CommandTemplate: "docker pull ${IMAGE} && docker stop ${CONTAINER_NAME} && docker rm ${CONTAINER_NAME} && docker run -d --name ${CONTAINER_NAME} ${OPTIONS} ${IMAGE}",
				DefaultTimeout:  600,
				RequirePreCheck: true,
				CanRollback:     true,
				Description:     "更新镜像并重建容器",
			},
			{
				Operation:       models.OperationTypeRollback,
				CommandTemplate: "docker stop ${CONTAINER_NAME} && docker rm ${CONTAINER_NAME} && docker run -d --name ${CONTAINER_NAME} ${OPTIONS} ${OLD_IMAGE}",
				DefaultTimeout:  300,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "回滚到指定版本",
			},
			{
				Operation:       models.OperationTypeUninstall,
				CommandTemplate: "docker stop ${CONTAINER_NAME} && docker rm ${CONTAINER_NAME}",
				DefaultTimeout:  120,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "停止并删除容器",
			},
		},
		SupportsHealthCheck:    true,
		SupportsAtomicRollback: true,
		StatusCheckTemplate:    "docker inspect ${CONTAINER_NAME} --format '{{.State.Status}}'",
	}
}

// getKubernetesConfig K8s 部署配置
func getKubernetesConfig() *DeployTypeConfig {
	return &DeployTypeConfig{
		Type:          models.DeployTypeKubernetes,
		VersionSource: "image_tag",
		CoreTool:      "kubectl",
		Operations: []OperationMapping{
			{
				Operation:       models.OperationTypeInstall,
				CommandTemplate: "kubectl apply -f ${YAML_FILE} -n ${NAMESPACE}",
				DefaultTimeout:  600,
				RequirePreCheck: true,
				CanRollback:     true,
				Description:     "应用 K8s 资源",
			},
			{
				Operation:       models.OperationTypeUpdate,
				CommandTemplate: "kubectl set image deployment/${DEPLOYMENT} ${CONTAINER}=${IMAGE} -n ${NAMESPACE} && kubectl rollout status deployment/${DEPLOYMENT} -n ${NAMESPACE}",
				DefaultTimeout:  600,
				RequirePreCheck: true,
				CanRollback:     true,
				Description:     "更新镜像并等待滚动更新",
			},
			{
				Operation:       models.OperationTypeRollback,
				CommandTemplate: "kubectl rollout undo deployment/${DEPLOYMENT} -n ${NAMESPACE} --to-revision=${REVISION}",
				DefaultTimeout:  300,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "回滚到指定版本",
			},
			{
				Operation:       models.OperationTypeUninstall,
				CommandTemplate: "kubectl delete -f ${YAML_FILE} -n ${NAMESPACE} --ignore-not-found=true",
				DefaultTimeout:  120,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "删除 K8s 资源",
			},
		},
		SupportsHealthCheck:    true,
		SupportsAtomicRollback: true, // K8s 内置回滚
		StatusCheckTemplate:    "kubectl get deployment/${DEPLOYMENT} -n ${NAMESPACE} -o jsonpath='{.status.readyReplicas}'",
	}
}

// getGitPullConfig Git 拉取部署配置
func getGitPullConfig() *DeployTypeConfig {
	return &DeployTypeConfig{
		Type:          models.DeployTypeGitPull,
		VersionSource: "git_ref",
		CoreTool:      "git",
		Operations: []OperationMapping{
			{
				Operation:       models.OperationTypeInstall,
				CommandTemplate: "git clone ${REPO_URL} ${WORK_DIR} && cd ${WORK_DIR} && git checkout ${REF} && ${POST_SCRIPT}",
				DefaultTimeout:  600,
				RequirePreCheck: true,
				CanRollback:     true,
				Description:     "克隆仓库并执行部署脚本",
			},
			{
				Operation:       models.OperationTypeUpdate,
				CommandTemplate: "cd ${WORK_DIR} && git fetch --all && git checkout ${REF} && git pull origin ${BRANCH} && ${POST_SCRIPT}",
				DefaultTimeout:  300,
				RequirePreCheck: false,
				CanRollback:     true,
				Description:     "拉取最新代码并执行部署脚本",
			},
			{
				Operation:       models.OperationTypeRollback,
				CommandTemplate: "cd ${WORK_DIR} && git checkout ${OLD_REF} && ${POST_SCRIPT}",
				DefaultTimeout:  180,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "切换到指定版本",
			},
			{
				Operation:       models.OperationTypeUninstall,
				CommandTemplate: "cd ${WORK_DIR} && ${UNINSTALL_SCRIPT} && rm -rf ${WORK_DIR}",
				DefaultTimeout:  120,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "执行卸载脚本并清理目录",
			},
		},
		SupportsHealthCheck:    false,
		SupportsAtomicRollback: false, // Git 回滚需要手动操作
		StatusCheckTemplate:    "cd ${WORK_DIR} && git rev-parse HEAD",
	}
}

// getScriptConfig 脚本部署配置
func getScriptConfig() *DeployTypeConfig {
	return &DeployTypeConfig{
		Type:          models.DeployTypeScript,
		VersionSource: "version_number",
		CoreTool:      "bash",
		Operations: []OperationMapping{
			{
				Operation:       models.OperationTypeInstall,
				CommandTemplate: "${INSTALL_SCRIPT}",
				DefaultTimeout:  600,
				RequirePreCheck: true,
				CanRollback:     true,
				Description:     "执行安装脚本",
			},
			{
				Operation:       models.OperationTypeUpdate,
				CommandTemplate: "${UPDATE_SCRIPT}",
				DefaultTimeout:  300,
				RequirePreCheck: false,
				CanRollback:     true,
				Description:     "执行更新脚本",
			},
			{
				Operation:       models.OperationTypeRollback,
				CommandTemplate: "${ROLLBACK_SCRIPT}",
				DefaultTimeout:  180,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "执行回滚脚本",
			},
			{
				Operation:       models.OperationTypeUninstall,
				CommandTemplate: "${UNINSTALL_SCRIPT}",
				DefaultTimeout:  120,
				RequirePreCheck: false,
				CanRollback:     false,
				Description:     "执行卸载脚本",
			},
		},
		SupportsHealthCheck:    false,
		SupportsAtomicRollback: false, // 脚本回滚需要自定义
		StatusCheckTemplate:    "test -f ${WORK_DIR}/version.txt && cat ${WORK_DIR}/version.txt",
	}
}

// DetermineActualOperation 根据当前状态确定实际操作类型
func DetermineActualOperation(deployType models.DeployType, currentStatus models.InstallStatus, requestedOp models.OperationType) models.OperationType {
	// 如果不是 deploy 操作，直接返回
	if requestedOp != models.OperationTypeDeploy {
		return requestedOp
	}

	// deploy 类型需要根据当前状态自动判断
	switch currentStatus {
	case models.InstallStatusInstalled:
		return models.OperationTypeUpdate
	case models.InstallStatusUninstalled, models.InstallStatusUnknown:
		return models.OperationTypeInstall
	case models.InstallStatusFailed:
		// 失败状态下，尝试重新安装
		return models.OperationTypeInstall
	default:
		return models.OperationTypeInstall
	}
}

// GetOperationTimeout 获取操作超时时间
func GetOperationTimeout(deployType models.DeployType, operation models.OperationType) int {
	config := GetDeployTypeConfig(deployType)
	if config == nil {
		return 300 // 默认 5 分钟
	}

	for _, op := range config.Operations {
		if op.Operation == operation {
			return op.DefaultTimeout
		}
	}

	return 300
}

// CanRollback 检查操作是否支持回滚
func CanRollback(deployType models.DeployType, operation models.OperationType) bool {
	config := GetDeployTypeConfig(deployType)
	if config == nil {
		return false
	}

	for _, op := range config.Operations {
		if op.Operation == operation {
			return op.CanRollback
		}
	}

	return false
}

// GetVersionSource 获取版本来源类型
func GetVersionSource(deployType models.DeployType) string {
	config := GetDeployTypeConfig(deployType)
	if config == nil {
		return "version_number"
	}
	return config.VersionSource
}

// SupportsHealthCheck 检查是否支持健康检查
func SupportsHealthCheck(deployType models.DeployType) bool {
	config := GetDeployTypeConfig(deployType)
	if config == nil {
		return false
	}
	return config.SupportsHealthCheck
}
