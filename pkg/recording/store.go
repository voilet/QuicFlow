package recording

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store manages recording files
type Store struct {
	basePath string
	mu       sync.RWMutex
}

// NewStore creates a new recording store
func NewStore(basePath string) (*Store, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &Store{
		basePath: basePath,
	}, nil
}

// List returns all recordings matching the filter
func (s *Store) List(ctx context.Context, filter *RecordingFilter) ([]*RecordingMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var recordings []*RecordingMeta

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Only process .meta files
		if !strings.HasSuffix(path, ".meta") {
			return nil
		}

		meta, err := s.loadMeta(path)
		if err != nil {
			return nil // Skip invalid files
		}

		// Apply filters
		if filter != nil {
			if filter.SessionID != "" && meta.SessionID != filter.SessionID {
				return nil
			}
			if filter.ClientID != "" && meta.ClientID != filter.ClientID {
				return nil
			}
			if filter.Username != "" && meta.Username != filter.Username {
				return nil
			}
			if filter.StartTime != nil && meta.CreatedAt.Before(*filter.StartTime) {
				return nil
			}
			if filter.EndTime != nil && meta.CreatedAt.After(*filter.EndTime) {
				return nil
			}
		}

		recordings = append(recordings, meta)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by creation time (newest first)
	sort.Slice(recordings, func(i, j int) bool {
		return recordings[i].CreatedAt.After(recordings[j].CreatedAt)
	})

	// Apply pagination
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(recordings) {
			recordings = recordings[filter.Offset:]
		} else if filter.Offset >= len(recordings) {
			return []*RecordingMeta{}, nil
		}

		limit := filter.Limit
		if limit <= 0 {
			limit = 100
		}
		if len(recordings) > limit {
			recordings = recordings[:limit]
		}
	}

	return recordings, nil
}

// loadMeta loads metadata from a .meta file
func (s *Store) loadMeta(path string) (*RecordingMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta RecordingMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// Get returns a specific recording by ID
func (s *Store) Get(ctx context.Context, id string) (*RecordingMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found *RecordingMeta

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != nil {
			return nil
		}

		if strings.HasSuffix(path, id+".cast.meta") {
			meta, err := s.loadMeta(path)
			if err == nil && meta.ID == id {
				found = meta
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if found == nil {
		return nil, fmt.Errorf("recording not found: %s", id)
	}

	return found, nil
}

// GetFilePath returns the file path for a recording
func (s *Store) GetFilePath(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var foundPath string

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || foundPath != "" {
			return nil
		}

		if strings.HasSuffix(path, id+".cast") {
			foundPath = path
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if foundPath == "" {
		return "", fmt.Errorf("recording not found: %s", id)
	}

	return foundPath, nil
}

// StreamEvents returns an iterator over recording events
func (s *Store) StreamEvents(ctx context.Context, id string) (<-chan *Event, error) {
	filePath, err := s.GetFilePath(id)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	events := make(chan *Event, 100)

	go func() {
		defer close(events)
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		// Skip header
		if !scanner.Scan() {
			return
		}

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var raw []interface{}
			if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
				continue
			}

			if len(raw) < 3 {
				continue
			}

			time, ok1 := raw[0].(float64)
			typ, ok2 := raw[1].(string)
			data, ok3 := raw[2].(string)

			if !ok1 || !ok2 || !ok3 {
				continue
			}

			events <- &Event{
				Time: time,
				Type: typ,
				Data: data,
			}
		}
	}()

	return events, nil
}

// ReadFile returns the raw recording file content
func (s *Store) ReadFile(id string) (io.ReadCloser, error) {
	filePath, err := s.GetFilePath(id)
	if err != nil {
		return nil, err
	}

	return os.Open(filePath)
}

// Delete deletes a recording by ID
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath, err := s.GetFilePath(id)
	if err != nil {
		return err
	}

	// Remove both .cast and .meta files
	if err := os.Remove(filePath); err != nil {
		return err
	}

	os.Remove(filePath + ".meta") // Ignore error for meta file

	return nil
}

// DeleteOldRecordings deletes recordings older than the specified days
func (s *Store) DeleteOldRecordings(ctx context.Context, olderThanDays int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	var deletedCount int64

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if strings.HasSuffix(path, ".meta") {
			meta, err := s.loadMeta(path)
			if err == nil && meta.CreatedAt.Before(cutoff) {
				castPath := strings.TrimSuffix(path, ".meta")
				os.Remove(castPath)
				os.Remove(path)
				deletedCount++
			}
		}

		return nil
	})

	return deletedCount, err
}

// GetStats returns statistics about recordings
func (s *Store) GetStats(ctx context.Context) (*StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StoreStats{}

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if strings.HasSuffix(path, ".cast") {
			stats.TotalRecordings++
			stats.TotalSize += info.Size()
		}

		return nil
	})

	return stats, err
}

// StoreStats contains statistics about the recording store
type StoreStats struct {
	TotalRecordings int64 `json:"total_recordings"`
	TotalSize       int64 `json:"total_size_bytes"`
}
