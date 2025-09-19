package utils

import (
	"fmt"
	"time"
)

// FormatAge formats a time duration into a human-readable age string
func FormatAge(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	// Handle future times or negative durations
	if duration < 0 {
		return "0s"
	}

	if duration < time.Minute {
		seconds := int(duration.Seconds())
		if seconds < 0 {
			seconds = 0
		}
		return fmt.Sprintf("%ds", seconds)
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}
