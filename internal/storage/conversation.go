package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
)


const conversationsDir = "conversations"

// ConversationMetadata holds summary information about a conversation
type ConversationMetadata struct {
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Preview string    `json:"preview"` // First user message
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// GetConversationsDir returns the conversations directory for the given working directory.
// Conversations are stored under <workDir>/.emmai/conversations/.
func GetConversationsDir(workDir string) string {
	return filepath.Join(workDir, config.ConfigDir, conversationsDir)
}

// SaveConversation saves a conversation to <workDir>/.emmai/conversations/.
func SaveConversation(conv *client.Conversation, workDir string) error {
	dir := GetConversationsDir(workDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create conversations dir: %w", err)
	}

	filename := filepath.Join(dir, conv.ID+".json")
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal conversation: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write conversation file: %w", err)
	}

	return nil
}

// LoadConversation loads a conversation by ID from workDir.
func LoadConversation(id, workDir string) (*client.Conversation, error) {
	filename := filepath.Join(GetConversationsDir(workDir), id+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read conversation file: %w", err)
	}

	var conv client.Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("unmarshal conversation: %w", err)
	}

	return &conv, nil
}

// LoadMostRecent loads the most recently updated conversation from workDir.
func LoadMostRecent(workDir string) (*client.Conversation, error) {
	metadatas, err := ListConversations(workDir)
	if err != nil {
		return nil, err
	}

	if len(metadatas) == 0 {
		return nil, nil
	}

	sort.Slice(metadatas, func(i, j int) bool {
		return metadatas[i].Updated.After(metadatas[j].Updated)
	})

	return LoadConversation(metadatas[0].ID, workDir)
}

// ListConversations returns metadata for all saved conversations in workDir.
func ListConversations(workDir string) ([]ConversationMetadata, error) {
	dir := GetConversationsDir(workDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConversationMetadata{}, nil
		}
		return nil, fmt.Errorf("read conversations dir: %w", err)
	}

	metadatas := make([]ConversationMetadata, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		conv, err := LoadConversation(id, workDir)
		if err != nil {
			continue
		}

		preview := "New conversation"
		for _, msg := range conv.Messages {
			if msg.Role == "user" {
				preview = truncate(msg.Content, 100)
				break
			}
		}

		metadatas = append(metadatas, ConversationMetadata{
			ID:      conv.ID,
			Model:   conv.Model,
			Preview: preview,
			Created: conv.Created,
			Updated: conv.Updated,
		})
	}

	return metadatas, nil
}

// DeleteConversation removes a conversation from workDir.
func DeleteConversation(id, workDir string) error {
	filename := filepath.Join(GetConversationsDir(workDir), id+".json")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete conversation file: %w", err)
	}
	return nil
}

// truncate shortens a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
