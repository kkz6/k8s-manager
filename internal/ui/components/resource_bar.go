package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ResourceBarStyle represents the visual style of a resource bar
type ResourceBarStyle struct {
	Width          int
	ShowPercentage bool
	ShowValues     bool
	Compact        bool
}

// DefaultResourceBarStyle returns the default resource bar style
func DefaultResourceBarStyle() ResourceBarStyle {
	return ResourceBarStyle{
		Width:          20,
		ShowPercentage: true,
		ShowValues:     true,
		Compact:        false,
	}
}

// CompactResourceBarStyle returns a compact resource bar style
func CompactResourceBarStyle() ResourceBarStyle {
	return ResourceBarStyle{
		Width:          16,
		ShowPercentage: true,
		ShowValues:     false,
		Compact:        true,
	}
}

// UsageLevel represents the usage level for color coding
type UsageLevel int

const (
	UsageLow      UsageLevel = iota // 0-50%
	UsageMedium                     // 50-75%
	UsageHigh                       // 75-90%
	UsageCritical                   // 90%+
)

// GetUsageLevel returns the usage level based on percentage
func GetUsageLevel(percent float64) UsageLevel {
	switch {
	case percent >= 90:
		return UsageCritical
	case percent >= 75:
		return UsageHigh
	case percent >= 50:
		return UsageMedium
	default:
		return UsageLow
	}
}

// Color definitions for resource bars
var (
	ColorLow      = lipgloss.Color("42")  // Green
	ColorMedium   = lipgloss.Color("226") // Yellow
	ColorHigh     = lipgloss.Color("214") // Orange
	ColorCritical = lipgloss.Color("196") // Red
	ColorEmpty    = lipgloss.Color("240") // Gray
	ColorLabel    = lipgloss.Color("252") // Light gray
)

// GetUsageColor returns the color for a given usage level
func GetUsageColor(level UsageLevel) lipgloss.Color {
	switch level {
	case UsageCritical:
		return ColorCritical
	case UsageHigh:
		return ColorHigh
	case UsageMedium:
		return ColorMedium
	default:
		return ColorLow
	}
}

// RenderResourceBar renders a single resource bar
func RenderResourceBar(label string, percent float64, used, total string, style ResourceBarStyle) string {
	// Clamp percentage to 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	level := GetUsageLevel(percent)
	barColor := GetUsageColor(level)

	// Calculate filled and empty portions
	filledWidth := int(float64(style.Width) * percent / 100)
	emptyWidth := style.Width - filledWidth

	// Build the bar
	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorEmpty)

	filled := filledStyle.Render(strings.Repeat("█", filledWidth))
	empty := emptyStyle.Render(strings.Repeat("░", emptyWidth))

	bar := filled + empty

	// Build the label
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorLabel).
		Width(4).
		Align(lipgloss.Right)

	// Build percentage
	percentStyle := lipgloss.NewStyle().
		Foreground(barColor).
		Bold(true).
		Width(5).
		Align(lipgloss.Right)

	var result strings.Builder

	if style.Compact {
		// Compact format: CPU: ████████░░░░ 52%
		result.WriteString(labelStyle.Render(label + ":"))
		result.WriteString(" ")
		result.WriteString(bar)
		if style.ShowPercentage {
			result.WriteString(percentStyle.Render(fmt.Sprintf("%3.0f%%", percent)))
		}
	} else {
		// Full format with values
		result.WriteString(labelStyle.Render(label + ":"))
		result.WriteString(" ")
		result.WriteString(bar)
		if style.ShowPercentage {
			result.WriteString(percentStyle.Render(fmt.Sprintf("%3.0f%%", percent)))
		}
		if style.ShowValues && used != "" && total != "" {
			valueStyle := lipgloss.NewStyle().Foreground(ColorMuted)
			result.WriteString(valueStyle.Render(fmt.Sprintf("  %s / %s", used, total)))
		}
	}

	return result.String()
}

// RenderDualResourceBar renders CPU and Memory bars side by side (for compact view)
func RenderDualResourceBar(cpuPercent, memPercent float64, style ResourceBarStyle) string {
	cpuLevel := GetUsageLevel(cpuPercent)
	memLevel := GetUsageLevel(memPercent)

	cpuColor := GetUsageColor(cpuLevel)
	memColor := GetUsageColor(memLevel)

	// Smaller bars for dual view
	barWidth := 12
	cpuFilled := int(float64(barWidth) * cpuPercent / 100)
	memFilled := int(float64(barWidth) * memPercent / 100)

	if cpuFilled > barWidth {
		cpuFilled = barWidth
	}
	if memFilled > barWidth {
		memFilled = barWidth
	}

	cpuBar := lipgloss.NewStyle().Foreground(cpuColor).Render(strings.Repeat("█", cpuFilled)) +
		lipgloss.NewStyle().Foreground(ColorEmpty).Render(strings.Repeat("░", barWidth-cpuFilled))

	memBar := lipgloss.NewStyle().Foreground(memColor).Render(strings.Repeat("█", memFilled)) +
		lipgloss.NewStyle().Foreground(ColorEmpty).Render(strings.Repeat("░", barWidth-memFilled))

	cpuPercentStyle := lipgloss.NewStyle().Foreground(cpuColor).Bold(true)
	memPercentStyle := lipgloss.NewStyle().Foreground(memColor).Bold(true)

	labelStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	return fmt.Sprintf("%s %s %s  %s %s %s",
		labelStyle.Render("CPU"),
		cpuBar,
		cpuPercentStyle.Render(fmt.Sprintf("%3.0f%%", cpuPercent)),
		labelStyle.Render("MEM"),
		memBar,
		memPercentStyle.Render(fmt.Sprintf("%3.0f%%", memPercent)),
	)
}

// RenderUsageIndicator renders a small usage indicator icon
func RenderUsageIndicator(percent float64) string {
	level := GetUsageLevel(percent)
	color := GetUsageColor(level)

	var icon string
	switch level {
	case UsageCritical:
		icon = "●" // Full circle - critical
	case UsageHigh:
		icon = "◕" // Three-quarter - high
	case UsageMedium:
		icon = "◑" // Half - medium
	default:
		icon = "◔" // Quarter - low
	}

	return lipgloss.NewStyle().Foreground(color).Render(icon)
}

// RenderSparkline renders a mini sparkline for historical data
func RenderSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return strings.Repeat("▁", width)
	}

	// Find max value
	max := 100.0 // Assume percentage

	// Sparkline characters from low to high
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var result strings.Builder

	// Sample or pad values to match width
	step := float64(len(values)) / float64(width)
	for i := 0; i < width; i++ {
		idx := int(float64(i) * step)
		if idx >= len(values) {
			idx = len(values) - 1
		}

		val := values[idx]
		if val < 0 {
			val = 0
		}
		if val > max {
			val = max
		}

		// Map value to character
		charIdx := int(val / max * float64(len(chars)-1))
		if charIdx >= len(chars) {
			charIdx = len(chars) - 1
		}

		level := GetUsageLevel(val)
		color := GetUsageColor(level)
		result.WriteString(lipgloss.NewStyle().Foreground(color).Render(string(chars[charIdx])))
	}

	return result.String()
}

// RenderClusterSummaryBox renders a summary box with cluster totals
func RenderClusterSummaryBox(title string, cpuPercent, memPercent float64, cpuUsed, cpuTotal, memUsed, memTotal string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorInfo).
		MarginBottom(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	style := DefaultResourceBarStyle()
	style.Width = 30

	var content strings.Builder
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")
	content.WriteString(RenderResourceBar("CPU", cpuPercent, cpuUsed, cpuTotal, style))
	content.WriteString("\n")
	content.WriteString(RenderResourceBar("MEM", memPercent, memUsed, memTotal, style))

	return boxStyle.Render(content.String())
}

// RenderPodResourceCard renders a card showing pod resources
func RenderPodResourceCard(name, namespace, status string, cpuPercent, memPercent float64, cpuUsed, cpuTotal, memUsed, memTotal string, selected bool) string {
	// Status color
	var statusColor lipgloss.Color
	switch status {
	case "Running":
		statusColor = ColorSuccess
	case "Pending":
		statusColor = ColorWarning
	default:
		statusColor = ColorError
	}

	// Pod name style
	nameStyle := lipgloss.NewStyle().Bold(true)
	if selected {
		nameStyle = nameStyle.Foreground(ColorHighlight).Background(lipgloss.Color("236"))
	} else {
		nameStyle = nameStyle.Foreground(ColorPrimary)
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	nsStyle := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)

	// Build compact resource bars
	style := CompactResourceBarStyle()

	var card strings.Builder

	// Header line: name + status
	if selected {
		card.WriteString("▸ ")
	} else {
		card.WriteString("  ")
	}
	card.WriteString(nameStyle.Render(truncateString(name, 45)))
	card.WriteString("  ")
	card.WriteString(statusStyle.Render(status))
	card.WriteString("\n")

	// Namespace line
	card.WriteString("    ")
	card.WriteString(nsStyle.Render(namespace))
	card.WriteString("\n")

	// Resource bars
	card.WriteString("    ")
	card.WriteString(RenderResourceBar("CPU", cpuPercent, cpuUsed, cpuTotal, style))
	card.WriteString("\n")
	card.WriteString("    ")
	card.WriteString(RenderResourceBar("MEM", memPercent, memUsed, memTotal, style))

	return card.String()
}

// truncateString truncates a string to max length with ellipsis
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// RenderLegend renders a color legend for resource usage
func RenderLegend() string {
	legendStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	items := []struct {
		color lipgloss.Color
		label string
	}{
		{ColorLow, "0-50% Low"},
		{ColorMedium, "50-75% Medium"},
		{ColorHigh, "75-90% High"},
		{ColorCritical, "90%+ Critical"},
	}

	var parts []string
	for _, item := range items {
		colorBox := lipgloss.NewStyle().Foreground(item.color).Render("██")
		parts = append(parts, colorBox+" "+legendStyle.Render(item.label))
	}

	return strings.Join(parts, "  ")
}
