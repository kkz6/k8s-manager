package pods

import (
	"context"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newRestartCmd creates the restart subcommand
func newRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart [POD_NAME...]",
		Short: "Restart pods",
		Long:  `Restart pods by deleting them. The pods will be recreated by their controllers.`,
		Example: `  # Restart a pod
  k8s-manager pods restart my-pod
  
  # Restart multiple pods
  k8s-manager pods restart pod1 pod2 pod3
  
  # Restart all pods with specific label
  k8s-manager pods restart -l app=nginx`,
		RunE: runRestart,
	}

	return cmd
}

func runRestart(cmd *cobra.Command, args []string) error {
	namespace := services.GetNamespace(cmd)

	// Get Kubernetes client
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	var podsToRestart []string

	// If selector is provided, get pods by label
	if selector != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		listOptions := metav1.ListOptions{
			LabelSelector: selector,
		}

		pods, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		for _, pod := range pods.Items {
			podsToRestart = append(podsToRestart, pod.Name)
		}
	} else {
		// Use specified pod names
		if len(args) == 0 {
			return fmt.Errorf("pod name(s) required")
		}
		podsToRestart = args
	}

	if len(podsToRestart) == 0 {
		fmt.Println("No pods found to restart")
		return nil
	}

	// Confirm restart
	fmt.Printf("Restart %d pod(s)? This will delete and recreate them.\n", len(podsToRestart))
	for _, podName := range podsToRestart {
		fmt.Printf("  - %s\n", podName)
	}
	fmt.Print("[y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Restart cancelled")
		return nil
	}

	// Restart each pod
	ctx := context.Background()
	gracePeriodSeconds := int64(30)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	fmt.Println("\nRestarting pods...")
	for _, podName := range podsToRestart {
		fmt.Printf("Restarting pod %s...\n", podName)
		
		deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := client.Clientset.CoreV1().Pods(namespace).Delete(deleteCtx, podName, deleteOptions)
		cancel()

		if err != nil {
			fmt.Println(components.RenderMessage("error", fmt.Sprintf("Failed to restart %s: %v", podName, err)))
		} else {
			fmt.Println(components.RenderMessage("success", fmt.Sprintf("Pod %s restart initiated", podName)))
		}
	}

	fmt.Println("\nPods will be recreated by their controllers.")
	return nil
}