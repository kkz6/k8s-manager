package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogsCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "logs help",
			args:    []string{"logs", "--help"},
			wantErr: false,
			contains: []string{
				"View pod logs",
				"View and follow logs from Kubernetes pods",
				"--namespace",
				"--container",
				"--follow",
				"--previous",
				"--since",
				"--tail",
				"--timestamps",
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

func TestLogsCommandArguments(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "logs missing pod name",
			args:    []string{"logs"},
			wantErr: true,
		},
		{
			name:    "logs too many arguments",
			args:    []string{"logs", "pod1", "pod2"},
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

func TestLogsCommandStructure(t *testing.T) {
	cmd := newLogsCmd()
	
	// Verify basic properties
	assert.Equal(t, "logs <pod-name>", cmd.Use)
	assert.Contains(t, cmd.Short, "View pod logs")
	assert.Contains(t, cmd.Long, "View and follow logs from Kubernetes pods")

	// Verify flags are present
	flags := cmd.Flags()
	
	expectedFlags := []string{
		"namespace",
		"container", 
		"follow",
		"previous",
		"since",
		"since-time",
		"tail",
		"timestamps",
	}

	for _, flagName := range expectedFlags {
		flag := flags.Lookup(flagName)
		assert.NotNil(t, flag, "Expected flag %s not found", flagName)
	}

	// Test flag aliases
	assert.NotNil(t, flags.ShorthandLookup("n"), "namespace flag should have shorthand 'n'")
	assert.NotNil(t, flags.ShorthandLookup("c"), "container flag should have shorthand 'c'")
	assert.NotNil(t, flags.ShorthandLookup("f"), "follow flag should have shorthand 'f'")
	assert.NotNil(t, flags.ShorthandLookup("p"), "previous flag should have shorthand 'p'")
}
