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

# Response Style
- Short, concise responses
- Technical accuracy and facts over emotion
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

// DefaultPhases defines the default workflow phases
var DefaultPhases = []PhaseConfig{
	{
		Name:      "explore",
		ReadOnly:  true,
		NextPhase: "plan",
		AllowedTools: []string{
			"list_files",
			"read_file",
			// start_phase automatically injected
		},
		Prompt: `# EXPLORE - Gather Information Only

⛔ DO NOT write or suggest any code in this phase
✅ DO read files and understand what exists

Steps:
1. list_files("**/*") - check if project is empty
2. If empty: Report "Empty project - ready for new implementation" → THEN call start_phase("plan")
3. If not empty: read_file on key files (max 5 files), then report:
   - What type of project is this?
   - What files/structure exists?
   - What does it currently do?
4. ALWAYS finish by calling: start_phase("plan")

NO code suggestions. Just report facts. Implementation comes later.
`,
	},
	{
		Name:      "plan",
		ReadOnly:  true,
		NextPhase: "execute",
		AllowedTools: []string{
			"list_files",
			"read_file",
			// start_phase automatically injected
		},
		Prompt: `# PLAN - Create Implementation Plan

⛔ DO NOT write or suggest code snippets
⛔ DO NOT read files (already explored in previous phase)
✅ DO create a clear, structured implementation plan

Your Task: Based on exploration findings, plan HOW to implement the requested changes.

Plan Structure:
1. FILES TO MODIFY: List each file with brief purpose
2. FILES TO CREATE: List each new file with brief purpose
3. KEY CHANGES: Bullet points of main modifications
4. POTENTIAL RISKS: Any edge cases or breaking changes

Keep it concise - this is a roadmap, not code.

Steps:
1. Think through the implementation approach
2. List files to create/modify
3. Describe what changes go in each file (1-2 sentences per file)
4. Ask user: "Does this plan look good? Should I proceed?"
5. If user approves → call start_phase("execute")
6. If user has concerns → adjust plan based on feedback

NO code examples. Just describe WHAT will change WHERE. Code comes in execute phase.
`,
	},
	{
		Name:      "execute",
		ReadOnly:  false,
		NextPhase: "verify",
		AllowedTools: []string{
			"list_files",
			"read_file",
			"write_file",
			// start_phase automatically injected
		},
		Prompt: `# EXECUTE - Implement Changes

✅ DO write code and modify files
✅ DO follow the plan from previous phase
⛔ DO NOT deviate from the plan without user approval
⛔ DO NOT skip files or make partial implementations

Your Task: Implement the planned changes exactly as specified.

Execution Rules:
1. Work through files in order (modify existing first, then create new)
2. Follow existing code style, patterns, and conventions
3. Make minimal, focused changes - no scope creep
4. Read files before editing to understand context
5. Report progress as you complete each file

Steps:
1. Review the plan from the previous phase
2. For each file in the plan:
   a. Read the file (if existing)
   b. Make the planned changes
   c. Report: "✓ Updated [filename] - [brief description]"
3. When ALL files are complete → call start_phase("verify")

Quality checklist before moving to verify:
- All planned files created/modified?
- Code follows existing patterns?
- No syntax errors introduced?
- Functionality matches plan?

IMPORTANT: If you encounter issues or need to deviate from the plan, STOP and ask the user first.
`,
	},
	{
		Name:      "verify",
		ReadOnly:  true,
		NextPhase: "explore",
		AllowedTools: []string{
			"list_files",
			"read_file",
			// start_phase automatically injected
		},
		Prompt: `# VERIFY - Confirm Implementation Quality

✅ DO run tests, linters, and builds
✅ DO review modified files for correctness
⛔ DO NOT modify files (read-only phase)
⛔ DO NOT skip verification steps

Your Task: Verify the implementation works correctly and meets requirements.

Verification Checklist:
1. BUILD: Run build command (make, npm build, go build, etc.)
2. TESTS: Run test suite if available (make test, npm test, pytest, etc.)
3. LINT: Run linters/formatters if configured (eslint, gofmt, black, etc.)
4. REVIEW: Read modified files and check for:
   - Syntax errors or typos
   - Logic matches the plan
   - Code style consistency
   - No unintended changes

Steps:
1. Run build/compile command → report results
2. Run tests (if test files exist) → report pass/fail count
3. Run linters (if config exists) → report warnings/errors
4. Review each modified file briefly
5. Create final summary

Final Summary Format:
✓ COMPLETED: [List what was implemented]
✓ VERIFIED: [Build status, test results, lint status]
⚠ ISSUES: [Any problems found, or "None"]
→ NEXT: [Suggest next steps or mark as complete]

If verification fails: Report the issues and ask user how to proceed (fix, rollback, or accept as-is).

This is the final phase. Provide a comprehensive report before completing the task.

IMPORTANT: After providing the final summary, call start_phase("explore") to reset for the next task.
`,
	},
}
