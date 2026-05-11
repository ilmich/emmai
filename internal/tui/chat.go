package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ilmich/emmai/internal/client"
	"github.com/rivo/tview"
)

// ChatView displays the conversation history
type ChatView struct {
	*tview.TextView
	messages []client.Message
}

// NewChatView creates a new chat view component
func NewChatView() *ChatView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			// Auto-scroll to bottom when new content is added
		})

	tv.SetBorder(true).
		SetTitle(" Chat History ").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(ColorBorder)

	return &ChatView{
		TextView: tv,
		messages: make([]client.Message, 0),
	}
}

// AddUserMessage adds a user message to the chat view
func (cv *ChatView) AddUserMessage(content string) {
	cv.messages = append(cv.messages, client.Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	})
	cv.render()
}

// AddAssistantMessage adds an assistant message to the chat view
func (cv *ChatView) AddAssistantMessage(content string) {
	cv.messages = append(cv.messages, client.Message{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
	})
	cv.render()
}

// AddErrorMessage adds an error message to the chat view
func (cv *ChatView) AddErrorMessage(content string) {
	cv.messages = append(cv.messages, client.Message{
		Role:      "error",
		Content:   content,
		Timestamp: time.Now(),
	})
	cv.render()
}

// StartStreamingMessage begins a new assistant message that will be updated incrementally
func (cv *ChatView) StartStreamingMessage() {
	cv.messages = append(cv.messages, client.Message{
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
	})
}

// AppendToLastMessage appends text to the last message (for streaming)
func (cv *ChatView) AppendToLastMessage(text string) {
	if len(cv.messages) > 0 {
		lastIdx := len(cv.messages) - 1
		cv.messages[lastIdx].Content += text
		cv.render()
	}
}

// Clear removes all messages from the chat view
func (cv *ChatView) Clear() {
	cv.messages = make([]client.Message, 0)
	cv.TextView.Clear()
}

// LoadMessages loads messages from a conversation
func (cv *ChatView) LoadMessages(messages []client.Message) {
	cv.messages = messages
	cv.render()
}

// RemoveLastMessage removes the last message from the chat view
func (cv *ChatView) RemoveLastMessage() {
	if len(cv.messages) > 0 {
		cv.messages = cv.messages[:len(cv.messages)-1]
		cv.render()
	}
}

// render formats and displays all messages
func (cv *ChatView) render() {
	cv.TextView.Clear()
	var builder strings.Builder

	for i, msg := range cv.messages {
		if i > 0 {
			builder.WriteString("\n")
		}

		switch msg.Role {
		case "user":
			builder.WriteString(fmt.Sprintf("[%s]You:[white] %s",
				ColorUserStr, msg.Content))
		case "assistant":
			builder.WriteString(fmt.Sprintf("[%s]Assistant:[white] %s",
				ColorAssistantStr, msg.Content))
			// Add cursor for streaming
			if i == len(cv.messages)-1 && msg.Content != "" {
				builder.WriteString("[white]▊")
			}
		case "error":
			builder.WriteString(fmt.Sprintf("[%s]Error:[white] %s",
				ColorErrorStr, msg.Content))
		}
	}

	fmt.Fprintf(cv.TextView, "%s", builder.String())
	cv.ScrollToEnd()
}
