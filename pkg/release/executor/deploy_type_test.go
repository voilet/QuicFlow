package executor

import (
	"testing"

	"github.com/voilet/quic-flow/pkg/release/models"
)

func TestGetDeployTypeConfig(t *testing.T) {
	tests := []struct {
		name          string
		deployType    models.DeployType
		expectNil     bool
		expectedTool  string
		expectedSource string
	}{
		{
			name:          "Container config",
			deployType:    models.DeployTypeContainer,
			expectNil:     false,
			expectedTool:  "docker",
			expectedSource: "image_tag",
		},
		{
			name:          "Kubernetes config",
			deployType:    models.DeployTypeKubernetes,
			expectNil:     false,
			expectedTool:  "kubectl",
			expectedSource: "image_tag",
		},
		{
			name:          "GitPull config",
			deployType:    models.DeployTypeGitPull,
			expectNil:     false,
			expectedTool:  "git",
			expectedSource: "git_ref",
		},
		{
			name:          "Script config",
			deployType:    models.DeployTypeScript,
			expectNil:     false,
			expectedTool:  "bash",
			expectedSource: "version_number",
		},
		{
			name:       "Unknown type",
			deployType: "unknown",
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetDeployTypeConfig(tt.deployType)
			if tt.expectNil {
				if config != nil {
					t.Errorf("expected nil config, got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatalf("expected non-nil config")
			}

			if config.CoreTool != tt.expectedTool {
				t.Errorf("expected CoreTool %s, got %s", tt.expectedTool, config.CoreTool)
			}

			if config.VersionSource != tt.expectedSource {
				t.Errorf("expected VersionSource %s, got %s", tt.expectedSource, config.VersionSource)
			}
		})
	}
}

func TestContainerConfigOperations(t *testing.T) {
	config := GetDeployTypeConfig(models.DeployTypeContainer)
	if config == nil {
		t.Fatal("expected non-nil container config")
	}

	// Check expected operations
	expectedOps := []models.OperationType{
		models.OperationTypeInstall,
		models.OperationTypeUpdate,
		models.OperationTypeRollback,
		models.OperationTypeUninstall,
	}

	if len(config.Operations) != len(expectedOps) {
		t.Errorf("expected %d operations, got %d", len(expectedOps), len(config.Operations))
	}

	for i, expectedOp := range expectedOps {
		if i >= len(config.Operations) {
			break
		}
		if config.Operations[i].Operation != expectedOp {
			t.Errorf("operation %d: expected %s, got %s", i, expectedOp, config.Operations[i].Operation)
		}
	}

	// Check features
	if !config.SupportsHealthCheck {
		t.Error("Container should support health check")
	}
	if !config.SupportsAtomicRollback {
		t.Error("Container should support atomic rollback")
	}
}

func TestKubernetesConfigOperations(t *testing.T) {
	config := GetDeployTypeConfig(models.DeployTypeKubernetes)
	if config == nil {
		t.Fatal("expected non-nil kubernetes config")
	}

	// K8s should support health check and atomic rollback
	if !config.SupportsHealthCheck {
		t.Error("Kubernetes should support health check")
	}
	if !config.SupportsAtomicRollback {
		t.Error("Kubernetes should support atomic rollback")
	}
}

func TestGitPullConfigOperations(t *testing.T) {
	config := GetDeployTypeConfig(models.DeployTypeGitPull)
	if config == nil {
		t.Fatal("expected non-nil gitpull config")
	}

	// GitPull should NOT support health check or atomic rollback
	if config.SupportsHealthCheck {
		t.Error("GitPull should not support health check")
	}
	if config.SupportsAtomicRollback {
		t.Error("GitPull should not support atomic rollback")
	}
}

func TestDetermineActualOperation(t *testing.T) {
	tests := []struct {
		name          string
		deployType    models.DeployType
		currentStatus models.InstallStatus
		requestedOp   models.OperationType
		expectedOp    models.OperationType
	}{
		{
			name:          "Deploy on installed -> Update",
			deployType:    models.DeployTypeContainer,
			currentStatus: models.InstallStatusInstalled,
			requestedOp:   models.OperationTypeDeploy,
			expectedOp:    models.OperationTypeUpdate,
		},
		{
			name:          "Deploy on uninstalled -> Install",
			deployType:    models.DeployTypeContainer,
			currentStatus: models.InstallStatusUninstalled,
			requestedOp:   models.OperationTypeDeploy,
			expectedOp:    models.OperationTypeInstall,
		},
		{
			name:          "Deploy on failed -> Install",
			deployType:    models.DeployTypeContainer,
			currentStatus: models.InstallStatusFailed,
			requestedOp:   models.OperationTypeDeploy,
			expectedOp:    models.OperationTypeInstall,
		},
		{
			name:          "Deploy on unknown -> Install",
			deployType:    models.DeployTypeContainer,
			currentStatus: models.InstallStatusUnknown,
			requestedOp:   models.OperationTypeDeploy,
			expectedOp:    models.OperationTypeInstall,
		},
		{
			name:          "Explicit install passes through",
			deployType:    models.DeployTypeContainer,
			currentStatus: models.InstallStatusInstalled,
			requestedOp:   models.OperationTypeInstall,
			expectedOp:    models.OperationTypeInstall,
		},
		{
			name:          "Explicit rollback passes through",
			deployType:    models.DeployTypeContainer,
			currentStatus: models.InstallStatusInstalled,
			requestedOp:   models.OperationTypeRollback,
			expectedOp:    models.OperationTypeRollback,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineActualOperation(tt.deployType, tt.currentStatus, tt.requestedOp)
			if result != tt.expectedOp {
				t.Errorf("expected %s, got %s", tt.expectedOp, result)
			}
		})
	}
}

func TestGetOperationTimeout(t *testing.T) {
	tests := []struct {
		name       string
		deployType models.DeployType
		operation  models.OperationType
		minTimeout int
	}{
		{
			name:       "Container install timeout",
			deployType: models.DeployTypeContainer,
			operation:  models.OperationTypeInstall,
			minTimeout: 600,
		},
		{
			name:       "Container update timeout",
			deployType: models.DeployTypeContainer,
			operation:  models.OperationTypeUpdate,
			minTimeout: 600,
		},
		{
			name:       "K8s install timeout",
			deployType: models.DeployTypeKubernetes,
			operation:  models.OperationTypeInstall,
			minTimeout: 600,
		},
		{
			name:       "GitPull update timeout",
			deployType: models.DeployTypeGitPull,
			operation:  models.OperationTypeUpdate,
			minTimeout: 300,
		},
		{
			name:       "Unknown type fallback",
			deployType: "unknown",
			operation:  models.OperationTypeInstall,
			minTimeout: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := GetOperationTimeout(tt.deployType, tt.operation)
			if timeout < tt.minTimeout {
				t.Errorf("expected timeout >= %d, got %d", tt.minTimeout, timeout)
			}
		})
	}
}

func TestCanRollback(t *testing.T) {
	tests := []struct {
		name       string
		deployType models.DeployType
		operation  models.OperationType
		canRoll    bool
	}{
		{
			name:       "Container install can rollback",
			deployType: models.DeployTypeContainer,
			operation:  models.OperationTypeInstall,
			canRoll:    true,
		},
		{
			name:       "Container update can rollback",
			deployType: models.DeployTypeContainer,
			operation:  models.OperationTypeUpdate,
			canRoll:    true,
		},
		{
			name:       "Container rollback cannot rollback",
			deployType: models.DeployTypeContainer,
			operation:  models.OperationTypeRollback,
			canRoll:    false,
		},
		{
			name:       "K8s install can rollback",
			deployType: models.DeployTypeKubernetes,
			operation:  models.OperationTypeInstall,
			canRoll:    true,
		},
		{
			name:       "GitPull install can rollback",
			deployType: models.DeployTypeGitPull,
			operation:  models.OperationTypeInstall,
			canRoll:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanRollback(tt.deployType, tt.operation)
			if result != tt.canRoll {
				t.Errorf("expected CanRollback=%v, got %v", tt.canRoll, result)
			}
		})
	}
}

func TestGetVersionSource(t *testing.T) {
	tests := []struct {
		deployType models.DeployType
		expected   string
	}{
		{models.DeployTypeContainer, "image_tag"},
		{models.DeployTypeKubernetes, "image_tag"},
		{models.DeployTypeGitPull, "git_ref"},
		{models.DeployTypeScript, "version_number"},
		{"unknown", "version_number"},
	}

	for _, tt := range tests {
		t.Run(string(tt.deployType), func(t *testing.T) {
			result := GetVersionSource(tt.deployType)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSupportsHealthCheck(t *testing.T) {
	tests := []struct {
		deployType models.DeployType
		supports   bool
	}{
		{models.DeployTypeContainer, true},
		{models.DeployTypeKubernetes, true},
		{models.DeployTypeGitPull, false},
		{models.DeployTypeScript, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.deployType), func(t *testing.T) {
			result := SupportsHealthCheck(tt.deployType)
			if result != tt.supports {
				t.Errorf("expected %v, got %v", tt.supports, result)
			}
		})
	}
}
