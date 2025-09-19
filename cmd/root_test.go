package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "help command",
			args:    []string{"--help"},
			wantErr: false,
			contains: []string{
				"K8s Manager is a comprehensive CLI tool",
				"config",
				"secrets",
				"pods",
				"logs",
				"exec",
			},
		},
		{
			name:    "no arguments shows help",
			args:    []string{},
			wantErr: false,
			contains: []string{
				"Available Commands:",
				"config",
				"secrets",
				"pods",
			},
		},
		{
			name:    "version command",
			args:    []string{"version"},
			wantErr: false,
			contains: []string{
				"version",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd("test-version")
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

func TestRootCommandStructure(t *testing.T) {
	cmd := newRootCmd("test")

	// Verify basic properties
	assert.Equal(t, "k8s-manager", cmd.Use)
	assert.Contains(t, cmd.Short, "Kubernetes cluster manager")
	assert.Contains(t, cmd.Long, "K8s Manager is a comprehensive CLI tool")

	// Verify subcommands are registered
	subcommands := cmd.Commands()
	expectedCommands := []string{"version", "config", "secrets", "pods", "logs", "exec"}

	for _, expected := range expectedCommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subcommand %s not found", expected)
	}
}

func TestExecuteFunction(t *testing.T) {
	// Test successful execution
	err := Execute("test-version")
	// Since we're not providing any args, it should show help and return no error
	assert.NoError(t, err)
}
