package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
)

// 发布执行配置
const (
	releaseDefaultTimeout     = 300  // 5分钟
	releaseMaxTimeout         = 3600 // 1小时
	releaseMaxOutputSize      = 1024 * 1024 // 1MB
	releaseDefaultInterpreter = "/bin/bash"
)

// ReleaseExecute 执行发布任务
// 命令类型: release.execute
// 用法: r.Register(command.CmdReleaseExecute, handlers.ReleaseExecute)
func ReleaseExecute(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ReleaseExecuteParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	startedAt := time.Now()

	// 如果脚本为空，直接返回成功
	if strings.TrimSpace(params.Script) == "" {
		return json.Marshal(command.ReleaseExecuteResult{
			Success:    true,
			ReleaseID:  params.ReleaseID,
			TargetID:   params.TargetID,
			Operation:  string(params.Operation),
			ExitCode:   0,
			Output:     "script is empty, skipped",
			StartedAt:  startedAt.Format(time.RFC3339),
			FinishedAt: time.Now().Format(time.RFC3339),
			Duration:   time.Since(startedAt).Milliseconds(),
		})
	}

	// 设置超时
	timeout := releaseDefaultTimeout
	if params.Timeout > 0 {
		timeout = params.Timeout
	}
	if timeout > releaseMaxTimeout {
		timeout = releaseMaxTimeout
	}

	// 创建带超时的上下文
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 确定工作目录：如果为空则使用程序执行目录
	workDir := params.WorkDir
	if workDir == "" {
		// 获取程序执行目录
		execPath, err := os.Executable()
		if err == nil {
			workDir = filepath.Dir(execPath)
		} else {
			// 回退到当前工作目录
			workDir, _ = os.Getwd()
		}
	}

	// 确定临时目录：使用工作目录下的 tmp 子目录
	tmpDir := filepath.Join(workDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		// 如果创建失败，使用系统临时目录
		tmpDir = ""
	}

	// 创建临时脚本文件
	tmpFile, err := os.CreateTemp(tmpDir, "release-script-*.sh")
	if err != nil {
		return json.Marshal(command.ReleaseExecuteResult{
			Success:    false,
			ReleaseID:  params.ReleaseID,
			TargetID:   params.TargetID,
			Operation:  string(params.Operation),
			ExitCode:   -1,
			Error:      fmt.Sprintf("create temp file: %v", err),
			StartedAt:  startedAt.Format(time.RFC3339),
			FinishedAt: time.Now().Format(time.RFC3339),
			Duration:   time.Since(startedAt).Milliseconds(),
		})
	}
	defer os.Remove(tmpFile.Name())

	// 写入脚本内容
	if _, err := tmpFile.WriteString(params.Script); err != nil {
		return json.Marshal(command.ReleaseExecuteResult{
			Success:    false,
			ReleaseID:  params.ReleaseID,
			TargetID:   params.TargetID,
			Operation:  string(params.Operation),
			ExitCode:   -1,
			Error:      fmt.Sprintf("write script: %v", err),
			StartedAt:  startedAt.Format(time.RFC3339),
			FinishedAt: time.Now().Format(time.RFC3339),
			Duration:   time.Since(startedAt).Milliseconds(),
		})
	}
	tmpFile.Close()

	// 设置执行权限
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return json.Marshal(command.ReleaseExecuteResult{
			Success:    false,
			ReleaseID:  params.ReleaseID,
			TargetID:   params.TargetID,
			Operation:  string(params.Operation),
			ExitCode:   -1,
			Error:      fmt.Sprintf("chmod: %v", err),
			StartedAt:  startedAt.Format(time.RFC3339),
			FinishedAt: time.Now().Format(time.RFC3339),
			Duration:   time.Since(startedAt).Milliseconds(),
		})
	}

	// 获取解释器
	interpreter := releaseDefaultInterpreter
	if params.Interpreter != "" {
		interpreter = params.Interpreter
	}

	// 创建命令
	cmd := exec.CommandContext(execCtx, interpreter, tmpFile.Name())

	// 进程脱离设置：使脚本启动的进程独立于Client运行
	if params.DetachProcess {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // 创建新会话，脱离父进程控制
		}
	}

	// 设置工作目录
	// 确保目录存在
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return json.Marshal(command.ReleaseExecuteResult{
			Success:    false,
			ReleaseID:  params.ReleaseID,
			TargetID:   params.TargetID,
			Operation:  string(params.Operation),
			ExitCode:   -1,
			Error:      fmt.Sprintf("create work dir: %v", err),
			StartedAt:  startedAt.Format(time.RFC3339),
			FinishedAt: time.Now().Format(time.RFC3339),
			Duration:   time.Since(startedAt).Milliseconds(),
		})
	}
	cmd.Dir = workDir

	// 设置环境变量
	cmd.Env = os.Environ()
	if params.Environment != nil {
		for k, v := range params.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	// 添加发布相关环境变量
	cmd.Env = append(cmd.Env, fmt.Sprintf("RELEASE_ID=%s", params.ReleaseID))
	cmd.Env = append(cmd.Env, fmt.Sprintf("TARGET_ID=%s", params.TargetID))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RELEASE_OPERATION=%s", params.Operation))
	cmd.Env = append(cmd.Env, fmt.Sprintf("RELEASE_VERSION=%s", params.Version))

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, &limitedWriter{max: releaseMaxOutputSize})
	cmd.Stderr = io.MultiWriter(&stderr, &limitedWriter{max: releaseMaxOutputSize})

	// 执行
	err = cmd.Run()
	finishedAt := time.Now()

	result := command.ReleaseExecuteResult{
		Success:    err == nil,
		ReleaseID:  params.ReleaseID,
		TargetID:   params.TargetID,
		Operation:  string(params.Operation),
		ExitCode:   0,
		Output:     truncateOutput(stdout.String(), releaseMaxOutputSize),
		StartedAt:  startedAt.Format(time.RFC3339),
		FinishedAt: finishedAt.Format(time.RFC3339),
		Duration:   finishedAt.Sub(startedAt).Milliseconds(),
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Sprintf("script timeout after %d seconds", timeout)
			result.ExitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = truncateOutput(stderr.String(), releaseMaxOutputSize)
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
	}

	return json.Marshal(result)
}

// ReleaseCheck 检查安装状态
// 命令类型: release.check
// 用法: r.Register(command.CmdReleaseCheck, handlers.ReleaseCheck)
func ReleaseCheck(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ReleaseCheckParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	result := command.ReleaseCheckResult{}

	if params.WorkDir == "" {
		result.Error = "work_dir is required"
		return json.Marshal(result)
	}

	// 检查目录是否存在
	info, err := os.Stat(params.WorkDir)
	if os.IsNotExist(err) {
		result.Installed = false
		return json.Marshal(result)
	}
	if err != nil {
		result.Error = fmt.Sprintf("stat work dir: %v", err)
		return json.Marshal(result)
	}

	if !info.IsDir() {
		result.Error = "work_dir is not a directory"
		return json.Marshal(result)
	}

	result.Installed = true
	result.InstallPath = params.WorkDir

	// 检查版本文件
	versionFile := filepath.Join(params.WorkDir, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err == nil {
		result.Version = strings.TrimSpace(string(data))
	}

	// 检查安装时间
	installTimeFile := filepath.Join(params.WorkDir, ".installed_at")
	if data, err := os.ReadFile(installTimeFile); err == nil {
		result.InstalledAt = strings.TrimSpace(string(data))
	} else {
		// 使用目录修改时间作为安装时间
		result.InstalledAt = info.ModTime().Format(time.RFC3339)
	}

	// 检查最后更新时间
	updateTimeFile := filepath.Join(params.WorkDir, ".updated_at")
	if data, err := os.ReadFile(updateTimeFile); err == nil {
		result.LastUpdatedAt = strings.TrimSpace(string(data))
	}

	return json.Marshal(result)
}

// limitedWriter 限制写入大小
type limitedWriter struct {
	max     int
	written int
}

func (w *limitedWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.max {
		return len(p), nil // 丢弃超出部分
	}
	remaining := w.max - w.written
	if len(p) > remaining {
		w.written += remaining
		return len(p), nil
	}
	w.written += len(p)
	return len(p), nil
}
