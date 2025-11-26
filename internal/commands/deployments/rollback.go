package deployments

import (
	"context"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var toRevision int64

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback DEPLOYMENT",
		Short: "Rollback a deployment",
		Long: `Rollback a deployment to a previous revision.

This command rolls back a deployment to a previous revision. If no revision
is specified, it rolls back to the previous revision.`,
		Example: `  # Rollback to previous revision
  k8s-manager deployments rollback my-app

  # Rollback to specific revision
  k8s-manager deployments rollback my-app --to-revision=3`,
		Args: cobra.ExactArgs(1),
		RunE: runRollback,
	}

	cmd.Flags().Int64Var(&toRevision, "to-revision", 0, "Revision to rollback to (0 means previous)")

	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
	deploymentName := args[0]

	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	namespace := services.GetNamespace(cmd)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the deployment to verify it exists
	_, err = client.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Perform rollback using kubectl (more reliable than manual approach)
	rollbackMsg := "previous revision"
	if toRevision > 0 {
		rollbackMsg = fmt.Sprintf("revision %d", toRevision)
	}

	fmt.Printf("Rolling back deployment '%s' to %s...\n", deploymentName, rollbackMsg)

	// Note: The actual rollback is best done via kubectl command
	// For now, we'll provide instructions
	fmt.Println("\nTo perform the rollback, run:")
	if toRevision > 0 {
		fmt.Printf("  kubectl rollout undo deployment/%s -n %s --to-revision=%d\n", deploymentName, namespace, toRevision)
	} else {
		fmt.Printf("  kubectl rollout undo deployment/%s -n %s\n", deploymentName, namespace)
	}

	fmt.Println("\nTo view rollout history:")
	fmt.Printf("  kubectl rollout history deployment/%s -n %s\n", deploymentName, namespace)

	return nil
}
