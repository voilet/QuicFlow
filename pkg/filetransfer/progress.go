package filetransfer

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// ProgressTracker 进度追踪器
type ProgressTracker struct {
	mu          sync.RWMutex
	progress    map[string]*TransferProgress
	subscribers map[string][]chan ProgressUpdate
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewProgressTracker 创建进度追踪器
func NewProgressTracker() *ProgressTracker {
	ctx, cancel := context.WithCancel(context.Background())
	pt := &ProgressTracker{
		progress:    make(map[string]*TransferProgress),
		subscribers: make(map[string][]chan ProgressUpdate),
		ctx:         ctx,
		cancel:      cancel,
	}
	go pt.cleanupLoop()
	return pt
}

// TransferProgress 传输进度详情
type TransferProgress struct {
	TaskID       string
	Status       TaskStatus
	Transferred  int64
	Total        int64
	Speed        int64 // bytes/sec
	StartTime    time.Time
	UpdateTime   time.Time
	prevBytes    int64
	prevTime     time.Time
	speedWindow  []int64 // 用于计算平均速度
	windowSize   int
}

// NewTransferProgress 创建传输进度
func NewTransferProgress(taskID string, total int64) *TransferProgress {
	now := time.Now()
	return &TransferProgress{
		TaskID:      taskID,
		Status:      TaskStatusPending,
		Total:       total,
		StartTime:   now,
		UpdateTime:  now,
		prevTime:    now,
		speedWindow: make([]int64, 0, 10),
		windowSize:  10,
	}
}

// Create 创建进度追踪
func (pt *ProgressTracker) Create(taskID string, total int64) *TransferProgress {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress := NewTransferProgress(taskID, total)
	pt.progress[taskID] = progress
	return progress
}

// Get 获取进度
func (pt *ProgressTracker) Get(taskID string) (*TransferProgress, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress, ok := pt.progress[taskID]
	return progress, ok
}

// Update 更新传输进度
func (pt *ProgressTracker) Update(taskID string, transferred int64, status TaskStatus) {
	pt.mu.Lock()

	progress, ok := pt.progress[taskID]
	if !ok {
		pt.mu.Unlock()
		return
	}

	now := time.Now()
	progress.Transferred = transferred
	progress.Status = status
	progress.UpdateTime = now

	// 计算速度
	if !progress.prevTime.IsZero() {
		elapsed := now.Sub(progress.prevTime).Seconds()
		if elapsed > 0 {
			bytesDelta := transferred - progress.prevBytes
			instantSpeed := int64(float64(bytesDelta) / elapsed)

			// 使用滑动窗口计算平均速度
			progress.speedWindow = append(progress.speedWindow, instantSpeed)
			if len(progress.speedWindow) > progress.windowSize {
				progress.speedWindow = progress.speedWindow[1:]
			}

			// 计算平均速度
			if len(progress.speedWindow) > 0 {
				var sum int64
				for _, s := range progress.speedWindow {
					sum += s
				}
				progress.Speed = sum / int64(len(progress.speedWindow))
			}
		}
	}

	progress.prevBytes = transferred
	progress.prevTime = now

	// 创建进度更新（在持有锁时复制数据）
	update := pt.makeProgressUpdate(progress)

	// 获取订阅者通道列表（在持有锁时）
	subs := pt.subscribers[taskID]
	subsCopy := make([]chan ProgressUpdate, len(subs))
	copy(subsCopy, subs)

	pt.mu.Unlock()

	// 通知订阅者（在释放锁后）
	pt.notifySubscribers(subsCopy, update)
}

// SetStatus 设置状态
func (pt *ProgressTracker) SetStatus(taskID string, status TaskStatus) {
	pt.mu.Lock()

	progress, ok := pt.progress[taskID]
	if !ok {
		pt.mu.Unlock()
		return
	}

	progress.Status = status

	// 创建进度更新（在持有锁时复制数据）
	update := pt.makeProgressUpdate(progress)

	// 获取订阅者通道列表（在持有锁时）
	subs := pt.subscribers[taskID]
	subsCopy := make([]chan ProgressUpdate, len(subs))
	copy(subsCopy, subs)

	pt.mu.Unlock()

	// 通知订阅者（在释放锁后）
	pt.notifySubscribers(subsCopy, update)
}

// SetComplete 设置完成
func (pt *ProgressTracker) SetComplete(taskID string) {
	pt.Update(taskID, -1, TaskStatusCompleted)
}

// SetError 设置错误
func (pt *ProgressTracker) SetError(taskID string) {
	pt.SetStatus(taskID, TaskStatusFailed)
}

// Remove 移除进度追踪
func (pt *ProgressTracker) Remove(taskID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	delete(pt.progress, taskID)
	pt.unsubscribeAll(taskID)
}

// GetProgressUpdate 获取进度更新
func (pt *ProgressTracker) GetProgressUpdate(taskID string) (ProgressUpdate, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress, ok := pt.progress[taskID]
	if !ok {
		return ProgressUpdate{}, false
	}

	update := ProgressUpdate{
		TaskID:      progress.TaskID,
		Status:      progress.Status,
		Transferred: progress.Transferred,
		Total:       progress.Total,
		UpdatedAt:   progress.UpdateTime,
	}

	// 计算进度百分比
	if progress.Total > 0 {
		update.Progress = float64(progress.Transferred) / float64(progress.Total) * 100
	}

	// 格式化速度
	update.Speed = formatSpeed(progress.Speed)

	// 计算 ETA
	if progress.Speed > 0 && progress.Total > 0 {
		remaining := progress.Total - progress.Transferred
		seconds := remaining / progress.Speed
		update.ETA = formatDuration(time.Duration(seconds) * time.Second)
	}

	return update, true
}

// makeProgressUpdate 创建进度更新（必须在持有锁时调用）
func (pt *ProgressTracker) makeProgressUpdate(progress *TransferProgress) ProgressUpdate {
	update := ProgressUpdate{
		TaskID:      progress.TaskID,
		Status:      progress.Status,
		Transferred: progress.Transferred,
		Total:       progress.Total,
		UpdatedAt:   progress.UpdateTime,
	}

	// 计算进度百分比
	if progress.Total > 0 {
		update.Progress = float64(progress.Transferred) / float64(progress.Total) * 100
	}

	// 格式化速度
	update.Speed = formatSpeed(progress.Speed)

	// 计算 ETA
	if progress.Speed > 0 && progress.Total > 0 {
		remaining := progress.Total - progress.Transferred
		seconds := remaining / progress.Speed
		update.ETA = formatDuration(time.Duration(seconds) * time.Second)
	}

	return update
}

// Subscribe 订阅进度更新
func (pt *ProgressTracker) Subscribe(taskID string) <-chan ProgressUpdate {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	ch := make(chan ProgressUpdate, 10)
	pt.subscribers[taskID] = append(pt.subscribers[taskID], ch)
	return ch
}

// Unsubscribe 取消订阅
func (pt *ProgressTracker) Unsubscribe(taskID string, ch <-chan ProgressUpdate) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if subs, ok := pt.subscribers[taskID]; ok {
		for i, sub := range subs {
			if sub == ch {
				pt.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
				close(sub)
				break
			}
		}
	}
}

// unsubscribeAll 取消所有订阅
func (pt *ProgressTracker) unsubscribeAll(taskID string) {
	if subs, ok := pt.subscribers[taskID]; ok {
		for _, ch := range subs {
			close(ch)
		}
		delete(pt.subscribers, taskID)
	}
}

// notifySubscribers 通知订阅者（必须在未持有锁时调用）
func (pt *ProgressTracker) notifySubscribers(subs []chan ProgressUpdate, update ProgressUpdate) {
	for _, ch := range subs {
		select {
		case ch <- update:
		default:
			// 通道阻塞，跳过
		}
	}
}

// cleanupLoop 清理循环
func (pt *ProgressTracker) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pt.cleanupOldProgress()
		case <-pt.ctx.Done():
			return
		}
	}
}

// cleanupOldProgress 清理旧进度
func (pt *ProgressTracker) cleanupOldProgress() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	for taskID, progress := range pt.progress {
		// 清理已完成且超过1小时的进度
		if progress.Status == TaskStatusCompleted ||
			progress.Status == TaskStatusFailed ||
			progress.Status == TaskStatusCancelled {
			if now.Sub(progress.UpdateTime) > time.Hour {
				delete(pt.progress, taskID)
				pt.unsubscribeAll(taskID)
			}
		}
	}
}

// Close 关闭进度追踪器
func (pt *ProgressTracker) Close() {
	pt.cancel()
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for taskID := range pt.subscribers {
		pt.unsubscribeAll(taskID)
	}
}

// formatSpeed 格式化速度
func formatSpeed(bytesPerSec int64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
	const unit = 1024
	div, exp := int64(unit), 0
	for n := bytesPerSec / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB/s", float64(bytesPerSec)/float64(div), "KMGTPE"[exp])
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%02d:%02d", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// ProgressWriter 进度写入器
type ProgressWriter struct {
	taskID      string
	tracker     *ProgressTracker
	total       int64
	written     int64
	updateInterval time.Duration
	lastUpdate  time.Time
}

// NewProgressWriter 创建进度写入器
func NewProgressWriter(taskID string, tracker *ProgressTracker, total int64) *ProgressWriter {
	return &ProgressWriter{
		taskID:         taskID,
		tracker:        tracker,
		total:          total,
		updateInterval: 500 * time.Millisecond,
		lastUpdate:     time.Now(),
	}
}

// Write 实现 io.Writer
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	// 限制更新频率
	now := time.Now()
	if now.Sub(pw.lastUpdate) >= pw.updateInterval {
		pw.tracker.Update(pw.taskID, pw.written, TaskStatusTransferring)
		pw.lastUpdate = now
	}

	return n, nil
}

// Close 完成写入
func (pw *ProgressWriter) Close() {
	pw.tracker.Update(pw.taskID, pw.written, TaskStatusCompleted)
}

// ProgressReader 进度读取器
type ProgressReader struct {
	taskID      string
	tracker     *ProgressTracker
	total       int64
	read        int64
	reader      io.Reader
	updateInterval time.Duration
	lastUpdate  time.Time
}

// NewProgressReader 创建进度读取器
func NewProgressReader(taskID string, tracker *ProgressTracker, total int64, reader io.Reader) *ProgressReader {
	return &ProgressReader{
		taskID:         taskID,
		tracker:        tracker,
		total:          total,
		reader:         reader,
		updateInterval: 500 * time.Millisecond,
		lastUpdate:     time.Now(),
	}
}

// Read 实现 io.Reader
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	// 限制更新频率
	now := time.Now()
	if now.Sub(pr.lastUpdate) >= pr.updateInterval {
		pr.tracker.Update(pr.taskID, pr.read, TaskStatusTransferring)
		pr.lastUpdate = now
	}

	return n, err
}

// Close 完成读取
func (pr *ProgressReader) Close() {
	pr.tracker.Update(pr.taskID, pr.read, TaskStatusCompleted)
}
