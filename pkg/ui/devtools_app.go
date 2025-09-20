package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
)

// ShowDevToolsInterface shows the DevTools-style interface
func ShowDevToolsInterface() error {
	for {
		// Clear screen before showing menu
		fmt.Print("\033[H\033[2J")

		// Show main menu
		mainMenu := K8sManagerMenu()
		p := tea.NewProgram(mainMenu, tea.WithAltScreen())

		model, err := p.Run()
		if err != nil {
			return err
		}

		// Check what was selected
		if menu, ok := model.(*DevToolsMenu); ok {
			if menu.quitting {
				return nil
			}

			if menu.selected < 0 || menu.selected >= len(menu.items) {
				continue // No selection, show menu again
			}

			// Clear screen before action
			fmt.Print("\033[H\033[2J")

			// Handle the selection
			switch menu.selected {
			case 0: // Pods Manager
				if err := showDevToolsPods(); err != nil {
					fmt.Printf("Error: %v\n", err)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()
				}

			case 1: // Deployments
				fmt.Println("\n📦 Deployments feature coming soon!")
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()

			case 2: // Services
				fmt.Println("\n🌐 Services feature coming soon!")
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()

			case 3: // ConfigMaps & Secrets
				if err := showDevToolsSecrets(); err != nil {
					fmt.Printf("Error: %v\n", err)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()
				}

			case 4: // Namespaces
				fmt.Println("\n🏷️ Namespaces feature coming soon!")
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()

			case 5: // Nodes & Cluster
				fmt.Println("\n🖥️ Nodes & Cluster feature coming soon!")
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()

			case 6: // Logs & Events
				fmt.Println("\n📊 Logs & Events feature coming soon!")
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()

			case 7: // Configuration
				fmt.Println("\n⚙️ Configuration feature coming soon!")
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()

			case 8: // Exit
				fmt.Println("👋 Goodbye!")
				return nil

			default:
				continue
			}
		} else {
			return nil
		}
	}
}

func showDevToolsPods() error {
	// Show namespace selection menu in DevTools style
	fmt.Print("\033[H\033[2J") // Clear screen before namespace menu

	namespaceMenu := NewDevToolsMenu("📦 Namespace Selection", []DevToolsMenuItem{
		{
			Number:      "1",
			Title:       "Current Namespace",
			Description: "Use the current context's namespace",
		},
		{
			Number:      "2",
			Title:       "All Namespaces",
			Description: "Show pods from all namespaces",
		},
		{
			Number:      "3",
			Title:       "Specific Namespace",
			Description: "Enter a specific namespace name",
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
	if !ok {
		return nil
	}

	if menu.quitting || menu.selected < 0 {
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
		// Show namespace selector with instant number selection
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

	// Show pods interface
	for {
		podsModel := NewDevToolsPodsModel(namespace, allNamespaces)
		p := tea.NewProgram(podsModel, tea.WithAltScreen())

		result, err := p.Run()
		if err != nil {
			return err
		}

		// Check if a pod was selected
		if model, ok := result.(*DevToolsPodsModel); ok {
			selectedPod := model.GetSelectedPod()
			if selectedPod != nil {
				// Clear screen and show loading
				fmt.Print("\033[H\033[2J")
				fmt.Printf("Loading actions for pod %s...\n", selectedPod.Name)

				// Show pod actions
				if err := showDevToolsPodActions(*selectedPod); err != nil {
					fmt.Printf("Error: %v\n", err)
					fmt.Println("\nPress Enter to continue...")
					fmt.Scanln()
					continue
				}

				// After actions, show continuation menu
				continueMenu := NewDevToolsMenu("✅ Action Completed", []DevToolsMenuItem{
					{
						Number:      "1",
						Title:       "Return to Pods List",
						Description: "Go back to the pods list",
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
					// Otherwise continue to show pods list
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

func showDevToolsPodActions(pod PodInfo) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	// Create actions menu - consistent with main menu style
	actions := []DevToolsMenuItem{
		{
			Number:      "1",
			Title:       "View Logs",
			Description: "Show recent pod logs (last 100 lines)",
		},
		{
			Number:      "2",
			Title:       "Follow Logs",
			Description: "Stream logs in real-time",
		},
		{
			Number:      "3",
			Title:       "Execute Shell",
			Description: "Open interactive shell in pod",
		},
		{
			Number:      "4",
			Title:       "Describe Pod",
			Description: "Show detailed pod information and events",
		},
		{
			Number:      "5",
			Title:       "Port Forward",
			Description: "Forward local port to pod port",
		},
		{
			Number:      "6",
			Title:       "Restart Pod",
			Description: "Delete pod to trigger restart",
		},
		{
			Number:      "7",
			Title:       "Delete Pod",
			Description: "Permanently delete the pod",
		},
		{
			Number:      "8",
			Title:       "Environment Variables",
			Description: "View and manage environment variables",
		},
		{
			Number:      "9",
			Title:       "Resource Usage",
			Description: "Show CPU and memory metrics",
		},
		{
			Number:      "0",
			Title:       "Back to Pods",
			Description: "Return to pods list",
		},
	}

	actionsMenu := NewDevToolsMenu(fmt.Sprintf("🔧 Pod Actions: %s", pod.Name), actions)
	p := tea.NewProgram(actionsMenu, tea.WithAltScreen())

	model, err := p.Run()
	if err != nil {
		return err
	}

	if menu, ok := model.(*DevToolsMenu); ok {
		if menu.selected >= 0 && menu.selected < len(actions) {
			// Create enhanced model for actual action execution
			enhancedModel := NewEnhancedPodActionsModel(pod, client)

			// Map selection to action
			switch menu.selected {
			case 0: // View Logs
				msg := enhancedModel.viewLogs()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 1: // Follow Logs
				msg := enhancedModel.followLogs()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 2: // Execute Shell
				msg := enhancedModel.execShell()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 3: // Describe Pod
				msg := enhancedModel.describePod()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 4: // Port Forward
				msg := enhancedModel.portForward()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 5: // Restart Pod
				msg := enhancedModel.restartPod()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 6: // Delete Pod
				msg := enhancedModel.deletePod()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 7: // Environment Variables
				// Show environment variable management menu
				envMenu := NewDevToolsMenu(fmt.Sprintf("🔧 Environment Variables: %s", pod.Name), []DevToolsMenuItem{
					{
						Number:      "1",
						Title:       "View Current Environment",
						Description: "Show current environment variables",
					},
					{
						Number:      "2",
						Title:       "Assign Environment Variables",
						Description: "Add environment variables from secrets/configmaps",
					},
					{
						Number:      "0",
						Title:       "Back",
						Description: "Return to pod actions",
					},
				})

				envP := tea.NewProgram(envMenu, tea.WithAltScreen())
				envModel, err := envP.Run()
				if err != nil {
					return err
				}

				if envM, ok := envModel.(*DevToolsMenu); ok && envM.selected >= 0 {
					switch envM.selected {
					case 0: // View Current Environment
						fmt.Print(ViewPodEnvVars(pod.Pod, client))
						fmt.Println("\n\nPress Enter to continue...")
						fmt.Scanln()
					case 1: // Assign Environment Variables
						return ShowPodEnvAssignment(pod.Pod, client)
					}
				}
				return nil
			case 8: // Resource Usage
				msg := enhancedModel.resourceUsage()
				if errMsg, ok := msg.(actionResultMsg); ok && errMsg.err != nil {
					return errMsg.err
				}
				return nil
			case 9: // Back to Pods
				return nil
			}
		}
	}

	return nil
}