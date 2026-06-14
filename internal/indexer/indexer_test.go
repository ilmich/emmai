package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOnSelf(t *testing.T) {
	// Index the emmai repo itself (two levels up from this package)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up to repo root (internal/indexer → repo root)
	repoRoot, err := filepath.Abs(filepath.Join(wd, "../.."))
	if err != nil {
		t.Fatal(err)
	}

	idx, err := Build(repoRoot)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(idx.Files) == 0 {
		t.Error("expected files, got none")
	}
	if len(idx.Symbols) == 0 {
		t.Error("expected symbols, got none")
	}

	// Check that known symbols exist
	found := false
	for _, s := range idx.Symbols {
		if s.Name == "SetupModel" && s.Kind == "func" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find SetupModel func in symbols")
	}

	t.Logf("Indexed %d files, %d symbols", len(idx.Files), len(idx.Symbols))
}

func TestSummaryNonEmpty(t *testing.T) {
	wd, _ := os.Getwd()
	repoRoot, _ := filepath.Abs(filepath.Join(wd, "../.."))
	idx, err := Build(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	s := Summary(idx)
	if s == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.HasPrefix(s, "<codebase_index>") {
		t.Error("summary should start with <codebase_index>")
	}
	if !strings.HasSuffix(strings.TrimRight(s, "\n"), "</codebase_index>") {
		t.Error("summary should end with </codebase_index>")
	}
	t.Log(s)
}

func TestCtagsParserGo(t *testing.T) {
	// Go files should be handled by go_parser, not ctags
	syms := parseFileCtags("foo.go", []string{"func Hello() {}"})
	// .go extension has no ctags patterns — should return nil
	if syms != nil {
		t.Errorf("expected nil for .go, got %v", syms)
	}
}

func TestCtagsPython(t *testing.T) {
	lines := []string{
		"def my_func():",
		"class MyClass:",
		"    def method(self):",
	}
	syms := parseFileCtags("foo.py", lines)
	if len(syms) < 2 {
		t.Fatalf("expected at least 2 symbols, got %d", len(syms))
	}
	if syms[0].Name != "my_func" || syms[0].Kind != "func" {
		t.Errorf("unexpected first symbol: %+v", syms[0])
	}
	if syms[1].Name != "MyClass" || syms[1].Kind != "type" {
		t.Errorf("unexpected second symbol: %+v", syms[1])
	}
}

func TestCtagsTypeScript(t *testing.T) {
	lines := []string{
		"export async function fetchData() {",
		"export class UserService {",
		"export interface User {",
		"export const handler = async () => {",
	}
	syms := parseFileCtags("foo.ts", lines)
	if len(syms) != 4 {
		t.Fatalf("expected 4 symbols, got %d: %+v", len(syms), syms)
	}
}
