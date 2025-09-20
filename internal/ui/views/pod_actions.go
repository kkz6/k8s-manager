package views

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodActionsModel represents the pod actions menu
type PodActionsModel struct {
	namespace string
	name      string
	pod       *corev1.Pod
	menu      *components.Menu
	client    *services.K8sClient
	loading   bool
	quitting  bool
}

// ShowPodActionsView shows the pod actions menu
func ShowPodActionsView(namespace, name string) (tea.Model, tea.Cmd) {
	client, err := services.GetK8sClient()
	if err != nil {
		// Return error model
		return nil, nil
	}

	menuItems := []components.MenuItem{
		{
			ID:          "describe",
			Title:       "Describe Pod",
			Description: "Show detailed pod information and events",
			Icon:        "📋",
			Shortcut:    "d",
		},
		{
			ID:          "logs",
			Title:       "View Logs",
			Description: "View pod logs (last 100 lines)",
			Icon:        "📝",
			Shortcut:    "l",
		},
		{
			ID:          "logs-follow",
			Title:       "Follow Logs",
			Description: "Stream logs in real-time",
			Icon:        "📊",
			Shortcut:    "f",
		},
		{
			ID:          "exec",
			Title:       "Execute Shell",
			Description: "Open interactive shell in pod",
			Icon:        "⚡",
			Shortcut:    "e",
		},
		{
			ID:          "port-forward",
			Title:       "Port Forward",
			Description: "Forward local port to pod",
			Icon:        "🌐",
			Shortcut:    "p",
		},
		{
			ID:          "env",
			Title:       "Environment Variables",
			Description: "Manage environment variables",
			Icon:        "🔧",
			Shortcut:    "v",
		},
		{
			ID:          "restart",
			Title:       "Restart Pod",
			Description: "Delete pod to trigger restart",
			Icon:        "🔄",
			Shortcut:    "r",
		},
		{
			ID:          "delete",
			Title:       "Delete Pod",
			Description: "Permanently delete the pod",
			Icon:        "🗑️",
			Shortcut:    "x",
		},
		{
			ID:          "back",
			Title:       "Back to Pods List",
			Description: "Return to pods list",
			Icon:        "↩️",
			Shortcut:    "b",
		},
	}

	// Create DevTools-style menu
	menu := components.NewDevToolsMenu(fmt.Sprintf("⚡ Pod Actions: %s", name), menuItems)

	model := &PodActionsModel{
		namespace: namespace,
		name:      name,
		menu:      menu,
		client:    client,
	}

	// Store action handlers on the model
	for i := range menu.Items {
		item := &menu.Items[i]
		switch item.ID {
		case "describe":
			item.Action = model.describePod
		case "logs":
			item.Action = model.viewLogs
		case "logs-follow":
			item.Action = model.followLogs
		case "exec":
			item.Action = model.execShell
		case "port-forward":
			item.Action = model.portForward
		case "env":
			item.Action = model.manageEnv
		case "restart":
			item.Action = model.restartPod
		case "delete":
			item.Action = model.deletePod
		case "back":
			item.Action = func() tea.Cmd { return tea.Quit }
		}
	}

	return model, model.loadPod
}

// Init initializes the model
func (m *PodActionsModel) Init() tea.Cmd {
	return tea.Batch(
		m.menu.Init(),
		m.loadPod,
	)
}

// Update handles updates
func (m *PodActionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "b":
			// Quick back navigation
			return m, tea.Quit
		}

		// Handle menu selection by ID
		if msg.String() == "enter" || msg.String() == " " {
			selected := m.menu.GetSelected()
			if selected != nil {
				return m.handleAction(selected.ID)
			}
		}

		// Handle shortcut keys
		for _, item := range m.menu.Items {
			if item.Shortcut == msg.String() {
				return m.handleAction(item.ID)
			}
		}

	case podLoadedMsg:
		m.pod = msg.pod
		m.loading = false
		// Update menu title with pod status
		if m.pod != nil {
			status := services.GetPodStatus(m.pod)
			statusIcon := "⚪"
			switch status {
			case "Running":
				statusIcon = "🟢"
			case "Pending":
				statusIcon = "🟡"
			case "Failed", "Error", "CrashLoopBackOff":
				statusIcon = "🔴"
			case "Completed":
				statusIcon = "✅"
			}
			m.menu.Title = fmt.Sprintf("%s Pod: %s (%s %s)", statusIcon, m.name, status, m.namespace)
		}
		return m, nil
	}

	// Update menu
	newMenu, cmd := m.menu.Update(msg)
	if menu, ok := newMenu.(components.Menu); ok {
		m.menu = &menu
	}
	return m, cmd
}

// View renders the view
func (m *PodActionsModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		loadingView := components.NewLoadingScreen("Loading Pod Details")
		return loadingView.View()
	}

	// Show menu with pod info in the description
	return m.menu.View()
}

// handleAction handles menu item selection
func (m *PodActionsModel) handleAction(actionID string) (tea.Model, tea.Cmd) {
	switch actionID {
	case "describe":
		return m, m.describePod()
	case "logs":
		return m, m.viewLogs()
	case "logs-follow":
		return m, m.followLogs()
	case "exec":
		return m, m.execShell()
	case "port-forward":
		return m, m.portForward()
	case "env":
		return m, m.manageEnv()
	case "restart":
		return m, m.restartPod()
	case "delete":
		return m, m.deletePod()
	case "back":
		return m, tea.Quit
	}
	return m, nil
}

// Pod action implementations

type podLoadedMsg struct {
	pod *corev1.Pod
	err error
}

func (m *PodActionsModel) loadPod() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := m.client.Clientset.CoreV1().Pods(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
	if err != nil {
		return podLoadedMsg{err: err}
	}

	return podLoadedMsg{pod: pod}
}

func (m *PodActionsModel) describePod() tea.Cmd {
	return tea.ExecProcess(exec.Command("kubectl", "describe", "pod", m.name, "-n", m.namespace), func(err error) tea.Msg {
		if err != nil {
			return components.ErrorMsg{Error: err}
		}
		return nil
	})
}

func (m *PodActionsModel) viewLogs() tea.Cmd {
	return tea.ExecProcess(exec.Command("kubectl", "logs", m.name, "-n", m.namespace, "--tail=100"), func(err error) tea.Msg {
		if err != nil {
			return components.ErrorMsg{Error: err}
		}
		return nil
	})
}

func (m *PodActionsModel) followLogs() tea.Cmd {
	return tea.ExecProcess(exec.Command("kubectl", "logs", "-f", m.name, "-n", m.namespace), func(err error) tea.Msg {
		if err != nil {
			return components.ErrorMsg{Error: err}
		}
		return nil
	})
}

func (m *PodActionsModel) execShell() tea.Cmd {
	return func() tea.Msg {
		// Check for multiple containers
		containerName := ""
		if m.pod != nil && len(m.pod.Spec.Containers) > 1 {
			fmt.Println("Multiple containers found:")
			for i, c := range m.pod.Spec.Containers {
				fmt.Printf("  %d. %s\n", i+1, c.Name)
			}
			fmt.Print("Select container (1-", len(m.pod.Spec.Containers), "): ")
			
			var choice int
			fmt.Scanln(&choice)
			if choice > 0 && choice <= len(m.pod.Spec.Containers) {
				containerName = m.pod.Spec.Containers[choice-1].Name
			}
		}
		
		args := []string{"exec", "-it", m.name, "-n", m.namespace}
		if containerName != "" {
			args = append(args, "-c", containerName)
		}
		args = append(args, "--", "/bin/bash")
		
		cmd := exec.Command("kubectl", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		
		// Try bash first, fall back to sh
		if err := cmd.Run(); err != nil {
			args[len(args)-1] = "/bin/sh"
			cmd = exec.Command("kubectl", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			cmd.Run()
		}
		
		return nil
	}
}

func (m *PodActionsModel) portForward() tea.Cmd {
	return func() tea.Msg {
		var localPort, remotePort int
		
		fmt.Print("Enter local port: ")
		fmt.Scanln(&localPort)
		
		fmt.Print("Enter pod port: ")
		fmt.Scanln(&remotePort)
		
		fmt.Printf("Port forwarding %d:%d... Press Ctrl+C to stop\n", localPort, remotePort)
		
		cmd := exec.Command("kubectl", "port-forward", 
			fmt.Sprintf("pod/%s", m.name),
			fmt.Sprintf("%d:%d", localPort, remotePort),
			"-n", m.namespace)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
		
		return nil
	}
}

func (m *PodActionsModel) manageEnv() tea.Cmd {
	// TODO: Implement environment variable management
	return nil
}

func (m *PodActionsModel) restartPod() tea.Cmd {
	return func() tea.Msg {
		// TODO: Show confirmation dialog
		// dialog := components.NewConfirmDialog(
		//     "Restart Pod",
		//     fmt.Sprintf("Are you sure you want to restart pod '%s'?", m.name),
		// )
		
		// TODO: Show dialog and handle confirmation
		
		// For now, just delete the pod
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		gracePeriod := int64(30)
		err := m.client.Clientset.CoreV1().Pods(m.namespace).Delete(ctx, m.name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		
		if err != nil {
			fmt.Printf("Error restarting pod: %v\n", err)
		} else {
			fmt.Println("Pod restart initiated successfully")
		}
		
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		
		return nil
	}
}

func (m *PodActionsModel) deletePod() tea.Cmd {
	return func() tea.Msg {
		// Show confirmation dialog
		fmt.Printf("Are you sure you want to delete pod '%s'? [y/N]: ", m.name)
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Deletion cancelled")
			time.Sleep(1 * time.Second)
			return nil
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		gracePeriod := int64(0)
		err := m.client.Clientset.CoreV1().Pods(m.namespace).Delete(ctx, m.name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		
		if err != nil {
			fmt.Printf("Error deleting pod: %v\n", err)
		} else {
			fmt.Println("Pod deleted successfully")
		}
		
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		
		return tea.Quit
	}
}