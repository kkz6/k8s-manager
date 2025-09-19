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
	// Display header
	ui.DisplayHeader()

	for {
		// Show enhanced main menu with better visual design
		choice := ui.ShowEnhancedMainMenu()

		switch choice {
		case "config":
			// Show config submenu
			configChoice := ui.ShowConfigMenu()
			switch configChoice {
			case "init":
				if err := runConfigInit(nil, nil); err != nil {
					ui.ShowError(err.Error())
				} else {
					ui.ShowSuccess("Configuration initialized successfully!")
				}
			case "show":
				if err := runConfigShow(nil, nil); err != nil {
					ui.ShowError(err.Error())
				}
			case "validate":
				if err := runConfigValidate(nil, nil); err != nil {
					ui.ShowError(err.Error())
				} else {
					ui.ShowSuccess("Configuration validated successfully!")
				}
			case "set":
				ui.ShowInfo("Please use 'k8s-manager config set <key> <value>' from command line")
			}

		case "pods":
			ui.ShowInfo("Pods functionality - use 'k8s-manager pods' command")
			// TODO: Implement interactive pods listing

		case "logs":
			ui.ShowInfo("Logs functionality - use 'k8s-manager logs <pod-name>' command")

		case "exec":
			ui.ShowInfo("Exec functionality - use 'k8s-manager exec <pod-name> -- <command>' command")

		case "secrets":
			ui.ShowInfo("Secrets functionality - use 'k8s-manager secrets' command")

		case "resources":
			ui.ShowInfo("Resources functionality - coming soon!")

		case "port-forward":
			ui.ShowInfo("Port forwarding functionality - coming soon!")

		case "deployments":
			ui.ShowInfo("Deployments functionality - coming soon!")

		case "services":
			ui.ShowInfo("Services functionality - coming soon!")

		case "namespaces":
			ui.ShowInfo("Namespaces functionality - coming soon!")

		case "help":
			cmd := newRootCmd(version)
			cmd.Help()

		case "exit", "":
			fmt.Println("\n👋 Goodbye! Thank you for using K8s Manager.")
			return nil
		}

		if choice != "" && choice != "exit" {
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			// Clear screen for next iteration
			fmt.Print("\033[H\033[2J")
			ui.DisplayHeader()
		}
	}
}
