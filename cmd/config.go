package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/karthickk/k8s-manager/pkg/config"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage K8s Manager configuration",
		Long:  `Configure GCP project, Kubernetes cluster, and other settings for K8s Manager.`,
	}

	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigValidateCmd())

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration interactively",
		Long:  `Initialize K8s Manager configuration by prompting for required values.`,
		RunE:  runConfigInit,
	}

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  `Display the current K8s Manager configuration.`,
		RunE:  runConfigShow,
	}

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  `Set a specific configuration value. Use dot notation for nested keys (e.g., gcp.project_id).`,
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}

	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		Long:  `Validate the current configuration and test connectivity to GCP and Kubernetes.`,
		RunE:  runConfigValidate,
	}

	return cmd
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Initializing K8s Manager configuration...")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get GCP Project ID
	if cfg.GCP.ProjectID == "" {
		var projectID string
		prompt := &survey.Input{
			Message: "GCP Project ID:",
		}
		if err := survey.AskOne(prompt, &projectID, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
		cfg.GCP.ProjectID = projectID
	}

	// Get available clusters
	fmt.Println("📋 Fetching available clusters...")
	clusters, err := k8s.ListClusters(cfg.GCP.ProjectID, cfg.GCP.Zone, cfg.GCP.Region)
	if err != nil {
		fmt.Printf("⚠️  Could not fetch clusters: %v\n", err)
		fmt.Println("You can set the cluster name manually.")

		var clusterName string
		prompt := &survey.Input{
			Message: "Cluster name:",
		}
		if err := survey.AskOne(prompt, &clusterName, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
		cfg.K8s.ClusterName = clusterName
	} else if len(clusters) > 0 {
		var clusterName string
		prompt := &survey.Select{
			Message: "Select a cluster:",
			Options: clusters,
		}
		if err := survey.AskOne(prompt, &clusterName); err != nil {
			return err
		}
		cfg.K8s.ClusterName = clusterName
	} else {
		fmt.Println("No clusters found. Please enter cluster name manually.")
		var clusterName string
		prompt := &survey.Input{
			Message: "Cluster name:",
		}
		if err := survey.AskOne(prompt, &clusterName, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
		cfg.K8s.ClusterName = clusterName
	}

	// Get zone/region
	if cfg.GCP.Zone == "" && cfg.GCP.Region == "" {
		var location string
		prompt := &survey.Input{
			Message: "Zone or Region (e.g., us-central1-a or us-central1):",
			Default: "us-central1-a",
		}
		if err := survey.AskOne(prompt, &location); err != nil {
			return err
		}

		if strings.Contains(location, "-") && len(strings.Split(location, "-")) == 3 {
			cfg.GCP.Zone = location
		} else {
			cfg.GCP.Region = location
		}
	}

	// Get namespace
	var namespace string
	prompt := &survey.Input{
		Message: "Default namespace:",
		Default: "default",
	}
	if err := survey.AskOne(prompt, &namespace); err != nil {
		return err
	}
	cfg.K8s.Namespace = namespace

	// Save configuration
	if err := config.Update("gcp.project_id", cfg.GCP.ProjectID); err != nil {
		return fmt.Errorf("failed to save project ID: %w", err)
	}
	if err := config.Update("k8s.cluster_name", cfg.K8s.ClusterName); err != nil {
		return fmt.Errorf("failed to save cluster name: %w", err)
	}
	if err := config.Update("k8s.namespace", cfg.K8s.Namespace); err != nil {
		return fmt.Errorf("failed to save namespace: %w", err)
	}
	if cfg.GCP.Zone != "" {
		if err := config.Update("gcp.zone", cfg.GCP.Zone); err != nil {
			return fmt.Errorf("failed to save zone: %w", err)
		}
	}
	if cfg.GCP.Region != "" {
		if err := config.Update("gcp.region", cfg.GCP.Region); err != nil {
			return fmt.Errorf("failed to save region: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("✅ Configuration initialized successfully!")
	fmt.Println("Run 'k8s-manager config validate' to test the configuration.")

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("📋 Current K8s Manager Configuration:")
	fmt.Println()

	fmt.Println("🌐 GCP Settings:")
	fmt.Printf("  Project ID: %s\n", cfg.GCP.ProjectID)
	fmt.Printf("  Zone:       %s\n", cfg.GCP.Zone)
	fmt.Printf("  Region:     %s\n", cfg.GCP.Region)
	fmt.Println()

	fmt.Println("☸️  Kubernetes Settings:")
	fmt.Printf("  Cluster:    %s\n", cfg.K8s.ClusterName)
	fmt.Printf("  Namespace:  %s\n", cfg.K8s.Namespace)
	fmt.Printf("  Config:     %s\n", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	fmt.Println()

	fmt.Println("🔐 SSH Settings:")
	fmt.Printf("  Username:   %s\n", cfg.SSH.Username)
	fmt.Printf("  Port:       %d\n", cfg.SSH.Port)
	fmt.Printf("  Key Path:   %s\n", cfg.SSH.KeyPath)
	fmt.Println()

	fmt.Printf("📊 Log Level:   %s\n", cfg.LogLevel)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if err := config.Update(key, value); err != nil {
		return fmt.Errorf("failed to set %s: %w", key, err)
	}

	fmt.Printf("✅ Set %s = %s\n", key, value)
	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Validating K8s Manager configuration...")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	fmt.Println("✅ Configuration is valid")

	// Check gcloud CLI
	if _, err := exec.LookPath("gcloud"); err != nil {
		return fmt.Errorf("❌ gcloud CLI not found. Please install Google Cloud SDK")
	}
	fmt.Println("✅ gcloud CLI is available")

	// Check gcloud authentication
	cmd_auth := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := cmd_auth.Output()
	if err != nil {
		return fmt.Errorf("❌ failed to check gcloud auth: %w", err)
	}
	if len(output) == 0 {
		return fmt.Errorf("❌ no active gcloud authentication. Please run 'gcloud auth login'")
	}
	fmt.Println("✅ gcloud authentication is active")

	// Check kubectl
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("❌ kubectl not found. Please install kubectl")
	}
	fmt.Println("✅ kubectl is available")

	// Test Kubernetes connection
	fmt.Println("🔗 Testing Kubernetes connection...")
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("❌ failed to create Kubernetes client: %w", err)
	}

	ctx := cmd.Context()
	if err := client.ValidateConnection(ctx); err != nil {
		return fmt.Errorf("❌ failed to connect to Kubernetes cluster: %w", err)
	}
	fmt.Println("✅ Successfully connected to Kubernetes cluster")

	fmt.Println()
	fmt.Println("🎉 All validations passed! K8s Manager is ready to use.")

	return nil
}

