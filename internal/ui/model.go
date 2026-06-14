package ui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/phase"
)

// Model holds the entire application state
type Model struct {
	// Core dependencies
	client          *client.OpenAIClient
	config          *config.Config
	phaseManager    *phase.Manager
	phaseController *phase.Controller
	ctx             context.Context
	cancel          context.CancelFunc
	streamCtx       context.Context
	streamCancel    context.CancelFunc

	// UI Components (Bubbles)
	viewport viewport.Model // Chat message display
	textarea textarea.Model // User input
	spinner  spinner.Model  // Loading indicator

	// Application State
	messages       []client.Message // Chat history
	isProcessing   bool             // Is AI responding?
	currentPhase   string           // Current workflow phase
	systemMessage  string           // Top notification bar (transient)
	warnMessage    string           // Persistent warning shown in status bar
	tokenCount     int              // Token usage
	err            error            // Last error
	streamTextChan <-chan string    // Active stream text channel
	streamErrChan  <-chan error     // Active stream error channel

	// UI State
	width  int  // Terminal width
	height int  // Terminal height
	ready  bool // Has received first WindowSizeMsg?

	// Scrollbar positioning for mouse interaction
	scrollbarX      int // X coordinate of scrollbar column in terminal
	scrollbarY      int // Y coordinate of scrollbar start in terminal
	scrollbarHeight int // Height of scrollbar in lines
}

// NewModel creates a new Bubble Tea model
func NewModel(cfg *config.Config, aiClient *client.OpenAIClient, phaseManager *phase.Manager, phaseController *phase.Controller) Model {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type message or /plan, /execute, /verify (Enter to send, Shift+Enter for new line)..."
	ta.Focus()
	ta.CharLimit = 0 // No limit
	ta.SetWidth(80)  // Will be resized on first render
	ta.SetHeight(3)  // 3 lines by default

	// Initialize viewport for chat
	vp := viewport.New(80, 20) // Will be resized on first render
	vp.Style = chatViewportStyle

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorGreen)

	return Model{
		client:          aiClient,
		config:          cfg,
		phaseManager:    phaseManager,
		phaseController: phaseController,
		ctx:             ctx,
		cancel:          cancel,
		textarea:        ta,
		viewport:        vp,
		spinner:         sp,
		messages:        []client.Message{},
		isProcessing:    false,
		currentPhase:    phaseManager.GetInitialPhase(),
		systemMessage:   "Welcome to EmmAI",
		tokenCount:      0,
		ready:           false,
	}
}

// updateComponentSizes updates viewport and textarea sizes based on terminal dimensions
func (m *Model) updateComponentSizes() {
	const (
		statusBarHeight       = 1
		bannerHeight          = 5
		systemMessageHeight   = 3
		inputBoxHeight        = 5
		helpBarHeight         = 1
		borderPadding         = 4 // 2 for borders + 2 for padding
		scrollbarWidth        = 1 // Width for scrollbar
	)

	// Calculate available space
	totalFixedHeight := statusBarHeight + bannerHeight + systemMessageHeight + inputBoxHeight + helpBarHeight
	chatHeight := m.height - totalFixedHeight - borderPadding

	if chatHeight < 5 {
		chatHeight = 5 // Minimum height
	}

	// Update viewport (subtract scrollbar width from viewport content width)
	m.viewport.Width = m.width - borderPadding - scrollbarWidth
	m.viewport.Height = chatHeight

	// Calculate scrollbar position for mouse interaction
	// Scrollbar is on the right edge, inside the border
	m.scrollbarX = m.width - 2           // -2 for right border (1 char border + 1 char padding)
	m.scrollbarY = statusBarHeight + bannerHeight + systemMessageHeight + 1 // +1 for top border
	m.scrollbarHeight = chatHeight

	// Update textarea
	m.textarea.SetWidth(m.width - borderPadding)
}

// appendMessage adds a message to the chat history
func (m *Model) appendMessage(msg client.Message) {
	m.messages = append(m.messages, msg)
}

// appendToLastMessage appends text to the last message (for streaming)
func (m *Model) appendToLastMessage(text string) {
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].Content += text
	}
}

// removeLastMessage removes the last message from chat history
func (m *Model) removeLastMessage() {
	if len(m.messages) > 0 {
		m.messages = m.messages[:len(m.messages)-1]
	}
}

// startStreamingMessage begins a new assistant message for streaming
func (m *Model) startStreamingMessage() {
	m.appendMessage(client.Message{
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
	})
}

// clearMessages clears all chat messages
func (m *Model) clearMessages() {
	m.messages = []client.Message{}
	m.viewport.SetContent("")
}

// refreshViewportContent rebuilds the viewport content from current messages.
// Must be called on the real model (pointer receiver) whenever messages change.
func (m *Model) refreshViewportContent(formatMessage func(msg client.Message, isLast bool) string) {
	var sb strings.Builder
	for i, msg := range m.messages {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(formatMessage(msg, i == len(m.messages)-1))
	}
	m.viewport.SetContent(sb.String())
}
