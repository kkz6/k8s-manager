package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// SplashScreen shows a welcome screen with logo
type SplashScreen struct {
	logo     *Logo
	duration time.Duration
	elapsed  time.Duration
}

// splashTickMsg is sent every tick
type splashTickMsg time.Time

// NewSplashScreen creates a new splash screen
func NewSplashScreen(logo *Logo, duration time.Duration) *SplashScreen {
	return &SplashScreen{
		logo:     logo,
		duration: duration,
	}
}

// Init starts the timer
func (s *SplashScreen) Init() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

// Update handles timer updates
func (s *SplashScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case splashTickMsg:
		s.elapsed += 100 * time.Millisecond
		if s.elapsed >= s.duration {
			return s, tea.Quit
		}
		return s, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return splashTickMsg(t)
		})

	case tea.KeyMsg:
		// Allow skipping with any key
		return s, tea.Quit
	}

	return s, nil
}

// View renders the splash screen
func (s *SplashScreen) View() string {
	return s.logo.Render()
}