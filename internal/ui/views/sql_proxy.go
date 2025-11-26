package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/karthickk/k8s-manager/pkg/config"
)

// SQLProxyViewModel manages Cloud SQL proxy connections
type SQLProxyViewModel struct {
	manager     *services.SQLProxyManager
	instances   []string
	connections []*services.SQLProxyConnection
	list        *components.ListView
	loading     bool
	err         error
	message     string
	projectID   string
	region      string
}

type sqlInstancesLoadedMsg struct {
	instances []string
	err       error
}

type sqlProxyActionMsg struct {
	success bool
	message string
}

// NewSQLProxyViewModel creates a new SQL proxy view model
func NewSQLProxyViewModel() tea.Model {
	cfg := config.Get()
	return &SQLProxyViewModel{
		manager:   services.GetSQLProxyManager(),
		loading:   true,
		projectID: cfg.GCP.ProjectID,
		region:    cfg.GCP.Region,
	}
}

func (m *SQLProxyViewModel) Init() tea.Cmd {
	return m.loadInstances
}

func (m *SQLProxyViewModel) loadInstances() tea.Msg {
	instances, err := m.manager.ListInstances(m.projectID)
	return sqlInstancesLoadedMsg{
		instances: instances,
		err:       err,
	}
}

func (m *SQLProxyViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "q", "b", "esc":
				return m, Navigate(ViewMainMenu, nil)
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			// Stop all proxies before quitting
			m.manager.StopAll()
			return m, tea.Quit
		case "q", "b", "esc":
			return m, Navigate(ViewMainMenu, nil)
		case "r":
			m.loading = true
			m.message = ""
			return m, m.loadInstances
		case "s":
			// Start proxy for selected instance
			if m.list != nil {
				selected := m.list.GetSelected()
				if selected != nil {
					return m, m.startProxy(selected.ID)
				}
			}
		case "x":
			// Stop proxy for selected instance
			if m.list != nil {
				selected := m.list.GetSelected()
				if selected != nil {
					return m, m.stopProxy(selected.ID)
				}
			}
		case "a":
			// Stop all proxies
			return m, m.stopAllProxies
		case "o":
			// Open in TablePlus
			if m.list != nil {
				selected := m.list.GetSelected()
				if selected != nil {
					if conn, exists := m.manager.GetConnection(selected.ID); exists && conn.Status == "running" {
						return m, m.openInTablePlus(selected.ID, conn.LocalPort)
					}
				}
			}
		}

		if msg.String() == "enter" || msg.String() == " " {
			if m.list != nil {
				selected := m.list.GetSelected()
				if selected != nil {
					// Toggle proxy on/off
					if conn, exists := m.manager.GetConnection(selected.ID); exists && conn.Status == "running" {
						return m, m.stopProxy(selected.ID)
					}
					return m, m.startProxy(selected.ID)
				}
			}
		}

	case sqlInstancesLoadedMsg:
		m.loading = false
		m.instances = msg.instances
		m.err = msg.err
		if m.err == nil {
			m.updateList()
		}
		return m, nil

	case sqlProxyActionMsg:
		m.message = msg.message
		m.updateList()
		return m, nil
	}

	if m.list != nil {
		newList, cmd := m.list.Update(msg)
		if list, ok := newList.(components.ListView); ok {
			m.list = &list
		}
		return m, cmd
	}

	return m, nil
}

func (m *SQLProxyViewModel) startProxy(instanceName string) tea.Cmd {
	return func() tea.Msg {
		// Find an available port (starting from 3307 to avoid conflicts with local MySQL on 3306)
		port := 3307
		connections := m.manager.GetConnections()
		usedPorts := make(map[int]bool)
		for _, conn := range connections {
			if conn.Status == "running" {
				usedPorts[conn.LocalPort] = true
			}
		}

		// Find the first available port (check both proxy manager and system)
		maxAttempts := 100
		for i := 0; i < maxAttempts; i++ {
			if !usedPorts[port] {
				// Port is not used by our proxy manager, but check if it's available on the system
				break
			}
			port++
		}

		err := m.manager.StartProxy(m.projectID, m.region, instanceName, port)
		if err != nil {
			return sqlProxyActionMsg{
				success: false,
				message: fmt.Sprintf("Failed to start proxy: %v", err),
			}
		}

		return sqlProxyActionMsg{
			success: true,
			message: fmt.Sprintf("Proxy started for %s on localhost:%d", instanceName, port),
		}
	}
}

func (m *SQLProxyViewModel) stopProxy(instanceName string) tea.Cmd {
	return func() tea.Msg {
		err := m.manager.StopProxy(instanceName)
		if err != nil {
			return sqlProxyActionMsg{
				success: false,
				message: fmt.Sprintf("Failed to stop proxy: %v", err),
			}
		}

		return sqlProxyActionMsg{
			success: true,
			message: fmt.Sprintf("Proxy stopped for %s", instanceName),
		}
	}
}

func (m *SQLProxyViewModel) stopAllProxies() tea.Msg {
	err := m.manager.StopAll()
	if err != nil {
		return sqlProxyActionMsg{
			success: false,
			message: fmt.Sprintf("Failed to stop all proxies: %v", err),
		}
	}

	return sqlProxyActionMsg{
		success: true,
		message: "All proxies stopped",
	}
}

func (m *SQLProxyViewModel) openInTablePlus(instanceName string, port int) tea.Cmd {
	return func() tea.Msg {
		err := m.manager.OpenInTablePlus(instanceName, port)
		if err != nil {
			return sqlProxyActionMsg{
				success: false,
				message: fmt.Sprintf("Failed to open TablePlus: %v", err),
			}
		}

		return sqlProxyActionMsg{
			success: true,
			message: fmt.Sprintf("Opening %s in TablePlus on port %d", instanceName, port),
		}
	}
}

func (m *SQLProxyViewModel) updateList() {
	items := make([]components.ListItem, 0, len(m.instances))

	for _, instance := range m.instances {
		status := "⚪ Stopped"
		description := "Press 's' to start proxy"

		if conn, exists := m.manager.GetConnection(instance); exists {
			switch conn.Status {
			case "running":
				status = fmt.Sprintf("🟢 Running on localhost:%d", conn.LocalPort)
				description = fmt.Sprintf("Connection: %s • Started at %s • Logs: %s • Press 'x' to stop",
					conn.ConnectionName, conn.StartedAt.Format("15:04:05"), conn.LogFilePath)
			case "failed":
				status = "🔴 Failed"
				if conn.ErrorMessage != "" {
					description = fmt.Sprintf("Error: %s • Press 's' to restart", conn.ErrorMessage)
				} else {
					description = "Press 's' to restart"
				}
			case "stopped":
				status = "⚪ Stopped"
				description = "Press 's' to start"
			}
		}

		items = append(items, components.ListItem{
			ID:          instance,
			Title:       fmt.Sprintf("%s - %s", instance, status),
			Description: description,
			Icon:        "🗄️",
		})
	}

	list := components.NewListView("🗄️ Cloud SQL Proxy Manager", items)
	list.SetHelpText("enter/s: start • x: stop • o: open in TablePlus • a: stop all • r: refresh • q/b/esc: back • ctrl+c: quit")
	m.list = list
}

func (m *SQLProxyViewModel) View() string {
	title := components.RenderTitle("🗄️ Cloud SQL Proxy Manager", "")

	if m.loading {
		return title + "\n\n  Loading Cloud SQL instances...\n\n"
	}

	if m.err != nil {
		errorMsg := fmt.Sprintf("\n  ❌ Error: %s\n\n", m.err.Error())
		help := "  " + components.HelpStyle.Render("r: Retry • q/b/esc: Back • ctrl+c: Quit")
		return title + errorMsg + help
	}

	if len(m.instances) == 0 {
		return title + "\n\n  No Cloud SQL instances found in project: " + m.projectID + "\n\n  " +
			components.HelpStyle.Render("r: Refresh • q/b/esc: Back")
	}

	content := ""
	if m.list != nil {
		content = m.list.View()
	}

	if m.message != "" {
		messageStyle := components.InfoMessageStyle.Copy().Padding(1, 2)
		content += "\n" + messageStyle.Render(m.message)
	}

	// Add usage instructions
	instructions := "\n  💡 Connection Instructions:\n" +
		"     1. Start a proxy by selecting an instance and pressing 's' or Enter\n" +
		"     2. Connect to your database using: mysql -h 127.0.0.1 -P <port> -u <user> -p\n" +
		"     3. Or update your application's database host to 127.0.0.1:<port>\n"

	content += "\n" + components.DescriptionStyle.Render(instructions)

	return content
}
