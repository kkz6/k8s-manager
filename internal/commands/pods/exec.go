package pods

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
)

var (
	container string
	shell     string
)

// newExecCmd creates the exec subcommand
func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [POD_NAME] -- [COMMAND]",
		Short: "Execute a command in a pod",
		Long:  `Execute a command in a container of a pod. If no command is specified, an interactive shell is opened.`,
		Example: `  # Open a shell in a pod
  k8s-manager pods exec my-pod
  
  # Execute a command in a pod
  k8s-manager pods exec my-pod -- ls -la
  
  # Open shell in specific container
  k8s-manager pods exec my-pod -c my-container
  
  # Use specific shell
  k8s-manager pods exec my-pod --shell=/bin/bash`,
		Args: cobra.MinimumNArgs(1),
		RunE: runExec,
	}

	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name (if multiple containers)")
	cmd.Flags().StringVar(&shell, "shell", "/bin/sh", "Shell to use for interactive session")

	return cmd
}

func runExec(cmd *cobra.Command, args []string) error {
	podName := args[0]
	namespace := services.GetNamespace(cmd)

	// Build kubectl command
	kubectlArgs := []string{"exec", "-it", podName, "-n", namespace}
	
	if container != "" {
		kubectlArgs = append(kubectlArgs, "-c", container)
	}

	// Add command separator
	kubectlArgs = append(kubectlArgs, "--")

	// Add command or shell
	if len(args) > 1 {
		// Execute specific command
		kubectlArgs = append(kubectlArgs, args[1:]...)
	} else {
		// Open interactive shell
		kubectlArgs = append(kubectlArgs, shell)
	}

	// Create and run command
	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdin = os.Stdin
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr

	if err := kubectlCmd.Run(); err != nil {
		// Try with /bin/bash if /bin/sh failed
		if shell == "/bin/sh" && len(args) == 1 {
			kubectlArgs[len(kubectlArgs)-1] = "/bin/bash"
			kubectlCmd = exec.Command("kubectl", kubectlArgs...)
			kubectlCmd.Stdin = os.Stdin
			kubectlCmd.Stdout = os.Stdout
			kubectlCmd.Stderr = os.Stderr
			
			if err2 := kubectlCmd.Run(); err2 != nil {
				return fmt.Errorf("failed to exec into pod: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to exec into pod: %w", err)
	}

	return nil
}