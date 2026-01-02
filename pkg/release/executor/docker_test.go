package executor

import (
	"strings"
	"testing"

	"github.com/voilet/quic-flow/pkg/release/models"
)

func TestDockerCommandBuilder_BuildRunCommand(t *testing.T) {
	tests := []struct {
		name      string
		config    *models.ContainerDeployConfig
		expectErr bool
		contains  []string
	}{
		{
			name: "Basic run command",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
			},
			expectErr: false,
			contains:  []string{"docker", "run", "-d", "--name", "web-server", "nginx:latest"},
		},
		{
			name: "With port mapping",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				Ports: []models.PortMapping{
					{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
				},
			},
			expectErr: false,
			contains:  []string{"-p", "8080:80"},
		},
		{
			name: "With environment variables",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				Environment: map[string]string{
					"DEBUG": "true",
				},
			},
			expectErr: false,
			contains:  []string{"-e", "DEBUG=true"},
		},
		{
			name: "With volume mount",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				Volumes: []models.VolumeMount{
					{HostPath: "/data", ContainerPath: "/app/data", ReadOnly: false},
				},
			},
			expectErr: false,
			contains:  []string{"-v", "/data:/app/data"},
		},
		{
			name: "With memory limit",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				MemoryLimit:   "512m",
			},
			expectErr: false,
			contains:  []string{"--memory", "512m"},
		},
		{
			name: "With CPU limit",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				CPULimit:      "0.5",
			},
			expectErr: false,
			contains:  []string{"--cpus", "0.5"},
		},
		{
			name: "With restart policy",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				RestartPolicy: "always",
			},
			expectErr: false,
			contains:  []string{"--restart", "always"},
		},
		{
			name: "With network mode",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				NetworkMode:   "host",
			},
			expectErr: false,
			contains:  []string{"--network", "host"},
		},
		{
			name: "With health check",
			config: &models.ContainerDeployConfig{
				Image:         "nginx:latest",
				ContainerName: "web-server",
				HealthCheck: &models.ContainerHealthCheck{
					Command:  []string{"curl", "-f", "http://localhost/"},
					Interval: 30,
					Timeout:  10,
					Retries:  3,
				},
			},
			expectErr: false,
			contains:  []string{"--health-cmd", "--health-interval", "--health-timeout", "--health-retries"},
		},
		{
			name:      "Nil config returns error",
			config:    nil,
			expectErr: true,
		},
		{
			name: "Empty image returns error",
			config: &models.ContainerDeployConfig{
				ContainerName: "web-server",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewDockerCommandBuilder(tt.config)
			cmd, err := builder.BuildRunCommand()

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("expected command to contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

func TestDockerCommandBuilder_BuildPullCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.ContainerDeployConfig
		contains []string
	}{
		{
			name: "Basic pull",
			config: &models.ContainerDeployConfig{
				Image: "nginx:latest",
			},
			contains: []string{"docker", "pull", "nginx:latest"},
		},
		{
			name: "With platform",
			config: &models.ContainerDeployConfig{
				Image:    "nginx:latest",
				Platform: "linux/amd64",
			},
			contains: []string{"docker", "pull", "--platform", "linux/amd64", "nginx:latest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewDockerCommandBuilder(tt.config)
			cmd := builder.BuildPullCommand()

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("expected command to contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

func TestDockerCommandBuilder_BuildStopCommand(t *testing.T) {
	config := &models.ContainerDeployConfig{
		Image:       "nginx:latest",
		StopTimeout: 30,
	}
	builder := NewDockerCommandBuilder(config)
	cmd := builder.BuildStopCommand("my-container")

	if !strings.Contains(cmd, "docker stop") {
		t.Error("expected 'docker stop' in command")
	}
	if !strings.Contains(cmd, "-t 30") {
		t.Error("expected '-t 30' timeout in command")
	}
	if !strings.Contains(cmd, "my-container") {
		t.Error("expected container name in command")
	}
}

func TestDockerCommandBuilder_BuildRemoveCommand(t *testing.T) {
	config := &models.ContainerDeployConfig{Image: "nginx:latest"}
	builder := NewDockerCommandBuilder(config)
	cmd := builder.BuildRemoveCommand("my-container")

	if !strings.Contains(cmd, "docker rm -f") {
		t.Error("expected 'docker rm -f' in command")
	}
	if !strings.Contains(cmd, "my-container") {
		t.Error("expected container name in command")
	}
}

func TestDockerCommandBuilder_BuildLoginCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.ContainerDeployConfig
		isEmpty  bool
		contains []string
	}{
		{
			name: "With registry credentials",
			config: &models.ContainerDeployConfig{
				Image:        "nginx:latest",
				Registry:     "registry.example.com",
				RegistryUser: "user",
				RegistryPass: "pass123",
			},
			isEmpty:  false,
			contains: []string{"docker login", "registry.example.com", "-u", "user"},
		},
		{
			name: "Without credentials",
			config: &models.ContainerDeployConfig{
				Image: "nginx:latest",
			},
			isEmpty: true,
		},
		{
			name: "Partial credentials - missing password",
			config: &models.ContainerDeployConfig{
				Image:        "nginx:latest",
				Registry:     "registry.example.com",
				RegistryUser: "user",
			},
			isEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewDockerCommandBuilder(tt.config)
			cmd := builder.BuildLoginCommand()

			if tt.isEmpty {
				if cmd != "" {
					t.Errorf("expected empty command, got: %s", cmd)
				}
				return
			}

			for _, substr := range tt.contains {
				if !strings.Contains(cmd, substr) {
					t.Errorf("expected command to contain '%s', got: %s", substr, cmd)
				}
			}
		})
	}
}

func TestDockerCommandBuilder_BuildDeployScript(t *testing.T) {
	config := &models.ContainerDeployConfig{
		Image:          "nginx:latest",
		ContainerName:  "web-server",
		RemoveOld:      true,
		PullBeforeStop: false,
		Ports: []models.PortMapping{
			{HostPort: 80, ContainerPort: 80},
		},
	}

	builder := NewDockerCommandBuilder(config)
	script, err := builder.BuildDeployScript("web-server")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check script contains expected sections
	expectedSections := []string{
		"#!/bin/bash",
		"set -e",
		"docker stop",
		"docker rm",
		"docker pull",
		"docker run",
		"Deployment completed",
	}

	for _, section := range expectedSections {
		if !strings.Contains(script, section) {
			t.Errorf("expected script to contain '%s'", section)
		}
	}
}

func TestDockerCommandBuilder_BuildUninstallScript(t *testing.T) {
	config := &models.ContainerDeployConfig{
		Image:       "nginx:latest",
		StopTimeout: 10,
	}

	builder := NewDockerCommandBuilder(config)
	script := builder.BuildUninstallScript("web-server")

	expectedParts := []string{
		"#!/bin/bash",
		"docker stop",
		"docker rm",
		"Container removed",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("expected script to contain '%s'", part)
		}
	}
}

func TestGenerateDockerRunCommand(t *testing.T) {
	config := &models.ContainerDeployConfig{
		Image:         "nginx:latest",
		ContainerName: "test",
	}

	cmd, err := GenerateDockerRunCommand(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "docker run") {
		t.Error("expected 'docker run' in command")
	}
}

func TestDockerCommandBuilder_PrivilegedMode(t *testing.T) {
	config := &models.ContainerDeployConfig{
		Image:      "nginx:latest",
		Privileged: true,
	}

	builder := NewDockerCommandBuilder(config)
	cmd, err := builder.BuildRunCommand()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd, "--privileged") {
		t.Error("expected '--privileged' in command")
	}
}

func TestDockerCommandBuilder_Capabilities(t *testing.T) {
	config := &models.ContainerDeployConfig{
		Image:   "nginx:latest",
		CapAdd:  []string{"SYS_ADMIN", "NET_ADMIN"},
		CapDrop: []string{"MKNOD"},
	}

	builder := NewDockerCommandBuilder(config)
	cmd, err := builder.BuildRunCommand()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"--cap-add SYS_ADMIN",
		"--cap-add NET_ADMIN",
		"--cap-drop MKNOD",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}
