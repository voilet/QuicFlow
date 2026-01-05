package profiling

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Profiler 性能采集器
type Profiler struct {
	db        *gorm.DB
	storeDir  string // 采集文件存储目录
	mu        sync.RWMutex
	running   map[string]*activeProfile // 正在运行的采集
}

// activeProfile 正在运行的采集
type activeProfile struct {
	profileType ProfileType
	stopFunc    func() error
	startedAt   time.Time
}

// NewProfiler 创建性能采集器
func NewProfiler(db *gorm.DB, storeDir string) (*Profiler, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	return &Profiler{
		db:       db,
		storeDir: storeDir,
		running:  make(map[string]*activeProfile),
	}, nil
}

// Init 初始化采集器（创建数据库表）
func (p *Profiler) Init() error {
	return p.db.AutoMigrate(&Profile{})
}

// StartCPUProfile 启动 CPU 性能采集
func (p *Profiler) StartCPUProfile(name string, duration int32, createdBy string) (*Profile, error) {
	profileID := uuid.New().String()
	fileName := fmt.Sprintf("cpu-%s-%s.prof", profileID[:8], time.Now().Format("20060102-150405"))
	filePath := filepath.Join(p.storeDir, fileName)

	// 创建记录
	profile := &Profile{
		ID:        profileID,
		Type:      ProfileTypeCPU,
		Name:      name,
		Status:    StatusRunning,
		Duration:  duration,
		FilePath:  filePath,
		CreatedBy: createdBy,
	}

	if err := p.db.Create(profile).Error; err != nil {
		return nil, fmt.Errorf("failed to create profile record: %w", err)
	}

	// 创建文件
	f, err := os.Create(filePath)
	if err != nil {
		profile.Status = StatusFailed
		profile.Error = err.Error()
		p.db.Save(profile)
		return nil, fmt.Errorf("failed to create profile file: %w", err)
	}

	// 启动 CPU 采集
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		os.Remove(filePath)
		profile.Status = StatusFailed
		profile.Error = err.Error()
		p.db.Save(profile)
		return nil, fmt.Errorf("failed to start CPU profile: %w", err)
	}

	// 记录正在运行的采集
	p.mu.Lock()
	p.running[profileID] = &activeProfile{
		profileType: ProfileTypeCPU,
		stopFunc: func() error {
			pprof.StopCPUProfile()
			return f.Close()
		},
		startedAt: time.Now(),
	}
	p.mu.Unlock()

	// 后台自动停止
	go p.autoStopCPUProfile(profileID, time.Duration(duration)*time.Second)

	return profile, nil
}

// autoStopCPUProfile 自动停止 CPU 采集
func (p *Profiler) autoStopCPUProfile(profileID string, duration time.Duration) {
	time.Sleep(duration)

	p.mu.Lock()
	active, ok := p.running[profileID]
	if !ok {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	if active.stopFunc != nil {
		if err := active.stopFunc(); err != nil {
			p.handleStopError(profileID, err)
			return
		}
	}

	// 更新记录
	p.finalizeProfile(profileID, nil)
}

// CaptureMemoryProfile 采集内存快照
func (p *Profiler) CaptureMemoryProfile(name string, createdBy string) (*Profile, error) {
	profileID := uuid.New().String()
	fileName := fmt.Sprintf("heap-%s-%s.prof", profileID[:8], time.Now().Format("20060102-150405"))
	filePath := filepath.Join(p.storeDir, fileName)

	// 创建记录
	profile := &Profile{
		ID:        profileID,
		Type:      ProfileTypeMemory,
		Name:      name,
		Status:    StatusPending,
		Duration:  0, // 快照类型
		FilePath:  filePath,
		CreatedBy: createdBy,
	}

	if err := p.db.Create(profile).Error; err != nil {
		return nil, fmt.Errorf("failed to create profile record: %w", err)
	}

	// 采集堆内存
	if err := p.captureProfile("heap", filePath, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// CaptureGoroutineProfile 采集 Goroutine 快照
func (p *Profiler) CaptureGoroutineProfile(name string, createdBy string) (*Profile, error) {
	profileID := uuid.New().String()
	fileName := fmt.Sprintf("goroutine-%s-%s.prof", profileID[:8], time.Now().Format("20060102-150405"))
	filePath := filepath.Join(p.storeDir, fileName)

	// 创建记录
	profile := &Profile{
		ID:        profileID,
		Type:      ProfileTypeGoroutine,
		Name:      name,
		Status:    StatusPending,
		Duration:  0,
		FilePath:  filePath,
		CreatedBy: createdBy,
	}

	if err := p.db.Create(profile).Error; err != nil {
		return nil, fmt.Errorf("failed to create profile record: %w", err)
	}

	// 采集 goroutine
	if err := p.captureProfile("goroutine", filePath, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// SaveUploadedCPUProfile 保存已采集的 CPU profile（从标准 pprof 端点上传）
func (p *Profiler) SaveUploadedCPUProfile(fileHeader *multipart.FileHeader, name, createdBy string) (*Profile, error) {
	profileID := uuid.New().String()
	fileName := fmt.Sprintf("cpu-uploaded-%s-%s.prof", profileID[:8], time.Now().Format("20060102-150405"))
	filePath := filepath.Join(p.storeDir, fileName)

	// 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := dst.ReadFrom(src); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 获取文件大小
	info, _ := dst.Stat()
	fileSize := info.Size()
	completedAt := time.Now()

	// 创建记录
	profile := &Profile{
		ID:          profileID,
		Type:        ProfileTypeCPU,
		Name:        name,
		Status:      StatusCompleted,
		Duration:    0, // 上传的文件无法确定采集时长
		FilePath:    filePath,
		FileSize:    fileSize,
		CreatedBy:   createdBy,
		CompletedAt: &completedAt,
	}

	if err := p.db.Create(profile).Error; err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create profile record: %w", err)
	}

	return profile, nil
}

// captureProfile 通用的快照采集
func (p *Profiler) captureProfile(profileName, filePath string, profile *Profile) error {
	profile.Status = StatusRunning
	p.db.Save(profile)

	f, err := os.Create(filePath)
	if err != nil {
		profile.Status = StatusFailed
		profile.Error = err.Error()
		p.db.Save(profile)
		return fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	// 获取对应的 profile
	switch profileName {
	case "heap", "allocs", "inuse_space":
		if err := pprof.WriteHeapProfile(f); err != nil {
			return p.handleCaptureError(profile, err)
		}
	case "goroutine":
		g := pprof.Lookup("goroutine")
		if g == nil {
			return p.handleCaptureError(profile, fmt.Errorf("goroutine profile not found"))
		}
		if err := g.WriteTo(f, 0); err != nil {
			return p.handleCaptureError(profile, err)
		}
	case "block":
		g := pprof.Lookup("block")
		if g == nil {
			return p.handleCaptureError(profile, fmt.Errorf("block profile not found (need to set runtime.SetBlockProfileRate)"))
		}
		if err := g.WriteTo(f, 0); err != nil {
			return p.handleCaptureError(profile, err)
		}
	case "mutex":
		g := pprof.Lookup("mutex")
		if g == nil {
			return p.handleCaptureError(profile, fmt.Errorf("mutex profile not found (need to set runtime.SetMutexProfileFraction)"))
		}
		if err := g.WriteTo(f, 0); err != nil {
			return p.handleCaptureError(profile, err)
		}
	default:
		return p.handleCaptureError(profile, fmt.Errorf("unknown profile type: %s", profileName))
	}

	// 获取文件大小
	info, _ := os.Stat(filePath)
	fileSize := info.Size()

	return p.finalizeProfile(profile.ID, &fileSize)
}

// handleCaptureError 处理采集错误
func (p *Profiler) handleCaptureError(profile *Profile, err error) error {
	profile.Status = StatusFailed
	profile.Error = err.Error()
	p.db.Save(profile)
	return fmt.Errorf("failed to capture profile: %w", err)
}

// handleStopError 处理停止错误
func (p *Profiler) handleStopError(profileID string, err error) {
	p.db.Model(&Profile{}).Where("id = ?", profileID).Updates(map[string]interface{}{
		"status":     StatusFailed,
		"error":      err.Error(),
		"completed_at": time.Now(),
	})
}

// finalizeProfile 完成采集
func (p *Profiler) finalizeProfile(profileID string, fileSize *int64) error {
	updates := map[string]interface{}{
		"status":      StatusCompleted,
		"completed_at": time.Now(),
	}

	if fileSize != nil {
		updates["file_size"] = *fileSize
	}

	// 清理运行记录
	p.mu.Lock()
	delete(p.running, profileID)
	p.mu.Unlock()

	return p.db.Model(&Profile{}).Where("id = ?", profileID).Updates(updates).Error
}

// ListProfiles 获取采集列表
func (p *Profiler) ListProfiles(profileType ProfileType, status ProfileStatus, page, pageSize int) ([]Profile, int64, error) {
	var profiles []Profile
	var total int64

	query := p.db.Model(&Profile{})

	if profileType != "" {
		query = query.Where("type = ?", profileType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&profiles).Error

	return profiles, total, err
}

// GetProfile 获取单个采集
func (p *Profiler) GetProfile(profileID string) (*Profile, error) {
	var profile Profile
	err := p.db.Where("id = ?", profileID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// DeleteProfile 删除采集文件和记录
func (p *Profiler) DeleteProfile(profileID string) error {
	profile, err := p.GetProfile(profileID)
	if err != nil {
		return err
	}

	// 删除文件
	if profile.FilePath != "" {
		os.Remove(profile.FilePath)
	}
	if profile.FlamePath != "" {
		os.Remove(profile.FlamePath)
	}

	// 删除数据库记录
	return p.db.Delete(&Profile{}, "id = ?", profileID).Error
}

// GetProfilePath 获取采集文件路径
func (p *Profiler) GetProfilePath(profileID string) (string, error) {
	profile, err := p.GetProfile(profileID)
	if err != nil {
		return "", err
	}
	if profile.Status != StatusCompleted {
		return "", fmt.Errorf("profile is not completed")
	}
	return profile.FilePath, nil
}

// GetFlameGraphPath 获取火焰图路径
func (p *Profiler) GetFlameGraphPath(profileID string) (string, error) {
	profile, err := p.GetProfile(profileID)
	if err != nil {
		return "", err
	}
	if profile.FlamePath == "" {
		return "", fmt.Errorf("flame graph not generated")
	}
	return profile.FlamePath, nil
}

// SetFlameGraphPath 设置火焰图路径
func (p *Profiler) SetFlameGraphPath(profileID, flamePath string) error {
	return p.db.Model(&Profile{}).Where("id = ?", profileID).Update("flame_path", flamePath).Error
}

// GetSystemInfo 获取系统信息
func GetSystemInfo() *ProfileMeta {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &ProfileMeta{
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemStats: MemoryStats{
			Alloc:         m.Alloc,
			TotalAlloc:    m.TotalAlloc,
			Sys:           m.Sys,
			HeapAlloc:     m.HeapAlloc,
			HeapSys:       m.HeapSys,
			HeapInuse:     m.HeapInuse,
			HeapReleased:  m.HeapReleased,
			HeapObjects:   m.HeapObjects,
			StackInuse:    m.StackInuse,
			StackSys:      m.StackSys,
			MSpanInuse:    m.MSpanInuse,
			MSpanSys:      m.MSpanSys,
			MCacheInuse:   m.MCacheInuse,
			MCacheSys:     m.MCacheSys,
			BuckHashSys:   m.BuckHashSys,
			GCSys:         m.GCSys,
			NextGC:        m.NextGC,
			LastGC:        m.LastGC,
			NumGC:         m.NumGC,
			GCPauseTotal:  m.PauseTotalNs,
		},
	}
}

// CleanupOldProfiles 清理旧的采集文件
func (p *Profiler) CleanupOldProfiles(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	// 查找要删除的记录
	var profiles []Profile
	err := p.db.Where("created_at < ?", cutoff).Find(&profiles).Error
	if err != nil {
		return 0, err
	}

	// 删除文件和记录
	for _, profile := range profiles {
		if profile.FilePath != "" {
			os.Remove(profile.FilePath)
		}
		if profile.FlamePath != "" {
			os.Remove(profile.FlamePath)
		}
	}

	// 批量删除记录
	result := p.db.Where("created_at < ?", cutoff).Delete(&Profile{})

	return result.RowsAffected, result.Error
}
