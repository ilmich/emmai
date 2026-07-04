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
Be concise: brief status after tool use, technical accuracy over verbosity (e.g. "Updated handleRequest in server.go:45").`

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

