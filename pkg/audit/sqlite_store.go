package audit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store using SQLite database
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteStore creates a new SQLite-based audit store
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db:   db,
		path: path,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS command_logs (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		client_id TEXT NOT NULL,
		username TEXT NOT NULL,
		command TEXT NOT NULL,
		executed_at DATETIME NOT NULL,
		exit_code INTEGER,
		duration_ms INTEGER,
		remote_ip TEXT,
		output TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_command_logs_session ON command_logs(session_id);
	CREATE INDEX IF NOT EXISTS idx_command_logs_client ON command_logs(client_id);
	CREATE INDEX IF NOT EXISTS idx_command_logs_username ON command_logs(username);
	CREATE INDEX IF NOT EXISTS idx_command_logs_executed_at ON command_logs(executed_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveCommand saves a command log entry
func (s *SQLiteStore) SaveCommand(ctx context.Context, log *CommandLog) error {
	query := `
	INSERT INTO command_logs (id, session_id, client_id, username, command, executed_at, exit_code, duration_ms, remote_ip, output)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		log.ID,
		log.SessionID,
		log.ClientID,
		log.Username,
		log.Command,
		log.ExecutedAt,
		log.ExitCode,
		log.DurationMs,
		log.RemoteIP,
		log.Output,
	)

	return err
}

// QueryCommands queries command logs with filters
func (s *SQLiteStore) QueryCommands(ctx context.Context, filter *CommandFilter) ([]*CommandLog, error) {
	var conditions []string
	var args []interface{}

	if filter.SessionID != "" {
		conditions = append(conditions, "session_id = ?")
		args = append(args, filter.SessionID)
	}
	if filter.ClientID != "" {
		conditions = append(conditions, "client_id = ?")
		args = append(args, filter.ClientID)
	}
	if filter.Username != "" {
		conditions = append(conditions, "username = ?")
		args = append(args, filter.Username)
	}
	if filter.Command != "" {
		conditions = append(conditions, "command LIKE ?")
		args = append(args, "%"+filter.Command+"%")
	}
	if filter.StartTime != nil {
		conditions = append(conditions, "executed_at >= ?")
		args = append(args, *filter.StartTime)
	}
	if filter.EndTime != nil {
		conditions = append(conditions, "executed_at <= ?")
		args = append(args, *filter.EndTime)
	}

	query := "SELECT id, session_id, client_id, username, command, executed_at, exit_code, duration_ms, remote_ip, output FROM command_logs"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY executed_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*CommandLog
	for rows.Next() {
		var log CommandLog
		var output sql.NullString
		var exitCode, durationMs sql.NullInt64
		var remoteIP sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.SessionID,
			&log.ClientID,
			&log.Username,
			&log.Command,
			&log.ExecutedAt,
			&exitCode,
			&durationMs,
			&remoteIP,
			&output,
		)
		if err != nil {
			return nil, err
		}

		if exitCode.Valid {
			log.ExitCode = int(exitCode.Int64)
		}
		if durationMs.Valid {
			log.DurationMs = durationMs.Int64
		}
		if remoteIP.Valid {
			log.RemoteIP = remoteIP.String
		}
		if output.Valid {
			log.Output = output.String
		}

		results = append(results, &log)
	}

	return results, rows.Err()
}

// GetCommandsBySession returns all commands for a session
func (s *SQLiteStore) GetCommandsBySession(ctx context.Context, sessionID string) ([]*CommandLog, error) {
	return s.QueryCommands(ctx, &CommandFilter{
		SessionID: sessionID,
		Limit:     10000,
	})
}

// GetStats returns audit statistics
func (s *SQLiteStore) GetStats(ctx context.Context) (*AuditStats, error) {
	stats := &AuditStats{}

	// Get file size
	if info, err := os.Stat(s.path); err == nil {
		stats.StorageSize = info.Size()
	}

	// Get counts
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total_commands,
			COUNT(DISTINCT session_id) as total_sessions,
			COUNT(DISTINCT client_id) as total_clients,
			MIN(executed_at) as oldest,
			MAX(executed_at) as newest
		FROM command_logs
	`)

	var oldest, newest sql.NullTime
	err := row.Scan(
		&stats.TotalCommands,
		&stats.TotalSessions,
		&stats.TotalClients,
		&oldest,
		&newest,
	)
	if err != nil {
		return nil, err
	}

	if oldest.Valid {
		stats.OldestRecord = oldest.Time
	}
	if newest.Valid {
		stats.NewestRecord = newest.Time
	}

	return stats, nil
}

// DeleteOldRecords deletes records older than the specified days
func (s *SQLiteStore) DeleteOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	result, err := s.db.ExecContext(ctx, "DELETE FROM command_logs WHERE executed_at < ?", cutoff)
	if err != nil {
		return 0, err
	}

	// Vacuum to reclaim space
	s.db.ExecContext(ctx, "VACUUM")

	return result.RowsAffected()
}

// Close closes the store
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
