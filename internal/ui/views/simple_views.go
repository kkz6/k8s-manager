package views

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodsViewModelSimple is a simplified pods view
type PodsViewModelSimple struct {
	client  *services.K8sClient
	pods    []corev1.Pod
	list    *components.ListView
	loading bool
	spinner components.SpinnerModel
	err     error
}

func NewPodsViewModelSimple() tea.Model {
	client, _ := services.GetK8sClient()
	return &PodsViewModelSimple{
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading pods..."),
	}
}

func (m *PodsViewModelSimple) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.fetchPods)
}

func (m *PodsViewModelSimple) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b":
			if m.err == nil {
				return m, Navigate(ViewMainMenu, nil)
			}
			return m, tea.Quit
		case "r":
			m.loading = true
			m.err = nil
			return m, m.fetchPods
		case "l":
			if m.err != nil && services.IsAuthError(m.err) {
				services.LoginGCP()
				return m, nil
			}
		}

	case podsFetchedMsg:
		m.loading = false
		m.pods = msg.pods
		m.err = msg.err
		if m.err == nil {
			m.updateList()
		}
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
				pod := selected.Data.(*corev1.Pod)
				return m, Navigate(ViewPodActions, map[string]string{
					"namespace": pod.Namespace,
					"name":      pod.Name,
				})
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

func (m *PodsViewModelSimple) View() string {
	if m.loading {
		return components.NewLoadingScreen("Loading Pods").View()
	}

	if m.err != nil {
		helpText := "Press 'r' to retry, 'q/b/esc' to go back, 'ctrl+c' to quit"

		if services.IsAuthError(m.err) {
			authHelp := services.GetAuthHelp(m.err.Error())
			return components.BoxStyle.Render(
				components.RenderTitle("🔐 Authentication Required", "") + "\n\n" +
					components.RenderMessage("error", m.err.Error()) + "\n\n" +
					components.InfoMessageStyle.Render(authHelp),
			)
		}

		return components.BoxStyle.Render(
			components.RenderTitle("Pods", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render(helpText),
		)
	}

	if m.list == nil {
		return "No pods available"
	}

	return m.list.View()
}

func (m *PodsViewModelSimple) fetchPods() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := services.GetCurrentNamespace()
	pods, err := m.client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return podsFetchedMsg{err: err}
	}

	return podsFetchedMsg{pods: pods.Items}
}

func (m *PodsViewModelSimple) updateList() {
	items := []components.ListItem{}

	for i := range m.pods {
		pod := &m.pods[i]
		ready := services.GetPodReadyCount(pod)
		age := services.FormatAge(pod.CreationTimestamp.Time)
		status := services.GetPodStatus(pod)

		title := pod.Name
		description := fmt.Sprintf("Status: %s, Ready: %s, Age: %s", status, ready, age)

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

		items = append(items, components.ListItem{
			ID:          pod.Name,
			Title:       title,
			Description: description,
			Icon:        icon,
			Data:        pod,
		})
	}

	title := fmt.Sprintf("📦 Pods (%d items) - Namespace: %s", len(m.pods), services.GetCurrentNamespace())
	m.list = components.NewListView(title, items)
	m.list.SetHelpText("enter: select pod • r: refresh • esc/b: back • ctrl+c: quit")
}

// ConfigsMenuModelSimple is a simplified configs menu
type ConfigsMenuModelSimple struct {
	list *components.ListView
}

func NewConfigsMenuModelSimple() tea.Model {
	items := []components.ListItem{
		{
			ID:          "configmaps",
			Title:       "ConfigMaps",
			Description: "View and manage Kubernetes ConfigMaps",
			Icon:        "📋",
		},
		{
			ID:          "secrets",
			Title:       "Secrets",
			Description: "View and manage Kubernetes Secrets",
			Icon:        "🔐",
		},
	}

	list := components.NewListView("⚙️ ConfigMaps & Secrets", items)
	list.SetHelpText("enter: select • q/b/esc: back to menu • ctrl+c: quit")

	return &ConfigsMenuModelSimple{
		list: list,
	}
}

func (m *ConfigsMenuModelSimple) Init() tea.Cmd {
	return nil
}

func (m *ConfigsMenuModelSimple) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b":
			return m, Navigate(ViewMainMenu, nil)
		case "c":
			return m, Navigate(ViewConfigMaps, nil)
		case "s":
			return m, Navigate(ViewSecrets, nil)
		}

		if msg.String() == "enter" || msg.String() == " " {
			selected := m.list.GetSelected()
			if selected != nil {
				switch selected.ID {
				case "configmaps":
					return m, Navigate(ViewConfigMaps, nil)
				case "secrets":
					return m, Navigate(ViewSecrets, nil)
				}
			}
		}
	}

	newList, cmd := m.list.Update(msg)
	if list, ok := newList.(components.ListView); ok {
		m.list = &list
	}
	return m, cmd
}

func (m *ConfigsMenuModelSimple) View() string {
	return m.list.View()
}

// ComingSoonViewModel shows a placeholder for unimplemented features
type ComingSoonViewModel struct {
	featureName string
	description string
}

func NewComingSoonView(featureName, description string) tea.Model {
	return &ComingSoonViewModel{
		featureName: featureName,
		description: description,
	}
}

func (m *ComingSoonViewModel) Init() tea.Cmd {
	return nil
}

func (m *ComingSoonViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b":
			return m, Navigate(ViewMainMenu, nil)
		}
	}
	return m, nil
}

func (m *ComingSoonViewModel) View() string {
	title := components.RenderTitle(fmt.Sprintf("🚧 %s - Coming Soon", m.featureName), "")

	content := fmt.Sprintf("\n\n  %s\n\n", m.description)
	content += "  This feature is under development and will be available in a future release.\n\n\n"

	helpText := "q/b/esc: Back • ctrl+c: Quit"
	content += components.HelpStyle.Render(helpText)

	return title + content
}

