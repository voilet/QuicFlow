// Package recording provides terminal session recording and playback functionality.
package recording

import (
	"time"
)

// Header represents the asciicast v2 header
type Header struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Duration  float64           `json:"duration,omitempty"`
	Title     string            `json:"title,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// Event represents a single recording event
type Event struct {
	Time float64 `json:"time"` // Seconds since recording start
	Type string  `json:"type"` // "o" for output, "i" for input, "r" for resize
	Data string  `json:"data"` // The content
}

// ResizeData represents terminal resize event data
type ResizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// RecordingMeta contains metadata about a recording
type RecordingMeta struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	ClientID    string    `json:"client_id"`
	Username    string    `json:"username"`
	Title       string    `json:"title,omitempty"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Duration    float64   `json:"duration"` // seconds
	FileSize    int64     `json:"file_size"`
	CreatedAt   time.Time `json:"created_at"`
	RecordInput bool      `json:"record_input"` // Whether input was recorded
}

// RecordingFilter provides filtering options for querying recordings
type RecordingFilter struct {
	SessionID string     `json:"session_id,omitempty"`
	ClientID  string     `json:"client_id,omitempty"`
	Username  string     `json:"username,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// Config holds configuration for the recording system
type Config struct {
	// Enabled controls whether recording is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// StorePath is the directory to store recordings
	StorePath string `yaml:"store_path" json:"store_path"`

	// RetentionDays is how long to keep recordings (0 = forever)
	RetentionDays int `yaml:"retention_days" json:"retention_days"`

	// MaxFileSize is the maximum size of a recording file in bytes (0 = unlimited)
	MaxFileSize int64 `yaml:"max_file_size" json:"max_file_size"`

	// RecordInput controls whether to record user input (default: true)
	RecordInput bool `yaml:"record_input" json:"record_input"`
}

// DefaultConfig returns a default recording configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:       true,
		StorePath:     "data/recordings",
		RetentionDays: 30,
		MaxFileSize:   50 * 1024 * 1024, // 50MB
		RecordInput:   true,
	}
}
