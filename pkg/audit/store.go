package audit

import (
	"context"
	"fmt"

	"gorm.io/gorm"
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
	case "postgres", "postgresql":
		return nil, fmt.Errorf("postgres store requires database connection, use NewStoreWithDB instead")
	default:
		return NewFileStore(config.StorePath)
	}
}

// NewStoreWithDB creates a new store with an existing database connection
func NewStoreWithDB(config *Config, db *gorm.DB) (Store, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.StoreType {
	case "postgres", "postgresql":
		if db == nil {
			return nil, fmt.Errorf("database connection is required for postgres store")
		}
		return NewPostgresStore(db)
	case "file":
		return NewFileStore(config.StorePath)
	default:
		// Default to postgres if db is provided
		if db != nil {
			return NewPostgresStore(db)
		}
		return NewFileStore(config.StorePath)
	}
}
