package recording

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Recorder records a terminal session in asciicast v2 format
type Recorder struct {
	id        string
	sessionID string
	clientID  string
	username  string

	startTime   time.Time
	width       int
	height      int
	recordInput bool

	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex

	eventCount int
	fileSize   int64
	maxSize    int64

	closed bool

	// Database store for saving recording metadata
	dbStore *DBStore
}

// NewRecorder creates a new session recorder
func NewRecorder(config *Config, sessionID, clientID, username string, width, height int) (*Recorder, error) {
	return NewRecorderWithDB(config, sessionID, clientID, username, width, height, nil)
}

// NewRecorderWithDB creates a new session recorder with database storage
func NewRecorderWithDB(config *Config, sessionID, clientID, username string, width, height int, dbStore *DBStore) (*Recorder, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Generate unique ID
	id := uuid.New().String()

	// Create storage directory with date-based structure
	now := time.Now()
	dateDir := filepath.Join(config.StorePath, now.Format("2006/01/02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create recording file
	filename := fmt.Sprintf("%s.cast", id)
	filePath := filepath.Join(dateDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	r := &Recorder{
		id:          id,
		sessionID:   sessionID,
		clientID:    clientID,
		username:    username,
		startTime:   now,
		width:       width,
		height:      height,
		recordInput: config.RecordInput,
		file:        file,
		writer:      bufio.NewWriter(file),
		maxSize:     config.MaxFileSize,
		dbStore:     dbStore,
	}

	// Write header
	if err := r.writeHeader(); err != nil {
		file.Close()
		os.Remove(filePath)
		return nil, err
	}

	return r, nil
}

// writeHeader writes the asciicast v2 header
func (r *Recorder) writeHeader() error {
	header := Header{
		Version:   2,
		Width:     r.width,
		Height:    r.height,
		Timestamp: r.startTime.Unix(),
		Env: map[string]string{
			"TERM":  "xterm-256color",
			"SHELL": "/bin/sh",
		},
	}

	data, err := json.Marshal(header)
	if err != nil {
		return err
	}

	if _, err := r.writer.Write(append(data, '\n')); err != nil {
		return err
	}

	return r.writer.Flush()
}

// RecordOutput records terminal output
func (r *Recorder) RecordOutput(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("recorder is closed")
	}

	return r.writeEvent("o", string(data))
}

// RecordInput records user input
func (r *Recorder) RecordInput(data []byte) error {
	if len(data) == 0 || !r.recordInput {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("recorder is closed")
	}

	return r.writeEvent("i", string(data))
}

// RecordResize records terminal resize event
func (r *Recorder) RecordResize(cols, rows int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("recorder is closed")
	}

	r.width = cols
	r.height = rows

	resizeData, _ := json.Marshal(ResizeData{Cols: cols, Rows: rows})
	return r.writeEvent("r", string(resizeData))
}

// writeEvent writes a single event to the recording
func (r *Recorder) writeEvent(eventType, data string) error {
	// Check max file size
	if r.maxSize > 0 && r.fileSize > r.maxSize {
		return fmt.Errorf("max file size exceeded")
	}

	elapsed := time.Since(r.startTime).Seconds()

	// Asciicast v2 format: [time, type, data]
	event := []interface{}{elapsed, eventType, data}
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	n, err := r.writer.Write(append(eventData, '\n'))
	if err != nil {
		return err
	}

	r.fileSize += int64(n)
	r.eventCount++

	// Flush periodically
	if r.eventCount%100 == 0 {
		return r.writer.Flush()
	}

	return nil
}

// Close closes the recorder and finalizes the recording
func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	// Flush writer
	if err := r.writer.Flush(); err != nil {
		r.file.Close()
		return err
	}

	// Get file path before closing
	filePath := r.file.Name()

	// Write metadata file
	if err := r.writeMetaFile(); err != nil {
		r.file.Close()
		return err
	}

	// Close file
	if err := r.file.Close(); err != nil {
		return err
	}

	// Save to database if dbStore is available
	if r.dbStore != nil {
		meta := RecordingMeta{
			ID:          r.id,
			SessionID:   r.sessionID,
			ClientID:    r.clientID,
			Username:    r.username,
			Width:       r.width,
			Height:      r.height,
			Duration:    time.Since(r.startTime).Seconds(),
			FileSize:    r.fileSize,
			CreatedAt:   r.startTime,
			RecordInput: r.recordInput,
		}

		// Use background context for database save
		ctx := context.Background()
		if err := r.dbStore.SaveRecording(ctx, &meta, filePath); err != nil {
			// Log error but don't fail the close operation
			// The metadata file is already written, so the recording is still valid
			return fmt.Errorf("failed to save recording to database: %w", err)
		}
	}

	return nil
}

// writeMetaFile writes the metadata file alongside the recording
func (r *Recorder) writeMetaFile() error {
	meta := RecordingMeta{
		ID:          r.id,
		SessionID:   r.sessionID,
		ClientID:    r.clientID,
		Username:    r.username,
		Width:       r.width,
		Height:      r.height,
		Duration:    time.Since(r.startTime).Seconds(),
		FileSize:    r.fileSize,
		CreatedAt:   r.startTime,
		RecordInput: r.recordInput,
	}

	metaPath := r.file.Name() + ".meta"
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, metaData, 0644)
}

// GetID returns the recording ID
func (r *Recorder) GetID() string {
	return r.id
}

// GetSessionID returns the session ID
func (r *Recorder) GetSessionID() string {
	return r.sessionID
}

// GetFilePath returns the recording file path
func (r *Recorder) GetFilePath() string {
	if r.file != nil {
		return r.file.Name()
	}
	return ""
}

// GetDuration returns the current recording duration
func (r *Recorder) GetDuration() time.Duration {
	return time.Since(r.startTime)
}
