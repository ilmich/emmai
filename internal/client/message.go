package client

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ToolCall represents a tool call made by the assistant
type ToolCall struct {
	ID       string                 `json:"id"`       // Unique ID for this tool call
	Type     string                 `json:"type"`     // Type of tool call (e.g., "function")
	Function ToolCallFunction       `json:"function"` // Function details
}

// ToolCallFunction represents the function being called
type ToolCallFunction struct {
	Name      string `json:"name"`      // Name of the function
	Arguments string `json:"arguments"` // JSON string of arguments
}

// Message represents a single chat message
type Message struct {
	Role       string     `json:"role"`                 // "user", "assistant", "system", "tool"
	Content    string     `json:"content,omitempty"`    // The message text
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"` // Tool calls made by assistant
	ToolCallID string     `json:"tool_call_id,omitempty"` // ID of tool call this is responding to
	Name       string     `json:"name,omitempty"`       // Name of the tool/function
	Timestamp  time.Time  `json:"timestamp"`            // When the message was created
}

// Task represents a single task in the planner
type Task struct {
	ID        string    `json:"id"`         // Unique task ID
	Title     string    `json:"title"`      // Task title/description
	Status    string    `json:"status"`     // pending, in_progress, completed, cancelled
	Priority  string    `json:"priority"`   // high, medium, low
	CreatedAt time.Time `json:"created_at"` // Creation timestamp
	UpdatedAt time.Time `json:"updated_at"` // Last update timestamp
}

// Conversation represents a chat conversation with history
type Conversation struct {
	ID       string    `json:"id"`              // Unique conversation ID
	Messages []Message `json:"messages"`        // All messages in the conversation
	Model    string    `json:"model"`           // The AI model used
	Created  time.Time `json:"created"`         // When conversation was created
	Updated  time.Time `json:"updated"`         // Last message timestamp
	Tasks    []Task    `json:"tasks,omitempty"` // Task list for this conversation
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

// RemoveLastUserMessage removes the most recent user message from the conversation
func (c *Conversation) RemoveLastUserMessage() bool {
	// Search backwards to find the last user message
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == "user" {
			// Remove the message by slicing around it
			c.Messages = append(c.Messages[:i], c.Messages[i+1:]...)
			c.Updated = time.Now()
			return true
		}
	}
	return false // No user message found
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

// AddTask adds a new task to the conversation
func (c *Conversation) AddTask(title, priority string) *Task {
	task := Task{
		ID:        generateID(),
		Title:     title,
		Status:    "pending",
		Priority:  priority,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	c.Tasks = append(c.Tasks, task)
	c.Updated = time.Now()
	return &task
}

// GetTask retrieves a task by ID
func (c *Conversation) GetTask(id string) (*Task, error) {
	for i := range c.Tasks {
		if c.Tasks[i].ID == id {
			return &c.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", id)
}

// UpdateTask updates a task's fields
func (c *Conversation) UpdateTask(id string, updates map[string]interface{}) error {
	for i := range c.Tasks {
		if c.Tasks[i].ID == id {
			// Update fields if provided
			if status, ok := updates["status"].(string); ok {
				c.Tasks[i].Status = status
			}
			if priority, ok := updates["priority"].(string); ok {
				c.Tasks[i].Priority = priority
			}
			if title, ok := updates["title"].(string); ok {
				c.Tasks[i].Title = title
			}
			c.Tasks[i].UpdatedAt = time.Now()
			c.Updated = time.Now()
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", id)
}

// DeleteTask removes a task from the conversation
func (c *Conversation) DeleteTask(id string) error {
	for i := range c.Tasks {
		if c.Tasks[i].ID == id {
			c.Tasks = append(c.Tasks[:i], c.Tasks[i+1:]...)
			c.Updated = time.Now()
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", id)
}

// ListTasks returns tasks filtered by status and/or priority
func (c *Conversation) ListTasks(statusFilter, priorityFilter string) []Task {
	if statusFilter == "" && priorityFilter == "" {
		// Return all tasks
		result := make([]Task, len(c.Tasks))
		copy(result, c.Tasks)
		return result
	}

	filtered := make([]Task, 0)
	for _, task := range c.Tasks {
		// Check status filter
		if statusFilter != "" && task.Status != statusFilter {
			continue
		}
		// Check priority filter
		if priorityFilter != "" && task.Priority != priorityFilter {
			continue
		}
		filtered = append(filtered, task)
	}
	return filtered
}

