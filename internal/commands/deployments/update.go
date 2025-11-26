package deployments

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

var (
	image         string
	containerName string
	setEnv        []string
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update DEPLOYMENT",
		Short: "Update deployment image or configuration",
		Long: `Update a deployment's container image or environment variables.

This command allows you to:
- Update container images to new versions
- Set environment variables
- Trigger a rollout with the new configuration`,
		Example: `  # Update deployment image
  k8s-manager deployments update my-app --image=myapp:v2

  # Update specific container image
  k8s-manager deployments update my-app --container=app --image=myapp:v2

  # Update with environment variable
  k8s-manager deployments update my-app --set-env=DEBUG=true

  # Update image and set multiple environment variables
  k8s-manager deployments update my-app --image=myapp:v2 --set-env=DEBUG=true --set-env=LOG_LEVEL=info`,
		Args: cobra.ExactArgs(1),
		RunE: runUpdate,
	}

	cmd.Flags().StringVar(&image, "image", "", "New container image")
	cmd.Flags().StringVar(&containerName, "container", "", "Container name (defaults to first container)")
	cmd.Flags().StringArrayVar(&setEnv, "set-env", []string{}, "Set environment variable (KEY=VALUE)")

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	deploymentName := args[0]

	if image == "" && len(setEnv) == 0 {
		return fmt.Errorf("at least one of --image or --set-env must be specified")
	}

	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	namespace := services.GetNamespace(cmd)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the deployment
	deployment, err := client.Clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update container image if specified
	if image != "" {
		found := false
		for i := range deployment.Spec.Template.Spec.Containers {
			container := &deployment.Spec.Template.Spec.Containers[i]

			// If container name is specified, only update that container
			if containerName != "" {
				if container.Name == containerName {
					fmt.Printf("Updating container '%s' image from '%s' to '%s'\n", container.Name, container.Image, image)
					container.Image = image
					found = true
					break
				}
			} else {
				// Update the first container
				fmt.Printf("Updating container '%s' image from '%s' to '%s'\n", container.Name, container.Image, image)
				container.Image = image
				found = true
				break
			}
		}

		if !found {
			if containerName != "" {
				return fmt.Errorf("container '%s' not found in deployment", containerName)
			}
			return fmt.Errorf("no containers found in deployment")
		}
	}

	// Update environment variables if specified
	if len(setEnv) > 0 {
		for i := range deployment.Spec.Template.Spec.Containers {
			container := &deployment.Spec.Template.Spec.Containers[i]

			// Skip if container name is specified and doesn't match
			if containerName != "" && container.Name != containerName {
				continue
			}

			for _, env := range setEnv {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", env)
				}

				key := parts[0]
				value := parts[1]

				// Check if env var already exists
				found := false
				for j := range container.Env {
					if container.Env[j].Name == key {
						fmt.Printf("Updating environment variable '%s' in container '%s'\n", key, container.Name)
						container.Env[j].Value = value
						found = true
						break
					}
				}

				if !found {
					fmt.Printf("Adding environment variable '%s' to container '%s'\n", key, container.Name)
					container.Env = append(container.Env, corev1.EnvVar{
						Name:  key,
						Value: value,
					})
				}
			}

			// If container name was specified, we're done
			if containerName != "" {
				break
			}
		}
	}

	// Update the deployment
	fmt.Printf("Updating deployment '%s' in namespace '%s'...\n", deploymentName, namespace)
	_, err = client.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	fmt.Printf("✓ Deployment '%s' updated successfully\n", deploymentName)
	fmt.Println("Rollout has been triggered. Use 'kubectl rollout status' to monitor progress.")

	return nil
}
