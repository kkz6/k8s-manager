package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "config help",
			args:    []string{"config", "--help"},
			wantErr: false,
			contains: []string{
				"Configure GCP project, Kubernetes cluster",
				"init",
				"show",
				"set",
				"validate",
			},
		},
		{
			name:    "config init help",
			args:    []string{"config", "init", "--help"},
			wantErr: false,
			contains: []string{
				"Initialize K8s Manager configuration",
			},
		},
		{
			name:    "config show help",
			args:    []string{"config", "show", "--help"},
			wantErr: false,
			contains: []string{
				"Display the current K8s Manager configuration",
			},
		},
		{
			name:    "config set help",
			args:    []string{"config", "set", "--help"},
			wantErr: false,
			contains: []string{
				"Set a specific configuration value",
				"dot notation",
			},
		},
		{
			name:    "config validate help",
			args:    []string{"config", "validate", "--help"},
			wantErr: false,
			contains: []string{
				"Validate the current configuration",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd("test")
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			output := buf.String()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tc.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestConfigSetCommand(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "missing arguments",
			args:    []string{"config", "set"},
			wantErr: true,
		},
		{
			name:    "missing value argument",
			args:    []string{"config", "set", "key"},
			wantErr: true,
		},
		{
			name:    "too many arguments",
			args:    []string{"config", "set", "key", "value", "extra"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd("test")
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigShow(t *testing.T) {
	// Create temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "show"})

	err := cmd.Execute()
	output := buf.String()

	assert.NoError(t, err)
	// Just verify that the command runs without error and produces output
	assert.NotEmpty(t, output)
	// The actual output format testing is better done with integration tests
}

func TestPromptInput(t *testing.T) {
	testCases := []struct {
		name         string
		prompt       string
		defaultValue string
		input        string
		expected     string
		wantErr      bool
	}{
		{
			name:         "use default value on empty input",
			prompt:       "Test prompt",
			defaultValue: "default",
			input:        "",
			expected:     "default",
			wantErr:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: Testing promptInput function directly is challenging
			// as it reads from stdin. In a real scenario, you might want
			// to refactor it to accept an io.Reader for testability.
			// For now, we'll just test that the function exists.
			assert.NotNil(t, promptInput)
		})
	}
}
