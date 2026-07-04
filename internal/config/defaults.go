package config

const (
	// DefaultModel is the default OpenAI model to use
	DefaultModel = "qwen-2.5-coder"

	// DefaultTemperature controls randomness in responses (0.0-2.0)
	DefaultTemperature = 0.2

	// DefaultMaxTokens is the maximum length of responses
	DefaultMaxTokens = 4096

	// DefaultSystemPrompt sets the AI's behavior
	DefaultSystemPrompt = `You are EmmAI, an interactive coding agent for software engineering tasks.
Be concise: technical accuracy over verbosity.

When providing code, always use this exact format — a filename line followed immediately by the code block::
` + "`" + `path/to/file.go` + "`" + `
` + "```" + `go
// code here
` + "```" + `

Rules:
- The filename MUST be a inline code span (backtick) on its own line, directly above the fenced code block.
- The fenced code block MUST declare the language (go, python, typescript, etc.).
- NEVER embed the filename inside the code block or in prose.
- If a response modifies multiple files, repeat the pattern for each file in order.`

	// DefaultBaseURL is the default OpenAI API endpoint
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultInsecureSkipVerify controls TLS certificate verification
	DefaultInsecureSkipVerify = false

	// ConfigDir is the directory where config and data are stored
	ConfigDir = ".emmai"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"
)

// DefaultSecurityPolicy returns the default security policy
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		CommandExecution: CommandExecutionPolicy{
			Enabled: true,
			AllowedCommands: []AllowedCommand{
				{
					Prefix:      "go",
					Subcommands: []string{"test", "build", "mod", "fmt", "vet", "get"},
					BlockedArgs: []string{"run"},
				},
				{
					Prefix:      "git",
					Subcommands: []string{"status", "diff", "log", "branch", "show"},
					BlockedArgs: []string{"push", "pull", "clone", "reset"},
				},
				{
					Prefix:      "make",
					Subcommands: []string{"build", "test", "clean"},
				},
			},
			DefaultTimeoutSec:  30,
			MaxTimeoutSec:      120,
			MaxConcurrent:      3,
			BlockedDirectories: []string{"/etc", "/root", "~/.ssh", "/var"},
			MaxOutputSizeBytes: 1048576, // 1MB
		},
	}
}

