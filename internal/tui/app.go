package tui

import (
	"context"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/phase"
	"github.com/ilmich/emmai/internal/storage"
	"github.com/ilmich/emmai/internal/tools/execution"
	"github.com/ilmich/emmai/internal/tools/file"
	phasetool "github.com/ilmich/emmai/internal/tools/phase"
	"github.com/rivo/tview"
)

// App is the main TUI application
type App struct {
	app              *tview.Application
	client           *client.OpenAIClient
	config           *config.Config
	phaseManager     *phase.Manager
	bannerView       *BannerView
	systemMessageBar *SystemMessageBar
	chatView         *ChatView
	inputBox         *InputBox
	statusBar        *StatusBar
	helpBar          *tview.TextView
	layout           *tview.Flex
	isProcessing     bool
	ctx              context.Context
	cancel           context.CancelFunc
	streamCtx        context.Context
	streamCancel     context.CancelFunc
	toolExecutor     *client.SimpleToolExecutor
	phaseExecutor    *phasetool.PhaseExecutor
}

// NewApp creates a new TUI application
func NewApp(cfg *config.Config, aiClient *client.OpenAIClient) *App {
	app := tview.NewApplication()
	ctx, cancel := context.WithCancel(context.Background())

	// Create phase manager
	phaseManager := phase.NewManager(cfg.Phases, cfg.InitialPhase)

	tuiApp := &App{
		app:          app,
		client:       aiClient,
		config:       cfg,
		phaseManager: phaseManager,
		isProcessing: false,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Create components
	tuiApp.bannerView = NewBannerView()
	tuiApp.systemMessageBar = NewSystemMessageBar(app)
	tuiApp.chatView = NewChatView()
	tuiApp.inputBox = NewInputBox(app, tuiApp.handleSendMessage)
	tuiApp.statusBar = NewStatusBar(app, cfg.Model, cfg.IsCustomEndpoint())
	tuiApp.helpBar = tuiApp.createHelpBar()

	// Build layout
	tuiApp.layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tuiApp.statusBar, 1, 0, false).
		AddItem(tuiApp.bannerView, 5, 0, false).
		AddItem(tuiApp.systemMessageBar, 3, 0, false).
		AddItem(tuiApp.chatView, 0, 1, false).
		AddItem(tuiApp.inputBox, 5, 0, true).		
		AddItem(tuiApp.helpBar, 1, 0, false)

	// Set up global key bindings
	tuiApp.app.SetInputCapture(tuiApp.handleGlobalKeys)

	// Load most recent conversation if available
	tuiApp.loadRecentConversation()

	// Set up coding tools
	tuiApp.setupCodingTools()

	// Auto-inject initial phase
	tuiApp.initializePhase()

	return tuiApp
}

// initializePhase automatically injects the initial phase prompt
func (a *App) initializePhase() {
	initialPhase := a.phaseManager.GetInitialPhase()
	if initialPhase == "" {
		return
	}

	// Start the initial phase
	response, err := a.phaseManager.StartPhase(initialPhase)
	if err != nil {
		// Log error but don't fail - app can work without phases
		fmt.Printf("Warning: Failed to initialize phase %s: %v\n", initialPhase, err)
		return
	}

	// Inject phase prompt into client
	a.client.SetPhasePrompt(response.Prompt)

	// Set allowed tools for initial phase
	allowedTools := a.phaseManager.GetCurrentPhaseAllowedTools()
	a.client.SetPhaseAllowedTools(allowedTools)
}

// createHelpBar creates the help bar at the bottom
func (a *App) createHelpBar() *tview.TextView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	tv.SetBackgroundColor(tcell.ColorDefault)

	help := " [Enter] Send | [Shift+Enter] New Line | [ESC] Stop | [Ctrl+L] Clear | [Ctrl+R] Retry | [Ctrl+Q] Quit "
	fmt.Fprint(tv, tview.Escape(help))

	return tv
}

// handleGlobalKeys processes global keyboard shortcuts
func (a *App) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	switch {
	case event.Key() == tcell.KeyCtrlQ:
		// Quit
		a.Shutdown()
		return nil

	case event.Key() == tcell.KeyCtrlL:
		// Clear conversation
		if !a.isProcessing {
			a.clearConversation()
		}
		return nil

	case event.Key() == tcell.KeyCtrlR:
		// Retry last message
		if !a.isProcessing {
			a.retryLastMessage()
		}
		return nil

	case event.Key() == tcell.KeyCtrlC:
		// Force quit
		a.app.Stop()
		return nil

	case event.Key() == tcell.KeyEsc:
		// First ESC: Interrupt streaming
		if a.isProcessing {
			a.interruptStreaming()
			return nil
		}

		// Second ESC (when not processing): Reset to initial phase
		if a.phaseExecutor != nil {
			_ = a.phaseExecutor.ResetPhase()
			// Silent reset - no message shown to user
		}
		return nil
	}

	return event
}

// interruptStreaming cancels the current streaming response
func (a *App) interruptStreaming() {
	if a.streamCancel != nil {
		a.streamCancel()
	}
}

// handleSendMessage processes user message submission
func (a *App) handleSendMessage(message string) {
	if a.isProcessing || message == "" {
		return
	}

	a.isProcessing = true
	a.inputBox.Disable()
	a.statusBar.StartLoading()

	// Add user message to chat
	a.chatView.AddUserMessage(message)

	// Start streaming assistant response
	a.chatView.StartStreamingMessage()

	// Create cancellable context for this stream
	a.streamCtx, a.streamCancel = context.WithCancel(a.ctx)

	// Send to OpenAI
	textChan, errChan := a.client.SendMessage(a.streamCtx, message)

	go func() {
		for {
			select {
		case <-a.streamCtx.Done():
			// Stream was interrupted by ESC key
			a.app.QueueUpdateDraw(func() {
				// Remove the partial streaming message
				a.chatView.RemoveLastMessage()
				// Remove the user message from UI
				a.chatView.RemoveLastMessage()
				// Remove the user message from conversation history
				a.client.RemoveLastUserMessage()
				a.finishProcessing()
			})
			return

			case chunk, ok := <-textChan:
				if !ok {
					// Stream finished successfully
					a.app.QueueUpdateDraw(func() {
						a.finishProcessing()
						// Auto-save conversation
						if err := storage.SaveConversation(a.client.GetConversation()); err != nil {
							// Silently fail - don't interrupt user experience
						}
					})
					return
				}
				// Append chunk to last message
				a.app.QueueUpdateDraw(func() {
					a.chatView.AppendToLastMessage(chunk)
					a.statusBar.SetTokens(a.client.GetTokenCount())
				})

			case err := <-errChan:
				if err != nil {
					a.app.QueueUpdateDraw(func() {
						a.handleError(err)
						a.finishProcessing()
					})
					return
				}
			}
		}
	}()
}

// finishProcessing resets the UI after message processing
func (a *App) finishProcessing() {
	a.isProcessing = false
	a.inputBox.Enable()
	a.statusBar.StopLoading()
	a.statusBar.SetTokens(a.client.GetTokenCount())

	// Clean up stream context
	if a.streamCancel != nil {
		a.streamCancel()
		a.streamCancel = nil
		a.streamCtx = nil
	}
}

// handleError displays an error message
func (a *App) handleError(err error) {
	errorMsg := fmt.Sprintf("Failed to get response: %v\n\nPress Ctrl+R to retry or type a new message.", err)
	a.chatView.AddErrorMessage(errorMsg)
}

// clearConversation starts a new conversation
func (a *App) clearConversation() {
	a.client.ClearConversation()
	a.chatView.Clear()
	a.statusBar.SetTokens(0)

	// Reset to initial phase
	a.initializePhase()

	a.systemMessageBar.SetMessage("Started new conversation")
}

// retryLastMessage resends the last user message
func (a *App) retryLastMessage() {
	lastMsg := a.client.GetConversation().GetLastUserMessage()
	if lastMsg == "" {
		a.systemMessageBar.SetMessage("No message to retry")
		return
	}

	// Remove last assistant message if exists
	conv := a.client.GetConversation()
	if len(conv.Messages) > 0 && conv.Messages[len(conv.Messages)-1].Role == "assistant" {
		conv.Messages = conv.Messages[:len(conv.Messages)-1]
	}

	// Resend
	a.handleSendMessage(lastMsg)
}

// loadRecentConversation loads the most recent conversation if available
func (a *App) loadRecentConversation() {
	conv, err := storage.LoadMostRecent()
	if err != nil || conv == nil {
		// No conversation to load or error - start fresh
		a.systemMessageBar.SetMessage(fmt.Sprintf("Welcome to EmmAI! Using model: %s", a.config.Model))
		return
	}

	// Load conversation
	a.client.LoadConversation(conv)
	a.chatView.LoadMessages(conv.Messages)
	a.statusBar.SetTokens(a.client.GetTokenCount())

	a.systemMessageBar.SetMessage("Loaded previous conversation")
}

// Run starts the TUI application
func (a *App) Run(ctx context.Context) error {
	a.ctx = ctx
	return a.app.SetRoot(a.layout, true).EnableMouse(true).Run()
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() {
	// Save conversation before exit
	if err := storage.SaveConversation(a.client.GetConversation()); err != nil {
		// Best effort - ignore errors
	}

	a.cancel()
	a.app.Stop()
}

// setupCodingTools registers all coding assistant tools
func (a *App) setupCodingTools() {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	// Register read_file tool
	readTool := file.NewReadFileTool()
	a.client.RegisterTool(readTool)

	// Register search_files tool
	searchTool := file.NewSearchFilesTool()
	a.client.RegisterTool(searchTool)

	// Register glob_files tool
	globTool := file.NewGlobFilesTool()
	a.client.RegisterTool(globTool)

	// Register edit_file tool
	editTool := file.NewEditFileTool()
	a.client.RegisterTool(editTool)

	// Register run_command tool
	commandTool := execution.NewRunCommandTool()
	a.client.RegisterTool(commandTool)

	// Register start_phase tool
	phaseTool := phasetool.NewPhaseTool()
	a.client.RegisterTool(phaseTool)

	// Create tool executor
	executor := client.NewSimpleToolExecutor()

	// Register read_file handler
	readExecutor := file.NewReadExecutor(wd)
	executor.RegisterHandler("read_file", readExecutor.HandleReadFile)

	// Register search_files handler
	searchExecutor := file.NewSearchExecutor(wd)
	executor.RegisterHandler("search_files", searchExecutor.HandleSearchFiles)

	// Register glob_files handler
	globExecutor := file.NewGlobExecutor(wd)
	executor.RegisterHandler("glob_files", globExecutor.HandleGlobFiles)

	// Register edit_file handler
	editExecutor := file.NewEditExecutor(wd)
	executor.RegisterHandler("edit_file", editExecutor.HandleEditFile)

	// Register run_command handler
	commandExecutor := execution.NewCommandExecutor(wd, &a.config.Security.CommandExecution, a.phaseManager)
	executor.RegisterHandler("run_command", commandExecutor.HandleRunCommand)

	// Register start_phase handler
	a.phaseExecutor = phasetool.NewPhaseExecutor(a.phaseManager, a.client)
	executor.RegisterHandler("start_phase", a.phaseExecutor.HandleStartPhase)

	// Set executor on client
	a.client.SetToolExecutor(executor)
	a.toolExecutor = executor
}
