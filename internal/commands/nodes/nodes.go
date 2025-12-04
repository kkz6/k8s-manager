package nodes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewNodesCmd creates the nodes command
func NewNodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "nodes",
		Aliases: []string{"node", "no"},
		Short:   "View Kubernetes nodes",
		Long: `View Kubernetes cluster nodes.

List nodes and their resources including CPU, memory, and pod capacity.`,
		Example: `  # List all nodes
  k8s-manager nodes list

  # Get details of a specific node
  k8s-manager nodes get node-name`,
		RunE: listNodes,
	}

	cmd.AddCommand(newListNodesCmd())
	cmd.AddCommand(newGetNodeCmd())

	return cmd
}

func newListNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List nodes",
		RunE:    listNodes,
	}
}

func newGetNodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get NAME",
		Short: "Get node details",
		Args:  cobra.ExactArgs(1),
		RunE:  getNode,
	}
}

func listNodes(cmd *cobra.Command, args []string) error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeList, err := client.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodeList.Items) == 0 {
		fmt.Println("No nodes found")
		return nil
	}

	fmt.Printf("%-40s %-10s %-15s %-10s %-12s %s\n", "NAME", "STATUS", "ROLES", "CPU", "MEMORY", "AGE")

	for _, node := range nodeList.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					status = "Ready"
				}
				break
			}
		}

		roles := getNodeRoles(&node)
		if roles == "" {
			roles = "<none>"
		}

		cpu := node.Status.Allocatable.Cpu()
		mem := node.Status.Allocatable.Memory()

		cpuStr := "N/A"
		memStr := "N/A"
		if cpu != nil {
			cpuStr = services.FormatCPU(cpu.MilliValue())
		}
		if mem != nil {
			memStr = services.FormatMemory(mem.Value())
		}

		age := services.FormatAge(node.CreationTimestamp.Time)

		fmt.Printf("%-40s %-10s %-15s %-10s %-12s %s\n",
			node.Name, status, roles, cpuStr, memStr, age)
	}

	return nil
}

func getNode(cmd *cobra.Command, args []string) error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := client.Clientset.CoreV1().Nodes().Get(ctx, args[0], metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// Status
	status := "NotReady"
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				status = "Ready"
			}
			break
		}
	}

	fmt.Printf("Name:              %s\n", node.Name)
	fmt.Printf("Status:            %s\n", status)
	fmt.Printf("Roles:             %s\n", getNodeRoles(node))
	fmt.Printf("Age:               %s\n", services.FormatAge(node.CreationTimestamp.Time))
	fmt.Println()

	// Node Info
	fmt.Println("System Info:")
	fmt.Printf("  Kubelet Version:   %s\n", node.Status.NodeInfo.KubeletVersion)
	fmt.Printf("  Container Runtime: %s\n", node.Status.NodeInfo.ContainerRuntimeVersion)
	fmt.Printf("  OS:                %s\n", node.Status.NodeInfo.OperatingSystem)
	fmt.Printf("  OS Image:          %s\n", node.Status.NodeInfo.OSImage)
	fmt.Printf("  Architecture:      %s\n", node.Status.NodeInfo.Architecture)
	fmt.Println()

	// Resources
	fmt.Println("Allocatable Resources:")
	cpu := node.Status.Allocatable.Cpu()
	mem := node.Status.Allocatable.Memory()
	pods := node.Status.Allocatable.Pods()

	if cpu != nil {
		fmt.Printf("  CPU:     %s\n", services.FormatCPU(cpu.MilliValue()))
	}
	if mem != nil {
		fmt.Printf("  Memory:  %s\n", services.FormatMemory(mem.Value()))
	}
	if pods != nil {
		fmt.Printf("  Pods:    %s\n", pods.String())
	}
	fmt.Println()

	// Conditions
	fmt.Println("Conditions:")
	for _, cond := range node.Status.Conditions {
		fmt.Printf("  %-20s %s\n", cond.Type, cond.Status)
	}

	// Addresses
	if len(node.Status.Addresses) > 0 {
		fmt.Println()
		fmt.Println("Addresses:")
		for _, addr := range node.Status.Addresses {
			fmt.Printf("  %-15s %s\n", addr.Type, addr.Address)
		}
	}

	return nil
}

func getNodeRoles(node *corev1.Node) string {
	var roles []string
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	return strings.Join(roles, ",")
}
