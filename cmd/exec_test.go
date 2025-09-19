package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "exec help",
			args:    []string{"exec", "--help"},
			wantErr: false,
			contains: []string{
				"Execute commands on pods",
				"run",
				"shell",
			},
		},
		{
			name:    "exec run help",
			args:    []string{"exec", "run", "--help"},
			wantErr: false,
			contains: []string{
				"Execute a command on a pod",
				"--namespace",
				"--container",
				"--interactive",
				"--tty",
			},
		},
		{
			name:    "exec shell help",
			args:    []string{"exec", "shell", "--help"},
			wantErr: false,
			contains: []string{
				"Start an interactive shell on a pod",
				"--namespace",
				"--container",
				"--shell",
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

func TestExecCommandArguments(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "exec run missing arguments",
			args:    []string{"exec", "run"},
			wantErr: true,
		},
		{
			name:    "exec run missing command",
			args:    []string{"exec", "run", "pod-name"},
			wantErr: true,
		},
		{
			name:    "exec shell missing pod name",
			args:    []string{"exec", "shell"},
			wantErr: true,
		},
		{
			name:    "exec shell too many arguments",
			args:    []string{"exec", "shell", "pod1", "pod2"},
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

func TestExecCommandStructure(t *testing.T) {
	cmd := newExecCmd()

	// Verify basic properties
	assert.Equal(t, "exec", cmd.Use)
	assert.Contains(t, cmd.Short, "Execute commands on pods")
	assert.Contains(t, cmd.Long, "Execute commands on Kubernetes pods")

	// Verify subcommands are registered
	subcommands := cmd.Commands()
	expectedCommands := []string{"run", "shell"}

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

func TestExecRunCommandStructure(t *testing.T) {
	cmd := newExecRunCmd()

	// Verify basic properties
	assert.Equal(t, "run <pod-name> -- <command> [args...]", cmd.Use)
	assert.Contains(t, cmd.Short, "Execute a command on a pod")

	// Verify flags are present
	flags := cmd.Flags()

	expectedFlags := []string{
		"namespace",
		"container",
		"interactive",
		"tty",
	}

	for _, flagName := range expectedFlags {
		flag := flags.Lookup(flagName)
		assert.NotNil(t, flag, "Expected flag %s not found", flagName)
	}

	// Test flag aliases
	assert.NotNil(t, flags.ShorthandLookup("n"), "namespace flag should have shorthand 'n'")
	assert.NotNil(t, flags.ShorthandLookup("c"), "container flag should have shorthand 'c'")
	assert.NotNil(t, flags.ShorthandLookup("i"), "interactive flag should have shorthand 'i'")
	assert.NotNil(t, flags.ShorthandLookup("t"), "tty flag should have shorthand 't'")
}

func TestExecShellCommandStructure(t *testing.T) {
	cmd := newExecShellCmd()

	// Verify basic properties
	assert.Equal(t, "shell <pod-name>", cmd.Use)
	assert.Contains(t, cmd.Short, "Start an interactive shell on a pod")

	// Verify flags are present
	flags := cmd.Flags()

	expectedFlags := []string{
		"namespace",
		"container",
		"shell",
	}

	for _, flagName := range expectedFlags {
		flag := flags.Lookup(flagName)
		assert.NotNil(t, flag, "Expected flag %s not found", flagName)
	}

	// Test flag aliases
	assert.NotNil(t, flags.ShorthandLookup("n"), "namespace flag should have shorthand 'n'")
	assert.NotNil(t, flags.ShorthandLookup("c"), "container flag should have shorthand 'c'")
	assert.NotNil(t, flags.ShorthandLookup("s"), "shell flag should have shorthand 's'")
}
