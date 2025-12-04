package views

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespacesViewModel displays and manages Kubernetes namespaces
type NamespacesViewModel struct {
	client       *services.K8sClient
	namespaces   []corev1.Namespace
	list         *components.ListView
	loading      bool
	spinner      components.SpinnerModel
	err          error
	currentNs    string
	message      string
	messageType  string
}

// namespacesFetchedMsg is sent when namespaces are fetched
type namespacesFetchedMsg struct {
	namespaces []corev1.Namespace
	err        error
}

// namespaceSwitchedMsg is sent when namespace is switched
type namespaceSwitchedMsg struct {
	namespace string
}

// NewNamespacesViewModel creates a new namespaces view model
func NewNamespacesViewModel() tea.Model {
	client, _ := services.GetK8sClient()
	return &NamespacesViewModel{
		client:    client,
		loading:   true,
		spinner:   components.NewSpinner("Loading namespaces..."),
		currentNs: services.GetCurrentNamespace(),
	}
}

func (m *NamespacesViewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.fetchNamespaces)
}

func (m *NamespacesViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b", "0":
			return m, Navigate(ViewMainMenu, nil)
		case "r":
			m.loading = true
			m.err = nil
			m.message = ""
			return m, m.fetchNamespaces
		}

	case namespacesFetchedMsg:
		m.loading = false
		m.namespaces = msg.namespaces
		m.err = msg.err
		if m.err == nil {
			m.updateList()
		}
		return m, nil

	case namespaceSwitchedMsg:
		m.currentNs = msg.namespace
		m.message = fmt.Sprintf("Switched to namespace: %s", msg.namespace)
		m.messageType = "success"
		m.updateList()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if !m.loading && m.list != nil {
		if kMsg, ok := msg.(tea.KeyMsg); ok && (kMsg.String() == "enter" || kMsg.String() == " ") {
			selected := m.list.GetSelected()
			if selected != nil {
				ns := selected.Data.(*corev1.Namespace)
				return m, m.switchNamespace(ns.Name)
			}
		}

		newList, cmd := m.list.Update(msg)
		if list, ok := newList.(components.ListView); ok {
			m.list = &list
		}
		return m, cmd
	}

	return m, nil
}

func (m *NamespacesViewModel) View() string {
	if m.loading {
		return components.NewLoadingScreen("Loading Namespaces").View()
	}

	if m.err != nil {
		helpText := "Press 'r' to retry, 'q/b/esc' to go back, 'ctrl+c' to quit"

		if services.IsAuthError(m.err) {
			authHelp := services.GetAuthHelp(m.err.Error())
			return components.BoxStyle.Render(
				components.RenderTitle("Authentication Required", "") + "\n\n" +
					components.RenderMessage("error", m.err.Error()) + "\n\n" +
					components.InfoMessageStyle.Render(authHelp),
			)
		}

		return components.BoxStyle.Render(
			components.RenderTitle("Namespaces", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render(helpText),
		)
	}

	if m.list == nil {
		return "No namespaces available"
	}

	view := m.list.View()
	if m.message != "" {
		view += "\n" + components.RenderMessage(m.messageType, m.message)
	}
	return view
}

func (m *NamespacesViewModel) fetchNamespaces() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nsList, err := m.client.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return namespacesFetchedMsg{err: err}
	}

	return namespacesFetchedMsg{namespaces: nsList.Items}
}

func (m *NamespacesViewModel) updateList() {
	items := []components.ListItem{}

	for i := range m.namespaces {
		ns := &m.namespaces[i]

		// Determine status and icon
		status := string(ns.Status.Phase)
		icon := "📁"
		if ns.Name == m.currentNs {
			icon = "✅"
		}
		if status != "Active" {
			icon = "⚠️"
		}

		// Check if it's a system namespace
		isSystem := isSystemNamespace(ns.Name)
		systemLabel := ""
		if isSystem {
			systemLabel = " [system]"
		}

		age := services.FormatAge(ns.CreationTimestamp.Time)

		title := ns.Name
		if ns.Name == m.currentNs {
			title = fmt.Sprintf("%s (current)", ns.Name)
		}

		description := fmt.Sprintf("Status: %s | Age: %s%s", status, age, systemLabel)

		items = append(items, components.ListItem{
			ID:          ns.Name,
			Title:       title,
			Description: description,
			Icon:        icon,
			Data:        ns,
		})
	}

	title := fmt.Sprintf("📁 Namespaces (%d) - Current: %s", len(m.namespaces), m.currentNs)
	m.list = components.NewListView(title, items)
	m.list.SetHelpText("enter: switch namespace • r: refresh • esc/b: back • ctrl+c: quit")
}

func (m *NamespacesViewModel) switchNamespace(namespace string) tea.Cmd {
	return func() tea.Msg {
		// Update viper config
		viper.Set("k8s.namespace", namespace)

		// Try to save to config file (ignore errors if config doesn't exist)
		viper.WriteConfig()

		return namespaceSwitchedMsg{namespace: namespace}
	}
}

// isSystemNamespace checks if a namespace is a system namespace
func isSystemNamespace(name string) bool {
	systemNs := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"gke-managed-cim",
		"gmp-system",
		"gmp-public",
		"istio-system",
		"knative-serving",
		"config-management-system",
		"config-management-monitoring",
		"default",
	}

	for _, ns := range systemNs {
		if name == ns {
			return true
		}
	}
	return false
}
