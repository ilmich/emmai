package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InputBox handles user text input
type InputBox struct {
	*tview.TextArea
	onSend  func(string)
	app     *tview.Application
	enabled bool
}

// NewInputBox creates a new input box component
func NewInputBox(app *tview.Application, onSend func(string)) *InputBox {
	ta := tview.NewTextArea().
		SetPlaceholder("Type your message (Enter to send, Shift+Enter for new line)...")

	ta.SetBorder(true).
		SetTitle(" Input ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(ColorBorder)

	ib := &InputBox{
		TextArea: ta,
		onSend:   onSend,
		app:      app,
		enabled:  true,
	}

	// Set up key bindings
	ta.SetInputCapture(ib.handleInput)

	return ib
}

// handleInput processes keyboard input
func (ib *InputBox) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// Enter to send (unless Shift is held for new line)
	if event.Key() == tcell.KeyEnter {
		// If Shift+Enter, allow new line
		if event.Modifiers()&tcell.ModShift != 0 {
			return event // pass to TextArea for new line
		}
		
		// Plain Enter: send message
		if ib.enabled {
			text := ib.GetText()
			if text != "" {
				ib.onSend(text)
				ib.Clear()
			}
		}
		return nil // consume event
	}

	return event
}

// Clear empties the input box
func (ib *InputBox) Clear() {
	ib.SetText("", true)
}

// Enable allows input
func (ib *InputBox) Enable() {
	ib.enabled = true
	ib.SetPlaceholder("Type your message (Enter to send, Shift+Enter for new line)...")
	ib.SetBorderColor(ColorBorder)
}

// Disable prevents input
func (ib *InputBox) Disable() {
	ib.enabled = false
	ib.SetPlaceholder("Waiting for response...")
	ib.SetBorderColor(tcell.ColorGray)
}

// IsEnabled returns whether input is enabled
func (ib *InputBox) IsEnabled() bool {
	return ib.enabled
}
