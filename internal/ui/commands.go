package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/phase"
	"github.com/ilmich/emmai/internal/storage"
)

// transitionPhaseCmd transitions to a new phase. autoStart=true triggers AI immediately after.
func transitionPhaseCmd(controller *phase.Controller, phaseName string, autoStart bool) tea.Cmd {
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
			autoStart: autoStart,
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
