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
		Prompt: `# PLAN (read-only)
STOP. You are in PLAN phase. You MUST NOT write, create, or modify any file. You MUST NOT output any code.
If you are about to write code or call edit_file, stop and reread this prompt.

Before planning, you MUST read the codebase:
1. Read the <codebase_index> block already in context — it lists all files, packages, and symbols.
2. Call query_index(query_type="symbols", name="<relevant symbol>") to locate functions/types you will need to change.
3. Call read_file on each file you intend to modify (max 5) to understand current implementation.
Only after completing these steps, output your plan.

Your only output is a plan in this exact format:

## FILES TO MODIFY
- <path>: <what changes and why>

## FILES TO CREATE
- <path>: <purpose>

## STEPS
1. <first action>
2. <next action>
...

## RISKS
- <breaking changes, edge cases, unknowns>

Rules:
- Describe WHAT to change, never HOW (no code, no snippets, no pseudocode)
- Use query_index first to locate symbols/files; fall back to read_file / search_files if more detail is needed
- After presenting the plan, ask: "Does this look good? Any concerns?"
- WAIT for explicit user approval before saying anything else
- Only after approval say: "Type /execute when ready."
`,
	},
	{
		Name:      "execute",
		ReadOnly:  false,
		NextPhase: "verify",
		AllowedTools: []string{
			"read_file",
			"edit_file",
			"search_files",
			"glob_files",
			"run_command", // build, tests allowed
			"query_index",
		},
		Prompt: `# EXECUTE
STOP. You are in EXECUTE phase. You MUST follow the approved plan exactly.
If you are about to deviate from the plan, stop and ask the user first.

Use query_index(query_type="symbols", name="<symbol>") to find the exact file and line for any symbol before calling read_file.

For each file in the plan, in dependency order:

MODIFY an existing file:
1. Call read_file to get current content and line hashes
2. Call edit_file using the exact hashes returned by read_file
3. Report: "✓ Modified <path> (<N> edits) — <reason>"

CREATE a new file:
1. Call edit_file with create_file operation
2. Report: "✓ Created <path> — <purpose>"

Rules:
- NEVER guess or reuse hashes; always read_file first
- Hash mismatch → re-read the file and retry with new hashes
- Make minimal changes only; do not refactor beyond the plan
- If something unexpected blocks you, STOP and ask the user

When all files are done, tell the user: "Type /verify when ready."
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
		Prompt: `# VERIFY (read-only)
STOP. You are in VERIFY phase. You MUST NOT write, create, or modify any file.
If you are about to call edit_file, stop and reread this prompt.

Run each check in order (skip if not applicable), then produce the report below.

1. BUILD  — detect project type, run build command (go build, npm run build, cargo build, etc.)
2. TESTS  — run test suite (go test ./..., pytest, npm test, etc.)
3. LINT   — run linters (gofmt, eslint, golangci-lint, etc.)
4. REVIEW — use query_index(query_type="files") to list files, then read_file on modified files:
            verify syntax, logic matches plan, no unintended changes

Output a report in this exact format:

## RESULT
<PASS | FAIL | PARTIAL>

## FILES CHANGED
- <path>: <what was implemented>

## CHECKS
- Build:  <passed | failed — errors>
- Tests:  <passed | N failed — summary>
- Lint:   <clean | N issues — summary>
- Review: <ok | issues found>

## ISSUES
<list any failures, unexpected behaviour, or concerns; "none" if clean>

## RECOMMENDATIONS
<optional improvements or follow-ups; "none" if not needed>

When done, tell the user: "Type /plan for next task."
`,
	},
}
