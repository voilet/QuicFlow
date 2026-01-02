package process

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MatchRule 进程匹配规则
type MatchRule struct {
	Type    string // name, cmdline, pidfile, port
	Pattern string // 匹配模式（支持正则）
	Name    string // 显示名称
}

// Info 进程信息
type Info struct {
	PID        int       `json:"pid"`
	Name       string    `json:"name"`
	Cmdline    string    `json:"cmdline"`
	StartTime  time.Time `json:"start_time"`
	Status     string    `json:"status"`
	CPUPercent float64   `json:"cpu_percent"`
	MemoryMB   float64   `json:"memory_mb"`
	MemoryPct  float64   `json:"memory_pct"`
	MatchedBy  string    `json:"matched_by"`
}

// Collector 进程采集器
type Collector struct {
	rules []MatchRule
}

// NewCollector 创建进程采集器
func NewCollector(rules []MatchRule) *Collector {
	return &Collector{rules: rules}
}

// Collect 采集匹配规则的进程信息
func (c *Collector) Collect() ([]Info, error) {
	var result []Info

	for _, rule := range c.rules {
		var infos []Info
		var err error

		switch rule.Type {
		case "name":
			infos, err = c.collectByName(rule)
		case "cmdline":
			infos, err = c.collectByCmdline(rule)
		case "pidfile":
			infos, err = c.collectByPidfile(rule)
		case "port":
			infos, err = c.collectByPort(rule)
		default:
			continue
		}

		if err != nil {
			continue
		}

		for i := range infos {
			infos[i].MatchedBy = rule.Name
		}
		result = append(result, infos...)
	}

	// 去重（按PID）
	seen := make(map[int]bool)
	unique := make([]Info, 0)
	for _, info := range result {
		if !seen[info.PID] {
			seen[info.PID] = true
			unique = append(unique, info)
		}
	}

	return unique, nil
}

// collectByName 按进程名匹配
func (c *Collector) collectByName(rule MatchRule) ([]Info, error) {
	pids, err := c.getAllPids()
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		// 如果不是有效正则，使用精确匹配
		re = nil
	}

	var result []Info
	for _, pid := range pids {
		name, err := c.getProcessName(pid)
		if err != nil {
			continue
		}

		matched := false
		if re != nil {
			matched = re.MatchString(name)
		} else {
			matched = name == rule.Pattern
		}

		if matched {
			info, err := c.getProcessInfo(pid)
			if err != nil {
				continue
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// collectByCmdline 按命令行匹配
func (c *Collector) collectByCmdline(rule MatchRule) ([]Info, error) {
	pids, err := c.getAllPids()
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		re = nil
	}

	var result []Info
	for _, pid := range pids {
		cmdline, err := c.getProcessCmdline(pid)
		if err != nil {
			continue
		}

		matched := false
		if re != nil {
			matched = re.MatchString(cmdline)
		} else {
			matched = strings.Contains(cmdline, rule.Pattern)
		}

		if matched {
			info, err := c.getProcessInfo(pid)
			if err != nil {
				continue
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// collectByPidfile 按PID文件匹配
func (c *Collector) collectByPidfile(rule MatchRule) ([]Info, error) {
	data, err := os.ReadFile(rule.Pattern)
	if err != nil {
		return nil, fmt.Errorf("read pidfile: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, fmt.Errorf("parse pid: %w", err)
	}

	info, err := c.getProcessInfo(pid)
	if err != nil {
		return nil, err
	}

	return []Info{info}, nil
}

// collectByPort 按监听端口匹配
func (c *Collector) collectByPort(rule MatchRule) ([]Info, error) {
	port, err := strconv.Atoi(rule.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	pids, err := c.getPidsByPort(port)
	if err != nil {
		return nil, err
	}

	var result []Info
	for _, pid := range pids {
		info, err := c.getProcessInfo(pid)
		if err != nil {
			continue
		}
		result = append(result, info)
	}

	return result, nil
}

// getAllPids 获取所有进程PID
func (c *Collector) getAllPids() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}

	return pids, nil
}

// getProcessName 获取进程名
func (c *Collector) getProcessName(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// getProcessCmdline 获取进程命令行
func (c *Collector) getProcessCmdline(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	// 替换空字符为空格
	cmdline := strings.ReplaceAll(string(data), "\x00", " ")
	return strings.TrimSpace(cmdline), nil
}

// getProcessInfo 获取进程详细信息
func (c *Collector) getProcessInfo(pid int) (Info, error) {
	info := Info{PID: pid}

	// 获取进程名
	name, err := c.getProcessName(pid)
	if err != nil {
		return info, err
	}
	info.Name = name

	// 获取命令行
	cmdline, err := c.getProcessCmdline(pid)
	if err == nil {
		info.Cmdline = cmdline
	}

	// 获取状态
	status, startTime, err := c.getProcessStatus(pid)
	if err == nil {
		info.Status = status
		info.StartTime = startTime
	}

	// 获取内存信息
	memMB, memPct, err := c.getProcessMemory(pid)
	if err == nil {
		info.MemoryMB = memMB
		info.MemoryPct = memPct
	}

	// 获取CPU使用率（简化版，需要两次采样才准确）
	info.CPUPercent = 0

	return info, nil
}

// getProcessStatus 获取进程状态和启动时间
func (c *Collector) getProcessStatus(pid int) (string, time.Time, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", time.Time{}, err
	}

	// 解析 /proc/[pid]/stat
	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return "", time.Time{}, fmt.Errorf("invalid stat format")
	}

	// 状态字段 (index 2)
	status := "running"
	switch fields[2] {
	case "S":
		status = "sleeping"
	case "R":
		status = "running"
	case "Z":
		status = "zombie"
	case "D":
		status = "disk_sleep"
	case "T":
		status = "stopped"
	}

	// 启动时间 (index 21) - 以 jiffies 为单位
	startTicks, _ := strconv.ParseInt(fields[21], 10, 64)
	bootTime := c.getBootTime()
	startTime := bootTime.Add(time.Duration(startTicks) * time.Second / 100)

	return status, startTime, nil
}

// getBootTime 获取系统启动时间
func (c *Collector) getBootTime() time.Time {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				btime, _ := strconv.ParseInt(parts[1], 10, 64)
				return time.Unix(btime, 0)
			}
		}
	}

	return time.Time{}
}

// getProcessMemory 获取进程内存使用
func (c *Collector) getProcessMemory(pid int) (float64, float64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("invalid statm format")
	}

	// RSS (Resident Set Size) in pages
	rssPages, _ := strconv.ParseInt(fields[1], 10, 64)
	pageSize := int64(os.Getpagesize())
	memBytes := rssPages * pageSize
	memMB := float64(memBytes) / 1024 / 1024

	// 获取总内存
	totalMem := c.getTotalMemory()
	memPct := 0.0
	if totalMem > 0 {
		memPct = float64(memBytes) / float64(totalMem) * 100
	}

	return memMB, memPct, nil
}

// getTotalMemory 获取系统总内存
func (c *Collector) getTotalMemory() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				memKB, _ := strconv.ParseInt(parts[1], 10, 64)
				return memKB * 1024
			}
		}
	}

	return 0
}

// getPidsByPort 获取监听指定端口的进程PID
func (c *Collector) getPidsByPort(port int) ([]int, error) {
	var pids []int

	// 检查 TCP
	tcpPids, _ := c.getPidsFromNetFile("/proc/net/tcp", port)
	pids = append(pids, tcpPids...)

	// 检查 TCP6
	tcp6Pids, _ := c.getPidsFromNetFile("/proc/net/tcp6", port)
	pids = append(pids, tcp6Pids...)

	return pids, nil
}

// getPidsFromNetFile 从 /proc/net/tcp 或 tcp6 获取监听指定端口的 inode
func (c *Collector) getPidsFromNetFile(netFile string, port int) ([]int, error) {
	data, err := os.ReadFile(netFile)
	if err != nil {
		return nil, err
	}

	portHex := fmt.Sprintf("%04X", port)
	var inodes []string

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		// local_address 格式: IP:PORT (hex)
		localAddr := fields[1]
		parts := strings.Split(localAddr, ":")
		if len(parts) != 2 {
			continue
		}

		if parts[1] == portHex {
			// 状态 0A = LISTEN
			if fields[3] == "0A" {
				inodes = append(inodes, fields[9])
			}
		}
	}

	// 根据 inode 查找 PID
	return c.getPidsByInodes(inodes)
}

// getPidsByInodes 根据 socket inode 查找 PID
func (c *Collector) getPidsByInodes(inodes []string) ([]int, error) {
	if len(inodes) == 0 {
		return nil, nil
	}

	inodeSet := make(map[string]bool)
	for _, inode := range inodes {
		inodeSet[inode] = true
	}

	pids, _ := c.getAllPids()
	var result []int

	for _, pid := range pids {
		fdPath := fmt.Sprintf("/proc/%d/fd", pid)
		entries, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			link, err := os.Readlink(filepath.Join(fdPath, entry.Name()))
			if err != nil {
				continue
			}

			// socket:[inode] 格式
			if strings.HasPrefix(link, "socket:[") {
				inode := strings.TrimPrefix(strings.TrimSuffix(link, "]"), "socket:[")
				if inodeSet[inode] {
					result = append(result, pid)
					break
				}
			}
		}
	}

	return result, nil
}
