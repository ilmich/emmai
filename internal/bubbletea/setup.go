package bubbletea

import (
	"os"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/phase"
	"github.com/ilmich/emmai/internal/tools/execution"
	"github.com/ilmich/emmai/internal/tools/file"
)

// SetupModel initializes a new Model with all tools and phase configuration
func SetupModel(cfg *config.Config, aiClient *client.OpenAIClient) Model {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	// Create phase manager
	phaseManager := phase.NewManager(cfg.Phases, cfg.InitialPhase)

	// Register read_file tool
	readTool := file.NewReadFileTool()
	aiClient.RegisterTool(readTool)

	// Register search_files tool
	searchTool := file.NewSearchFilesTool()
	aiClient.RegisterTool(searchTool)

	// Register glob_files tool
	globTool := file.NewGlobFilesTool()
	aiClient.RegisterTool(globTool)

	// Register edit_file tool
	editTool := file.NewEditFileTool()
	aiClient.RegisterTool(editTool)

	// Register run_command tool
	commandTool := execution.NewRunCommandTool()
	aiClient.RegisterTool(commandTool)

	// Create phase controller for manual transitions (slash commands)
	phaseController := phase.NewController(phaseManager, aiClient)

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
	commandExecutor := execution.NewCommandExecutor(wd, &cfg.Security.CommandExecution, phaseManager)
	executor.RegisterHandler("run_command", commandExecutor.HandleRunCommand)

	// Set executor on client
	aiClient.SetToolExecutor(executor)

	// Initialize phase
	initializePhase(phaseManager, aiClient)

	// Create and return model
	return NewModel(cfg, aiClient, phaseManager, phaseController)
}

// initializePhase automatically injects the initial phase prompt
func initializePhase(phaseManager *phase.Manager, aiClient *client.OpenAIClient) {
	initialPhase := phaseManager.GetInitialPhase()
	if initialPhase == "" {
		return
	}

	// Start the initial phase
	response, err := phaseManager.StartPhase(initialPhase)
	if err != nil {
		return
	}

	// Inject phase prompt into client
	aiClient.SetPhasePrompt(response.Prompt)

	// Set allowed tools for initial phase
	allowedTools := phaseManager.GetCurrentPhaseAllowedTools()
	aiClient.SetPhaseAllowedTools(allowedTools)
}
