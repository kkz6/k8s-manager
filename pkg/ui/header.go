package ui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// DisplayHeader shows the application header with developer info
func DisplayHeader() {
	// Clear screen
	fmt.Print("\033[H\033[2J")

	// Create big text header using pterm
	s, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("K8s", pterm.NewStyle(pterm.FgCyan)),
		pterm.NewLettersFromStringWithStyle(" Manager", pterm.NewStyle(pterm.FgLightCyan)),
	).Srender()

	pterm.DefaultCenter.Print(s)

	// Create info panel with developer information
	data := [][]string{
		{"👨‍💻", "Developed by", "Karthick"},
		{"🌐", "Website", "https://karti.dev"},
		{"📧", "Email", "karthick@gigcodes.com"},
	}

	// Create a styled table
	pterm.DefaultTable.WithHasHeader(false).
		WithBoxed(true).
		WithData(data).
		Render()

	fmt.Println()
}

// SetupInterruptHandler sets up graceful shutdown on interrupt
func SetupInterruptHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n\n" + color.YellowString("👋 Gracefully shutting down K8s Manager..."))
		fmt.Println(color.CyanString("Thank you for using K8s Manager!"))
		os.Exit(0)
	}()
}

// ShowError displays an error message in red
func ShowError(msg string) {
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	fmt.Printf("%s %s\n", red("❌ Error:"), msg)
}

// ShowSuccess displays a success message in green
func ShowSuccess(msg string) {
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	fmt.Printf("%s %s\n", green("✅ Success:"), msg)
}

// ShowInfo displays an info message in blue
func ShowInfo(msg string) {
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s %s\n", blue("ℹ️  Info:"), msg)
}

// ShowWarning displays a warning message in yellow
func ShowWarning(msg string) {
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s %s\n", yellow("⚠️  Warning:"), msg)
}