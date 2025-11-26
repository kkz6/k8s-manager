package deployments

import (
	"context"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var replicas int32

func newScaleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scale DEPLOYMENT",
		Short: "Scale a deployment",
		Long: `Scale a deployment to a specified number of replicas.

This command updates the replica count of a deployment, causing Kubernetes
to scale the number of pods up or down to match the desired count.`,
		Example: `  # Scale deployment to 3 replicas
  k8s-manager deployments scale my-app --replicas=3

  # Scale deployment to 0 (stop all pods)
  k8s-manager deployments scale my-app --replicas=0`,
		Args: cobra.ExactArgs(1),
		RunE: runScale,
	}

	cmd.Flags().Int32Var(&replicas, "replicas", 1, "Number of replicas")
	cmd.MarkFlagRequired("replicas")

	return cmd
}

func runScale(cmd *cobra.Command, args []string) error {
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

	currentReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		currentReplicas = *deployment.Spec.Replicas
	}

	// Update replicas
	deployment.Spec.Replicas = &replicas

	// Update the deployment
	fmt.Printf("Scaling deployment '%s' from %d to %d replicas...\n", deploymentName, currentReplicas, replicas)
	_, err = client.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment: %w", err)
	}

	fmt.Printf("✓ Deployment '%s' scaled successfully\n", deploymentName)

	return nil
}
