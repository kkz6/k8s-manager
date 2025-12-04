package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
)

// ViewResourceManager is the view constant for navigation
const ViewResourceManager View = "resource_manager"

// ResourceManagerOptions contains options for the resource manager
type ResourceManagerOptions struct {
	Namespace        string
	AllNamespaces    bool
	PollInterval     time.Duration
	SortBy           services.SortBy
	Descending       bool
	ExcludeSystem    bool     // Exclude system namespaces
	ExcludedNs       []string // List of namespaces to exclude
}

// SystemNamespaces are the default system namespaces to exclude
var SystemNamespaces = []string{
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
	"asm-system",
	"anthos-identity-service",
	"gke-gmp-system",
}

// DefaultResourceManagerOptions returns default options
func DefaultResourceManagerOptions() ResourceManagerOptions {
	return ResourceManagerOptions{
		Namespace:     "",
		AllNamespaces: true,
		PollInterval:  5 * time.Second,
		SortBy:        services.SortByCPU,
		Descending:    true,
		ExcludeSystem: true, // By default, exclude system namespaces
		ExcludedNs:    SystemNamespaces,
	}
}

// ResourceManagerModel is the main resource manager view model
type ResourceManagerModel struct {
	options       ResourceManagerOptions
	metricsClient *services.MetricsClient
	podMetrics    []services.PodMetrics
	clusterMetrics *services.ClusterMetrics
	viewport      viewport.Model
	spinner       spinner.Model
	loading       bool
	err           error
	quitting      bool
	paused        bool
	selected      int
	width         int
	height        int
	lastUpdate    time.Time
	filterNs      string // Filter by namespace
	showHelp      bool
	ready         bool
}

// metricsTickMsg is sent when it's time to refresh metrics
type metricsTickMsg struct{}

// metricsUpdatedMsg is sent when metrics are fetched
type metricsUpdatedMsg struct {
	podMetrics     []services.PodMetrics
	clusterMetrics *services.ClusterMetrics
	err            error
}

// NewResourceManagerModel creates a new resource manager model
func NewResourceManagerModel(opts ResourceManagerOptions) *ResourceManagerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(components.ColorPrimary)

	return &ResourceManagerModel{
		options:  opts,
		spinner:  s,
		loading:  true,
		selected: 0,
	}
}

// Init initializes the model
func (m *ResourceManagerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchMetrics,
		m.scheduleTick(),
	)
}

// scheduleTick schedules the next metrics refresh
func (m *ResourceManagerModel) scheduleTick() tea.Cmd {
	return tea.Tick(m.options.PollInterval, func(t time.Time) tea.Msg {
		return metricsTickMsg{}
	})
}

// isExcludedNamespace checks if a namespace should be excluded
func (m *ResourceManagerModel) isExcludedNamespace(ns string) bool {
	if !m.options.ExcludeSystem {
		return false
	}
	for _, excluded := range m.options.ExcludedNs {
		if ns == excluded {
			return true
		}
	}
	return false
}

// fetchMetrics fetches metrics from the cluster
func (m *ResourceManagerModel) fetchMetrics() tea.Msg {
	client, err := services.GetMetricsClient()
	if err != nil {
		return metricsUpdatedMsg{err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := m.options.Namespace
	if m.options.AllNamespaces {
		namespace = ""
	}

	pods, err := client.GetPodMetrics(ctx, namespace)
	if err != nil {
		return metricsUpdatedMsg{err: err}
	}

	// Filter out system namespaces if enabled
	if m.options.ExcludeSystem && namespace == "" {
		filteredPods := make([]services.PodMetrics, 0, len(pods))
		for _, pod := range pods {
			if !m.isExcludedNamespace(pod.Namespace) {
				filteredPods = append(filteredPods, pod)
			}
		}
		pods = filteredPods
	}

	// Get cluster metrics (we'll recalculate based on filtered pods)
	cluster := &services.ClusterMetrics{
		TotalPods:   len(pods),
		LastUpdated: time.Now(),
		Namespaces:  make(map[string]services.NamespaceMetrics),
	}

	for _, pm := range pods {
		cluster.UsedCPU += pm.TotalCPU
		cluster.UsedMemory += pm.TotalMemory
		cluster.TotalCPU += pm.CPURequest
		cluster.TotalMemory += pm.MemoryRequest

		if pm.Status == "Running" {
			cluster.RunningPods++
		}

		// Aggregate by namespace
		ns, exists := cluster.Namespaces[pm.Namespace]
		if !exists {
			ns = services.NamespaceMetrics{Name: pm.Namespace}
		}
		ns.PodCount++
		ns.TotalCPU += pm.TotalCPU
		ns.TotalMemory += pm.TotalMemory
		ns.CPURequest += pm.CPURequest
		ns.MemoryRequest += pm.MemoryRequest
		cluster.Namespaces[pm.Namespace] = ns
	}

	if cluster.TotalCPU > 0 {
		cluster.CPUPercent = float64(cluster.UsedCPU) / float64(cluster.TotalCPU) * 100
	}
	if cluster.TotalMemory > 0 {
		cluster.MemoryPercent = float64(cluster.UsedMemory) / float64(cluster.TotalMemory) * 100
	}

	// Sort pods
	services.SortPodMetrics(pods, m.options.SortBy, m.options.Descending)

	return metricsUpdatedMsg{
		podMetrics:     pods,
		clusterMetrics: cluster,
	}
}

// Update handles messages
func (m *ResourceManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Initialize viewport
		headerHeight := 12 // Header + cluster summary
		footerHeight := 4  // Help line
		viewportHeight := m.height - headerHeight - footerHeight
		if viewportHeight < 5 {
			viewportHeight = 5
		}

		if !m.ready {
			m.viewport = viewport.New(m.width-4, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = m.width - 4
			m.viewport.Height = viewportHeight
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "esc", "b", "0":
			// Go back to main menu
			return m, Navigate(ViewMainMenu, nil)

		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.updateViewport()
			}

		case "down", "j":
			if m.selected < len(m.podMetrics)-1 {
				m.selected++
				m.updateViewport()
			}

		case "home", "g":
			m.selected = 0
			m.updateViewport()

		case "end", "G":
			m.selected = len(m.podMetrics) - 1
			m.updateViewport()

		case "r":
			// Force refresh
			m.loading = true
			return m, m.fetchMetrics

		case "p":
			// Toggle pause
			m.paused = !m.paused

		case "s":
			// Cycle sort order: CPU -> Memory -> Name
			switch m.options.SortBy {
			case services.SortByCPU:
				m.options.SortBy = services.SortByMemory
			case services.SortByMemory:
				m.options.SortBy = services.SortByName
				m.options.Descending = false
			case services.SortByName:
				m.options.SortBy = services.SortByCPU
				m.options.Descending = true
			}
			services.SortPodMetrics(m.podMetrics, m.options.SortBy, m.options.Descending)
			m.updateViewport()

		case "d":
			// Toggle descending
			m.options.Descending = !m.options.Descending
			services.SortPodMetrics(m.podMetrics, m.options.SortBy, m.options.Descending)
			m.updateViewport()

		case "a":
			// Toggle all namespaces
			m.options.AllNamespaces = !m.options.AllNamespaces
			m.loading = true
			return m, m.fetchMetrics

		case "u":
			// Toggle user-only mode (exclude system namespaces)
			m.options.ExcludeSystem = !m.options.ExcludeSystem
			m.loading = true
			return m, m.fetchMetrics

		case "n":
			// Cycle through namespaces
			if m.clusterMetrics != nil && len(m.clusterMetrics.Namespaces) > 0 {
				namespaces := make([]string, 0, len(m.clusterMetrics.Namespaces)+1)
				namespaces = append(namespaces, "") // All namespaces
				for ns := range m.clusterMetrics.Namespaces {
					namespaces = append(namespaces, ns)
				}
				// Find current and move to next
				currentIdx := 0
				for i, ns := range namespaces {
					if ns == m.filterNs {
						currentIdx = i
						break
					}
				}
				m.filterNs = namespaces[(currentIdx+1)%len(namespaces)]
				m.updateViewport()
			}

		case "?", "h":
			m.showHelp = !m.showHelp
		}

	case metricsTickMsg:
		if !m.paused {
			cmds = append(cmds, m.fetchMetrics)
		}
		cmds = append(cmds, m.scheduleTick())

	case metricsUpdatedMsg:
		m.loading = false
		m.lastUpdate = time.Now()
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.podMetrics = msg.podMetrics
			m.clusterMetrics = msg.clusterMetrics
			m.updateViewport()
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// updateViewport updates the viewport content
func (m *ResourceManagerModel) updateViewport() {
	if !m.ready {
		return
	}

	content := m.renderPodList()
	m.viewport.SetContent(content)

	// Ensure selected pod is visible
	linesPerPod := 5 // Each pod card takes ~5 lines
	selectedLine := m.selected * linesPerPod
	if selectedLine < m.viewport.YOffset {
		m.viewport.SetYOffset(selectedLine)
	} else if selectedLine >= m.viewport.YOffset+m.viewport.Height-linesPerPod {
		m.viewport.SetYOffset(selectedLine - m.viewport.Height + linesPerPod + 1)
	}
}

// View renders the view
func (m *ResourceManagerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Container style
	containerStyle := lipgloss.NewStyle().Padding(1, 2)

	// Render header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Render cluster summary
	if m.clusterMetrics != nil {
		b.WriteString(m.renderClusterSummary())
		b.WriteString("\n")
	}

	// Loading state
	if m.loading && len(m.podMetrics) == 0 {
		loadingStyle := lipgloss.NewStyle().
			Foreground(components.ColorMuted).
			Padding(2, 0)
		b.WriteString(loadingStyle.Render(m.spinner.View() + " Loading metrics..."))
		return containerStyle.Render(b.String())
	}

	// Error state
	if m.err != nil {
		b.WriteString(components.RenderMessage("error", m.err.Error()))
		b.WriteString("\n")
	}

	// Render pod list header
	b.WriteString(m.renderPodListHeader())
	b.WriteString("\n")

	// Render viewport with pods
	if m.ready {
		b.WriteString(m.viewport.View())
	}

	// Render footer/help
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return containerStyle.Render(b.String())
}

// renderHeader renders the header section
func (m *ResourceManagerModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginRight(2)

	infoStyle := lipgloss.NewStyle().
		Foreground(components.ColorMuted)

	modeStyle := lipgloss.NewStyle().
		Foreground(components.ColorSuccess).
		Bold(true)

	statusStyle := lipgloss.NewStyle()
	if m.paused {
		statusStyle = statusStyle.Foreground(components.ColorWarning)
	} else {
		statusStyle = statusStyle.Foreground(components.ColorSuccess)
	}

	var parts []string
	parts = append(parts, titleStyle.Render("📊 Resource Manager"))

	// Mode indicator
	if m.options.ExcludeSystem {
		parts = append(parts, modeStyle.Render("[User Pods]"))
	} else {
		parts = append(parts, infoStyle.Render("[All Pods]"))
	}

	// Pod count
	if m.clusterMetrics != nil {
		parts = append(parts, infoStyle.Render(fmt.Sprintf("Pods: %d", m.clusterMetrics.TotalPods)))
	}

	// Polling status
	pollStatus := "●"
	if m.paused {
		pollStatus = "⏸"
	} else if m.loading {
		pollStatus = m.spinner.View()
	}
	parts = append(parts, statusStyle.Render(pollStatus))
	parts = append(parts, infoStyle.Render(fmt.Sprintf("%ds", int(m.options.PollInterval.Seconds()))))

	// Last update
	if !m.lastUpdate.IsZero() {
		parts = append(parts, infoStyle.Render(fmt.Sprintf("Updated: %s", m.lastUpdate.Format("15:04:05"))))
	}

	return strings.Join(parts, "  ")
}

// renderClusterSummary renders the cluster summary section
func (m *ResourceManagerModel) renderClusterSummary() string {
	if m.clusterMetrics == nil {
		return ""
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(components.ColorBorder).
		Padding(0, 2).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(components.ColorInfo).
		Bold(true).
		Width(8)

	// Resource bars
	style := components.ResourceBarStyle{
		Width:          35,
		ShowPercentage: true,
		ShowValues:     true,
	}

	cpuUsed := services.FormatCPU(m.clusterMetrics.UsedCPU)
	cpuTotal := services.FormatCPU(m.clusterMetrics.TotalCPU)
	memUsed := services.FormatMemory(m.clusterMetrics.UsedMemory)
	memTotal := services.FormatMemory(m.clusterMetrics.TotalMemory)

	var content strings.Builder
	content.WriteString(labelStyle.Render("CLUSTER"))
	content.WriteString("  ")
	content.WriteString(fmt.Sprintf("Running: %d/%d", m.clusterMetrics.RunningPods, m.clusterMetrics.TotalPods))
	content.WriteString("\n")
	content.WriteString(components.RenderResourceBar("CPU", m.clusterMetrics.CPUPercent, cpuUsed, cpuTotal, style))
	content.WriteString("\n")
	content.WriteString(components.RenderResourceBar("MEM", m.clusterMetrics.MemoryPercent, memUsed, memTotal, style))

	return boxStyle.Render(content.String())
}

// renderPodListHeader renders the pod list header
func (m *ResourceManagerModel) renderPodListHeader() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(components.ColorMuted).
		Bold(true)

	sortIndicator := func(field services.SortBy) string {
		if m.options.SortBy == field {
			if m.options.Descending {
				return " ▼"
			}
			return " ▲"
		}
		return ""
	}

	nsFilter := "all"
	if m.filterNs != "" {
		nsFilter = m.filterNs
	}

	header := fmt.Sprintf("%-50s  %-10s  CPU%s          MEM%s          Namespace: %s",
		"POD",
		"STATUS",
		sortIndicator(services.SortByCPU),
		sortIndicator(services.SortByMemory),
		nsFilter,
	)

	return headerStyle.Render(header)
}

// renderPodList renders the list of pods
func (m *ResourceManagerModel) renderPodList() string {
	if len(m.podMetrics) == 0 {
		return lipgloss.NewStyle().
			Foreground(components.ColorMuted).
			Padding(2, 0).
			Render("No pods found")
	}

	var content strings.Builder

	for i, pod := range m.podMetrics {
		// Filter by namespace if set
		if m.filterNs != "" && pod.Namespace != m.filterNs {
			continue
		}

		cpuUsed := services.FormatCPU(pod.TotalCPU)
		cpuTotal := services.FormatCPU(pod.CPURequest)
		if pod.CPURequest == 0 {
			cpuTotal = services.FormatCPU(pod.CPULimit)
		}
		if cpuTotal == "0m" {
			cpuTotal = "-"
		}

		memUsed := services.FormatMemory(pod.TotalMemory)
		memTotal := services.FormatMemory(pod.MemoryRequest)
		if pod.MemoryRequest == 0 {
			memTotal = services.FormatMemory(pod.MemoryLimit)
		}
		if memTotal == "0B" {
			memTotal = "-"
		}

		card := components.RenderPodResourceCard(
			pod.Name,
			pod.Namespace,
			pod.Status,
			pod.CPUPercent,
			pod.MemoryPercent,
			cpuUsed,
			cpuTotal,
			memUsed,
			memTotal,
			i == m.selected,
		)

		content.WriteString(card)
		content.WriteString("\n\n")
	}

	return content.String()
}

// renderHelp renders the help footer
func (m *ResourceManagerModel) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(components.ColorMuted)

	keyStyle := lipgloss.NewStyle().
		Foreground(components.ColorSecondary).
		Bold(true)

	if m.showHelp {
		// Extended help
		var lines []string
		lines = append(lines, "")
		lines = append(lines, keyStyle.Render("Navigation"))
		lines = append(lines, "  ↑/k up  ↓/j down  g/G home/end  enter select")
		lines = append(lines, "")
		lines = append(lines, keyStyle.Render("Controls"))
		lines = append(lines, "  r refresh  p pause/resume  s cycle sort  d toggle desc")
		lines = append(lines, "  u toggle user/all pods  n cycle namespace filter")
		lines = append(lines, "")
		lines = append(lines, keyStyle.Render("Exit"))
		lines = append(lines, "  q quit  b/esc back  ? toggle help")
		lines = append(lines, "")
		lines = append(lines, components.RenderLegend())

		return helpStyle.Render(strings.Join(lines, "\n"))
	}

	// Compact help
	parts := []string{
		keyStyle.Render("↑↓") + " navigate",
		keyStyle.Render("r") + " refresh",
		keyStyle.Render("p") + " pause",
		keyStyle.Render("s") + " sort",
		keyStyle.Render("u") + " user/all",
		keyStyle.Render("n") + " filter ns",
		keyStyle.Render("?") + " help",
		keyStyle.Render("q") + " quit",
	}

	return helpStyle.Render(strings.Join(parts, "  "))
}

// ShowResourceManager shows the resource manager view
func ShowResourceManager(opts ResourceManagerOptions) error {
	model := NewResourceManagerModel(opts)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
