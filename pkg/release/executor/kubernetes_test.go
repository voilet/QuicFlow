package executor

import (
	"strings"
	"testing"

	"github.com/voilet/quic-flow/pkg/release/models"
)

func TestK8sCommandBuilder_BuildApplyCommand(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceName: "my-app",
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildApplyCommand("/path/to/manifest.yaml")

	expectedParts := []string{
		"kubectl apply",
		"-f /path/to/manifest.yaml",
		"-n production",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}

func TestK8sCommandBuilder_DefaultNamespace(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		ResourceName: "my-app",
		// No namespace specified
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildApplyCommand("/path/to/manifest.yaml")

	if !strings.Contains(cmd, "-n default") {
		t.Errorf("expected default namespace, got: %s", cmd)
	}
}

func TestK8sCommandBuilder_BuildSetImageCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.KubernetesDeployConfig
		newImage string
		contains []string
	}{
		{
			name: "Basic set image",
			config: &models.KubernetesDeployConfig{
				Namespace:     "production",
				ResourceType:  "deployment",
				ResourceName:  "my-app",
				ContainerName: "app-container",
			},
			newImage: "nginx:v2.0",
			contains: []string{
				"kubectl set image",
				"deployment/my-app",
				"app-container=nginx:v2.0",
				"-n production",
			},
		},
		{
			name: "Default resource type",
			config: &models.KubernetesDeployConfig{
				Namespace:     "production",
				ResourceName:  "my-app",
				ContainerName: "app",
			},
			newImage: "nginx:v2.0",
			contains: []string{
				"deployment/my-app",
			},
		},
		{
			name: "Container name fallback to resource name",
			config: &models.KubernetesDeployConfig{
				Namespace:    "production",
				ResourceName: "my-app",
			},
			newImage: "nginx:v2.0",
			contains: []string{
				"my-app=nginx:v2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewK8sCommandBuilder(tt.config)
			cmd := builder.BuildSetImageCommand(tt.newImage)

			for _, part := range tt.contains {
				if !strings.Contains(cmd, part) {
					t.Errorf("expected command to contain '%s', got: %s", part, cmd)
				}
			}
		})
	}
}

func TestK8sCommandBuilder_BuildScaleCommand(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceType: "deployment",
		ResourceName: "my-app",
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildScaleCommand(3)

	expectedParts := []string{
		"kubectl scale",
		"deployment/my-app",
		"--replicas=3",
		"-n production",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}

func TestK8sCommandBuilder_BuildRolloutStatusCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.KubernetesDeployConfig
		contains []string
	}{
		{
			name: "With custom timeout",
			config: &models.KubernetesDeployConfig{
				Namespace:      "production",
				ResourceName:   "my-app",
				RolloutTimeout: 600,
			},
			contains: []string{
				"kubectl rollout status",
				"deployment/my-app",
				"--timeout=600s",
			},
		},
		{
			name: "Default timeout",
			config: &models.KubernetesDeployConfig{
				Namespace:    "production",
				ResourceName: "my-app",
			},
			contains: []string{
				"--timeout=300s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewK8sCommandBuilder(tt.config)
			cmd := builder.BuildRolloutStatusCommand()

			for _, part := range tt.contains {
				if !strings.Contains(cmd, part) {
					t.Errorf("expected command to contain '%s', got: %s", part, cmd)
				}
			}
		})
	}
}

func TestK8sCommandBuilder_BuildRolloutUndoCommand(t *testing.T) {
	tests := []struct {
		name       string
		config     *models.KubernetesDeployConfig
		toRevision int
		contains   []string
	}{
		{
			name: "With specific revision",
			config: &models.KubernetesDeployConfig{
				Namespace:    "production",
				ResourceName: "my-app",
			},
			toRevision: 5,
			contains: []string{
				"kubectl rollout undo",
				"deployment/my-app",
				"--to-revision=5",
			},
		},
		{
			name: "Without revision (rollback to previous)",
			config: &models.KubernetesDeployConfig{
				Namespace:    "production",
				ResourceName: "my-app",
			},
			toRevision: 0,
			contains: []string{
				"kubectl rollout undo",
				"deployment/my-app",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewK8sCommandBuilder(tt.config)
			cmd := builder.BuildRolloutUndoCommand(tt.toRevision)

			for _, part := range tt.contains {
				if !strings.Contains(cmd, part) {
					t.Errorf("expected command to contain '%s', got: %s", part, cmd)
				}
			}

			// Check that revision is NOT present when 0
			if tt.toRevision == 0 && strings.Contains(cmd, "--to-revision") {
				t.Error("--to-revision should not be present when revision is 0")
			}
		})
	}
}

func TestK8sCommandBuilder_BuildDeleteCommand(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceName: "my-app",
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildDeleteCommand("/path/to/manifest.yaml")

	expectedParts := []string{
		"kubectl delete",
		"-f /path/to/manifest.yaml",
		"-n production",
		"--ignore-not-found=true",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}

func TestK8sCommandBuilder_BuildGetCommand(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceType: "deployment",
		ResourceName: "my-app",
	}

	builder := NewK8sCommandBuilder(config)

	// Test with specific format
	cmd := builder.BuildGetCommand("json")
	if !strings.Contains(cmd, "-o json") {
		t.Errorf("expected '-o json', got: %s", cmd)
	}

	// Test default format
	cmd = builder.BuildGetCommand("")
	if !strings.Contains(cmd, "-o wide") {
		t.Errorf("expected '-o wide' for default, got: %s", cmd)
	}
}

func TestK8sCommandBuilder_BuildGetPodsCommand(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceName: "my-app",
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildGetPodsCommand()

	expectedParts := []string{
		"kubectl get pods",
		"-l app=my-app",
		"-o wide",
		"-n production",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}

func TestK8sCommandBuilder_BuildLogsCommand(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:     "production",
		ContainerName: "app-container",
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildLogsCommand("my-pod-abc123", 100)

	expectedParts := []string{
		"kubectl logs",
		"my-pod-abc123",
		"-c app-container",
		"--tail=100",
		"-n production",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}

func TestK8sCommandBuilder_BuildCreateSecretCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.KubernetesDeployConfig
		isEmpty  bool
		contains []string
	}{
		{
			name: "With registry credentials",
			config: &models.KubernetesDeployConfig{
				Namespace:    "production",
				Registry:     "registry.example.com",
				RegistryUser: "user",
				RegistryPass: "pass123",
			},
			isEmpty: false,
			contains: []string{
				"kubectl create secret docker-registry",
				"--docker-server=registry.example.com",
				"--docker-username=user",
				"--docker-password=pass123",
				"--dry-run=client",
				"kubectl apply -f -",
			},
		},
		{
			name: "Without credentials",
			config: &models.KubernetesDeployConfig{
				Namespace: "production",
			},
			isEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewK8sCommandBuilder(tt.config)
			cmd := builder.BuildCreateSecretCommand("my-registry-secret")

			if tt.isEmpty {
				if cmd != "" {
					t.Errorf("expected empty command, got: %s", cmd)
				}
				return
			}

			for _, part := range tt.contains {
				if !strings.Contains(cmd, part) {
					t.Errorf("expected command to contain '%s', got: %s", part, cmd)
				}
			}
		})
	}
}

func TestK8sCommandBuilder_WithKubeConfig(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceName: "my-app",
		KubeConfig:   "/path/to/kubeconfig",
		KubeContext:  "my-context",
	}

	builder := NewK8sCommandBuilder(config)
	cmd := builder.BuildApplyCommand("/path/to/manifest.yaml")

	expectedParts := []string{
		"--kubeconfig /path/to/kubeconfig",
		"--context my-context",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd, part) {
			t.Errorf("expected command to contain '%s', got: %s", part, cmd)
		}
	}
}

func TestK8sCommandBuilder_BuildDeployScript(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:       "production",
		ResourceName:    "my-app",
		ResourceType:    "deployment",
		ImagePullSecret: "my-registry-secret",
		Registry:        "registry.example.com",
		RegistryUser:    "user",
		RegistryPass:    "pass",
		RolloutTimeout:  300,
	}

	builder := NewK8sCommandBuilder(config)
	yamlContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 2`

	script, err := builder.BuildDeployScript(yamlContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSections := []string{
		"#!/bin/bash",
		"set -e",
		"Create/Update image pull secret",
		"Apply Kubernetes resources",
		"Wait for rollout",
		"Show deployment status",
		"kubectl get pods",
		"deployment completed successfully",
	}

	for _, section := range expectedSections {
		if !strings.Contains(script, section) {
			t.Errorf("expected script to contain '%s'", section)
		}
	}
}

func TestK8sCommandBuilder_BuildUpdateScript(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:     "production",
		ResourceName:  "my-app",
		ResourceType:  "deployment",
		ContainerName: "app-container",
	}

	builder := NewK8sCommandBuilder(config)
	script, err := builder.BuildUpdateScript("nginx:v2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"kubectl set image",
		"app-container=nginx:v2.0",
		"kubectl rollout status",
		"update completed successfully",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("expected script to contain '%s'", part)
		}
	}
}

func TestK8sCommandBuilder_BuildRollbackScript(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:    "production",
		ResourceName: "my-app",
		ResourceType: "deployment",
	}

	builder := NewK8sCommandBuilder(config)
	script, err := builder.BuildRollbackScript(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"Show rollout history",
		"kubectl rollout undo",
		"--to-revision=3",
		"Wait for rollout",
		"rollback completed successfully",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("expected script to contain '%s'", part)
		}
	}
}

func TestK8sCommandBuilder_BuildUninstallScript(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:       "production",
		ResourceName:    "my-app",
		ResourceType:    "deployment",
		ImagePullSecret: "my-registry-secret",
	}

	builder := NewK8sCommandBuilder(config)
	script, err := builder.BuildUninstallScript("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"#!/bin/bash",
		"Delete Kubernetes resource",
		"Delete image pull secret",
		"--ignore-not-found=true",
		"deleted successfully",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("expected script to contain '%s'", part)
		}
	}
}

func TestK8sCommandBuilder_GenerateDeploymentYAML(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:       "production",
		ResourceName:    "my-app",
		ContainerName:   "app-container",
		Image:           "nginx:latest",
		Replicas:        3,
		ImagePullPolicy: "Always",
		CPURequest:      "100m",
		MemoryRequest:   "128Mi",
		CPULimit:        "500m",
		MemoryLimit:     "512Mi",
		ImagePullSecret: "my-registry-secret",
		ServicePorts: []models.K8sPort{
			{Name: "http", Port: 80, TargetPort: 8080, Protocol: "TCP"},
		},
		Environment: map[string]string{
			"APP_ENV": "production",
		},
	}

	builder := NewK8sCommandBuilder(config)
	yaml := builder.GenerateDeploymentYAML()

	expectedParts := []string{
		"apiVersion: apps/v1",
		"kind: Deployment",
		"name: my-app",
		"namespace: production",
		"replicas: 3",
		"image: nginx:latest",
		"imagePullPolicy: Always",
		"containerPort: 8080",
		"cpu: 100m",
		"memory: 128Mi",
		"name: APP_ENV",
		"imagePullSecrets:",
		"name: my-registry-secret",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("expected YAML to contain '%s'", part)
		}
	}
}

func TestK8sCommandBuilder_GenerateDeploymentYAML_Defaults(t *testing.T) {
	config := &models.KubernetesDeployConfig{
		Namespace:     "production",
		ResourceName:  "my-app",
		ContainerName: "app",
		Image:         "nginx:latest",
		// No replicas - should default to 1
	}

	builder := NewK8sCommandBuilder(config)
	yaml := builder.GenerateDeploymentYAML()

	if !strings.Contains(yaml, "replicas: 1") {
		t.Error("expected default replicas to be 1")
	}

	if !strings.Contains(yaml, "imagePullPolicy: IfNotPresent") {
		t.Error("expected default imagePullPolicy to be IfNotPresent")
	}
}
