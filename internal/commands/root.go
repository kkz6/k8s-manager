package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/karthickk/k8s-manager/internal/ui/views"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	namespace string
	context   string
	debug     bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "k8s-manager",
	Short: "A powerful Kubernetes cluster management tool",
	Long: `K8s Manager is a comprehensive CLI tool for managing Kubernetes clusters.
	
It provides an intuitive interface for common Kubernetes operations including:
- Pod management (view, exec, logs, restart)
- Secret and ConfigMap management
- Environment variable management
- Resource monitoring
- And much more!`,
	Version: "1.0.0",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, launch interactive mode
		return runInteractiveMode()
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, components.RenderMessage("error", err.Error()))
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.k8s-manager/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	rootCmd.PersistentFlags().StringVar(&context, "context", "", "Kubernetes context")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode")

	// Bind flags to viper
	viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	viper.BindPFlag("context", rootCmd.PersistentFlags().Lookup("context"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Set up command aliases
	rootCmd.SetUsageTemplate(customUsageTemplate())
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, components.RenderMessage("error", err.Error()))
			os.Exit(1)
		}

		// Search config in home directory and current directory
		viper.AddConfigPath(filepath.Join(home, ".k8s-manager"))
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")  // Current directory
		viper.SetConfigName("k8s-manager")
		viper.SetConfigType("yaml")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("K8S_MANAGER")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if debug {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// customUsageTemplate returns a custom usage template
func customUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

// AddCommand adds a command to the root command
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// runInteractiveMode launches the interactive UI
func runInteractiveMode() error {
	return views.ShowMainMenu()
}