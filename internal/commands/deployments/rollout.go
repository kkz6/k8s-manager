package deployments

import (
	"context"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	watchRollout bool
)

func newRolloutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout",
		Short: "Manage deployment rollouts",
		Long:  `Manage deployment rollouts including restart, status, and history.`,
	}

	cmd.AddCommand(newRolloutRestartCmd())
	cmd.AddCommand(newRolloutStatusCmd())
	cmd.AddCommand(newRolloutHistoryCmd())

	return cmd
}

func newRolloutRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart DEPLOYMENT",
		Short: "Restart a deployment rollout",
		Long: `Restart a deployment by triggering a new rollout.

This command performs a rolling restart of all pods in the deployment by
adding/updating a restart annotation. This causes Kubernetes to perform
a rolling update with the same configuration.`,
		Example: `  # Restart a deployment
  k8s-manager deployments rollout restart my-app

  # Restart and watch the rollout status
  k8s-manager deployments rollout restart my-app --watch

  # Restart in a specific namespace
  k8s-manager deployments rollout restart my-app -n production`,
		Args: cobra.ExactArgs(1),
		RunE: runRolloutRestart,
	}

	cmd.Flags().BoolVarP(&watchRollout, "watch", "w", false, "Watch the rollout status")

	return cmd
}

func newRolloutStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status DEPLOYMENT",
		Short: "Show rollout status",
		Long:  `Show the status of a deployment rollout.`,
		Example: `  # Show rollout status
  k8s-manager deployments rollout status my-app

  # Watch rollout status continuously
  k8s-manager deployments rollout status my-app --watch`,
		Args: cobra.ExactArgs(1),
		RunE: runRolloutStatus,
	}
}

func newRolloutHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history DEPLOYMENT",
		Short: "Show rollout history",
		Long:  `Show the rollout history of a deployment.`,
		Example: `  # Show rollout history
  k8s-manager deployments rollout history my-app`,
		Args: cobra.ExactArgs(1),
		RunE: runRolloutHistory,
	}
}

func runRolloutRestart(cmd *cobra.Command, args []string) error {
	deploymentName := args[0]

	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	namespace := services.GetNamespace(cmd)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the deployment
	deployment, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Add/update restart annotation
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Update the deployment
	fmt.Printf("🔄 Restarting deployment '%s' in namespace '%s'...\n", deploymentName, namespace)
	_, err = client.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %w", err)
	}

	fmt.Printf("✓ Deployment '%s' restart triggered successfully\n", deploymentName)

	if watchRollout {
		fmt.Println("\n📊 Watching rollout status (Ctrl+C to stop)...")
		return watchDeploymentRollout(client, namespace, deploymentName)
	}

	fmt.Println("\nUse 'kubectl rollout status' to monitor progress:")
	fmt.Printf("  kubectl rollout status deployment/%s -n %s\n", deploymentName, namespace)

	return nil
}

func runRolloutStatus(cmd *cobra.Command, args []string) error {
	deploymentName := args[0]

	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	namespace := services.GetNamespace(cmd)

	if watchRollout {
		fmt.Printf("📊 Watching rollout status for '%s' (Ctrl+C to stop)...\n\n", deploymentName)
		return watchDeploymentRollout(client, namespace, deploymentName)
	}

	return showRolloutStatus(client, namespace, deploymentName)
}

func runRolloutHistory(cmd *cobra.Command, args []string) error {
	deploymentName := args[0]
	namespace := services.GetNamespace(cmd)

	fmt.Printf("📜 Rollout history for deployment '%s':\n\n", deploymentName)
	fmt.Println("To view rollout history, run:")
	fmt.Printf("  kubectl rollout history deployment/%s -n %s\n", deploymentName, namespace)

	return nil
}

func showRolloutStatus(client *services.K8sClient, namespace, deploymentName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployment, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	fmt.Printf("📊 Rollout Status: %s/%s\n\n", namespace, deploymentName)

	desired := int32(0)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	fmt.Printf("  Desired:     %d replicas\n", desired)
	fmt.Printf("  Current:     %d replicas\n", deployment.Status.Replicas)
	fmt.Printf("  Updated:     %d replicas\n", deployment.Status.UpdatedReplicas)
	fmt.Printf("  Ready:       %d replicas\n", deployment.Status.ReadyReplicas)
	fmt.Printf("  Available:   %d replicas\n", deployment.Status.AvailableReplicas)
	fmt.Printf("  Unavailable: %d replicas\n", deployment.Status.UnavailableReplicas)

	fmt.Println("\n  Conditions:")
	for _, cond := range deployment.Status.Conditions {
		status := "✓"
		if cond.Status != "True" {
			status = "✗"
		}
		fmt.Printf("    %s %s: %s - %s\n", status, cond.Type, cond.Status, cond.Message)
	}

	// Check if rollout is complete
	if deployment.Status.UpdatedReplicas == desired &&
		deployment.Status.ReadyReplicas == desired &&
		deployment.Status.AvailableReplicas == desired {
		fmt.Println("\n✓ Rollout complete!")
	} else {
		fmt.Println("\n⏳ Rollout in progress...")
	}

	return nil
}

func watchDeploymentRollout(client *services.K8sClient, namespace, deploymentName string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStatus := ""

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			deployment, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			cancel()

			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			desired := int32(0)
			if deployment.Spec.Replicas != nil {
				desired = *deployment.Spec.Replicas
			}

			status := fmt.Sprintf("Replicas: %d/%d updated, %d/%d ready, %d available",
				deployment.Status.UpdatedReplicas, desired,
				deployment.Status.ReadyReplicas, desired,
				deployment.Status.AvailableReplicas)

			// Only print if status changed
			if status != lastStatus {
				fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), status)
				lastStatus = status
			}

			// Check if rollout is complete
			if deployment.Status.UpdatedReplicas == desired &&
				deployment.Status.ReadyReplicas == desired &&
				deployment.Status.AvailableReplicas == desired {
				fmt.Println("\n✓ Rollout complete!")
				return nil
			}
		}
	}
}
