package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
)

// DiskBenchmark 执行磁盘 IO 读写测试
// 命令类型: disk.benchmark
// 用法: r.Register(command.CmdDiskBenchmark, handlers.DiskBenchmark)
func DiskBenchmark(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	// 解析参数
	var params command.DiskBenchmarkParams
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	// 设置默认值
	if params.TestSize == "" {
		params.TestSize = "1G"
	}
	if params.Runtime == 0 {
		params.Runtime = 60 // 默认60秒
	}
	if params.BlockSize == "" {
		params.BlockSize = "4k"
	}
	if params.NumJobs == 0 {
		params.NumJobs = 1
	}
	if params.IODepth == 0 {
		params.IODepth = 128 // 默认128队列深度
	}

	response := &command.DiskBenchmarkResponse{
		TestedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 检查 fio 是否安装
	if !checkFioInstalled() {
		response.Success = false
		response.Message = "fio is not installed. Please install with: yum install fio"
		return json.Marshal(response)
	}

	// 获取要测试的磁盘列表
	disks, err := getTestableDisks(ctx, params.Device)
	if err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("failed to get testable disks: %v", err)
		return json.Marshal(response)
	}

	if len(disks) == 0 {
		response.Success = false
		response.Message = "no testable disks found (all disks are system disks or specified device not found)"
		return json.Marshal(response)
	}

	// 对每个磁盘执行测试
	if params.Concurrent && len(disks) > 1 {
		// 并发测试多块磁盘
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make([]*command.DiskBenchmarkResult, 0, len(disks))

		for _, disk := range disks {
			wg.Add(1)
			go func(d *testDiskInfo) {
				defer wg.Done()
				result := runDiskBenchmark(ctx, d, &params)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}(disk)
		}

		wg.Wait()
		response.Results = results
	} else {
		// 顺序测试磁盘
		for _, disk := range disks {
			result := runDiskBenchmark(ctx, disk, &params)
			response.Results = append(response.Results, result)
		}
	}

	response.Success = true
	response.TotalDisks = len(response.Results)
	testMode := "sequential"
	if params.Concurrent && len(disks) > 1 {
		testMode = "concurrent"
	}
	response.Message = fmt.Sprintf("completed benchmark for %d disk(s) in %s mode", response.TotalDisks, testMode)

	return json.Marshal(response)
}

// testDiskInfo 用于测试的磁盘信息
type testDiskInfo struct {
	device    string
	model     string
	kind      string
	mountPath string // 用于测试的挂载路径
}

// checkFioInstalled 检查 fio 是否已安装
func checkFioInstalled() bool {
	_, err := exec.LookPath("fio")
	return err == nil
}

// getTestableDisks 获取可测试的磁盘列表（排除系统盘）
func getTestableDisks(ctx context.Context, specificDevice string) ([]*testDiskInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("disk benchmark only supported on Linux")
	}

	var result []*testDiskInfo

	// 获取硬件信息以确定哪些是系统盘
	hwInfo, err := GetHardwareInfo(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get hardware info: %w", err)
	}

	var hwResult command.HardwareInfoResult
	if err := json.Unmarshal(hwInfo, &hwResult); err != nil {
		return nil, fmt.Errorf("failed to parse hardware info: %w", err)
	}

	// 构建非系统盘列表
	for _, disk := range hwResult.Disks {
		// 跳过系统盘
		if disk.IsSystemDisk {
			continue
		}

		// 如果指定了特定设备，只测试该设备
		if specificDevice != "" && disk.Device != specificDevice {
			continue
		}

		info := &testDiskInfo{
			device: disk.Device,
			model:  disk.Model,
			kind:   disk.Kind,
		}

		// 查找可用的测试路径
		info.mountPath = findTestPath(disk)

		if info.mountPath != "" {
			result = append(result, info)
		}
	}

	return result, nil
}

// findTestPath 为磁盘找到可用的测试路径
func findTestPath(disk command.DiskInfo) string {
	// 优先使用已挂载的路径
	if len(disk.MountUsages) > 0 {
		for _, usage := range disk.MountUsages {
			// 检查路径是否可写
			testFile := filepath.Join(usage.MountPoint, ".fio_test_check")
			f, err := os.Create(testFile)
			if err == nil {
				f.Close()
				os.Remove(testFile)
				return usage.MountPoint
			}
		}
	}

	// 尝试使用裸设备（需要 root 权限）
	devicePath := "/dev/" + disk.Device
	if _, err := os.Stat(devicePath); err == nil {
		return devicePath
	}

	return ""
}

// runDiskBenchmark 对单个磁盘运行基准测试
func runDiskBenchmark(ctx context.Context, disk *testDiskInfo, params *command.DiskBenchmarkParams) *command.DiskBenchmarkResult {
	result := &command.DiskBenchmarkResult{
		Device:   disk.device,
		Model:    disk.model,
		Kind:     disk.kind,
		TestPath: disk.mountPath,
		TestSize: params.TestSize,
	}

	startTime := time.Now()

	// 确定测试文件路径
	var testFile string
	var isRawDevice bool

	if strings.HasPrefix(disk.mountPath, "/dev/") {
		// 裸设备测试
		testFile = disk.mountPath
		isRawDevice = true
	} else {
		// 文件系统测试
		testFile = filepath.Join(disk.mountPath, fmt.Sprintf(".fio_benchmark_%s", disk.device))
		defer os.Remove(testFile)
	}

	var errors []string

	// 运行各项测试
	// 1. 顺序读
	seqRead := runFioTest(ctx, testFile, "read", "1M", params, isRawDevice)
	if seqRead != nil {
		if seqRead.err != "" {
			errors = append(errors, seqRead.err)
		} else {
			result.SeqReadIOPS = seqRead.iops
			result.SeqReadBWMBps = seqRead.bwMBps
			result.SeqReadLatencyUs = seqRead.latencyUs
		}
	}

	// 2. 顺序写
	seqWrite := runFioTest(ctx, testFile, "write", "1M", params, isRawDevice)
	if seqWrite != nil {
		if seqWrite.err != "" {
			errors = append(errors, seqWrite.err)
		} else {
			result.SeqWriteIOPS = seqWrite.iops
			result.SeqWriteBWMBps = seqWrite.bwMBps
			result.SeqWriteLatencyUs = seqWrite.latencyUs
		}
	}

	// 3. 随机读 4K
	randRead := runFioTest(ctx, testFile, "randread", params.BlockSize, params, isRawDevice)
	if randRead != nil {
		if randRead.err != "" {
			errors = append(errors, randRead.err)
		} else {
			result.RandReadIOPS = randRead.iops
			result.RandReadBWMBps = randRead.bwMBps
			result.RandReadLatencyUs = randRead.latencyUs
		}
	}

	// 4. 随机写 4K
	randWrite := runFioTest(ctx, testFile, "randwrite", params.BlockSize, params, isRawDevice)
	if randWrite != nil {
		if randWrite.err != "" {
			errors = append(errors, randWrite.err)
		} else {
			result.RandWriteIOPS = randWrite.iops
			result.RandWriteBWMBps = randWrite.bwMBps
			result.RandWriteLatencyUs = randWrite.latencyUs
		}
	}

	// 5. 混合随机读写 (70% 读 30% 写)
	mixed := runFioMixedTest(ctx, testFile, params, isRawDevice)
	if mixed != nil {
		if mixed.err != "" {
			errors = append(errors, mixed.err)
		} else {
			result.MixedIOPS = mixed.iops
			result.MixedBWMBps = mixed.bwMBps
			result.MixedLatencyUs = mixed.latencyUs
		}
	}

	result.Duration = int(time.Since(startTime).Seconds())

	// 收集所有错误
	if len(errors) > 0 {
		result.Error = strings.Join(errors, "; ")
	}

	return result
}

// fioResult FIO 测试结果
type fioResult struct {
	iops      float64
	bwMBps    float64
	latencyUs float64
	err       string // 错误信息
}

// runFioTest 运行单项 FIO 测试
func runFioTest(ctx context.Context, filename, rwMode, bs string, params *command.DiskBenchmarkParams, isRawDevice bool) *fioResult {
	// 构建 FIO 命令参数
	args := []string{
		fmt.Sprintf("-filename=%s", filename),
		"-direct=1",
		"-ioengine=libaio",
		fmt.Sprintf("-bs=%s", bs),
		fmt.Sprintf("-size=%s", params.TestSize),
		fmt.Sprintf("-numjobs=%d", params.NumJobs),
		fmt.Sprintf("-iodepth=%d", params.IODepth),
		fmt.Sprintf("-runtime=%ds", params.Runtime),
		"-thread",
		fmt.Sprintf("-rw=%s", rwMode),
		"-group_reporting",
		fmt.Sprintf("-name=%s", rwMode),
		"--output-format=json",
	}

	if isRawDevice {
		args = append(args, "--allow_file_create=0")
	}

	cmd := exec.CommandContext(ctx, "fio", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &fioResult{err: fmt.Sprintf("fio %s failed: %v, output: %s", rwMode, err, string(output))}
	}

	return parseFioOutput(output, rwMode)
}

// runFioMixedTest 运行混合读写测试
func runFioMixedTest(ctx context.Context, filename string, params *command.DiskBenchmarkParams, isRawDevice bool) *fioResult {
	args := []string{
		fmt.Sprintf("-filename=%s", filename),
		"-direct=1",
		"-ioengine=libaio",
		fmt.Sprintf("-bs=%s", params.BlockSize),
		fmt.Sprintf("-size=%s", params.TestSize),
		fmt.Sprintf("-numjobs=%d", params.NumJobs),
		fmt.Sprintf("-iodepth=%d", params.IODepth),
		fmt.Sprintf("-runtime=%ds", params.Runtime),
		"-thread",
		"-rw=randrw",
		"-rwmixread=70",
		"-group_reporting",
		"-name=randrw",
		"--output-format=json",
	}

	if isRawDevice {
		args = append(args, "--allow_file_create=0")
	}

	cmd := exec.CommandContext(ctx, "fio", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &fioResult{err: fmt.Sprintf("fio randrw failed: %v, output: %s", err, string(output))}
	}

	return parseFioOutput(output, "randrw")
}

// parseFioOutput 解析 FIO JSON 输出
func parseFioOutput(output []byte, rwMode string) *fioResult {
	var fioJSON struct {
		Jobs []struct {
			Read struct {
				Iops     float64 `json:"iops"`
				BwBytes  float64 `json:"bw_bytes"`
				LatNs    struct {
					Mean float64 `json:"mean"`
				} `json:"lat_ns"`
			} `json:"read"`
			Write struct {
				Iops     float64 `json:"iops"`
				BwBytes  float64 `json:"bw_bytes"`
				LatNs    struct {
					Mean float64 `json:"mean"`
				} `json:"lat_ns"`
			} `json:"write"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(output, &fioJSON); err != nil {
		return nil
	}

	if len(fioJSON.Jobs) == 0 {
		return nil
	}

	job := fioJSON.Jobs[0]
	result := &fioResult{}

	switch rwMode {
	case "read", "randread":
		result.iops = job.Read.Iops
		result.bwMBps = job.Read.BwBytes / 1024 / 1024
		result.latencyUs = job.Read.LatNs.Mean / 1000
	case "write", "randwrite":
		result.iops = job.Write.Iops
		result.bwMBps = job.Write.BwBytes / 1024 / 1024
		result.latencyUs = job.Write.LatNs.Mean / 1000
	case "randrw":
		// 混合模式，合计读写
		result.iops = job.Read.Iops + job.Write.Iops
		result.bwMBps = (job.Read.BwBytes + job.Write.BwBytes) / 1024 / 1024
		// 加权平均延迟
		if job.Read.Iops+job.Write.Iops > 0 {
			result.latencyUs = (job.Read.LatNs.Mean*job.Read.Iops + job.Write.LatNs.Mean*job.Write.Iops) / (job.Read.Iops + job.Write.Iops) / 1000
		}
	}

	return result
}

// RunLocalBenchmark 本地执行磁盘基准测试（供 CLI 使用）
func RunLocalBenchmark(device string, testSize string, runtime int, concurrent bool) (*command.DiskBenchmarkResponse, error) {
	params := command.DiskBenchmarkParams{
		Device:     device,
		TestSize:   testSize,
		Runtime:    runtime,
		BlockSize:  "4k",
		NumJobs:    1,
		IODepth:    128,
		Concurrent: concurrent,
	}

	payload, _ := json.Marshal(params)
	result, err := DiskBenchmark(context.Background(), payload)
	if err != nil {
		return nil, err
	}

	var response command.DiskBenchmarkResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// FormatBenchmarkResult 格式化测试结果为文本
func FormatBenchmarkResult(result *command.DiskBenchmarkResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("设备: %s (%s, %s)\n", result.Device, result.Model, result.Kind))
	sb.WriteString(fmt.Sprintf("测试路径: %s\n", result.TestPath))
	sb.WriteString(fmt.Sprintf("测试大小: %s\n", result.TestSize))
	sb.WriteString(fmt.Sprintf("测试耗时: %d 秒\n", result.Duration))

	// 显示错误信息
	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("\n【错误信息】\n  %s\n", result.Error))
	}

	sb.WriteString("\n")

	sb.WriteString("【顺序读 (1M block)】\n")
	sb.WriteString(fmt.Sprintf("  IOPS: %.0f\n", result.SeqReadIOPS))
	sb.WriteString(fmt.Sprintf("  带宽: %.2f MB/s\n", result.SeqReadBWMBps))
	sb.WriteString(fmt.Sprintf("  延迟: %.2f μs\n", result.SeqReadLatencyUs))
	sb.WriteString("\n")

	sb.WriteString("【顺序写 (1M block)】\n")
	sb.WriteString(fmt.Sprintf("  IOPS: %.0f\n", result.SeqWriteIOPS))
	sb.WriteString(fmt.Sprintf("  带宽: %.2f MB/s\n", result.SeqWriteBWMBps))
	sb.WriteString(fmt.Sprintf("  延迟: %.2f μs\n", result.SeqWriteLatencyUs))
	sb.WriteString("\n")

	sb.WriteString("【随机读 (4K block)】\n")
	sb.WriteString(fmt.Sprintf("  IOPS: %.0f\n", result.RandReadIOPS))
	sb.WriteString(fmt.Sprintf("  带宽: %.2f MB/s\n", result.RandReadBWMBps))
	sb.WriteString(fmt.Sprintf("  延迟: %.2f μs\n", result.RandReadLatencyUs))
	sb.WriteString("\n")

	sb.WriteString("【随机写 (4K block)】\n")
	sb.WriteString(fmt.Sprintf("  IOPS: %.0f\n", result.RandWriteIOPS))
	sb.WriteString(fmt.Sprintf("  带宽: %.2f MB/s\n", result.RandWriteBWMBps))
	sb.WriteString(fmt.Sprintf("  延迟: %.2f μs\n", result.RandWriteLatencyUs))
	sb.WriteString("\n")

	sb.WriteString("【混合随机读写 (70R/30W, 4K block)】\n")
	sb.WriteString(fmt.Sprintf("  IOPS: %.0f\n", result.MixedIOPS))
	sb.WriteString(fmt.Sprintf("  带宽: %.2f MB/s\n", result.MixedBWMBps))
	sb.WriteString(fmt.Sprintf("  延迟: %.2f μs\n", result.MixedLatencyUs))

	return sb.String()
}

// parseSize 解析大小字符串为字节数
func parseSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	if len(sizeStr) == 0 {
		return 0
	}

	multiplier := int64(1)
	suffix := sizeStr[len(sizeStr)-1]

	switch suffix {
	case 'K':
		multiplier = 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case 'M':
		multiplier = 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	}

	num, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0
	}

	return num * multiplier
}

// ============================================================================
// 简化的磁盘 IOPS 检测（使用单一 FIO 命令）
// ============================================================================

// DiskIOPS 执行简化的磁盘 IOPS 检测
// 使用命令: fio -iodepth=128 -numjobs=[cpu 核数] -bs=4k -time_based=1 -runtime=60s
// 只返回 Read IOPS 和 Write IOPS
// 命令类型: disk.iops
// 用法: r.Register(command.CmdDiskIOPS, handlers.DiskIOPS)
func DiskIOPS(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	// 解析参数
	var params command.DiskIOPSParams
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	// 设置默认值
	if params.Runtime == 0 {
		params.Runtime = 60 // 默认60秒
	}

	response := &command.DiskIOPSResponse{
		TestedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 检查 fio 是否安装
	if !checkFioInstalled() {
		response.Success = false
		response.Message = "fio is not installed. Please install with: yum install fio"
		return json.Marshal(response)
	}

	// 获取要测试的磁盘列表
	disks, err := getTestableDisks(ctx, params.Device)
	if err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("failed to get testable disks: %v", err)
		return json.Marshal(response)
	}

	if len(disks) == 0 {
		response.Success = false
		response.Message = "no testable disks found (all disks are system disks or specified device not found)"
		return json.Marshal(response)
	}

	// 获取 CPU 核心数
	numCPU := runtime.NumCPU()

	// 对每个磁盘执行测试
	for _, disk := range disks {
		result := runDiskIOPSTest(ctx, disk, params.Runtime, numCPU)
		response.Results = append(response.Results, result)
	}

	response.Success = true
	response.TotalDisks = len(response.Results)
	response.Message = fmt.Sprintf("completed IOPS test for %d disk(s)", response.TotalDisks)

	return json.Marshal(response)
}

// runDiskIOPSTest 对单个磁盘运行简化的 IOPS 测试
// 使用命令: fio -iodepth=128 -numjobs=[cpu 核数] -bs=4k -time_based=1 -runtime=60s -rw=randrw
func runDiskIOPSTest(ctx context.Context, disk *testDiskInfo, runtime int, numJobs int) *command.DiskIOPSResult {
	result := &command.DiskIOPSResult{
		Device:   disk.device,
		Model:    disk.model,
		Kind:     disk.kind,
		TestPath: disk.mountPath,
	}

	startTime := time.Now()

	// 确定测试文件路径
	var testFile string
	var isRawDevice bool

	if strings.HasPrefix(disk.mountPath, "/dev/") {
		// 裸设备测试
		testFile = disk.mountPath
		isRawDevice = true
	} else {
		// 文件系统测试
		testFile = filepath.Join(disk.mountPath, fmt.Sprintf(".fio_iops_%s", disk.device))
		defer os.Remove(testFile)
	}

	// 构建 FIO 命令参数
	// fio -iodepth=128 -numjobs=[cpu 核数] -bs=4k -time_based=1 -runtime=60s -rw=randrw
	args := []string{
		fmt.Sprintf("-filename=%s", testFile),
		"-direct=1",
		"-ioengine=libaio",
		"-bs=4k",
		"-size=1G",
		fmt.Sprintf("-numjobs=%d", numJobs),
		"-iodepth=128",
		"-time_based=1",
		fmt.Sprintf("-runtime=%d", runtime),
		"-thread",
		"-rw=randrw",
		"-rwmixread=50", // 50% 读 50% 写，便于获取独立的读写 IOPS
		"-group_reporting",
		"-name=iops_test",
		"--output-format=json",
	}

	if isRawDevice {
		args = append(args, "--allow_file_create=0")
	}

	cmd := exec.CommandContext(ctx, "fio", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("fio failed: %v, output: %s", err, string(output))
		result.Duration = int(time.Since(startTime).Seconds())
		return result
	}

	// 解析 FIO JSON 输出
	iopsResult := parseIOPSOutput(output)
	if iopsResult != nil {
		result.ReadIOPS = iopsResult.readIOPS
		result.WriteIOPS = iopsResult.writeIOPS
	}

	result.Duration = int(time.Since(startTime).Seconds())
	return result
}

// iopsOnlyResult 简化的 IOPS 结果
type iopsOnlyResult struct {
	readIOPS  float64
	writeIOPS float64
}

// parseIOPSOutput 解析 FIO JSON 输出，只提取 Read/Write IOPS
func parseIOPSOutput(output []byte) *iopsOnlyResult {
	var fioJSON struct {
		Jobs []struct {
			Read struct {
				Iops float64 `json:"iops"`
			} `json:"read"`
			Write struct {
				Iops float64 `json:"iops"`
			} `json:"write"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(output, &fioJSON); err != nil {
		return nil
	}

	if len(fioJSON.Jobs) == 0 {
		return nil
	}

	job := fioJSON.Jobs[0]
	return &iopsOnlyResult{
		readIOPS:  job.Read.Iops,
		writeIOPS: job.Write.Iops,
	}
}

// FormatIOPSResult 格式化 IOPS 测试结果为文本
func FormatIOPSResult(result *command.DiskIOPSResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("设备: %s (%s, %s)\n", result.Device, result.Model, result.Kind))
	sb.WriteString(fmt.Sprintf("测试路径: %s\n", result.TestPath))
	sb.WriteString(fmt.Sprintf("测试耗时: %d 秒\n", result.Duration))

	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("\n【错误信息】\n  %s\n", result.Error))
	} else {
		sb.WriteString("\n【IOPS 结果】\n")
		sb.WriteString(fmt.Sprintf("  Read IOPS:  %.0f\n", result.ReadIOPS))
		sb.WriteString(fmt.Sprintf("  Write IOPS: %.0f\n", result.WriteIOPS))
	}

	return sb.String()
}
