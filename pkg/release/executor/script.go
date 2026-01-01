package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
	"github.com/voilet/quic-flow/pkg/release/variable"
)

// ScriptExecutor 脚本执行器
type ScriptExecutor struct {
	varManager *variable.Manager
	mu         sync.Mutex

	// 默认配置
	defaultInterpreter string
	defaultTimeout     int
}

// NewScriptExecutor 创建脚本执行器
func NewScriptExecutor(varManager *variable.Manager) *ScriptExecutor {
	return &ScriptExecutor{
		varManager:         varManager,
		defaultInterpreter: "/bin/bash",
		defaultTimeout:     300, // 5 minutes
	}
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	// 操作类型
	Operation models.OperationType

	// 脚本配置
	Config *models.ScriptDeployConfig

	// 变量上下文
	VarContext *variable.Context

	// 输出回调
	OnOutput func(line string)

	// 进度回调
	OnProgress func(progress int, message string)
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Success    bool
	ExitCode   int
	Output     string
	Error      string
	Duration   time.Duration
	StartedAt  time.Time
	FinishedAt time.Time
}

// Execute 执行脚本
func (e *ScriptExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := &ExecuteResult{
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

	// 获取超时时间
	timeout := e.getTimeout(req.Operation, req.Config)

	// 执行脚本
	execResult, err := e.executeScript(ctx, resolvedScript, req.Config, timeout, req.OnOutput)
	if err != nil {
		result.Error = err.Error()
	}

	result.Success = execResult.Success
	result.ExitCode = execResult.ExitCode
	result.Output = execResult.Output
	if execResult.Error != "" {
		result.Error = execResult.Error
	}
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)

	return result, nil
}

// getScript 获取脚本内容
func (e *ScriptExecutor) getScript(op models.OperationType, config *models.ScriptDeployConfig) (string, error) {
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
func (e *ScriptExecutor) getTimeout(op models.OperationType, config *models.ScriptDeployConfig) int {
	if config == nil {
		return e.defaultTimeout
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
		return e.defaultTimeout
	}
}

// executeScript 执行脚本
func (e *ScriptExecutor) executeScript(
	ctx context.Context,
	script string,
	config *models.ScriptDeployConfig,
	timeout int,
	onOutput func(string),
) (*ExecuteResult, error) {
	result := &ExecuteResult{}

	// 创建临时脚本文件
	tmpFile, err := os.CreateTemp("", "release-script-*.sh")
	if err != nil {
		return result, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入脚本内容
	if _, err := tmpFile.WriteString(script); err != nil {
		return result, fmt.Errorf("write script: %w", err)
	}
	tmpFile.Close()

	// 设置执行权限
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return result, fmt.Errorf("chmod: %w", err)
	}

	// 获取解释器
	interpreter := e.defaultInterpreter
	if config != nil && config.Interpreter != "" {
		interpreter = config.Interpreter
	}

	// 创建带超时的上下文
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 创建命令
	cmd := exec.CommandContext(execCtx, interpreter, tmpFile.Name())

	// 设置工作目录
	if config != nil && config.WorkDir != "" {
		// 确保目录存在
		if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
			return result, fmt.Errorf("create work dir: %w", err)
		}
		cmd.Dir = config.WorkDir
	}

	// 设置环境变量
	cmd.Env = os.Environ()
	if config != nil && config.Environment != nil {
		for k, v := range config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// 捕获输出
	var stdout, stderr bytes.Buffer
	if onOutput != nil {
		// 实时输出
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return result, fmt.Errorf("stdout pipe: %w", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return result, fmt.Errorf("stderr pipe: %w", err)
		}

		go e.streamOutput(stdoutPipe, &stdout, onOutput)
		go e.streamOutput(stderrPipe, &stderr, onOutput)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	// 执行
	err = cmd.Run()

	result.Output = stdout.String()

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Sprintf("script timeout after %d seconds", timeout)
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = stderr.String()
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
		return result, nil
	}

	result.Success = true
	result.ExitCode = 0
	return result, nil
}

// streamOutput 流式输出
func (e *ScriptExecutor) streamOutput(r io.Reader, buf *bytes.Buffer, onOutput func(string)) {
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			chunk := string(b[:n])
			buf.WriteString(chunk)

			// 按行回调
			lines := strings.Split(chunk, "\n")
			for _, line := range lines {
				if line != "" {
					onOutput(line)
				}
			}
		}
		if err != nil {
			break
		}
	}
}

// CheckInstallation 检查安装状态
func (e *ScriptExecutor) CheckInstallation(ctx context.Context, config *models.ScriptDeployConfig, varCtx *variable.Context) (bool, string, error) {
	if config == nil || config.WorkDir == "" {
		return false, "", nil
	}

	// 解析工作目录
	workDir, err := e.varManager.Resolve(ctx, config.WorkDir, varCtx)
	if err != nil {
		return false, "", err
	}

	// 检查目录是否存在
	info, err := os.Stat(workDir)
	if os.IsNotExist(err) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}

	if !info.IsDir() {
		return false, "", nil
	}

	// 检查版本文件
	versionFile := filepath.Join(workDir, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		// 目录存在但没有版本文件，可能是旧安装
		return true, "unknown", nil
	}

	return true, strings.TrimSpace(string(data)), nil
}

// ValidateScript 验证脚本
func (e *ScriptExecutor) ValidateScript(script string) error {
	if script == "" {
		return fmt.Errorf("script is empty")
	}

	// 检查是否包含危险命令
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero",
		"mkfs",
		":(){:|:&};:",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(script, pattern) {
			return fmt.Errorf("script contains dangerous command: %s", pattern)
		}
	}

	return nil
}

// DetermineOperation 确定操作类型
func (e *ScriptExecutor) DetermineOperation(ctx context.Context, config *models.ScriptDeployConfig, varCtx *variable.Context, requestedOp models.OperationType) (models.OperationType, error) {
	// 如果明确指定了操作类型（非 deploy），直接返回
	if requestedOp != models.OperationTypeDeploy {
		return requestedOp, nil
	}

	// deploy 类型需要检查当前安装状态
	installed, _, err := e.CheckInstallation(ctx, config, varCtx)
	if err != nil {
		return "", fmt.Errorf("check installation: %w", err)
	}

	if installed {
		return models.OperationTypeUpdate, nil
	}
	return models.OperationTypeInstall, nil
}
