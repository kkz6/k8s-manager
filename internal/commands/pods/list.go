package pods

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	output string
	watch  bool
)

// newListCmd creates the list subcommand
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pods",
		Long:    `List all pods in the specified namespace(s) with their status information.`,
		Example: `  # List pods in current namespace
  k8s-manager pods list
  
  # List pods in specific namespace
  k8s-manager pods list -n production
  
  # List pods in all namespaces
  k8s-manager pods list -A
  
  # List pods with specific label
  k8s-manager pods list -l app=nginx
  
  # Watch pods (auto-refresh)
  k8s-manager pods list -w`,
		RunE: runList,
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table|wide|json|yaml)")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	// Get Kubernetes client
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Determine namespace
	namespace := services.GetNamespace(cmd)
	if allNamespaces {
		namespace = ""
	}

	// Create list options
	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}

	// Create table
	table := components.NewTable("Pods", []components.TableColumn{
		{Title: "NAMESPACE", Width: 20},
		{Title: "NAME", Width: 30},
		{Title: "READY", Width: 10},
		{Title: "STATUS", Width: 15},
		{Title: "RESTARTS", Width: 10},
		{Title: "AGE", Width: 10},
		{Title: "NODE", Width: 20},
	})

	// Function to fetch and display pods
	fetchPods := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pods, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		// Convert to table rows
		rows := []components.TableRow{}
		for _, pod := range pods.Items {
			ready := services.GetPodReadyCount(&pod)
			age := services.FormatAge(pod.CreationTimestamp.Time)
			restarts := services.GetPodRestarts(&pod)
			
			row := components.TableRow{
				pod.Namespace,
				pod.Name,
				ready,
				string(pod.Status.Phase),
				fmt.Sprintf("%d", restarts),
				age,
				pod.Spec.NodeName,
			}
			rows = append(rows, row)
		}

		table.SetRows(rows)
		
		// Clear screen and display
		fmt.Print("\033[H\033[2J")
		fmt.Println(table.View())
		
		if !watch {
			fmt.Printf("\nFound %d pods\n", len(pods.Items))
		}

		return nil
	}

	// Initial fetch
	if err := fetchPods(); err != nil {
		return err
	}

	// Watch mode
	if watch {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		fmt.Println("\nWatching pods... Press Ctrl+C to stop")
		
		for range ticker.C {
			if err := fetchPods(); err != nil {
				fmt.Fprintf(os.Stderr, "Error refreshing: %v\n", err)
			}
		}
	}

	return nil
}