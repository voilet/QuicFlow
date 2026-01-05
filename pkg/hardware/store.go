package hardware

import (
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store 硬件信息存储
type Store struct {
	db *gorm.DB
}

// NewStore 创建硬件信息存储
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// SaveHardwareInfo 保存硬件信息（Upsert: 按 client_id 去重，存在则更新，不存在则插入）
func (s *Store) SaveHardwareInfo(clientID string, info *command.HardwareInfoResult) (*Device, error) {
	now := time.Now()

	// 构建 Device 对象（只填充简化后的冗余字段）
	device := &Device{
		ClientID:     clientID,
		Status:       string(DeviceStatusOnline),
		CPUModel:     info.ModelName,
		MemoryTotalGB: info.Memory.TotalGB,
		DiskTotalTB:   info.TotalDiskCapacityTBDecimal,
		PrimaryMAC:    formatMAC(info.MAC),
		Hostname:      info.Host.Hostname,
		OS:            info.Host.OS,
		KernelArch:    info.Host.KernelArch,

		// 完整硬件信息（JSONB）
		FullHardwareInfo: HardwareInfoResultJSONB(*info),

		// 时间
		LastSeenAt: &now,
	}

	// 执行 Upsert（存在则更新，不存在则插入）
	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"cpu_model", "memory_total_gb", "disk_total_tb",
			"primary_mac", "hostname", "os", "kernel_arch",
			"full_hardware_info",
			"status", "last_seen_at", "updated_at",
		}),
	}).Create(device).Error

	if err != nil {
		return nil, fmt.Errorf("failed to save hardware info: %w", err)
	}

	// 检查是否需要记录变更历史（新设备）
	if err := s.recordChangeHistoryIfNeeded(clientID, info); err != nil {
		// 记录历史失败不影响主流程
		fmt.Printf("Warning: failed to record hardware history: %v\n", err)
	}

	return device, nil
}

// recordChangeHistoryIfNeeded 检测硬件变更并记录历史
func (s *Store) recordChangeHistoryIfNeeded(clientID string, info *command.HardwareInfoResult) error {
	// 获取上一次的硬件信息
	var existing Device
	err := s.db.Where("client_id = ?", clientID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 新设备，记录创建历史
			return s.createHistory(clientID, "created", "New device registered", info)
		}
		return err
	}

	// 检查是否有重大变更（CPU、内存、磁盘）
	changeType := ""
	changeDesc := ""

	// CPU 变更
	if info.ModelName != existing.CPUModel {
		changeType = "cpu_changed"
		changeDesc = fmt.Sprintf("CPU changed from %s to %s", existing.CPUModel, info.ModelName)
	}

	// 内存变更
	if changeType == "" && info.Memory.TotalGB != existing.MemoryTotalGB {
		changeType = "memory_changed"
		changeDesc = fmt.Sprintf("Memory changed from %.0fGB to %.0fGB", existing.MemoryTotalGB, info.Memory.TotalGB)
	}

	// 磁盘变更
	if changeType == "" && info.TotalDiskCapacityTBDecimal != existing.DiskTotalTB {
		changeType = "disk_changed"
		changeDesc = fmt.Sprintf("Disk changed from %.2fTB to %.2fTB", existing.DiskTotalTB, info.TotalDiskCapacityTBDecimal)
	}

	// 如果有变更，记录历史
	if changeType != "" {
		return s.createHistory(clientID, changeType, changeDesc, info)
	}

	return nil
}

// createHistory 创建硬件变更历史记录
func (s *Store) createHistory(clientID, changeType, changeDesc string, info *command.HardwareInfoResult) error {
	history := &DeviceHardwareHistory{
		ClientID:          clientID,
		ChangeType:        changeType,
		ChangeDescription: changeDesc,
		HardwareInfo:      HardwareInfoResultJSONB(*info),
	}
	return s.db.Create(history).Error
}

// GetDeviceByClientID 根据 client_id 获取设备信息
func (s *Store) GetDeviceByClientID(clientID string) (*Device, error) {
	var device Device
	err := s.db.Where("client_id = ?", clientID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// ListDevices 列出所有设备
func (s *Store) ListDevices(offset, limit int) ([]Device, int64, error) {
	var devices []Device
	var total int64

	query := s.db.Model(&Device{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询（limit=0 表示全部，不传 Limit）
	query = query.Order("last_seen_at DESC").Offset(offset)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&devices).Error
	return devices, total, err
}

// ListDevicesByStatus 按状态列出设备
func (s *Store) ListDevicesByStatus(status string, offset, limit int) ([]Device, int64, error) {
	var devices []Device
	var total int64

	query := s.db.Model(&Device{}).Where("status = ?", status)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询（limit=0 表示全部，不传 Limit）
	query = query.Order("last_seen_at DESC").Offset(offset)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&devices).Error
	return devices, total, err
}

// UpdateDeviceStatus 更新设备状态
func (s *Store) UpdateDeviceStatus(clientID string, status DeviceStatus) error {
	return s.db.Model(&Device{}).
		Where("client_id = ?", clientID).
		Update("status", string(status)).Error
}

// UpdateLastSeenTime 更新最后在线时间
func (s *Store) UpdateLastSeenTime(clientID string) error {
	now := time.Now()
	return s.db.Model(&Device{}).
		Where("client_id = ?", clientID).
		Updates(map[string]interface{}{
			"last_seen_at": &now,
			"status":       string(DeviceStatusOnline),
			"updated_at":   now,
		}).Error
}

// MarkOfflineDevices 标记超时设备为离线
func (s *Store) MarkOfflineDevices(timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	result := s.db.Model(&Device{}).
		Where("status = ? AND last_seen_at < ?", string(DeviceStatusOnline), cutoff).
		Update("status", string(DeviceStatusOffline))
	return result.RowsAffected, result.Error
}

// GetDeviceHistory 获取设备硬件变更历史
func (s *Store) GetDeviceHistory(clientID string, limit int) ([]DeviceHardwareHistory, error) {
	var history []DeviceHardwareHistory
	err := s.db.Where("client_id = ?", clientID).
		Order("recorded_at DESC").
		Limit(limit).
		Find(&history).Error
	return history, err
}

// DeleteDevice 删除设备（软删除）
func (s *Store) DeleteDevice(clientID string) error {
	return s.db.Where("client_id = ?", clientID).Delete(&Device{}).Error
}

// formatMAC 格式化 MAC 地址（统一为大写带冒号格式）
func formatMAC(mac string) string {
	if mac == "" {
		return ""
	}
	// 如果已经是带分隔符的格式，保持原样
	if len(mac) == 17 { // AA:BB:CC:DD:EE:FF
		return mac
	}
	// 如果是无分隔符格式，添加冒号
	if len(mac) == 12 { // AABBCCDDEEFF
		return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6] + ":" + mac[6:8] + ":" + mac[8:10] + ":" + mac[10:12]
	}
	return mac
}

// SearchDevicesByHostname 按主机名搜索设备（模糊匹配）
func (s *Store) SearchDevicesByHostname(keyword string, offset, limit int) ([]Device, int64, error) {
	var devices []Device
	var total int64

	query := s.db.Model(&Device{}).Where("hostname ILIKE ?", "%"+keyword+"%")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	err := query.Order("hostname ASC").Offset(offset).Limit(limit).Find(&devices).Error
	return devices, total, err
}

// GetDeviceByMAC 根据 MAC 地址获取设备
func (s *Store) GetDeviceByMAC(mac string) (*Device, error) {
	var device Device
	err := s.db.Where("primary_mac = ?", mac).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceStats 获取设备统计信息
func (s *Store) GetDeviceStats() (*DeviceStats, error) {
	stats := &DeviceStats{}

	// 总设备数
	if err := s.db.Model(&Device{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// 在线设备数
	if err := s.db.Model(&Device{}).Where("status = ?", string(DeviceStatusOnline)).Count(&stats.Online).Error; err != nil {
		return nil, err
	}

	// 离线设备数
	if err := s.db.Model(&Device{}).Where("status = ?", string(DeviceStatusOffline)).Count(&stats.Offline).Error; err != nil {
		return nil, err
	}

	// 未知状态设备数
	stats.Unknown = stats.Total - stats.Online - stats.Offline

	return stats, nil
}

// DeviceStats 设备统计信息
type DeviceStats struct {
	Total   int64 `json:"total"`
	Online  int64 `json:"online"`
	Offline int64 `json:"offline"`
	Unknown int64 `json:"unknown"`
}
