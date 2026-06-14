package ui

import (
	"github.com/ilmich/emmai/internal/phase"
)

// AI Streaming Messages

// streamChunkMsg represents a chunk of text received from AI
type streamChunkMsg string

// streamDoneMsg signals that AI streaming has completed successfully
type streamDoneMsg struct{}

// streamErrorMsg signals an error occurred during streaming
type streamErrorMsg struct {
	err error
}

// Phase Transition Messages

// phaseTransitionMsg signals a successful phase transition
type phaseTransitionMsg struct {
	phaseName string
	response  *phase.PhaseResponse
	autoStart bool // if true, automatically trigger AI after transition
}

// phaseTransitionErrorMsg signals a phase transition failure
type phaseTransitionErrorMsg struct {
	phaseName string
	err       error
}

// System Messages

// systemNotificationMsg displays a notification in the system message bar
type systemNotificationMsg string

// conversationSavedMsg signals conversation was saved successfully
type conversationSavedMsg struct{}

// conversationLoadedMsg signals conversation was loaded successfully
type conversationLoadedMsg struct {
	messageCount int
}

// UI State Messages

// focusInputMsg returns focus to the input box
type focusInputMsg struct{}

// clearConversationMsg clears all messages
type clearConversationMsg struct{}

// interruptStreamMsg cancels current AI streaming
type interruptStreamMsg struct{}
