package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatAge(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "30 seconds ago",
			time:     now.Add(-30 * time.Second),
			expected: "30s",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m",
		},
		{
			name:     "2 hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2h",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-72 * time.Hour),
			expected: "3d",
		},
		{
			name:     "1 second ago",
			time:     now.Add(-1 * time.Second),
			expected: "1s",
		},
		{
			name:     "59 seconds ago",
			time:     now.Add(-59 * time.Second),
			expected: "59s",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1m",
		},
		{
			name:     "59 minutes ago",
			time:     now.Add(-59 * time.Minute),
			expected: "59m",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1h",
		},
		{
			name:     "23 hours ago",
			time:     now.Add(-23 * time.Hour),
			expected: "23h",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1d",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatAge(tc.time)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatAgeEdgeCases(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name        string
		time        time.Time
		description string
	}{
		{
			name:        "future time",
			time:        now.Add(1 * time.Hour),
			description: "should handle future times gracefully",
		},
		{
			name:        "exactly now",
			time:        now,
			description: "should handle current time",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatAge(tc.time)
			// For edge cases, we just check that we get a string result
			// The exact value may vary due to timing, but it should be a valid format
			assert.Regexp(t, `^\d+[smhd]$`, result, tc.description)
		})
	}
}
