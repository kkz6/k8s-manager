package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/karthickk/k8s-manager/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gopkg.in/yaml.v2"
)

// PodEnvManager handles environment variables for pods
type PodEnvManager struct {
	pod         *corev1.Pod
	client      *k8s.Client
	envVars     []EnvVarInfo
	secrets     []string
	configMaps  []string
}

// EnvVarInfo holds environment variable information
type EnvVarInfo struct {
	Name   string
	Value  string
	Source string // "direct", "secret", "configmap", "field"
	Ref    string // Reference name if from secret/configmap
}

// NewPodEnvManager creates a new environment manager for a pod
func NewPodEnvManager(pod *corev1.Pod, client *k8s.Client) *PodEnvManager {
	return &PodEnvManager{
		pod:    pod,
		client: client,
	}
}

// LoadEnvVars loads all environment variables from the pod
func (m *PodEnvManager) LoadEnvVars() error {
	m.envVars = []EnvVarInfo{}
	m.secrets = []string{}
	m.configMaps = []string{}

	// Process each container
	for _, container := range m.pod.Spec.Containers {
		// Direct env vars
		for _, env := range container.Env {
			info := EnvVarInfo{
				Name: env.Name,
			}

			if env.Value != "" {
				info.Value = env.Value
				info.Source = "direct"
			} else if env.ValueFrom != nil {
				if env.ValueFrom.SecretKeyRef != nil {
					info.Source = "secret"
					info.Ref = env.ValueFrom.SecretKeyRef.Name
					info.Value = fmt.Sprintf("secret:%s/%s",
						env.ValueFrom.SecretKeyRef.Name,
						env.ValueFrom.SecretKeyRef.Key)

					// Track secret reference
					if !contains(m.secrets, env.ValueFrom.SecretKeyRef.Name) {
						m.secrets = append(m.secrets, env.ValueFrom.SecretKeyRef.Name)
					}
				} else if env.ValueFrom.ConfigMapKeyRef != nil {
					info.Source = "configmap"
					info.Ref = env.ValueFrom.ConfigMapKeyRef.Name
					info.Value = fmt.Sprintf("configmap:%s/%s",
						env.ValueFrom.ConfigMapKeyRef.Name,
						env.ValueFrom.ConfigMapKeyRef.Key)

					// Track configmap reference
					if !contains(m.configMaps, env.ValueFrom.ConfigMapKeyRef.Name) {
						m.configMaps = append(m.configMaps, env.ValueFrom.ConfigMapKeyRef.Name)
					}
				} else if env.ValueFrom.FieldRef != nil {
					info.Source = "field"
					info.Value = fmt.Sprintf("field:%s", env.ValueFrom.FieldRef.FieldPath)
				}
			}

			m.envVars = append(m.envVars, info)
		}

		// EnvFrom sources (entire secrets/configmaps)
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil {
				if !contains(m.secrets, envFrom.SecretRef.Name) {
					m.secrets = append(m.secrets, envFrom.SecretRef.Name)
				}
				// Add placeholder for all keys from secret
				m.envVars = append(m.envVars, EnvVarInfo{
					Name:   fmt.Sprintf("* All from secret: %s", envFrom.SecretRef.Name),
					Source: "secret",
					Ref:    envFrom.SecretRef.Name,
					Value:  "Multiple values",
				})
			}
			if envFrom.ConfigMapRef != nil {
				if !contains(m.configMaps, envFrom.ConfigMapRef.Name) {
					m.configMaps = append(m.configMaps, envFrom.ConfigMapRef.Name)
				}
				// Add placeholder for all keys from configmap
				m.envVars = append(m.envVars, EnvVarInfo{
					Name:   fmt.Sprintf("* All from configmap: %s", envFrom.ConfigMapRef.Name),
					Source: "configmap",
					Ref:    envFrom.ConfigMapRef.Name,
					Value:  "Multiple values",
				})
			}
		}
	}

	return nil
}

// GetEnvTemplate generates an environment template from the pod
func (m *PodEnvManager) GetEnvTemplate() (string, error) {
	template := EnvTemplate{
		Name:        m.pod.Name + "-env-template",
		Description: fmt.Sprintf("Environment template from pod %s", m.pod.Name),
		EnvVars:     []EnvTemplateVar{},
		Secrets:     m.secrets,
		ConfigMaps:  m.configMaps,
	}

	for _, env := range m.envVars {
		if !strings.HasPrefix(env.Name, "*") { // Skip aggregate entries
			template.EnvVars = append(template.EnvVars, EnvTemplateVar{
				Name:   env.Name,
				Value:  env.Value,
				Source: env.Source,
				Ref:    env.Ref,
			})
		}
	}

	data, err := yaml.Marshal(template)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// CopyEnvToNewPod copies environment configuration to a new pod spec
func (m *PodEnvManager) CopyEnvToNewPod(targetPod *corev1.Pod) error {
	if len(targetPod.Spec.Containers) == 0 {
		return fmt.Errorf("target pod has no containers")
	}

	// Copy to first container (could be enhanced to handle multiple)
	container := &targetPod.Spec.Containers[0]

	// Clear existing env
	container.Env = []corev1.EnvVar{}
	container.EnvFrom = []corev1.EnvFromSource{}

	// Copy env vars from source pod
	if len(m.pod.Spec.Containers) > 0 {
		sourceContainer := m.pod.Spec.Containers[0]
		container.Env = append(container.Env, sourceContainer.Env...)
		container.EnvFrom = append(container.EnvFrom, sourceContainer.EnvFrom...)
	}

	return nil
}

// EnvTemplate represents a reusable environment configuration
type EnvTemplate struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	EnvVars     []EnvTemplateVar  `yaml:"envVars"`
	Secrets     []string          `yaml:"secrets"`
	ConfigMaps  []string          `yaml:"configMaps"`
}

// EnvTemplateVar represents a single environment variable in a template
type EnvTemplateVar struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value"`
	Source string `yaml:"source"`
	Ref    string `yaml:"ref,omitempty"`
}

// ViewPodEnvVars displays environment variables for a pod
func ViewPodEnvVars(pod *corev1.Pod, client *k8s.Client) string {
	var s strings.Builder

	s.WriteString("\033[H\033[2J") // Clear screen
	s.WriteString(devToolsTitleStyle.Render(fmt.Sprintf("🔧 Environment Variables: %s", pod.Name)))
	s.WriteString("\n\n")

	manager := NewPodEnvManager(pod, client)
	if err := manager.LoadEnvVars(); err != nil {
		s.WriteString(devToolsErrorStyle.Render("Error loading environment variables: " + err.Error()))
		return s.String()
	}

	// Group by source
	direct := []EnvVarInfo{}
	secrets := []EnvVarInfo{}
	configmaps := []EnvVarInfo{}
	fields := []EnvVarInfo{}

	for _, env := range manager.envVars {
		switch env.Source {
		case "direct":
			direct = append(direct, env)
		case "secret":
			secrets = append(secrets, env)
		case "configmap":
			configmaps = append(configmaps, env)
		case "field":
			fields = append(fields, env)
		}
	}

	// Display direct values
	if len(direct) > 0 {
		s.WriteString(devToolsNumberStyle.Render("Direct Environment Variables:"))
		s.WriteString("\n")
		for _, env := range direct {
			s.WriteString(fmt.Sprintf("  %s = %s\n",
				devToolsInfoStyle.Render(env.Name),
				devToolsDescriptionStyle.Render(truncateValue(env.Value, 50))))
		}
		s.WriteString("\n")
	}

	// Display secret references
	if len(secrets) > 0 {
		s.WriteString(devToolsNumberStyle.Render("From Secrets:"))
		s.WriteString("\n")
		for _, env := range secrets {
			s.WriteString(fmt.Sprintf("  %s → %s\n",
				devToolsInfoStyle.Render(env.Name),
				devToolsWarningStyle.Render(env.Value)))
		}
		s.WriteString("\n")
	}

	// Display configmap references
	if len(configmaps) > 0 {
		s.WriteString(devToolsNumberStyle.Render("From ConfigMaps:"))
		s.WriteString("\n")
		for _, env := range configmaps {
			s.WriteString(fmt.Sprintf("  %s → %s\n",
				devToolsInfoStyle.Render(env.Name),
				devToolsSuccessStyle.Render(env.Value)))
		}
		s.WriteString("\n")
	}

	// Display field references
	if len(fields) > 0 {
		s.WriteString(devToolsNumberStyle.Render("From Pod Fields:"))
		s.WriteString("\n")
		for _, env := range fields {
			s.WriteString(fmt.Sprintf("  %s → %s\n",
				devToolsInfoStyle.Render(env.Name),
				devToolsDescriptionStyle.Render(env.Value)))
		}
		s.WriteString("\n")
	}

	// Summary
	s.WriteString("\n")
	s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf(
		"Total: %d env vars | %d from secrets | %d from configmaps",
		len(manager.envVars), len(secrets), len(configmaps))))

	return s.String()
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func truncateValue(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen-3] + "..."
}

// EnvActionsMenu creates the environment actions menu
func EnvActionsMenu() []DevToolsMenuItem {
	return []DevToolsMenuItem{
		{
			Number:      "1",
			Title:       "View Environment Variables",
			Description: "Display all environment variables",
		},
		{
			Number:      "2",
			Title:       "Export as Template",
			Description: "Save environment configuration as template",
		},
		{
			Number:      "3",
			Title:       "Copy to New Pod",
			Description: "Apply environment to a new pod",
		},
		{
			Number:      "4",
			Title:       "Edit Environment",
			Description: "Modify environment variables",
		},
		{
			Number:      "5",
			Title:       "Link Secret",
			Description: "Add environment variables from secret",
		},
		{
			Number:      "6",
			Title:       "Link ConfigMap",
			Description: "Add environment variables from configmap",
		},
		{
			Number:      "0",
			Title:       "Back to Pod Actions",
			Description: "Return to pod actions menu",
		},
	}
}

// ApplyEnvTemplate applies an environment template to a deployment
func ApplyEnvTemplate(client *k8s.Client, namespace string, deploymentName string, template *EnvTemplate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the deployment
	deployment, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Apply to all containers in the pod template
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]

		// Add environment variables
		for _, envVar := range template.EnvVars {
			switch envVar.Source {
			case "direct":
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  envVar.Name,
					Value: envVar.Value,
				})
			case "secret":
				// Parse secret reference
				parts := strings.Split(envVar.Value, "/")
				if len(parts) == 2 {
					secretName := strings.TrimPrefix(parts[0], "secret:")
					key := parts[1]
					container.Env = append(container.Env, corev1.EnvVar{
						Name: envVar.Name,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secretName,
								},
								Key: key,
							},
						},
					})
				}
			case "configmap":
				// Parse configmap reference
				parts := strings.Split(envVar.Value, "/")
				if len(parts) == 2 {
					configMapName := strings.TrimPrefix(parts[0], "configmap:")
					key := parts[1]
					container.Env = append(container.Env, corev1.EnvVar{
						Name: envVar.Name,
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: configMapName,
								},
								Key: key,
							},
						},
					})
				}
			}
		}

		// Add envFrom for complete secrets/configmaps
		for _, secretName := range template.Secrets {
			container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
			})
		}

		for _, configMapName := range template.ConfigMaps {
			container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			})
		}
	}

	// Update the deployment
	_, err = client.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

// ListEnvTemplates lists saved environment templates
func ListEnvTemplates() ([]string, error) {
	// This would typically read from a ConfigMap or local storage
	// For now, return a placeholder
	return []string{
		"production-env",
		"staging-env",
		"development-env",
	}, nil
}