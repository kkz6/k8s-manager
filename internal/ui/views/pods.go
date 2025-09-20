package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodsOptions contains options for the pods view
type PodsOptions struct {
	AllNamespaces bool
	Selector      string
	ShowAll       bool
}

// PodsViewModel represents the pods list view
type PodsViewModel struct {
	options  PodsOptions
	client   *services.K8sClient
	pods     []corev1.Pod
	menu     *components.Menu
	loading  bool
	spinner  components.SpinnerModel
	err      error
	quitting bool
}

// podsFetchedMsg is sent when pods are fetched
type podsFetchedMsg struct {
	pods []corev1.Pod
	err  error
}

// ShowPodsView shows the interactive pods view
func ShowPodsView(options PodsOptions) error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	model := &PodsViewModel{
		options: options,
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading pods..."),
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// Init initializes the model
func (m *PodsViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Init(),
		m.fetchPods,
	)
}

// Update handles messages
func (m *PodsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, m.fetchPods
		case "n":
			// Namespace switcher
			return m, m.showNamespaceSwitcher
		case "?", "h":
			// Show help
			return m, m.showHelp
		}

	case tea.WindowSizeMsg:
		// Handle resize

	case podsFetchedMsg:
		m.loading = false
		m.pods = msg.pods
		m.err = msg.err
		if m.err == nil {
			m.updateMenu()
		}
		return m, nil

	case showPodActionsMsg:
		// Switch to pod actions view
		return ShowPodActionsView(msg.namespace, msg.name)

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update menu
	if !m.loading && m.menu != nil {
		var cmd tea.Cmd
		// Check if enter was pressed
		if kMsg, ok := msg.(tea.KeyMsg); ok && (kMsg.String() == "enter" || kMsg.String() == " ") {
			selected := m.menu.GetSelected()
			if selected != nil && selected.ID != "back" {
				// Extract namespace and name from ID (format: "namespace/name")
				parts := strings.Split(selected.ID, "/")
				if len(parts) == 2 {
					return m, func() tea.Msg {
						return showPodActionsMsg{
							namespace: parts[0],
							name:      parts[1],
						}
					}
				}
			} else if selected != nil && selected.ID == "back" {
				m.quitting = true
				return m, tea.Quit
			}
		}
		
		newMenu, cmd := m.menu.Update(msg)
		if menu, ok := newMenu.(components.Menu); ok {
			m.menu = &menu
		}
		return m, cmd
	}

	return m, nil
}

// View renders the view
func (m *PodsViewModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		loadingView := components.NewLoadingScreen("Loading Pods")
		return loadingView.View()
	}

	if m.err != nil {
		return components.BoxStyle.Render(
			components.RenderTitle("Pods", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render("Press 'r' to retry, 'q' to quit"),
		)
	}

	// Show menu
	if m.menu == nil {
		return "No pods available"
	}
	
	return m.menu.View()
}

// fetchPods fetches the pods list
func (m *PodsViewModel) fetchPods() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := ""
	if !m.options.AllNamespaces {
		namespace = services.GetCurrentNamespace()
	}

	listOptions := metav1.ListOptions{}
	if m.options.Selector != "" {
		listOptions.LabelSelector = m.options.Selector
	}

	pods, err := m.client.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return podsFetchedMsg{err: err}
	}

	return podsFetchedMsg{pods: pods.Items}
}

// updateMenu updates the menu with pod data
func (m *PodsViewModel) updateMenu() {
	menuItems := []components.MenuItem{}
	
	// Create menu items for each pod
	for _, pod := range m.pods {
		ready := services.GetPodReadyCount(&pod)
		age := services.FormatAge(pod.CreationTimestamp.Time)
		status := services.GetPodStatus(&pod)
		
		// Create a formatted title with pod info
		title := fmt.Sprintf("%s", pod.Name)
		description := fmt.Sprintf("Status: %s, Ready: %s, Age: %s", status, ready, age)
		
		// Add status icon based on pod status
		icon := "⚪"
		switch status {
		case "Running":
			icon = "🟢"
		case "Pending":
			icon = "🟡"
		case "Failed", "Error", "CrashLoopBackOff":
			icon = "🔴"
		case "Completed":
			icon = "✅"
		}
		
		menuItems = append(menuItems, components.MenuItem{
			ID:          fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
			Title:       title,
			Description: description,
			Icon:        icon,
		})
	}
	
	// Add back option
	menuItems = append(menuItems, components.MenuItem{
		ID:          "back",
		Title:       "Back to Main Menu",
		Description: "Return to the main menu",
		Icon:        "⬅️",
	})
	
	// Create title based on namespace
	title := fmt.Sprintf("📦 Pods (%d items)", len(m.pods))
	namespace := services.GetCurrentNamespace()
	if m.options.AllNamespaces {
		title += " - All Namespaces"
	} else {
		title += fmt.Sprintf(" - Namespace: %s", namespace)
	}
	
	// Create menu with DevTools style
	m.menu = components.NewDevToolsMenu(title, menuItems)
}

// showNamespaceSwitcher shows the namespace switcher
func (m *PodsViewModel) showNamespaceSwitcher() tea.Msg {
	// TODO: Implement namespace switcher
	return nil
}

// showHelp shows the help dialog
func (m *PodsViewModel) showHelp() tea.Msg {
	// TODO: Implement help dialog
	return nil
}

// showPodActionsMsg is sent when a pod is selected
type showPodActionsMsg struct {
	namespace string
	name      string
}