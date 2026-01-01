package api

import (
	"io"

	"github.com/voilet/quic-flow/pkg/audit"
	"github.com/voilet/quic-flow/pkg/recording"
)

// AuditRecordingWriter wraps a writer to record and audit input
type AuditRecordingWriter struct {
	inner    io.WriteCloser
	auditor  *audit.SessionAuditor
	recorder *recording.Recorder
}

// NewAuditRecordingWriter creates a new audit/recording writer
func NewAuditRecordingWriter(inner io.WriteCloser, auditor *audit.SessionAuditor, recorder *recording.Recorder) *AuditRecordingWriter {
	return &AuditRecordingWriter{
		inner:    inner,
		auditor:  auditor,
		recorder: recorder,
	}
}

// Write writes data and records/audits it
func (w *AuditRecordingWriter) Write(p []byte) (int, error) {
	// Record input
	if w.recorder != nil {
		w.recorder.RecordInput(p)
	}

	// Audit input
	if w.auditor != nil {
		w.auditor.RecordInput(p)
	}

	return w.inner.Write(p)
}

// Close closes the writer
func (w *AuditRecordingWriter) Close() error {
	return w.inner.Close()
}

// AuditRecordingReader wraps a reader to record output
type AuditRecordingReader struct {
	inner    io.Reader
	recorder *recording.Recorder
}

// NewAuditRecordingReader creates a new recording reader
func NewAuditRecordingReader(inner io.Reader, recorder *recording.Recorder) *AuditRecordingReader {
	return &AuditRecordingReader{
		inner:    inner,
		recorder: recorder,
	}
}

// Read reads data and records it
func (r *AuditRecordingReader) Read(p []byte) (int, error) {
	n, err := r.inner.Read(p)
	if n > 0 && r.recorder != nil {
		r.recorder.RecordOutput(p[:n])
	}
	return n, err
}

// SessionRecordingConfig holds the configuration for session recording
type SessionRecordingConfig struct {
	// AuditEnabled enables command auditing
	AuditEnabled bool

	// RecordingEnabled enables session recording
	RecordingEnabled bool

	// AuditStore is the audit store to use
	AuditStore audit.Store

	// RecordingConfig is the recording configuration
	RecordingConfig *recording.Config
}
