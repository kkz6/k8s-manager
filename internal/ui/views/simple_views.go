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
			return m, Navigate(ViewMainMenu, nil)
		case "r":
			m.loading = true
			return m, m.fetchPods
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
		return components.BoxStyle.Render(
			components.RenderTitle("Pods", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render("Press 'r' to retry, 'q' to back"),
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
	list.SetHelpText("enter: select • b: back to menu • q: quit")

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

// ConfigMapsViewModelSimple is a simplified configmaps view
type ConfigMapsViewModelSimple struct {
	client     *services.K8sClient
	configMaps []corev1.ConfigMap
	list       *components.ListView
	loading    bool
	spinner    components.SpinnerModel
	err        error
}

func NewConfigMapsViewModelSimple() tea.Model {
	client, _ := services.GetK8sClient()
	return &ConfigMapsViewModelSimple{
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading ConfigMaps..."),
	}
}

func (m *ConfigMapsViewModelSimple) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.fetchConfigMaps)
}

func (m *ConfigMapsViewModelSimple) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b":
			return m, Navigate(ViewConfigsMenu, nil)
		case "r":
			m.loading = true
			return m, m.fetchConfigMaps
		case "a":
			// TODO: Add new ConfigMap
			return m, nil
		}

	case configMapsFetchedMsg:
		m.loading = false
		m.configMaps = msg.configMaps
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
				parts := strings.Split(selected.ID, "/")
				if len(parts) == 2 {
					return m, Navigate(ViewConfigMapDetail, map[string]string{
						"namespace": parts[0],
						"name":      parts[1],
					})
				}
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

func (m *ConfigMapsViewModelSimple) View() string {
	if m.loading {
		return components.NewLoadingScreen("Loading ConfigMaps").View()
	}

	if m.err != nil {
		return components.BoxStyle.Render(
			components.RenderTitle("ConfigMaps", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render("Press 'r' to retry, 'b' to back"),
		)
	}

	if m.list == nil {
		return "No ConfigMaps available"
	}

	return m.list.View()
}

func (m *ConfigMapsViewModelSimple) fetchConfigMaps() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := services.GetCurrentNamespace()
	configMaps, err := m.client.Clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return configMapsFetchedMsg{err: err}
	}

	return configMapsFetchedMsg{configMaps: configMaps.Items}
}

func (m *ConfigMapsViewModelSimple) updateList() {
	items := []components.ListItem{}

	for _, cm := range m.configMaps {
		age := services.FormatAge(cm.CreationTimestamp.Time)
		title := cm.Name
		description := fmt.Sprintf("Keys: %d, Namespace: %s, Age: %s",
			len(cm.Data), cm.Namespace, age)

		items = append(items, components.ListItem{
			ID:          fmt.Sprintf("%s/%s", cm.Namespace, cm.Name),
			Title:       title,
			Description: description,
			Icon:        "📋",
		})
	}

	title := fmt.Sprintf("📋 ConfigMaps (%d items) - Namespace: %s", len(m.configMaps), services.GetCurrentNamespace())
	m.list = components.NewListView(title, items)
	m.list.SetHelpText("enter: view details • a: add new • r: refresh • esc/b: back • ctrl+c: quit")
}

// SecretsViewModelSimple is a simplified secrets view
type SecretsViewModelSimple struct {
	client  *services.K8sClient
	secrets []corev1.Secret
	list    *components.ListView
	loading bool
	spinner components.SpinnerModel
	err     error
}

func NewSecretsViewModelSimple() tea.Model {
	client, _ := services.GetK8sClient()
	return &SecretsViewModelSimple{
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading Secrets..."),
	}
}

func (m *SecretsViewModelSimple) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.fetchSecrets)
}

func (m *SecretsViewModelSimple) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b":
			return m, Navigate(ViewConfigsMenu, nil)
		case "r":
			m.loading = true
			return m, m.fetchSecrets
		case "a":
			// TODO: Add new Secret
			return m, nil
		}

	case secretsFetchedMsg:
		m.loading = false
		m.secrets = msg.secrets
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
				parts := strings.Split(selected.ID, "/")
				if len(parts) == 2 {
					return m, Navigate(ViewSecretDetail, map[string]string{
						"namespace": parts[0],
						"name":      parts[1],
					})
				}
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

func (m *SecretsViewModelSimple) View() string {
	if m.loading {
		return components.NewLoadingScreen("Loading Secrets").View()
	}

	if m.err != nil {
		return components.BoxStyle.Render(
			components.RenderTitle("Secrets", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render("Press 'r' to retry, 'b' to back"),
		)
	}

	if m.list == nil {
		return "No Secrets available"
	}

	return m.list.View()
}

func (m *SecretsViewModelSimple) fetchSecrets() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := services.GetCurrentNamespace()
	secrets, err := m.client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return secretsFetchedMsg{err: err}
	}

	return secretsFetchedMsg{secrets: secrets.Items}
}

func (m *SecretsViewModelSimple) updateList() {
	items := []components.ListItem{}

	for _, secret := range m.secrets {
		age := services.FormatAge(secret.CreationTimestamp.Time)
		secretType := string(secret.Type)
		if secretType == string(corev1.SecretTypeOpaque) {
			secretType = "Opaque"
		}

		title := secret.Name
		description := fmt.Sprintf("Type: %s, Keys: %d, Namespace: %s, Age: %s",
			secretType, len(secret.Data), secret.Namespace, age)

		icon := "🔐"
		if strings.Contains(string(secret.Type), "tls") {
			icon = "🔒"
		} else if strings.Contains(string(secret.Type), "docker") {
			icon = "🐳"
		}

		items = append(items, components.ListItem{
			ID:          fmt.Sprintf("%s/%s", secret.Namespace, secret.Name),
			Title:       title,
			Description: description,
			Icon:        icon,
		})
	}

	title := fmt.Sprintf("🔐 Secrets (%d items) - Namespace: %s", len(m.secrets), services.GetCurrentNamespace())
	m.list = components.NewListView(title, items)
	m.list.SetHelpText("enter: view details • a: add new • r: refresh • esc/b: back • ctrl+c: quit")
}