// Package audit provides command auditing functionality for SSH sessions.
package audit

import (
	"time"
)

// CommandLog represents a single command execution record
type CommandLog struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	ClientID    string    `json:"client_id"`
	Username    string    `json:"username"`
	Command     string    `json:"command"`
	ExecutedAt  time.Time `json:"executed_at"`
	ExitCode    int       `json:"exit_code,omitempty"`
	DurationMs  int64     `json:"duration_ms,omitempty"`
	RemoteIP    string    `json:"remote_ip,omitempty"`
	Output      string    `json:"output,omitempty"` // Optional: first N bytes of output
}

// CommandFilter provides filtering options for querying command logs
type CommandFilter struct {
	SessionID   string     `json:"session_id,omitempty"`
	ClientID    string     `json:"client_id,omitempty"`
	Username    string     `json:"username,omitempty"`
	Command     string     `json:"command,omitempty"`    // Substring match
	StartTime   *time.Time `json:"start_time,omitempty"` // Commands after this time
	EndTime     *time.Time `json:"end_time,omitempty"`   // Commands before this time
	Limit       int        `json:"limit,omitempty"`      // Max results (default 100)
	Offset      int        `json:"offset,omitempty"`     // Pagination offset
}

// AuditStats provides statistics about audit logs
type AuditStats struct {
	TotalCommands   int64     `json:"total_commands"`
	TotalSessions   int64     `json:"total_sessions"`
	TotalClients    int64     `json:"total_clients"`
	OldestRecord    time.Time `json:"oldest_record,omitempty"`
	NewestRecord    time.Time `json:"newest_record,omitempty"`
	StorageSize     int64     `json:"storage_size_bytes"`
}

// Config holds configuration for the audit system
type Config struct {
	// Enabled controls whether auditing is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// StoreType specifies the storage backend ("postgres", "sqlite", or "file")
	StoreType string `yaml:"store_type" json:"store_type"`

	// StorePath is the path to the storage file/database (for sqlite/file types)
	StorePath string `yaml:"store_path" json:"store_path"`

	// RetentionDays is how long to keep audit logs (0 = forever)
	RetentionDays int `yaml:"retention_days" json:"retention_days"`

	// MaxOutputCapture is the max bytes of command output to store (0 = don't capture)
	MaxOutputCapture int `yaml:"max_output_capture" json:"max_output_capture"`
}

// DefaultConfig returns a default audit configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:          true,
		StoreType:        "postgres",
		StorePath:        "",
		RetentionDays:    90,
		MaxOutputCapture: 0, // Don't capture output by default
	}
}
