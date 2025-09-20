package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	GCP      GCPConfig `mapstructure:"gcp"`
	K8s      K8sConfig `mapstructure:"k8s"`
	SSH      SSHConfig `mapstructure:"ssh"`
	LogLevel string    `mapstructure:"log_level"`
}

// GCPConfig holds GCP-specific configuration
type GCPConfig struct {
	ProjectID string `mapstructure:"project_id"`
	Zone      string `mapstructure:"zone"`
	Region    string `mapstructure:"region"`
	// Using gcloud CLI for authentication, no need for credentials file
}

// K8sConfig holds Kubernetes-specific configuration
type K8sConfig struct {
	ClusterName string `mapstructure:"cluster_name"`
	Namespace   string `mapstructure:"namespace"`
	Context     string `mapstructure:"context"`
}

// SSHConfig holds SSH-specific configuration
type SSHConfig struct {
	KeyPath  string `mapstructure:"key_path"`
	Username string `mapstructure:"username"`
	Port     int    `mapstructure:"port"`
}

var cfg *Config

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	// Reset viper to ensure fresh load
	viper.Reset()

	viper.SetConfigName("k8s-manager")
	viper.SetConfigType("yaml")

	// Add config paths
	viper.AddConfigPath(".")
	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		viper.AddConfigPath(filepath.Join(homeDir, ".config", "k8s-manager"))
	}
	viper.AddConfigPath("$HOME/.config/k8s-manager")
	viper.AddConfigPath("/etc/k8s-manager")

	// Set default values
	setDefaults()

	// Enable environment variable support
	viper.SetEnvPrefix("K8S_MANAGER")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// For config init command, we need to allow initialization even without existing config
			// Just log the error and continue with defaults
			if !viper.IsSet("gcp.project_id") && !viper.IsSet("k8s.cluster_name") {
				// This is likely a fresh installation, continue with defaults
			} else {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
		// Config file not found is ok, we'll create one during init
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return cfg, nil
}

// Get returns the current configuration
func Get() *Config {
	if cfg == nil {
		cfg, _ = Load()
	}
	return cfg
}

// Save saves the current configuration to file
func Save() error {
	if cfg == nil {
		return fmt.Errorf("no configuration to save")
	}

	configDir := filepath.Join(os.Getenv("HOME"), ".config", "k8s-manager")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "k8s-manager.yaml")
	return viper.WriteConfigAs(configFile)
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("gcp.zone", "us-central1-a")
	viper.SetDefault("gcp.region", "us-central1")
	viper.SetDefault("k8s.namespace", "default")
	viper.SetDefault("ssh.port", 22)
	viper.SetDefault("ssh.username", "root")
	viper.SetDefault("log_level", "info")
}

// Update updates a configuration value
func Update(key string, value interface{}) error {
	viper.Set(key, value)

	// Reload config
	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("error unmarshaling updated config: %w", err)
	}

	return Save()
}

// Validate validates the current configuration
func (c *Config) Validate() error {
	if c.GCP.ProjectID == "" {
		return fmt.Errorf("GCP project ID is required")
	}

	if c.K8s.ClusterName == "" {
		return fmt.Errorf("Kubernetes cluster name is required")
	}

	return nil
}
