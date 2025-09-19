package k8s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/karthickk/k8s-manager/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes client with additional functionality
type Client struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
	cfg       *config.Config
}

// NewClient creates a new Kubernetes client using gcloud CLI for authentication
func NewClient() (*Client, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	// Ensure gcloud is authenticated and configured
	if err := ensureGcloudAuth(cfg); err != nil {
		return nil, fmt.Errorf("gcloud authentication failed: %w", err)
	}

	// Get cluster credentials using gcloud
	if err := getClusterCredentials(cfg); err != nil {
		return nil, fmt.Errorf("failed to get cluster credentials: %w", err)
	}

	// Create Kubernetes client config
	kubeConfig, err := buildKubeConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build kube config: %w", err)
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		Clientset: clientset,
		Config:    kubeConfig,
		cfg:       cfg,
	}, nil
}

// ensureGcloudAuth ensures gcloud is authenticated and project is set
func ensureGcloudAuth(cfg *config.Config) error {
	// Check if gcloud is installed
	if _, err := exec.LookPath("gcloud"); err != nil {
		return fmt.Errorf("gcloud CLI not found. Please install Google Cloud SDK")
	}

	// Set the project
	if cfg.GCP.ProjectID != "" {
		cmd := exec.Command("gcloud", "config", "set", "project", cfg.GCP.ProjectID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set gcloud project: %w", err)
		}
	}

	// Check authentication status
	cmd := exec.Command("gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check gcloud auth status: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("no active gcloud authentication found. Please run 'gcloud auth login'")
	}

	return nil
}

// getClusterCredentials gets GKE cluster credentials using gcloud
func getClusterCredentials(cfg *config.Config) error {
	if cfg.K8s.ClusterName == "" {
		return fmt.Errorf("cluster name not configured")
	}

	args := []string{
		"container", "clusters", "get-credentials",
		cfg.K8s.ClusterName,
	}

	if cfg.GCP.Zone != "" {
		args = append(args, "--zone", cfg.GCP.Zone)
	} else if cfg.GCP.Region != "" {
		args = append(args, "--region", cfg.GCP.Region)
	}

	if cfg.GCP.ProjectID != "" {
		args = append(args, "--project", cfg.GCP.ProjectID)
	}

	cmd := exec.Command("gcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get cluster credentials: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// buildKubeConfig builds the Kubernetes client configuration
func buildKubeConfig(cfg *config.Config) (*rest.Config, error) {
	kubeConfigPath := cfg.K8s.ConfigPath
	if kubeConfigPath == "" {
		kubeConfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	// Use the kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// GetNamespace returns the configured namespace or default
func (c *Client) GetNamespace() string {
	if c.cfg.K8s.Namespace != "" {
		return c.cfg.K8s.Namespace
	}
	return "default"
}

// ValidateConnection validates the connection to the Kubernetes cluster
func (c *Client) ValidateConnection(ctx context.Context) error {
	_, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}
	return nil
}

// SwitchNamespace switches the current namespace context
func (c *Client) SwitchNamespace(namespace string) error {
	c.cfg.K8s.Namespace = namespace
	return config.Update("k8s.namespace", namespace)
}

// GetCurrentContext returns the current kubectl context
func GetCurrentContext() (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current context: %w", err)
	}
	return string(output), nil
}

// ListClusters lists available GKE clusters
func ListClusters(projectID, zone, region string) ([]string, error) {
	args := []string{"container", "clusters", "list", "--format=value(name)"}

	if projectID != "" {
		args = append(args, "--project", projectID)
	}

	if zone != "" {
		args = append(args, "--zone", zone)
	} else if region != "" {
		args = append(args, "--region", region)
	}

	cmd := exec.Command("gcloud", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Parse output into slice of cluster names
	clusters := []string{}
	if len(output) > 0 {
		// Split by newlines and filter empty strings
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				clusters = append(clusters, strings.TrimSpace(line))
			}
		}
	}

	return clusters, nil
}

// ExecIntoPod executes an interactive shell in a pod
func ExecIntoPod(namespace, podName, containerName, shell string) error {
	args := []string{
		"exec", "-it",
		"-n", namespace,
		podName,
	}

	if containerName != "" {
		args = append(args, "-c", containerName)
	}

	args = append(args, "--", shell)

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Handle interrupt signals properly
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Run()
}
