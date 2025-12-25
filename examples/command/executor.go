package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SimpleCommandExecutor 简单的命令执行器示例
type SimpleCommandExecutor struct{}

// RestartParams 重启命令参数
type RestartParams struct {
	DelaySeconds int `json:"delay_seconds"`
}

// RestartResult 重启命令结果
type RestartResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateConfigParams 更新配置命令参数
type UpdateConfigParams struct {
	Config map[string]interface{} `json:"config"`
}

// UpdateConfigResult 更新配置命令结果
type UpdateConfigResult struct {
	Success       bool   `json:"success"`
	UpdatedFields int    `json:"updated_fields"`
	Message       string `json:"message"`
}

// GetStatusResult 获取状态命令结果
type GetStatusResult struct {
	Status  string `json:"status"`
	Uptime  int64  `json:"uptime_seconds"`
	Version string `json:"version"`
}

// ExecShellParams 执行Shell命令参数
type ExecShellParams struct {
	Command string `json:"command"`           // Shell命令
	Timeout int    `json:"timeout,omitempty"` // 超时时间（秒），默认30秒
}

// ExecShellResult 执行Shell命令结果
type ExecShellResult struct {
	Success  bool   `json:"success"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Message  string `json:"message"`
}

// Execute 执行命令
func (e *SimpleCommandExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
	switch commandType {
	case "restart":
		return e.executeRestart(payload)
	case "update_config":
		return e.executeUpdateConfig(payload)
	case "get_status":
		return e.executeGetStatus(payload)
	case "exec_shell":
		return e.executeShell(payload)
	default:
		return nil, fmt.Errorf("unknown command type: %s", commandType)
	}
}

// executeRestart 执行重启命令
func (e *SimpleCommandExecutor) executeRestart(payload []byte) ([]byte, error) {
	var params RestartParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid restart params: %w", err)
	}

	// 模拟重启延迟
	if params.DelaySeconds > 0 {
		time.Sleep(time.Duration(params.DelaySeconds) * time.Second)
	}

	result := RestartResult{
		Success: true,
		Message: fmt.Sprintf("Restarted successfully (delay: %d seconds)", params.DelaySeconds),
	}

	return json.Marshal(result)
}

// executeUpdateConfig 执行更新配置命令
func (e *SimpleCommandExecutor) executeUpdateConfig(payload []byte) ([]byte, error) {
	var params UpdateConfigParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid update_config params: %w", err)
	}

	// 模拟更新配置
	updatedFields := len(params.Config)

	result := UpdateConfigResult{
		Success:       true,
		UpdatedFields: updatedFields,
		Message:       fmt.Sprintf("Updated %d configuration fields", updatedFields),
	}

	return json.Marshal(result)
}

// executeGetStatus 执行获取状态命令
func (e *SimpleCommandExecutor) executeGetStatus(payload []byte) ([]byte, error) {
	// 返回客户端状态
	result := GetStatusResult{
		Status:  "running",
		Uptime:  3600, // 假设运行了1小时
		Version: "1.0.0",
	}

	return json.Marshal(result)
}

// executeShell 执行Shell命令
func (e *SimpleCommandExecutor) executeShell(payload []byte) ([]byte, error) {
	var params ExecShellParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid exec_shell params: %w", err)
	}

	// 验证命令
	if strings.TrimSpace(params.Command) == "" {
		return nil, fmt.Errorf("command is empty")
	}

	// 设置默认超时
	timeout := 30
	if params.Timeout > 0 {
		timeout = params.Timeout
	}
	// 限制最大超时为5分钟
	if timeout > 300 {
		timeout = 300
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 执行Shell命令（使用 sh -c 以支持管道和复杂命令）
	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()

	// 限制输出大小（最大10KB）
	const maxOutputSize = 10 * 1024
	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	if len(stdoutStr) > maxOutputSize {
		stdoutStr = stdoutStr[:maxOutputSize] + "... (truncated)"
	}
	if len(stderrStr) > maxOutputSize {
		stderrStr = stderrStr[:maxOutputSize] + "... (truncated)"
	}

	// 构建结果
	result := ExecShellResult{
		Success:  err == nil,
		ExitCode: 0,
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
		Message:  "Command executed successfully",
	}

	// 处理错误
	if err != nil {
		result.Message = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.Message = fmt.Sprintf("Command timed out after %d seconds", timeout)
			result.ExitCode = -1
		} else {
			result.ExitCode = -1
		}
	}

	return json.Marshal(result)
}
