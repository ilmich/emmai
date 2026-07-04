package ui

import (
	"os"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
)

// SetupModel initializes a new Model
func SetupModel(cfg *config.Config, aiClient *client.OpenAIClient) Model {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	executor := client.NewSimpleToolExecutor()
	aiClient.SetToolExecutor(executor)

	m := NewModel(cfg, aiClient, wd)

	if cfg.ContextSize == 0 {
		m.warnMessage = "context_size not set — compaction disabled"
	}

	return m
}
