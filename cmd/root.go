package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-manager",
		Short: "Kubernetes cluster manager for GCP",
		Long: `K8s Manager is a comprehensive CLI tool for managing Kubernetes clusters on Google Cloud Platform.
It provides functionality for configuration management, secrets handling, pod operations, and more.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newSecretsCmd())
	cmd.AddCommand(newPodsCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newExecCmd())

	return cmd
}

// Execute invokes the command.
func Execute(version string) error {
	if err := newRootCmd(version).Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}

	return nil
}
