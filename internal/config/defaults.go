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
Be concise: brief status after tool use, technical accuracy over verbosity (e.g. "Updated handleRequest in server.go:45").
Phase transitions are controlled by slash commands: /plan /execute /verify /reset.`

	// DefaultBaseURL is the default OpenAI API endpoint
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultInsecureSkipVerify controls TLS certificate verification
	DefaultInsecureSkipVerify = false

	// ConfigDir is the directory where config and data are stored
	ConfigDir = ".emmai"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"

	// DefaultInitialPhase is the starting phase
	DefaultInitialPhase = "plan"
)

// DefaultSecurityPolicy returns the default security policy
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		CommandExecution: CommandExecutionPolicy{
			Enabled: true,
			AllowedCommands: []AllowedCommand{
				{
					Prefix:        "go",
					Subcommands:   []string{"test", "build", "mod", "fmt", "vet", "get"},
					BlockedArgs:   []string{"run"},
					AllowedPhases: []string{"execute", "verify"},
				},
				{
					Prefix:        "git",
					Subcommands:   []string{"status", "diff", "log", "branch", "show"},
					BlockedArgs:   []string{"push", "pull", "clone", "reset"},
					AllowedPhases: []string{"plan", "execute", "verify"},
				},
				{
					Prefix:        "make",
					Subcommands:   []string{"build", "test", "clean"},
					AllowedPhases: []string{"execute", "verify"},
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

// DefaultPhases defines the default workflow phases
var DefaultPhases = []PhaseConfig{
	{
		Name:      "plan",
		ReadOnly:  true,
		NextPhase: "execute",
		AllowedTools: []string{
			"read_file",
			"search_files",
			"glob_files",
			"query_index",
		},
		Prompt: `# PLAN (read-only, no file edits, no code output)

Do these steps in order:
1. Call query_index(query_type="files") to see all project files.
2. Call query_index(query_type="symbols", name="<relevant term>") to find functions/types to change.
3. Call read_file on each file you will modify (max 3).
4. Output the plan below.

## FILES TO MODIFY
- <path>: <what and why>

## FILES TO CREATE
- <path>: <purpose>

## STEPS
1. ...

## RISKS
- ...

Describe WHAT to change, not HOW. No code or snippets.
Ask "Does this look good?" and wait for approval, then say "Type /execute when ready."
`,
	},
	{
		Name:      "execute",
		ReadOnly:  false,
		NextPhase: "verify",
		AllowedTools: []string{
			"read_file",
			"edit_file",
			"delete_file",
			"search_files",
			"glob_files",
			"run_command", // build, tests allowed
			"query_index",
		},
		Prompt: `# EXECUTE (follow the approved plan exactly, no deviations)

For each file in the plan:

To MODIFY a file:
1. query_index(query_type="symbols", name="<symbol>") — locate exact file/line
2. read_file — get content and line hashes
3. edit_file — use hashes from step 2 (never guess; on mismatch re-read and retry)
4. Report: "✓ Modified <path> — <reason>"

To CREATE a file:
1. edit_file with create_file operation
2. Report: "✓ Created <path> — <purpose>"

Work through all files, then say: "Type /verify when ready."
If something unexpected blocks progress, stop and ask the user.
`,
	},
	{
		Name:      "verify",
		ReadOnly:  true,
		NextPhase: "plan",
		AllowedTools: []string{
			"read_file",
			"search_files",
			"glob_files",
			"run_command", // tests, linters, builds allowed
			"query_index",
		},
		Prompt: `# VERIFY (read-only, no file edits)

Run in order, skip if not applicable:
1. run_command — build (go build ./..., npm run build, cargo build, etc.)
2. run_command — tests (go test ./..., pytest, npm test, etc.)
3. run_command — lint (go vet ./..., eslint, gofmt -l, etc.)
4. read_file on each modified file — confirm logic matches plan, no unintended changes

Then output:

## RESULT
<PASS | FAIL | PARTIAL>
## CHECKS
- Build: <passed | failed: errors>
- Tests: <passed | N failed: summary>
- Lint:  <clean | issues: summary>
- Review: <ok | issues found>
## ISSUES
<failures or concerns; "none" if clean>

Tell the user: "Type /plan for next task."
`,
	},
}
