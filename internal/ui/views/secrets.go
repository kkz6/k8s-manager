package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretsViewModel represents the secrets list view
type SecretsViewModel struct {
	client        *services.K8sClient
	secrets       []corev1.Secret
	table         *components.Table
	loading       bool
	err           error
	quitting      bool
	allNamespaces bool
}

// secretsFetchedMsg is sent when secrets are fetched
type secretsFetchedMsg struct {
	secrets []corev1.Secret
	err     error
}

// ShowSecretsView shows the interactive secrets view
func ShowSecretsView() error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	model := &SecretsViewModel{
		client:  client,
		loading: true,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// Init initializes the model
func (m *SecretsViewModel) Init() tea.Cmd {
	return m.fetchSecrets
}

// Update handles messages
func (m *SecretsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q", "b", "esc":
			return m, Navigate(ViewConfigsMenu, nil)
		case "r":
			m.loading = true
			return m, m.fetchSecrets
		case "enter", " ":
			if m.table != nil {
				selected := m.table.GetSelected()
				if len(selected) >= 2 {
					namespace := services.GetCurrentNamespace()
					name := selected[0]
					if m.allNamespaces && len(selected) > 0 {
						namespace = selected[0]
						name = selected[1]
					}
					return m, Navigate(ViewSecretDetail, map[string]string{
						"namespace": namespace,
						"name":      name,
					})
				}
			}
		}

	case tea.WindowSizeMsg:
		// Handle resize

	case secretsFetchedMsg:
		m.loading = false
		m.secrets = msg.secrets
		m.err = msg.err
		if m.err == nil && len(m.secrets) > 0 {
			m.table = m.createTable()
		}
		return m, nil
	}

	// Update table
	if m.table != nil && !m.loading {
		newTable, cmd := m.table.Update(msg)
		m.table = newTable.(*components.Table)
		return m, cmd
	}

	return m, nil
}

// View renders the view
func (m *SecretsViewModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		loadingStyle := components.InfoMessageStyle.Copy().
			Padding(2, 4)
		return "\n" + loadingStyle.Render("⏳ Loading Secrets...") + "\n\n"
	}

	if m.err != nil {
		errorMsg := components.RenderMessage("error", m.err.Error())
		helpText := "Press 'r' to refresh • Press 'q/b/esc' to go back • Press 'ctrl+c' to quit"

		containerStyle := lipgloss.NewStyle().Padding(1, 2)
		return "\n" + containerStyle.Render(errorMsg+"\n\n"+components.HelpStyle.Render(helpText)) + "\n"
	}

	if len(m.secrets) == 0 {
		emptyStyle := components.DescriptionStyle.Copy().
			Padding(2, 4)
		helpStyle := components.HelpStyle.Copy().
			Padding(1, 2)
		return "\n" + emptyStyle.Render("No Secrets found") + "\n" +
			helpStyle.Render("Press 'r' to refresh • Press 'q' to back") + "\n"
	}

	var b strings.Builder

	// Table
	b.WriteString("\n")
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Help footer
	helpStyle := components.HelpStyle.Copy().
		Foreground(components.ColorMuted).
		Padding(1, 2)

	helpText := components.RenderKeyBinding("↑/↓/j/k", "navigate") + " • " +
		components.RenderKeyBinding("enter", "view details") + " • " +
		components.RenderKeyBinding("r", "refresh") + " • " +
		components.RenderKeyBinding("q", "back")

	b.WriteString(helpStyle.Render(helpText))
	b.WriteString("\n")

	return b.String()
}

// createTable creates a table from the secrets
func (m *SecretsViewModel) createTable() *components.Table {
	var columns []components.TableColumn
	var rows []components.TableRow

	if m.allNamespaces {
		columns = []components.TableColumn{
			{Title: "Namespace", Width: 25, Align: "left"},
			{Title: "Name", Width: 40, Align: "left"},
			{Title: "Type", Width: 25, Align: "left"},
			{Title: "Keys", Width: 10, Align: "center"},
			{Title: "Age", Width: 10, Align: "right"},
		}
		for _, secret := range m.secrets {
			keys := fmt.Sprintf("%d", len(secret.Data))
			age := services.FormatAge(secret.CreationTimestamp.Time)
			secretType := string(secret.Type)
			if len(secretType) > 25 {
				secretType = secretType[:22] + "..."
			}

			rows = append(rows, components.TableRow{
				secret.Namespace,
				secret.Name,
				secretType,
				keys,
				age,
			})
		}
	} else {
		columns = []components.TableColumn{
			{Title: "Name", Width: 45, Align: "left"},
			{Title: "Type", Width: 30, Align: "left"},
			{Title: "Keys", Width: 10, Align: "center"},
			{Title: "Age", Width: 10, Align: "right"},
		}
		for _, secret := range m.secrets {
			keys := fmt.Sprintf("%d", len(secret.Data))
			age := services.FormatAge(secret.CreationTimestamp.Time)
			secretType := string(secret.Type)
			if len(secretType) > 30 {
				secretType = secretType[:27] + "..."
			}

			rows = append(rows, components.TableRow{
				secret.Name,
				secretType,
				keys,
				age,
			})
		}
	}

	namespace := services.GetCurrentNamespace()
	title := fmt.Sprintf("Secrets (%s)", namespace)
	if m.allNamespaces {
		title = "Secrets (All Namespaces)"
	}

	table := components.NewTable(title, columns)
	table.SetRows(rows)
	return table
}

// fetchSecrets fetches the secrets list
func (m *SecretsViewModel) fetchSecrets() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := ""
	if !m.allNamespaces {
		namespace = services.GetCurrentNamespace()
	}

	listOptions := metav1.ListOptions{}

	secrets, err := m.client.Clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
	if err != nil {
		return secretsFetchedMsg{err: err}
	}

	return secretsFetchedMsg{secrets: secrets.Items}
}

// NewSecretsViewModelSimple creates a simple secrets view model for navigation
func NewSecretsViewModelSimple() tea.Model {
	client, err := services.GetK8sClient()
	if err != nil {
		return &SecretsViewModel{err: err}
	}

	return &SecretsViewModel{
		client:  client,
		loading: true,
	}
}
