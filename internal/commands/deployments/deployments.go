package deployments

import (
	"github.com/karthickk/k8s-manager/internal/ui/views"
	"github.com/spf13/cobra"
)

var (
	allNamespaces bool
	selector      string
)

// NewDeploymentsCmd creates the deployments command
func NewDeploymentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployments",
		Aliases: []string{"deployment", "deploy"},
		Short:   "Manage Kubernetes deployments",
		Long: `Manage Kubernetes deployments with an interactive interface.

This command provides various subcommands for deployment operations including:
- Viewing deployment lists and details
- Updating deployment images
- Scaling deployments
- Restarting deployments
- Rolling back deployments`,
		Example: `  # List all deployments in current namespace
  k8s-manager deployments list

  # List deployments in all namespaces
  k8s-manager deployments list -A

  # Update deployment image
  k8s-manager deployments update my-deployment --image=myapp:v2

  # Restart deployment
  k8s-manager deployments restart my-deployment`,
		RunE: runDeploymentsInteractive,
	}

	// Add flags
	cmd.PersistentFlags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List deployments across all namespaces")
	cmd.PersistentFlags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to filter on")

	// Add subcommands
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newRestartCmd())
	cmd.AddCommand(newScaleCmd())
	cmd.AddCommand(newRollbackCmd())
	cmd.AddCommand(newRolloutCmd())

	return cmd
}

// runDeploymentsInteractive runs the interactive deployment management UI
func runDeploymentsInteractive(cmd *cobra.Command, args []string) error {
	// This will launch the interactive deployments UI
	return views.ShowDeploymentsView(views.DeploymentsOptions{
		AllNamespaces: allNamespaces,
		Selector:      selector,
	})
}
