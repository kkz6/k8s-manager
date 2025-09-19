package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestAllCommandsCanExecute tests that all commands can be created and executed without panicking
func TestAllCommandsCanExecute(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"root help", []string{"--help"}},
		{"config help", []string{"config", "--help"}},
		{"secrets help", []string{"secrets", "--help"}},
		{"pods help", []string{"pods", "--help"}},
		{"logs help", []string{"logs", "--help"}},
		{"exec help", []string{"exec", "--help"}},
		{"version", []string{"version"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd("test")
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tc.args)

			// Should not panic and should execute successfully for help commands
			err := cmd.Execute()
			assert.NoError(t, err, "Command should execute without error")

			// Should produce some output
			output := buf.String()
			assert.NotEmpty(t, output, "Command should produce output")
		})
	}
}

// TestCommandStructure tests that all expected commands are registered
func TestCommandStructure(t *testing.T) {
	cmd := newRootCmd("test")

	// Get all subcommands
	subcommands := cmd.Commands()
	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Name()
	}

	// Check that all expected commands are present
	expectedCommands := []string{"config", "secrets", "pods", "logs", "exec", "version"}
	for _, expected := range expectedCommands {
		assert.Contains(t, commandNames, expected, "Expected command %s should be registered", expected)
	}
}

// TestCommandsHaveHelp tests that all commands have help text
func TestCommandsHaveHelp(t *testing.T) {
	cmd := newRootCmd("test")

	// Test root command
	assert.NotEmpty(t, cmd.Short, "Root command should have short description")
	assert.NotEmpty(t, cmd.Long, "Root command should have long description")

	// Test subcommands
	for _, subcmd := range cmd.Commands() {
		assert.NotEmpty(t, subcmd.Short, "Command %s should have short description", subcmd.Name())
	}
}

// TestSubcommandStructure tests specific subcommand structures
func TestSubcommandStructure(t *testing.T) {
	testCases := []struct {
		commandName         string
		expectedSubcommands []string
	}{
		{"config", []string{"init", "show", "set", "validate"}},
		{"secrets", []string{"list", "get", "create", "update", "delete", "decode"}},
		{"pods", []string{"list", "get", "restart", "delete", "ssh"}},
		{"exec", []string{"run", "shell"}},
	}

	for _, tc := range testCases {
		t.Run(tc.commandName, func(t *testing.T) {
			rootCmd := newRootCmd("test")
			var targetCmd *cobra.Command

			// Find the target command
			for _, subcmd := range rootCmd.Commands() {
				if subcmd.Name() == tc.commandName {
					targetCmd = subcmd
					break
				}
			}

			assert.NotNil(t, targetCmd, "Command %s should exist", tc.commandName)

			if targetCmd != nil {
				subcommands := targetCmd.Commands()
				subcommandNames := make([]string, len(subcommands))
				for i, subcmd := range subcommands {
					subcommandNames[i] = subcmd.Name()
				}

				for _, expected := range tc.expectedSubcommands {
					assert.Contains(t, subcommandNames, expected,
						"Command %s should have subcommand %s", tc.commandName, expected)
				}
			}
		})
	}
}
