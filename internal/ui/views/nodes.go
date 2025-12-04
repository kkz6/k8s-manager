package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodesViewModel displays Kubernetes nodes and their resources
type NodesViewModel struct {
	client   *services.K8sClient
	nodes    []corev1.Node
	selected int
	loading  bool
	spinner  components.SpinnerModel
	err      error
	width    int
	height   int
}

// nodesFetchedMsg is sent when nodes are fetched
type nodesFetchedMsg struct {
	nodes []corev1.Node
	err   error
}

// NewNodesViewModel creates a new nodes view model
func NewNodesViewModel() tea.Model {
	client, _ := services.GetK8sClient()
	return &NodesViewModel{
		client:   client,
		loading:  true,
		spinner:  components.NewSpinner("Loading nodes..."),
		selected: 0,
	}
}

func (m *NodesViewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.fetchNodes)
}

func (m *NodesViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b", "0":
			return m, Navigate(ViewMainMenu, nil)
		case "r":
			m.loading = true
			m.err = nil
			return m, m.fetchNodes
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.nodes)-1 {
				m.selected++
			}
		}

	case nodesFetchedMsg:
		m.loading = false
		m.nodes = msg.nodes
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *NodesViewModel) View() string {
	if m.loading {
		return components.NewLoadingScreen("Loading Nodes").View()
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
			components.RenderTitle("Nodes", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render(helpText),
		)
	}

	return m.renderNodes()
}

func (m *NodesViewModel) fetchNodes() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeList, err := m.client.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nodesFetchedMsg{err: err}
	}

	return nodesFetchedMsg{nodes: nodeList.Items}
}

func (m *NodesViewModel) renderNodes() string {
	var b strings.Builder

	containerStyle := lipgloss.NewStyle().Padding(1, 2)

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render(fmt.Sprintf("🖥️  Nodes & Cluster (%d nodes)", len(m.nodes))))
	b.WriteString("\n\n")

	// Cluster summary
	b.WriteString(m.renderClusterSummary())
	b.WriteString("\n\n")

	// Node list
	for i, node := range m.nodes {
		b.WriteString(m.renderNodeCard(&node, i == m.selected))
		b.WriteString("\n")
	}

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(components.ColorMuted).
		MarginTop(1)

	help := "↑/k up • ↓/j down • r refresh • esc/b back • ctrl+c quit"
	b.WriteString(helpStyle.Render(help))

	return containerStyle.Render(b.String())
}

func (m *NodesViewModel) renderClusterSummary() string {
	var totalCPU, totalMem int64
	var readyNodes, totalNodes int

	for _, node := range m.nodes {
		totalNodes++
		if isNodeReady(&node) {
			readyNodes++
		}

		// Get allocatable resources
		cpu := node.Status.Allocatable.Cpu()
		mem := node.Status.Allocatable.Memory()
		if cpu != nil {
			totalCPU += cpu.MilliValue()
		}
		if mem != nil {
			totalMem += mem.Value()
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(components.ColorBorder).
		Padding(0, 2)

	labelStyle := lipgloss.NewStyle().
		Foreground(components.ColorInfo).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(components.ColorSuccess)

	var content strings.Builder
	content.WriteString(labelStyle.Render("CLUSTER SUMMARY"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Nodes: %s  |  ", valueStyle.Render(fmt.Sprintf("%d/%d Ready", readyNodes, totalNodes))))
	content.WriteString(fmt.Sprintf("Total CPU: %s  |  ", valueStyle.Render(services.FormatCPU(totalCPU))))
	content.WriteString(fmt.Sprintf("Total Memory: %s", valueStyle.Render(services.FormatMemory(totalMem))))

	return boxStyle.Render(content.String())
}

func (m *NodesViewModel) renderNodeCard(node *corev1.Node, selected bool) string {
	// Get node info
	ready := isNodeReady(node)
	status := "Ready"
	statusColor := components.ColorSuccess
	icon := "🟢"
	if !ready {
		status = "NotReady"
		statusColor = components.ColorError
		icon = "🔴"
	}

	// Check for conditions
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeMemoryPressure && cond.Status == corev1.ConditionTrue {
			icon = "🟡"
			statusColor = components.ColorWarning
		}
		if cond.Type == corev1.NodeDiskPressure && cond.Status == corev1.ConditionTrue {
			icon = "🟡"
			statusColor = components.ColorWarning
		}
	}

	// Get resources
	cpu := node.Status.Allocatable.Cpu()
	mem := node.Status.Allocatable.Memory()
	pods := node.Status.Allocatable.Pods()

	cpuStr := "N/A"
	memStr := "N/A"
	podsStr := "N/A"

	if cpu != nil {
		cpuStr = services.FormatCPU(cpu.MilliValue())
	}
	if mem != nil {
		memStr = services.FormatMemory(mem.Value())
	}
	if pods != nil {
		podsStr = pods.String()
	}

	// Get node info
	kubeletVersion := node.Status.NodeInfo.KubeletVersion
	osImage := node.Status.NodeInfo.OSImage
	containerRuntime := node.Status.NodeInfo.ContainerRuntimeVersion

	// Get roles
	roles := getNodeRoles(node)
	if roles == "" {
		roles = "<none>"
	}

	age := services.FormatAge(node.CreationTimestamp.Time)

	// Build card
	var card strings.Builder

	nameStyle := lipgloss.NewStyle().Bold(true)
	if selected {
		nameStyle = nameStyle.Foreground(components.ColorHighlight).Background(lipgloss.Color("236"))
	} else {
		nameStyle = nameStyle.Foreground(components.ColorPrimary)
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	labelStyle := lipgloss.NewStyle().Foreground(components.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	if selected {
		card.WriteString("▸ ")
	} else {
		card.WriteString("  ")
	}
	card.WriteString(icon)
	card.WriteString(" ")
	card.WriteString(nameStyle.Render(node.Name))
	card.WriteString("  ")
	card.WriteString(statusStyle.Render(status))
	card.WriteString("  ")
	card.WriteString(labelStyle.Render(fmt.Sprintf("Age: %s", age)))
	card.WriteString("\n")

	card.WriteString("    ")
	card.WriteString(labelStyle.Render("Roles: "))
	card.WriteString(valueStyle.Render(roles))
	card.WriteString("  ")
	card.WriteString(labelStyle.Render("Version: "))
	card.WriteString(valueStyle.Render(kubeletVersion))
	card.WriteString("\n")

	card.WriteString("    ")
	card.WriteString(labelStyle.Render("CPU: "))
	card.WriteString(valueStyle.Render(cpuStr))
	card.WriteString("  ")
	card.WriteString(labelStyle.Render("Memory: "))
	card.WriteString(valueStyle.Render(memStr))
	card.WriteString("  ")
	card.WriteString(labelStyle.Render("Pods: "))
	card.WriteString(valueStyle.Render(podsStr))
	card.WriteString("\n")

	card.WriteString("    ")
	card.WriteString(labelStyle.Render("OS: "))
	card.WriteString(valueStyle.Render(truncateStr(osImage, 40)))
	card.WriteString("  ")
	card.WriteString(labelStyle.Render("Runtime: "))
	card.WriteString(valueStyle.Render(containerRuntime))

	return card.String()
}

// isNodeReady checks if a node is ready
func isNodeReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// getNodeRoles returns the roles of a node
func getNodeRoles(node *corev1.Node) string {
	var roles []string
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	return strings.Join(roles, ",")
}

// truncateStr truncates a string to max length
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
