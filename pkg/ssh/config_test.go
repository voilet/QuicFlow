package ssh

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	if config.Shell != "/bin/sh" {
		t.Errorf("Shell = %v, want /bin/sh", config.Shell)
	}
	if config.MaxAuthTries != 3 {
		t.Errorf("MaxAuthTries = %v, want 3", config.MaxAuthTries)
	}
	if !config.AllowPty {
		t.Error("AllowPty should be true by default")
	}
	if !config.AllowTcpForwarding {
		t.Error("AllowTcpForwarding should be true by default")
	}
	if !config.PasswordAuth {
		t.Error("PasswordAuth should be true by default")
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *ServerConfig
		expectErr bool
	}{
		{
			name:      "valid with password auth",
			config:    &ServerConfig{PasswordAuth: true, MaxAuthTries: 3},
			expectErr: false,
		},
		{
			name: "valid with public key auth",
			config: &ServerConfig{
				PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
					return nil, nil
				},
				MaxAuthTries: 3,
			},
			expectErr: false,
		},
		{
			name:      "valid with no client auth",
			config:    &ServerConfig{NoClientAuth: true, MaxAuthTries: 3},
			expectErr: false,
		},
		{
			name:      "invalid - no auth method",
			config:    &ServerConfig{MaxAuthTries: 3},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}

func TestServerConfig_BuildSSHConfig(t *testing.T) {
	config := &ServerConfig{
		PasswordAuth: true,
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return nil, nil
		},
		MaxAuthTries: 5,
		Banner:       "Welcome!",
	}

	sshConfig, err := config.BuildSSHConfig()
	if err != nil {
		t.Fatalf("BuildSSHConfig() error = %v", err)
	}

	if sshConfig.MaxAuthTries != 5 {
		t.Errorf("MaxAuthTries = %v, want 5", sshConfig.MaxAuthTries)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.User != "root" {
		t.Errorf("User = %v, want root", config.User)
	}
	if config.Timeout != 30e9 { // 30 seconds in nanoseconds
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}
}

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *ClientConfig
		expectErr bool
	}{
		{
			name:      "valid with password",
			config:    &ClientConfig{User: "test", Password: "secret"},
			expectErr: false,
		},
		{
			name:      "valid with private key path",
			config:    &ClientConfig{User: "test", PrivateKeyPath: "/path/to/key"},
			expectErr: false,
		},
		{
			name:      "valid with private key data",
			config:    &ClientConfig{User: "test", PrivateKey: []byte("key data")},
			expectErr: false,
		},
		{
			name:      "invalid - no user",
			config:    &ClientConfig{Password: "secret"},
			expectErr: true,
		},
		{
			name:      "invalid - no auth",
			config:    &ClientConfig{User: "test"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr = %v", err, tt.expectErr)
			}
		})
	}
}

func TestDefaultSessionConfig(t *testing.T) {
	config := DefaultSessionConfig()

	if config.Term != "xterm-256color" {
		t.Errorf("Term = %v, want xterm-256color", config.Term)
	}
	if config.Width != 80 {
		t.Errorf("Width = %v, want 80", config.Width)
	}
	if config.Height != 24 {
		t.Errorf("Height = %v, want 24", config.Height)
	}
}
