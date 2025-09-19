package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretsCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "secrets help",
			args:    []string{"secrets", "--help"},
			wantErr: false,
			contains: []string{
				"Manage Kubernetes secrets",
				"list",
				"get",
				"create",
				"update",
				"delete",
				"decode",
			},
		},
		{
			name:    "secrets list help",
			args:    []string{"secrets", "list", "--help"},
			wantErr: false,
			contains: []string{
				"List all secrets in the namespace",
				"--namespace",
				"--all-namespaces",
			},
		},
		{
			name:    "secrets get help",
			args:    []string{"secrets", "get", "--help"},
			wantErr: false,
			contains: []string{
				"Get details of a specific secret",
				"--namespace",
				"--decode",
			},
		},
		{
			name:    "secrets create help",
			args:    []string{"secrets", "create", "--help"},
			wantErr: false,
			contains: []string{
				"Create a new secret",
				"--from-literal",
				"--from-file",
				"--type",
			},
		},
		{
			name:    "secrets update help",
			args:    []string{"secrets", "update", "--help"},
			wantErr: false,
			contains: []string{
				"Update an existing secret",
				"--from-literal",
				"--from-file",
				"--remove-key",
			},
		},
		{
			name:    "secrets delete help",
			args:    []string{"secrets", "delete", "--help"},
			wantErr: false,
			contains: []string{
				"Delete a secret",
				"--force",
			},
		},
		{
			name:    "secrets decode help",
			args:    []string{"secrets", "decode", "--help"},
			wantErr: false,
			contains: []string{
				"Decode a specific key from a secret",
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

func TestSecretsCommandArguments(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "secrets get missing argument",
			args:    []string{"secrets", "get"},
			wantErr: true,
		},
		{
			name:    "secrets create missing argument",
			args:    []string{"secrets", "create"},
			wantErr: true,
		},
		{
			name:    "secrets update missing argument",
			args:    []string{"secrets", "update"},
			wantErr: true,
		},
		{
			name:    "secrets delete missing argument",
			args:    []string{"secrets", "delete"},
			wantErr: true,
		},
		{
			name:    "secrets decode missing arguments",
			args:    []string{"secrets", "decode"},
			wantErr: true,
		},
		{
			name:    "secrets decode missing second argument",
			args:    []string{"secrets", "decode", "secret-name"},
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

func TestSecretsCommandStructure(t *testing.T) {
	cmd := newSecretsCmd()

	// Verify basic properties
	assert.Equal(t, "secrets", cmd.Use)
	assert.Contains(t, cmd.Short, "Manage Kubernetes secrets")
	assert.Contains(t, cmd.Long, "Create, view, update, and delete Kubernetes secrets")

	// Verify subcommands are registered
	subcommands := cmd.Commands()
	expectedCommands := []string{"list", "get", "create", "update", "delete", "decode"}

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
