package version

import (
	"fmt"
	"runtime"
)

// 编译时通过 -ldflags 注入的变量
var (
	// Version 版本号
	Version = "dev"
	// GitCommit Git 提交哈希
	GitCommit = "unknown"
	// BuildTime 编译时间
	BuildTime = "unknown"
)

// Info 版本信息结构
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get 获取版本信息
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String 返回版本字符串
func String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildTime)
}

// Short 返回简短版本字符串
func Short() string {
	return Version
}

// Print 打印完整版本信息
func Print(appName string) {
	info := Get()
	fmt.Printf("%s version %s\n", appName, info.Version)
	fmt.Printf("  Git commit:  %s\n", info.GitCommit)
	fmt.Printf("  Build time:  %s\n", info.BuildTime)
	fmt.Printf("  Go version:  %s\n", info.GoVersion)
	fmt.Printf("  Platform:    %s\n", info.Platform)
}
