package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/ui"
)

func main() {
	var modelProfile string
	var listModels bool
	flag.StringVar(&modelProfile, "model", "", "model profile name from ~/.emmai/config.yaml (required)")
	flag.BoolVar(&listModels, "list-models", false, "list all configured model profiles and exit")
	flag.Parse()

	if listModels {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}
		names := cfg.ProfileNames()
		if len(names) == 0 {
			fmt.Fprintf(os.Stderr, "No profiles defined. Add a 'models:' section to %s\n", config.GetConfigPath())
			os.Exit(1)
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if modelProfile == "" {
		names := cfg.ProfileNames()
		if len(names) == 0 {
			fmt.Fprintf(os.Stderr, "Error: -model is required\n")
			fmt.Fprintf(os.Stderr, "Usage: emmai -model <profile>\n\n")
			fmt.Fprintf(os.Stderr, "No profiles defined. Add a 'models:' section to %s\n", config.GetConfigPath())
			os.Exit(1)
		}
		first := names[0]
		fmt.Fprintf(os.Stderr, "No model selected. Use %q? [Y/n]: ", first)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" || line == "y" || line == "yes" {
			modelProfile = first
		} else {
			fmt.Fprintf(os.Stderr, "\nAvailable profiles:\n")
			for _, n := range names {
				fmt.Fprintf(os.Stderr, "  %s\n", n)
			}
			fmt.Fprintf(os.Stderr, "Usage: emmai -model <profile>\n")
			os.Exit(1)
		}
	}

	if err := cfg.ApplyProfile(modelProfile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if names := cfg.ProfileNames(); len(names) > 0 {
			fmt.Fprintf(os.Stderr, "\nAvailable profiles:\n")
			for _, n := range names {
				fmt.Fprintf(os.Stderr, "  %s\n", n)
			}
		} else {
			fmt.Fprintf(os.Stderr, "\nNo profiles defined. Add a 'models:' section to %s\n", config.GetConfigPath())
		}
		os.Exit(1)
	}
	cfg.BaseURL = config.NormalizeBaseURL(cfg.BaseURL)

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Ensure config directory exists
	if err := config.EnsureConfigDir(); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// Initialize OpenAI client
	aiClient, err := client.NewOpenAIClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	// Create Bubble Tea model with all tools configured
	model := ui.SetupModel(cfg, aiClient)

	// Create Bubble Tea program with alt screen and mouse support
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the application
	if _, err := p.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
