package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/karthickk/k8s-manager/pkg/utils"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage Kubernetes secrets",
		Long:  `Create, view, update, and delete Kubernetes secrets in your cluster.`,
	}

	cmd.AddCommand(newSecretsListCmd())
	cmd.AddCommand(newSecretsGetCmd())
	cmd.AddCommand(newSecretsCreateCmd())
	cmd.AddCommand(newSecretsUpdateCmd())
	cmd.AddCommand(newSecretsDeleteCmd())
	cmd.AddCommand(newSecretsDecodeCmd())

	return cmd
}

func newSecretsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all secrets in the namespace",
		Long:  `List all Kubernetes secrets in the current namespace.`,
		RunE:  runSecretsList,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace to list secrets from (overrides config)")
	cmd.Flags().BoolP("all-namespaces", "A", false, "List secrets from all namespaces")

	return cmd
}

func newSecretsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <secret-name>",
		Short: "Get details of a specific secret",
		Long:  `Get detailed information about a specific Kubernetes secret.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSecretsGet,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the secret (overrides config)")
	cmd.Flags().BoolP("decode", "d", false, "Decode base64 values")

	return cmd
}

func newSecretsCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <secret-name>",
		Short: "Create a new secret",
		Long:  `Create a new Kubernetes secret with key-value pairs.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSecretsCreate,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace to create the secret in (overrides config)")
	cmd.Flags().StringSliceP("from-literal", "l", []string{}, "Key-value pairs (key=value)")
	cmd.Flags().StringSliceP("from-file", "f", []string{}, "Files to include in secret")
	cmd.Flags().StringP("type", "t", "Opaque", "Secret type")

	return cmd
}

func newSecretsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <secret-name>",
		Short: "Update an existing secret",
		Long:  `Update an existing Kubernetes secret with new key-value pairs.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSecretsUpdate,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the secret (overrides config)")
	cmd.Flags().StringSliceP("from-literal", "l", []string{}, "Key-value pairs to add/update (key=value)")
	cmd.Flags().StringSliceP("from-file", "f", []string{}, "Files to add/update in secret")
	cmd.Flags().StringSliceP("remove-key", "r", []string{}, "Keys to remove from secret")

	return cmd
}

func newSecretsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <secret-name>",
		Short: "Delete a secret",
		Long:  `Delete a Kubernetes secret from the cluster.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSecretsDelete,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the secret (overrides config)")
	cmd.Flags().BoolP("force", "", false, "Skip confirmation prompt")

	return cmd
}

func newSecretsDecodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode <secret-name> <key>",
		Short: "Decode a specific key from a secret",
		Long:  `Decode and display the value of a specific key from a Kubernetes secret.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runSecretsDecode,
	}

	cmd.Flags().StringP("namespace", "n", "", "Namespace of the secret (overrides config)")

	return cmd
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	allNamespaces, _ := cmd.Flags().GetBool("all-namespaces")

	if namespace == "" && !allNamespaces {
		namespace = client.GetNamespace()
	}

	ctx := cmd.Context()
	var secrets *corev1.SecretList

	if allNamespaces {
		secrets, err = client.Clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list secrets: %w", err)
		}
	} else {
		secrets, err = client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list secrets in namespace %s: %w", namespace, err)
		}
	}

	if len(secrets.Items) == 0 {
		if allNamespaces {
			fmt.Println("No secrets found in any namespace")
		} else {
			fmt.Printf("No secrets found in namespace '%s'\n", namespace)
		}
		return nil
	}

	// Display secrets in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if allNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tTYPE\tDATA\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tTYPE\tDATA\tAGE")
	}

	for _, secret := range secrets.Items {
		age := utils.FormatAge(secret.CreationTimestamp.Time)
		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				secret.Namespace, secret.Name, secret.Type, len(secret.Data), age)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
				secret.Name, secret.Type, len(secret.Data), age)
		}
	}
	w.Flush()

	return nil
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	secretName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	decode, _ := cmd.Flags().GetBool("decode")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	ctx := cmd.Context()
	secret, err := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	fmt.Printf("Name:         %s\n", secret.Name)
	fmt.Printf("Namespace:    %s\n", secret.Namespace)
	fmt.Printf("Type:         %s\n", secret.Type)
	fmt.Printf("Created:      %s\n", secret.CreationTimestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Data Keys:    %d\n", len(secret.Data))
	fmt.Println()

	if len(secret.Data) > 0 {
		fmt.Println("Data:")
		for key, value := range secret.Data {
			if decode {
				decoded, err := base64.StdEncoding.DecodeString(string(value))
				if err != nil {
					fmt.Printf("  %s: <failed to decode>\n", key)
				} else {
					fmt.Printf("  %s: %s\n", key, string(decoded))
				}
			} else {
				fmt.Printf("  %s: <base64 encoded, %d bytes>\n", key, len(value))
			}
		}
	}

	return nil
}

func runSecretsCreate(cmd *cobra.Command, args []string) error {
	secretName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	fromLiteral, _ := cmd.Flags().GetStringSlice("from-literal")
	fromFile, _ := cmd.Flags().GetStringSlice("from-file")
	secretType, _ := cmd.Flags().GetString("type")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	if len(fromLiteral) == 0 && len(fromFile) == 0 {
		return fmt.Errorf("must specify either --from-literal or --from-file")
	}

	secretData := make(map[string][]byte)

	// Process literal values
	for _, literal := range fromLiteral {
		parts := strings.SplitN(literal, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid literal format: %s (expected key=value)", literal)
		}
		secretData[parts[0]] = []byte(parts[1])
	}

	// Process files
	for _, filePath := range fromFile {
		parts := strings.SplitN(filePath, "=", 2)
		var key, path string

		if len(parts) == 2 {
			key = parts[0]
			path = parts[1]
		} else {
			// Use filename as key
			path = parts[0]
			key = filepath.Base(path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		secretData[key] = data
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretType(secretType),
		Data: secretData,
	}

	ctx := cmd.Context()
	_, err = client.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret %s: %w", secretName, err)
	}

	fmt.Printf("✅ Secret '%s' created successfully in namespace '%s'\n", secretName, namespace)
	return nil
}

func runSecretsUpdate(cmd *cobra.Command, args []string) error {
	secretName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	fromLiteral, _ := cmd.Flags().GetStringSlice("from-literal")
	fromFile, _ := cmd.Flags().GetStringSlice("from-file")
	removeKeys, _ := cmd.Flags().GetStringSlice("remove-key")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	ctx := cmd.Context()
	secret, err := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	// Process literal values
	for _, literal := range fromLiteral {
		parts := strings.SplitN(literal, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid literal format: %s (expected key=value)", literal)
		}
		secret.Data[parts[0]] = []byte(parts[1])
	}

	// Process files
	for _, filePath := range fromFile {
		parts := strings.SplitN(filePath, "=", 2)
		var key, path string

		if len(parts) == 2 {
			key = parts[0]
			path = parts[1]
		} else {
			path = parts[0]
			key = filepath.Base(path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		secret.Data[key] = data
	}

	// Remove keys
	for _, key := range removeKeys {
		delete(secret.Data, key)
	}

	_, err = client.Clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %s: %w", secretName, err)
	}

	fmt.Printf("✅ Secret '%s' updated successfully in namespace '%s'\n", secretName, namespace)
	return nil
}

func runSecretsDelete(cmd *cobra.Command, args []string) error {
	secretName := args[0]
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	force, _ := cmd.Flags().GetBool("force")

	if namespace == "" {
		namespace = client.GetNamespace()
	}

	if !force {
		fmt.Printf("Are you sure you want to delete secret '%s' in namespace '%s'? (y/N): ", secretName, namespace)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	ctx := cmd.Context()
	err = client.Clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", secretName, err)
	}

	fmt.Printf("✅ Secret '%s' deleted successfully from namespace '%s'\n", secretName, namespace)
	return nil
}

func runSecretsDecode(cmd *cobra.Command, args []string) error {
	secretName := args[0]
	key := args[1]

	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace, _ := cmd.Flags().GetString("namespace")
	if namespace == "" {
		namespace = client.GetNamespace()
	}

	ctx := cmd.Context()
	secret, err := client.Clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	value, exists := secret.Data[key]
	if !exists {
		return fmt.Errorf("key '%s' not found in secret '%s'", key, secretName)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(value))
	if err != nil {
		return fmt.Errorf("failed to decode value for key '%s': %w", key, err)
	}

	fmt.Print(string(decoded))
	return nil
}
