package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CommandLogModel GORM model for command logs (PostgreSQL)
type CommandLogModel struct {
	ID         string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID  string    `gorm:"size:100;index;not null"`
	ClientID   string    `gorm:"size:100;index;not null"`
	Username   string    `gorm:"size:100;index;not null"`
	Command    string    `gorm:"type:text;not null"`
	ExecutedAt time.Time `gorm:"index;not null"`
	ExitCode   int       `gorm:"default:0"`
	DurationMs int64     `gorm:"default:0"`
	RemoteIP   string    `gorm:"size:50"`
	Output     string    `gorm:"type:text"`
	CreatedAt  time.Time
}

func (CommandLogModel) TableName() string {
	return "command_logs"
}

// PostgresStore implements Store using PostgreSQL database via GORM
type PostgresStore struct {
	db *gorm.DB
}

// NewPostgresStore creates a new PostgreSQL-based audit store
func NewPostgresStore(db *gorm.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	store := &PostgresStore{
		db: db,
	}

	// Auto migrate the command log table
	if err := db.AutoMigrate(&CommandLogModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate command_logs table: %w", err)
	}

	return store, nil
}

// SaveCommand saves a command log entry
func (s *PostgresStore) SaveCommand(ctx context.Context, log *CommandLog) error {
	model := &CommandLogModel{
		ID:         log.ID,
		SessionID:  log.SessionID,
		ClientID:   log.ClientID,
		Username:   log.Username,
		Command:    log.Command,
		ExecutedAt: log.ExecutedAt,
		ExitCode:   log.ExitCode,
		DurationMs: log.DurationMs,
		RemoteIP:   log.RemoteIP,
		Output:     log.Output,
	}

	return s.db.WithContext(ctx).Create(model).Error
}

// QueryCommands queries command logs with filters
func (s *PostgresStore) QueryCommands(ctx context.Context, filter *CommandFilter) ([]*CommandLog, error) {
	query := s.db.WithContext(ctx).Model(&CommandLogModel{})

	if filter.SessionID != "" {
		query = query.Where("session_id = ?", filter.SessionID)
	}
	if filter.ClientID != "" {
		query = query.Where("client_id = ?", filter.ClientID)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.Command != "" {
		query = query.Where("command LIKE ?", "%"+filter.Command+"%")
	}
	if filter.StartTime != nil {
		query = query.Where("executed_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("executed_at <= ?", *filter.EndTime)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	query = query.Order("executed_at DESC").Limit(limit).Offset(filter.Offset)

	var models []CommandLogModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	results := make([]*CommandLog, len(models))
	for i, m := range models {
		results[i] = &CommandLog{
			ID:         m.ID,
			SessionID:  m.SessionID,
			ClientID:   m.ClientID,
			Username:   m.Username,
			Command:    m.Command,
			ExecutedAt: m.ExecutedAt,
			ExitCode:   m.ExitCode,
			DurationMs: m.DurationMs,
			RemoteIP:   m.RemoteIP,
			Output:     m.Output,
		}
	}

	return results, nil
}

// GetCommandsBySession returns all commands for a session
func (s *PostgresStore) GetCommandsBySession(ctx context.Context, sessionID string) ([]*CommandLog, error) {
	return s.QueryCommands(ctx, &CommandFilter{
		SessionID: sessionID,
		Limit:     10000,
	})
}

// GetStats returns audit statistics
func (s *PostgresStore) GetStats(ctx context.Context) (*AuditStats, error) {
	stats := &AuditStats{}

	// Get counts
	var result struct {
		TotalCommands int64
		TotalSessions int64
		TotalClients  int64
		Oldest        *time.Time
		Newest        *time.Time
	}

	err := s.db.WithContext(ctx).Model(&CommandLogModel{}).Select(`
		COUNT(*) as total_commands,
		COUNT(DISTINCT session_id) as total_sessions,
		COUNT(DISTINCT client_id) as total_clients,
		MIN(executed_at) as oldest,
		MAX(executed_at) as newest
	`).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	stats.TotalCommands = result.TotalCommands
	stats.TotalSessions = result.TotalSessions
	stats.TotalClients = result.TotalClients
	if result.Oldest != nil {
		stats.OldestRecord = *result.Oldest
	}
	if result.Newest != nil {
		stats.NewestRecord = *result.Newest
	}

	// Get table size (PostgreSQL specific)
	var tableSize int64
	s.db.WithContext(ctx).Raw(`
		SELECT pg_total_relation_size('command_logs')
	`).Scan(&tableSize)
	stats.StorageSize = tableSize

	return stats, nil
}

// DeleteOldRecords deletes records older than the specified days
func (s *PostgresStore) DeleteOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	result := s.db.WithContext(ctx).Where("executed_at < ?", cutoff).Delete(&CommandLogModel{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// ListSessions returns metadata for all sessions (PostgreSQL implementation)
func (s *PostgresStore) ListSessions(ctx context.Context) ([]*SessionMeta, error) {
	var results []struct {
		SessionID    string
		ClientID     string
		Username     string
		StartTime    time.Time
		EndTime      time.Time
		CommandCount int
	}

	err := s.db.WithContext(ctx).Model(&CommandLogModel{}).
		Select(`
			session_id,
			MIN(client_id) as client_id,
			MIN(username) as username,
			MIN(executed_at) as start_time,
			MAX(executed_at) as end_time,
			COUNT(*) as command_count
		`).
		Group("session_id").
		Order("MAX(executed_at) DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	sessions := make([]*SessionMeta, len(results))
	for i, r := range results {
		sessions[i] = &SessionMeta{
			SessionID:    r.SessionID,
			ClientID:     r.ClientID,
			Username:     r.Username,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			CommandCount: r.CommandCount,
			FileSize:     0, // Not applicable for PostgreSQL
		}
	}

	return sessions, nil
}

// CloseSession is a no-op for PostgreSQL (no file handles to close)
func (s *PostgresStore) CloseSession(sessionID string) error {
	return nil
}

// Close closes the store (no-op for PostgreSQL as connection is managed externally)
func (s *PostgresStore) Close() error {
	// Connection is managed by the main database connection pool
	return nil
}

// SearchCommands performs full-text search on commands
func (s *PostgresStore) SearchCommands(ctx context.Context, query string, limit int) ([]*CommandLog, error) {
	if limit <= 0 {
		limit = 100
	}

	// Use PostgreSQL's ILIKE for case-insensitive search
	searchPattern := "%" + strings.ReplaceAll(query, "%", "\\%") + "%"

	var models []CommandLogModel
	err := s.db.WithContext(ctx).
		Where("command ILIKE ? OR output ILIKE ?", searchPattern, searchPattern).
		Order("executed_at DESC").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	results := make([]*CommandLog, len(models))
	for i, m := range models {
		results[i] = &CommandLog{
			ID:         m.ID,
			SessionID:  m.SessionID,
			ClientID:   m.ClientID,
			Username:   m.Username,
			Command:    m.Command,
			ExecutedAt: m.ExecutedAt,
			ExitCode:   m.ExitCode,
			DurationMs: m.DurationMs,
			RemoteIP:   m.RemoteIP,
			Output:     m.Output,
		}
	}

	return results, nil
}
