package client

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Message represents a single chat message
type Message struct {
	Role      string    `json:"role"`      // "user", "assistant", "system"
	Content   string    `json:"content"`   // The message text
	Timestamp time.Time `json:"timestamp"` // When the message was created
}

// Conversation represents a chat conversation with history
type Conversation struct {
	ID       string    `json:"id"`       // Unique conversation ID
	Messages []Message `json:"messages"` // All messages in the conversation
	Model    string    `json:"model"`    // The AI model used
	Created  time.Time `json:"created"`  // When conversation was created
	Updated  time.Time `json:"updated"`  // Last message timestamp
}

// NewConversation creates a new conversation
func NewConversation(model string) *Conversation {
	return &Conversation{
		ID:       generateID(),
		Model:    model,
		Messages: make([]Message, 0),
		Created:  time.Now(),
		Updated:  time.Now(),
	}
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	c.Updated = time.Now()
}

// GetLastUserMessage returns the last user message, if any
func (c *Conversation) GetLastUserMessage() string {
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == "user" {
			return c.Messages[i].Content
		}
	}
	return ""
}

// Clear removes all messages from the conversation
func (c *Conversation) Clear() {
	c.Messages = make([]Message, 0)
	c.Updated = time.Now()
}

// generateID creates a random conversation ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
