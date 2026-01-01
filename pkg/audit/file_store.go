package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileStore implements Store using JSON lines file storage
type FileStore struct {
	path string
	mu   sync.RWMutex
	file *os.File
}

// NewFileStore creates a new file-based audit store
func NewFileStore(path string) (*FileStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &FileStore{
		path: path,
		file: file,
	}, nil
}

// SaveCommand saves a command log entry
func (s *FileStore) SaveCommand(ctx context.Context, log *CommandLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return s.file.Sync()
}

// QueryCommands queries command logs with filters
func (s *FileStore) QueryCommands(ctx context.Context, filter *CommandFilter) ([]*CommandLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*CommandLog{}, nil
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var results []*CommandLog
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	scanner := bufio.NewScanner(file)
	// Set max buffer size for long lines
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		var log CommandLog
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			continue // Skip invalid lines
		}

		if s.matchesFilter(&log, filter) {
			results = append(results, &log)
		}
	}

	// Apply offset and limit
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	} else if filter.Offset >= len(results) {
		return []*CommandLog{}, nil
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, scanner.Err()
}

// matchesFilter checks if a log entry matches the filter
func (s *FileStore) matchesFilter(log *CommandLog, filter *CommandFilter) bool {
	if filter.SessionID != "" && log.SessionID != filter.SessionID {
		return false
	}
	if filter.ClientID != "" && log.ClientID != filter.ClientID {
		return false
	}
	if filter.Username != "" && log.Username != filter.Username {
		return false
	}
	if filter.Command != "" && !strings.Contains(log.Command, filter.Command) {
		return false
	}
	if filter.StartTime != nil && log.ExecutedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && log.ExecutedAt.After(*filter.EndTime) {
		return false
	}
	return true
}

// GetCommandsBySession returns all commands for a session
func (s *FileStore) GetCommandsBySession(ctx context.Context, sessionID string) ([]*CommandLog, error) {
	return s.QueryCommands(ctx, &CommandFilter{
		SessionID: sessionID,
		Limit:     10000,
	})
}

// GetStats returns audit statistics
func (s *FileStore) GetStats(ctx context.Context) (*AuditStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &AuditStats{}

	fileInfo, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return stats, nil
		}
		return nil, err
	}
	stats.StorageSize = fileInfo.Size()

	file, err := os.Open(s.path)
	if err != nil {
		return stats, nil
	}
	defer file.Close()

	sessions := make(map[string]struct{})
	clients := make(map[string]struct{})
	var oldest, newest time.Time

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var log CommandLog
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			continue
		}

		stats.TotalCommands++
		sessions[log.SessionID] = struct{}{}
		clients[log.ClientID] = struct{}{}

		if oldest.IsZero() || log.ExecutedAt.Before(oldest) {
			oldest = log.ExecutedAt
		}
		if newest.IsZero() || log.ExecutedAt.After(newest) {
			newest = log.ExecutedAt
		}
	}

	stats.TotalSessions = int64(len(sessions))
	stats.TotalClients = int64(len(clients))
	stats.OldestRecord = oldest
	stats.NewestRecord = newest

	return stats, nil
}

// DeleteOldRecords deletes records older than the specified days
func (s *FileStore) DeleteOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	// Read all records
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var keptRecords [][]byte
	var deletedCount int64

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var log CommandLog
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			continue
		}

		if log.ExecutedAt.Before(cutoff) {
			deletedCount++
		} else {
			keptRecords = append(keptRecords, append([]byte(nil), scanner.Bytes()...))
		}
	}
	file.Close()

	if deletedCount == 0 {
		return 0, nil
	}

	// Rewrite file with kept records
	s.file.Close()
	tmpPath := s.path + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return 0, err
	}

	for _, record := range keptRecords {
		tmpFile.Write(append(record, '\n'))
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, s.path); err != nil {
		return 0, err
	}

	// Reopen file for appending
	s.file, err = os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return deletedCount, err
	}

	return deletedCount, nil
}

// Close closes the store
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
