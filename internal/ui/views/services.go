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

// ServicesViewModel displays Kubernetes services
type ServicesViewModel struct {
	client       *services.K8sClient
	services     []corev1.Service
	list         *components.ListView
	loading      bool
	spinner      components.SpinnerModel
	err          error
	allNamespaces bool
}

// servicesFetchedMsg is sent when services are fetched
type servicesFetchedMsg struct {
	services []corev1.Service
	err      error
}

// NewServicesViewModel creates a new services view model
func NewServicesViewModel() tea.Model {
	client, _ := services.GetK8sClient()
	return &ServicesViewModel{
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading services..."),
		allNamespaces: false,
	}
}

func (m *ServicesViewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.fetchServices)
}

func (m *ServicesViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, m.fetchServices
		case "a":
			m.allNamespaces = !m.allNamespaces
			m.loading = true
			return m, m.fetchServices
		}

	case servicesFetchedMsg:
		m.loading = false
		m.services = msg.services
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
				svc := selected.Data.(*corev1.Service)
				return m, m.showServiceDetails(svc)
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

func (m *ServicesViewModel) View() string {
	if m.loading {
		return components.NewLoadingScreen("Loading Services").View()
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
			components.RenderTitle("Services", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render(helpText),
		)
	}

	if m.list == nil {
		return "No services available"
	}

	return m.list.View()
}

func (m *ServicesViewModel) fetchServices() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := ""
	if !m.allNamespaces {
		namespace = services.GetCurrentNamespace()
	}

	svcList, err := m.client.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return servicesFetchedMsg{err: err}
	}

	return servicesFetchedMsg{services: svcList.Items}
}

func (m *ServicesViewModel) updateList() {
	items := []components.ListItem{}

	for i := range m.services {
		svc := &m.services[i]

		// Determine service type and icon
		svcType := string(svc.Spec.Type)
		icon := "🌐"
		switch svc.Spec.Type {
		case corev1.ServiceTypeClusterIP:
			icon = "🔵"
		case corev1.ServiceTypeNodePort:
			icon = "🟡"
		case corev1.ServiceTypeLoadBalancer:
			icon = "🟢"
		case corev1.ServiceTypeExternalName:
			icon = "🔗"
		}

		// Build ports string
		var ports []string
		for _, port := range svc.Spec.Ports {
			portStr := fmt.Sprintf("%d", port.Port)
			if port.NodePort != 0 {
				portStr = fmt.Sprintf("%d:%d", port.Port, port.NodePort)
			}
			if port.Name != "" {
				portStr = fmt.Sprintf("%s/%s", port.Name, portStr)
			}
			ports = append(ports, portStr)
		}
		portsStr := strings.Join(ports, ", ")
		if portsStr == "" {
			portsStr = "none"
		}

		// Get cluster IP or external IP
		ip := svc.Spec.ClusterIP
		if ip == "None" {
			ip = "Headless"
		}
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer && len(svc.Status.LoadBalancer.Ingress) > 0 {
			if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				ip = svc.Status.LoadBalancer.Ingress[0].IP
			} else if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
				ip = svc.Status.LoadBalancer.Ingress[0].Hostname
			}
		}

		title := svc.Name
		if m.allNamespaces {
			title = fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
		}

		description := fmt.Sprintf("Type: %s | IP: %s | Ports: %s", svcType, ip, portsStr)

		items = append(items, components.ListItem{
			ID:          fmt.Sprintf("%s/%s", svc.Namespace, svc.Name),
			Title:       title,
			Description: description,
			Icon:        icon,
			Data:        svc,
		})
	}

	nsText := services.GetCurrentNamespace()
	if m.allNamespaces {
		nsText = "all namespaces"
	}
	title := fmt.Sprintf("🌐 Services (%d) - %s", len(m.services), nsText)
	m.list = components.NewListView(title, items)
	m.list.SetHelpText("enter: details • a: toggle all-ns • r: refresh • esc/b: back • ctrl+c: quit")
}

func (m *ServicesViewModel) showServiceDetails(svc *corev1.Service) tea.Cmd {
	return func() tea.Msg {
		// For now, just show a message. Can be extended to show detailed view
		return nil
	}
}
