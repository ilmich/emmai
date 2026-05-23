package bubbletea

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/ilmich/emmai/internal/client"
)

// Update handles all messages and returns updated model + commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// === Window Resize ===
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateComponentSizes()
		return m, nil

	// === Keyboard Input ===
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	// === Mouse Input ===
	case tea.MouseMsg:
		cmd := m.handleMouseEvent(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Continue to update other components
		return m.updateComponents(msg, cmds)

	// === AI Streaming Messages ===
	case streamChunkMsg:
		m.appendToLastMessage(string(msg))
		m.tokenCount = m.client.GetTokenCount()
		
		// Continue listening for more chunks if channels are still active
		if m.streamTextChan != nil && m.streamErrChan != nil {
			return m, listenForNextStreamMsg(m.streamTextChan, m.streamErrChan, m.streamCtx)
		}
		return m, nil

	case streamDoneMsg:
		m.isProcessing = false
		m.streamTextChan = nil
		m.streamErrChan = nil
		m.systemMessage = "✓ Response complete"
		return m, tea.Batch(
			saveConversationCmd(m.client),
		)

	case streamErrorMsg:
		m.isProcessing = false
		m.streamTextChan = nil
		m.streamErrChan = nil
		m.err = msg.err
		m.systemMessage = fmt.Sprintf("✗ Error: %v", msg.err)
		
		// Add error message to chat
		m.appendMessage(client.Message{
			Role:      "error",
			Content:   msg.err.Error(),
			Timestamp: time.Now(),
		})
		return m, nil

	case interruptStreamMsg:
		// Stream was interrupted - remove partial messages
		m.streamTextChan = nil
		m.streamErrChan = nil
		if len(m.messages) > 0 {
			m.removeLastMessage() // Remove streaming assistant message
		}
		if len(m.messages) > 0 {
			m.removeLastMessage() // Remove user message
		}
		m.client.RemoveLastUserMessage()
		m.isProcessing = false
		m.systemMessage = "✗ Interrupted"
		
		// Cancel stream context
		if m.streamCancel != nil {
			m.streamCancel()
		}
		return m, nil

	// === Phase Transitions ===
	case phaseTransitionMsg:
		m.currentPhase = msg.phaseName
		m.systemMessage = fmt.Sprintf("✓ %s phase", msg.phaseName)
		m.appendMessage(client.Message{
			Role:      "system",
			Content:   fmt.Sprintf("Switched to %s phase", msg.phaseName),
			Timestamp: time.Now(),
		})
		return m, nil

	case phaseTransitionErrorMsg:
		m.systemMessage = fmt.Sprintf("✗ Failed to switch to %s: %v", msg.phaseName, msg.err)
		return m, nil

	// === Conversation Management ===
	case conversationLoadedMsg:
		if msg.messageCount > 0 {
			// Load messages from client
			conv := m.client.GetConversation()
			m.messages = conv.Messages
			m.tokenCount = m.client.GetTokenCount()
			m.systemMessage = fmt.Sprintf("Loaded previous conversation (%d messages)", msg.messageCount)
			m.shouldAutoScroll = true // Scroll to bottom after loading
		}
		return m, nil

	case conversationSavedMsg:
		// Silent success
		return m, nil

	// === Clear Conversation ===
	case clearConversationMsg:
		m.clearMessages()
		m.client.ClearConversation()
		m.tokenCount = 0
		m.systemMessage = "Conversation cleared"
		return m, nil

	// === Spinner Animation ===
	case spinner.TickMsg:
		if m.isProcessing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update child components (viewport, textarea)
	return m.updateComponents(msg, cmds)
}

// handleKeyPress processes keyboard shortcuts
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global shortcuts (work even when processing)
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyCtrlQ:
		// Quit application
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit

	case tea.KeyEsc:
		// Interrupt streaming if processing
		if m.isProcessing {
			if m.streamCancel != nil {
				m.streamCancel()
			}
			return m, func() tea.Msg { return interruptStreamMsg{} }
		}
		return m, nil
	}

	// Don't process other keys while AI is responding
	if m.isProcessing {
		return m, nil
	}

	// Shortcuts that only work when not processing
	switch msg.Type {
	case tea.KeyCtrlL:
		// Clear conversation
		return m, func() tea.Msg { return clearConversationMsg{} }

	case tea.KeyCtrlR:
		// Retry last message
		return m.retryLastMessage()

	case tea.KeyEnter:
		// Check if Shift is held (for multi-line)
		if msg.Alt {
			// Shift+Enter: pass to textarea for new line
			break
		}

		// Plain Enter: send message
		text := strings.TrimSpace(m.textarea.Value())
		if text != "" {
			return m.handleSendMessage(text)
		}
		return m, nil
	}

	// Pass to textarea for normal typing
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// handleSendMessage processes user message submission
func (m Model) handleSendMessage(message string) (tea.Model, tea.Cmd) {
	// Check for slash commands
	if strings.HasPrefix(message, "/") {
		m.textarea.Reset()
		return m.handleSlashCommand(message)
	}

	// Regular message - send to AI
	m.isProcessing = true
	m.systemMessage = "Sending..."
	m.textarea.Reset()

	// Add user message to chat
	m.appendMessage(client.Message{
		Role:      "user",
		Content:   message,
		Timestamp: time.Now(),
	})

	// Start streaming message
	m.startStreamingMessage()

	// Create cancellable context for streaming
	m.streamCtx, m.streamCancel = context.WithCancel(m.ctx)

	// Get channels and store them
	textChan, errChan := m.client.SendMessage(m.streamCtx, message)
	m.streamTextChan = textChan
	m.streamErrChan = errChan

	// Start AI streaming
	return m, listenForNextStreamMsg(textChan, errChan, m.streamCtx)
}

// handleSlashCommand processes slash commands
func (m Model) handleSlashCommand(command string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return m, nil
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/plan":
		m.systemMessage = "Switching to PLAN phase..."
		return m, transitionPhaseCmd(m.phaseController, "plan")

	case "/execute":
		m.systemMessage = "Switching to EXECUTE phase..."
		return m, transitionPhaseCmd(m.phaseController, "execute")

	case "/verify":
		m.systemMessage = "Switching to VERIFY phase..."
		return m, transitionPhaseCmd(m.phaseController, "verify")

	case "/explore":
		m.systemMessage = "Switching to EXPLORE phase..."
		return m, transitionPhaseCmd(m.phaseController, "explore")

	case "/reset":
		m.systemMessage = "Resetting to initial phase..."
		return m, resetPhaseCmd(m.phaseController)

	case "/status":
		isReadOnly := map[bool]string{true: "read-only", false: "writable"}[m.phaseManager.IsReadOnly()]
		status := fmt.Sprintf("Current phase: %s (%s)", m.currentPhase, isReadOnly)
		m.systemMessage = status
		m.appendMessage(client.Message{
			Role:      "system",
			Content:   status,
			Timestamp: time.Now(),
		})
		return m, nil

	case "/help":
		help := `Available commands:
  /plan     - Switch to PLAN phase
  /execute  - Switch to EXECUTE phase
  /verify   - Switch to VERIFY phase
  /explore  - Switch to EXPLORE phase
  /reset    - Reset to initial phase
  /status   - Show current phase
  /help     - Show this help`

		m.appendMessage(client.Message{
			Role:      "system",
			Content:   help,
			Timestamp: time.Now(),
		})
		m.systemMessage = "Available commands: /plan, /execute, /verify, /explore, /reset, /status, /help"
		return m, nil

	default:
		m.systemMessage = fmt.Sprintf("Unknown command: %s (try /help)", cmd)
		return m, nil
	}
}

// streamAIResponse is no longer needed - integrated into handleSendMessage

// listenForNextStreamMsg creates a command that waits for the next stream message
func listenForNextStreamMsg(textChan <-chan string, errChan <-chan error, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return interruptStreamMsg{}

		case chunk, ok := <-textChan:
			if !ok {
				return streamDoneMsg{}
			}
			return streamChunkMsg(chunk)

		case err := <-errChan:
			if err != nil {
				return streamErrorMsg{err: err}
			}
			return streamDoneMsg{}
		}
	}
}

// retryLastMessage resends the last user message
func (m Model) retryLastMessage() (tea.Model, tea.Cmd) {
	// Find last user message
	var lastUserMsg string
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			lastUserMsg = m.messages[i].Content
			break
		}
	}

	if lastUserMsg == "" {
		m.systemMessage = "No previous message to retry"
		return m, nil
	}

	// Remove last assistant response if exists
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
		m.removeLastMessage()
		// Also remove from client conversation history
		conv := m.client.GetConversation()
		if len(conv.Messages) > 0 && conv.Messages[len(conv.Messages)-1].Role == "assistant" {
			conv.Messages = conv.Messages[:len(conv.Messages)-1]
		}
	}

	// Resend the message
	return m.handleSendMessage(lastUserMsg)
}

// updateComponents updates child components (viewport, textarea)
func (m Model) updateComponents(msg tea.Msg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Save scroll state BEFORE updating viewport
	wasAtBottom := m.viewport.AtBottom()

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Auto-scroll to bottom if new content arrived AND user was at bottom
	if m.shouldAutoScroll && wasAtBottom {
		m.viewport.GotoBottom()
		m.shouldAutoScroll = false
	}

	// Update textarea (only if not processing)
	if !m.isProcessing {
		m.textarea, cmd = m.textarea.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update spinner
	m.spinner, cmd = m.spinner.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// handleMouseEvent processes mouse clicks and scrolling
func (m *Model) handleMouseEvent(msg tea.MouseMsg) tea.Cmd {
	// Check if mouse is over scrollbar column
	if msg.X != m.scrollbarX {
		// Not on scrollbar, let viewport and other components handle it
		return nil
	}

	// Calculate relative Y position within scrollbar
	relY := msg.Y - m.scrollbarY

	// Check if click is within scrollbar bounds
	if relY < 0 || relY >= m.scrollbarHeight {
		return nil
	}

	// Handle different mouse actions
	switch msg.Button {
	case tea.MouseButtonLeft:
		// Left click on scrollbar
		if msg.Action == tea.MouseActionPress {
			return m.handleScrollbarClick(relY)
		}

	case tea.MouseButtonWheelUp:
		// Scroll wheel up
		m.viewport.ScrollUp(3)
		return nil

	case tea.MouseButtonWheelDown:
		// Scroll wheel down
		m.viewport.ScrollDown(3)
		return nil
	}

	return nil
}

// handleScrollbarClick processes clicks on different scrollbar regions
func (m *Model) handleScrollbarClick(relY int) tea.Cmd {
	// Top arrow (line 0)
	if relY == 0 {
		m.viewport.ScrollUp(1)
		return nil
	}

	// Bottom arrow (last line)
	if relY == m.scrollbarHeight-1 {
		m.viewport.ScrollDown(1)
		return nil
	}

	// Calculate thumb position and height
	totalLines := m.viewport.TotalLineCount()
	visibleLines := m.viewport.VisibleLineCount()

	// If all content is visible, no scrolling needed
	if totalLines <= visibleLines {
		return nil
	}

	// Calculate thumb dimensions (same logic as in renderScrollbar)
	thumbHeight := max(1, (visibleLines*m.scrollbarHeight)/totalLines)
	scrollPercent := m.viewport.ScrollPercent()
	maxThumbPos := m.scrollbarHeight - thumbHeight - 2 // -2 for arrows
	thumbPos := 1 + int(float64(maxThumbPos)*scrollPercent)

	// Check if clicked on thumb itself
	if relY >= thumbPos && relY < thumbPos+thumbHeight {
		// Clicked on thumb - do nothing (no drag support yet)
		return nil
	}

	// Clicked on track above thumb - page up (full page)
	if relY < thumbPos {
		m.viewport.ScrollUp(m.viewport.Height)
		return nil
	}

	// Clicked on track below thumb - page down (full page)
	if relY > thumbPos+thumbHeight {
		m.viewport.ScrollDown(m.viewport.Height)
		return nil
	}

	return nil
}
