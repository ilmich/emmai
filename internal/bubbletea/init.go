package bubbletea

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/storage"
)

// Init is called when the program starts
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		textarea.Blink,
		loadRecentConversationCmd(m.client),
	)
}

// loadRecentConversationCmd loads the most recent conversation
func loadRecentConversationCmd(client *client.OpenAIClient) tea.Cmd {
	return func() tea.Msg {
		conv, err := storage.LoadMostRecent()
		if err == nil && conv != nil && len(conv.Messages) > 0 {
			// Load conversation into client
			client.LoadConversation(conv)
			return conversationLoadedMsg{messageCount: len(conv.Messages)}
		}
		return conversationLoadedMsg{messageCount: 0}
	}
}
