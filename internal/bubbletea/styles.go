package bubbletea

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ilmich/emmai/internal/version"
)

// Color scheme matching current tview implementation
var (
	colorCyan      = lipgloss.Color("cyan")
	colorGreen     = lipgloss.Color("green")
	colorYellow    = lipgloss.Color("yellow")
	colorRed       = lipgloss.Color("red")
	colorBlue      = lipgloss.Color("blue")
	colorDarkCyan  = lipgloss.Color("#008B8B")
	colorGray      = lipgloss.Color("gray")
	colorWhite     = lipgloss.Color("white")
)

// Component styles
var (
	// statusBarStyle renders the top status bar
	statusBarStyle = lipgloss.NewStyle().
		Foreground(colorWhite).
		Background(colorDarkCyan).
		Bold(true).
		Padding(0, 1)

	// bannerStyle renders the ASCII art banner
	bannerStyle = lipgloss.NewStyle().
		Foreground(colorBlue).
		Bold(true)

	// systemMessageStyle renders the system notification bar
	systemMessageStyle = lipgloss.NewStyle().
		Foreground(colorYellow).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(0, 1).
		Margin(0, 1)

	// chatViewportStyle renders the chat message area
	chatViewportStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue).
		Padding(0, 1)

	// inputBoxStyle renders the input text area
	inputBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue)

	// inputBoxFocusedStyle renders the input box when focused
	inputBoxFocusedStyle = inputBoxStyle.Copy().
		BorderForeground(colorCyan)

	// inputBoxDisabledStyle renders the input box when disabled
	inputBoxDisabledStyle = inputBoxStyle.Copy().
		BorderForeground(colorGray)

	// helpBarStyle renders the bottom help text
	helpBarStyle = lipgloss.NewStyle().
		Foreground(colorGray).
		Padding(0, 1)

	// Message role styles
	userMessageStyle = lipgloss.NewStyle().
		Foreground(colorCyan).
		Bold(true)

	assistantMessageStyle = lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)

	systemMessageStyle2 = lipgloss.NewStyle().
		Foreground(colorYellow).
		Bold(true)

	errorMessageStyle = lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)

	// Content style
	messageContentStyle = lipgloss.NewStyle().
		Foreground(colorWhite)

	// Cursor style for streaming messages
	cursorStyle = lipgloss.NewStyle().
		Foreground(colorWhite).
		Bold(true)

	// Scrollbar styles
	scrollbarTrackStyle = lipgloss.NewStyle().
		Foreground(colorBlue)

	scrollbarThumbStyle = lipgloss.NewStyle().
		Foreground(colorCyan).
		Bold(true)
)

// GetBanner returns the ASCII art banner with version
func GetBanner() string {
	banner := ` ___                    _    ___
| __|_ __  _ __   __ _ (_)  |_ _|
| _|| '  \| '  \ / _| || |   | |
|___|_|_|_|_|_|_|\__,_||_|  |___|
    Terminal AI Chat Interface ` + version.GetVersion()

	return bannerStyle.Render(banner)
}

// GetHelpText returns the help bar text
func GetHelpText() string {
	return "[Enter] Send | [/plan /execute /verify] Phase | [ESC] Stop | [Ctrl+L] Clear | [Ctrl+R] Retry | [Ctrl+Q] Quit"
}
