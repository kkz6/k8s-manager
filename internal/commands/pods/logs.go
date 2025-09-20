package pods

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
)

var (
	follow     bool
	tail       int
	since      string
	timestamps bool
	previous   bool
)

// newLogsCmd creates the logs subcommand
func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [POD_NAME]",
		Short: "View pod logs",
		Long:  `View logs from a pod. If the pod has multiple containers, you must specify the container name.`,
		Example: `  # View last 100 lines of pod logs
  k8s-manager pods logs my-pod
  
  # Follow pod logs in real-time
  k8s-manager pods logs my-pod -f
  
  # View logs from specific container
  k8s-manager pods logs my-pod -c my-container
  
  # View last 50 lines with timestamps
  k8s-manager pods logs my-pod --tail=50 --timestamps
  
  # View logs from previous container instance
  k8s-manager pods logs my-pod --previous`,
		Args: cobra.ExactArgs(1),
		RunE: runLogs,
	}

	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name (if multiple containers)")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVar(&tail, "tail", 100, "Number of lines to show from the end of the logs")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since timestamp (e.g. 10s, 2m, 3h)")
	cmd.Flags().BoolVarP(&timestamps, "timestamps", "t", false, "Include timestamps in log output")
	cmd.Flags().BoolVarP(&previous, "previous", "p", false, "Show logs from previous container instance")

	return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	podName := args[0]
	namespace := services.GetNamespace(cmd)

	// Build kubectl command
	kubectlArgs := []string{"logs", podName, "-n", namespace}

	if container != "" {
		kubectlArgs = append(kubectlArgs, "-c", container)
	}

	if follow {
		kubectlArgs = append(kubectlArgs, "-f")
	}

	if tail > 0 {
		kubectlArgs = append(kubectlArgs, fmt.Sprintf("--tail=%d", tail))
	}

	if since != "" {
		kubectlArgs = append(kubectlArgs, fmt.Sprintf("--since=%s", since))
	}

	if timestamps {
		kubectlArgs = append(kubectlArgs, "--timestamps")
	}

	if previous {
		kubectlArgs = append(kubectlArgs, "--previous")
	}

	// Create and run command
	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr

	if follow {
		// For follow mode, allow interruption
		kubectlCmd.Stdin = os.Stdin
		fmt.Println("Following logs... Press Ctrl+C to stop")
	}

	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("failed to get pod logs: %w", err)
	}

	return nil
}