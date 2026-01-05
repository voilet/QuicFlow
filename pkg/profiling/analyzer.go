package profiling

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/google/pprof/profile"
)

// Analyzer 性能分析器
type Analyzer struct {
	profiler *Profiler
}

// NewAnalyzer 创建分析器
func NewAnalyzer(profiler *Profiler) *Analyzer {
	return &Analyzer{profiler: profiler}
}

// Analyze 分析 profile 并生成报告
func (a *Analyzer) Analyze(profileID string) (*AnalysisReport, error) {
	prof, err := a.profiler.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("profile not found: %w", err)
	}

	if prof.Status != StatusCompleted {
		return nil, fmt.Errorf("profile is not completed")
	}

	// 读取 profile 文件
	f, err := os.Open(prof.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile file: %w", err)
	}
	defer f.Close()

	p, err := profile.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	// 根据类型分析
	switch prof.Type {
	case ProfileTypeCPU:
		return a.analyzeCPU(prof, p)
	case ProfileTypeMemory, ProfileTypeHeap:
		return a.analyzeMemory(prof, p)
	case ProfileTypeGoroutine:
		return a.analyzeGoroutine(prof, p)
	case ProfileTypeBlock:
		return a.analyzeBlock(prof, p)
	case ProfileTypeMutex:
		return a.analyzeMutex(prof, p)
	default:
		return a.analyzeGeneric(prof, p)
	}
}

// analyzeCPU 分析 CPU 性能
func (a *Analyzer) analyzeCPU(profile *Profile, p *profile.Profile) (*AnalysisReport, error) {
	report := &AnalysisReport{
		ProfileID:   profile.ID,
		ProfileType: profile.Type,
		GeneratedAt: time.Now(),
		Summary:     ReportSummary{},
		Issues:      []AnalysisIssue{},
		TopFunctions: []TopFunction{},
		Metrics: ProfileMetrics{
			GCPercentage: 0,
		},
	}

	// 获取样本总数
	totalSamples := int64(0)
	for _, s := range p.Sample {
		for _, v := range s.Value {
			totalSamples += v
		}
	}
	report.Summary.TotalSamples = totalSamples

	// 按函数聚合
	funcSamples := make(map[string]int64)
	funcFlat := make(map[string]int64) // 自身时间

	for _, s := range p.Sample {
		sampleValue := int64(0)
		for _, v := range s.Value {
			sampleValue += v
		}

		// 找到最底层的函数（叶子节点）
		if len(s.Location) > 0 {
			loc := s.Location[0]
			for _, line := range loc.Line {
				if line.Function != nil {
					name := line.Function.Name
					if name == "" {
						name = line.Function.SystemName
					}
					if name != "" {
						funcFlat[name] += sampleValue
					}
				}
			}
		}

		// 所有调用栈中的函数都累计
		for _, loc := range s.Location {
			for _, line := range loc.Line {
				if line.Function != nil {
					name := line.Function.Name
					if name == "" {
						name = line.Function.SystemName
					}
					if name != "" {
						funcSamples[name] += sampleValue
					}
				}
			}
		}
	}

	// 找出热点函数
	type funcInfo struct {
		name  string
		samples int64
		flat  int64
	}
	funcs := make([]funcInfo, 0, len(funcSamples))
	for name, cum := range funcSamples {
		funcs = append(funcs, funcInfo{
			name:   name,
			samples: cum,
			flat:   funcFlat[name],
		})
	}
	sort.Slice(funcs, func(i, j int) bool {
		return funcs[i].samples > funcs[j].samples
	})

	// 取前 20 个函数
	for i, f := range funcs {
		if i >= 20 {
			break
		}
		percentage := float64(f.samples) / float64(totalSamples) * 100
		report.TopFunctions = append(report.TopFunctions, TopFunction{
			Name:       shortenName(f.name),
			Percentage: percentage,
			Samples:    f.samples,
			Flat:       f.flat,
			Cum:        f.samples,
		})

		// 生成问题
		if percentage > 20 {
			report.Issues = append(report.Issues, AnalysisIssue{
				ID:          fmt.Sprintf("cpu-high-%d", len(report.Issues)),
				Type:        IssueTypeCPUHigh,
				Severity:    SeverityHigh,
				Title:       "高 CPU 占用函数",
				Description: fmt.Sprintf("函数 %s 占用了 %.1f%% 的 CPU 时间", shortenName(f.name), percentage),
				Location:    f.name,
				Value:       percentage,
				Threshold:   20.0,
				Suggestion:  "考虑优化该函数的算法或减少调用频率。如果这是热点代码，可以考虑使用更高效的数据结构或算法。",
				RelatedFunc: f.name,
			})
		}
	}

	// 检查 GC 时间
	gcTime := int64(0)
	for name, samples := range funcSamples {
		if containsGC(name) {
			gcTime += samples
		}
	}
	if totalSamples > 0 {
		gcPercentage := float64(gcTime) / float64(totalSamples) * 100
		report.Metrics.GCPercentage = gcPercentage

		if gcPercentage > 10 {
			report.Issues = append(report.Issues, AnalysisIssue{
				ID:          "gc-high",
				Type:        IssueTypeGCHigh,
				Severity:    SeverityMedium,
				Title:       "GC 时间过高",
				Description: fmt.Sprintf("垃圾回收占用了 %.1f%% 的 CPU 时间", gcPercentage),
				Value:       gcPercentage,
				Threshold:   10.0,
				Suggestion:  "考虑减少内存分配频率，使用对象池（sync.Pool），或调整 GOGC 环境变量。",
			})
		}
	}

	// 统计问题数量
	a.finalizeSummary(report)

	return report, nil
}

// analyzeMemory 分析内存
func (a *Analyzer) analyzeMemory(profile *Profile, p *profile.Profile) (*AnalysisReport, error) {
	report := &AnalysisReport{
		ProfileID:   profile.ID,
		ProfileType: profile.Type,
		GeneratedAt: time.Now(),
		Summary:     ReportSummary{},
		Issues:      []AnalysisIssue{},
		TopFunctions: []TopFunction{},
	}

	// 获取总分配量
	totalBytes := int64(0)
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			totalBytes += s.Value[0]
		}
	}
	report.Summary.TotalSamples = totalBytes

	// 按函数聚合分配
	funcAllocs := make(map[string]int64)
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			value := s.Value[0]
			// 找到分配位置
			if len(s.Location) > 0 {
				loc := s.Location[0]
				for _, line := range loc.Line {
					if line.Function != nil {
						name := line.Function.Name
						if name == "" {
							name = line.Function.SystemName
						}
						if name != "" {
							funcAllocs[name] += value
						}
					}
				}
			}
		}
	}

	// 找出热点分配函数
	type allocInfo struct {
		name  string
		bytes int64
	}
	allocs := make([]allocInfo, 0, len(funcAllocs))
	for name, bytes := range funcAllocs {
		allocs = append(allocs, allocInfo{name: name, bytes: bytes})
	}
	sort.Slice(allocs, func(i, j int) bool {
		return allocs[i].bytes > allocs[j].bytes
	})

	// 取前 20 个
	for i, a := range allocs {
		if i >= 20 {
			break
		}
		percentage := float64(a.bytes) / float64(totalBytes) * 100
		report.TopFunctions = append(report.TopFunctions, TopFunction{
			Name:       shortenName(a.name),
			Percentage: percentage,
			Samples:    a.bytes,
			Flat:       a.bytes,
			Cum:        a.bytes,
		})

		// 检查内存泄漏风险
		if percentage > 10 {
			report.Issues = append(report.Issues, AnalysisIssue{
				ID:          fmt.Sprintf("mem-high-%d", len(report.Issues)),
				Type:        IssueTypeMemoryLeak,
				Severity:    SeverityHigh,
				Title:       "高内存分配点",
				Description: fmt.Sprintf("函数 %s 分配了 %.1f%% 的内存 (%.2f MB)",
					shortenName(a.name), percentage, float64(a.bytes)/(1024*1024)),
				Location:    a.name,
				Value:       percentage,
				Threshold:   10.0,
				Suggestion:  "检查是否有内存泄漏，考虑使用对象池复用对象，减少频繁的内存分配。",
				RelatedFunc: a.name,
			})
		}
	}

	// 获取当前内存信息
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	report.Metrics.TotalAlloc = m.TotalAlloc
	report.Metrics.HeapAlloc = m.HeapAlloc
	report.Metrics.HeapObjects = m.HeapObjects

	a.finalizeSummary(report)

	return report, nil
}

// analyzeGoroutine 分析 Goroutine
func (a *Analyzer) analyzeGoroutine(profile *Profile, p *profile.Profile) (*AnalysisReport, error) {
	report := &AnalysisReport{
		ProfileID:   profile.ID,
		ProfileType: profile.Type,
		GeneratedAt: time.Now(),
		Summary:     ReportSummary{},
		Issues:      []AnalysisIssue{},
		TopFunctions: []TopFunction{},
	}

	// 统计 goroutine 数量
	goroutineCount := int64(0)
	if len(p.Sample) > 0 {
		for _, s := range p.Sample {
			if len(s.Value) > 0 {
				goroutineCount += s.Value[0]
			}
		}
	} else {
		// 如果没有样本，使用运行时数据
		goroutineCount = int64(runtime.NumGoroutine())
	}
	report.Summary.TotalSamples = goroutineCount
	report.Metrics.NumGoroutine = int(goroutineCount)

	// 按调用栈聚合
	stackCounts := make(map[string]int64)
	for _, s := range p.Sample {
		// 构建调用栈字符串
		var stack string
		for i := len(s.Location) - 1; i >= 0; i-- {
			loc := s.Location[i]
			for _, line := range loc.Line {
				if line.Function != nil && line.Function.Name != "" {
					if stack != "" {
						stack += " <- "
					}
					stack += line.Function.Name
				}
			}
		}

		if len(s.Value) > 0 {
			stackCounts[stack] += s.Value[0]
		}
	}

	// 找出热点调用栈
	type stackInfo struct {
		stack string
		count int64
	}
	stacks := make([]stackInfo, 0, len(stackCounts))
	for stack, count := range stackCounts {
		stacks = append(stacks, stackInfo{stack: stack, count: count})
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].count > stacks[j].count
	})

	// 取前 20 个
	for i, s := range stacks {
		if i >= 20 {
			break
		}
		percentage := float64(s.count) / float64(goroutineCount) * 100
		report.TopFunctions = append(report.TopFunctions, TopFunction{
			Name:       truncateText(s.stack, 60),
			Percentage: percentage,
			Samples:    s.count,
		})
	}

	// 检查 goroutine 泄漏
	if goroutineCount > 10000 {
		report.Issues = append(report.Issues, AnalysisIssue{
			ID:       "goroutine-critical",
			Type:     IssueTypeGoroutineLeak,
			Severity: SeverityCritical,
			Title:    "严重的 Goroutine 泄漏",
			Description: fmt.Sprintf("当前有 %d 个 goroutine，远超正常范围", goroutineCount),
			Value:    goroutineCount,
			Threshold: 10000,
			Suggestion: "可能存在 goroutine 泄漏。检查是否有未关闭的 channel、没有超时的 context，或无限循环的 goroutine。",
		})
	} else if goroutineCount > 1000 {
		report.Issues = append(report.Issues, AnalysisIssue{
			ID:       "goroutine-high",
			Type:     IssueTypeGoroutineLeak,
			Severity: SeverityHigh,
			Title:    "Goroutine 数量过多",
			Description: fmt.Sprintf("当前有 %d 个 goroutine，数量较多", goroutineCount),
			Value:    goroutineCount,
			Threshold: 1000,
			Suggestion: "检查 goroutine 创建逻辑，确保使用 context 控制生命周期，避免无限制创建 goroutine。",
		})
	}

	a.finalizeSummary(report)

	return report, nil
}

// analyzeBlock 分析阻塞
func (a *Analyzer) analyzeBlock(profile *Profile, p *profile.Profile) (*AnalysisReport, error) {
	report := &AnalysisReport{
		ProfileID:   profile.ID,
		ProfileType: profile.Type,
		GeneratedAt: time.Now(),
		Summary:     ReportSummary{},
		Issues:      []AnalysisIssue{},
		TopFunctions: []TopFunction{},
	}

	totalDelay := int64(0)
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			totalDelay += s.Value[0]
		}
	}
	report.Summary.TotalSamples = totalDelay

	// 按函数聚合
	funcDelays := make(map[string]int64)
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			delay := s.Value[0]
			if len(s.Location) > 0 {
				loc := s.Location[0]
				for _, line := range loc.Line {
					if line.Function != nil {
						name := line.Function.Name
						if name == "" {
							name = line.Function.SystemName
						}
						if name != "" {
							funcDelays[name] += delay
						}
					}
				}
			}
		}
	}

	// 找出热点阻塞
	type delayInfo struct {
		name  string
		delay int64
	}
	delays := make([]delayInfo, 0, len(funcDelays))
	for name, delay := range funcDelays {
		delays = append(delays, delayInfo{name: name, delay: delay})
	}
	sort.Slice(delays, func(i, j int) bool {
		return delays[i].delay > delays[j].delay
	})

	for i, d := range delays {
		if i >= 20 {
			break
		}
		percentage := float64(d.delay) / float64(totalDelay) * 100
		report.TopFunctions = append(report.TopFunctions, TopFunction{
			Name:       shortenName(d.name),
			Percentage: percentage,
			Samples:    d.delay,
		})

		if percentage > 5 {
			report.Issues = append(report.Issues, AnalysisIssue{
				Type:        IssueTypeBlocking,
				Severity:    SeverityMedium,
				Title:       "高阻塞时间",
				Description: fmt.Sprintf("%s 阻塞了 %.1f%% 的时间", shortenName(d.name), percentage),
				Location:    d.name,
				Value:       percentage,
				Threshold:   5.0,
				Suggestion:  "考虑使用缓冲 channel、减少锁粒度，或使用异步模式减少阻塞。",
			})
		}
	}

	a.finalizeSummary(report)

	return report, nil
}

// analyzeMutex 分析互斥锁竞争
func (a *Analyzer) analyzeMutex(profile *Profile, p *profile.Profile) (*AnalysisReport, error) {
	report := &AnalysisReport{
		ProfileID:   profile.ID,
		ProfileType: profile.Type,
		GeneratedAt: time.Now(),
		Summary:     ReportSummary{},
		Issues:      []AnalysisIssue{},
		TopFunctions: []TopFunction{},
	}

	totalContention := int64(0)
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			totalContention += s.Value[0]
		}
	}
	report.Summary.TotalSamples = totalContention

	// 按锁位置聚合
	lockContentions := make(map[string]int64)
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			contention := s.Value[0]
			if len(s.Location) > 0 {
				loc := s.Location[0]
				for _, line := range loc.Line {
					if line.Function != nil {
						name := line.Function.Name
						if name == "" {
							name = line.Function.SystemName
						}
						if name != "" {
							lockContentions[name] += contention
						}
					}
				}
			}
		}
	}

	// 找出热点锁
	type contentionInfo struct {
		name       string
		contention int64
	}
	contentions := make([]contentionInfo, 0, len(lockContentions))
	for name, c := range lockContentions {
		contentions = append(contentions, contentionInfo{name: name, contention: c})
	}
	sort.Slice(contentions, func(i, j int) bool {
		return contentions[i].contention > contentions[j].contention
	})

	for i, c := range contentions {
		if i >= 20 {
			break
		}
		percentage := float64(c.contention) / float64(totalContention) * 100
		report.TopFunctions = append(report.TopFunctions, TopFunction{
			Name:       shortenName(c.name),
			Percentage: percentage,
			Samples:    c.contention,
		})

		if percentage > 5 {
			report.Issues = append(report.Issues, AnalysisIssue{
				Type:        IssueTypeContention,
				Severity:    SeverityMedium,
				Title:       "高锁竞争",
				Description: fmt.Sprintf("%s 发生了 %.1f%% 的锁竞争", shortenName(c.name), percentage),
				Location:    c.name,
				Value:       percentage,
				Threshold:   5.0,
				Suggestion:  "考虑减少锁的粒度，使用 sync.Map 或原子操作替代锁，或使用读写锁（sync.RWMutex）。",
			})
		}
	}

	a.finalizeSummary(report)

	return report, nil
}

// analyzeGeneric 通用分析
func (a *Analyzer) analyzeGeneric(profile *Profile, p *profile.Profile) (*AnalysisReport, error) {
	return &AnalysisReport{
		ProfileID:   profile.ID,
		ProfileType: profile.Type,
		GeneratedAt: time.Now(),
		Summary: ReportSummary{
			TotalSamples: 0,
			OverallStatus: "info",
			Recommendations: "此类型的分析正在开发中",
		},
	}, nil
}

// finalizeSummary 完成摘要统计
func (a *Analyzer) finalizeSummary(report *AnalysisReport) {
	critical := 0
	high := 0
	medium := 0

	for _, issue := range report.Issues {
		switch issue.Severity {
		case SeverityCritical:
			critical++
		case SeverityHigh:
			high++
		case SeverityMedium:
			medium++
		}
	}

	report.Summary.CriticalCount = critical
	report.Summary.HighCount = high
	report.Summary.MediumCount = medium
	report.Summary.IssueCount = len(report.Issues)

	// 确定整体状态
	if critical > 0 {
		report.Summary.OverallStatus = "critical"
		report.Summary.Recommendations = "发现严重性能问题，建议立即处理关键问题。"
	} else if high > 0 {
		report.Summary.OverallStatus = "warning"
		report.Summary.Recommendations = fmt.Sprintf("发现 %d 个高优先级问题，建议尽快优化。", high)
	} else if medium > 0 {
		report.Summary.OverallStatus = "warning"
		report.Summary.Recommendations = fmt.Sprintf("发现 %d 个中优先级问题，可以在空闲时处理。", medium)
	} else {
		report.Summary.OverallStatus = "healthy"
		report.Summary.Recommendations = "未发现明显性能问题，系统状态良好。"
	}
}

// containsGC 检查是否是 GC 相关函数
func containsGC(name string) bool {
	gcFuncs := []string{
		"runtime.GC", "runtime.mallocgc", "runtime.gcForceMarkWorkerTenured",
		"runtime.gcBgMarkWorker", "runtime.gcStopTheWorld",
	}
	for _, gc := range gcFuncs {
		if contains(name, gc) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
