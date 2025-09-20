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

var (
	force       bool
	gracePeriod int
)

// newDeleteCmd creates the delete subcommand
func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [POD_NAME...]",
		Short: "Delete pods",
		Long:  `Delete one or more pods.`,
		Example: `  # Delete a pod
  k8s-manager pods delete my-pod
  
  # Delete multiple pods
  k8s-manager pods delete pod1 pod2 pod3
  
  # Force delete a pod immediately
  k8s-manager pods delete my-pod --force --grace-period=0
  
  # Delete pods with specific label
  k8s-manager pods delete -l app=nginx`,
		RunE: runDelete,
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force deletion")
	cmd.Flags().IntVar(&gracePeriod, "grace-period", 30, "Grace period in seconds")

	return cmd
}

func runDelete(cmd *cobra.Command, args []string) error {
	namespace := services.GetNamespace(cmd)

	// Get Kubernetes client
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// If selector is provided, delete by label
	if selector != "" {
		return deleteBySelector(client, namespace)
	}

	// Otherwise, delete specified pods
	if len(args) == 0 {
		return fmt.Errorf("pod name(s) required")
	}

	// Confirm deletion
	if !force {
		fmt.Printf("Delete %d pod(s)? [y/N]: ", len(args))
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete each pod
	ctx := context.Background()
	gracePeriodSeconds := int64(gracePeriod)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	for _, podName := range args {
		fmt.Printf("Deleting pod %s...\n", podName)
		
		deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := client.Clientset.CoreV1().Pods(namespace).Delete(deleteCtx, podName, deleteOptions)
		cancel()

		if err != nil {
			fmt.Println(components.RenderMessage("error", fmt.Sprintf("Failed to delete %s: %v", podName, err)))
		} else {
			fmt.Println(components.RenderMessage("success", fmt.Sprintf("Pod %s deleted", podName)))
		}
	}

	return nil
}

func deleteBySelector(client *services.K8sClient, namespace string) error {
	// List pods with selector
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	pods, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		fmt.Println("No pods found with selector:", selector)
		return nil
	}

	// Confirm deletion
	fmt.Printf("Delete %d pod(s) with selector '%s'?\n", len(pods.Items), selector)
	for _, pod := range pods.Items {
		fmt.Printf("  - %s\n", pod.Name)
	}
	
	if !force {
		fmt.Print("[y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete pods
	gracePeriodSeconds := int64(gracePeriod)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	for _, pod := range pods.Items {
		fmt.Printf("Deleting pod %s...\n", pod.Name)
		
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := client.Clientset.CoreV1().Pods(namespace).Delete(deleteCtx, pod.Name, deleteOptions)
		cancel()

		if err != nil {
			fmt.Println(components.RenderMessage("error", fmt.Sprintf("Failed to delete %s: %v", pod.Name, err)))
		} else {
			fmt.Println(components.RenderMessage("success", fmt.Sprintf("Pod %s deleted", pod.Name)))
		}
	}

	return nil
}