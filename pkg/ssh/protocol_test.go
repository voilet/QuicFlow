package ssh

import (
	"bytes"
	"testing"
)

func TestStreamType_String(t *testing.T) {
	tests := []struct {
		streamType StreamType
		expected   string
	}{
		{StreamTypeData, "Data"},
		{StreamTypeSSH, "SSH"},
		{StreamTypeFileTransfer, "FileTransfer"},
		{StreamTypePortForward, "PortForward"},
		{StreamType(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.streamType.String(); got != tt.expected {
				t.Errorf("StreamType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWriteAndReadHeader(t *testing.T) {
	tests := []struct {
		name       string
		streamType StreamType
	}{
		{"Data stream", StreamTypeData},
		{"SSH stream", StreamTypeSSH},
		{"FileTransfer stream", StreamTypeFileTransfer},
		{"PortForward stream", StreamTypePortForward},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			// Write header
			if err := WriteHeader(&buf, tt.streamType); err != nil {
				t.Fatalf("WriteHeader() error = %v", err)
			}

			// Verify size
			if buf.Len() != HeaderSize {
				t.Errorf("Header size = %d, want %d", buf.Len(), HeaderSize)
			}

			// Read header
			header, err := ReadHeader(&buf)
			if err != nil {
				t.Fatalf("ReadHeader() error = %v", err)
			}

			// Verify values
			if header.Magic != MagicNumber {
				t.Errorf("Magic = 0x%X, want 0x%X", header.Magic, MagicNumber)
			}
			if header.Version != ProtocolVersion {
				t.Errorf("Version = %d, want %d", header.Version, ProtocolVersion)
			}
			if header.Type != tt.streamType {
				t.Errorf("Type = %v, want %v", header.Type, tt.streamType)
			}
		})
	}
}

func TestReadHeader_InvalidMagic(t *testing.T) {
	// Create invalid header with wrong magic
	buf := bytes.NewBuffer([]byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x01})

	_, err := ReadHeader(buf)
	if err != ErrInvalidMagic {
		t.Errorf("ReadHeader() error = %v, want %v", err, ErrInvalidMagic)
	}
}

func TestReadHeader_UnsupportedVersion(t *testing.T) {
	var buf bytes.Buffer

	// Write magic
	buf.Write([]byte{0x51, 0x53, 0x48, 0x53}) // "QSSH"
	// Write unsupported version
	buf.WriteByte(99)
	// Write type
	buf.WriteByte(0x01)

	_, err := ReadHeader(&buf)
	if err == nil {
		t.Error("ReadHeader() should return error for unsupported version")
	}
}

func TestTryReadHeader_SSHStream(t *testing.T) {
	var buf bytes.Buffer
	WriteHeader(&buf, StreamTypeSSH)

	header, peeked, err := TryReadHeader(&buf)
	if err != nil {
		t.Fatalf("TryReadHeader() error = %v", err)
	}
	if peeked != nil {
		t.Error("TryReadHeader() should not return peeked data for SSH stream")
	}
	if header == nil {
		t.Fatal("TryReadHeader() header should not be nil")
	}
	if header.Type != StreamTypeSSH {
		t.Errorf("TryReadHeader() type = %v, want %v", header.Type, StreamTypeSSH)
	}
}

func TestTryReadHeader_NonSSHStream(t *testing.T) {
	// Create non-SSH data (doesn't start with magic)
	buf := bytes.NewBuffer([]byte("Hello, World!"))

	header, peeked, err := TryReadHeader(buf)
	if err != nil {
		t.Fatalf("TryReadHeader() error = %v", err)
	}
	if header != nil {
		t.Error("TryReadHeader() should return nil header for non-SSH stream")
	}
	if len(peeked) != HeaderSize {
		t.Errorf("TryReadHeader() peeked size = %d, want %d", len(peeked), HeaderSize)
	}
}

func TestPortForwardRequest(t *testing.T) {
	tests := []struct {
		name string
		host string
		port uint16
	}{
		{"localhost", "localhost", 8080},
		{"IP address", "192.168.1.1", 22},
		{"domain", "example.com", 443},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			req := &PortForwardRequest{
				Host: tt.host,
				Port: tt.port,
			}

			// Write request
			if err := WritePortForwardRequest(&buf, req); err != nil {
				t.Fatalf("WritePortForwardRequest() error = %v", err)
			}

			// Read request
			readReq, err := ReadPortForwardRequest(&buf)
			if err != nil {
				t.Fatalf("ReadPortForwardRequest() error = %v", err)
			}

			// Verify values
			if readReq.Host != tt.host {
				t.Errorf("Host = %v, want %v", readReq.Host, tt.host)
			}
			if readReq.Port != tt.port {
				t.Errorf("Port = %v, want %v", readReq.Port, tt.port)
			}
		})
	}
}

func TestPortForwardRequest_HostTooLong(t *testing.T) {
	var buf bytes.Buffer
	req := &PortForwardRequest{
		Host: string(make([]byte, 256)), // Host name > 255 bytes
		Port: 8080,
	}

	err := WritePortForwardRequest(&buf, req)
	if err == nil {
		t.Error("WritePortForwardRequest() should return error for host name > 255 bytes")
	}
}
