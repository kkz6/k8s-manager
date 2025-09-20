package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/ui/components"
)

// View represents different views in the application
type View string

const (
	ViewMainMenu        View = "main_menu"
	ViewPods            View = "pods"
	ViewPodActions      View = "pod_actions"
	ViewConfigsMenu     View = "configs_menu"
	ViewConfigMaps      View = "configmaps"
	ViewSecrets         View = "secrets"
	ViewConfigMapDetail View = "configmap_detail"
	ViewSecretDetail    View = "secret_detail"
	ViewLogs            View = "logs"
	ViewEnvManager      View = "env_manager"
	ViewAddSecretKey    View = "add_secret_key"
	ViewAddConfigMapKey View = "add_configmap_key"
)

// NavigateMsg is sent to navigate between views
type NavigateMsg struct {
	To     View
	Params map[string]string
}

// AppModel is the main application model that manages all views
type AppModel struct {
	currentView  View
	mainMenu     tea.Model
	currentModel tea.Model
	params       map[string]string
	width        int
	height       int
	quitting     bool
}

// NewAppModel creates a new application model
func NewAppModel() *AppModel {
	mainMenu := createMainMenu()
	return &AppModel{
		currentView:  ViewMainMenu,
		mainMenu:     mainMenu,
		currentModel: mainMenu,
	}
}

// Init initializes the app
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.ClearScreen,
		m.currentModel.Init(),
	)
}

// Update handles all messages and navigation
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
	}

	// Handle navigation
	if nav, ok := msg.(NavigateMsg); ok {
		return m.navigate(nav)
	}

	// Handle quit
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	// Update current model
	newModel, cmd := m.currentModel.Update(msg)
	m.currentModel = newModel
	return m, cmd
}

// View renders the current view
func (m *AppModel) View() string {
	if m.quitting {
		return ""
	}
	return m.currentModel.View()
}

// navigate switches between views
func (m *AppModel) navigate(nav NavigateMsg) (tea.Model, tea.Cmd) {
	m.params = nav.Params

	// Clear screen when navigating
	clearCmd := tea.ClearScreen

	switch nav.To {
	case ViewMainMenu:
		m.currentView = ViewMainMenu
		m.currentModel = m.mainMenu
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewPods:
		m.currentView = ViewPods
		m.currentModel = NewPodsViewModelSimple()
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewPodActions:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		model, cmd := ShowPodActionsView(namespace, name)
		m.currentView = ViewPodActions
		m.currentModel = model
		return m, tea.Batch(clearCmd, cmd)

	case ViewConfigsMenu:
		m.currentView = ViewConfigsMenu
		m.currentModel = NewConfigsMenuModelSimple()
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewConfigMaps:
		m.currentView = ViewConfigMaps
		m.currentModel = NewConfigMapsViewModelSimple()
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewSecrets:
		m.currentView = ViewSecrets
		m.currentModel = NewSecretsViewModelSimple()
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewConfigMapDetail:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		m.currentView = ViewConfigMapDetail
		m.currentModel = NewConfigMapDetailsModel(namespace, name)
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewSecretDetail:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		m.currentView = ViewSecretDetail
		m.currentModel = NewSecretDetailsModel(namespace, name)
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewLogs:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		container := nav.Params["container"]
		follow := nav.Params["follow"] == "true"
		m.currentView = ViewLogs
		m.currentModel = NewLogsViewModel(namespace, name, container, follow)
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewEnvManager:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		m.currentView = ViewEnvManager
		m.currentModel = NewEnvManagerModel(namespace, name)
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewAddSecretKey:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		m.currentView = ViewAddSecretKey
		m.currentModel = NewAddSecretKeyModel(namespace, name)
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	case ViewAddConfigMapKey:
		namespace := nav.Params["namespace"]
		name := nav.Params["name"]
		m.currentView = ViewAddConfigMapKey
		m.currentModel = NewAddConfigMapKeyModel(namespace, name)
		return m, tea.Batch(clearCmd, m.currentModel.Init())

	default:
		return m, nil
	}
}

// Navigate creates a navigation command
func Navigate(to View, params map[string]string) tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{To: to, Params: params}
	}
}

// createMainMenu creates the main menu
func createMainMenu() tea.Model {
	menuItems := []components.MenuItem{
		{
			ID:          "pods",
			Title:       "Pods Manager",
			Description: "List, manage, and interact with Kubernetes pods",
			Icon:        "📦",
			Shortcut:    "p",
		},
		{
			ID:          "deployments",
			Title:       "Deployments",
			Description: "Manage Kubernetes deployments and rollouts",
			Icon:        "🚀",
			Shortcut:    "d",
		},
		{
			ID:          "services",
			Title:       "Services",
			Description: "View and manage Kubernetes services",
			Icon:        "🌐",
			Shortcut:    "s",
		},
		{
			ID:          "secrets",
			Title:       "ConfigMaps & Secrets",
			Description: "Manage configuration and secret resources",
			Icon:        "🔐",
			Shortcut:    "c",
		},
		{
			ID:          "namespaces",
			Title:       "Namespaces",
			Description: "Switch and manage Kubernetes namespaces",
			Icon:        "📁",
			Shortcut:    "n",
		},
		{
			ID:          "nodes",
			Title:       "Nodes & Cluster",
			Description: "View cluster nodes and resource usage",
			Icon:        "🖥️",
			Shortcut:    "o",
		},
		{
			ID:          "logs",
			Title:       "Logs & Events",
			Description: "View pod logs and cluster events",
			Icon:        "📊",
			Shortcut:    "l",
		},
		{
			ID:          "config",
			Title:       "Configuration",
			Description: "Manage K8s Manager settings and contexts",
			Icon:        "⚙️",
			Shortcut:    "g",
		},
		{
			ID:          "quit",
			Title:       "Exit",
			Description: "Quit the application",
			Icon:        "🚪",
			Shortcut:    "q",
		},
	}

	menu := components.NewDevToolsMenu("🚀 K8s Manager by Karthick", menuItems)
	return &MainMenuModelSimple{
		menu: *menu,
	}
}

// MainMenuModelSimple is a simple main menu that works with navigation
type MainMenuModelSimple struct {
	menu     components.Menu
	quitting bool
}

func (m *MainMenuModelSimple) Init() tea.Cmd {
	return m.menu.Init()
}

func (m *MainMenuModelSimple) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "p":
			return m, Navigate(ViewPods, nil)
		case "c":
			return m, Navigate(ViewConfigsMenu, nil)
		}
	}

	// Update menu
	newMenu, cmd := m.menu.Update(msg)
	if menu, ok := newMenu.(components.Menu); ok {
		m.menu = menu
		// Check if an item was selected
		if kMsg, ok := msg.(tea.KeyMsg); ok && (kMsg.String() == "enter" || kMsg.String() == " ") {
			selected := m.menu.GetSelected()
			if selected != nil {
				switch selected.ID {
				case "pods":
					return m, Navigate(ViewPods, nil)
				case "secrets":
					return m, Navigate(ViewConfigsMenu, nil)
				case "quit":
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}

	return m, cmd
}

func (m *MainMenuModelSimple) View() string {
	if m.quitting {
		return ""
	}
	return m.menu.View()
}

// RunApp starts the application with unified navigation
func RunApp() error {
	// Show splash screen
	logo := components.NewLogo("K8S MANAGER")
	logo.Subtitle = "Your Kubernetes cluster management toolkit"
	logo.Author = "Karthick"

	splash := components.NewSplashScreen(logo, 1500*time.Millisecond)
	if p := tea.NewProgram(splash, tea.WithAltScreen()); p != nil {
		p.Run()
	}

	// Start main app
	app := NewAppModel()
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running application: %w", err)
	}
	return nil
}