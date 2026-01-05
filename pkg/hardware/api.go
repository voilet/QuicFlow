package hardware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/voilet/quic-flow/pkg/command"
)

// Handler 硬件信息 API 处理器
type Handler struct {
	store *Store
}

// NewHandler 创建硬件信息 API 处理器
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	api := r.Group("/hardware")
	{
		// 设备列表
		api.GET("/devices", h.ListDevices)
		api.GET("/devices/stats", h.GetDeviceStats)

		// 单个设备
		api.GET("/devices/:client_id", h.GetDevice)
		api.GET("/devices/:client_id/hardware", h.GetHardwareInfo)
		api.PUT("/devices/:client_id/status", h.UpdateDeviceStatus)
		api.DELETE("/devices/:client_id", h.DeleteDevice)

		// 设备搜索
		api.GET("/devices/search/by-hostname", h.SearchByHostname)
		api.GET("/devices/search/by-mac/:mac", h.GetByMAC)

		// 设备历史
		api.GET("/devices/:client_id/history", h.GetDeviceHistory)

		// 批量操作
		api.POST("/devices/mark-offline", h.MarkOfflineDevices)
	}
}

// ListDevicesRequest 设备列表请求
type ListDevicesRequest struct {
	Status string `form:"status"`
	Page   int    `form:"page"`
	PageSize int  `form:"page_size"`
}

// ListDevicesResponse 设备列表响应
type ListDevicesResponse struct {
	Success bool     `json:"success"`
	Total   int64    `json:"total"`
	Page    int      `json:"page"`
	Devices []Device `json:"devices"`
	Message string   `json:"message,omitempty"`
}

// ListDevices 获取设备列表
// GET /hardware/devices?status=online&page=1&page_size=20
func (h *Handler) ListDevices(c *gin.Context) {
	var req ListDevicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ListDevicesResponse{
			Success: false,
			Message: "Invalid request parameters",
		})
		return
	}

	// 默认分页
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize

	var devices []Device
	var total int64
	var err error

	if req.Status != "" {
		devices, total, err = h.store.ListDevicesByStatus(req.Status, offset, req.PageSize)
	} else {
		devices, total, err = h.store.ListDevices(offset, req.PageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ListDevicesResponse{
			Success: false,
			Message: "Failed to list devices",
		})
		return
	}

	c.JSON(http.StatusOK, ListDevicesResponse{
		Success: true,
		Total:   total,
		Page:    req.Page,
		Devices: devices,
	})
}

// GetDeviceResponse 获取单个设备响应
type GetDeviceResponse struct {
	Success bool        `json:"success"`
	Device  *Device     `json:"device,omitempty"`
	Message string      `json:"message,omitempty"`
}

// GetDevice 获取单个设备信息
// GET /hardware/devices/:client_id
func (h *Handler) GetDevice(c *gin.Context) {
	clientID := c.Param("client_id")

	device, err := h.store.GetDeviceByClientID(clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetDeviceResponse{
			Success: false,
			Message: "Device not found",
		})
		return
	}

	c.JSON(http.StatusOK, GetDeviceResponse{
		Success: true,
		Device:  device,
	})
}

// GetHardwareInfoResponse 获取硬件信息响应
type GetHardwareInfoResponse struct {
	Success bool                       `json:"success"`
	HardwareInfo *command.HardwareInfoResult `json:"hardware_info,omitempty"`
	Message string                     `json:"message,omitempty"`
}

// GetHardwareInfo 获取设备硬件信息（从数据库）
// GET /hardware/devices/:client_id/hardware
func (h *Handler) GetHardwareInfo(c *gin.Context) {
	clientID := c.Param("client_id")

	device, err := h.store.GetDeviceByClientID(clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, GetHardwareInfoResponse{
			Success: false,
			Message: "Device not found",
		})
		return
	}

	// 从 FullHardwareInfo JSONB 字段中提取硬件信息
	hwInfo := command.HardwareInfoResult(device.FullHardwareInfo)

	c.JSON(http.StatusOK, GetHardwareInfoResponse{
		Success: true,
		HardwareInfo: &hwInfo,
	})
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateStatusResponse 更新状态响应
type UpdateStatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// UpdateDeviceStatus 更新设备状态
// PUT /hardware/devices/:client_id/status
func (h *Handler) UpdateDeviceStatus(c *gin.Context) {
	clientID := c.Param("client_id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, UpdateStatusResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// 验证状态
	if req.Status != string(DeviceStatusOnline) &&
	   req.Status != string(DeviceStatusOffline) &&
	   req.Status != string(DeviceStatusUnknown) {
		c.JSON(http.StatusBadRequest, UpdateStatusResponse{
			Success: false,
			Message: "Invalid status. Must be online, offline, or unknown",
		})
		return
	}

	if err := h.store.UpdateDeviceStatus(clientID, DeviceStatus(req.Status)); err != nil {
		c.JSON(http.StatusInternalServerError, UpdateStatusResponse{
			Success: false,
			Message: "Failed to update device status",
		})
		return
	}

	c.JSON(http.StatusOK, UpdateStatusResponse{
		Success: true,
		Message: "Device status updated",
	})
}

// DeleteDeviceResponse 删除设备响应
type DeleteDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// DeleteDevice 删除设备
// DELETE /hardware/devices/:client_id
func (h *Handler) DeleteDevice(c *gin.Context) {
	clientID := c.Param("client_id")

	if err := h.store.DeleteDevice(clientID); err != nil {
		c.JSON(http.StatusInternalServerError, DeleteDeviceResponse{
			Success: false,
			Message: "Failed to delete device",
		})
		return
	}

	c.JSON(http.StatusOK, DeleteDeviceResponse{
		Success: true,
		Message: "Device deleted",
	})
}

// SearchByHostnameResponse 按主机名搜索响应
type SearchByHostnameResponse struct {
	Success bool     `json:"success"`
	Total   int64    `json:"total"`
	Devices []Device `json:"devices"`
	Message string   `json:"message,omitempty"`
}

// SearchByHostname 按主机名搜索设备
// GET /hardware/devices/search/by-hostname?q=server&page=1&page_size=20
func (h *Handler) SearchByHostname(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, SearchByHostnameResponse{
			Success: false,
			Message: "Missing search keyword",
		})
		return
	}

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	p, _ := strconv.Atoi(page)
	ps, _ := strconv.Atoi(pageSize)

	if p < 1 {
		p = 1
	}
	if ps < 1 || ps > 100 {
		ps = 20
	}
	offset := (p - 1) * ps

	devices, total, err := h.store.SearchDevicesByHostname(keyword, offset, ps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SearchByHostnameResponse{
			Success: false,
			Message: "Search failed",
		})
		return
	}

	c.JSON(http.StatusOK, SearchByHostnameResponse{
		Success: true,
		Total:   total,
		Devices: devices,
	})
}

// GetByMACResponse 按 MAC 获取设备响应
type GetByMACResponse struct {
	Success bool    `json:"success"`
	Device  *Device `json:"device,omitempty"`
	Message string  `json:"message,omitempty"`
}

// GetByMAC 根据 MAC 地址获取设备
// GET /hardware/devices/search/by-mac/:mac
func (h *Handler) GetByMAC(c *gin.Context) {
	mac := c.Param("mac")

	device, err := h.store.GetDeviceByMAC(mac)
	if err != nil {
		c.JSON(http.StatusNotFound, GetByMACResponse{
			Success: false,
			Message: "Device not found",
		})
		return
	}

	c.JSON(http.StatusOK, GetByMACResponse{
		Success: true,
		Device:  device,
	})
}

// GetDeviceHistoryResponse 获取设备历史响应
type GetDeviceHistoryResponse struct {
	Success bool                        `json:"success"`
	History []DeviceHardwareHistory     `json:"history"`
	Message string                      `json:"message,omitempty"`
}

// GetDeviceHistory 获取设备硬件变更历史
// GET /hardware/devices/:client_id/history?limit=50
func (h *Handler) GetDeviceHistory(c *gin.Context) {
	clientID := c.Param("client_id")
	limit := c.DefaultQuery("limit", "50")

	lim, _ := strconv.Atoi(limit)
	if lim < 1 || lim > 500 {
		lim = 50
	}

	history, err := h.store.GetDeviceHistory(clientID, lim)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GetDeviceHistoryResponse{
			Success: false,
			Message: "Failed to get device history",
		})
		return
	}

	c.JSON(http.StatusOK, GetDeviceHistoryResponse{
		Success: true,
		History: history,
	})
}

// MarkOfflineDevicesRequest 标记离线设备请求
type MarkOfflineDevicesRequest struct {
	TimeoutMinutes int `json:"timeout_minutes" binding:"required,min=1"`
}

// MarkOfflineDevicesResponse 标记离线设备响应
type MarkOfflineDevicesResponse struct {
	Success    bool   `json:"success"`
	MarkedCount int64  `json:"marked_count"`
	Message    string `json:"message,omitempty"`
}

// MarkOfflineDevices 标记超时设备为离线
// POST /hardware/devices/mark-offline
func (h *Handler) MarkOfflineDevices(c *gin.Context) {
	var req MarkOfflineDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MarkOfflineDevicesResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	timeout := time.Duration(req.TimeoutMinutes) * time.Minute
	count, err := h.store.MarkOfflineDevices(timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, MarkOfflineDevicesResponse{
			Success: false,
			Message: "Failed to mark offline devices",
		})
		return
	}

	c.JSON(http.StatusOK, MarkOfflineDevicesResponse{
		Success:    true,
		MarkedCount: count,
		Message:    "Marked " + strconv.FormatInt(count, 10) + " devices as offline",
	})
}

// GetDeviceStatsResponse 获取设备统计响应
type GetDeviceStatsResponse struct {
	Success bool        `json:"success"`
	Stats   *DeviceStats `json:"stats,omitempty"`
	Message string      `json:"message,omitempty"`
}

// GetDeviceStats 获取设备统计信息
// GET /hardware/devices/stats
func (h *Handler) GetDeviceStats(c *gin.Context) {
	stats, err := h.store.GetDeviceStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, GetDeviceStatsResponse{
			Success: false,
			Message: "Failed to get device stats",
		})
		return
	}

	c.JSON(http.StatusOK, GetDeviceStatsResponse{
		Success: true,
		Stats:   stats,
	})
}

// SaveHardwareInfo 保存硬件信息（供命令处理器调用）
// clientID: 客户端ID
// info: 硬件信息
func (h *Handler) SaveHardwareInfo(clientID string, info *command.HardwareInfoResult) (*Device, error) {
	return h.store.SaveHardwareInfo(clientID, info)
}

// UpdateLastSeenTime 更新最后在线时间（供心跳/连接时调用）
func (h *Handler) UpdateLastSeenTime(clientID string) error {
	return h.store.UpdateLastSeenTime(clientID)
}
