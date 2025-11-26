package views

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddSecretKeyModel handles adding a new key to a secret
type AddSecretKeyModel struct {
	namespace string
	name      string
	form      tea.Model
	saving    bool
	errorMsg  string
}

// NewAddSecretKeyModel creates a new add secret key model
func NewAddSecretKeyModel(namespace, name string) *AddSecretKeyModel {
	// Create form fields
	keyField := components.NewInputField("Key")
	keyField.Placeholder = "e.g., API_KEY"
	keyField.CharLimit = 64
	keyField.Focus()

	valueField := components.NewInputField("Value")
	valueField.Placeholder = "e.g., your-secret-value"
	valueField.CharLimit = 1024

	fields := []*components.InputField{keyField, valueField}
	form := components.NewForm(fmt.Sprintf("Add Key to Secret: %s", name), fields)

	return &AddSecretKeyModel{
		namespace: namespace,
		name:      name,
		form:      form,
	}
}

// Init initializes the model
func (m *AddSecretKeyModel) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles updates
func (m *AddSecretKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Cancel and go back
			return m, Navigate(ViewSecretDetail, map[string]string{
				"namespace": m.namespace,
				"name":      m.name,
			})
		}

	case secretKeyAddedMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.saving = false
			return m, nil
		}
		// Success - go back to secret detail
		return m, Navigate(ViewSecretDetail, map[string]string{
			"namespace": m.namespace,
			"name":      m.name,
		})
	}

	// Check if form was submitted
	formModel, cmd := m.form.Update(msg)
	m.form = formModel
	
	if form, ok := formModel.(components.FormModel); ok {
		if form.IsSubmitted() {
			values := form.GetValues()
			key := values["Key"]
			value := values["Value"]
			
			if key != "" && value != "" {
				m.saving = true
				return m, m.addKey(key, value)
			} else {
				m.errorMsg = "Please fill in both key and value fields"
				return m, nil
			}
		}
	}

	return m, cmd
}

// View renders the view
func (m *AddSecretKeyModel) View() string {
	if m.saving {
		return components.NewLoadingScreen("Adding Secret Key...").View()
	}

	view := m.form.View()
	
	if m.errorMsg != "" {
		errorBox := components.ErrorMessageStyle.Render(m.errorMsg)
		view += "\n\n" + errorBox
	}

	return view
}

// addKey adds a new key to the secret
func (m *AddSecretKeyModel) addKey(key, value string) tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return secretKeyAddedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get the current secret
		secret, err := client.Clientset.CoreV1().Secrets(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return secretKeyAddedMsg{err: err}
		}

		// Check if key already exists
		if _, exists := secret.Data[key]; exists {
			return secretKeyAddedMsg{err: fmt.Errorf("key '%s' already exists", key)}
		}

		// Add the new key (base64 encode the value)
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[key] = []byte(value)

		// Update the secret
		_, err = client.Clientset.CoreV1().Secrets(m.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return secretKeyAddedMsg{err: err}
		}

		return secretKeyAddedMsg{}
	}
}

// secretKeyAddedMsg is sent when a key is added
type secretKeyAddedMsg struct {
	err error
}

// EditSecretKeyModel handles editing an existing key in a secret
type EditSecretKeyModel struct {
	namespace string
	name      string
	key       string
	form      tea.Model
	saving    bool
	errorMsg  string
}

// NewEditSecretKeyModel creates a new edit secret key model
func NewEditSecretKeyModel(namespace, name, key string) *EditSecretKeyModel {
	// Create form field for value only (key is fixed)
	valueField := components.NewInputField("Value")
	valueField.Placeholder = "Loading current value..."
	valueField.CharLimit = 1024
	valueField.Focus()

	fields := []*components.InputField{valueField}
	form := components.NewForm(fmt.Sprintf("Edit Key '%s' in Secret: %s", key, name), fields)

	return &EditSecretKeyModel{
		namespace: namespace,
		name:      name,
		key:       key,
		form:      form,
	}
}

// Init initializes the model
func (m *EditSecretKeyModel) Init() tea.Cmd {
	return tea.Batch(
		m.form.Init(),
		m.loadCurrentValue(),
	)
}

// Update handles updates
func (m *EditSecretKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Cancel and go back
			return m, Navigate(ViewSecretDetail, map[string]string{
				"namespace": m.namespace,
				"name":      m.name,
			})
		}

	case secretKeyUpdatedMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.saving = false
			return m, nil
		}
		// Success - go back to secret detail
		return m, Navigate(ViewSecretDetail, map[string]string{
			"namespace": m.namespace,
			"name":      m.name,
		})

	case secretValueLoadedMsg:
		if msg.err == nil {
			// Set the current value in the form
			if form, ok := m.form.(components.FormModel); ok {
				form.SetFieldValue("Value", msg.value)
			}
		}
	}

	// Check if form was submitted
	formModel, cmd := m.form.Update(msg)
	m.form = formModel

	if form, ok := formModel.(components.FormModel); ok {
		if form.IsSubmitted() {
			values := form.GetValues()
			value := values["Value"]

			if value != "" {
				m.saving = true
				return m, m.updateKey(value)
			} else {
				m.errorMsg = "Please enter a value"
				return m, nil
			}
		}
	}

	return m, cmd
}

// View renders the view
func (m *EditSecretKeyModel) View() string {
	if m.saving {
		return components.NewLoadingScreen("Updating Secret Key...").View()
	}

	view := m.form.View()

	if m.errorMsg != "" {
		errorBox := components.ErrorMessageStyle.Render(m.errorMsg)
		view += "\n\n" + errorBox
	}

	return view
}

// loadCurrentValue loads the current value of the key
func (m *EditSecretKeyModel) loadCurrentValue() tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return secretValueLoadedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		secret, err := client.Clientset.CoreV1().Secrets(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return secretValueLoadedMsg{err: err}
		}

		if value, exists := secret.Data[m.key]; exists {
			return secretValueLoadedMsg{value: string(value)}
		}

		return secretValueLoadedMsg{err: fmt.Errorf("key not found")}
	}
}

// updateKey updates an existing key in the secret
func (m *EditSecretKeyModel) updateKey(value string) tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return secretKeyUpdatedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get the current secret
		secret, err := client.Clientset.CoreV1().Secrets(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return secretKeyUpdatedMsg{err: err}
		}

		// Check if key exists
		if _, exists := secret.Data[m.key]; !exists {
			return secretKeyUpdatedMsg{err: fmt.Errorf("key '%s' not found", m.key)}
		}

		// Update the key value
		secret.Data[m.key] = []byte(value)

		// Update the secret
		_, err = client.Clientset.CoreV1().Secrets(m.namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return secretKeyUpdatedMsg{err: err}
		}

		return secretKeyUpdatedMsg{}
	}
}

// secretKeyUpdatedMsg is sent when a key is updated
type secretKeyUpdatedMsg struct {
	err error
}

// secretValueLoadedMsg is sent when the current value is loaded
type secretValueLoadedMsg struct {
	value string
	err   error
}

// Similar model for ConfigMap
type AddConfigMapKeyModel struct {
	namespace string
	name      string
	form      tea.Model
	saving    bool
	errorMsg  string
}

// NewAddConfigMapKeyModel creates a new add configmap key model
func NewAddConfigMapKeyModel(namespace, name string) *AddConfigMapKeyModel {
	// Create form fields
	keyField := components.NewInputField("Key")
	keyField.Placeholder = "e.g., app.properties"
	keyField.CharLimit = 64
	keyField.Focus()

	valueField := components.NewInputField("Value")
	valueField.Placeholder = "Paste your configuration content here"
	valueField.CharLimit = 10240 // Allow larger content for config files

	fields := []*components.InputField{keyField, valueField}
	form := components.NewForm(fmt.Sprintf("Add Key to ConfigMap: %s", name), fields)

	return &AddConfigMapKeyModel{
		namespace: namespace,
		name:      name,
		form:      form,
	}
}

// Init initializes the model
func (m *AddConfigMapKeyModel) Init() tea.Cmd {
	return m.form.Init()
}

// Update handles updates
func (m *AddConfigMapKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Cancel and go back
			return m, Navigate(ViewConfigMapDetail, map[string]string{
				"namespace": m.namespace,
				"name":      m.name,
			})
		}

	case configMapKeyAddedMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.saving = false
			return m, nil
		}
		// Success - go back to configmap detail
		return m, Navigate(ViewConfigMapDetail, map[string]string{
			"namespace": m.namespace,
			"name":      m.name,
		})
	}

	// Check if form was submitted
	formModel, cmd := m.form.Update(msg)
	m.form = formModel
	
	if form, ok := formModel.(components.FormModel); ok {
		if form.IsSubmitted() {
			values := form.GetValues()
			key := values["Key"]
			value := values["Value"]
			
			if key != "" && value != "" {
				m.saving = true
				return m, m.addKey(key, value)
			} else {
				m.errorMsg = "Please fill in both key and value fields"
				return m, nil
			}
		}
	}

	return m, cmd
}

// View renders the view
func (m *AddConfigMapKeyModel) View() string {
	if m.saving {
		return components.NewLoadingScreen("Adding ConfigMap Key...").View()
	}

	view := m.form.View()
	
	if m.errorMsg != "" {
		errorBox := components.ErrorMessageStyle.Render(m.errorMsg)
		view += "\n\n" + errorBox
	}

	return view
}

// addKey adds a new key to the configmap
func (m *AddConfigMapKeyModel) addKey(key, value string) tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return configMapKeyAddedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get the current configmap
		cm, err := client.Clientset.CoreV1().ConfigMaps(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return configMapKeyAddedMsg{err: err}
		}

		// Check if key already exists
		if _, exists := cm.Data[key]; exists {
			return configMapKeyAddedMsg{err: fmt.Errorf("key '%s' already exists", key)}
		}

		// Add the new key
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[key] = value

		// Update the configmap
		_, err = client.Clientset.CoreV1().ConfigMaps(m.namespace).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			return configMapKeyAddedMsg{err: err}
		}

		return configMapKeyAddedMsg{}
	}
}

// configMapKeyAddedMsg is sent when a key is added
type configMapKeyAddedMsg struct {
	err error
}

// EditConfigMapKeyModel handles editing an existing key in a configmap
type EditConfigMapKeyModel struct {
	namespace string
	name      string
	key       string
	form      tea.Model
	saving    bool
	errorMsg  string
}

// NewEditConfigMapKeyModel creates a new edit configmap key model
func NewEditConfigMapKeyModel(namespace, name, key string) *EditConfigMapKeyModel {
	// Create form field for value only (key is fixed)
	valueField := components.NewInputField("Value")
	valueField.Placeholder = "Loading current value..."
	valueField.CharLimit = 10240
	valueField.Focus()

	fields := []*components.InputField{valueField}
	form := components.NewForm(fmt.Sprintf("Edit Key '%s' in ConfigMap: %s", key, name), fields)

	return &EditConfigMapKeyModel{
		namespace: namespace,
		name:      name,
		key:       key,
		form:      form,
	}
}

// Init initializes the model
func (m *EditConfigMapKeyModel) Init() tea.Cmd {
	return tea.Batch(
		m.form.Init(),
		m.loadCurrentValue(),
	)
}

// Update handles updates
func (m *EditConfigMapKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Cancel and go back
			return m, Navigate(ViewConfigMapDetail, map[string]string{
				"namespace": m.namespace,
				"name":      m.name,
			})
		}

	case configMapKeyUpdatedMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.saving = false
			return m, nil
		}
		// Success - go back to configmap detail
		return m, Navigate(ViewConfigMapDetail, map[string]string{
			"namespace": m.namespace,
			"name":      m.name,
		})

	case configMapValueLoadedMsg:
		if msg.err == nil {
			// Set the current value in the form
			if form, ok := m.form.(components.FormModel); ok {
				form.SetFieldValue("Value", msg.value)
			}
		}
	}

	// Check if form was submitted
	formModel, cmd := m.form.Update(msg)
	m.form = formModel

	if form, ok := formModel.(components.FormModel); ok {
		if form.IsSubmitted() {
			values := form.GetValues()
			value := values["Value"]

			if value != "" {
				m.saving = true
				return m, m.updateKey(value)
			} else {
				m.errorMsg = "Please enter a value"
				return m, nil
			}
		}
	}

	return m, cmd
}

// View renders the view
func (m *EditConfigMapKeyModel) View() string {
	if m.saving {
		return components.NewLoadingScreen("Updating ConfigMap Key...").View()
	}

	view := m.form.View()

	if m.errorMsg != "" {
		errorBox := components.ErrorMessageStyle.Render(m.errorMsg)
		view += "\n\n" + errorBox
	}

	return view
}

// loadCurrentValue loads the current value of the key
func (m *EditConfigMapKeyModel) loadCurrentValue() tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return configMapValueLoadedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cm, err := client.Clientset.CoreV1().ConfigMaps(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return configMapValueLoadedMsg{err: err}
		}

		if value, exists := cm.Data[m.key]; exists {
			return configMapValueLoadedMsg{value: value}
		}

		return configMapValueLoadedMsg{err: fmt.Errorf("key not found")}
	}
}

// updateKey updates an existing key in the configmap
func (m *EditConfigMapKeyModel) updateKey(value string) tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return configMapKeyUpdatedMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get the current configmap
		cm, err := client.Clientset.CoreV1().ConfigMaps(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
		if err != nil {
			return configMapKeyUpdatedMsg{err: err}
		}

		// Check if key exists
		if _, exists := cm.Data[m.key]; !exists {
			return configMapKeyUpdatedMsg{err: fmt.Errorf("key '%s' not found", m.key)}
		}

		// Update the key value
		cm.Data[m.key] = value

		// Update the configmap
		_, err = client.Clientset.CoreV1().ConfigMaps(m.namespace).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			return configMapKeyUpdatedMsg{err: err}
		}

		return configMapKeyUpdatedMsg{}
	}
}

// configMapKeyUpdatedMsg is sent when a key is updated
type configMapKeyUpdatedMsg struct {
	err error
}

// configMapValueLoadedMsg is sent when the current value is loaded
type configMapValueLoadedMsg struct {
	value string
	err   error
}