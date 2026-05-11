package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/storage"
	"github.com/rivo/tview"
)

// App is the main TUI application
type App struct {
	app              *tview.Application
	client           *client.OpenAIClient
	config           *config.Config
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
}

// NewApp creates a new TUI application
func NewApp(cfg *config.Config, aiClient *client.OpenAIClient) *App {
	app := tview.NewApplication()
	ctx, cancel := context.WithCancel(context.Background())

	tuiApp := &App{
		app:          app,
		client:       aiClient,
		config:       cfg,
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

	return tuiApp
}

// createHelpBar creates the help bar at the bottom
func (a *App) createHelpBar() *tview.TextView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	tv.SetBackgroundColor(tcell.ColorDefault)

	help := " Enter -> Send | Shift+Enter -> New Line | ESC -> Stop | Ctrl+L -> Clear | Ctrl+R -> Retry | Ctrl+Q -> Quit "
	fmt.Fprint(tv, help)

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
		// Interrupt streaming
		if a.isProcessing {
			a.interruptStreaming()
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
