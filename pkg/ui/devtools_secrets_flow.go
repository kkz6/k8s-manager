package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
)

// showDevToolsSecrets shows the secrets management interface
func showDevToolsSecrets() error {
	// Show namespace selection menu
	namespaceMenu := NewDevToolsMenu("🔒 Select Namespace for Secrets", []DevToolsMenuItem{
		{
			Number:      "1",
			Title:       "Current Namespace",
			Description: "Use the current context's namespace",
		},
		{
			Number:      "2",
			Title:       "All Namespaces",
			Description: "Show secrets from all namespaces",
		},
		{
			Number:      "3",
			Title:       "Specific Namespace",
			Description: "Select a specific namespace",
		},
		{
			Number:      "0",
			Title:       "Back to Main Menu",
			Description: "Return to the main menu",
		},
	})

	p := tea.NewProgram(namespaceMenu, tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return err
	}

	menu, ok := model.(*DevToolsMenu)
	if !ok || menu.quitting || menu.selected < 0 {
		return nil
	}

	namespace := ""
	allNamespaces := false

	switch menu.selected {
	case 0: // Current namespace
		// Use default
	case 1: // All namespaces
		allNamespaces = true
	case 2: // Specific namespace
		// Show namespace selector
		nsModel := NewDevToolsNamespaceModel()
		nsProgram := tea.NewProgram(nsModel, tea.WithAltScreen())

		nsResult, err := nsProgram.Run()
		if err != nil {
			return err
		}

		if nsM, ok := nsResult.(*DevToolsNamespaceModel); ok {
			namespace = nsM.GetSelectedNamespace()
			if namespace == "" {
				return nil // User cancelled
			}
		} else {
			return nil
		}
	case 3: // Back
		return nil
	default:
		return nil
	}

	// Show secrets interface
	for {
		secretsModel := NewDevToolsSecretsModel(namespace, allNamespaces)
		secretsProgram := tea.NewProgram(secretsModel, tea.WithAltScreen())

		result, err := secretsProgram.Run()
		if err != nil {
			return err
		}

		// Check if a secret was selected
		if model, ok := result.(*DevToolsSecretsModel); ok {
			if model.IsCreateSelected() {
				// Handle secret creation
				ns := namespace
				if ns == "" && !allNamespaces {
					// Get the current namespace from client
					if client, err := k8s.NewClient(); err == nil {
						ns = client.GetNamespace()
					}
				}
				if err := ShowSecretCreator(ns); err != nil {
					fmt.Printf("Error creating secret: %v\n", err)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()
				}
				continue
			}

			selectedSecret := model.GetSelectedSecret()
			if selectedSecret != nil {
				// Show secret actions
				if err := showDevToolsSecretActions(*selectedSecret); err != nil {
					fmt.Printf("Error: %v\n", err)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()
					continue
				}

				// After actions, show continuation menu
				continueMenu := NewDevToolsMenu("✅ Action Completed", []DevToolsMenuItem{
					{
						Number:      "1",
						Title:       "Return to Secrets List",
						Description: "Go back to the secrets list",
					},
					{
						Number:      "2",
						Title:       "Return to Main Menu",
						Description: "Go back to the main menu",
					},
				})

				contProgram := tea.NewProgram(continueMenu, tea.WithAltScreen())
				contResult, _ := contProgram.Run()

				if contMenu, ok := contResult.(*DevToolsMenu); ok {
					if contMenu.selected == 1 { // Return to main menu
						return nil
					}
					// Otherwise continue to show secrets list
				}
			} else {
				// No selection, go back
				return nil
			}
		} else {
			return nil
		}
	}
}

// showDevToolsSecretActions shows actions for a specific secret
func showDevToolsSecretActions(secret SecretInfo) error {
	actionsMenu := NewDevToolsMenu(
		fmt.Sprintf("🔒 Secret Actions: %s", secret.Name),
		SecretActionsMenu(secret),
	)

	p := tea.NewProgram(actionsMenu, tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return err
	}

	if menu, ok := model.(*DevToolsMenu); ok && menu.selected >= 0 {
		switch menu.selected {
		case 0: // View Secret Data
			fmt.Print(ViewSecretData(secret.Secret))
			fmt.Println("\n\nPress Enter to continue...")
			fmt.Scanln()

		case 1: // Edit Secret
			// Use the new secret editor
			if secret.Secret != nil {
				editor := NewSecretEditorModel(secret.Secret, nil)
				p := tea.NewProgram(editor, tea.WithAltScreen())
				_, err := p.Run()
				if err != nil {
					fmt.Printf("Error editing secret: %v\n", err)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()
				}
			}

		case 2: // Copy Secret
			fmt.Print("\033[H\033[2J")
			fmt.Println("📋 Copy Secret")
			fmt.Println("\nFeature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case 3: // Export as YAML
			fmt.Print("\033[H\033[2J")
			fmt.Println("📄 Export as YAML")
			fmt.Println("\nFeature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case 4: // Export as ENV
			fmt.Print("\033[H\033[2J")
			fmt.Println("📝 Environment Variables Export")
			fmt.Println("\n" + ExportSecretAsEnv(secret.Secret))
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case 5: // Delete Secret
			fmt.Print("\033[H\033[2J")
			fmt.Println("🗑️ Delete Secret")
			fmt.Printf("\nAre you sure you want to delete secret '%s'? (y/N): ", secret.Name)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm == "y" || confirm == "Y" {
				// Delete logic would go here
				fmt.Println("Secret deletion feature coming soon!")
			}
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case 6: // Back
			return nil
		}
	}

	return nil
}