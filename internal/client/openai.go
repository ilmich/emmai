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
	client                   *openai.Client
	config                   *config.Config
	conversation             *Conversation
	toolRegistry             *ToolRegistry
	toolExecutor             ToolExecutor
	toolChoice               ToolChoice
	phasePrompt              string   // Current phase-specific prompt
	currentPhaseAllowedTools []string // Tools allowed in current phase
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
		client:                   openai.NewClientWithConfig(clientConfig),
		config:                   cfg,
		conversation:             NewConversation(cfg.Model),
		toolRegistry:             NewToolRegistry(),
		toolChoice:               "auto", // Default to auto tool selection
		phasePrompt:              "",     // No phase prompt initially
		currentPhaseAllowedTools: []string{}, // No phase filtering initially
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

		// Process with tool calling loop
		c.processWithTools(ctx, textChan, errChan)
	}()

	return textChan, errChan
}

// processWithTools handles the tool calling loop
func (c *OpenAIClient) processWithTools(ctx context.Context, textChan chan<- string, errChan chan<- error) {
	maxIterations := 10 // Prevent infinite loops
	for i := 0; i < maxIterations; i++ {
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

		// Add tools if any are registered
		if c.toolRegistry.HasTools() {
			// Get filtered tools based on current phase
			tools := c.getFilteredTools()
			openaiTools := make([]openai.Tool, len(tools))
			for i, tool := range tools {
				openaiTools[i] = openai.Tool{
					Type: openai.ToolType(tool.Type),
					Function: &openai.FunctionDefinition{
						Name:        tool.Function.Name,
						Description: tool.Function.Description,
						Parameters:  tool.Function.Parameters,
						Strict:      tool.Function.Strict,
					},
				}
			}
			req.Tools = openaiTools
			
			// Set tool choice if specified
			if c.toolChoice != nil {
				req.ToolChoice = c.toolChoice
			}
		}

		stream, err := c.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			errChan <- fmt.Errorf("create completion stream: %w", err)
			return
		}

		var fullResponse strings.Builder
		var toolCalls []ToolCall
		var currentToolCall *ToolCall
		var toolCallIndex int = -1

		// Stream response chunks
		for {
			// Check if context was cancelled (ESC pressed)
			select {
			case <-ctx.Done():
				stream.Close()
				return
			default:
				// Continue processing
			}

			response, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				stream.Close()
				errChan <- fmt.Errorf("receive stream: %w", err)
				return
			}

			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta
				
				// Handle regular content
				if delta.Content != "" {
					fullResponse.WriteString(delta.Content)
					textChan <- delta.Content
				}

				// Handle tool calls
				if len(delta.ToolCalls) > 0 {
					for _, tc := range delta.ToolCalls {
						// Check if this is a new tool call or continuation
						if tc.Index != nil && *tc.Index != toolCallIndex {
							// New tool call
							if currentToolCall != nil {
								toolCalls = append(toolCalls, *currentToolCall)
							}
							toolCallIndex = *tc.Index
							currentToolCall = &ToolCall{
								ID:   tc.ID,
								Type: string(tc.Type),
								Function: ToolCallFunction{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							}
						} else if currentToolCall != nil {
							// Continuation of current tool call
							if tc.Function.Name != "" {
								currentToolCall.Function.Name += tc.Function.Name
							}
							if tc.Function.Arguments != "" {
								currentToolCall.Function.Arguments += tc.Function.Arguments
							}
						}
					}
				}
			}
		}
		stream.Close()

		// Add the last tool call if any
		if currentToolCall != nil {
			toolCalls = append(toolCalls, *currentToolCall)
		}

		// Add assistant response to conversation
		msg := Message{
			Role:      "assistant",
			Content:   fullResponse.String(),
			ToolCalls: toolCalls,
			Timestamp: c.conversation.Messages[len(c.conversation.Messages)-1].Timestamp,
		}
		c.conversation.Messages = append(c.conversation.Messages, msg)

		// If there are tool calls, execute them
		if len(toolCalls) > 0 && c.toolExecutor != nil {
			for _, tc := range toolCalls {
				// Show tool call feedback to user
				textChan <- fmt.Sprintf("\n🔧 Calling %s\n", tc.Function.Name)
				
				// Show arguments if not too long
				args := tc.Function.Arguments
				if len(args) <= 200 {
					textChan <- fmt.Sprintf("   %s\n", args)
				} else {
					textChan <- fmt.Sprintf("   %s...\n", args[:197])
				}
				
				// Execute the tool
				result, err := c.toolExecutor.Execute(tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					textChan <- fmt.Sprintf("❌ Error: %v\n\n", err)
					result = fmt.Sprintf("Error executing tool: %v", err)
				} else {
					textChan <- "✅ Completed\n\n"
				}

				// Add tool response to conversation
				toolMsg := Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Timestamp:  c.conversation.Messages[len(c.conversation.Messages)-1].Timestamp,
				}
				c.conversation.Messages = append(c.conversation.Messages, toolMsg)
			}
			// Continue the loop to get the next response
			continue
		}

		// No tool calls, we're done
		break
	}
}

// buildAPIMessages converts conversation history to OpenAI API format
func (c *OpenAIClient) buildAPIMessages() []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(c.conversation.Messages)+1)

	// Build complete system prompt: base + phase
	systemPrompt := c.config.SystemPrompt
	if c.phasePrompt != "" {
		systemPrompt = systemPrompt + "\n\n" + c.phasePrompt
	}

	// Add system prompt if configured and not already in history
	if systemPrompt != "" {
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
				Content: systemPrompt,
			})
		}
	}

	// Add conversation history
	for _, msg := range c.conversation.Messages {
		apiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Handle tool calls in assistant messages
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			apiMsg.ToolCalls = toolCalls
		}

		// Handle tool response messages
		if msg.Role == "tool" {
			apiMsg.ToolCallID = msg.ToolCallID
			apiMsg.Name = msg.Name
		}

		messages = append(messages, apiMsg)
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

// RemoveLastUserMessage removes the last user message from conversation
func (c *OpenAIClient) RemoveLastUserMessage() {
	c.conversation.RemoveLastUserMessage()
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
	if c.phasePrompt != "" {
		total += len(c.phasePrompt) / 4
	}
	return total
}

// RegisterTool adds a tool to the client's tool registry
func (c *OpenAIClient) RegisterTool(tool Tool) {
	c.toolRegistry.RegisterTool(tool)
}

// SetToolExecutor sets the tool executor for handling tool calls
func (c *OpenAIClient) SetToolExecutor(executor ToolExecutor) {
	c.toolExecutor = executor
}

// SetToolChoice sets how the model should use tools
// Can be "none", "auto", "required", or a specific function
func (c *OpenAIClient) SetToolChoice(choice ToolChoice) {
	c.toolChoice = choice
}

// GetToolRegistry returns the tool registry
func (c *OpenAIClient) GetToolRegistry() *ToolRegistry {
	return c.toolRegistry
}

// SetPhasePrompt updates the current phase-specific prompt
func (c *OpenAIClient) SetPhasePrompt(prompt string) {
	c.phasePrompt = prompt
}

// GetPhasePrompt returns the current phase prompt
func (c *OpenAIClient) GetPhasePrompt() string {
	return c.phasePrompt
}

// SetPhaseAllowedTools updates the allowed tools for the current phase
func (c *OpenAIClient) SetPhaseAllowedTools(allowedTools []string) {
	c.currentPhaseAllowedTools = allowedTools
}

// getFilteredTools returns tools filtered by phase allowed list
func (c *OpenAIClient) getFilteredTools() []Tool {
	// If no phase filtering is active, return all tools
	if len(c.currentPhaseAllowedTools) == 0 {
		return c.toolRegistry.GetAllTools()
	}

	// Create lookup map for fast checking
	allowed := make(map[string]bool)
	for _, name := range c.currentPhaseAllowedTools {
		allowed[name] = true
	}

	// Filter tools
	filtered := make([]Tool, 0)
	for _, tool := range c.toolRegistry.GetAllTools() {
		if allowed[tool.Function.Name] {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}
