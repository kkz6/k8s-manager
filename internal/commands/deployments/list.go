package deployments

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List deployments",
		Long:    `List all deployments in the current or specified namespace.`,
		Example: `  # List deployments in current namespace
  k8s-manager deployments list

  # List deployments in all namespaces
  k8s-manager deployments list -A

  # List deployments with label selector
  k8s-manager deployments list -l app=myapp`,
		RunE: runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	namespace := services.GetNamespace(cmd)
	if allNamespaces {
		namespace = ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	deployments, err := client.Clientset.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments.Items) == 0 {
		fmt.Println("No deployments found")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print header
	if allNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")
	}

	// Print deployments
	for _, deploy := range deployments.Items {
		ready := fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas)
		upToDate := fmt.Sprintf("%d", deploy.Status.UpdatedReplicas)
		available := fmt.Sprintf("%d", deploy.Status.AvailableReplicas)
		age := services.FormatAge(deploy.CreationTimestamp.Time)

		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				deploy.Namespace, deploy.Name, ready, upToDate, available, age)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				deploy.Name, ready, upToDate, available, age)
		}
	}

	w.Flush()
	return nil
}
