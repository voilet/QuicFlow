package audit

import (
	"context"
)

// Store defines the interface for audit log storage
type Store interface {
	// SaveCommand saves a command log entry
	SaveCommand(ctx context.Context, log *CommandLog) error

	// QueryCommands queries command logs with filters
	QueryCommands(ctx context.Context, filter *CommandFilter) ([]*CommandLog, error)

	// GetCommandsBySession returns all commands for a session
	GetCommandsBySession(ctx context.Context, sessionID string) ([]*CommandLog, error)

	// GetStats returns audit statistics
	GetStats(ctx context.Context) (*AuditStats, error)

	// DeleteOldRecords deletes records older than the specified days
	DeleteOldRecords(ctx context.Context, olderThanDays int) (int64, error)

	// Close closes the store
	Close() error
}

// NewStore creates a new store based on the configuration
func NewStore(config *Config) (Store, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.StoreType {
	case "file":
		return NewFileStore(config.StorePath)
	case "sqlite":
		return NewSQLiteStore(config.StorePath)
	default:
		return NewSQLiteStore(config.StorePath)
	}
}
