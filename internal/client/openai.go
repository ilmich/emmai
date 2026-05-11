package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ilmich/emmai/internal/config"
	"github.com/sashabaranov/go-openai"
)

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	client       *openai.Client
	config       *config.Config
	conversation *Conversation
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(cfg *config.Config) (*OpenAIClient, error) {
	// API key validation already done in config.Validate()
	// Use empty string if not provided (for LocalAI and similar)
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = "" // Explicitly use empty string for go-openai SDK
	}

	// Create OpenAI client config with API key
	clientConfig := openai.DefaultConfig(apiKey)
	
	// Set custom base URL if different from default
	if cfg.BaseURL != config.DefaultBaseURL {
		clientConfig.BaseURL = cfg.BaseURL
	}
	
	// Configure TLS certificate verification
	if cfg.InsecureSkipVerify {
		// Create custom HTTP client that skips TLS verification
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		clientConfig.HTTPClient = &http.Client{Transport: tr}
	}

	return &OpenAIClient{
		client:       openai.NewClientWithConfig(clientConfig),
		config:       cfg,
		conversation: NewConversation(cfg.Model),
	}, nil
}

// SendMessage sends a user message and returns streaming response channels
func (c *OpenAIClient) SendMessage(ctx context.Context, userMsg string) (<-chan string, <-chan error) {
	textChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		// Add user message to conversation history
		c.conversation.AddMessage("user", userMsg)

		// Build messages for API
		messages := c.buildAPIMessages()

		// Create streaming request
		req := openai.ChatCompletionRequest{
			Model:       c.config.Model,
			Messages:    messages,
			Temperature: c.config.Temperature,
			MaxTokens:   c.config.MaxTokens,
			Stream:      true,
		}

		stream, err := c.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			errChan <- fmt.Errorf("create completion stream: %w", err)
			return
		}
		defer stream.Close()

	var fullResponse strings.Builder

	// Stream response chunks
	for {
		// Check if context was cancelled (ESC pressed)
		select {
		case <-ctx.Done():
			// Context cancelled - exit cleanly
			return
		default:
			// Continue processing
		}

		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			errChan <- fmt.Errorf("receive stream: %w", err)
			return
		}

		if len(response.Choices) > 0 {
			chunk := response.Choices[0].Delta.Content
			if chunk != "" {
				fullResponse.WriteString(chunk)
				textChan <- chunk
			}
		}
	}

		// Add assistant response to conversation history
		c.conversation.AddMessage("assistant", fullResponse.String())
	}()

	return textChan, errChan
}

// buildAPIMessages converts conversation history to OpenAI API format
func (c *OpenAIClient) buildAPIMessages() []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(c.conversation.Messages)+1)

	// Add system prompt if configured and not already in history
	if c.config.SystemPrompt != "" {
		hasSystemMessage := false
		for _, msg := range c.conversation.Messages {
			if msg.Role == "system" {
				hasSystemMessage = true
				break
			}
		}
		if !hasSystemMessage {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: c.config.SystemPrompt,
			})
		}
	}

	// Add conversation history
	for _, msg := range c.conversation.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

// GetConversation returns the current conversation
func (c *OpenAIClient) GetConversation() *Conversation {
	return c.conversation
}

// LoadConversation loads an existing conversation
func (c *OpenAIClient) LoadConversation(conv *Conversation) {
	c.conversation = conv
}

// ClearConversation starts a new conversation
func (c *OpenAIClient) ClearConversation() {
	c.conversation = NewConversation(c.config.Model)
}

// GetTokenCount estimates the number of tokens used
func (c *OpenAIClient) GetTokenCount() int {
	// Rough estimate: ~4 characters per token
	total := 0
	for _, msg := range c.conversation.Messages {
		total += len(msg.Content) / 4
	}
	if c.config.SystemPrompt != "" {
		total += len(c.config.SystemPrompt) / 4
	}
	return total
}
