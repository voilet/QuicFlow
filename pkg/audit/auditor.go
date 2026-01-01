package audit

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ANSI escape sequence patterns to filter
var (
	// Bracketed paste mode: \e[200~ and \e[201~
	bracketedPasteRegex = regexp.MustCompile(`\x1b\[\d+~`)
	// General ANSI escape sequences: \e[...
	ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z~]`)
	// OSC sequences: \e]...BEL or \e]...\e\\
	oscSequenceRegex = regexp.MustCompile(`\x1b\][^\x07\x1b]*(\x07|\x1b\\)`)
)

// SessionAuditor captures and audits commands from a PTY session
type SessionAuditor struct {
	store     Store
	sessionID string
	clientID  string
	username  string
	remoteIP  string

	// Command buffer
	inputBuffer bytes.Buffer
	mu          sync.Mutex

	// Escape sequence tracking
	inEscapeSeq bool
	escapeBuffer bytes.Buffer

	// Current command tracking
	currentCommand string
	commandStart   time.Time

	// Context for async operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSessionAuditor creates a new session auditor
func NewSessionAuditor(store Store, sessionID, clientID, username, remoteIP string) *SessionAuditor {
	ctx, cancel := context.WithCancel(context.Background())
	return &SessionAuditor{
		store:     store,
		sessionID: sessionID,
		clientID:  clientID,
		username:  username,
		remoteIP:  remoteIP,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RecordInput records user input and detects commands
func (a *SessionAuditor) RecordInput(data []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := 0; i < len(data); i++ {
		b := data[i]

		// Handle escape sequences
		if b == 0x1b { // ESC
			a.inEscapeSeq = true
			a.escapeBuffer.Reset()
			a.escapeBuffer.WriteByte(b)
			continue
		}

		if a.inEscapeSeq {
			a.escapeBuffer.WriteByte(b)

			// Check if escape sequence is complete
			if a.isEscapeSequenceComplete() {
				a.inEscapeSeq = false
				a.escapeBuffer.Reset()
			}
			continue
		}

		switch b {
		case '\r', '\n':
			// User pressed Enter - record the command
			cmd := a.cleanCommand(a.inputBuffer.String())
			if cmd != "" && !a.isPromptOnly(cmd) {
				a.currentCommand = cmd
				a.commandStart = time.Now()

				// Save command immediately
				go a.saveCommand(cmd, 0, 0)
			}
			a.inputBuffer.Reset()

		case 0x7f, 0x08: // Backspace/Delete
			if a.inputBuffer.Len() > 0 {
				buf := a.inputBuffer.Bytes()
				a.inputBuffer.Reset()
				a.inputBuffer.Write(buf[:len(buf)-1])
			}

		case 0x03: // Ctrl+C
			a.inputBuffer.Reset()

		case 0x15: // Ctrl+U (clear line)
			a.inputBuffer.Reset()

		default:
			// Only record printable characters
			if b >= 32 && b < 127 {
				a.inputBuffer.WriteByte(b)
			}
		}
	}
}

// isEscapeSequenceComplete checks if the current escape sequence buffer is complete
func (a *SessionAuditor) isEscapeSequenceComplete() bool {
	seq := a.escapeBuffer.Bytes()
	if len(seq) < 2 {
		return false
	}

	// CSI sequences: ESC [ ... <letter or ~>
	if seq[1] == '[' {
		if len(seq) >= 3 {
			last := seq[len(seq)-1]
			// CSI sequence ends with a letter or ~
			if (last >= 'A' && last <= 'Z') || (last >= 'a' && last <= 'z') || last == '~' {
				return true
			}
		}
		// Max length check to avoid infinite buffering
		if len(seq) > 20 {
			return true
		}
		return false
	}

	// OSC sequences: ESC ] ... BEL or ESC ] ... ESC \
	if seq[1] == ']' {
		last := seq[len(seq)-1]
		if last == 0x07 { // BEL
			return true
		}
		if len(seq) >= 3 && seq[len(seq)-2] == 0x1b && last == '\\' {
			return true
		}
		if len(seq) > 256 {
			return true
		}
		return false
	}

	// SS2, SS3: ESC N, ESC O followed by one char
	if seq[1] == 'N' || seq[1] == 'O' {
		return len(seq) >= 3
	}

	// Simple escape sequences: ESC <letter>
	if len(seq) == 2 && ((seq[1] >= 'A' && seq[1] <= 'Z') || (seq[1] >= 'a' && seq[1] <= 'z')) {
		return true
	}

	// Unknown sequence, consider complete after 2 chars
	return len(seq) >= 2
}

// cleanCommand removes any remaining ANSI sequences from the command
func (a *SessionAuditor) cleanCommand(s string) string {
	// Remove bracketed paste sequences
	s = bracketedPasteRegex.ReplaceAllString(s, "")
	// Remove other ANSI escape sequences
	s = ansiEscapeRegex.ReplaceAllString(s, "")
	// Remove OSC sequences
	s = oscSequenceRegex.ReplaceAllString(s, "")
	// Clean up any remaining escape characters
	s = strings.ReplaceAll(s, "\x1b", "")
	// Trim whitespace
	return strings.TrimSpace(s)
}

// isPromptOnly checks if the string looks like just a shell prompt
func (a *SessionAuditor) isPromptOnly(s string) bool {
	// Common prompt patterns
	prompts := []string{"$", "#", ">", "%"}
	s = strings.TrimSpace(s)
	for _, p := range prompts {
		if s == p {
			return true
		}
	}
	return false
}

// saveCommand saves a command to the audit store
func (a *SessionAuditor) saveCommand(command string, exitCode int, durationMs int64) {
	log := &CommandLog{
		ID:         uuid.New().String(),
		SessionID:  a.sessionID,
		ClientID:   a.clientID,
		Username:   a.username,
		Command:    command,
		ExecutedAt: time.Now(),
		ExitCode:   exitCode,
		DurationMs: durationMs,
		RemoteIP:   a.remoteIP,
	}

	if a.store != nil {
		a.store.SaveCommand(a.ctx, log)
	}
}

// RecordOutput records terminal output (optional, for context)
// This is a no-op by default to avoid storing sensitive output
func (a *SessionAuditor) RecordOutput(data []byte) {
	// Override in subclass if output capture is needed
}

// Close closes the auditor
func (a *SessionAuditor) Close() error {
	a.cancel()
	return nil
}

// GetSessionID returns the session ID
func (a *SessionAuditor) GetSessionID() string {
	return a.sessionID
}

// GetClientID returns the client ID
func (a *SessionAuditor) GetClientID() string {
	return a.clientID
}
