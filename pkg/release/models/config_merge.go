package models

// FinalContainerConfig 最终的容器部署配置（合并后）
// 用于实际执行部署时使用
type FinalContainerConfig struct {
	// ========== 镜像配置 ==========
	Image           string `json:"image"`
	Registry        string `json:"registry,omitempty"`
	RegistryUser    string `json:"registry_user,omitempty"`
	RegistryPass    string `json:"registry_pass,omitempty"`
	ImagePullPolicy string `json:"image_pull_policy,omitempty"`

	// ========== 容器基础配置 ==========
	ContainerName string            `json:"container_name"`
	Hostname      string            `json:"hostname,omitempty"`
	User          string            `json:"user,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Command       []string          `json:"command,omitempty"`
	Entrypoint    []string          `json:"entrypoint,omitempty"`

	// ========== 端口配置 ==========
	Ports []PortMapping `json:"ports,omitempty"`

	// ========== 网络配置 ==========
	NetworkMode string   `json:"network_mode,omitempty"`
	Networks    []string `json:"networks,omitempty"`
	DNS         []string `json:"dns,omitempty"`
	ExtraHosts  []string `json:"extra_hosts,omitempty"`

	// ========== 存储配置 ==========
	Volumes     []VolumeMount `json:"volumes,omitempty"`
	TmpfsMounts []TmpfsMount  `json:"tmpfs_mounts,omitempty"`

	// ========== 安全配置 ==========
	Privileged     bool     `json:"privileged,omitempty"`
	CapAdd         []string `json:"cap_add,omitempty"`
	CapDrop        []string `json:"cap_drop,omitempty"`
	SecurityOpt    []string `json:"security_opt,omitempty"`
	ReadOnlyRootfs bool     `json:"read_only_rootfs,omitempty"`

	// ========== 设备配置 ==========
	Devices []DeviceMapping `json:"devices,omitempty"`
	GPUs    string          `json:"gpus,omitempty"`

	// ========== 资源限制 ==========
	MemoryLimit   string `json:"memory_limit,omitempty"`
	MemoryReserve string `json:"memory_reserve,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`
	CPUShares     int    `json:"cpu_shares,omitempty"`

	// ========== 运行时配置 ==========
	Runtime    string            `json:"runtime,omitempty"`
	Init       bool              `json:"init,omitempty"`
	StopSignal string            `json:"stop_signal,omitempty"`
	Sysctls    map[string]string `json:"sysctls,omitempty"`

	// ========== 日志配置 ==========
	LogDriver string            `json:"log_driver,omitempty"`
	LogOpts   map[string]string `json:"log_opts,omitempty"`

	// ========== 健康检查 ==========
	HealthCheck *ContainerHealthCheck `json:"health_check,omitempty"`

	// ========== 重启策略 ==========
	RestartPolicy     string `json:"restart_policy,omitempty"`
	RestartMaxRetries int    `json:"restart_max_retries,omitempty"`

	// ========== 部署策略 ==========
	StopTimeout    int  `json:"stop_timeout,omitempty"`
	RemoveOld      bool `json:"remove_old,omitempty"`
	PullBeforeStop bool `json:"pull_before_stop,omitempty"`
	AutoRemove     bool `json:"auto_remove,omitempty"`

	// ========== 部署脚本 ==========
	PreScript  string `json:"pre_script,omitempty"`
	PostScript string `json:"post_script,omitempty"`
}

// FinalK8sConfig 最终的 K8s 部署配置（合并后）
type FinalK8sConfig struct {
	// ========== 集群配置 ==========
	KubeConfig  string `json:"kubeconfig,omitempty"`
	KubeContext string `json:"kube_context,omitempty"`
	Namespace   string `json:"namespace,omitempty"`

	// ========== 资源配置 ==========
	ResourceType  string `json:"resource_type,omitempty"`
	ResourceName  string `json:"resource_name,omitempty"`
	ContainerName string `json:"container_name,omitempty"`

	// ========== 镜像配置 ==========
	Image           string `json:"image"`
	Registry        string `json:"registry,omitempty"`
	ImagePullPolicy string `json:"image_pull_policy,omitempty"`
	ImagePullSecret string `json:"image_pull_secret,omitempty"`

	// ========== 副本配置 ==========
	Replicas    int  `json:"replicas"`
	AutoScaling bool `json:"auto_scaling,omitempty"`

	// ========== 资源限制 ==========
	CPURequest    string `json:"cpu_request,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`
	MemoryRequest string `json:"memory_request,omitempty"`
	MemoryLimit   string `json:"memory_limit,omitempty"`

	// ========== 环境变量 ==========
	Environment map[string]string `json:"environment,omitempty"`

	// ========== 启动命令 ==========
	Command    []string `json:"command,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`

	// ========== 服务配置 ==========
	ServiceType  string    `json:"service_type,omitempty"`
	ServicePorts []K8sPort `json:"service_ports,omitempty"`

	// ========== 更新策略 ==========
	UpdateStrategy  string `json:"update_strategy,omitempty"`
	MaxUnavailable  string `json:"max_unavailable,omitempty"`
	MaxSurge        string `json:"max_surge,omitempty"`
	MinReadySeconds int    `json:"min_ready_seconds,omitempty"`

	// ========== 超时配置 ==========
	DeployTimeout  int `json:"deploy_timeout,omitempty"`
	RolloutTimeout int `json:"rollout_timeout,omitempty"`

	// ========== YAML 配置 ==========
	YAML         string `json:"yaml,omitempty"`
	YAMLTemplate string `json:"yaml_template,omitempty"`

	// ========== 部署脚本 ==========
	PreScript  string `json:"pre_script,omitempty"`
	PostScript string `json:"post_script,omitempty"`
}

// MergeContainerConfig 合并容器部署配置
// 优先级: 项目配置 < 版本配置 < 任务覆盖配置
func MergeContainerConfig(
	project *ContainerDeployConfig,
	version *VersionDeployConfig,
	task *TaskOverrideConfig,
) *FinalContainerConfig {
	if project == nil {
		return nil
	}

	final := &FinalContainerConfig{
		// 从项目配置复制基础设施配置（不可覆盖）
		Registry:        project.Registry,
		RegistryUser:    project.RegistryUser,
		RegistryPass:    project.RegistryPass,
		ImagePullPolicy: project.ImagePullPolicy,
		ContainerName:   project.ContainerName,
		Hostname:        project.Hostname,
		User:            project.User,
		Ports:           project.Ports,
		NetworkMode:     project.NetworkMode,
		Networks:        project.Networks,
		DNS:             project.DNS,
		ExtraHosts:      project.ExtraHosts,
		Volumes:         project.Volumes,
		TmpfsMounts:     project.TmpfsMounts,
		Privileged:      project.Privileged,
		CapAdd:          project.CapAdd,
		CapDrop:         project.CapDrop,
		SecurityOpt:     project.SecurityOpt,
		ReadOnlyRootfs:  project.ReadOnlyRootfs,
		Devices:         project.Devices,
		GPUs:            project.GPUs,
		Runtime:         project.Runtime,
		Init:            project.Init,
		StopSignal:      project.StopSignal,
		Sysctls:         project.Sysctls,
		LogDriver:       project.LogDriver,
		LogOpts:         project.LogOpts,
		RestartPolicy:   project.RestartPolicy,
		StopTimeout:     project.StopTimeout,
		RemoveOld:       project.RemoveOld,
		PullBeforeStop:  project.PullBeforeStop,
		AutoRemove:      project.AutoRemove,
	}

	// 从项目配置复制默认值（可被覆盖）
	final.Image = project.Image
	final.WorkingDir = project.WorkingDir
	final.Environment = copyStringMap(project.Environment)
	final.Labels = copyStringMap(project.Labels)
	final.Command = copyStringSlice(project.Command)
	final.Entrypoint = copyStringSlice(project.Entrypoint)
	final.HealthCheck = project.HealthCheck
	// 项目脚本作为默认值
	final.PreScript = project.PreScript
	final.PostScript = project.PostScript

	// 使用默认资源限制
	if project.DefaultResources != nil {
		final.MemoryLimit = project.DefaultResources.MemoryLimit
		final.CPULimit = project.DefaultResources.CPULimit
		final.MemoryReserve = project.DefaultResources.MemoryRequest
	} else {
		// 回退到直接字段（向后兼容）
		final.MemoryLimit = project.MemoryLimit
		final.CPULimit = project.CPULimit
		final.MemoryReserve = project.MemoryReserve
	}

	// 应用版本配置覆盖
	if version != nil {
		if version.Image != "" {
			final.Image = version.Image
		}
		if version.WorkingDir != "" {
			final.WorkingDir = version.WorkingDir
		}
		if len(version.Environment) > 0 {
			final.Environment = mergeStringMaps(final.Environment, version.Environment)
		}
		if len(version.Command) > 0 {
			final.Command = version.Command
		}
		if len(version.Entrypoint) > 0 {
			final.Entrypoint = version.Entrypoint
		}
		if version.HealthCheck != nil {
			final.HealthCheck = version.HealthCheck
		}
		if version.Resources != nil {
			if version.Resources.MemoryLimit != "" {
				final.MemoryLimit = version.Resources.MemoryLimit
			}
			if version.Resources.CPULimit != "" {
				final.CPULimit = version.Resources.CPULimit
			}
			if version.Resources.MemoryRequest != "" {
				final.MemoryReserve = version.Resources.MemoryRequest
			}
		}
		// 条件覆盖：版本脚本非空时覆盖项目脚本
		if version.PreScript != "" {
			final.PreScript = version.PreScript
		}
		if version.PostScript != "" {
			final.PostScript = version.PostScript
		}
	}

	// 应用任务覆盖配置
	if task != nil {
		if task.Image != "" {
			final.Image = task.Image
		}
		if len(task.EnvironmentAdd) > 0 {
			final.Environment = mergeStringMaps(final.Environment, task.EnvironmentAdd)
		}
		if len(task.Command) > 0 {
			final.Command = task.Command
		}
		if task.Resources != nil {
			if task.Resources.MemoryLimit != "" {
				final.MemoryLimit = task.Resources.MemoryLimit
			}
			if task.Resources.CPULimit != "" {
				final.CPULimit = task.Resources.CPULimit
			}
			if task.Resources.MemoryRequest != "" {
				final.MemoryReserve = task.Resources.MemoryRequest
			}
		}
	}

	return final
}

// MergeK8sConfig 合并 K8s 部署配置
// 优先级: 项目配置 < 版本配置 < 任务覆盖配置
func MergeK8sConfig(
	project *KubernetesDeployConfig,
	version *VersionDeployConfig,
	task *TaskOverrideConfig,
) *FinalK8sConfig {
	if project == nil {
		return nil
	}

	final := &FinalK8sConfig{
		// 从项目配置复制基础设施配置（不可覆盖）
		KubeConfig:      project.KubeConfig,
		KubeContext:     project.KubeContext,
		Namespace:       project.Namespace,
		ResourceType:    project.ResourceType,
		ResourceName:    project.ResourceName,
		ContainerName:   project.ContainerName,
		Registry:        project.Registry,
		ImagePullPolicy: project.ImagePullPolicy,
		ImagePullSecret: project.ImagePullSecret,
		AutoScaling:     project.AutoScaling,
		ServiceType:     project.ServiceType,
		ServicePorts:    project.ServicePorts,
		UpdateStrategy:  project.UpdateStrategy,
		MaxUnavailable:  project.MaxUnavailable,
		MaxSurge:        project.MaxSurge,
		MinReadySeconds: project.MinReadySeconds,
		DeployTimeout:   project.DeployTimeout,
		RolloutTimeout:  project.RolloutTimeout,
		YAML:            project.YAML,
		YAMLTemplate:    project.YAMLTemplate,
	}

	// 从项目配置复制默认值（可被覆盖）
	final.Image = project.Image
	final.Environment = copyStringMap(project.Environment)
	// 项目脚本作为默认值
	final.PreScript = project.PreScript
	final.PostScript = project.PostScript

	// 使用默认副本数
	if project.DefaultReplicas > 0 {
		final.Replicas = project.DefaultReplicas
	} else if project.Replicas > 0 {
		final.Replicas = project.Replicas
	} else {
		final.Replicas = 1
	}

	// 使用默认资源限制
	if project.DefaultResources != nil {
		final.CPURequest = project.DefaultResources.CPURequest
		final.CPULimit = project.DefaultResources.CPULimit
		final.MemoryRequest = project.DefaultResources.MemoryRequest
		final.MemoryLimit = project.DefaultResources.MemoryLimit
	} else {
		// 回退到直接字段（向后兼容）
		final.CPURequest = project.CPURequest
		final.CPULimit = project.CPULimit
		final.MemoryRequest = project.MemoryRequest
		final.MemoryLimit = project.MemoryLimit
	}

	// 应用版本配置覆盖
	if version != nil {
		if version.Image != "" {
			final.Image = version.Image
		}
		if version.WorkingDir != "" {
			final.WorkingDir = version.WorkingDir
		}
		if len(version.Environment) > 0 {
			final.Environment = mergeStringMaps(final.Environment, version.Environment)
		}
		if len(version.Command) > 0 {
			final.Command = version.Command
		}
		if len(version.Entrypoint) > 0 {
			final.Entrypoint = version.Entrypoint
		}
		if version.Replicas != nil && *version.Replicas > 0 {
			final.Replicas = *version.Replicas
		}
		if version.Resources != nil {
			if version.Resources.CPURequest != "" {
				final.CPURequest = version.Resources.CPURequest
			}
			if version.Resources.CPULimit != "" {
				final.CPULimit = version.Resources.CPULimit
			}
			if version.Resources.MemoryRequest != "" {
				final.MemoryRequest = version.Resources.MemoryRequest
			}
			if version.Resources.MemoryLimit != "" {
				final.MemoryLimit = version.Resources.MemoryLimit
			}
		}
		// 处理 K8s YAML 覆盖
		if version.K8sYAMLFull != "" {
			final.YAML = version.K8sYAMLFull
		} else if version.K8sYAMLPatch != "" {
			// TODO: 实现 YAML patch 逻辑
			final.YAMLTemplate = version.K8sYAMLPatch
		}
		// 条件覆盖：版本脚本非空时覆盖项目脚本
		if version.PreScript != "" {
			final.PreScript = version.PreScript
		}
		if version.PostScript != "" {
			final.PostScript = version.PostScript
		}
	}

	// 应用任务覆盖配置
	if task != nil {
		if task.Image != "" {
			final.Image = task.Image
		}
		if len(task.EnvironmentAdd) > 0 {
			final.Environment = mergeStringMaps(final.Environment, task.EnvironmentAdd)
		}
		if len(task.Command) > 0 {
			final.Command = task.Command
		}
		if task.Replicas != nil && *task.Replicas > 0 {
			final.Replicas = *task.Replicas
		}
		if task.Resources != nil {
			if task.Resources.CPURequest != "" {
				final.CPURequest = task.Resources.CPURequest
			}
			if task.Resources.CPULimit != "" {
				final.CPULimit = task.Resources.CPULimit
			}
			if task.Resources.MemoryRequest != "" {
				final.MemoryRequest = task.Resources.MemoryRequest
			}
			if task.Resources.MemoryLimit != "" {
				final.MemoryLimit = task.Resources.MemoryLimit
			}
		}
	}

	return final
}

// 辅助函数

// copyStringMap 复制字符串映射
func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// mergeStringMaps 合并两个字符串映射（后者覆盖前者）
func mergeStringMaps(base, override map[string]string) map[string]string {
	if base == nil && override == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// copyStringSlice 复制字符串切片
func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	result := make([]string, len(s))
	copy(result, s)
	return result
}
