package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	k8sservices "github.com/karthickk/k8s-manager/internal/services"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	allNamespaces bool
)

// NewServicesCmd creates the services command
func NewServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "services",
		Aliases: []string{"svc", "service"},
		Short:   "Manage Kubernetes services",
		Long: `Manage Kubernetes services.

List and view services in your cluster with details about
their type, cluster IP, external IP, and ports.`,
		Example: `  # List services in current namespace
  k8s-manager services list

  # List services in all namespaces
  k8s-manager services list -A

  # Get details of a specific service
  k8s-manager services get my-service`,
		RunE: listServices,
	}

	cmd.PersistentFlags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List services across all namespaces")

	cmd.AddCommand(newListServicesCmd())
	cmd.AddCommand(newGetServiceCmd())

	return cmd
}

func newListServicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List services",
		RunE:    listServices,
	}
}

func newGetServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get NAME",
		Short: "Get service details",
		Args:  cobra.ExactArgs(1),
		RunE:  getService,
	}
}

func listServices(cmd *cobra.Command, args []string) error {
	client, err := k8sservices.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := ""
	if !allNamespaces {
		namespace = k8sservices.GetCurrentNamespace()
	}

	svcList, err := client.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(svcList.Items) == 0 {
		fmt.Println("No services found")
		return nil
	}

	// Print header
	if allNamespaces {
		fmt.Printf("%-20s %-30s %-12s %-16s %-20s %s\n", "NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORTS")
	} else {
		fmt.Printf("%-30s %-12s %-16s %-20s %s\n", "NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORTS")
	}

	for _, svc := range svcList.Items {
		// Get external IP
		externalIP := "<none>"
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			if len(svc.Status.LoadBalancer.Ingress) > 0 {
				if svc.Status.LoadBalancer.Ingress[0].IP != "" {
					externalIP = svc.Status.LoadBalancer.Ingress[0].IP
				} else if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
					externalIP = svc.Status.LoadBalancer.Ingress[0].Hostname
				}
			} else {
				externalIP = "<pending>"
			}
		} else if svc.Spec.Type == corev1.ServiceTypeNodePort {
			externalIP = "<nodes>"
		}

		// Get ports
		var ports []string
		for _, port := range svc.Spec.Ports {
			portStr := fmt.Sprintf("%d/%s", port.Port, port.Protocol)
			if port.NodePort != 0 {
				portStr = fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol)
			}
			ports = append(ports, portStr)
		}
		portsStr := strings.Join(ports, ",")
		if portsStr == "" {
			portsStr = "<none>"
		}

		clusterIP := svc.Spec.ClusterIP
		if clusterIP == "" {
			clusterIP = "None"
		}

		if allNamespaces {
			fmt.Printf("%-20s %-30s %-12s %-16s %-20s %s\n",
				svc.Namespace, svc.Name, svc.Spec.Type, clusterIP, externalIP, portsStr)
		} else {
			fmt.Printf("%-30s %-12s %-16s %-20s %s\n",
				svc.Name, svc.Spec.Type, clusterIP, externalIP, portsStr)
		}
	}

	return nil
}

func getService(cmd *cobra.Command, args []string) error {
	client, err := k8sservices.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := k8sservices.GetCurrentNamespace()
	svc, err := client.Clientset.CoreV1().Services(namespace).Get(ctx, args[0], metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	fmt.Printf("Name:         %s\n", svc.Name)
	fmt.Printf("Namespace:    %s\n", svc.Namespace)
	fmt.Printf("Type:         %s\n", svc.Spec.Type)
	fmt.Printf("Cluster IP:   %s\n", svc.Spec.ClusterIP)

	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer && len(svc.Status.LoadBalancer.Ingress) > 0 {
		fmt.Printf("External IP:  %s\n", svc.Status.LoadBalancer.Ingress[0].IP)
	}

	fmt.Printf("Ports:\n")
	for _, port := range svc.Spec.Ports {
		fmt.Printf("  - %s: %d", port.Name, port.Port)
		if port.NodePort != 0 {
			fmt.Printf(" -> %d", port.NodePort)
		}
		fmt.Printf(" (%s)\n", port.Protocol)
	}

	if len(svc.Spec.Selector) > 0 {
		fmt.Printf("Selector:\n")
		for k, v := range svc.Spec.Selector {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}

	fmt.Printf("Age:          %s\n", k8sservices.FormatAge(svc.CreationTimestamp.Time))

	return nil
}
