package recording

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// RecordingModel GORM model for terminal recordings
type RecordingModel struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SessionID   string    `gorm:"size:100;index;not null" json:"session_id"`
	ClientID    string    `gorm:"size:100;index;not null" json:"client_id"`
	Username    string    `gorm:"size:100;index;not null" json:"username"`
	Width       int       `gorm:"not null" json:"width"`
	Height      int       `gorm:"not null" json:"height"`
	Duration    float64   `gorm:"not null" json:"duration"` // seconds
	FileSize    int64     `gorm:"not null" json:"file_size"` // bytes
	FilePath    string    `gorm:"size:500" json:"file_path"` // 录制文件路径
	RecordInput bool      `gorm:"default:true" json:"record_input"`
	CreatedAt   time.Time `gorm:"index;not null" json:"created_at"`
}

func (RecordingModel) TableName() string {
	return "terminal_recordings"
}

// DBStore implements database storage for recordings
type DBStore struct {
	db *gorm.DB
}

// NewDBStore creates a new database-based recording store
func NewDBStore(db *gorm.DB) (*DBStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	store := &DBStore{
		db: db,
	}

	// Auto migrate the recording table
	if err := db.AutoMigrate(&RecordingModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate terminal_recordings table: %w", err)
	}

	return store, nil
}

// SaveRecording saves a recording metadata to database
func (s *DBStore) SaveRecording(ctx context.Context, meta *RecordingMeta, filePath string) error {
	model := &RecordingModel{
		ID:          meta.ID,
		SessionID:   meta.SessionID,
		ClientID:    meta.ClientID,
		Username:    meta.Username,
		Width:       meta.Width,
		Height:      meta.Height,
		Duration:    meta.Duration,
		FileSize:    meta.FileSize,
		FilePath:    filePath,
		RecordInput: meta.RecordInput,
		CreatedAt:   meta.CreatedAt,
	}

	return s.db.WithContext(ctx).Create(model).Error
}

// List returns all recordings matching the filter from database
func (s *DBStore) List(ctx context.Context, filter *RecordingFilter) ([]*RecordingMeta, error) {
	var models []RecordingModel
	query := s.db.WithContext(ctx).Model(&RecordingModel{})

	// Apply filters
	if filter != nil {
		if filter.SessionID != "" {
			query = query.Where("session_id = ?", filter.SessionID)
		}
		if filter.ClientID != "" {
			query = query.Where("client_id = ?", filter.ClientID)
		}
		if filter.Username != "" {
			query = query.Where("username = ?", filter.Username)
		}
		if filter.StartTime != nil {
			query = query.Where("created_at >= ?", *filter.StartTime)
		}
		if filter.EndTime != nil {
			query = query.Where("created_at <= ?", *filter.EndTime)
		}
	}

	// Order by created_at DESC (newest first)
	query = query.Order("created_at DESC")

	// Apply pagination
	if filter != nil {
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
		limit := filter.Limit
		if limit <= 0 {
			limit = 100
		}
		query = query.Limit(limit)
	} else {
		query = query.Limit(100)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	// Convert models to RecordingMeta
	recordings := make([]*RecordingMeta, len(models))
	for i, model := range models {
		recordings[i] = &RecordingMeta{
			ID:          model.ID,
			SessionID:   model.SessionID,
			ClientID:    model.ClientID,
			Username:    model.Username,
			Width:       model.Width,
			Height:      model.Height,
			Duration:    model.Duration,
			FileSize:    model.FileSize,
			CreatedAt:   model.CreatedAt,
			RecordInput: model.RecordInput,
		}
	}

	return recordings, nil
}

// Get returns a specific recording by ID from database
func (s *DBStore) Get(ctx context.Context, id string) (*RecordingMeta, error) {
	var model RecordingModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}

	return &RecordingMeta{
		ID:          model.ID,
		SessionID:   model.SessionID,
		ClientID:    model.ClientID,
		Username:    model.Username,
		Width:       model.Width,
		Height:      model.Height,
		Duration:    model.Duration,
		FileSize:    model.FileSize,
		CreatedAt:   model.CreatedAt,
		RecordInput: model.RecordInput,
	}, nil
}

// GetFilePath returns the file path for a recording from database
func (s *DBStore) GetFilePath(ctx context.Context, id string) (string, error) {
	var model RecordingModel
	if err := s.db.WithContext(ctx).Where("id = ?", id).Select("file_path").First(&model).Error; err != nil {
		return "", err
	}
	return model.FilePath, nil
}

// GetStats returns statistics about recordings from database
func (s *DBStore) GetStats(ctx context.Context) (*StoreStats, error) {
	var totalRecordings int64
	var totalSize int64

	if err := s.db.WithContext(ctx).Model(&RecordingModel{}).Count(&totalRecordings).Error; err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&RecordingModel{}).Select("COALESCE(SUM(file_size), 0)").Scan(&totalSize).Error; err != nil {
		return nil, err
	}

	return &StoreStats{
		TotalRecordings: totalRecordings,
		TotalSize:       totalSize,
	}, nil
}

