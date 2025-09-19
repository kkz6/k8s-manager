# Enhanced UI Features for K8s Manager

## 🚀 Overview

The K8s Manager now includes a comprehensive, reusable UI interface with enhanced navigation features including:
- Number-based quick navigation (press 1-9 to jump to items)
- Improved visual styling with better spacing and colors
- Consistent navigation patterns across all views
- Enhanced keyboard shortcuts
- Better status indicators and icons

## 🎮 Key Features

### 1. Common UI Components (`pkg/ui/common.go`)

The new common UI package provides:
- **Reusable Styles**: Consistent styling across the application
- **List Component**: Scrollable lists with multi-select support
- **Menu Component**: Number-based navigation menus
- **Navigation Keys**: Standardized key bindings
- **Loading Spinners**: Consistent loading indicators
- **Message Rendering**: Success, error, and info messages

### 2. Enhanced Pods View (`pkg/ui/pods_enhanced.go`)

Features:
- **Number Navigation**: Press 1-9 to quickly select pods
- **Enhanced Filtering**: Real-time search with visual feedback
- **Quick Actions**:
  - `d` - Delete pod
  - `l` - View logs
  - `x` - Exec into pod
  - `r` - Restart pod
- **Better Visual Indicators**:
  - ✅ Running pods
  - ⏳ Pending pods
  - ❌ Failed pods
  - 🔄 Terminating pods

### 3. Enhanced Pod Actions (`pkg/ui/pod_actions_enhanced.go`)

Nine quick actions accessible via number keys:
1. **Describe Pod** - Show detailed information
2. **View Logs** - View recent pod logs
3. **Follow Logs** - Stream logs in real-time
4. **Execute Shell** - Open interactive shell
5. **Port Forward** - Forward ports to localhost
6. **Resource Usage** - View CPU/memory metrics
7. **Edit Pod** - Edit pod configuration
8. **Restart Pod** - Delete and recreate
9. **Delete Pod** - Permanently remove

## 🎯 Usage

### Interactive Mode
```bash
# Launch interactive mode
./k8s-manager -i

# Or directly to pods view
./k8s-manager pods list -i
```

### Navigation Keys

#### Universal Keys
- `↑/k`, `↓/j` - Navigate up/down
- `PgUp/b`, `PgDn/f` - Page up/down
- `Home/g`, `End/G` - Jump to first/last
- `1-9` - Quick select by number
- `Enter/Space` - Select item
- `/` - Search/filter
- `?/h` - Show help
- `Esc/Backspace` - Go back
- `q/Ctrl+C` - Quit

#### Pod-Specific Keys
- `d` - Quick delete
- `l` - View logs
- `x` - Exec shell
- `r` - Restart pod
- `R/F5` - Refresh list

## 🎨 Visual Enhancements

### Status Colors
- **Green** (42): Running, Ready, Success
- **Yellow** (214): Pending, Waiting
- **Red** (196): Failed, Error, CrashLoopBackOff
- **Blue** (86): Information, Headers
- **Purple** (170): Selected items

### Layout Components
- **Title Bar**: Centered with subtitle support
- **Content Boxes**: Rounded borders with padding
- **List Views**: Scrollable with indicators
- **Help Section**: Context-sensitive help
- **Message Boxes**: Color-coded status messages

## 📦 Architecture

The enhanced UI follows a modular architecture:

```
pkg/ui/
├── common.go           # Shared UI components and styles
├── pods_enhanced.go    # Enhanced pods list view
├── pod_actions_enhanced.go # Enhanced pod actions menu
└── app.go             # Main application entry point
```

### Key Components

1. **List Component**
   - Handles scrolling and selection
   - Supports multi-select mode
   - Number-based navigation
   - Search/filter capability

2. **Menu Component**
   - Quick number navigation
   - Icon support
   - Description text
   - Handler functions

3. **Navigation Keys**
   - Standardized key bindings
   - Help text generation
   - Customizable shortcuts

## 🔧 Customization

### Adding New Menu Items
```go
menuItems := []MenuItem{
    {
        Title:       "Your Action",
        Description: "Action description",
        Icon:        "🎯",
        Number:      1,
        Handler:     yourHandlerFunc,
    },
}
```

### Custom Styles
```go
CustomStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("86")).
    Bold(true).
    Border(lipgloss.RoundedBorder())
```

### Custom List Items
```go
items := []ListItem{
    {
        ID:          "item-1",
        Title:       "Item Title",
        Status:      "running",
        Description: "Item description",
        Icon:        "📦",
        Details: map[string]string{
            "Key": "Value",
        },
    },
}
```

## 🚦 Status Indicators

The UI uses consistent status indicators:
- **Icons**: Visual representation of resource types
- **Colors**: Status-based color coding
- **Badges**: Numeric indicators (Ready: 2/2)
- **Progress**: Loading spinners and animations

## 📱 Responsive Design

The UI adapts to terminal size:
- Dynamic list height adjustment
- Scrollable content areas
- Responsive column widths
- Adaptive help text

## 🎉 Benefits

1. **Improved Usability**: Number navigation makes selection faster
2. **Better Visibility**: Enhanced colors and icons improve readability
3. **Consistency**: Common UI components ensure consistent UX
4. **Accessibility**: Multiple navigation methods (arrows, vim keys, numbers)
5. **Extensibility**: Easy to add new features using common components

## 🔜 Future Enhancements

Potential future improvements:
- Mouse support for click navigation
- Configurable color themes
- Export/import UI preferences
- Command palette with fuzzy search
- Split pane views
- Resource graphs and charts

## 📝 Notes

- The enhanced UI is fully backward compatible
- All original CLI commands still work
- Interactive mode (`-i` flag) enables the enhanced UI
- The UI gracefully degrades on terminals without color support