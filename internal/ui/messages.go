package ui

// AI Streaming Messages

// streamChunkMsg represents a chunk of text received from AI
type streamChunkMsg string

// streamDoneMsg signals that AI streaming has completed successfully
type streamDoneMsg struct{}

// streamErrorMsg signals an error occurred during streaming
type streamErrorMsg struct {
	err error
}

// System Messages

// conversationSavedMsg signals conversation was saved successfully
type conversationSavedMsg struct{}

// conversationLoadedMsg signals conversation was loaded successfully
type conversationLoadedMsg struct {
	messageCount int
}

// UI State Messages

// clearConversationMsg clears all messages
type clearConversationMsg struct{}

// interruptStreamMsg cancels current AI streaming
type interruptStreamMsg struct{}
