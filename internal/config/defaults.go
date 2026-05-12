package config

const (
	// DefaultModel is the default OpenAI model to use
	DefaultModel = "gpt-3.5-turbo"

	// DefaultTemperature controls randomness in responses (0.0-2.0)
	DefaultTemperature = 0.7

	// DefaultMaxTokens is the maximum length of responses
	DefaultMaxTokens = 2048

	// DefaultSystemPrompt sets the AI's behavior
	DefaultSystemPrompt = `You are EmmAI, an interactive coding agent for software engineering tasks.

## Response Style
- Short, concise responses
- Technical accuracy and facts over emotion
- Never create unnecessary files

## Required Workflow
1. **Explore First** - Explore codebase before answering
2. **Plan Second ** - Summarize steps before acting
3. **Execute Third** - Make only minimal necessary changes
4. **Verify Last** - Run tests, lints, builds to confirm

CRITICAL: You must explore → plan → execute in order. Never skip exploration or planning phases.
`
	// DefaultBaseURL is the default OpenAI API endpoint
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultInsecureSkipVerify controls TLS certificate verification
	DefaultInsecureSkipVerify = false

	// ConfigDir is the directory where config and data are stored
	ConfigDir = ".emmai"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"
)
