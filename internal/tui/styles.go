package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ilmich/emmai/internal/version"
)

// Color scheme for the TUI (using hex colors for compatibility)
var (
	ColorUser      = tcell.NewRGBColor(0, 255, 255)   // Cyan
	ColorAssistant = tcell.NewRGBColor(0, 255, 0)     // Green
	ColorSystem    = tcell.NewRGBColor(255, 255, 0)   // Yellow
	ColorError     = tcell.NewRGBColor(255, 0, 0)     // Red
	ColorBorder    = tcell.NewRGBColor(0, 0, 255)     // Blue
	ColorStatusBar = tcell.NewRGBColor(0, 139, 139)   // DarkCyan
	ColorHelp      = tcell.NewRGBColor(128, 128, 128) // Gray
)

// Color names as strings for tview dynamic colors
const (
	ColorUserStr      = "cyan"
	ColorAssistantStr = "green"
	ColorSystemStr    = "yellow"
	ColorErrorStr     = "red"
)

// Box characters for borders
const (
	BorderHorizontal  = "─"
	BorderVertical    = "│"
	BorderTopLeft     = "┌"
	BorderTopRight    = "┐"
	BorderBottomLeft  = "└"
	BorderBottomRight = "┘"
)

// Loading spinner characters
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// GetBanner returns the ASCII art banner with version
func GetBanner() string {
	return " ___                    _    ___\n" +
		"| __|_ __  _ __   __ _ (_)  |_ _|\n" +
		"| _|| '  \\| '  \\ / _` || |   | |\n" +
		"|___|_|_|_|_|_|_|\\__,_||_|  |___|\n" +
		"    Terminal AI Chat Interface " + version.GetVersion()
}
