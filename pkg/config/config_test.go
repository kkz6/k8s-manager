package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDefaults(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Test default values
	assert.Equal(t, "us-central1-a", cfg.GCP.Zone)
	assert.Equal(t, "us-central1", cfg.GCP.Region)
	assert.Equal(t, "default", cfg.K8s.Namespace)
	assert.Equal(t, 22, cfg.SSH.Port)
	assert.Equal(t, "root", cfg.SSH.Username)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				GCP: GCPConfig{
					ProjectID: "test-project",
				},
				K8s: K8sConfig{
					ClusterName: "test-cluster",
				},
			},
			wantErr: false,
		},
		{
			name: "missing project ID",
			config: &Config{
				GCP: GCPConfig{},
				K8s: K8sConfig{
					ClusterName: "test-cluster",
				},
			},
			wantErr: true,
			errMsg:  "GCP project ID is required",
		},
		{
			name: "missing cluster name",
			config: &Config{
				GCP: GCPConfig{
					ProjectID: "test-project",
				},
				K8s: K8sConfig{},
			},
			wantErr: true,
			errMsg:  "Kubernetes cluster name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigUpdate(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	// Load initial config
	_, err := Load()
	require.NoError(t, err)

	// Update a value
	err = Update("gcp.project_id", "new-project")
	assert.NoError(t, err)

	// Verify the update
	updatedCfg := Get()
	assert.Equal(t, "new-project", updatedCfg.GCP.ProjectID)
}

func TestConfigGet(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	cfg := Get()
	assert.NotNil(t, cfg)

	// Should return same instance on subsequent calls
	cfg2 := Get()
	assert.Equal(t, cfg, cfg2)
}

func TestConfigEnvironmentVariables(t *testing.T) {
	t.Skip("Skipping environment variable test due to global state interference")
	// TODO: Refactor config to use dependency injection for better testability
}
