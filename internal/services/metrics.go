package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// PodMetrics contains resource usage metrics for a pod
type PodMetrics struct {
	Name           string
	Namespace      string
	Status         string
	Containers     []ContainerMetrics
	TotalCPU       int64 // in millicores
	TotalMemory    int64 // in bytes
	CPURequest     int64 // in millicores
	MemoryRequest  int64 // in bytes
	CPULimit       int64 // in millicores
	MemoryLimit    int64 // in bytes
	CPUPercent     float64
	MemoryPercent  float64
	Age            time.Duration
	Ready          string
	Restarts       int32
}

// ContainerMetrics contains metrics for a single container
type ContainerMetrics struct {
	Name        string
	CPU         int64 // in millicores
	Memory      int64 // in bytes
	CPURequest  int64
	MemoryRequest int64
	CPULimit    int64
	MemoryLimit int64
}

// ClusterMetrics contains aggregated cluster resource metrics
type ClusterMetrics struct {
	TotalPods        int
	RunningPods      int
	TotalCPU         int64
	TotalMemory      int64
	UsedCPU          int64
	UsedMemory       int64
	CPUPercent       float64
	MemoryPercent    float64
	LastUpdated      time.Time
	Namespaces       map[string]NamespaceMetrics
}

// NamespaceMetrics contains metrics aggregated by namespace
type NamespaceMetrics struct {
	Name         string
	PodCount     int
	TotalCPU     int64
	TotalMemory  int64
	CPURequest   int64
	MemoryRequest int64
}

// SortBy represents the field to sort pods by
type SortBy string

const (
	SortByCPU    SortBy = "cpu"
	SortByMemory SortBy = "memory"
	SortByName   SortBy = "name"
)

// MetricsClient wraps the Kubernetes metrics client
type MetricsClient struct {
	metricsClient *metricsclient.Clientset
	k8sClient     *kubernetes.Clientset
}

var metricsInstance *MetricsClient

// GetMetricsClient returns a singleton metrics client
func GetMetricsClient() (*MetricsClient, error) {
	if metricsInstance != nil {
		return metricsInstance, nil
	}

	k8sClient, err := GetK8sClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	metricsClientset, err := metricsclient.NewForConfig(k8sClient.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	metricsInstance = &MetricsClient{
		metricsClient: metricsClientset,
		k8sClient:     k8sClient.Clientset,
	}

	return metricsInstance, nil
}

// GetPodMetrics fetches metrics for all pods in the specified namespace (empty for all namespaces)
func (m *MetricsClient) GetPodMetrics(ctx context.Context, namespace string) ([]PodMetrics, error) {
	// Get pod metrics from metrics server
	var podMetricsList *metricsv1beta1.PodMetricsList
	var err error

	if namespace == "" {
		podMetricsList, err = m.metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	} else {
		podMetricsList, err = m.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	// Get pod details for resource requests/limits
	var podList *corev1.PodList
	if namespace == "" {
		podList, err = m.k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	} else {
		podList, err = m.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	// Create a map for quick pod lookup
	podMap := make(map[string]*corev1.Pod)
	for i := range podList.Items {
		pod := &podList.Items[i]
		key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		podMap[key] = pod
	}

	// Build pod metrics list
	result := make([]PodMetrics, 0, len(podMetricsList.Items))
	for _, pm := range podMetricsList.Items {
		key := fmt.Sprintf("%s/%s", pm.Namespace, pm.Name)
		pod, exists := podMap[key]
		if !exists {
			continue
		}

		podMetric := PodMetrics{
			Name:      pm.Name,
			Namespace: pm.Namespace,
			Status:    GetPodStatus(pod),
			Age:       time.Since(pod.CreationTimestamp.Time),
			Ready:     GetPodReadyCount(pod),
			Restarts:  GetPodRestarts(pod),
		}

		// Calculate container metrics
		for _, container := range pm.Containers {
			cpu := container.Usage.Cpu().MilliValue()
			memory := container.Usage.Memory().Value()

			// Find container spec for requests/limits
			var cpuReq, memReq, cpuLim, memLim int64
			for _, c := range pod.Spec.Containers {
				if c.Name == container.Name {
					cpuReq = c.Resources.Requests.Cpu().MilliValue()
					memReq = c.Resources.Requests.Memory().Value()
					cpuLim = c.Resources.Limits.Cpu().MilliValue()
					memLim = c.Resources.Limits.Memory().Value()
					break
				}
			}

			podMetric.Containers = append(podMetric.Containers, ContainerMetrics{
				Name:          container.Name,
				CPU:           cpu,
				Memory:        memory,
				CPURequest:    cpuReq,
				MemoryRequest: memReq,
				CPULimit:      cpuLim,
				MemoryLimit:   memLim,
			})

			podMetric.TotalCPU += cpu
			podMetric.TotalMemory += memory
			podMetric.CPURequest += cpuReq
			podMetric.MemoryRequest += memReq
			podMetric.CPULimit += cpuLim
			podMetric.MemoryLimit += memLim
		}

		// Calculate percentages based on requests (or limits if no requests)
		if podMetric.CPURequest > 0 {
			podMetric.CPUPercent = float64(podMetric.TotalCPU) / float64(podMetric.CPURequest) * 100
		} else if podMetric.CPULimit > 0 {
			podMetric.CPUPercent = float64(podMetric.TotalCPU) / float64(podMetric.CPULimit) * 100
		}

		if podMetric.MemoryRequest > 0 {
			podMetric.MemoryPercent = float64(podMetric.TotalMemory) / float64(podMetric.MemoryRequest) * 100
		} else if podMetric.MemoryLimit > 0 {
			podMetric.MemoryPercent = float64(podMetric.TotalMemory) / float64(podMetric.MemoryLimit) * 100
		}

		result = append(result, podMetric)
	}

	return result, nil
}

// GetClusterMetrics fetches aggregated cluster metrics
func (m *MetricsClient) GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error) {
	podMetrics, err := m.GetPodMetrics(ctx, "")
	if err != nil {
		return nil, err
	}

	cluster := &ClusterMetrics{
		TotalPods:   len(podMetrics),
		LastUpdated: time.Now(),
		Namespaces:  make(map[string]NamespaceMetrics),
	}

	for _, pm := range podMetrics {
		cluster.UsedCPU += pm.TotalCPU
		cluster.UsedMemory += pm.TotalMemory
		cluster.TotalCPU += pm.CPURequest
		cluster.TotalMemory += pm.MemoryRequest

		if pm.Status == "Running" {
			cluster.RunningPods++
		}

		// Aggregate by namespace
		ns, exists := cluster.Namespaces[pm.Namespace]
		if !exists {
			ns = NamespaceMetrics{Name: pm.Namespace}
		}
		ns.PodCount++
		ns.TotalCPU += pm.TotalCPU
		ns.TotalMemory += pm.TotalMemory
		ns.CPURequest += pm.CPURequest
		ns.MemoryRequest += pm.MemoryRequest
		cluster.Namespaces[pm.Namespace] = ns
	}

	if cluster.TotalCPU > 0 {
		cluster.CPUPercent = float64(cluster.UsedCPU) / float64(cluster.TotalCPU) * 100
	}
	if cluster.TotalMemory > 0 {
		cluster.MemoryPercent = float64(cluster.UsedMemory) / float64(cluster.TotalMemory) * 100
	}

	return cluster, nil
}

// SortPodMetrics sorts pod metrics by the specified field
func SortPodMetrics(pods []PodMetrics, by SortBy, descending bool) {
	sort.Slice(pods, func(i, j int) bool {
		var less bool
		switch by {
		case SortByCPU:
			less = pods[i].CPUPercent < pods[j].CPUPercent
		case SortByMemory:
			less = pods[i].MemoryPercent < pods[j].MemoryPercent
		case SortByName:
			less = pods[i].Name < pods[j].Name
		default:
			less = pods[i].CPUPercent < pods[j].CPUPercent
		}
		if descending {
			return !less
		}
		return less
	})
}

// FormatCPU formats CPU millicores to a human-readable string
func FormatCPU(millicores int64) string {
	if millicores >= 1000 {
		return fmt.Sprintf("%.1f", float64(millicores)/1000)
	}
	return fmt.Sprintf("%dm", millicores)
}

// FormatMemory formats bytes to a human-readable string
func FormatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.0fMi", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.0fKi", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// ParseResourceQuantity parses a Kubernetes resource quantity string
func ParseResourceQuantity(s string) (int64, error) {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return 0, err
	}
	return q.Value(), nil
}
