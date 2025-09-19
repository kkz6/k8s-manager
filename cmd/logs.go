package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <pod-name>",
		Short: "View pod logs",
		Long:  `View and follow logs from Kubernetes pods.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runLogs,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod (overrides config)")
	cmd.Flags().StringP("container", "c", "", "Container name (if pod has multiple containers)")
	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().BoolP("previous", "p", false, "Show logs from previous container instance")
	cmd.Flags().StringP("since", "", "", "Show logs since duration (e.g., 5s, 2m, 3h)")
	cmd.Flags().StringP("since-time", "", "", "Show logs since timestamp (RFC3339)")
	cmd.Flags().Int64P("tail", "", -1, "Number of lines to show from the end of the logs")
	cmd.Flags().BoolP("timestamps", "", false, "Include timestamps in log output")

	return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	podName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	container, _ := cmd.Flags().GetString("container")
	follow, _ := cmd.Flags().GetBool("follow")
	previous, _ := cmd.Flags().GetBool("previous")
	since, _ := cmd.Flags().GetString("since")
	sinceTime, _ := cmd.Flags().GetString("since-time")
	tail, _ := cmd.Flags().GetInt64("tail")
	timestamps, _ := cmd.Flags().GetBool("timestamps")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	// Build kubectl logs command
	kubectlArgs := []string{"logs"}

	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}

	kubectlArgs = append(kubectlArgs, podName)

	if container != "" {
		kubectlArgs = append(kubectlArgs, "-c", container)
	}

	if follow {
		kubectlArgs = append(kubectlArgs, "-f")
	}

	if previous {
		kubectlArgs = append(kubectlArgs, "-p")
	}

	if since != "" {
		kubectlArgs = append(kubectlArgs, "--since", since)
	}

	if sinceTime != "" {
		kubectlArgs = append(kubectlArgs, "--since-time", sinceTime)
	}

	if tail >= 0 {
		kubectlArgs = append(kubectlArgs, "--tail", fmt.Sprintf("%d", tail))
	}

	if timestamps {
		kubectlArgs = append(kubectlArgs, "--timestamps")
	}

	// Execute kubectl logs command
	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	kubectlCmd.Stdin = os.Stdin

	// Handle interrupt signals for graceful shutdown when following logs
	if follow {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			if kubectlCmd.Process != nil {
				kubectlCmd.Process.Kill()
			}
		}()
	}

	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("failed to get logs for pod %s: %w", podName, err)
	}

	return nil
}
