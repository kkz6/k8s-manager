package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/pterm/pterm"
)

// ShowEnhancedPodsInterface shows the enhanced pods interface with better navigation
func ShowEnhancedPodsInterface(namespace string, allNamespaces bool) error {
	for {
		// Show loading spinner first
		spinner, _ := pterm.DefaultSpinner.Start("Initializing K8s Manager...")

		m := NewEnhancedPodsModel(namespace, allNamespaces)
		p := tea.NewProgram(m, tea.WithAltScreen())

		spinner.Stop()

		result, err := p.Run()
		if err != nil {
			return err
		}

		// Check if a pod was selected
		if model, ok := result.(EnhancedPodsModel); ok {
			selectedPod := model.GetSelectedPod()
			if selectedPod != nil {
				// Clear screen and show pod actions
				fmt.Print("\033[H\033[2J")

				// Get K8s client
				client, err := k8s.NewClient()
				if err != nil {
					pterm.Error.Printf("Failed to create Kubernetes client: %v\n", err)
					fmt.Println("\nPress Enter to return to pod list...")
					fmt.Scanln()
					continue
				}

				// Show the enhanced pod actions menu
				actionsModel := NewEnhancedPodActionsModel(*selectedPod, client)
				actionsProgram := tea.NewProgram(actionsModel, tea.WithAltScreen())

				_, err = actionsProgram.Run()
				if err != nil {
					pterm.Error.Printf("Error in pod actions: %v\n", err)
				}

				// After action completes, ask if they want to continue
				fmt.Print("\033[H\033[2J")
				pterm.DefaultHeader.Println("Action Completed")

				continuePrompt := pterm.DefaultInteractiveSelect.
					WithOptions([]string{"Return to pod list", "Quit application"}).
					WithDefaultOption("Return to pod list")

				choice, _ := continuePrompt.Show()

				if choice == "Quit application" {
					return nil
				}
				// Continue to show pod list again
			} else {
				// No pod selected, exit
				return nil
			}
		} else {
			return nil
		}
	}
}

// ShowEnhancedMenu shows the main menu with enhanced UI
func ShowEnhancedMenu() error {
	menuItems := []MenuItem{
		{
			Title:       "Pods",
			Description: "Manage Kubernetes pods",
			Icon:        "📦",
			Number:      1,
		},
		{
			Title:       "Deployments",
			Description: "Manage deployments",
			Icon:        "🚀",
			Number:      2,
		},
		{
			Title:       "Services",
			Description: "Manage services",
			Icon:        "🌐",
			Number:      3,
		},
		{
			Title:       "ConfigMaps",
			Description: "Manage config maps",
			Icon:        "⚙️",
			Number:      4,
		},
		{
			Title:       "Secrets",
			Description: "Manage secrets",
			Icon:        "🔒",
			Number:      5,
		},
		{
			Title:       "Namespaces",
			Description: "Manage namespaces",
			Icon:        "🏷️",
			Number:      6,
		},
		{
			Title:       "Nodes",
			Description: "View cluster nodes",
			Icon:        "🖥️",
			Number:      7,
		},
		{
			Title:       "Events",
			Description: "View cluster events",
			Icon:        "📊",
			Number:      8,
		},
		{
			Title:       "Quit",
			Description: "Exit the application",
			Icon:        "❌",
			Number:      9,
		},
	}

	for {
		fmt.Print("\033[H\033[2J") // Clear screen

		// Show title
		pterm.DefaultBigText.WithLetters(
			pterm.NewLettersFromString("K8s Manager"),
		).Render()

		fmt.Println(RenderTitle("🚀 Kubernetes Resource Manager", "Enhanced UI with Number Navigation"))
		fmt.Println()

		// Create menu
		menu := NewMenu(menuItems)
		menu.SetSize(80, 20)

		// Show menu
		fmt.Println(menu.View())
		fmt.Println()
		fmt.Println(FooterStyle.Render("Press 1-9 to select • ↑/↓ to navigate • Enter to confirm • q to quit"))

		// Get user choice
		fmt.Print("\nSelect an option (1-9): ")
		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			// Show namespace selection first
			namespaceChoice := selectNamespaceOption()
			namespace := ""
			allNamespaces := false

			switch namespaceChoice {
			case "current":
				// Use current namespace
			case "all":
				allNamespaces = true
			case "specific":
				fmt.Print("Enter namespace name: ")
				fmt.Scanln(&namespace)
			}

			err := ShowEnhancedPodsInterface(namespace, allNamespaces)
			if err != nil {
				pterm.Error.Printf("Error: %v\n", err)
				fmt.Println("\nPress Enter to continue...")
				fmt.Scanln()
			}

		case "2":
			pterm.Info.Println("Deployments feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "3":
			pterm.Info.Println("Services feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "4":
			pterm.Info.Println("ConfigMaps feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "5":
			pterm.Info.Println("Secrets feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "6":
			pterm.Info.Println("Namespaces feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "7":
			pterm.Info.Println("Nodes feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "8":
			pterm.Info.Println("Events feature coming soon!")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "9", "q", "Q":
			pterm.Success.Println("Thank you for using K8s Manager!")
			return nil

		default:
			pterm.Warning.Printf("Invalid option: %s\n", choice)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
		}
	}
}

func selectNamespaceOption() string {
	options := []string{
		"Current namespace",
		"All namespaces",
		"Specific namespace",
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		WithDefaultText("Select namespace scope").
		Show()

	switch selectedOption {
	case "Current namespace":
		return "current"
	case "All namespaces":
		return "all"
	case "Specific namespace":
		return "specific"
	default:
		return "current"
	}
}