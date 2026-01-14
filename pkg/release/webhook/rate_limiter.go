package webhook

import (
	"sync"
	"time"
)

// RateLimiter Webhook速率限制器
type RateLimiter struct {
	mu        sync.RWMutex
	requests  map[string]*requestWindow // key: webhookID or IP
	maxReqs   int                       // 时间窗口内最大请求数
	window    time.Duration             // 时间窗口
	cleanTick time.Duration             // 清理间隔
	stopCh    chan struct{}
}

// requestWindow 请求时间窗口
type requestWindow struct {
	timestamps []time.Time
	blocked    bool      // 是否被临时封禁
	blockedAt  time.Time // 封禁时间
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	MaxRequests int           `json:"max_requests"` // 最大请求数
	Window      time.Duration `json:"window"`       // 时间窗口
	BlockTime   time.Duration `json:"block_time"`   // 超限后封禁时间
}

// DefaultRateLimitConfig 默认配置
var DefaultRateLimitConfig = RateLimitConfig{
	MaxRequests: 60,              // 每分钟60次
	Window:      time.Minute,     // 1分钟窗口
	BlockTime:   5 * time.Minute, // 超限封禁5分钟
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.MaxRequests <= 0 {
		config.MaxRequests = DefaultRateLimitConfig.MaxRequests
	}
	if config.Window <= 0 {
		config.Window = DefaultRateLimitConfig.Window
	}
	if config.BlockTime <= 0 {
		config.BlockTime = DefaultRateLimitConfig.BlockTime
	}

	rl := &RateLimiter{
		requests:  make(map[string]*requestWindow),
		maxReqs:   config.MaxRequests,
		window:    config.Window,
		cleanTick: config.Window * 2, // 清理间隔为窗口的2倍
		stopCh:    make(chan struct{}),
	}

	// 启动清理协程
	go rl.cleaner()

	return rl
}

// Allow 检查是否允许请求
// 返回: 是否允许, 剩余配额, 重置时间
func (rl *RateLimiter) Allow(key string) (allowed bool, remaining int, resetAt time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 获取或创建请求窗口
	window, exists := rl.requests[key]
	if !exists {
		window = &requestWindow{
			timestamps: make([]time.Time, 0, rl.maxReqs),
		}
		rl.requests[key] = window
	}

	// 检查是否被封禁
	if window.blocked {
		if now.Sub(window.blockedAt) < DefaultRateLimitConfig.BlockTime {
			// 仍在封禁期
			resetAt = window.blockedAt.Add(DefaultRateLimitConfig.BlockTime)
			return false, 0, resetAt
		}
		// 解除封禁
		window.blocked = false
		window.timestamps = nil
	}

	// 清理过期的时间戳
	validTimestamps := make([]time.Time, 0, len(window.timestamps))
	for _, ts := range window.timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	window.timestamps = validTimestamps

	// 计算剩余配额
	remaining = rl.maxReqs - len(window.timestamps)
	resetAt = now.Add(rl.window)

	// 检查是否超限
	if len(window.timestamps) >= rl.maxReqs {
		// 超限，启动封禁
		window.blocked = true
		window.blockedAt = now
		return false, 0, now.Add(DefaultRateLimitConfig.BlockTime)
	}

	// 记录请求
	window.timestamps = append(window.timestamps, now)
	remaining--

	return true, remaining, resetAt
}

// GetStats 获取指定key的统计信息
func (rl *RateLimiter) GetStats(key string) (requestCount int, blocked bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	window, exists := rl.requests[key]
	if !exists {
		return 0, false
	}

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 统计有效请求数
	count := 0
	for _, ts := range window.timestamps {
		if ts.After(windowStart) {
			count++
		}
	}

	return count, window.blocked
}

// Reset 重置指定key的计数
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.requests, key)
}

// Stop 停止速率限制器
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// cleaner 定期清理过期数据
func (rl *RateLimiter) cleaner() {
	ticker := time.NewTicker(rl.cleanTick)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup 清理过期数据
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	for key, window := range rl.requests {
		// 清理过期时间戳
		validTimestamps := make([]time.Time, 0)
		for _, ts := range window.timestamps {
			if ts.After(windowStart) {
				validTimestamps = append(validTimestamps, ts)
			}
		}
		window.timestamps = validTimestamps

		// 如果封禁已过期且无有效请求，删除记录
		if window.blocked && now.Sub(window.blockedAt) >= DefaultRateLimitConfig.BlockTime {
			window.blocked = false
		}

		if !window.blocked && len(window.timestamps) == 0 {
			delete(rl.requests, key)
		}
	}
}

// 全局速率限制器实例
var globalRateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// GetGlobalRateLimiter 获取全局速率限制器
func GetGlobalRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		globalRateLimiter = NewRateLimiter(DefaultRateLimitConfig)
	})
	return globalRateLimiter
}
