package pods

import (
	"github.com/karthickk/k8s-manager/internal/ui/views"
	"github.com/spf13/cobra"
)

var (
	allNamespaces bool
	selector      string
	showAll       bool
)

// NewPodsCmd creates the pods command
func NewPodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pods",
		Aliases: []string{"pod", "po"},
		Short:   "Manage Kubernetes pods",
		Long: `Manage Kubernetes pods with an interactive interface.
		
This command provides various subcommands for pod operations including:
- Viewing pod lists and details
- Executing commands in pods
- Viewing and following logs
- Managing pod environment variables
- Restarting and deleting pods`,
		Example: `  # List all pods in current namespace
  k8s-manager pods list
  
  # List pods in all namespaces
  k8s-manager pods list -A
  
  # Get pod logs
  k8s-manager pods logs my-pod
  
  # Execute into a pod
  k8s-manager pods exec my-pod`,
		RunE: runPodsInteractive,
	}

	// Add flags
	cmd.PersistentFlags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List pods across all namespaces")
	cmd.PersistentFlags().StringVarP(&selector, "selector", "l", "", "Selector (label query) to filter on")
	cmd.PersistentFlags().BoolVar(&showAll, "show-all", false, "Show all pods (including terminated)")

	// Add subcommands
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newDescribeCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRestartCmd())
	cmd.AddCommand(newEnvCmd())

	return cmd
}

// runPodsInteractive runs the interactive pod management UI
func runPodsInteractive(cmd *cobra.Command, args []string) error {
	// This will launch the interactive pods UI
	return views.ShowPodsView(views.PodsOptions{
		AllNamespaces: allNamespaces,
		Selector:      selector,
		ShowAll:       showAll,
	})
}