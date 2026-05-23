package bubbletea

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/phase"
	"github.com/ilmich/emmai/internal/storage"
)

// streamAIResponseCmd starts streaming from OpenAI and converts channel messages to Bubble Tea messages
func streamAIResponseCmd(ctx context.Context, client *client.OpenAIClient, userMsg string) tea.Cmd {
	return func() tea.Msg {
		textChan, errChan := client.SendMessage(ctx, userMsg)

		// Wait for first message from either channel
		select {
		case <-ctx.Done():
			// Context cancelled (ESC pressed)
			return interruptStreamMsg{}

		case chunk, ok := <-textChan:
			if !ok {
				// Channel closed, streaming done
				return streamDoneMsg{}
			}
			// Return chunk and listen for more
			return streamChunkMsg(chunk)

		case err := <-errChan:
			// Error occurred
			if err != nil {
				return streamErrorMsg{err: err}
			}
			return streamDoneMsg{}
		}
	}
}

// listenForStreamCmd continues listening for stream messages
func listenForStreamCmd(textChan <-chan string, errChan <-chan error, ctx context.Context) tea.Cmd {
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

// transitionPhaseCmd transitions to a new phase
func transitionPhaseCmd(controller *phase.Controller, phaseName string) tea.Cmd {
	return func() tea.Msg {
		resp, err := controller.TransitionToPhase(phaseName)
		if err != nil {
			return phaseTransitionErrorMsg{
				phaseName: phaseName,
				err:       err,
			}
		}
		return phaseTransitionMsg{
			phaseName: phaseName,
			response:  resp,
		}
	}
}

// resetPhaseCmd resets to the initial phase
func resetPhaseCmd(controller *phase.Controller) tea.Cmd {
	return func() tea.Msg {
		resp, err := controller.ResetToInitial()
		if err != nil {
			return phaseTransitionErrorMsg{
				phaseName: "reset",
				err:       err,
			}
		}
		return phaseTransitionMsg{
			phaseName: resp.Phase,
			response:  resp,
		}
	}
}

// saveConversationCmd saves conversation to disk
func saveConversationCmd(client *client.OpenAIClient) tea.Cmd {
	return func() tea.Msg {
		conv := client.GetConversation()
		if err := storage.SaveConversation(conv); err != nil {
			// Silent failure - don't interrupt user experience
			return nil
		}
		return conversationSavedMsg{}
	}
}

// loadConversationCmd loads the most recent conversation from disk
func loadConversationCmd() tea.Cmd {
	return func() tea.Msg {
		conv, err := storage.LoadMostRecent()
		if err != nil || conv == nil {
			// No conversation found
			return conversationLoadedMsg{messageCount: 0}
		}
		return conversationLoadedMsg{messageCount: len(conv.Messages)}
	}
}
