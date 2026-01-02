package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/release/models"
	"github.com/voilet/quic-flow/pkg/release/variable"
)

// CommandSender 命令发送接口
type CommandSender interface {
	SendCommand(clientID, commandType string, payload json.RawMessage, timeout time.Duration) (*command.Command, error)
	GetCommand(commandID string) (*command.Command, error)
}

// RemoteExecutor 远程执行器
// 用于通过 QUIC 发送发布命令到 Client 端执行
type RemoteExecutor struct {
	cmdSender  CommandSender
	varManager *variable.Manager
}

// NewRemoteExecutor 创建远程执行器
func NewRemoteExecutor(cmdSender CommandSender, varManager *variable.Manager) *RemoteExecutor {
	return &RemoteExecutor{
		cmdSender:  cmdSender,
		varManager: varManager,
	}
}

// RemoteExecuteRequest 远程执行请求
type RemoteExecuteRequest struct {
	// 发布信息
	ReleaseID string
	TargetID  string
	ClientID  string // QUIC 客户端 ID
	Operation models.OperationType
	Version   string

	// 脚本配置
	Config *models.ScriptDeployConfig

	// 变量上下文
	VarContext *variable.Context

	// 超时设置
	Timeout int
}

// RemoteExecuteResult 远程执行结果
type RemoteExecuteResult struct {
	Success    bool
	ExitCode   int
	Output     string
	Error      string
	Duration   time.Duration
	StartedAt  time.Time
	FinishedAt time.Time
}

// ContainerDeployRequest 容器部署请求
type ContainerDeployRequest struct {
	ReleaseID     string
	TargetID      string
	ClientID      string
	Operation     models.OperationType
	Version       string
	Config        *models.ContainerDeployConfig
	Image         string            // 解析后的镜像地址
	ContainerName string            // 解析后的容器名称
	Environment   map[string]string // 解析后的环境变量
	VarContext    *variable.Context
}

// ContainerDeployResult 容器部署结果
type ContainerDeployResult struct {
	Success       bool
	ContainerID   string
	ContainerName string
	ImagePulled   bool
	OldRemoved    bool
	Output        string
	Error         string
	Duration      time.Duration
	StartedAt     time.Time
	FinishedAt    time.Time
}

// GitPullDeployRequest Git 拉取部署请求
type GitPullDeployRequest struct {
	ReleaseID   string
	TargetID    string
	ClientID    string
	Operation   models.OperationType
	Version     string
	Config      *models.GitPullDeployConfig
	RepoURL     string            // 解析后的仓库地址
	Branch      string            // 解析后的分支
	WorkDir     string            // 解析后的工作目录
	PreScript   string            // 解析后的部署前脚本
	PostScript  string            // 解析后的部署后脚本
	Environment map[string]string // 解析后的环境变量
	VarContext  *variable.Context
}

// GitPullDeployResult Git 拉取部署结果
type GitPullDeployResult struct {
	Success        bool
	GitOutput      string
	ScriptOutput   string
	Commit         string
	Branch         string
	BackupPath     string
	CleanedBefore  bool
	BackedUpBefore bool
	Error          string
	Duration       time.Duration
	StartedAt      time.Time
	FinishedAt     time.Time
}

// GitVersionsRequest Git 版本查询请求
type GitVersionsRequest struct {
	ClientID string                      // QUIC 客户端 ID
	Config   *models.GitPullDeployConfig // Git 配置
	RepoURL  string                      // 解析后的仓库地址
	WorkDir  string                      // 工作目录（如已 clone）
	MaxTags    int                       // 最大返回 tag 数量
	MaxCommits int                       // 最大返回 commit 数量
	IncludeBranches bool                 // 是否包含分支列表
}

// K8sDeployRequest Kubernetes 部署请求
type K8sDeployRequest struct {
	ReleaseID   string
	TargetID    string
	ClientID    string
	Operation   models.OperationType
	Version     string
	Config      *models.KubernetesDeployConfig
	Image       string            // 解析后的镜像地址
	YAML        string            // 解析后的 YAML
	Environment map[string]string // 解析后的环境变量
	ToRevision  int               // 回滚到指定版本
	VarContext  *variable.Context
}

// K8sDeployResult Kubernetes 部署结果
type K8sDeployResult struct {
	Success       bool
	Namespace     string
	ResourceType  string
	ResourceName  string
	Image         string
	Replicas      int
	ReadyReplicas int
	Revision      int
	RolloutStatus string
	Output        string
	Error         string
	Duration      time.Duration
	StartedAt     time.Time
	FinishedAt    time.Time
}

// GitVersionsResult Git 版本查询结果
type GitVersionsResult struct {
	Success       bool
	RepoURL       string
	DefaultBranch string
	Tags          []command.GitTag
	Branches      []command.GitBranch
	RecentCommits []command.GitCommit
	CurrentCommit string
	CurrentBranch string
	Error         string
}

// Execute 执行远程发布
func (e *RemoteExecutor) Execute(ctx context.Context, req *RemoteExecuteRequest) (*RemoteExecuteResult, error) {
	result := &RemoteExecuteResult{
		StartedAt: time.Now(),
	}

	// 获取脚本内容
	script, err := e.getScript(req.Operation, req.Config)
	if err != nil {
		result.Error = err.Error()
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析变量
	resolvedScript, err := e.varManager.Resolve(ctx, script, req.VarContext)
	if err != nil {
		result.Error = fmt.Sprintf("resolve variables: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析工作目录
	workDir := ""
	if req.Config != nil && req.Config.WorkDir != "" {
		workDir, err = e.varManager.Resolve(ctx, req.Config.WorkDir, req.VarContext)
		if err != nil {
			result.Error = fmt.Sprintf("resolve work dir: %v", err)
			result.FinishedAt = time.Now()
			result.Duration = result.FinishedAt.Sub(result.StartedAt)
			return result, err
		}
	}

	// 解析环境变量
	env := make(map[string]string)
	if req.Config != nil && req.Config.Environment != nil {
		for k, v := range req.Config.Environment {
			resolved, err := e.varManager.Resolve(ctx, v, req.VarContext)
			if err != nil {
				result.Error = fmt.Sprintf("resolve env %s: %v", k, err)
				result.FinishedAt = time.Now()
				result.Duration = result.FinishedAt.Sub(result.StartedAt)
				return result, err
			}
			env[k] = resolved
		}
	}

	// 获取超时时间
	timeout := e.getTimeout(req.Operation, req.Config)
	if req.Timeout > 0 {
		timeout = req.Timeout
	}

	// 构建命令参数
	params := command.ReleaseExecuteParams{
		ReleaseID:   req.ReleaseID,
		TargetID:    req.TargetID,
		Operation:   e.toReleaseOpType(req.Operation),
		Version:     req.Version,
		Script:      resolvedScript,
		WorkDir:     workDir,
		Environment: env,
		Timeout:     timeout,
	}

	if req.Config != nil && req.Config.Interpreter != "" {
		params.Interpreter = req.Config.Interpreter
	}

	// 序列化参数
	payload, err := json.Marshal(params)
	if err != nil {
		result.Error = fmt.Sprintf("marshal params: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 发送命令
	cmd, err := e.cmdSender.SendCommand(
		req.ClientID,
		command.CmdReleaseExecute,
		payload,
		time.Duration(timeout+30)*time.Second, // 额外30秒作为网络缓冲
	)
	if err != nil {
		result.Error = fmt.Sprintf("send command: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 等待命令完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, time.Duration(timeout+60)*time.Second)
	if err != nil {
		result.Error = fmt.Sprintf("wait completion: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析结果
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if finalCmd.Status == command.CommandStatusCompleted {
		var execResult command.ReleaseExecuteResult
		if err := json.Unmarshal(finalCmd.Result, &execResult); err != nil {
			result.Error = fmt.Sprintf("unmarshal result: %v", err)
			return result, err
		}

		result.Success = execResult.Success
		result.ExitCode = execResult.ExitCode
		result.Output = execResult.Output
		result.Error = execResult.Error
	} else {
		result.Success = false
		result.ExitCode = -1
		result.Error = finalCmd.Error
		if result.Error == "" {
			result.Error = fmt.Sprintf("command status: %s", finalCmd.Status)
		}
	}

	return result, nil
}

// CheckInstallation 远程检查安装状态
func (e *RemoteExecutor) CheckInstallation(ctx context.Context, clientID string, workDir string, varCtx *variable.Context) (bool, string, error) {
	// 解析工作目录
	resolvedWorkDir, err := e.varManager.Resolve(ctx, workDir, varCtx)
	if err != nil {
		return false, "", err
	}

	// 构建检查参数
	params := command.ReleaseCheckParams{
		WorkDir: resolvedWorkDir,
	}

	payload, err := json.Marshal(params)
	if err != nil {
		return false, "", err
	}

	// 发送检查命令
	cmd, err := e.cmdSender.SendCommand(
		clientID,
		command.CmdReleaseCheck,
		payload,
		30*time.Second,
	)
	if err != nil {
		return false, "", err
	}

	// 等待完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, 60*time.Second)
	if err != nil {
		return false, "", err
	}

	if finalCmd.Status != command.CommandStatusCompleted {
		return false, "", fmt.Errorf("check command failed: %s", finalCmd.Error)
	}

	// 解析结果
	var checkResult command.ReleaseCheckResult
	if err := json.Unmarshal(finalCmd.Result, &checkResult); err != nil {
		return false, "", err
	}

	return checkResult.Installed, checkResult.Version, nil
}

// DetermineOperation 确定操作类型
func (e *RemoteExecutor) DetermineOperation(ctx context.Context, clientID string, config *models.ScriptDeployConfig, varCtx *variable.Context, requestedOp models.OperationType) (models.OperationType, error) {
	// 如果明确指定了操作类型（非 deploy），直接返回
	if requestedOp != models.OperationTypeDeploy {
		return requestedOp, nil
	}

	// deploy 类型需要检查当前安装状态
	if config == nil || config.WorkDir == "" {
		return models.OperationTypeInstall, nil
	}

	installed, _, err := e.CheckInstallation(ctx, clientID, config.WorkDir, varCtx)
	if err != nil {
		return "", fmt.Errorf("check installation: %w", err)
	}

	if installed {
		return models.OperationTypeUpdate, nil
	}
	return models.OperationTypeInstall, nil
}

// getScript 获取脚本内容
func (e *RemoteExecutor) getScript(op models.OperationType, config *models.ScriptDeployConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("script config is nil")
	}

	switch op {
	case models.OperationTypeInstall:
		if config.InstallScript == "" {
			return "", fmt.Errorf("install script is empty")
		}
		return config.InstallScript, nil

	case models.OperationTypeUpdate, models.OperationTypeDeploy:
		if config.UpdateScript == "" {
			return "", fmt.Errorf("update script is empty")
		}
		return config.UpdateScript, nil

	case models.OperationTypeRollback:
		if config.RollbackScript == "" {
			return "", fmt.Errorf("rollback script is empty")
		}
		return config.RollbackScript, nil

	case models.OperationTypeUninstall:
		if config.UninstallScript == "" {
			return "", fmt.Errorf("uninstall script is empty")
		}
		return config.UninstallScript, nil

	default:
		return "", fmt.Errorf("unknown operation type: %s", op)
	}
}

// getTimeout 获取超时时间
func (e *RemoteExecutor) getTimeout(op models.OperationType, config *models.ScriptDeployConfig) int {
	defaultTimeout := 300 // 5 minutes

	if config == nil {
		return defaultTimeout
	}

	switch op {
	case models.OperationTypeInstall:
		if config.Timeouts.Install > 0 {
			return config.Timeouts.Install
		}
		return 600 // 10 minutes

	case models.OperationTypeUpdate, models.OperationTypeDeploy:
		if config.Timeouts.Update > 0 {
			return config.Timeouts.Update
		}
		return 300 // 5 minutes

	case models.OperationTypeRollback:
		if config.Timeouts.Rollback > 0 {
			return config.Timeouts.Rollback
		}
		return 180 // 3 minutes

	case models.OperationTypeUninstall:
		if config.Timeouts.Uninstall > 0 {
			return config.Timeouts.Uninstall
		}
		return 120 // 2 minutes

	default:
		return defaultTimeout
	}
}

// toReleaseOpType 转换操作类型
func (e *RemoteExecutor) toReleaseOpType(op models.OperationType) command.ReleaseOperationType {
	switch op {
	case models.OperationTypeDeploy:
		return command.ReleaseOpDeploy
	case models.OperationTypeInstall:
		return command.ReleaseOpInstall
	case models.OperationTypeUpdate:
		return command.ReleaseOpUpdate
	case models.OperationTypeRollback:
		return command.ReleaseOpRollback
	case models.OperationTypeUninstall:
		return command.ReleaseOpUninstall
	default:
		return command.ReleaseOpDeploy
	}
}

// waitForCompletion 等待命令完成
func (e *RemoteExecutor) waitForCompletion(ctx context.Context, cmd *command.Command, timeout time.Duration) (*command.Command, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	commandID := cmd.CommandID

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// 从 CommandManager 获取最新状态
			latestCmd, err := e.cmdSender.GetCommand(commandID)
			if err != nil {
				return nil, fmt.Errorf("get command status: %w", err)
			}

			switch latestCmd.Status {
			case command.CommandStatusCompleted, command.CommandStatusFailed, command.CommandStatusTimeout, command.CommandStatusCancelled:
				return latestCmd, nil
			}

			if time.Now().After(deadline) {
				return nil, fmt.Errorf("wait timeout")
			}
		}
	}
}

// ExecuteScript 直接执行脚本（简化接口）
func (e *RemoteExecutor) ExecuteScript(ctx context.Context, clientID, script, workDir string) (*RemoteExecuteResult, error) {
	result := &RemoteExecuteResult{
		StartedAt: time.Now(),
	}

	// 构建命令参数
	params := command.ReleaseExecuteParams{
		Operation:   command.ReleaseOpDeploy,
		Script:      script,
		WorkDir:     workDir,
		Interpreter: "/bin/bash",
		Timeout:     300, // 5 minutes default
	}

	// 序列化参数
	payload, err := json.Marshal(params)
	if err != nil {
		result.Error = fmt.Sprintf("marshal params: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 发送命令
	cmd, err := e.cmdSender.SendCommand(
		clientID,
		command.CmdReleaseExecute,
		payload,
		330*time.Second, // 5 min + 30 sec buffer
	)
	if err != nil {
		result.Error = fmt.Sprintf("send command: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 等待命令完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, 360*time.Second)
	if err != nil {
		result.Error = fmt.Sprintf("wait completion: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析结果
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if finalCmd.Status == command.CommandStatusCompleted {
		var execResult command.ReleaseExecuteResult
		if err := json.Unmarshal(finalCmd.Result, &execResult); err != nil {
			result.Error = fmt.Sprintf("unmarshal result: %v", err)
			return result, err
		}

		result.Success = execResult.Success
		result.ExitCode = execResult.ExitCode
		result.Output = execResult.Output
		result.Error = execResult.Error
	} else {
		result.Success = false
		result.ExitCode = -1
		result.Error = finalCmd.Error
		if result.Error == "" {
			result.Error = fmt.Sprintf("command status: %s", finalCmd.Status)
		}
	}

	return result, nil
}

// ExecuteContainerDeploy 执行容器部署
func (e *RemoteExecutor) ExecuteContainerDeploy(ctx context.Context, req *ContainerDeployRequest) (*ContainerDeployResult, error) {
	result := &ContainerDeployResult{
		StartedAt: time.Now(),
	}

	// 构建命令参数
	params := command.ContainerDeployParams{
		ReleaseID:     req.ReleaseID,
		TargetID:      req.TargetID,
		Operation:     e.toReleaseOpType(req.Operation),
		Version:       req.Version,
		Image:         req.Image,
		ContainerName: req.ContainerName,
		Environment:   req.Environment,
	}

	// 从配置中复制其他参数
	if req.Config != nil {
		params.Registry = req.Config.Registry
		params.RegistryUser = req.Config.RegistryUser
		params.RegistryPass = req.Config.RegistryPass
		params.ImagePullPolicy = req.Config.ImagePullPolicy
		params.RestartPolicy = req.Config.RestartPolicy
		params.Command = req.Config.Command
		params.Entrypoint = req.Config.Entrypoint
		params.MemoryLimit = req.Config.MemoryLimit
		params.CPULimit = req.Config.CPULimit
		params.StopTimeout = req.Config.StopTimeout
		params.RemoveOld = req.Config.RemoveOld
		params.PullBeforeStop = req.Config.PullBeforeStop

		// 端口映射
		for _, p := range req.Config.Ports {
			params.Ports = append(params.Ports, command.PortMappingCmd{
				HostPort:      p.HostPort,
				ContainerPort: p.ContainerPort,
				Protocol:      p.Protocol,
				HostIP:        p.HostIP,
			})
		}

		// 卷挂载
		for _, v := range req.Config.Volumes {
			params.Volumes = append(params.Volumes, command.VolumeMountCmd{
				HostPath:      v.HostPath,
				ContainerPath: v.ContainerPath,
				ReadOnly:      v.ReadOnly,
			})
		}

		// 网络
		params.Networks = req.Config.Networks

		// 健康检查
		if req.Config.HealthCheck != nil {
			params.HealthCheck = &command.ContainerHealthCheckCmd{
				Command:     req.Config.HealthCheck.Command,
				Interval:    req.Config.HealthCheck.Interval,
				Timeout:     req.Config.HealthCheck.Timeout,
				Retries:     req.Config.HealthCheck.Retries,
				StartPeriod: req.Config.HealthCheck.StartPeriod,
			}
		}
	}

	// 序列化参数
	payload, err := json.Marshal(params)
	if err != nil {
		result.Error = fmt.Sprintf("marshal params: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 获取超时时间
	timeout := 600 // 默认10分钟
	if req.Config != nil && req.Config.StopTimeout > 0 {
		timeout = req.Config.StopTimeout + 300 // 停止超时 + 5分钟缓冲
	}

	// 发送命令
	cmd, err := e.cmdSender.SendCommand(
		req.ClientID,
		command.CmdContainerDeploy,
		payload,
		time.Duration(timeout+30)*time.Second,
	)
	if err != nil {
		result.Error = fmt.Sprintf("send command: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 等待命令完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, time.Duration(timeout+60)*time.Second)
	if err != nil {
		result.Error = fmt.Sprintf("wait completion: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析结果
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if finalCmd.Status == command.CommandStatusCompleted {
		var deployResult command.ContainerDeployResult
		if err := json.Unmarshal(finalCmd.Result, &deployResult); err != nil {
			result.Error = fmt.Sprintf("unmarshal result: %v", err)
			return result, err
		}

		result.Success = deployResult.Success
		result.ContainerID = deployResult.ContainerID
		result.ContainerName = deployResult.ContainerName
		result.ImagePulled = deployResult.ImagePulled
		result.OldRemoved = deployResult.OldRemoved
		result.Output = deployResult.Output
		result.Error = deployResult.Error
	} else {
		result.Success = false
		result.Error = finalCmd.Error
		if result.Error == "" {
			result.Error = fmt.Sprintf("command status: %s", finalCmd.Status)
		}
	}

	return result, nil
}

// ExecuteGitPullDeploy 执行 Git 拉取部署
func (e *RemoteExecutor) ExecuteGitPullDeploy(ctx context.Context, req *GitPullDeployRequest) (*GitPullDeployResult, error) {
	result := &GitPullDeployResult{
		StartedAt: time.Now(),
	}

	// 构建命令参数
	params := command.GitPullDeployParams{
		ReleaseID:   req.ReleaseID,
		TargetID:    req.TargetID,
		Operation:   e.toReleaseOpType(req.Operation),
		Version:     req.Version,
		RepoURL:     req.RepoURL,
		Branch:      req.Branch,
		WorkDir:     req.WorkDir,
		PreScript:   req.PreScript,
		PostScript:  req.PostScript,
		Environment: req.Environment,
	}

	// 从配置中复制其他参数
	if req.Config != nil {
		params.Tag = req.Config.Tag
		params.Commit = req.Config.Commit
		params.Depth = req.Config.Depth
		params.Submodules = req.Config.Submodules
		params.AuthType = req.Config.AuthType
		params.SSHKey = req.Config.SSHKey
		params.Token = req.Config.Token
		params.Username = req.Config.Username
		params.Password = req.Config.Password
		params.CleanBefore = req.Config.CleanBefore
		params.BackupBefore = req.Config.BackupBefore
		params.BackupDir = req.Config.BackupDir
		params.Interpreter = req.Config.Interpreter
		params.CloneTimeout = req.Config.CloneTimeout
		params.ScriptTimeout = req.Config.ScriptTimeout
	}

	// 序列化参数
	payload, err := json.Marshal(params)
	if err != nil {
		result.Error = fmt.Sprintf("marshal params: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 计算超时时间
	timeout := 600 // 默认10分钟
	if req.Config != nil {
		if req.Config.CloneTimeout > 0 {
			timeout = req.Config.CloneTimeout
		}
		if req.Config.ScriptTimeout > 0 {
			timeout += req.Config.ScriptTimeout
		}
	}

	// 发送命令
	cmd, err := e.cmdSender.SendCommand(
		req.ClientID,
		command.CmdGitPullDeploy,
		payload,
		time.Duration(timeout+30)*time.Second,
	)
	if err != nil {
		result.Error = fmt.Sprintf("send command: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 等待命令完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, time.Duration(timeout+60)*time.Second)
	if err != nil {
		result.Error = fmt.Sprintf("wait completion: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析结果
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if finalCmd.Status == command.CommandStatusCompleted {
		var deployResult command.GitPullDeployResult
		if err := json.Unmarshal(finalCmd.Result, &deployResult); err != nil {
			result.Error = fmt.Sprintf("unmarshal result: %v", err)
			return result, err
		}

		result.Success = deployResult.Success
		result.GitOutput = deployResult.GitOutput
		result.ScriptOutput = deployResult.ScriptOutput
		result.Commit = deployResult.Commit
		result.Branch = deployResult.Branch
		result.BackupPath = deployResult.BackupPath
		result.CleanedBefore = deployResult.CleanedBefore
		result.BackedUpBefore = deployResult.BackedUpBefore
		result.Error = deployResult.Error
	} else {
		result.Success = false
		result.Error = finalCmd.Error
		if result.Error == "" {
			result.Error = fmt.Sprintf("command status: %s", finalCmd.Status)
		}
	}

	return result, nil
}

// FetchGitVersions 获取 Git 仓库版本信息
func (e *RemoteExecutor) FetchGitVersions(ctx context.Context, req *GitVersionsRequest) (*GitVersionsResult, error) {
	result := &GitVersionsResult{
		RepoURL: req.RepoURL,
	}

	// 设置默认值
	maxTags := req.MaxTags
	if maxTags <= 0 {
		maxTags = 20
	}
	maxCommits := req.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 10
	}

	// 构建命令参数
	params := command.GitVersionsParams{
		RepoURL:         req.RepoURL,
		WorkDir:         req.WorkDir,
		MaxTags:         maxTags,
		MaxCommits:      maxCommits,
		IncludeBranches: req.IncludeBranches,
	}

	// 从配置中复制认证信息
	if req.Config != nil {
		params.AuthType = req.Config.AuthType
		params.SSHKey = req.Config.SSHKey
		params.Token = req.Config.Token
		params.Username = req.Config.Username
		params.Password = req.Config.Password
	}

	// 序列化参数
	payload, err := json.Marshal(params)
	if err != nil {
		result.Error = fmt.Sprintf("marshal params: %v", err)
		return result, err
	}

	// 发送命令（30秒超时）
	cmd, err := e.cmdSender.SendCommand(
		req.ClientID,
		command.CmdGitVersions,
		payload,
		60*time.Second,
	)
	if err != nil {
		result.Error = fmt.Sprintf("send command: %v", err)
		return result, err
	}

	// 等待命令完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, 90*time.Second)
	if err != nil {
		result.Error = fmt.Sprintf("wait completion: %v", err)
		return result, err
	}

	// 解析结果
	if finalCmd.Status == command.CommandStatusCompleted {
		var versionsResult command.GitVersionsResult
		if err := json.Unmarshal(finalCmd.Result, &versionsResult); err != nil {
			result.Error = fmt.Sprintf("unmarshal result: %v", err)
			return result, err
		}

		result.Success = versionsResult.Success
		result.DefaultBranch = versionsResult.DefaultBranch
		result.Tags = versionsResult.Tags
		result.Branches = versionsResult.Branches
		result.RecentCommits = versionsResult.RecentCommits
		result.CurrentCommit = versionsResult.CurrentCommit
		result.CurrentBranch = versionsResult.CurrentBranch
		result.Error = versionsResult.Error
	} else {
		result.Success = false
		result.Error = finalCmd.Error
		if result.Error == "" {
			result.Error = fmt.Sprintf("command status: %s", finalCmd.Status)
		}
	}

	return result, nil
}

// ExecuteK8sDeploy 执行 Kubernetes 部署
func (e *RemoteExecutor) ExecuteK8sDeploy(ctx context.Context, req *K8sDeployRequest) (*K8sDeployResult, error) {
	result := &K8sDeployResult{
		StartedAt: time.Now(),
	}

	// 构建命令参数
	params := command.K8sDeployParams{
		ReleaseID: req.ReleaseID,
		TargetID:  req.TargetID,
		Operation: e.toReleaseOpType(req.Operation),
		Version:   req.Version,
		Image:     req.Image,
		YAML:      req.YAML,
	}

	// 从配置中复制其他参数
	if req.Config != nil {
		params.Namespace = req.Config.Namespace
		params.ResourceType = req.Config.ResourceType
		params.ResourceName = req.Config.ResourceName
		params.ContainerName = req.Config.ContainerName
		params.YAMLTemplate = req.Config.YAMLTemplate
		params.Registry = req.Config.Registry
		params.RegistryUser = req.Config.RegistryUser
		params.RegistryPass = req.Config.RegistryPass
		params.ImagePullPolicy = req.Config.ImagePullPolicy
		params.ImagePullSecret = req.Config.ImagePullSecret
		params.Replicas = req.Config.Replicas
		params.UpdateStrategy = req.Config.UpdateStrategy
		params.MaxUnavailable = req.Config.MaxUnavailable
		params.MaxSurge = req.Config.MaxSurge
		params.MinReadySeconds = req.Config.MinReadySeconds
		params.CPURequest = req.Config.CPURequest
		params.CPULimit = req.Config.CPULimit
		params.MemoryRequest = req.Config.MemoryRequest
		params.MemoryLimit = req.Config.MemoryLimit
		params.KubeConfig = req.Config.KubeConfig
		params.KubeContext = req.Config.KubeContext
		params.Timeout = req.Config.DeployTimeout
		params.RolloutTimeout = req.Config.RolloutTimeout

		// 环境变量（优先使用解析后的）
		if len(req.Environment) > 0 {
			params.Environment = req.Environment
		} else if len(req.Config.Environment) > 0 {
			params.Environment = req.Config.Environment
		}
	}

	// 回滚版本
	params.ToRevision = req.ToRevision

	// 序列化参数
	payload, err := json.Marshal(params)
	if err != nil {
		result.Error = fmt.Sprintf("marshal params: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 计算超时时间
	timeout := 600 // 默认10分钟
	if req.Config != nil {
		if req.Config.DeployTimeout > 0 {
			timeout = req.Config.DeployTimeout
		}
		if req.Config.RolloutTimeout > 0 && req.Config.RolloutTimeout > timeout {
			timeout = req.Config.RolloutTimeout
		}
	}

	// 发送命令
	cmd, err := e.cmdSender.SendCommand(
		req.ClientID,
		command.CmdK8sDeploy,
		payload,
		time.Duration(timeout+30)*time.Second,
	)
	if err != nil {
		result.Error = fmt.Sprintf("send command: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 等待命令完成
	finalCmd, err := e.waitForCompletion(ctx, cmd, time.Duration(timeout+60)*time.Second)
	if err != nil {
		result.Error = fmt.Sprintf("wait completion: %v", err)
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		return result, err
	}

	// 解析结果
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	if finalCmd.Status == command.CommandStatusCompleted {
		var deployResult command.K8sDeployResult
		if err := json.Unmarshal(finalCmd.Result, &deployResult); err != nil {
			result.Error = fmt.Sprintf("unmarshal result: %v", err)
			return result, err
		}

		result.Success = deployResult.Success
		result.Namespace = deployResult.Namespace
		result.ResourceType = deployResult.ResourceType
		result.ResourceName = deployResult.ResourceName
		result.Image = deployResult.Image
		result.Replicas = deployResult.Replicas
		result.ReadyReplicas = deployResult.ReadyReplicas
		result.Revision = deployResult.Revision
		result.RolloutStatus = deployResult.RolloutStatus
		result.Output = deployResult.Output
		result.Error = deployResult.Error
	} else {
		result.Success = false
		result.Error = finalCmd.Error
		if result.Error == "" {
			result.Error = fmt.Sprintf("command status: %s", finalCmd.Status)
		}
	}

	return result, nil
}
