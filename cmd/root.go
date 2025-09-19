package cmd

import (
	"fmt"
	"os"

	"github.com/karthickk/k8s-manager/pkg/ui"
	"github.com/spf13/cobra"
)

var interactiveMode bool

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-manager",
		Short: "Kubernetes cluster manager for GCP",
		Long: `K8s Manager is a comprehensive CLI tool for managing Kubernetes clusters on Google Cloud Platform.
It provides functionality for configuration management, secrets handling, pod operations, and more.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no arguments provided, show interactive mode
			if len(os.Args) == 1 || interactiveMode {
				return runInteractiveMode(version)
			}
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

	// Add flags
	cmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Run in interactive mode")

	return cmd
}

// Execute invokes the command.
func Execute(version string) error {
	// Set up graceful interrupt handling
	ui.SetupInterruptHandler()

	if err := newRootCmd(version).Execute(); err != nil {
		return fmt.Errorf("error executing root command: %w", err)
	}

	return nil
}

// runInteractiveMode runs the application in interactive mode
func runInteractiveMode(version string) error {
	// Use the new DevTools-style interface
	return ui.ShowDevToolsInterface()
}
