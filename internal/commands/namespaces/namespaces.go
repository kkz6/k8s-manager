package namespaces

import (
	"context"
	"fmt"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewNamespacesCmd creates the namespaces command
func NewNamespacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespaces",
		Aliases: []string{"ns", "namespace"},
		Short:   "Manage Kubernetes namespaces",
		Long: `Manage Kubernetes namespaces.

List namespaces and switch between them.`,
		Example: `  # List all namespaces
  k8s-manager namespaces list

  # Get current namespace
  k8s-manager namespaces current

  # Switch to a namespace
  k8s-manager namespaces use my-namespace`,
		RunE: listNamespaces,
	}

	cmd.AddCommand(newListNamespacesCmd())
	cmd.AddCommand(newCurrentNamespaceCmd())
	cmd.AddCommand(newUseNamespaceCmd())

	return cmd
}

func newListNamespacesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List namespaces",
		RunE:    listNamespaces,
	}
}

func newCurrentNamespaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show current namespace",
		RunE: func(cmd *cobra.Command, args []string) error {
			ns := services.GetCurrentNamespace()
			fmt.Printf("Current namespace: %s\n", ns)
			return nil
		},
	}
}

func newUseNamespaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use NAMESPACE",
		Short: "Switch to a namespace",
		Args:  cobra.ExactArgs(1),
		RunE:  useNamespace,
	}
}

func listNamespaces(cmd *cobra.Command, args []string) error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nsList, err := client.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	currentNs := services.GetCurrentNamespace()

	fmt.Printf("%-30s %-10s %s\n", "NAME", "STATUS", "AGE")

	for _, ns := range nsList.Items {
		marker := ""
		if ns.Name == currentNs {
			marker = " *"
		}

		age := services.FormatAge(ns.CreationTimestamp.Time)
		fmt.Printf("%-30s %-10s %s%s\n", ns.Name, ns.Status.Phase, age, marker)
	}

	fmt.Printf("\n* = current namespace\n")

	return nil
}

func useNamespace(cmd *cobra.Command, args []string) error {
	namespace := args[0]

	// Verify namespace exists
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = client.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("namespace %q not found: %w", namespace, err)
	}

	// Update config
	viper.Set("k8s.namespace", namespace)
	if err := viper.WriteConfig(); err != nil {
		// Config file might not exist, try to create it
		fmt.Printf("Note: Could not save to config file: %v\n", err)
	}

	fmt.Printf("Switched to namespace: %s\n", namespace)

	return nil
}
