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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentActionsModel represents the deployment actions menu
type DeploymentActionsModel struct {
	namespace      string
	name           string
	deployment     *appsv1.Deployment
	menu           *components.Menu
	client         *services.K8sClient
	loading        bool
	quitting       bool
	executing      bool
	watchingStatus bool
	currentAction  string
	statusMessage  string
}

// deploymentStatusTickMsg is sent periodically to update deployment status
type deploymentStatusTickMsg time.Time

// ShowDeploymentActionsView shows the deployment actions menu
func ShowDeploymentActionsView(namespace, name string) (tea.Model, tea.Cmd) {
	client, err := services.GetK8sClient()
	if err != nil {
		return nil, nil
	}

	menuItems := []components.MenuItem{
		{
			ID:          "rollout-restart",
			Title:       "Rollout Restart",
			Description: "Trigger a rolling restart (like kubectl rollout restart)",
			Icon:        "🔄",
			Shortcut:    "r",
		},
		{
			ID:          "update-image",
			Title:       "Update Image",
			Description: "Update container image to a new version",
			Icon:        "🖼️",
			Shortcut:    "u",
		},
		{
			ID:          "scale",
			Title:       "Scale Deployment",
			Description: "Scale replicas up or down",
			Icon:        "📊",
			Shortcut:    "s",
		},
		{
			ID:          "rollout-status",
			Title:       "Rollout Status",
			Description: "Show live rollout status",
			Icon:        "📈",
			Shortcut:    "t",
		},
		{
			ID:          "describe",
			Title:       "Describe",
			Description: "Show detailed deployment information",
			Icon:        "📋",
			Shortcut:    "d",
		},
		{
			ID:          "pods",
			Title:       "View Pods",
			Description: "List pods for this deployment",
			Icon:        "📦",
			Shortcut:    "p",
		},
		{
			ID:          "rollback",
			Title:       "Rollback",
			Description: "Rollback to previous revision",
			Icon:        "⏪",
			Shortcut:    "x",
		},
		{
			ID:          "back",
			Title:       "Back to Deployments",
			Description: "Return to deployments list",
			Icon:        "↩️",
			Shortcut:    "b",
		},
	}

	menu := components.NewDevToolsMenu(fmt.Sprintf("⚙️ Deployment Actions: %s", name), menuItems)

	model := &DeploymentActionsModel{
		namespace: namespace,
		name:      name,
		menu:      menu,
		client:    client,
	}

	// Store action handlers
	for i := range menu.Items {
		item := &menu.Items[i]
		switch item.ID {
		case "rollout-restart":
			item.Action = model.rolloutRestart
		case "update-image":
			item.Action = model.updateImage
		case "scale":
			item.Action = model.scaleDeployment
		case "rollout-status":
			item.Action = model.watchRolloutStatus
		case "describe":
			item.Action = model.describeDeployment
		case "pods":
			item.Action = model.viewPods
		case "rollback":
			item.Action = model.rollbackDeployment
		case "back":
			item.Action = func() tea.Cmd { return Navigate(ViewDeployments, nil) }
		}
	}

	return model, model.loadDeployment
}

// Init initializes the model
func (m *DeploymentActionsModel) Init() tea.Cmd {
	return tea.Batch(
		m.menu.Init(),
		m.loadDeployment,
		m.tickStatus(), // Start status updates
	)
}

// Update handles updates
func (m *DeploymentActionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q", "b", "esc":
			return m, Navigate(ViewDeployments, nil)
		}

		if msg.String() == "enter" || msg.String() == " " {
			selected := m.menu.GetSelected()
			if selected != nil {
				return m.handleAction(selected.ID)
			}
		}

		for _, item := range m.menu.Items {
			if item.Shortcut == msg.String() {
				return m.handleAction(item.ID)
			}
		}

	case deploymentLoadedMsg:
		m.deployment = msg.deployment
		m.loading = false
		if m.deployment != nil {
			m.updateMenuTitle()
		}
		return m, nil

	case deploymentStatusTickMsg:
		// Reload deployment for live status
		if !m.executing && !m.loading {
			return m, tea.Batch(
				m.loadDeployment,
				m.tickStatus(),
			)
		}
		return m, m.tickStatus()

	case actionCompletedMsg:
		m.executing = false
		m.watchingStatus = false
		return m, nil

	case components.ErrorMsg:
		m.executing = false
		m.watchingStatus = false
		return m, nil
	}

	newMenu, cmd := m.menu.Update(msg)
	if menu, ok := newMenu.(components.Menu); ok {
		m.menu = &menu
	}
	return m, cmd
}

// View renders the view
func (m *DeploymentActionsModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		loadingView := components.NewLoadingScreen("Loading Deployment Details")
		return loadingView.View()
	}

	if m.executing || m.watchingStatus {
		return m.renderExecutingView()
	}

	var b strings.Builder

	// Show deployment status at top
	if m.deployment != nil {
		b.WriteString(m.renderDeploymentStatus())
		b.WriteString("\n\n")
	}

	// Show menu
	b.WriteString(m.menu.View())

	return b.String()
}

// renderDeploymentStatus renders the current deployment status
func (m *DeploymentActionsModel) renderDeploymentStatus() string {
	if m.deployment == nil {
		return ""
	}

	statusBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(components.ColorBorder).
		Padding(1, 2).
		Width(70)

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(components.ColorPrimary)
	b.WriteString(titleStyle.Render(fmt.Sprintf("📊 %s", m.name)))
	b.WriteString("\n\n")

	desired := int32(0)
	if m.deployment.Spec.Replicas != nil {
		desired = *m.deployment.Spec.Replicas
	}

	// Replica status
	replicaStyle := lipgloss.NewStyle().Foreground(components.ColorInfo)
	b.WriteString(replicaStyle.Render("Replicas:"))
	b.WriteString(fmt.Sprintf("  %d desired | %d updated | %d ready | %d available\n",
		desired,
		m.deployment.Status.UpdatedReplicas,
		m.deployment.Status.ReadyReplicas,
		m.deployment.Status.AvailableReplicas))

	// Conditions
	b.WriteString("\n")
	condStyle := lipgloss.NewStyle().Foreground(components.ColorMuted)
	b.WriteString(condStyle.Render("Conditions:\n"))

	for _, cond := range m.deployment.Status.Conditions {
		statusIcon := "✓"
		statusColor := components.ColorSuccess
		if cond.Status != "True" {
			statusIcon = "✗"
			statusColor = components.ColorError
		}

		iconStyle := lipgloss.NewStyle().Foreground(statusColor)
		b.WriteString(fmt.Sprintf("  %s %s: %s\n",
			iconStyle.Render(statusIcon),
			cond.Type,
			cond.Status))
	}

	// Check rollout status
	b.WriteString("\n")
	if m.deployment.Status.UpdatedReplicas == desired &&
		m.deployment.Status.ReadyReplicas == desired &&
		m.deployment.Status.AvailableReplicas == desired {
		successStyle := lipgloss.NewStyle().Foreground(components.ColorSuccess).Bold(true)
		b.WriteString(successStyle.Render("✓ Rollout Complete"))
	} else {
		progressStyle := lipgloss.NewStyle().Foreground(components.ColorWarning)
		b.WriteString(progressStyle.Render("⏳ Rollout In Progress..."))
	}

	return statusBox.Render(b.String())
}

// renderExecutingView renders the executing/watching view
func (m *DeploymentActionsModel) renderExecutingView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(components.ColorPrimary).
		Padding(1, 2)

	b.WriteString(titleStyle.Render(fmt.Sprintf("🔄 %s", m.currentAction)))
	b.WriteString("\n\n")

	if m.watchingStatus && m.deployment != nil {
		b.WriteString(m.renderDeploymentStatus())
		b.WriteString("\n\n")

		helpStyle := lipgloss.NewStyle().
			Foreground(components.ColorMuted).
			Italic(true).
			Padding(1, 2)
		b.WriteString(helpStyle.Render("Watching live status... Press 'q' to stop"))
	} else {
		loadingView := components.NewLoadingScreen(m.currentAction)
		b.WriteString(loadingView.View())
	}

	if m.statusMessage != "" {
		b.WriteString("\n\n")
		msgStyle := lipgloss.NewStyle().
			Foreground(components.ColorInfo).
			Padding(0, 2)
		b.WriteString(msgStyle.Render(m.statusMessage))
	}

	return b.String()
}

// updateMenuTitle updates the menu title with current status
func (m *DeploymentActionsModel) updateMenuTitle() {
	if m.deployment == nil {
		return
	}

	replicas := int32(0)
	if m.deployment.Spec.Replicas != nil {
		replicas = *m.deployment.Spec.Replicas
	}

	statusIcon := "⏳"
	if m.deployment.Status.ReadyReplicas == replicas {
		statusIcon = "✓"
	}

	m.menu.Title = fmt.Sprintf("%s %s (%d/%d ready)", statusIcon, m.name, m.deployment.Status.ReadyReplicas, replicas)
}

// tickStatus returns a command that ticks every 3 seconds for status updates
func (m *DeploymentActionsModel) tickStatus() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return deploymentStatusTickMsg(t)
	})
}

// handleAction handles menu item selection
func (m *DeploymentActionsModel) handleAction(actionID string) (tea.Model, tea.Cmd) {
	switch actionID {
	case "rollout-restart":
		m.executing = true
		m.watchingStatus = true
		m.currentAction = "Rollout Restart"
		return m, m.rolloutRestart()
	case "update-image":
		m.executing = true
		m.currentAction = "Update Image"
		return m, m.updateImage()
	case "scale":
		m.executing = true
		m.currentAction = "Scale Deployment"
		return m, m.scaleDeployment()
	case "rollout-status":
		m.watchingStatus = true
		m.currentAction = "Rollout Status"
		return m, nil
	case "describe":
		m.executing = true
		m.currentAction = "Describe Deployment"
		return m, m.describeDeployment()
	case "pods":
		m.executing = true
		m.currentAction = "View Pods"
		return m, m.viewPods()
	case "rollback":
		m.executing = true
		m.currentAction = "Rollback Deployment"
		return m, m.rollbackDeployment()
	case "back":
		return m, Navigate(ViewDeployments, nil)
	}
	return m, nil
}

// Deployment action implementations

type deploymentLoadedMsg struct {
	deployment *appsv1.Deployment
	err        error
}

func (m *DeploymentActionsModel) loadDeployment() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployment, err := m.client.Clientset.AppsV1().Deployments(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
	if err != nil {
		return deploymentLoadedMsg{err: err}
	}

	return deploymentLoadedMsg{deployment: deployment}
}

func (m *DeploymentActionsModel) rolloutRestart() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get latest deployment
		deployment, err := m.client.Clientset.AppsV1().Deployments(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error getting deployment: %v\n", err)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			return actionCompletedMsg{}
		}

		// Add restart annotation
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

		_, err = m.client.Clientset.AppsV1().Deployments(m.namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("Error restarting deployment: %v\n", err)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			return actionCompletedMsg{}
		}

		fmt.Println("✓ Rollout restart triggered successfully")
		fmt.Println("\nWatch the status above to see the rollout progress...")
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()

		return actionCompletedMsg{}
	}
}

func (m *DeploymentActionsModel) watchRolloutStatus() tea.Cmd {
	return func() tea.Msg {
		// This will be handled by the View rendering and status ticker
		return nil
	}
}

func (m *DeploymentActionsModel) updateImage() tea.Cmd {
	return func() tea.Msg {
		// Get current image
		currentImage := ""
		if m.deployment != nil && len(m.deployment.Spec.Template.Spec.Containers) > 0 {
			currentImage = m.deployment.Spec.Template.Spec.Containers[0].Image
		}

		fmt.Printf("\nCurrent image: %s\n", currentImage)
		fmt.Print("Enter new image (with tag): ")

		var newImage string
		fmt.Scanln(&newImage)

		if newImage == "" {
			fmt.Println("Update cancelled")
			time.Sleep(1 * time.Second)
			return actionCompletedMsg{}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get latest and update
		deployment, err := m.client.Clientset.AppsV1().Deployments(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			return actionCompletedMsg{}
		}

		deployment.Spec.Template.Spec.Containers[0].Image = newImage

		_, err = m.client.Clientset.AppsV1().Deployments(m.namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("Error updating deployment: %v\n", err)
		} else {
			fmt.Printf("✓ Deployment image updated to '%s'\n", newImage)
			fmt.Println("Rollout has been triggered")
		}

		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()

		return actionCompletedMsg{}
	}
}

func (m *DeploymentActionsModel) scaleDeployment() tea.Cmd {
	return func() tea.Msg {
		currentReplicas := int32(0)
		if m.deployment.Spec.Replicas != nil {
			currentReplicas = *m.deployment.Spec.Replicas
		}

		fmt.Printf("\nCurrent replicas: %d\n", currentReplicas)
		fmt.Print("Enter new replica count: ")

		var newReplicas int32
		fmt.Scanf("%d", &newReplicas)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		deployment, err := m.client.Clientset.AppsV1().Deployments(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			return actionCompletedMsg{}
		}

		deployment.Spec.Replicas = &newReplicas

		_, err = m.client.Clientset.AppsV1().Deployments(m.namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("Error scaling deployment: %v\n", err)
		} else {
			fmt.Printf("✓ Deployment scaled from %d to %d replicas\n", currentReplicas, newReplicas)
		}

		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()

		return actionCompletedMsg{}
	}
}

func (m *DeploymentActionsModel) describeDeployment() tea.Cmd {
	return func() tea.Msg {
		var sb strings.Builder

		sb.WriteString(fmt.Sprintf("Name:         %s\n", m.deployment.Name))
		sb.WriteString(fmt.Sprintf("Namespace:    %s\n", m.deployment.Namespace))
		sb.WriteString(fmt.Sprintf("Labels:       %v\n", m.deployment.Labels))

		if m.deployment.Spec.Replicas != nil {
			sb.WriteString(fmt.Sprintf("Replicas:     %d desired | %d updated | %d available | %d ready\n",
				*m.deployment.Spec.Replicas,
				m.deployment.Status.UpdatedReplicas,
				m.deployment.Status.AvailableReplicas,
				m.deployment.Status.ReadyReplicas))
		}

		sb.WriteString("\nContainers:\n")
		for _, container := range m.deployment.Spec.Template.Spec.Containers {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", container.Name, container.Image))
		}

		if len(m.deployment.Status.Conditions) > 0 {
			sb.WriteString("\nConditions:\n")
			for _, cond := range m.deployment.Status.Conditions {
				sb.WriteString(fmt.Sprintf("  %s: %s (%s)\n", cond.Type, cond.Status, cond.Reason))
			}
		}

		model := NewPodDetailsModel("Deployment Details", sb.String())
		p := tea.NewProgram(model, tea.WithAltScreen())
		p.Run()

		return actionCompletedMsg{}
	}
}

func (m *DeploymentActionsModel) viewPods() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get pods with deployment label
		labelSelector := fmt.Sprintf("app=%s", m.name)
		if m.deployment != nil && len(m.deployment.Spec.Selector.MatchLabels) > 0 {
			labels := []string{}
			for k, v := range m.deployment.Spec.Selector.MatchLabels {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v))
			}
			labelSelector = strings.Join(labels, ",")
		}

		pods, err := m.client.Clientset.CoreV1().Pods(m.namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})

		if err != nil {
			fmt.Printf("Error listing pods: %v\n", err)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			return actionCompletedMsg{}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Pods for deployment '%s':\n\n", m.name))

		if len(pods.Items) == 0 {
			sb.WriteString("No pods found\n")
		} else {
			for _, pod := range pods.Items {
				status := services.GetPodStatus(&pod)
				ready := services.GetPodReadyCount(&pod)
				age := services.FormatAge(pod.CreationTimestamp.Time)
				sb.WriteString(fmt.Sprintf("  %s\t%s\t%s\t%s\n", pod.Name, status, ready, age))
			}
		}

		model := NewPodDetailsModel("Deployment Pods", sb.String())
		p := tea.NewProgram(model, tea.WithAltScreen())
		p.Run()

		return actionCompletedMsg{}
	}
}

func (m *DeploymentActionsModel) rollbackDeployment() tea.Cmd {
	return func() tea.Msg {
		fmt.Println("\nRollback deployment:")
		fmt.Println("1. Rollback to previous revision")
		fmt.Println("2. Rollback to specific revision")
		fmt.Print("\nEnter choice (1-2): ")

		var choice int
		fmt.Scanf("%d", &choice)

		if choice == 1 {
			fmt.Printf("\nTo rollback '%s' to previous revision, run:\n", m.name)
			fmt.Printf("  kubectl rollout undo deployment/%s -n %s\n", m.name, m.namespace)
		} else if choice == 2 {
			fmt.Print("\nEnter revision number: ")
			var revision int64
			fmt.Scanf("%d", &revision)
			fmt.Printf("\nTo rollback '%s' to revision %d, run:\n", m.name, revision)
			fmt.Printf("  kubectl rollout undo deployment/%s -n %s --to-revision=%d\n", m.name, m.namespace, revision)
		}

		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()

		return actionCompletedMsg{}
	}
}
