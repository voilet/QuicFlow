// Package api defines the public interfaces for the QUIC backbone network
// This is a contract file showing the expected API surface
// Actual implementation will be in pkg/ directory
package api

import (
	"context"
	"time"
)

// Server represents the QUIC server interface
type Server interface {
	// Start starts the server on the specified address
	// addr format: "host:port" (e.g., "0.0.0.0:8474")
	Start(addr string) error

	// Stop gracefully stops the server
	Stop(ctx context.Context) error

	// SendTo sends a message to a specific client
	// Returns error if client not connected or message fails to send
	SendTo(clientID string, msg *Message) (*Response, error)

	// Broadcast sends a message to all connected clients
	// Returns the number of successful deliveries
	Broadcast(msg *Message) (int, error)

	// ListClients returns all currently connected client IDs
	ListClients() []string

	// GetClientInfo returns detailed information about a specific client
	GetClientInfo(clientID string) (*ClientInfo, error)

	// GetMetrics returns current server metrics snapshot
	GetMetrics() *MetricsSnapshot

	// RegisterHandler registers a message handler for a specific message type
	RegisterHandler(msgType MessageType, handler MessageHandler)

	// SetEventHooks sets event hooks for connection lifecycle events
	SetEventHooks(hooks EventHooks)
}

// Client represents the QUIC client interface
type Client interface {
	// Connect connects to the server at the specified address
	// Implements automatic reconnection with exponential backoff
	Connect(addr string) error

	// Disconnect closes the connection to the server
	Disconnect() error

	// SendMessage sends a message to the server
	// If waitAck is true, waits for server response (with timeout)
	SendMessage(msg *Message) (*Response, error)

	// GetState returns the current client connection state
	GetState() ClientState

	// SetEventHooks sets event hooks for connection lifecycle events
	SetEventHooks(hooks EventHooks)

	// RegisterHandler registers a message handler for incoming messages
	RegisterHandler(msgType MessageType, handler MessageHandler)
}

// MessageHandler handles incoming messages
type MessageHandler interface {
	// OnMessage is called when a message of the registered type is received
	// ctx has a timeout (default 30s) - handler must complete within timeout
	// Returns response (if message has WaitAck=true) or error
	OnMessage(ctx context.Context, msg *IncomingMessage) (*Response, error)
}

// Message represents a message to be sent
type Message struct {
	// Type is the message type (Command/Event/Query/Response)
	Type MessageType

	// Payload is the business data (JSON, Protobuf, or any []byte)
	Payload []byte

	// WaitAck indicates if sender wants acknowledgment/response
	WaitAck bool

	// Timeout is the max wait time for response (if WaitAck=true)
	// Default: 30s
	Timeout time.Duration
}

// IncomingMessage represents a received message with context
type IncomingMessage struct {
	// MsgID is the unique message ID
	MsgID string

	// SenderID is the ID of the sender
	SenderID string

	// Type is the message type
	Type MessageType

	// Payload is the message data
	Payload []byte

	// Timestamp is when the message was sent
	Timestamp time.Time

	// WaitAck indicates if sender is waiting for response
	WaitAck bool
}

// Response represents a response to a message
type Response struct {
	// Status is the execution status
	Status AckStatus

	// Result is the response data (optional)
	Result []byte

	// Error is the error message if Status == FAILURE
	Error error
}

// ClientInfo contains information about a connected client
type ClientInfo struct {
	ClientID      string
	RemoteAddr    string
	ConnectedAt   time.Time
	LastHeartbeat time.Time
	State         ClientState
}

// MetricsSnapshot contains current metrics
type MetricsSnapshot struct {
	ConnectedClients  int64
	TotalConnections  int64
	MessageThroughput int64 // messages per second
	AverageLatency    int64 // milliseconds
	P99Latency        int64 // milliseconds
	Timestamp         time.Time
}

// EventHooks defines callbacks for connection lifecycle events
type EventHooks struct {
	// OnConnect is called when a client connects
	OnConnect func(clientID string)

	// OnDisconnect is called when a client disconnects
	OnDisconnect func(clientID string, reason error)

	// OnHeartbeatTimeout is called when a client heartbeat times out
	OnHeartbeatTimeout func(clientID string)

	// OnReconnect is called when a client successfully reconnects
	OnReconnect func(clientID string, attemptCount int)

	// OnMessageSent is called when a message is sent
	OnMessageSent func(msgID string, clientID string)

	// OnMessageReceived is called when a message is received
	OnMessageReceived func(msgID string, clientID string)
}

// MessageType represents the type of a message
type MessageType int32

const (
	MessageTypeCommand  MessageType = 1 // Command - requires action
	MessageTypeEvent    MessageType = 2 // Event - notification
	MessageTypeQuery    MessageType = 3 // Query - request data
	MessageTypeResponse MessageType = 4 // Response - return result
)

// AckStatus represents the status of a message acknowledgment
type AckStatus int32

const (
	AckStatusSuccess AckStatus = 1 // Successfully executed
	AckStatusFailure AckStatus = 2 // Execution failed
	AckStatusTimeout AckStatus = 3 // Response timeout
)

// ClientState represents the connection state of a client
type ClientState int32

const (
	ClientStateIdle       ClientState = 1 // Not connected
	ClientStateConnecting ClientState = 2 // Connecting
	ClientStateConnected  ClientState = 3 // Connected
)

// ServerConfig contains configuration for the server
type ServerConfig struct {
	// TLSCertFile is the path to TLS certificate file
	TLSCertFile string

	// TLSKeyFile is the path to TLS key file
	TLSKeyFile string

	// MaxClients is the maximum number of concurrent clients (default: 10000)
	MaxClients int

	// HeartbeatInterval is how often clients send heartbeats (default: 15s)
	HeartbeatInterval time.Duration

	// HeartbeatTimeout is max time without heartbeat before disconnect (default: 45s)
	HeartbeatTimeout time.Duration

	// MaxPromises is the max number of pending callbacks (default: 50000)
	MaxPromises int

	// PromiseWarnThreshold is when to warn about high Promise count (default: 40000)
	PromiseWarnThreshold int

	// DefaultMessageTimeout is default timeout for WaitAck messages (default: 30s)
	DefaultMessageTimeout time.Duration
}

// ClientConfig contains configuration for the client
type ClientConfig struct {
	// ClientID is the unique identifier for this client
	ClientID string

	// TLSCertFile is the path to client TLS certificate (for mutual TLS)
	// Optional - if not provided, uses server-only authentication
	TLSCertFile string

	// TLSKeyFile is the path to client TLS key
	TLSKeyFile string

	// InsecureSkipVerify skips TLS certificate verification (NOT for production)
	InsecureSkipVerify bool

	// ReconnectEnabled enables automatic reconnection (default: true)
	ReconnectEnabled bool

	// InitialBackoff is the initial reconnection delay (default: 1s)
	InitialBackoff time.Duration

	// MaxBackoff is the maximum reconnection delay (default: 60s)
	MaxBackoff time.Duration
}

// NewServer creates a new QUIC server with the given configuration
func NewServer(config ServerConfig) (Server, error) {
	// Implementation in pkg/transport/server/
	panic("not implemented - see pkg/transport/server/")
}

// NewClient creates a new QUIC client with the given configuration
func NewClient(config ClientConfig) (Client, error) {
	// Implementation in pkg/transport/client/
	panic("not implemented - see pkg/transport/client/")
}
