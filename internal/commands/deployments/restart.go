package deployments

import (
	"context"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart DEPLOYMENT",
		Short: "Restart a deployment",
		Long: `Restart a deployment by triggering a rollout.

This command performs a rolling restart of all pods in the deployment by
adding/updating a restart annotation. This causes Kubernetes to perform
a rolling update with the same configuration.`,
		Example: `  # Restart a deployment
  k8s-manager deployments restart my-app

  # Restart a deployment in a specific namespace
  k8s-manager deployments restart my-app -n production`,
		Args: cobra.ExactArgs(1),
		RunE: runRestart,
	}
}

func runRestart(cmd *cobra.Command, args []string) error {
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
	fmt.Printf("Restarting deployment '%s' in namespace '%s'...\n", deploymentName, namespace)
	_, err = client.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %w", err)
	}

	fmt.Printf("✓ Deployment '%s' restart triggered successfully\n", deploymentName)
	fmt.Println("Rolling restart has been initiated. Use 'kubectl rollout status' to monitor progress.")

	return nil
}
