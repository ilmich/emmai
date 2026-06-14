package ui

import (
	"os"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/indexer"
	"github.com/ilmich/emmai/internal/phase"
	"github.com/ilmich/emmai/internal/tools/execution"
	"github.com/ilmich/emmai/internal/tools/file"
	toolindex "github.com/ilmich/emmai/internal/tools/index"
)

// SetupModel initializes a new Model with all tools and phase configuration
func SetupModel(cfg *config.Config, aiClient *client.OpenAIClient) Model {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	// Build or load codebase index (best-effort; ref holds nil if it fails)
	idx, _ := indexer.BuildOrLoad(wd)
	idxRef := indexer.NewIndexRef(idx)

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

	// Register query_index tool
	queryTool := toolindex.NewQueryIndexTool()
	aiClient.RegisterTool(queryTool)

	// Create phase controller and attach summary provider from the live index ref
	phaseController := phase.NewController(phaseManager, aiClient)
	phaseController.SetSummaryProvider(func() string {
		return indexer.Summary(idxRef.Get())
	})

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

	// Register edit_file handler — wrapped to rebuild index after each successful edit
	editExecutor := file.NewEditExecutor(wd)
	executor.RegisterHandler("edit_file", func(args map[string]interface{}) (string, error) {
		result, err := editExecutor.HandleEditFile(args)
		if err == nil {
			go rebuildIndex(wd, idxRef)
		}
		return result, err
	})

	// Register run_command handler
	commandExecutor := execution.NewCommandExecutor(wd, &cfg.Security.CommandExecution, phaseManager)
	executor.RegisterHandler("run_command", commandExecutor.HandleRunCommand)

	// Register query_index handler — backed by the live index ref
	queryExecutor := toolindex.NewQueryExecutor(idxRef)
	executor.RegisterHandler("query_index", queryExecutor.HandleQueryIndex)

	// Set executor on client
	aiClient.SetToolExecutor(executor)

	// Initialize phase (injects prompt + index summary)
	initializePhase(phaseManager, aiClient, idxRef)

	// Create and return model
	return NewModel(cfg, aiClient, phaseManager, phaseController)
}

// initializePhase injects the initial phase prompt, prepending the codebase index summary if available.
func initializePhase(phaseManager *phase.Manager, aiClient *client.OpenAIClient, idxRef *indexer.IndexRef) {
	initialPhase := phaseManager.GetInitialPhase()
	if initialPhase == "" {
		return
	}

	response, err := phaseManager.StartPhase(initialPhase)
	if err != nil {
		return
	}

	prompt := response.Prompt
	if summary := indexer.Summary(idxRef.Get()); summary != "" {
		prompt = summary + "\n\n" + prompt
	}

	aiClient.SetPhasePrompt(prompt)

	allowedTools := phaseManager.GetCurrentPhaseAllowedTools()
	aiClient.SetPhaseAllowedTools(allowedTools)
}

// rebuildIndex rebuilds the codebase index in the background and updates the ref.
func rebuildIndex(wd string, idxRef *indexer.IndexRef) {
	newIdx, err := indexer.Build(wd)
	if err != nil {
		return
	}
	idxRef.Set(newIdx)
	_ = indexer.Save(newIdx)
}
