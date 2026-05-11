package tui

import (
	"fmt"

	"github.com/rivo/tview"
)

// SystemMessageBar displays system messages outside the chat
type SystemMessageBar struct {
	*tview.TextView
	app *tview.Application
}

// NewSystemMessageBar creates a new system message bar
func NewSystemMessageBar(app *tview.Application) *SystemMessageBar {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)

	tv.SetBorder(true).
		SetTitle(" System ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(ColorBorder)

	smb := &SystemMessageBar{
		TextView: tv,
		app:      app,
	}

	return smb
}

// SetMessage updates the system message
func (smb *SystemMessageBar) SetMessage(message string) {
	smb.TextView.Clear()
	fmt.Fprintf(smb.TextView, "[%s]%s[white]", ColorSystemStr, message)	
}

// Clear removes the current message
func (smb *SystemMessageBar) Clear() {
	smb.TextView.Clear()
}
