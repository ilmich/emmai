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
)

// Model holds the entire application state
type Model struct {
	// Core dependencies
	client       *client.OpenAIClient
	config       *config.Config
	ctx          context.Context
	cancel       context.CancelFunc
	streamCtx    context.Context
	streamCancel context.CancelFunc

	// UI Components (Bubbles)
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// Application State
	messages       []client.Message
	isProcessing   bool
	systemMessage  string
	warnMessage    string
	tokenCount     int
	err            error
	streamTextChan <-chan string
	streamErrChan  <-chan error

	// UI State
	width   int
	height  int
	ready   bool
	workDir string

	// Scrollbar positioning for mouse interaction
	scrollbarX      int
	scrollbarY      int
	scrollbarHeight int
}

// NewModel creates a new Bubble Tea model
func NewModel(cfg *config.Config, aiClient *client.OpenAIClient, workDir string) Model {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type a message (Enter to send, Shift+Enter for new line)..."
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
		client:        aiClient,
		config:        cfg,
		ctx:           ctx,
		cancel:        cancel,
		textarea:      ta,
		viewport:      vp,
		spinner:       sp,
		messages:      []client.Message{},
		isProcessing:  false,
		systemMessage: "Welcome to EmmAI",
		tokenCount:    0,
		ready:         false,
		workDir:       workDir,
	}
}

const SidebarWidth = 36 // visible content width of the sidebar panel

// updateComponentSizes updates viewport and textarea sizes based on terminal dimensions
func (m *Model) updateComponentSizes() {
	const (
		statusBarHeight     = 1
		bannerHeight        = 5
		systemMessageHeight = 3
		inputBoxHeight      = 5
		helpBarHeight       = 1
		borderPadding       = 4 // 2 for borders + 2 for padding
		scrollbarWidth      = 1
		sidebarTotal        = SidebarWidth + borderPadding // sidebar content + its own borders/padding
	)

	totalFixedHeight := statusBarHeight + bannerHeight + systemMessageHeight + inputBoxHeight + helpBarHeight
	chatHeight := m.height - totalFixedHeight - borderPadding

	if chatHeight < 5 {
		chatHeight = 5
	}

	// Viewport occupies width minus scrollbar minus sidebar panel
	m.viewport.Width = m.width - borderPadding - scrollbarWidth - sidebarTotal
	m.viewport.Height = chatHeight

	m.scrollbarX = m.width - sidebarTotal - 2
	m.scrollbarY = statusBarHeight + bannerHeight + systemMessageHeight + 1
	m.scrollbarHeight = chatHeight

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
	wrapStyle := lipgloss.NewStyle().Width(m.viewport.Width)
	var sb strings.Builder
	for i, msg := range m.messages {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(wrapStyle.Render(formatMessage(msg, i == len(m.messages)-1)))
	}
	m.viewport.SetContent(sb.String())
}
