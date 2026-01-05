package hardware

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"gorm.io/gorm"
)

// ==================== 主模型 ====================

// Device 设备基本信息表
// 按 client_id 去重，存在则更新，不存在则插入
type Device struct {
	// 主键
	ID string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`

	// 业务标识（唯一键）
	ClientID string `gorm:"uniqueIndex;size:100;not null" json:"client_id"`

	// ========== 冗余字段（用于快速查询/排序，JSONB查询较慢） ==========
	CPUModel         string  `gorm:"size:255;column:cpu_model;index" json:"cpu_model"`         // CPU 型号（用于搜索）
	MemoryTotalGB    float64 `gorm:"type:decimal(10,2);column:memory_total_gb" json:"memory_total_gb"` // 内存GB（用于排序）
	DiskTotalTB      float64 `gorm:"type:decimal(10,2);column:disk_total_tb" json:"disk_total_tb"`     // 磁盘TB（用于排序）
	PrimaryMAC       string  `gorm:"size:17;column:primary_mac;index" json:"primary_mac"`       // 主MAC（用于查询）
	Hostname         string  `gorm:"size:255;column:hostname;index" json:"hostname"`           // 主机名（用于搜索）
	OS               string  `gorm:"size:50;column:os" json:"os"`                               // 操作系统
	KernelArch       string  `gorm:"size:50;column:kernel_arch" json:"kernel_arch"`             // 架构

	// ========== 完整硬件信息（JSONB，包含所有详情） ==========
	FullHardwareInfo HardwareInfoResultJSONB `gorm:"type:jsonb;column:full_hardware_info" json:"full_hardware_info,omitempty"`

	// ========== 状态与时间 ==========
	Status       string     `gorm:"size:20;default:'online';column:status;index" json:"status"`
	LastSeenAt   *time.Time `gorm:"column:last_seen_at" json:"last_seen_at,omitempty"`
	FirstSeenAt  time.Time  `gorm:"default:NOW();column:first_seen_at" json:"first_seen_at"`
	CreatedAt    time.Time  `gorm:"default:NOW();column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"default:NOW();column:updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}

// BeforeCreate GORM hook
func (d *Device) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now
	if d.FirstSeenAt.IsZero() {
		d.FirstSeenAt = now
	}
	return nil
}

// BeforeUpdate GORM hook
func (d *Device) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return nil
}

// DeviceStatus 设备状态
type DeviceStatus string

const (
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
	DeviceStatusUnknown DeviceStatus = "unknown"
)

// ==================== 历史模型 ====================

// DeviceHardwareHistory 硬件变更历史
type DeviceHardwareHistory struct {
	ID                string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ClientID          string    `gorm:"size:100;not null;index" json:"client_id"`
	ChangeType        string    `gorm:"size:50;not null" json:"change_type"`
	ChangeDescription string    `gorm:"type:text" json:"change_description,omitempty"`
	HardwareInfo      HardwareInfoResultJSONB `gorm:"type:jsonb;not null" json:"hardware_info"`
	RecordedAt        time.Time `gorm:"default:NOW();index" json:"recorded_at"`
}

func (DeviceHardwareHistory) TableName() string {
	return "device_hardware_history"
}

// ==================== JSONB 类型 ====================

// HardwareInfoResultJSONB 包装 HardwareInfoResult 用于 JSONB 存储
type HardwareInfoResultJSONB command.HardwareInfoResult

// Value 实现 driver.Valuer 接口
func (h HardwareInfoResultJSONB) Value() (driver.Value, error) {
	if len(h.DMI.BoardName) == 0 && len(h.Host.Hostname) == 0 {
		return nil, nil
	}
	return json.Marshal(h)
}

// Scan 实现 sql.Scanner 接口
func (h *HardwareInfoResultJSONB) Scan(value interface{}) error {
	if value == nil {
		*h = HardwareInfoResultJSONB{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, h)
}

// ==================== 数据库迁移 ====================

// AutoMigrateFixes 修复旧版本的列名问题
func AutoMigrateFixes(db *gorm.DB) error {
	// 检查是否存在旧的错误列名
	if db.Migrator().HasColumn(&Device{}, "cpu_frequency_m_hz") {
		// 重命名列
		if err := db.Migrator().RenameColumn(&Device{}, "cpu_frequency_m_hz", "cpu_frequency_mhz"); err != nil {
			return err
		}
	}
	return nil
}
