# DevTools-Style UI for K8s Manager

## 🚀 Overview

The K8s Manager now features a **DevTools Manager-style UI** that closely mimics the clean, minimalist interface shown in your reference image. The key improvement is **instant number selection** - just press a number key (1-9) to immediately select and execute that option without needing to press Enter.

## ✨ Key Features

### Instant Number Selection
- **No Enter Required**: Press 1-9 to instantly select an option
- **Visual Feedback**: Clean numbered list with descriptions
- **Fast Navigation**: Jump directly to any option

### Clean Visual Design
```
🚀 K8s Manager by Karthick

1. Pods Manager
   List, manage, and interact with Kubernetes pods
2. Deployments
   Manage Kubernetes deployments and rollouts
3. Services
   View and manage Kubernetes services
4. ConfigMaps & Secrets
   Manage configuration and secret resources
5. Namespaces
   Switch and manage Kubernetes namespaces
6. Nodes & Cluster
   View cluster nodes and resource usage
7. Logs & Events
   View pod logs and cluster events
8. Configuration
   Manage K8s Manager settings and contexts
9. Exit
   Quit the application

↑/k up • ↓/j down • 1-9 quick select • enter select • q quit
```

## 🎮 Navigation

### Main Menu
- **1-9**: Instant selection of menu items
- **↑/k**: Move up
- **↓/j**: Move down
- **Enter**: Select current item (alternative to numbers)
- **q**: Quit application

### Pods View
```
📦 Kubernetes Pods

1. nginx-deployment-7fb96c846b-kxjtm [Running]
   Ready: 1/1, Restarts: 0, Age: 2d
2. redis-master-6b54579d85-qzm8x [Running]
   Ready: 1/1, Restarts: 2, Age: 5d
3. web-app-8566784f97-zh9pc [Pending]
   Ready: 0/1, Restarts: 0, Age: 10m

↑/k up • ↓/j down • 1-9 quick select • enter select • / filter • d delete • l logs • r refresh • q quit
```

- **1-9**: Instantly select pod (first 9 pods)
- **/**: Filter pods
- **d**: Delete selected pod
- **l**: View logs
- **r**: Refresh list

## 🎯 Usage Examples

### Quick Pod Management
```bash
# Launch the app
./k8s-manager -i

# Press '1' to instantly go to Pods Manager
# Press '3' to instantly select the third pod
# Press 'd' to delete it
```

### Fast Navigation Flow
1. Launch: `./k8s-manager -i`
2. Press `1` → Instantly opens Pods Manager
3. Press `2` → Instantly selects second pod
4. Press `l` → Views logs

No Enter key required at any step!

## 🎨 Design Principles

### Minimalist Interface
- Clean, uncluttered layout
- Clear visual hierarchy
- Consistent spacing
- Subtle color coding

### Color Scheme
- **Blue (#86)**: Numbers and selections
- **Gray (#244)**: Descriptions and help text
- **Green (#42)**: Running/Success status
- **Yellow (#214)**: Pending/Warning status
- **Red (#196)**: Error/Failed status

### Typography
- **Bold Numbers**: Easy to identify options
- **Indented Descriptions**: Clear hierarchy
- **Consistent Spacing**: 2-space indentation

## 📁 Architecture

```
pkg/ui/
├── devtools_menu.go      # Main menu with instant selection
├── devtools_pods.go      # Pods view with number navigation
├── devtools_app.go       # Application flow controller
└── common.go             # Shared styles and components
```

## 🔧 Customization

### Adding New Menu Items
```go
items := []DevToolsMenuItem{
    {
        Number:      "1",
        Title:       "Your Feature",
        Description: "Feature description",
        Action: func() tea.Cmd {
            return func() tea.Msg {
                return "feature_key"
            }
        },
    },
}
```

### Styling
```go
// Clean, minimal styles
devToolsNumberStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("86")).
    Bold(true)

devToolsItemStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("252"))
```

## 🚦 Status Indicators

- `[Running]` - Green, pod is healthy
- `[Pending]` - Yellow, pod is starting
- `[Failed]` - Red, pod has errors
- `[Terminating]` - Yellow, pod is shutting down

## 💡 Benefits

1. **Speed**: Instant selection without Enter key
2. **Clarity**: Clean, focused interface
3. **Efficiency**: Number keys for rapid navigation
4. **Consistency**: Uniform design across all views
5. **Simplicity**: Minimal visual noise

## 🎉 Comparison with Original

### Before (Traditional Menu)
1. Navigate with arrows
2. Press Enter to select
3. Multiple keystrokes required

### After (DevTools Style)
1. Press number key
2. Instantly executed
3. Single keystroke operation

## 📝 Notes

- First 9 items get number shortcuts (1-9)
- Additional items accessible via arrow navigation
- All shortcuts work without modifier keys
- Interface adapts to terminal size
- Gracefully degrades on limited terminals

## 🔜 Future Enhancements

- Support for 0 key (10th item)
- Two-digit number support (10-99)
- Customizable number assignments
- Vim-style marks and jumps
- Command palette with fuzzy finding

---

The DevTools-style UI provides a **fast, clean, and efficient** interface that makes Kubernetes management as simple as pressing a number key!