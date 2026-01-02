package executor

import (
	"fmt"
	"strings"

	"github.com/voilet/quic-flow/pkg/release/models"
)

// K8sCommandBuilder Kubernetes 命令构建器
type K8sCommandBuilder struct {
	config    *models.KubernetesDeployConfig
	namespace string
}

// NewK8sCommandBuilder 创建 K8s 命令构建器
func NewK8sCommandBuilder(config *models.KubernetesDeployConfig) *K8sCommandBuilder {
	namespace := config.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return &K8sCommandBuilder{
		config:    config,
		namespace: namespace,
	}
}

// BuildApplyCommand 构建 kubectl apply 命令
func (b *K8sCommandBuilder) BuildApplyCommand(yamlFile string) string {
	args := []string{"kubectl", "apply", "-f", yamlFile}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildApplyFromStdinCommand 构建从 stdin 应用 YAML 的命令
func (b *K8sCommandBuilder) BuildApplyFromStdinCommand() string {
	args := []string{"kubectl", "apply", "-f", "-"}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildSetImageCommand 构建更新镜像命令
func (b *K8sCommandBuilder) BuildSetImageCommand(newImage string) string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	containerName := b.config.ContainerName
	if containerName == "" {
		containerName = b.config.ResourceName
	}

	args := []string{
		"kubectl", "set", "image",
		fmt.Sprintf("%s/%s", resourceType, b.config.ResourceName),
		fmt.Sprintf("%s=%s", containerName, newImage),
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildScaleCommand 构建扩缩容命令
func (b *K8sCommandBuilder) BuildScaleCommand(replicas int) string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	args := []string{
		"kubectl", "scale",
		fmt.Sprintf("%s/%s", resourceType, b.config.ResourceName),
		fmt.Sprintf("--replicas=%d", replicas),
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildRolloutStatusCommand 构建等待滚动更新完成命令
func (b *K8sCommandBuilder) BuildRolloutStatusCommand() string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	timeout := b.config.RolloutTimeout
	if timeout <= 0 {
		timeout = 300
	}

	args := []string{
		"kubectl", "rollout", "status",
		fmt.Sprintf("%s/%s", resourceType, b.config.ResourceName),
		fmt.Sprintf("--timeout=%ds", timeout),
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildRolloutUndoCommand 构建回滚命令
func (b *K8sCommandBuilder) BuildRolloutUndoCommand(toRevision int) string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	args := []string{
		"kubectl", "rollout", "undo",
		fmt.Sprintf("%s/%s", resourceType, b.config.ResourceName),
	}

	if toRevision > 0 {
		args = append(args, fmt.Sprintf("--to-revision=%d", toRevision))
	}

	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildRolloutHistoryCommand 构建查看滚动更新历史命令
func (b *K8sCommandBuilder) BuildRolloutHistoryCommand() string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	args := []string{
		"kubectl", "rollout", "history",
		fmt.Sprintf("%s/%s", resourceType, b.config.ResourceName),
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildDeleteCommand 构建删除命令
func (b *K8sCommandBuilder) BuildDeleteCommand(yamlFile string) string {
	args := []string{"kubectl", "delete", "-f", yamlFile}
	args = append(args, b.buildCommonArgs()...)
	args = append(args, "--ignore-not-found=true")
	return strings.Join(args, " ")
}

// BuildDeleteResourceCommand 构建删除特定资源命令
func (b *K8sCommandBuilder) BuildDeleteResourceCommand() string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	args := []string{
		"kubectl", "delete",
		resourceType,
		b.config.ResourceName,
	}
	args = append(args, b.buildCommonArgs()...)
	args = append(args, "--ignore-not-found=true")
	return strings.Join(args, " ")
}

// BuildGetCommand 构建获取资源状态命令
func (b *K8sCommandBuilder) BuildGetCommand(outputFormat string) string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	if outputFormat == "" {
		outputFormat = "wide"
	}

	args := []string{
		"kubectl", "get",
		resourceType,
		b.config.ResourceName,
		"-o", outputFormat,
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildGetPodsCommand 构建获取 Pod 列表命令
func (b *K8sCommandBuilder) BuildGetPodsCommand() string {
	args := []string{
		"kubectl", "get", "pods",
		"-l", fmt.Sprintf("app=%s", b.config.ResourceName),
		"-o", "wide",
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildDescribeCommand 构建描述资源命令
func (b *K8sCommandBuilder) BuildDescribeCommand() string {
	resourceType := b.config.ResourceType
	if resourceType == "" {
		resourceType = "deployment"
	}

	args := []string{
		"kubectl", "describe",
		resourceType,
		b.config.ResourceName,
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildLogsCommand 构建获取日志命令
func (b *K8sCommandBuilder) BuildLogsCommand(podName string, tailLines int) string {
	args := []string{"kubectl", "logs", podName}

	if b.config.ContainerName != "" {
		args = append(args, "-c", b.config.ContainerName)
	}

	if tailLines > 0 {
		args = append(args, fmt.Sprintf("--tail=%d", tailLines))
	}

	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// BuildCreateSecretCommand 构建创建 imagePullSecret 命令
func (b *K8sCommandBuilder) BuildCreateSecretCommand(secretName string) string {
	if b.config.Registry == "" || b.config.RegistryUser == "" || b.config.RegistryPass == "" {
		return ""
	}

	args := []string{
		"kubectl", "create", "secret", "docker-registry", secretName,
		"--docker-server=" + b.config.Registry,
		"--docker-username=" + b.config.RegistryUser,
		"--docker-password=" + b.config.RegistryPass,
		"--dry-run=client", "-o", "yaml", "|", "kubectl", "apply", "-f", "-",
	}
	args = append(args, b.buildCommonArgs()...)
	return strings.Join(args, " ")
}

// buildCommonArgs 构建通用参数
func (b *K8sCommandBuilder) buildCommonArgs() []string {
	var args []string

	args = append(args, "-n", b.namespace)

	if b.config.KubeConfig != "" {
		args = append(args, "--kubeconfig", b.config.KubeConfig)
	}

	if b.config.KubeContext != "" {
		args = append(args, "--context", b.config.KubeContext)
	}

	return args
}

// BuildDeployScript 构建完整的部署脚本
func (b *K8sCommandBuilder) BuildDeployScript(yamlContent string) (string, error) {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	// 创建 imagePullSecret（如果需要）
	if b.config.ImagePullSecret != "" && b.config.Registry != "" {
		script.WriteString("# Create/Update image pull secret\n")
		script.WriteString(b.BuildCreateSecretCommand(b.config.ImagePullSecret) + "\n\n")
	}

	// 应用 YAML
	if yamlContent != "" {
		script.WriteString("# Apply Kubernetes resources\n")
		script.WriteString(fmt.Sprintf("cat <<'EOF' | %s\n%s\nEOF\n\n",
			b.BuildApplyFromStdinCommand(), yamlContent))
	}

	// 等待部署完成
	script.WriteString("# Wait for rollout to complete\n")
	script.WriteString(b.BuildRolloutStatusCommand() + "\n\n")

	// 显示部署状态
	script.WriteString("# Show deployment status\n")
	script.WriteString(b.BuildGetCommand("wide") + "\n")
	script.WriteString(b.BuildGetPodsCommand() + "\n\n")

	script.WriteString("echo 'Kubernetes deployment completed successfully'\n")

	return script.String(), nil
}

// BuildUpdateScript 构建更新脚本（仅更新镜像）
func (b *K8sCommandBuilder) BuildUpdateScript(newImage string) (string, error) {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	// 更新镜像
	script.WriteString("# Update image\n")
	script.WriteString(b.BuildSetImageCommand(newImage) + "\n\n")

	// 等待部署完成
	script.WriteString("# Wait for rollout to complete\n")
	script.WriteString(b.BuildRolloutStatusCommand() + "\n\n")

	// 显示部署状态
	script.WriteString("# Show deployment status\n")
	script.WriteString(b.BuildGetCommand("wide") + "\n")
	script.WriteString(b.BuildGetPodsCommand() + "\n\n")

	script.WriteString("echo 'Kubernetes update completed successfully'\n")

	return script.String(), nil
}

// BuildRollbackScript 构建回滚脚本
func (b *K8sCommandBuilder) BuildRollbackScript(toRevision int) (string, error) {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	// 显示当前历史
	script.WriteString("# Show rollout history\n")
	script.WriteString(b.BuildRolloutHistoryCommand() + "\n\n")

	// 执行回滚
	script.WriteString("# Rollback\n")
	script.WriteString(b.BuildRolloutUndoCommand(toRevision) + "\n\n")

	// 等待回滚完成
	script.WriteString("# Wait for rollout to complete\n")
	script.WriteString(b.BuildRolloutStatusCommand() + "\n\n")

	// 显示部署状态
	script.WriteString("# Show deployment status\n")
	script.WriteString(b.BuildGetCommand("wide") + "\n")
	script.WriteString(b.BuildGetPodsCommand() + "\n\n")

	script.WriteString("echo 'Kubernetes rollback completed successfully'\n")

	return script.String(), nil
}

// BuildUninstallScript 构建卸载脚本
func (b *K8sCommandBuilder) BuildUninstallScript(yamlContent string) (string, error) {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	if yamlContent != "" {
		// 使用 YAML 删除
		script.WriteString("# Delete Kubernetes resources\n")
		script.WriteString(fmt.Sprintf("cat <<'EOF' | kubectl delete -f - -n %s --ignore-not-found=true\n%s\nEOF\n\n",
			b.namespace, yamlContent))
	} else {
		// 删除特定资源
		script.WriteString("# Delete Kubernetes resource\n")
		script.WriteString(b.BuildDeleteResourceCommand() + "\n\n")
	}

	// 删除 imagePullSecret（如果是我们创建的）
	if b.config.ImagePullSecret != "" {
		script.WriteString("# Delete image pull secret\n")
		script.WriteString(fmt.Sprintf("kubectl delete secret %s -n %s --ignore-not-found=true\n\n",
			b.config.ImagePullSecret, b.namespace))
	}

	script.WriteString("echo 'Kubernetes resources deleted successfully'\n")

	return script.String(), nil
}

// BuildCheckStatusScript 构建检查状态脚本
func (b *K8sCommandBuilder) BuildCheckStatusScript() string {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n\n")

	// 获取 Deployment 状态
	script.WriteString("# Check deployment status\n")
	script.WriteString(b.BuildGetCommand("json") + " 2>/dev/null\n")

	return script.String()
}

// GenerateDeploymentYAML 生成 Deployment YAML
func (b *K8sCommandBuilder) GenerateDeploymentYAML() string {
	c := b.config

	replicas := c.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	yaml := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s
        imagePullPolicy: %s
`,
		c.ResourceName, b.namespace, c.ResourceName,
		replicas, c.ResourceName, c.ResourceName,
		c.ContainerName, c.Image, b.getImagePullPolicy())

	// 添加端口
	if len(c.ServicePorts) > 0 {
		yaml += "        ports:\n"
		for _, port := range c.ServicePorts {
			protocol := port.Protocol
			if protocol == "" {
				protocol = "TCP"
			}
			yaml += fmt.Sprintf("        - containerPort: %d\n          protocol: %s\n",
				port.TargetPort, protocol)
		}
	}

	// 添加资源限制
	if c.CPURequest != "" || c.MemoryRequest != "" || c.CPULimit != "" || c.MemoryLimit != "" {
		yaml += "        resources:\n"
		if c.CPURequest != "" || c.MemoryRequest != "" {
			yaml += "          requests:\n"
			if c.CPURequest != "" {
				yaml += fmt.Sprintf("            cpu: %s\n", c.CPURequest)
			}
			if c.MemoryRequest != "" {
				yaml += fmt.Sprintf("            memory: %s\n", c.MemoryRequest)
			}
		}
		if c.CPULimit != "" || c.MemoryLimit != "" {
			yaml += "          limits:\n"
			if c.CPULimit != "" {
				yaml += fmt.Sprintf("            cpu: %s\n", c.CPULimit)
			}
			if c.MemoryLimit != "" {
				yaml += fmt.Sprintf("            memory: %s\n", c.MemoryLimit)
			}
		}
	}

	// 添加环境变量
	if len(c.Environment) > 0 {
		yaml += "        env:\n"
		for k, v := range c.Environment {
			yaml += fmt.Sprintf("        - name: %s\n          value: \"%s\"\n", k, v)
		}
	}

	// 添加 imagePullSecrets
	if c.ImagePullSecret != "" {
		yaml += fmt.Sprintf("      imagePullSecrets:\n      - name: %s\n", c.ImagePullSecret)
	}

	return yaml
}

func (b *K8sCommandBuilder) getImagePullPolicy() string {
	if b.config.ImagePullPolicy != "" {
		return b.config.ImagePullPolicy
	}
	return "IfNotPresent"
}
