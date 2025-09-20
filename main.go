package main

import (
	"fmt"

	"github.com/karthickk/k8s-manager/internal/commands"
	"github.com/karthickk/k8s-manager/internal/commands/pods"
	"github.com/karthickk/k8s-manager/internal/ui/views"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	// Register all commands
	registerCommands()
	
	// Execute the root command
	commands.Execute()
}

// registerCommands registers all available commands
func registerCommands() {
	// Pod management
	commands.AddCommand(pods.NewPodsCmd())
	
	// TODO: Add more commands as they are implemented
	// commands.AddCommand(secrets.NewSecretsCmd())
	// commands.AddCommand(configmaps.NewConfigMapsCmd()) 
	// commands.AddCommand(namespaces.NewNamespacesCmd())

	// Interactive mode command
	commands.AddCommand(newInteractiveCmd())

	// Configuration management
	commands.AddCommand(newConfigCmd())
}

// newInteractiveCmd creates the interactive mode command
func newInteractiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "interactive",
		Aliases: []string{"i", "ui"},
		Short:   "Launch interactive UI",
		Long:    `Launch the interactive terminal UI for managing Kubernetes resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Launch the main interactive UI
			return views.ShowMainMenu()
		},
	}
}

// newConfigCmd creates the config command
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage K8s Manager configuration",
		Long:  `Manage K8s Manager configuration including contexts, namespaces, and preferences.`,
	}
	
	// Subcommands
	cmd.AddCommand(newConfigGetContextCmd())
	cmd.AddCommand(newConfigSetContextCmd())
	cmd.AddCommand(newConfigGetNamespaceCmd())
	cmd.AddCommand(newConfigSetNamespaceCmd())
	
	return cmd
}

func newConfigGetContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-context",
		Short: "Get current Kubernetes context",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Current context:", viper.GetString("context"))
			return nil
		},
	}
}

func newConfigSetContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-context CONTEXT",
		Short: "Set Kubernetes context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set("context", args[0])
			return viper.WriteConfig()
		},
	}
}

func newConfigGetNamespaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-namespace",
		Short: "Get current namespace",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Current namespace:", viper.GetString("k8s.namespace"))
			return nil
		},
	}
}

func newConfigSetNamespaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-namespace NAMESPACE",
		Short: "Set default namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set("k8s.namespace", args[0])
			return viper.WriteConfig()
		},
	}
}
