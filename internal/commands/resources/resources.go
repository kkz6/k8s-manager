package resources

import (
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/views"
	"github.com/spf13/cobra"
)

var (
	allNamespaces  bool
	namespace      string
	pollInterval   int
	sortBy         string
	descending     bool
	userOnly       bool
	includeSystem  bool
)

// NewResourcesCmd creates the resources command
func NewResourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resources",
		Aliases: []string{"res", "resource", "rm"},
		Short:   "Monitor pod resource usage",
		Long: `Monitor CPU and memory usage of pods in your Kubernetes cluster.

This command provides a real-time dashboard showing:
- Cluster-wide resource utilization
- Per-pod CPU and memory usage with visual bars
- Color-coded usage levels (green/yellow/orange/red)
- Auto-refreshing metrics with configurable polling interval

The dashboard helps identify:
- Over-provisioned pods (consistently low usage)
- Under-provisioned pods (high usage, needs more resources)
- Resource hogs consuming cluster capacity`,
		Example: `  # Monitor user pods only (excludes kube-system, gmp-system, etc.)
  k8s-manager resources

  # Monitor ALL pods including system namespaces
  k8s-manager resources --include-system

  # Monitor pods in a specific namespace
  k8s-manager resources -n laravel-app

  # Monitor with custom poll interval (10 seconds)
  k8s-manager resources --poll 10

  # Sort by memory usage (descending)
  k8s-manager resources --sort memory

  # Sort by name (ascending)
  k8s-manager resources --sort name --asc`,
		RunE: runResourceManager,
	}

	// Add flags
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", true, "Monitor pods across all namespaces")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Monitor pods in specific namespace only")
	cmd.Flags().IntVarP(&pollInterval, "poll", "p", 5, "Polling interval in seconds (1-60)")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "cpu", "Sort by: cpu, memory, or name")
	cmd.Flags().BoolVar(&descending, "desc", true, "Sort in descending order")
	cmd.Flags().BoolVar(&descending, "asc", false, "Sort in ascending order (overrides --desc)")
	cmd.Flags().BoolVar(&includeSystem, "include-system", false, "Include system namespaces (kube-system, gmp-system, etc.)")

	return cmd
}

// runResourceManager launches the resource manager UI
func runResourceManager(cmd *cobra.Command, args []string) error {
	// Build options
	opts := views.DefaultResourceManagerOptions()

	// Namespace handling
	if namespace != "" {
		opts.Namespace = namespace
		opts.AllNamespaces = false
	} else {
		opts.AllNamespaces = allNamespaces
	}

	// Poll interval (clamp to 1-60 seconds)
	if pollInterval < 1 {
		pollInterval = 1
	}
	if pollInterval > 60 {
		pollInterval = 60
	}
	opts.PollInterval = time.Duration(pollInterval) * time.Second

	// Sort options
	switch sortBy {
	case "memory", "mem", "m":
		opts.SortBy = services.SortByMemory
	case "name", "n":
		opts.SortBy = services.SortByName
	default:
		opts.SortBy = services.SortByCPU
	}

	// Handle --asc flag (it's actually inverted in cobra when using BoolVar with false default)
	if cmd.Flags().Changed("asc") {
		opts.Descending = false
	} else {
		opts.Descending = descending
	}

	// Handle system namespace inclusion
	opts.ExcludeSystem = !includeSystem

	return views.ShowResourceManager(opts)
}
