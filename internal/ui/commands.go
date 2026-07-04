package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/storage"
)

// saveConversationCmd saves conversation to the project's .emmai/conversations/ dir.
func saveConversationCmd(client *client.OpenAIClient, workDir string) tea.Cmd {
	return func() tea.Msg {
		conv := client.GetConversation()
		if err := storage.SaveConversation(conv, workDir); err != nil {
			return nil
		}
		return conversationSavedMsg{}
	}
}

// loadConversationCmd loads the most recent conversation from the project dir.
func loadConversationCmd(workDir string) tea.Cmd {
	return func() tea.Msg {
		conv, err := storage.LoadMostRecent(workDir)
		if err != nil || conv == nil {
			return conversationLoadedMsg{messageCount: 0}
		}
		return conversationLoadedMsg{messageCount: len(conv.Messages)}
	}
}
