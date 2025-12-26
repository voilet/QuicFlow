package handlers

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/monitoring"
)

// StatusHandler 状态查询处理器
type StatusHandler struct {
	logger    *monitoring.Logger
	startTime time.Time
	version   string
}

// NewStatusHandler 创建状态处理器
func NewStatusHandler(logger *monitoring.Logger, version string) *StatusHandler {
	return &StatusHandler{
		logger:    logger,
		startTime: time.Now(),
		version:   version,
	}
}

// Handle 处理get_status命令
func (h *StatusHandler) Handle(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	hostname, _ := os.Hostname()

	// 使用共享类型
	result := command.StatusResult{
		Status:       "running",
		Uptime:       int64(time.Since(h.startTime).Seconds()),
		Version:      h.version,
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}

	h.logger.Debug("Status query handled", "uptime", result.Uptime)

	return json.Marshal(result)
}
