package pods

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
)

// newDescribeCmd creates the describe subcommand
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [POD_NAME]",
		Short: "Describe a pod",
		Long:  `Show detailed information about a pod including events.`,
		Example: `  # Describe a pod
  k8s-manager pods describe my-pod
  
  # Describe a pod in specific namespace
  k8s-manager pods describe my-pod -n production`,
		Args: cobra.ExactArgs(1),
		RunE: runDescribe,
	}

	return cmd
}

func runDescribe(cmd *cobra.Command, args []string) error {
	podName := args[0]
	namespace := services.GetNamespace(cmd)

	// Use kubectl describe for detailed output
	kubectlCmd := exec.Command("kubectl", "describe", "pod", podName, "-n", namespace)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr

	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("failed to describe pod: %w", err)
	}

	return nil
}