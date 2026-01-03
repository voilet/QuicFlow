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
	"time"

	"github.com/voilet/quic-flow/pkg/command"
)

// Git 部署配置
const (
	gitDefaultCloneTimeout  = 300  // 5分钟
	gitDefaultScriptTimeout = 300  // 5分钟
	gitMaxOutputSize        = 1024 * 1024 // 1MB
)

// GitPullDeploy 执行 Git 拉取部署
// 命令类型: gitpull.deploy
// 用法: r.Register(command.CmdGitPullDeploy, handlers.GitPullDeploy)
func GitPullDeploy(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.GitPullDeployParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	startedAt := time.Now()

	result := command.GitPullDeployResult{
		ReleaseID:  params.ReleaseID,
		TargetID:   params.TargetID,
		Operation:  string(params.Operation),
		StartedAt:  startedAt.Format(time.RFC3339),
	}

	// 验证必须参数
	if params.RepoURL == "" {
		result.Error = "repo_url is required"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		result.Duration = time.Since(startedAt).Milliseconds()
		return json.Marshal(result)
	}

	// 确定工作目录：如果为空则使用程序执行目录下的 tmp
	workDir := params.WorkDir
	if workDir == "" {
		// 获取程序执行目录
		execPath, err := os.Executable()
		if err == nil {
			workDir = filepath.Join(filepath.Dir(execPath), "tmp")
		} else {
			// 回退到当前工作目录下的 tmp
			cwd, _ := os.Getwd()
			workDir = filepath.Join(cwd, "tmp")
		}
		// 更新 params.WorkDir 以便后续函数使用
		params.WorkDir = workDir
	}

	// 设置超时
	cloneTimeout := gitDefaultCloneTimeout
	if params.CloneTimeout > 0 {
		cloneTimeout = params.CloneTimeout
	}
	scriptTimeout := gitDefaultScriptTimeout
	if params.ScriptTimeout > 0 {
		scriptTimeout = params.ScriptTimeout
	}

	// 确保工作目录的父目录存在
	parentDir := filepath.Dir(workDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		result.Error = fmt.Sprintf("create parent dir: %v", err)
		result.FinishedAt = time.Now().Format(time.RFC3339)
		result.Duration = time.Since(startedAt).Milliseconds()
		return json.Marshal(result)
	}

	// 执行部署前脚本（PreScript）
	if strings.TrimSpace(params.PreScript) != "" {
		preOutput, err := executeScript(ctx, params.PreScript, workDir, params.Environment, params.Interpreter, scriptTimeout)
		if err != nil {
			result.Error = fmt.Sprintf("pre_script failed: %v", err)
			result.ScriptOutput = preOutput
			result.FinishedAt = time.Now().Format(time.RFC3339)
			result.Duration = time.Since(startedAt).Milliseconds()
			return json.Marshal(result)
		}
		result.ScriptOutput = "=== PreScript Output ===\n" + preOutput + "\n"
	}

	// 备份（如果需要）
	if params.BackupBefore && dirExists(workDir) {
		backupPath, err := backupDirectory(workDir, params.BackupDir)
		if err != nil {
			result.Error = fmt.Sprintf("backup failed: %v", err)
			result.FinishedAt = time.Now().Format(time.RFC3339)
			result.Duration = time.Since(startedAt).Milliseconds()
			return json.Marshal(result)
		}
		result.BackupPath = backupPath
		result.BackedUpBefore = true
	}

	// 清理（如果需要）
	if params.CleanBefore && dirExists(workDir) {
		if err := os.RemoveAll(workDir); err != nil {
			result.Error = fmt.Sprintf("clean failed: %v", err)
			result.FinishedAt = time.Now().Format(time.RFC3339)
			result.Duration = time.Since(startedAt).Milliseconds()
			return json.Marshal(result)
		}
		result.CleanedBefore = true
	}

	// 执行 Git 操作
	gitOutput, commit, branch, err := executeGitOperation(ctx, params, cloneTimeout)
	result.GitOutput = gitOutput
	result.Commit = commit
	result.Branch = branch

	if err != nil {
		result.Error = fmt.Sprintf("git operation failed: %v", err)
		result.FinishedAt = time.Now().Format(time.RFC3339)
		result.Duration = time.Since(startedAt).Milliseconds()
		return json.Marshal(result)
	}

	// 执行部署后脚本（PostScript）
	if strings.TrimSpace(params.PostScript) != "" {
		postOutput, err := executeScript(ctx, params.PostScript, params.WorkDir, params.Environment, params.Interpreter, scriptTimeout)
		if err != nil {
			result.Error = fmt.Sprintf("post_script failed: %v", err)
			result.ScriptOutput += "=== PostScript Output ===\n" + postOutput
			result.FinishedAt = time.Now().Format(time.RFC3339)
			result.Duration = time.Since(startedAt).Milliseconds()
			return json.Marshal(result)
		}
		result.ScriptOutput += "=== PostScript Output ===\n" + postOutput
	}

	result.Success = true
	result.FinishedAt = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startedAt).Milliseconds()

	return json.Marshal(result)
}

// executeGitOperation 执行 Git 操作（clone 或 pull）
func executeGitOperation(ctx context.Context, params command.GitPullDeployParams, timeout int) (output, commit, branch string, err error) {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var outputBuf bytes.Buffer
	writer := io.MultiWriter(&outputBuf, &limitedWriter{max: gitMaxOutputSize})

	// 设置 Git 认证
	env := os.Environ()
	if params.AuthType == "token" && params.Token != "" {
		// 对于 token 认证，修改 URL
		// 暂时跳过，使用 git credential
	}
	for k, v := range params.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 判断是 clone 还是 pull
	isClone := !dirExists(filepath.Join(params.WorkDir, ".git"))

	if isClone {
		// Clone
		args := []string{"clone"}

		// 设置深度
		if params.Depth > 0 {
			args = append(args, "--depth", fmt.Sprintf("%d", params.Depth))
		}

		// 设置分支
		if params.Branch != "" {
			args = append(args, "-b", params.Branch)
		} else if params.Tag != "" {
			args = append(args, "-b", params.Tag)
		}

		// 添加 URL 和目标目录
		repoURL := params.RepoURL
		if params.AuthType == "token" && params.Token != "" {
			// 将 token 嵌入 URL（适用于 HTTPS）
			repoURL = injectTokenInURL(params.RepoURL, params.Token, params.Username)
		}
		args = append(args, repoURL, params.WorkDir)

		// 记录执行的命令（隐藏敏感信息）
		logArgs := make([]string, len(args))
		copy(logArgs, args)
		// 隐藏 URL 中的 token
		for i, arg := range logArgs {
			if strings.Contains(arg, "@") && (strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://")) {
				logArgs[i] = hideCredentialsInURL(arg)
			}
		}
		cmdStr := fmt.Sprintf("git %s", strings.Join(logArgs, " "))
		fmt.Fprintf(writer, "[CMD] %s\n", cmdStr)
		fmt.Fprintf(writer, "[WorkDir] %s\n", params.WorkDir)

		cmd := exec.CommandContext(execCtx, "git", args...)
		cmd.Env = env
		cmd.Stdout = writer
		cmd.Stderr = writer

		if err := cmd.Run(); err != nil {
			errOutput := outputBuf.String()
			return errOutput, "", "", fmt.Errorf("git clone failed: %w\n%s", err, errOutput)
		}

		// 初始化子模块
		if params.Submodules {
			fmt.Fprintf(writer, "[CMD] git submodule update --init --recursive\n")
			subCmd := exec.CommandContext(execCtx, "git", "submodule", "update", "--init", "--recursive")
			subCmd.Dir = params.WorkDir
			subCmd.Env = env
			subCmd.Stdout = writer
			subCmd.Stderr = writer
			subCmd.Run() // 忽略错误
		}
	} else {
		// Pull
		fmt.Fprintf(writer, "[INFO] Repository exists, updating...\n")
		fmt.Fprintf(writer, "[WorkDir] %s\n", params.WorkDir)

		// 首先 fetch
		fmt.Fprintf(writer, "[CMD] git fetch --all --prune\n")
		fetchCmd := exec.CommandContext(execCtx, "git", "fetch", "--all", "--prune")
		fetchCmd.Dir = params.WorkDir
		fetchCmd.Env = env
		fetchCmd.Stdout = writer
		fetchCmd.Stderr = writer
		if err := fetchCmd.Run(); err != nil {
			errOutput := outputBuf.String()
			return errOutput, "", "", fmt.Errorf("git fetch failed: %w\n%s", err, errOutput)
		}

		// 切换到指定分支/tag/commit
		var checkoutRef string
		if params.Commit != "" {
			checkoutRef = params.Commit
		} else if params.Tag != "" {
			checkoutRef = params.Tag
		} else if params.Branch != "" {
			checkoutRef = params.Branch
		} else {
			checkoutRef = "origin/main" // 默认
		}

		fmt.Fprintf(writer, "[CMD] git checkout %s\n", checkoutRef)
		checkoutCmd := exec.CommandContext(execCtx, "git", "checkout", checkoutRef)
		checkoutCmd.Dir = params.WorkDir
		checkoutCmd.Env = env
		checkoutCmd.Stdout = writer
		checkoutCmd.Stderr = writer
		if err := checkoutCmd.Run(); err != nil {
			// 尝试 origin/branch
			if params.Branch != "" {
				originBranch := "origin/" + params.Branch
				fmt.Fprintf(writer, "[CMD] git checkout %s (retry with origin prefix)\n", originBranch)
				checkoutCmd2 := exec.CommandContext(execCtx, "git", "checkout", originBranch)
				checkoutCmd2.Dir = params.WorkDir
				checkoutCmd2.Env = env
				checkoutCmd2.Stdout = writer
				checkoutCmd2.Stderr = writer
				if err2 := checkoutCmd2.Run(); err2 != nil {
					errOutput := outputBuf.String()
					return errOutput, "", "", fmt.Errorf("git checkout failed: %w\n%s", err, errOutput)
				}
			} else {
				errOutput := outputBuf.String()
				return errOutput, "", "", fmt.Errorf("git checkout failed: %w\n%s", err, errOutput)
			}
		}

		// 如果是分支，执行 pull
		if params.Commit == "" && params.Tag == "" {
			fmt.Fprintf(writer, "[CMD] git pull --ff-only\n")
			pullCmd := exec.CommandContext(execCtx, "git", "pull", "--ff-only")
			pullCmd.Dir = params.WorkDir
			pullCmd.Env = env
			pullCmd.Stdout = writer
			pullCmd.Stderr = writer
			pullCmd.Run() // 忽略错误（可能是 detached HEAD）
		}

		// 更新子模块
		if params.Submodules {
			fmt.Fprintf(writer, "[CMD] git submodule update --init --recursive\n")
			subCmd := exec.CommandContext(execCtx, "git", "submodule", "update", "--init", "--recursive")
			subCmd.Dir = params.WorkDir
			subCmd.Env = env
			subCmd.Stdout = writer
			subCmd.Stderr = writer
			subCmd.Run() // 忽略错误
		}
	}

	// 获取当前 commit 和 branch
	commitCmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	commitCmd.Dir = params.WorkDir
	commitOutput, _ := commitCmd.Output()
	commit = strings.TrimSpace(string(commitOutput))

	branchCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = params.WorkDir
	branchOutput, _ := branchCmd.Output()
	branch = strings.TrimSpace(string(branchOutput))

	fmt.Fprintf(writer, "[SUCCESS] Commit: %s, Branch: %s\n", commit, branch)

	return outputBuf.String(), commit, branch, nil
}

// hideCredentialsInURL 隐藏 URL 中的凭证信息
func hideCredentialsInURL(urlStr string) string {
	// https://user:token@github.com/... -> https://***@github.com/...
	if idx := strings.Index(urlStr, "@"); idx > 0 {
		prefix := urlStr[:strings.Index(urlStr, "//")+2]
		suffix := urlStr[idx:]
		return prefix + "***" + suffix
	}
	return urlStr
}

// executeScript 执行脚本
func executeScript(ctx context.Context, script, workDir string, env map[string]string, interpreter string, timeout int) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", nil
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 确定临时目录
	tmpDir := ""
	if workDir != "" {
		tmpDir = filepath.Join(workDir, "tmp")
		os.MkdirAll(tmpDir, 0755)
	}

	// 创建临时脚本文件
	tmpFile, err := os.CreateTemp(tmpDir, "deploy-script-*.sh")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(script); err != nil {
		return "", fmt.Errorf("write script: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return "", fmt.Errorf("chmod: %w", err)
	}

	// 获取解释器
	if interpreter == "" {
		interpreter = "/bin/bash"
	}

	cmd := exec.CommandContext(execCtx, interpreter, tmpFile.Name())

	// 设置工作目录（如果存在）
	if dirExists(workDir) {
		cmd.Dir = workDir
	}

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("script timeout after %d seconds", timeout)
		}
		return output, fmt.Errorf("script failed: %w, stderr: %s", err, stderr.String())
	}

	return output, nil
}

// backupDirectory 备份目录
func backupDirectory(srcDir, backupDir string) (string, error) {
	if backupDir == "" {
		backupDir = filepath.Dir(srcDir)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.backup.%s", filepath.Base(srcDir), timestamp))

	// 使用 cp -a 进行备份
	cmd := exec.Command("cp", "-a", srcDir, backupPath)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return backupPath, nil
}

// dirExists 检查目录是否存在
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// injectTokenInURL 将 token 注入 HTTPS URL
func injectTokenInURL(repoURL, token, username string) string {
	// https://github.com/user/repo.git -> https://token@github.com/user/repo.git
	// 或 https://username:token@github.com/user/repo.git
	if !strings.HasPrefix(repoURL, "https://") {
		return repoURL
	}

	urlPart := strings.TrimPrefix(repoURL, "https://")
	if username != "" {
		return fmt.Sprintf("https://%s:%s@%s", username, token, urlPart)
	}
	return fmt.Sprintf("https://%s@%s", token, urlPart)
}

// GitVersions 获取 Git 仓库版本信息
// 命令类型: git.versions
// 用法: r.Register(command.CmdGitVersions, handlers.GitVersions)
func GitVersions(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.GitVersionsParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	result := command.GitVersionsResult{
		RepoURL: params.RepoURL,
	}

	// 如果有工作目录且存在 .git，从本地获取
	if params.WorkDir != "" && dirExists(filepath.Join(params.WorkDir, ".git")) {
		// 获取当前 commit
		commitCmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
		commitCmd.Dir = params.WorkDir
		if out, err := commitCmd.Output(); err == nil {
			result.CurrentCommit = strings.TrimSpace(string(out))
		}

		// 获取当前分支
		branchCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
		branchCmd.Dir = params.WorkDir
		if out, err := branchCmd.Output(); err == nil {
			result.CurrentBranch = strings.TrimSpace(string(out))
		}

		// Fetch 最新
		fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--all", "--tags")
		fetchCmd.Dir = params.WorkDir
		fetchCmd.Run()
	}

	// 获取 tags
	workDir := params.WorkDir
	if workDir == "" || !dirExists(filepath.Join(workDir, ".git")) {
		// 需要先 clone
		result.Error = "repository not cloned yet"
		return json.Marshal(result)
	}

	// 获取 tags
	maxTags := params.MaxTags
	if maxTags <= 0 {
		maxTags = 20
	}

	tagCmd := exec.CommandContext(ctx, "git", "tag", "-l", "--sort=-creatordate")
	tagCmd.Dir = workDir
	if out, err := tagCmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for i, line := range lines {
			if i >= maxTags || line == "" {
				break
			}
			tag := command.GitTag{Name: line}
			// 获取 tag 对应的 commit
			commitCmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", line)
			commitCmd.Dir = workDir
			if commitOut, err := commitCmd.Output(); err == nil {
				tag.Commit = strings.TrimSpace(string(commitOut))
			}
			// 获取 tag 对应的 commit message
			msgCmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%s", line)
			msgCmd.Dir = workDir
			if msgOut, err := msgCmd.Output(); err == nil {
				tag.Message = strings.TrimSpace(string(msgOut))
			}
			result.Tags = append(result.Tags, tag)
		}
	}

	// 获取分支
	if params.IncludeBranches {
		branchCmd := exec.CommandContext(ctx, "git", "branch", "-r")
		branchCmd.Dir = workDir
		if out, err := branchCmd.Output(); err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.Contains(line, "->") {
					continue
				}
				branch := command.GitBranch{
					Name:     strings.TrimPrefix(line, "origin/"),
					IsRemote: true,
				}
				// 获取分支的最新 commit
				commitCmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", line)
				commitCmd.Dir = workDir
				if commitOut, err := commitCmd.Output(); err == nil {
					branch.Commit = strings.TrimSpace(string(commitOut))
				}
				result.Branches = append(result.Branches, branch)
			}
		}
	}

	// 获取最近的 commits
	maxCommits := params.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 10
	}

	logCmd := exec.CommandContext(ctx, "git", "log", fmt.Sprintf("-%d", maxCommits), "--pretty=format:%h|%H|%an|%ae|%s|%ci")
	logCmd.Dir = workDir
	if out, err := logCmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 6)
			if len(parts) >= 6 {
				result.RecentCommits = append(result.RecentCommits, command.GitCommit{
					Hash:      parts[0],
					FullHash:  parts[1],
					Author:    parts[2],
					Email:     parts[3],
					Message:   parts[4],
					CreatedAt: parts[5],
				})
			}
		}
	}

	// 获取默认分支
	defaultCmd := exec.CommandContext(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	defaultCmd.Dir = workDir
	if out, err := defaultCmd.Output(); err == nil {
		result.DefaultBranch = strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/")
	}

	result.Success = true
	return json.Marshal(result)
}
