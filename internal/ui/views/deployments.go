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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentsOptions contains options for the deployments view
type DeploymentsOptions struct {
	AllNamespaces bool
	Selector      string
}

// DeploymentsModel represents the deployments list view
type DeploymentsModel struct {
	options     DeploymentsOptions
	client      *services.K8sClient
	deployments []appsv1.Deployment
	table       *components.Table
	loading     bool
	quitting    bool
	err         error
	spinner     components.SpinnerModel
}

// deploymentsLoadedMsg is sent when deployments are loaded
type deploymentsLoadedMsg struct {
	deployments []appsv1.Deployment
	err         error
}

// ShowDeploymentsView shows the deployments list view
func ShowDeploymentsView(options DeploymentsOptions) error {
	client, err := services.GetK8sClient()
	if err != nil {
		return err
	}

	model := &DeploymentsModel{
		options: options,
		client:  client,
		loading: true,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// Init initializes the model
func (m *DeploymentsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.loadDeployments)
}

// Update handles updates
func (m *DeploymentsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, Navigate(ViewMainMenu, nil)
		case "r":
			// Refresh deployments
			m.loading = true
			m.err = nil
			return m, m.loadDeployments
		case "l":
			if m.err != nil && services.IsAuthError(m.err) {
				services.LoginGCP()
				return m, nil
			}
		case "enter", " ":
			// Show deployment actions
			if m.table != nil {
				selected := m.table.GetSelected()
				if len(selected) >= 2 {
					namespace := services.GetCurrentNamespace()
					if m.options.AllNamespaces && len(selected) > 0 {
						namespace = selected[0]
					}
					deploymentName := selected[0]
					if m.options.AllNamespaces {
						deploymentName = selected[1]
					}
					return m, Navigate(ViewDeploymentActions, map[string]string{
						"namespace": namespace,
						"name":      deploymentName,
					})
				}
			}
		}

	case deploymentsLoadedMsg:
		m.loading = false
		m.deployments = msg.deployments
		m.err = msg.err

		if m.err == nil && len(m.deployments) > 0 {
			m.table = m.createTable()
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
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
func (m *DeploymentsModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		title := components.RenderTitle("📦 Deployments", "")
		return title + "\n\n  " + m.spinner.View() + "\n\n"
	}

	if m.err != nil {
		if services.IsAuthError(m.err) {
			authHelp := services.GetAuthHelp(m.err.Error())
			return components.BoxStyle.Render(
				components.RenderTitle("🔐 Authentication Required", "") + "\n\n" +
					components.RenderMessage("error", m.err.Error()) + "\n\n" +
					components.InfoMessageStyle.Render(authHelp),
			)
		}

		errorStyle := components.ErrorMessageStyle.Copy().
			Padding(1, 2)
		helpStyle := components.HelpStyle.Copy().
			Padding(1, 2)
		return "\n" + errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err)) + "\n" +
			helpStyle.Render("Press 'r' to retry • Press 'q/b/esc' to go back • Press 'ctrl+c' to quit") + "\n"
	}

	if len(m.deployments) == 0 {
		emptyStyle := components.DescriptionStyle.Copy().
			Padding(2, 4)
		helpStyle := components.HelpStyle.Copy().
			Padding(1, 2)
		return "\n" + emptyStyle.Render("No deployments found") + "\n" +
			helpStyle.Render("Press 'r' to refresh • Press 'q/b/esc' to go back • Press 'ctrl+c' to quit") + "\n"
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
		components.RenderKeyBinding("enter", "actions") + " • " +
		components.RenderKeyBinding("r", "refresh") + " • " +
		components.RenderKeyBinding("q", "back")

	b.WriteString(helpStyle.Render(helpText))
	b.WriteString("\n")

	return b.String()
}

// createTable creates a table from the deployments
func (m *DeploymentsModel) createTable() *components.Table {
	var columns []components.TableColumn
	var rows []components.TableRow

	if m.options.AllNamespaces {
		columns = []components.TableColumn{
			{Title: "Namespace", Width: 25, Align: "left"},
			{Title: "Name", Width: 45, Align: "left"},
			{Title: "Ready", Width: 10, Align: "center"},
			{Title: "Up-to-Date", Width: 10, Align: "center"},
			{Title: "Available", Width: 10, Align: "center"},
			{Title: "Age", Width: 10, Align: "right"},
		}
		for _, deploy := range m.deployments {
			ready := fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas)
			upToDate := fmt.Sprintf("%d", deploy.Status.UpdatedReplicas)
			available := fmt.Sprintf("%d", deploy.Status.AvailableReplicas)
			age := services.FormatAge(deploy.CreationTimestamp.Time)

			rows = append(rows, components.TableRow{
				deploy.Namespace,
				deploy.Name,
				ready,
				upToDate,
				available,
				age,
			})
		}
	} else {
		columns = []components.TableColumn{
			{Title: "Name", Width: 50, Align: "left"},
			{Title: "Ready", Width: 10, Align: "center"},
			{Title: "Up-to-Date", Width: 10, Align: "center"},
			{Title: "Available", Width: 10, Align: "center"},
			{Title: "Age", Width: 10, Align: "right"},
		}
		for _, deploy := range m.deployments {
			ready := fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas)
			upToDate := fmt.Sprintf("%d", deploy.Status.UpdatedReplicas)
			available := fmt.Sprintf("%d", deploy.Status.AvailableReplicas)
			age := services.FormatAge(deploy.CreationTimestamp.Time)

			rows = append(rows, components.TableRow{
				deploy.Name,
				ready,
				upToDate,
				available,
				age,
			})
		}
	}

	table := components.NewTable("Kubernetes Deployments", columns)
	table.SetRows(rows)
	return table
}

// NewDeploymentsViewModelSimple creates a simple deployments view model for navigation
func NewDeploymentsViewModelSimple() tea.Model {
	client, err := services.GetK8sClient()
	if err != nil {
		return &DeploymentsModel{err: err, spinner: components.NewSpinner("Loading deployments...")}
	}

	return &DeploymentsModel{
		options: DeploymentsOptions{
			AllNamespaces: false,
			Selector:      "",
		},
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading deployments..."),
	}
}

// loadDeployments loads the deployments from Kubernetes
func (m *DeploymentsModel) loadDeployments() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := services.GetCurrentNamespace()
	if m.options.AllNamespaces {
		namespace = ""
	}

	listOptions := metav1.ListOptions{}
	if m.options.Selector != "" {
		listOptions.LabelSelector = m.options.Selector
	}

	deployments, err := m.client.Clientset.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		return deploymentsLoadedMsg{err: err}
	}

	return deploymentsLoadedMsg{deployments: deployments.Items}
}
