package tui

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

// StatusBar displays application status
type StatusBar struct {
	*tview.TextView
	model        string
	tokens       int
	isLoading    bool
	isCustom     bool // Track if using custom endpoint
	spinnerFrame int
	stopSpinner  chan struct{}
	app          *tview.Application
}

// NewStatusBar creates a new status bar component
func NewStatusBar(app *tview.Application, model string, isCustomEndpoint bool) *StatusBar {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	tv.SetBackgroundColor(ColorStatusBar)

	sb := &StatusBar{
		TextView:     tv,
		model:        model,
		tokens:       0,
		isLoading:    false,
		isCustom:     isCustomEndpoint,
		spinnerFrame: 0,
		stopSpinner:  make(chan struct{}),
		app:          app,
	}

	sb.update()
	return sb
}

// SetModel updates the model name
func (sb *StatusBar) SetModel(model string) {
	sb.model = model
	sb.update()
}

// SetTokens updates the token count
func (sb *StatusBar) SetTokens(tokens int) {
	sb.tokens = tokens
	sb.update()
}

// StartLoading shows the loading indicator
func (sb *StatusBar) StartLoading() {
	if sb.isLoading {
		return
	}
	sb.isLoading = true
	sb.spinnerFrame = 0

	go sb.animateSpinner()
}

// StopLoading hides the loading indicator
func (sb *StatusBar) StopLoading() {
	if !sb.isLoading {
		return
	}
	sb.isLoading = false
	select {
	case sb.stopSpinner <- struct{}{}:
	default:
	}
	sb.update()
}

// animateSpinner runs the loading animation
func (sb *StatusBar) animateSpinner() {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sb.stopSpinner:
			return
		case <-ticker.C:
			sb.spinnerFrame = (sb.spinnerFrame + 1) % len(SpinnerFrames)
			sb.app.QueueUpdateDraw(func() {
				sb.update()
			})
		}
	}
}

// update refreshes the status bar display
func (sb *StatusBar) update() {
	var status string
	
	// Format token count
	tokenStr := formatTokenCount(sb.tokens)
	
	// Add custom endpoint indicator
	modelDisplay := sb.model
	if sb.isCustom {
		modelDisplay = sb.model + " (Custom)"
	}
	
	// Build status line
	if sb.isLoading {
		spinner := SpinnerFrames[sb.spinnerFrame]
		status = fmt.Sprintf(" EmmAI - %s | Tokens: %s | %s Loading...", 
			modelDisplay, tokenStr, spinner)
	} else {
		status = fmt.Sprintf(" EmmAI - %s | Tokens: %s", 
			modelDisplay, tokenStr)
	}

	sb.Clear()
	fmt.Fprint(sb.TextView, status)
}

// formatTokenCount formats the token count with K suffix
func formatTokenCount(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%.1fK", float64(tokens)/1000.0)
}
