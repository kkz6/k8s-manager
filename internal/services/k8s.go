package services

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sClient wraps the Kubernetes client
type K8sClient struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

var clientInstance *K8sClient

// GetK8sClient returns a singleton Kubernetes client
func GetK8sClient() (*K8sClient, error) {
	if clientInstance != nil {
		return clientInstance, nil
	}

	config, err := buildConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	clientInstance = &K8sClient{
		Clientset: clientset,
		Config:    config,
	}

	return clientInstance, nil
}

// buildConfig builds the Kubernetes client configuration
func buildConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig
	kubeconfig := getKubeconfig()
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	// Override context if specified
	context := viper.GetString("context")
	if context == "" {
		// Check for cluster_name as fallback
		context = viper.GetString("k8s.cluster_name")
	}
	
	if context != "" {
		// First, check if the context exists
		configAccess := clientcmd.NewDefaultPathOptions()
		kubeConfig, err := configAccess.GetStartingConfig()
		if err == nil {
			if _, exists := kubeConfig.Contexts[context]; exists {
				// Context exists, use it
				configOverrides := &clientcmd.ConfigOverrides{
					CurrentContext: context,
				}
				config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
					&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
					configOverrides,
				).ClientConfig()
				if err != nil {
					return nil, fmt.Errorf("failed to build config with context override: %w", err)
				}
			} else {
				// Context doesn't exist, log warning and use default
				fmt.Printf("Warning: Context '%s' not found in kubeconfig, using current context\n", context)
			}
		}
	}

	return config, nil
}

// getKubeconfig returns the path to the kubeconfig file
func getKubeconfig() string {
	// Check environment variable
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".kube", "config")
	}

	return ""
}

// GetNamespace returns the namespace to use
func GetNamespace(cmd *cobra.Command) string {
	// Check flag first if cmd is provided
	if cmd != nil {
		if ns, _ := cmd.Flags().GetString("namespace"); ns != "" {
			return ns
		}
	}

	// Check viper config
	if ns := viper.GetString("namespace"); ns != "" {
		return ns
	}

	// Check k8s.namespace in config
	if ns := viper.GetString("k8s.namespace"); ns != "" {
		return ns
	}

	// Default to "default" namespace
	return "default"
}

// GetCurrentNamespace returns the current namespace without requiring a command
func GetCurrentNamespace() string {
	return GetNamespace(nil)
}

// GetPodReadyCount returns the ready count string for a pod
func GetPodReadyCount(pod *corev1.Pod) string {
	readyContainers := 0
	totalContainers := len(pod.Status.ContainerStatuses)

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			readyContainers++
		}
	}

	return fmt.Sprintf("%d/%d", readyContainers, totalContainers)
}

// GetPodRestarts returns the total restart count for a pod
func GetPodRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}

// FormatAge formats a duration in a human-readable way
func FormatAge(t time.Time) string {
	duration := time.Since(t)
	
	if duration.Hours() > 24*365 {
		return fmt.Sprintf("%.0fy", duration.Hours()/(24*365))
	} else if duration.Hours() > 24*30 {
		return fmt.Sprintf("%.0fmo", duration.Hours()/(24*30))
	} else if duration.Hours() > 24 {
		return fmt.Sprintf("%.0fd", duration.Hours()/24)
	} else if duration.Hours() > 1 {
		return fmt.Sprintf("%.0fh", duration.Hours())
	} else if duration.Minutes() > 1 {
		return fmt.Sprintf("%.0fm", duration.Minutes())
	} else {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	}
}

// GetPodStatus returns a formatted pod status
func GetPodStatus(pod *corev1.Pod) string {
	// Check for init container statuses
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil {
			return fmt.Sprintf("Init:%s", cs.State.Waiting.Reason)
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return fmt.Sprintf("Init:Error")
		}
	}

	// Check container statuses
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil {
			return cs.State.Terminated.Reason
		}
	}

	// Check pod conditions
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
			return "NotReady"
		}
	}

	return string(pod.Status.Phase)
}