package pods

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gopkg.in/yaml.v2"
)

// newGetCmd creates the get subcommand
func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [POD_NAME]",
		Short: "Get pod details",
		Long:  `Get detailed information about a specific pod.`,
		Example: `  # Get pod details
  k8s-manager pods get my-pod
  
  # Get pod details in YAML format
  k8s-manager pods get my-pod -o yaml
  
  # Get pod details in JSON format
  k8s-manager pods get my-pod -o json`,
		Args: cobra.ExactArgs(1),
		RunE: runGet,
	}

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	podName := args[0]
	namespace := services.GetNamespace(cmd)

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

	// Format output
	switch output {
	case "json":
		data, err := json.MarshalIndent(pod, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal pod to JSON: %w", err)
		}
		fmt.Println(string(data))

	case "yaml":
		data, err := yaml.Marshal(pod)
		if err != nil {
			return fmt.Errorf("failed to marshal pod to YAML: %w", err)
		}
		fmt.Println(string(data))

	default:
		// Table format
		fmt.Println(components.RenderTitle("Pod Details", ""))
		fmt.Printf("\nName:      %s\n", pod.Name)
		fmt.Printf("Namespace: %s\n", pod.Namespace)
		fmt.Printf("Node:      %s\n", pod.Spec.NodeName)
		fmt.Printf("Status:    %s\n", components.RenderStatus(string(pod.Status.Phase)))
		fmt.Printf("IP:        %s\n", pod.Status.PodIP)
		fmt.Printf("Created:   %s (%s ago)\n", 
			pod.CreationTimestamp.Format(time.RFC3339),
			services.FormatAge(pod.CreationTimestamp.Time))

		// Containers
		fmt.Println("\nContainers:")
		for _, container := range pod.Spec.Containers {
			fmt.Printf("  %s:\n", container.Name)
			fmt.Printf("    Image: %s\n", container.Image)
			fmt.Printf("    Ports: ")
			for i, port := range container.Ports {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%d/%s", port.ContainerPort, port.Protocol)
			}
			fmt.Println()
		}

		// Container statuses
		fmt.Println("\nContainer Statuses:")
		for _, cs := range pod.Status.ContainerStatuses {
			status := "Unknown"
			if cs.State.Running != nil {
				status = components.RenderStatus("Running")
			} else if cs.State.Waiting != nil {
				status = fmt.Sprintf("Waiting (%s)", cs.State.Waiting.Reason)
			} else if cs.State.Terminated != nil {
				status = fmt.Sprintf("Terminated (%s)", cs.State.Terminated.Reason)
			}
			
			fmt.Printf("  %s: %s (Restarts: %d)\n", cs.Name, status, cs.RestartCount)
		}

		// Conditions
		fmt.Println("\nConditions:")
		for _, condition := range pod.Status.Conditions {
			fmt.Printf("  %s: %s\n", condition.Type, condition.Status)
		}
	}

	return nil
}