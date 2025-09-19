package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute commands on pods",
		Long:  `Execute commands on Kubernetes pods, either interactively or non-interactively.`,
	}

	cmd.AddCommand(newExecRunCmd())
	cmd.AddCommand(newExecShellCmd())

	return cmd
}

func newExecRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <pod-name> -- <command> [args...]",
		Short: "Execute a command on a pod",
		Long:  `Execute a specific command on a Kubernetes pod and return the output.`,
		Args:  cobra.MinimumNArgs(2),
		RunE:  runExecRun,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod (overrides config)")
	cmd.Flags().StringP("container", "c", "", "Container name (if pod has multiple containers)")
	cmd.Flags().BoolP("interactive", "i", false, "Keep STDIN open")
	cmd.Flags().BoolP("tty", "t", false, "Allocate a TTY")

	return cmd
}

func newExecShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell <pod-name>",
		Short: "Start an interactive shell on a pod",
		Long:  `Start an interactive shell session on a Kubernetes pod.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runExecShell,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod (overrides config)")
	cmd.Flags().StringP("container", "c", "", "Container name (if pod has multiple containers)")
	cmd.Flags().StringP("shell", "s", "/bin/bash", "Shell to use (/bin/bash, /bin/sh, etc.)")

	return cmd
}

func runExecRun(cmd *cobra.Command, args []string) error {
	podName := args[0]
	command := args[1:]

	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	container, _ := cmd.Flags().GetString("container")
	interactive, _ := cmd.Flags().GetBool("interactive")
	tty, _ := cmd.Flags().GetBool("tty")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	// Build kubectl exec command
	kubectlArgs := []string{"exec"}

	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}

	if interactive {
		kubectlArgs = append(kubectlArgs, "-i")
	}

	if tty {
		kubectlArgs = append(kubectlArgs, "-t")
	}

	kubectlArgs = append(kubectlArgs, podName)

	if container != "" {
		kubectlArgs = append(kubectlArgs, "-c", container)
	}

	kubectlArgs = append(kubectlArgs, "--")
	kubectlArgs = append(kubectlArgs, command...)

	// Execute kubectl exec command
	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr

	if interactive {
		kubectlCmd.Stdin = os.Stdin
	}

	// Handle TTY properly
	if tty {
		kubectlCmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	}

	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command on pod %s: %w", podName, err)
	}

	return nil
}

func runExecShell(cmd *cobra.Command, args []string) error {
	podName := args[0]

	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	container, _ := cmd.Flags().GetString("container")
	shell, _ := cmd.Flags().GetString("shell")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	// Check if pod exists and get container info if needed
	ctx := cmd.Context()
	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	// If container not specified and pod has multiple containers, list them
	if container == "" && len(pod.Spec.Containers) > 1 {
		fmt.Println("Pod has multiple containers:")
		for i, c := range pod.Spec.Containers {
			fmt.Printf("  %d. %s\n", i+1, c.Name)
		}
		fmt.Printf("Select container (1-%d) or press Enter for %s: ", len(pod.Spec.Containers), pod.Spec.Containers[0].Name)

		var input string
		fmt.Scanln(&input)

		if input != "" {
			var choice int
			if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(pod.Spec.Containers) {
				return fmt.Errorf("invalid container selection")
			}
			container = pod.Spec.Containers[choice-1].Name
		} else {
			container = pod.Spec.Containers[0].Name
		}
	} else if container == "" {
		container = pod.Spec.Containers[0].Name
	}

	fmt.Printf("🔗 Starting interactive shell in pod '%s', container '%s'...\n", podName, container)
	fmt.Printf("💡 Use 'exit' or Ctrl+D to close the session\n\n")

	// Use the ExecIntoPod function
	return k8s.ExecIntoPod(namespace, podName, container, shell)
}
