package tui

import (
	"fmt"

	"github.com/rivo/tview"
)

// BannerView displays the ASCII art banner
type BannerView struct {
	*tview.TextView
}

// NewBannerView creates a new banner view component
func NewBannerView() *BannerView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	tv.SetBorder(false)

	bv := &BannerView{
		TextView: tv,
	}

	// Set banner content
	banner := GetBanner()
	fmt.Fprintf(tv, "[blue]%s[white]", banner)

	return bv
}
