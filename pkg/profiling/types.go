package profiling

import "time"

// ProfileType 性能分析类型
type ProfileType string

const (
	ProfileTypeCPU       ProfileType = "cpu"
	ProfileTypeMemory    ProfileType = "memory"
	ProfileTypeGoroutine ProfileType = "goroutine"
	ProfileTypeHeap      ProfileType = "heap"
	ProfileTypeBlock     ProfileType = "block"
	ProfileTypeMutex     ProfileType = "mutex"
)

// ProfileStatus 采集状态
type ProfileStatus string

const (
	StatusPending   ProfileStatus = "pending"
	StatusRunning   ProfileStatus = "running"
	StatusCompleted ProfileStatus = "completed"
	StatusFailed    ProfileStatus = "failed"
)

// Profile 采集记录
type Profile struct {
	ID          string        `json:"id" gorm:"primaryKey"`
	Type        ProfileType   `json:"type" gorm:"index"`
	Name        string        `json:"name" gorm:"size:255"`
	Status      ProfileStatus `json:"status" gorm:"index"`
	Duration    int32         `json:"duration"` // 采集时长（秒），快照类型为0
	FilePath    string        `json:"file_path" gorm:"size:500"`
	FlamePath   string        `json:"flame_path" gorm:"size:500"`
	FileSize    int64         `json:"file_size"`
	SampleCount int64         `json:"sample_count"`
	Error       string        `json:"error" gorm:"size:500"`
	CreatedBy   string        `json:"created_by" gorm:"size:100"` // 用户名
	CreatedAt   time.Time     `json:"created_at" gorm:"autoCreateTime"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// ProfileMeta 采集元数据（用于存储在 JSONB 中）
type ProfileMeta struct {
	GoVersion    string            `json:"go_version"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	NumCPU       int               `json:"num_cpu"`
	NumGoroutine int               `json:"num_goroutine"`
	MemStats     MemoryStats       `json:"mem_stats"`
	CustomData   map[string]string `json:"custom_data,omitempty"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`         // 已分配内存（字节）
	TotalAlloc   uint64 `json:"total_alloc"`   // 累计分配（字节）
	Sys          uint64 `json:"sys"`           // 从系统获取的内存
	HeapAlloc    uint64 `json:"heap_alloc"`    // 堆已分配
	HeapSys      uint64 `json:"heap_sys"`      // 堆系统内存
	HeapInuse    uint64 `json:"heap_inuse"`    // 堆使用中
	HeapReleased uint64 `json:"heap_released"` // 堆已释放
	HeapObjects  uint64 `json:"heap_objects"`  // 堆对象数
	StackInuse   uint64 `json:"stack_inuse"`   // 栈使用中
	StackSys     uint64 `json:"stack_sys"`     // 栈系统内存
	MSpanInuse   uint64 `json:"mspan_inuse"`   // MSpan 使用中
	MSpanSys     uint64 `json:"mspan_sys"`     // MSpan 系统内存
	MCacheInuse  uint64 `json:"mcache_inuse"`  // MCache 使用中
	MCacheSys    uint64 `json:"mcache_sys"`    // MCache 系统内存
	BuckHashSys  uint64 `json:"buck_hash_sys"` // Bucket hash 系统内存
	GCSys        uint64 `json:"gc_sys"`        // GC 系统内存
	NextGC       uint64 `json:"next_gc"`       // 下次 GC 阈值
	LastGC       uint64 `json:"last_gc"`       // 上次 GC 时间
	NumGC        uint32 `json:"num_gc"`        // GC 次数
	GCPauseTotal uint64 `json:"gc_pause_total"` // GC 总暂停时间（纳秒）
}

// AnalysisIssue 分析问题
type AnalysisIssue struct {
	ID          string       `json:"id"`
	Type        IssueType    `json:"type"`
	Severity    Severity     `json:"severity"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Location    string       `json:"location,omitempty"`    // 函数/文件位置
	Value       interface{}  `json:"value"`                 // 实际值
	Threshold   interface{}  `json:"threshold"`             // 阈值
	Suggestion  string       `json:"suggestion"`            // 优化建议
	RelatedFunc string       `json:"related_func,omitempty"` // 相关函数
}

// IssueType 问题类型
type IssueType string

const (
	IssueTypeCPUHigh       IssueType = "cpu_high"
	IssueTypeMemoryLeak    IssueType = "memory_leak"
	IssueTypeGoroutineLeak IssueType = "goroutine_leak"
	IssueTypeGCHigh        IssueType = "gc_high"
	IssueTypeBlocking      IssueType = "blocking"
	IssueTypeContention    IssueType = "contention"
)

// Severity 严重程度
type Severity string

const (
	SeverityCritical Severity = "critical" // 严重
	SeverityHigh     Severity = "high"     // 高
	SeverityMedium   Severity = "medium"   // 中
	SeverityLow      Severity = "low"      // 低
	SeverityInfo     Severity = "info"     // 信息
)

// AnalysisReport 分析报告
type AnalysisReport struct {
	ProfileID    string          `json:"profile_id"`
	ProfileType  ProfileType     `json:"profile_type"`
	GeneratedAt  time.Time       `json:"generated_at"`
	Summary      ReportSummary   `json:"summary"`
	TopFunctions []TopFunction   `json:"top_functions"`
	Issues       []AnalysisIssue `json:"issues"`
	Metrics      ProfileMetrics  `json:"metrics"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalSamples    int64  `json:"total_samples"`
	TotalDuration   int64  `json:"total_duration_ms"` // 毫秒
	IssueCount      int    `json:"issue_count"`
	CriticalCount   int    `json:"critical_count"`
	HighCount       int    `json:"high_count"`
	MediumCount     int    `json:"medium_count"`
	OverallStatus   string `json:"overall_status"` // healthy, warning, critical
	Recommendations string `json:"recommendations"`
}

// TopFunction 热点函数
type TopFunction struct {
	Name      string  `json:"name"`
	File      string  `json:"file,omitempty"`
	Percentage float64 `json:"percentage"` // 占比（百分比）
	Samples   int64   `json:"samples"`
	Flat      int64   `json:"flat"`       // 自身时间
	Cum       int64   `json:"cum"`        // 累计时间
}

// ProfileMetrics 性能指标
type ProfileMetrics struct {
	// CPU 指标
	GCPercentage   float64 `json:"gc_percentage,omitempty"`    // GC 占比
	SystemCallPercentage float64 `json:"system_call_percentage,omitempty"`

	// 内存指标
	TotalAlloc     uint64  `json:"total_alloc,omitempty"`
	HeapAlloc      uint64  `json:"heap_alloc,omitempty"`
	HeapObjects    uint64  `json:"heap_objects,omitempty"`
	AllocationRate float64 `json:"allocation_rate,omitempty"` // 分配速率（MB/s）

	// Goroutine 指标
	NumGoroutine   int     `json:"num_goroutine,omitempty"`
	ActiveGoroutine int    `json:"active_goroutine,omitempty"`
	BlockedGoroutine int   `json:"blocked_goroutine,omitempty"`

	// 阻塞/互斥指标
	BlockTime      float64 `json:"block_time,omitempty"`
	ContentionRate float64 `json:"contention_rate,omitempty"`
}

// FlameGraphNode 火焰图节点
type FlameGraphNode struct {
	Name     string           `json:"name"`
	Value    int64            `json:"value"`
	Children []*FlameGraphNode `json:"children,omitempty"`
}

// FlameGraphConfig 火焰图配置
type FlameGraphConfig struct {
	Width       int     `json:"width"`        // 宽度（像素）
	Height      int     `json:"height"`       // 每个栈的高度（像素）
	FontHeight  int     `json:"font_height"`  // 字体高度
	MinWidth    float64 `json:"min_width"`    // 最小宽度（像素）- 太小的块不显示
	ColorScheme string  `json:"color_scheme"` // 配色方案: warm, cool, rainbow
}

// DefaultFlameGraphConfig 默认火焰图配置
var DefaultFlameGraphConfig = FlameGraphConfig{
	Width:       1200,
	Height:      24,
	FontHeight:  12,
	MinWidth:    0.5,
	ColorScheme: "warm",
}

// Request types

// StartCPUProfileRequest 启动 CPU 采集请求
type StartCPUProfileRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Duration int32  `json:"duration" binding:"required,min=1,max=300"` // 1-300秒
}

// CaptureProfileRequest 采集快照请求
type CaptureProfileRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Debug       int    `json:"debug"` // pprof debug level (0-2)
	GoroutineDepth int `json:"goroutine_depth"` // goroutine 堆栈深度
}

// ProfileListRequest 采集列表请求
type ProfileListRequest struct {
	Type   ProfileType `form:"type"`
	Status ProfileStatus `form:"status"`
	Page   int         `form:"page"`
	PageSize int       `form:"page_size"`
}

// ProfileListResponse 采集列表响应
type ProfileListResponse struct {
	Success bool      `json:"success"`
	Total   int64     `json:"total"`
	Page    int       `json:"page"`
	Profiles []Profile `json:"profiles"`
	Message string    `json:"message,omitempty"`
}

// StartCPUProfileResponse 启动 CPU 采集响应
type StartCPUProfileResponse struct {
	Success bool   `json:"success"`
	ProfileID string `json:"profile_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// CaptureProfileResponse 采集快照响应
type CaptureProfileResponse struct {
	Success bool   `json:"success"`
	ProfileID string `json:"profile_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// AnalysisResponse 分析响应
type AnalysisResponse struct {
	Success bool           `json:"success"`
	Report  *AnalysisReport `json:"report,omitempty"`
	Message string         `json:"message,omitempty"`
}
