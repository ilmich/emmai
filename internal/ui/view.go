package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ilmich/emmai/internal/client"
)

// View renders the entire UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Build UI from top to bottom
	sections := []string{
		m.renderStatusBar(),
		GetBanner(),
		m.renderSystemMessage(),
		m.renderChatViewport(),
		m.renderInputBox(),
		m.renderHelpBar(),
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderStatusBar renders the top status bar
func (m Model) renderStatusBar() string {
	customEndpoint := ""
	if m.config.IsCustomEndpoint() {
		customEndpoint = " (Custom)"
	}

	status := fmt.Sprintf("Model: %s%s | Tokens: %d | Phase: %s",
		m.config.Model, customEndpoint, m.tokenCount, m.currentPhase)

	if m.warnMessage != "" {
		status += " | ⚠ " + m.warnMessage
	}

	if m.isProcessing {
		status = m.spinner.View() + " " + status
	}

	return statusBarStyle.Width(m.width).Render(status)
}

// renderSystemMessage renders the notification bar
func (m Model) renderSystemMessage() string {
	if m.systemMessage == "" {
		return ""
	}
	return systemMessageStyle.Width(m.width - 4).Render(m.systemMessage)
}

// renderChatViewport renders the scrollable chat area + scrollbar + sidebar side by side.
func (m Model) renderChatViewport() string {
	viewportContent := m.viewport.View()
	scrollbar := m.renderScrollbar(m.viewport.Height)

	chatWithBar := lipgloss.JoinHorizontal(lipgloss.Top, viewportContent, scrollbar)
	chatBox := chatViewportStyle.Width(m.width - 2 - SidebarWidth - 4).Render(chatWithBar)

	sidebar := m.renderSidebar()

	return lipgloss.JoinHorizontal(lipgloss.Top, chatBox, sidebar)
}

// renderSidebar renders the info panel showing model settings and token usage.
func (m Model) renderSidebar() string {
	w := SidebarWidth

	label := func(s string) string { return sidebarLabelStyle.Render(s) }
	value := func(s string) string {
		if len(s) > w {
			s = s[:w-1] + "…"
		}
		return sidebarValueStyle.Render(s)
	}
	section := func(s string) string { return sidebarSectionStyle.Render(s) }

	ctxInfo := "not set"
	if m.config.ContextSize > 0 {
		used := float64(m.tokenCount) / float64(m.config.ContextSize) * 100
		ctxInfo = fmt.Sprintf("%d / %d (%.0f%%)", m.tokenCount, m.config.ContextSize, used)
	}

	compactionStatus := "disabled"
	if m.config.ContextSize > 0 {
		threshold := int(float64(m.config.ContextSize) * 0.85)
		compactionStatus = fmt.Sprintf("at %d tok", threshold)
	}

	lines := []string{
		section("── Model ──"),
		label("name:"),
		value(m.config.Model),
		label("endpoint:"),
		value(endpointShort(m.config.BaseURL)),
		label("temp:"),
		value(fmt.Sprintf("%.2f", m.config.Temperature)),
		label("max tokens:"),
		value(fmt.Sprintf("%d", m.config.MaxTokens)),
		"",
		section("── Context ──"),
		label("used:"),
		value(ctxInfo),
		label("compact:"),
		value(compactionStatus),
		"",
		section("── Session ──"),
		label("phase:"),
		value(m.currentPhase),
		label("messages:"),
		value(fmt.Sprintf("%d", len(m.messages))),
	}

	return sidebarStyle.Width(w).Height(m.viewport.Height).Render(strings.Join(lines, "\n"))
}

// endpointShort returns a short representation of a base URL.
func endpointShort(baseURL string) string {
	baseURL = strings.TrimPrefix(baseURL, "https://")
	baseURL = strings.TrimPrefix(baseURL, "http://")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	return baseURL
}

// formatMessage formats a single message with role coloring
func (m Model) formatMessage(msg client.Message, isLast bool) string {
	var roleStyle lipgloss.Style
	var roleName string

	switch msg.Role {
	case "user":
		roleStyle = userMessageStyle
		roleName = "You"
	case "assistant":
		roleStyle = assistantMessageStyle
		roleName = "[" + m.currentPhase + "]" + m.config.Model
	case "system":
		roleStyle = systemMessageStyle2
		roleName = "System"
	case "error":
		roleStyle = errorMessageStyle
		roleName = "Error"
	default:
		roleStyle = messageContentStyle
		roleName = msg.Role
	}

	// Format: "You: message content"
	header := roleStyle.Render(roleName + ":")
	content := messageContentStyle.Render(" " + msg.Content)

	// Add streaming cursor if this is the last assistant message and we're processing
	if isLast && msg.Role == "assistant" && m.isProcessing && msg.Content != "" {
		content += cursorStyle.Render("▊")
	}

	return header + content
}

// renderInputBox renders the user input textarea
func (m Model) renderInputBox() string {
	style := inputBoxStyle
	if m.textarea.Focused() {
		style = inputBoxFocusedStyle
	}
	if m.isProcessing {
		style = inputBoxDisabledStyle
	}

	return style.Width(m.width - 2).Render(m.textarea.View())
}

// renderHelpBar renders the bottom help text
func (m Model) renderHelpBar() string {
	return helpBarStyle.Width(m.width).Render(GetHelpText())
}

// renderScrollbar creates a vertical scrollbar indicator
func (m Model) renderScrollbar(height int) string {
	// Calculate scrollbar dimensions
	totalLines := m.viewport.TotalLineCount()
	visibleLines := m.viewport.VisibleLineCount()

	// If height is too small, return minimal scrollbar
	if height < 3 {
		return scrollbarTrackStyle.Render("┃")
	}

	// Calculate thumb size (proportional to visible/total ratio)
	// Minimum thumb size is 1
	thumbHeight := 1
	if totalLines > visibleLines {
		thumbHeight = max(1, (visibleLines*height)/totalLines)
	}

	// Calculate thumb position based on scroll percent
	scrollPercent := m.viewport.ScrollPercent()
	maxThumbPos := height - thumbHeight - 2 // -2 for arrows
	thumbPos := 1 + int(float64(maxThumbPos)*scrollPercent)

	// Build scrollbar
	var sb strings.Builder

	// Top arrow or indicator
	if m.viewport.AtTop() {
		sb.WriteString(scrollbarThumbStyle.Render("▲"))
	} else {
		sb.WriteString(scrollbarTrackStyle.Render("▲"))
	}
	sb.WriteString("\n")

	// Track and thumb (height - 2 for the arrows)
	for i := 1; i < height-1; i++ {
		if i >= thumbPos && i < thumbPos+thumbHeight {
			sb.WriteString(scrollbarThumbStyle.Render("█")) // Thumb
		} else {
			sb.WriteString(scrollbarTrackStyle.Render("┃")) // Track
		}
		sb.WriteString("\n")
	}

	// Bottom arrow or indicator
	if m.viewport.AtBottom() {
		sb.WriteString(scrollbarThumbStyle.Render("▼"))
	} else {
		sb.WriteString(scrollbarTrackStyle.Render("▼"))
	}

	return sb.String()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
