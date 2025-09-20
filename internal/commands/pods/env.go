package pods

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/karthickk/k8s-manager/internal/ui/views"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newEnvCmd creates the env subcommand
func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env [POD_NAME]",
		Short: "Manage pod environment variables",
		Long:  `View and manage environment variables in a pod.`,
		Example: `  # View environment variables
  k8s-manager pods env my-pod
  
  # Manage environment variables interactively
  k8s-manager pods env my-pod --interactive`,
		Args: cobra.ExactArgs(1),
		RunE: runEnv,
	}

	cmd.Flags().BoolP("interactive", "i", false, "Interactive mode for managing environment variables")

	return cmd
}

func runEnv(cmd *cobra.Command, args []string) error {
	podName := args[0]
	namespace := services.GetNamespace(cmd)
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Get Kubernetes client
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Get pod
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	if interactive {
		// Launch interactive environment variable management
		model := views.NewPodEnvModel(pod, client)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	}

	// Just display current environment variables
	fmt.Println(components.RenderTitle("Environment Variables", fmt.Sprintf("Pod: %s", podName)))
	
	for _, container := range pod.Spec.Containers {
		fmt.Printf("\nContainer: %s\n", container.Name)
		fmt.Println(strings.Repeat("-", 50))
		
		if len(container.Env) == 0 && len(container.EnvFrom) == 0 {
			fmt.Println("  No environment variables defined")
			continue
		}

		// Direct environment variables
		if len(container.Env) > 0 {
			fmt.Println("  Direct Variables:")
			for _, env := range container.Env {
				if env.Value != "" {
					fmt.Printf("    %s = %s\n", env.Name, env.Value)
				} else if env.ValueFrom != nil {
					source := getEnvSource(env.ValueFrom)
					fmt.Printf("    %s = %s\n", env.Name, source)
				}
			}
		}

		// Environment from sources
		if len(container.EnvFrom) > 0 {
			fmt.Println("\n  From Sources:")
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					fmt.Printf("    ConfigMap: %s", envFrom.ConfigMapRef.Name)
					if envFrom.Prefix != "" {
						fmt.Printf(" (prefix: %s)", envFrom.Prefix)
					}
					fmt.Println()
				}
				if envFrom.SecretRef != nil {
					fmt.Printf("    Secret: %s", envFrom.SecretRef.Name)
					if envFrom.Prefix != "" {
						fmt.Printf(" (prefix: %s)", envFrom.Prefix)
					}
					fmt.Println()
				}
			}
		}
	}

	return nil
}

func getEnvSource(valueFrom *corev1.EnvVarSource) string {
	if valueFrom.FieldRef != nil {
		return fmt.Sprintf("(field: %s)", valueFrom.FieldRef.FieldPath)
	}
	if valueFrom.ResourceFieldRef != nil {
		return fmt.Sprintf("(resource: %s)", valueFrom.ResourceFieldRef.Resource)
	}
	if valueFrom.ConfigMapKeyRef != nil {
		return fmt.Sprintf("(configmap: %s/%s)", valueFrom.ConfigMapKeyRef.Name, valueFrom.ConfigMapKeyRef.Key)
	}
	if valueFrom.SecretKeyRef != nil {
		return fmt.Sprintf("(secret: %s/%s)", valueFrom.SecretKeyRef.Name, valueFrom.SecretKeyRef.Key)
	}
	return "(unknown source)"
}