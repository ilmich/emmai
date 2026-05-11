package config

const (
	// DefaultModel is the default OpenAI model to use
	DefaultModel = "gpt-3.5-turbo"

	// DefaultTemperature controls randomness in responses (0.0-2.0)
	DefaultTemperature = 0.7

	// DefaultMaxTokens is the maximum length of responses
	DefaultMaxTokens = 2048

	// DefaultSystemPrompt sets the AI's behavior
	DefaultSystemPrompt = `You are an expert software engineer and coding assistant. Your role is to provide working code solutions across multiple programming languages.

Guidelines:
- Follow language-specific best practices, idioms, and conventions
- Write clean, production-ready code with inline comments
- Ask clarifying questions when requirements are ambiguous
- Break complex goals into meaningful sub-tasks and use planner tool to track execution  
- Provide code solutions directly without verbose explanations

Be concise and practical. Focus on delivering code that works.`

	// DefaultBaseURL is the default OpenAI API endpoint
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultInsecureSkipVerify controls TLS certificate verification
	DefaultInsecureSkipVerify = false

	// ConfigDir is the directory where config and data are stored
	ConfigDir = ".emmai"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"
)
