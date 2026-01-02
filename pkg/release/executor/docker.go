package executor

import (
	"fmt"
	"strings"

	"github.com/voilet/quic-flow/pkg/release/models"
)

// DockerCommandBuilder Docker 命令构建器
type DockerCommandBuilder struct {
	config *models.ContainerDeployConfig
}

// NewDockerCommandBuilder 创建 Docker 命令构建器
func NewDockerCommandBuilder(config *models.ContainerDeployConfig) *DockerCommandBuilder {
	return &DockerCommandBuilder{config: config}
}

// BuildRunCommand 构建 docker run 命令
func (b *DockerCommandBuilder) BuildRunCommand() (string, error) {
	if b.config == nil {
		return "", fmt.Errorf("container config is nil")
	}

	if b.config.Image == "" {
		return "", fmt.Errorf("image is required")
	}

	args := []string{"docker", "run", "-d"}

	// 容器名称
	if b.config.ContainerName != "" {
		args = append(args, "--name", b.config.ContainerName)
	}

	// 镜像拉取策略
	if b.config.ImagePullPolicy != "" {
		args = append(args, "--pull", b.config.ImagePullPolicy)
	}

	// 平台
	if b.config.Platform != "" {
		args = append(args, "--platform", b.config.Platform)
	}

	// 主机名和域名
	if b.config.Hostname != "" {
		args = append(args, "--hostname", b.config.Hostname)
	}
	if b.config.Domainname != "" {
		args = append(args, "--domainname", b.config.Domainname)
	}

	// 用户配置
	if b.config.User != "" {
		args = append(args, "--user", b.config.User)
	}
	for _, group := range b.config.GroupAdd {
		args = append(args, "--group-add", group)
	}

	// 工作目录
	if b.config.WorkingDir != "" {
		args = append(args, "--workdir", b.config.WorkingDir)
	}

	// 环境变量
	for key, value := range b.config.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// 标签
	for key, value := range b.config.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// 端口映射
	for _, port := range b.config.Ports {
		portArg := b.buildPortMapping(port)
		args = append(args, "-p", portArg)
	}

	// 暴露端口
	for _, port := range b.config.ExposePorts {
		args = append(args, "--expose", fmt.Sprintf("%d", port))
	}

	// 网络配置
	args = append(args, b.buildNetworkArgs()...)

	// 存储配置
	args = append(args, b.buildStorageArgs()...)

	// 安全配置
	args = append(args, b.buildSecurityArgs()...)

	// 设备配置
	args = append(args, b.buildDeviceArgs()...)

	// 资源限制
	args = append(args, b.buildResourceArgs()...)

	// 运行时配置
	args = append(args, b.buildRuntimeArgs()...)

	// 日志配置
	args = append(args, b.buildLoggingArgs()...)

	// 健康检查
	args = append(args, b.buildHealthCheckArgs()...)

	// 重启策略
	args = append(args, b.buildRestartArgs()...)

	// 自动删除
	if b.config.AutoRemove {
		args = append(args, "--rm")
	}

	// 镜像
	args = append(args, b.config.Image)

	// 启动命令
	if len(b.config.Command) > 0 {
		args = append(args, b.config.Command...)
	}

	return strings.Join(args, " "), nil
}

// BuildPullCommand 构建 docker pull 命令
func (b *DockerCommandBuilder) BuildPullCommand() string {
	args := []string{"docker", "pull"}

	if b.config.Platform != "" {
		args = append(args, "--platform", b.config.Platform)
	}

	args = append(args, b.config.Image)

	return strings.Join(args, " ")
}

// BuildStopCommand 构建 docker stop 命令
func (b *DockerCommandBuilder) BuildStopCommand(containerName string) string {
	args := []string{"docker", "stop"}

	if b.config.StopTimeout > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", b.config.StopTimeout))
	}

	args = append(args, containerName)

	return strings.Join(args, " ")
}

// BuildRemoveCommand 构建 docker rm 命令
func (b *DockerCommandBuilder) BuildRemoveCommand(containerName string) string {
	return fmt.Sprintf("docker rm -f %s", containerName)
}

// BuildLoginCommand 构建 docker login 命令
func (b *DockerCommandBuilder) BuildLoginCommand() string {
	if b.config.Registry == "" || b.config.RegistryUser == "" || b.config.RegistryPass == "" {
		return ""
	}

	return fmt.Sprintf("echo '%s' | docker login %s -u %s --password-stdin",
		b.config.RegistryPass, b.config.Registry, b.config.RegistryUser)
}

// buildPortMapping 构建端口映射参数
func (b *DockerCommandBuilder) buildPortMapping(port models.PortMapping) string {
	var parts []string

	// 主机 IP
	if port.HostIP != "" {
		parts = append(parts, port.HostIP)
	}

	// 主机端口
	parts = append(parts, fmt.Sprintf("%d", port.HostPort))

	// 容器端口
	containerPort := fmt.Sprintf("%d", port.ContainerPort)
	if port.Protocol != "" && port.Protocol != "tcp" {
		containerPort += "/" + port.Protocol
	}

	if len(parts) > 0 && port.HostIP != "" {
		return fmt.Sprintf("%s:%d:%s", port.HostIP, port.HostPort, containerPort)
	}
	return fmt.Sprintf("%d:%s", port.HostPort, containerPort)
}

// buildNetworkArgs 构建网络参数
func (b *DockerCommandBuilder) buildNetworkArgs() []string {
	var args []string

	// 网络模式
	if b.config.NetworkMode != "" {
		args = append(args, "--network", b.config.NetworkMode)
	}

	// 加入网络
	for _, network := range b.config.Networks {
		args = append(args, "--network", network)
	}

	// DNS 配置
	for _, dns := range b.config.DNS {
		args = append(args, "--dns", dns)
	}
	for _, search := range b.config.DNSSearch {
		args = append(args, "--dns-search", search)
	}
	for _, opt := range b.config.DNSOpt {
		args = append(args, "--dns-opt", opt)
	}

	// 额外主机
	for _, host := range b.config.ExtraHosts {
		args = append(args, "--add-host", host)
	}

	// MAC 地址
	if b.config.MacAddress != "" {
		args = append(args, "--mac-address", b.config.MacAddress)
	}

	// IP 地址
	if b.config.IPv4Address != "" {
		args = append(args, "--ip", b.config.IPv4Address)
	}
	if b.config.IPv6Address != "" {
		args = append(args, "--ip6", b.config.IPv6Address)
	}

	// 连接容器
	for _, link := range b.config.Links {
		args = append(args, "--link", link)
	}

	return args
}

// buildStorageArgs 构建存储参数
func (b *DockerCommandBuilder) buildStorageArgs() []string {
	var args []string

	// 卷挂载
	for _, vol := range b.config.Volumes {
		volArg := b.buildVolumeMount(vol)
		args = append(args, "-v", volArg)
	}

	// tmpfs 挂载
	for _, tmpfs := range b.config.TmpfsMounts {
		tmpfsArg := b.buildTmpfsMount(tmpfs)
		args = append(args, "--tmpfs", tmpfsArg)
	}

	// 卷驱动
	if b.config.VolumeDriver != "" {
		args = append(args, "--volume-driver", b.config.VolumeDriver)
	}

	// 存储选项
	for key, value := range b.config.StorageOpts {
		args = append(args, "--storage-opt", fmt.Sprintf("%s=%s", key, value))
	}

	return args
}

// buildVolumeMount 构建卷挂载参数
func (b *DockerCommandBuilder) buildVolumeMount(vol models.VolumeMount) string {
	mount := fmt.Sprintf("%s:%s", vol.HostPath, vol.ContainerPath)
	if vol.ReadOnly {
		mount += ":ro"
	}
	return mount
}

// buildTmpfsMount 构建 tmpfs 挂载参数
func (b *DockerCommandBuilder) buildTmpfsMount(tmpfs models.TmpfsMount) string {
	mount := tmpfs.ContainerPath

	var opts []string
	if tmpfs.Size != "" {
		opts = append(opts, fmt.Sprintf("size=%s", tmpfs.Size))
	}
	if tmpfs.Mode != 0 {
		opts = append(opts, fmt.Sprintf("mode=%04o", tmpfs.Mode))
	}

	if len(opts) > 0 {
		mount += ":" + strings.Join(opts, ",")
	}

	return mount
}

// buildSecurityArgs 构建安全参数
func (b *DockerCommandBuilder) buildSecurityArgs() []string {
	var args []string

	// 特权模式
	if b.config.Privileged {
		args = append(args, "--privileged")
	}

	// Capabilities
	for _, cap := range b.config.CapAdd {
		args = append(args, "--cap-add", cap)
	}
	for _, cap := range b.config.CapDrop {
		args = append(args, "--cap-drop", cap)
	}

	// 安全选项
	for _, opt := range b.config.SecurityOpt {
		args = append(args, "--security-opt", opt)
	}

	// 只读根文件系统
	if b.config.ReadOnlyRootfs {
		args = append(args, "--read-only")
	}

	// 禁止新权限
	if b.config.NoNewPrivileges {
		args = append(args, "--security-opt", "no-new-privileges")
	}

	// 用户命名空间
	if b.config.UsernsMode != "" {
		args = append(args, "--userns", b.config.UsernsMode)
	}

	return args
}

// buildDeviceArgs 构建设备参数
func (b *DockerCommandBuilder) buildDeviceArgs() []string {
	var args []string

	// 设备映射
	for _, device := range b.config.Devices {
		deviceArg := b.buildDeviceMapping(device)
		args = append(args, "--device", deviceArg)
	}

	// GPU 配置
	if b.config.GPUs != "" {
		args = append(args, "--gpus", b.config.GPUs)
	}

	// 设备 cgroup 规则
	for _, rule := range b.config.DeviceCgroupRules {
		args = append(args, "--device-cgroup-rule", rule)
	}

	return args
}

// buildDeviceMapping 构建设备映射参数
func (b *DockerCommandBuilder) buildDeviceMapping(device models.DeviceMapping) string {
	mapping := device.HostPath
	if device.ContainerPath != "" {
		mapping += ":" + device.ContainerPath
	}
	if device.Permissions != "" {
		if device.ContainerPath == "" {
			mapping += ":"
		}
		mapping += ":" + device.Permissions
	}
	return mapping
}

// buildResourceArgs 构建资源限制参数
func (b *DockerCommandBuilder) buildResourceArgs() []string {
	var args []string

	// 内存限制
	if b.config.MemoryLimit != "" {
		args = append(args, "--memory", b.config.MemoryLimit)
	}
	if b.config.MemoryReserve != "" {
		args = append(args, "--memory-reservation", b.config.MemoryReserve)
	}
	if b.config.MemorySwap != "" {
		args = append(args, "--memory-swap", b.config.MemorySwap)
	}
	if b.config.MemorySwappiness != nil {
		args = append(args, "--memory-swappiness", fmt.Sprintf("%d", *b.config.MemorySwappiness))
	}

	// CPU 限制
	if b.config.CPULimit != "" {
		args = append(args, "--cpus", b.config.CPULimit)
	}
	if b.config.CPUShares > 0 {
		args = append(args, "--cpu-shares", fmt.Sprintf("%d", b.config.CPUShares))
	}
	if b.config.CpusetCpus != "" {
		args = append(args, "--cpuset-cpus", b.config.CpusetCpus)
	}
	if b.config.CpusetMems != "" {
		args = append(args, "--cpuset-mems", b.config.CpusetMems)
	}

	// 进程数限制
	if b.config.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", b.config.PidsLimit))
	}

	// Ulimits
	for _, ulimit := range b.config.Ulimits {
		args = append(args, "--ulimit", fmt.Sprintf("%s=%d:%d", ulimit.Name, ulimit.Soft, ulimit.Hard))
	}

	// OOM 配置
	if b.config.OomKillDisable {
		args = append(args, "--oom-kill-disable")
	}
	if b.config.OomScoreAdj != 0 {
		args = append(args, "--oom-score-adj", fmt.Sprintf("%d", b.config.OomScoreAdj))
	}

	// 共享内存大小
	if b.config.ShmSize != "" {
		args = append(args, "--shm-size", b.config.ShmSize)
	}

	return args
}

// buildRuntimeArgs 构建运行时参数
func (b *DockerCommandBuilder) buildRuntimeArgs() []string {
	var args []string

	// 运行时
	if b.config.Runtime != "" {
		args = append(args, "--runtime", b.config.Runtime)
	}

	// Init 进程
	if b.config.Init {
		args = append(args, "--init")
	}

	// PID 模式
	if b.config.PidMode != "" {
		args = append(args, "--pid", b.config.PidMode)
	}

	// IPC 模式
	if b.config.IpcMode != "" {
		args = append(args, "--ipc", b.config.IpcMode)
	}

	// UTS 模式
	if b.config.UtsMode != "" {
		args = append(args, "--uts", b.config.UtsMode)
	}

	// Cgroup 父级
	if b.config.CgroupParent != "" {
		args = append(args, "--cgroup-parent", b.config.CgroupParent)
	}

	// 内核参数
	for key, value := range b.config.Sysctls {
		args = append(args, "--sysctl", fmt.Sprintf("%s=%s", key, value))
	}

	// 停止信号
	if b.config.StopSignal != "" {
		args = append(args, "--stop-signal", b.config.StopSignal)
	}

	// TTY 和 stdin
	if b.config.Tty {
		args = append(args, "-t")
	}
	if b.config.StdinOpen {
		args = append(args, "-i")
	}

	// 入口点
	if len(b.config.Entrypoint) > 0 {
		args = append(args, "--entrypoint", strings.Join(b.config.Entrypoint, " "))
	}

	return args
}

// buildLoggingArgs 构建日志参数
func (b *DockerCommandBuilder) buildLoggingArgs() []string {
	var args []string

	// 日志驱动
	if b.config.LogDriver != "" {
		args = append(args, "--log-driver", b.config.LogDriver)
	}

	// 日志选项
	for key, value := range b.config.LogOpts {
		args = append(args, "--log-opt", fmt.Sprintf("%s=%s", key, value))
	}

	return args
}

// buildHealthCheckArgs 构建健康检查参数
func (b *DockerCommandBuilder) buildHealthCheckArgs() []string {
	var args []string

	if b.config.HealthCheck == nil {
		return args
	}

	hc := b.config.HealthCheck

	// 健康检查命令
	if len(hc.Command) > 0 {
		args = append(args, "--health-cmd", strings.Join(hc.Command, " "))
	}

	// 检查间隔
	if hc.Interval > 0 {
		args = append(args, "--health-interval", fmt.Sprintf("%ds", hc.Interval))
	}

	// 超时时间
	if hc.Timeout > 0 {
		args = append(args, "--health-timeout", fmt.Sprintf("%ds", hc.Timeout))
	}

	// 重试次数
	if hc.Retries > 0 {
		args = append(args, "--health-retries", fmt.Sprintf("%d", hc.Retries))
	}

	// 启动等待期
	if hc.StartPeriod > 0 {
		args = append(args, "--health-start-period", fmt.Sprintf("%ds", hc.StartPeriod))
	}

	return args
}

// buildRestartArgs 构建重启策略参数
func (b *DockerCommandBuilder) buildRestartArgs() []string {
	var args []string

	if b.config.RestartPolicy == "" {
		return args
	}

	policy := b.config.RestartPolicy
	if policy == "on-failure" && b.config.RestartMaxRetries > 0 {
		policy = fmt.Sprintf("on-failure:%d", b.config.RestartMaxRetries)
	}

	args = append(args, "--restart", policy)

	return args
}

// BuildDeployScript 构建完整的部署脚本
func (b *DockerCommandBuilder) BuildDeployScript(containerName string) (string, error) {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	// 登录镜像仓库
	loginCmd := b.BuildLoginCommand()
	if loginCmd != "" {
		script.WriteString("# Login to registry\n")
		script.WriteString(loginCmd + "\n\n")
	}

	// 拉取镜像（如果配置了先拉取）
	if b.config.PullBeforeStop {
		script.WriteString("# Pull image before stopping\n")
		script.WriteString(b.BuildPullCommand() + "\n\n")
	}

	// 停止旧容器
	if b.config.RemoveOld {
		script.WriteString("# Stop and remove old container\n")
		script.WriteString(fmt.Sprintf("docker stop %s 2>/dev/null || true\n", containerName))
		script.WriteString(fmt.Sprintf("docker rm %s 2>/dev/null || true\n\n", containerName))
	}

	// 拉取镜像（如果没有配置先拉取）
	if !b.config.PullBeforeStop {
		script.WriteString("# Pull image\n")
		script.WriteString(b.BuildPullCommand() + "\n\n")
	}

	// 运行容器
	runCmd, err := b.BuildRunCommand()
	if err != nil {
		return "", err
	}
	script.WriteString("# Run container\n")
	script.WriteString(runCmd + "\n\n")

	// 等待健康检查
	if b.config.HealthCheck != nil && len(b.config.HealthCheck.Command) > 0 {
		script.WriteString("# Wait for container to be healthy\n")
		script.WriteString(fmt.Sprintf(`
for i in $(seq 1 30); do
    STATUS=$(docker inspect --format='{{.State.Health.Status}}' %s 2>/dev/null || echo "unknown")
    if [ "$STATUS" = "healthy" ]; then
        echo "Container is healthy"
        exit 0
    fi
    echo "Waiting for container to be healthy... ($i/30)"
    sleep 2
done

echo "Container health check timeout"
exit 1
`, containerName))
	}

	script.WriteString("\necho 'Deployment completed successfully'\n")

	return script.String(), nil
}

// BuildUninstallScript 构建卸载脚本
func (b *DockerCommandBuilder) BuildUninstallScript(containerName string) string {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")

	script.WriteString("# Stop container\n")
	script.WriteString(b.BuildStopCommand(containerName) + "\n\n")

	script.WriteString("# Remove container\n")
	script.WriteString(b.BuildRemoveCommand(containerName) + "\n\n")

	script.WriteString("echo 'Container removed successfully'\n")

	return script.String()
}

// GenerateDockerRunCommand 生成 docker run 命令（便捷函数）
func GenerateDockerRunCommand(config *models.ContainerDeployConfig) (string, error) {
	builder := NewDockerCommandBuilder(config)
	return builder.BuildRunCommand()
}

// GenerateDockerDeployScript 生成完整部署脚本（便捷函数）
func GenerateDockerDeployScript(config *models.ContainerDeployConfig, containerName string) (string, error) {
	builder := NewDockerCommandBuilder(config)
	return builder.BuildDeployScript(containerName)
}
