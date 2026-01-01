package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SessionFileStore implements Store using per-session file storage
// Each session gets its own JSON file containing all commands
type SessionFileStore struct {
	basePath string
	mu       sync.RWMutex
	files    map[string]*os.File // session_id -> file handle
}

// SessionMeta contains metadata about a session file
type SessionMeta struct {
	SessionID    string    `json:"session_id"`
	ClientID     string    `json:"client_id"`
	Username     string    `json:"username"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	CommandCount int       `json:"command_count"`
	FileSize     int64     `json:"file_size"`
}

// NewSessionFileStore creates a new per-session file-based audit store
func NewSessionFileStore(basePath string) (*SessionFileStore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &SessionFileStore{
		basePath: basePath,
		files:    make(map[string]*os.File),
	}, nil
}

// getSessionFile gets or creates a file for the session
func (s *SessionFileStore) getSessionFile(sessionID string) (*os.File, error) {
	if f, ok := s.files[sessionID]; ok {
		return f, nil
	}

	filePath := filepath.Join(s.basePath, sessionID+".jsonl")
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	s.files[sessionID] = f
	return f, nil
}

// SaveCommand saves a command log entry to the session's file
func (s *SessionFileStore) SaveCommand(ctx context.Context, log *CommandLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.getSessionFile(log.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get session file: %w", err)
	}

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return f.Sync()
}

// QueryCommands queries command logs with filters across all session files
func (s *SessionFileStore) QueryCommands(ctx context.Context, filter *CommandFilter) ([]*CommandLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*CommandLog
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	// If filtering by session ID, only read that file
	if filter.SessionID != "" {
		logs, err := s.readSessionFile(ctx, filter.SessionID)
		if err != nil {
			return nil, err
		}
		for _, log := range logs {
			if s.matchesFilter(log, filter) {
				results = append(results, log)
			}
		}
	} else {
		// Read all session files
		files, err := filepath.Glob(filepath.Join(s.basePath, "*.jsonl"))
		if err != nil {
			return nil, err
		}

		for _, filePath := range files {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			default:
			}

			sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")
			logs, err := s.readSessionFile(ctx, sessionID)
			if err != nil {
				continue
			}

			for _, log := range logs {
				if s.matchesFilter(log, filter) {
					results = append(results, log)
				}
			}
		}
	}

	// Sort by execution time (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ExecutedAt.After(results[j].ExecutedAt)
	})

	// Apply offset and limit
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	} else if filter.Offset >= len(results) {
		return []*CommandLog{}, nil
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// readSessionFile reads all commands from a session file
func (s *SessionFileStore) readSessionFile(ctx context.Context, sessionID string) ([]*CommandLog, error) {
	filePath := filepath.Join(s.basePath, sessionID+".jsonl")
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*CommandLog{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var logs []*CommandLog
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return logs, ctx.Err()
		default:
		}

		var log CommandLog
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, scanner.Err()
}

// matchesFilter checks if a log entry matches the filter
func (s *SessionFileStore) matchesFilter(log *CommandLog, filter *CommandFilter) bool {
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
func (s *SessionFileStore) GetCommandsBySession(ctx context.Context, sessionID string) ([]*CommandLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readSessionFile(ctx, sessionID)
}

// GetStats returns audit statistics
func (s *SessionFileStore) GetStats(ctx context.Context) (*AuditStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &AuditStats{}

	files, err := filepath.Glob(filepath.Join(s.basePath, "*.jsonl"))
	if err != nil {
		return stats, err
	}

	clients := make(map[string]struct{})
	var oldest, newest time.Time

	for _, filePath := range files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		stats.StorageSize += fileInfo.Size()
		stats.TotalSessions++

		sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")
		logs, err := s.readSessionFile(ctx, sessionID)
		if err != nil {
			continue
		}

		stats.TotalCommands += int64(len(logs))

		for _, log := range logs {
			clients[log.ClientID] = struct{}{}

			if oldest.IsZero() || log.ExecutedAt.Before(oldest) {
				oldest = log.ExecutedAt
			}
			if newest.IsZero() || log.ExecutedAt.After(newest) {
				newest = log.ExecutedAt
			}
		}
	}

	stats.TotalClients = int64(len(clients))
	stats.OldestRecord = oldest
	stats.NewestRecord = newest

	return stats, nil
}

// ListSessions returns metadata for all sessions
func (s *SessionFileStore) ListSessions(ctx context.Context) ([]*SessionMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := filepath.Glob(filepath.Join(s.basePath, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var sessions []*SessionMeta

	for _, filePath := range files {
		select {
		case <-ctx.Done():
			return sessions, ctx.Err()
		default:
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")
		logs, err := s.readSessionFile(ctx, sessionID)
		if err != nil || len(logs) == 0 {
			continue
		}

		meta := &SessionMeta{
			SessionID:    sessionID,
			ClientID:     logs[0].ClientID,
			Username:     logs[0].Username,
			StartTime:    logs[0].ExecutedAt,
			EndTime:      logs[len(logs)-1].ExecutedAt,
			CommandCount: len(logs),
			FileSize:     fileInfo.Size(),
		}

		sessions = append(sessions, meta)
	}

	// Sort by start time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	return sessions, nil
}

// DeleteOldRecords deletes session files older than the specified days
func (s *SessionFileStore) DeleteOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	files, err := filepath.Glob(filepath.Join(s.basePath, "*.jsonl"))
	if err != nil {
		return 0, err
	}

	var deletedCount int64

	for _, filePath := range files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if fileInfo.ModTime().Before(cutoff) {
			sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")

			// Close file handle if open
			if f, ok := s.files[sessionID]; ok {
				f.Close()
				delete(s.files, sessionID)
			}

			if err := os.Remove(filePath); err == nil {
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}

// CloseSession closes the file handle for a specific session
func (s *SessionFileStore) CloseSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if f, ok := s.files[sessionID]; ok {
		err := f.Close()
		delete(s.files, sessionID)
		return err
	}
	return nil
}

// Close closes all open file handles
func (s *SessionFileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastErr error
	for sessionID, f := range s.files {
		if err := f.Close(); err != nil {
			lastErr = err
		}
		delete(s.files, sessionID)
	}
	return lastErr
}
