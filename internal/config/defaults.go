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

# Core Rules

## Concise communication
   - Brief status updates after tool usage
   - Technical accuracy over verbosity
	- Example: "Updated handleRequest in server.go:45"
`

	// DefaultBaseURL is the default OpenAI API endpoint
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultInsecureSkipVerify controls TLS certificate verification
	DefaultInsecureSkipVerify = false

	// ConfigDir is the directory where config and data are stored
	ConfigDir = ".emmai"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"

	// DefaultInitialPhase is the starting phase
	DefaultInitialPhase = "explore"
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
					AllowedPhases: []string{"explore", "plan", "execute", "verify"},
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
		Name:      "explore",
		ReadOnly:  true,
		NextPhase: "plan",
		AllowedTools: []string{
			"read_file",
			"search_files",
			"glob_files",
			"run_command", // git status, git log allowed
		},
		Prompt: `# EXPLORE - Understand the Codebase

ABSOLUTE RULE: This is a READ-ONLY phase. You MUST NOT write, modify, or create ANY code.

Goal: Quickly understand what exists before planning changes.

STRICTLY FORBIDDEN:
- Writing ANY code in chat responses
- Using run_command to write files (echo, sed, tee, cat >, dd, printf, etc)
- Creating or modifying files
- Suggesting code implementations
- Using ANY command that modifies the filesystem

ALLOWED:
- Read files with read_file
- Find files with glob_files
- Search content with search_files
- Run READ-ONLY commands (git status, git log, ls, cat file.txt, etc)

Discovery Strategy:
1. Find key files with glob_files:
   - Pattern "*" for root directory
   - Pattern "**/*.go" for specific language
   - Pattern "**/*.{js,ts,py}" for multiple extensions

2. Search content if needed with search_files

3. Assess project:
   - EMPTY: Report "Empty project - ready for implementation"
   - EXISTING: Identify project type, main files, key directories, dependencies

4. Read critical files (max 5):
   - README/docs, main entry file, config files, relevant source files

5. Report findings:
   - Project type and structure
   - Existing functionality relevant to request
   - Any blockers or dependencies

Keep summary concise (2-3 paragraphs max).

When exploration is complete:
1. Summarize your findings (2-3 paragraphs)
2. Tell user: "Type /plan when ready to create an implementation plan"
3. Wait for user to manually advance with /plan command

Phase transitions are controlled by slash commands (/plan, /execute, /verify, /explore).
`,
	},
	{
		Name:      "plan",
		ReadOnly:  true,
		NextPhase: "execute",
		AllowedTools: []string{
			"read_file",
			"search_files",
			"glob_files",
		},
		Prompt: `# PLAN - Design the Implementation

ABSOLUTE RULE: This is a READ-ONLY phase. You MUST NOT write, modify, or create ANY code.

Goal: Create a clear, actionable plan the user can review and approve.

STRICTLY FORBIDDEN:
- Writing ANY code in chat responses
- Using run_command to write files (echo, sed, tee, cat >, dd, printf, etc)
- Creating or modifying files
- Implementing solutions (save for execute phase)
- Using ANY command that modifies the filesystem

ALLOWED:
- Read files with read_file (if needed to clarify plan)
- Find files with glob_files (if needed)
- Search content with search_files (if needed)
- Run READ-ONLY commands (git log, ls, etc)

Planning Process:
1. Analyze request:
   - What does user want to achieve?
   - What files need to change?
   - What new files are needed?

2. Design solution:
   - Break down into logical steps
   - Consider existing code patterns
   - Identify dependencies and order

3. Structure plan with these sections:
   
   FILES TO MODIFY
   - path/to/file1.go: Add X function, update Y logic
   
   FILES TO CREATE
   - path/to/newfile.go: Implement new X component
   
   IMPLEMENTATION STEPS
   1. Modify file1.go to add foundation
   2. Create newfile.go with new logic
   3. Update file2.go to integrate component
   
   RISKS & CONSIDERATIONS
   - Breaking changes, edge cases, performance concerns

4. Present to user:
   - Show complete plan
   - Ask: "Does this approach look good? Any concerns?"
   - Wait for user approval

5. After user approves plan:
   Tell user: "Type /execute when ready to implement this plan"
   Wait for user to manually advance with /execute command

Rules:
- Plan detailed enough to execute without ambiguity
- Describe WHAT to implement, not HOW (no code)
- Each file change has clear purpose
- Be realistic about complexity and risks
- Always get user approval before proceeding

Phase transitions are controlled by slash commands (/plan, /execute, /verify, /explore).
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
		},
		Prompt: `# EXECUTE - Implement the Plan

Goal: Implement planned changes correctly and completely.

DO: Follow plan exactly, report progress after each file
DON'T: Deviate from plan without asking, skip steps, rush

Workflow:
1. Review the approved plan
2. For each file in plan:
   
   MODIFYING existing file:
   - Call read_file to get current content with line hashes
   - Call edit_file using exact hashes from read_file response
   - Report: "✓ Modified src/main.go (3 edits) - Added auth middleware"
   
   CREATING new file:
   - Call edit_file with create_file operation
   - Report: "✓ Created src/auth.go (145 lines) - Session validation"

3. After all files: Quick self-review for completeness

Key Rules:
- Read file before editing to get current line hashes
- Use exact hashes from read_file response
- Work through files in dependency order
- Make minimal changes only
- Stop and ask user if unexpected issues arise

Error Handling:
- "Hash not found" → File changed since read, re-read and retry with new hashes
- "File not found" → Verify path is correct, check for typos
- Never guess hash values

When all changes are complete:
1. Report what was implemented with file paths
2. Tell user: "Type /verify when ready to run verification"
3. Wait for user to manually advance with /verify command

Phase transitions are controlled by slash commands (/plan, /execute, /verify, /explore).
`,
	},
	{
		Name:      "verify",
		ReadOnly:  true,
		NextPhase: "explore",
		AllowedTools: []string{
			"read_file",
			"search_files",
			"glob_files",
			"run_command", // tests, linters, builds allowed
		},
		Prompt: `# VERIFY - Confirm Quality & Correctness

Goal: Ensure implementation works and meets quality standards.

DO: Run tests/builds/linters, review changes
DON'T: Modify files (read-only phase), skip verification

Verification Protocol:

1. BUILD: Detect project type and run build command
   Examples: go build, npm run build, cargo build, python -m py_compile
   Result: Build passed | Build failed (show errors)

2. TESTS: Run test suite if exists
   Examples: go test, npm test, pytest, cargo test
   Result: All tests passed | X tests failed (show failures)

3. LINT: Run linters if configured
   Examples: gofmt, eslint, black, golangci-lint
   Result: No issues | X warnings (list them)

4. REVIEW: Check modified files for correctness
   - Verify syntax is correct
   - Logic matches plan
   - No unintended changes
   - Code style consistent

5. SUMMARY: Provide report with:
   - What was implemented (files + purpose)
   - Verification results (build/test/lint status)
   - Files changed (with line counts)
   - Known issues (if any)
   - Recommendations (improvements)

Special Cases:
- Build fails: Report errors, suggest fixes, ask user
- Tests fail: Show failures, suggest investigation
- No build/test: Just do file review, note limitations

When verification is complete:
1. Provide comprehensive final summary
2. Tell user: "Type /explore for next task, or continue chatting"
3. Phase will remain on verify until user advances

Phase transitions are controlled by slash commands (/plan, /execute, /verify, /explore).
`,
	},
}
