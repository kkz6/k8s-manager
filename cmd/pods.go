package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/karthickk/k8s-manager/pkg/utils"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newPodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pods",
		Short: "Manage Kubernetes pods",
		Long:  `List, describe, restart, and perform other operations on Kubernetes pods.`,
	}

	cmd.AddCommand(newPodsListCmd())
	cmd.AddCommand(newPodsGetCmd())
	cmd.AddCommand(newPodsRestartCmd())
	cmd.AddCommand(newPodsDeleteCmd())
	cmd.AddCommand(newPodsSSHCmd())

	return cmd
}

func newPodsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pods in the namespace",
		Long:  `List all Kubernetes pods in the current namespace.`,
		RunE:  runPodsList,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace to list pods from (overrides config)")
	cmd.Flags().BoolP("all-namespaces", "A", false, "List pods from all namespaces")
	cmd.Flags().StringP("selector", "l", "", "Selector (label query) to filter on")
	cmd.Flags().StringP("field-selector", "", "", "Field selector to filter on")
	cmd.Flags().BoolP("show-labels", "", false, "Show pod labels")

	return cmd
}

func newPodsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <pod-name>",
		Short: "Get details of a specific pod",
		Long:  `Get detailed information about a specific Kubernetes pod.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runPodsGet,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod (overrides config)")
	cmd.Flags().BoolP("yaml", "y", false, "Output in YAML format")

	return cmd
}

func newPodsRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart <pod-name-or-deployment>",
		Short: "Restart pods",
		Long:  `Restart a specific pod or all pods in a deployment.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runPodsRestart,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod/deployment (overrides config)")
	cmd.Flags().BoolP("deployment", "d", false, "Restart deployment instead of individual pod")
	cmd.Flags().BoolP("force", "", false, "Skip confirmation prompt")

	return cmd
}

func newPodsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <pod-name>",
		Short: "Delete a pod",
		Long:  `Delete a Kubernetes pod from the cluster.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runPodsDelete,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod (overrides config)")
	cmd.Flags().BoolP("force", "", false, "Skip confirmation prompt")
	cmd.Flags().Int64P("grace-period", "", 30, "Grace period in seconds")

	return cmd
}

func newPodsSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh <pod-name>",
		Short: "SSH into a pod",
		Long:  `Execute an interactive shell session in a Kubernetes pod.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runPodsSSH,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod (overrides config)")
	cmd.Flags().StringP("container", "c", "", "Container name (if pod has multiple containers)")
	cmd.Flags().StringP("shell", "s", "/bin/bash", "Shell to use (/bin/bash, /bin/sh, etc.)")

	return cmd
}

func runPodsList(cmd *cobra.Command, args []string) error {
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	allNamespaces, _ := cmd.Flags().GetBool("all-namespaces")
	selector, _ := cmd.Flags().GetString("selector")
	fieldSelector, _ := cmd.Flags().GetString("field-selector")
	showLabels, _ := cmd.Flags().GetBool("show-labels")

	if namespace == "" && !allNamespaces {
		namespace = client.GetNamespace()
	}

	listOptions := metav1.ListOptions{}
	if selector != "" {
		listOptions.LabelSelector = selector
	}
	if fieldSelector != "" {
		listOptions.FieldSelector = fieldSelector
	}

	ctx := cmd.Context()
	var pods *corev1.PodList

	if allNamespaces {
		pods, err = client.Clientset.CoreV1().Pods("").List(ctx, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}
	} else {
		pods, err = client.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
		}
	}

	if len(pods.Items) == 0 {
		if allNamespaces {
			fmt.Println("No pods found in any namespace")
		} else {
			fmt.Printf("No pods found in namespace '%s'\n", namespace)
		}
		return nil
	}

	// Display pods in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if allNamespaces {
		if showLabels {
			fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tLABELS")
		} else {
			fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE")
		}
	} else {
		if showLabels {
			fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tLABELS")
		} else {
			fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")
		}
	}

	for _, pod := range pods.Items {
		ready := getPodReadyStatus(&pod)
		status := string(pod.Status.Phase)
		restarts := getPodRestartCount(&pod)
		age := utils.FormatAge(pod.CreationTimestamp.Time)

		var labels string
		if showLabels {
			labelPairs := []string{}
			for k, v := range pod.Labels {
				labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
			}
			labels = strings.Join(labelPairs, ",")
		}

		if allNamespaces {
			if showLabels {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
					pod.Namespace, pod.Name, ready, status, restarts, age, labels)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
					pod.Namespace, pod.Name, ready, status, restarts, age)
			}
		} else {
			if showLabels {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
					pod.Name, ready, status, restarts, age, labels)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					pod.Name, ready, status, restarts, age)
			}
		}
	}
	w.Flush()

	return nil
}

func runPodsGet(cmd *cobra.Command, args []string) error {
	podName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	outputYAML, _ := cmd.Flags().GetBool("yaml")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	ctx := cmd.Context()
	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	if outputYAML {
		// TODO: Implement YAML output
		fmt.Println("YAML output not yet implemented")
		return nil
	}

	fmt.Printf("Name:         %s\n", pod.Name)
	fmt.Printf("Namespace:    %s\n", pod.Namespace)
	fmt.Printf("Status:       %s\n", pod.Status.Phase)
	fmt.Printf("Node:         %s\n", pod.Spec.NodeName)
	fmt.Printf("Created:      %s\n", pod.CreationTimestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Ready:        %s\n", getPodReadyStatus(pod))
	fmt.Printf("Restarts:     %d\n", getPodRestartCount(pod))
	fmt.Println()

	if len(pod.Spec.Containers) > 0 {
		fmt.Println("Containers:")
		for _, container := range pod.Spec.Containers {
			fmt.Printf("  - Name:   %s\n", container.Name)
			fmt.Printf("    Image:  %s\n", container.Image)
			if len(container.Ports) > 0 {
				fmt.Printf("    Ports:  ")
				for i, port := range container.Ports {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%d/%s", port.ContainerPort, port.Protocol)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}

	if len(pod.Status.ContainerStatuses) > 0 {
		fmt.Println("Container Status:")
		for _, status := range pod.Status.ContainerStatuses {
			fmt.Printf("  - %s: Ready=%t, RestartCount=%d\n",
				status.Name, status.Ready, status.RestartCount)
		}
	}

	return nil
}

func runPodsRestart(cmd *cobra.Command, args []string) error {
	name := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	isDeployment, _ := cmd.Flags().GetBool("deployment")
	force, _ := cmd.Flags().GetBool("force")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	if !force {
		if isDeployment {
			fmt.Printf("Are you sure you want to restart deployment '%s' in namespace '%s'? (y/N): ", name, namespace)
		} else {
			fmt.Printf("Are you sure you want to restart pod '%s' in namespace '%s'? (y/N): ", name, namespace)
		}
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Restart cancelled")
			return nil
		}
	}

	ctx := cmd.Context()

	if isDeployment {
		// Restart deployment by adding/updating restart annotation
		deployment, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get deployment %s: %w", name, err)
		}

		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")

		_, err = client.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to restart deployment %s: %w", name, err)
		}

		fmt.Printf("✅ Deployment '%s' restart initiated in namespace '%s'\n", name, namespace)
	} else {
		// Delete pod to restart it (if managed by a controller)
		err = client.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to restart pod %s: %w", name, err)
		}

		fmt.Printf("✅ Pod '%s' restart initiated in namespace '%s'\n", name, namespace)
	}

	return nil
}

func runPodsDelete(cmd *cobra.Command, args []string) error {
	podName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	force, _ := cmd.Flags().GetBool("force")
	gracePeriod, _ := cmd.Flags().GetInt64("grace-period")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	if !force {
		fmt.Printf("Are you sure you want to delete pod '%s' in namespace '%s'? (y/N): ", podName, namespace)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	ctx := cmd.Context()
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	err = client.Clientset.CoreV1().Pods(namespace).Delete(ctx, podName, deleteOptions)
	if err != nil {
		return fmt.Errorf("failed to delete pod %s: %w", podName, err)
	}

	fmt.Printf("✅ Pod '%s' deleted successfully from namespace '%s'\n", podName, namespace)
	return nil
}

func runPodsSSH(cmd *cobra.Command, args []string) error {
	podName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	container, _ := cmd.Flags().GetString("container")
	shell, _ := cmd.Flags().GetString("shell")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	// Check if pod exists and get container info
	ctx := cmd.Context()
	pod, err := client.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	// If container not specified and pod has multiple containers, ask user to choose
	if container == "" && len(pod.Spec.Containers) > 1 {
		fmt.Println("Pod has multiple containers:")
		for i, c := range pod.Spec.Containers {
			fmt.Printf("  %d. %s\n", i+1, c.Name)
		}
		fmt.Printf("Select container (1-%d): ", len(pod.Spec.Containers))

		var choice int
		if _, err := fmt.Scanln(&choice); err != nil || choice < 1 || choice > len(pod.Spec.Containers) {
			return fmt.Errorf("invalid container selection")
		}
		container = pod.Spec.Containers[choice-1].Name
	} else if container == "" {
		container = pod.Spec.Containers[0].Name
	}

	fmt.Printf("🔗 Connecting to pod '%s', container '%s'...\n", podName, container)

	// Use kubectl exec for interactive session
	return k8s.ExecIntoPod(namespace, podName, container, shell)
}

// Helper functions

func getPodReadyStatus(pod *corev1.Pod) string {
	readyContainers := 0
	totalContainers := len(pod.Spec.Containers)

	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			readyContainers++
		}
	}

	return fmt.Sprintf("%d/%d", readyContainers, totalContainers)
}

func getPodRestartCount(pod *corev1.Pod) int32 {
	var totalRestarts int32
	for _, status := range pod.Status.ContainerStatuses {
		totalRestarts += status.RestartCount
	}
	return totalRestarts
}
