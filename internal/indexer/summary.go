package indexer

import (
	"fmt"
	"sort"
	"strings"
)

// Summary returns a compact string suitable for prompt injection.
func Summary(idx *Index) string {
	if idx == nil || len(idx.Files) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("<codebase_index>\n## CODEBASE INDEX\n")

	// Language summary line
	langs := countLangs(idx.Files)
	primary, total := primaryLang(langs)
	pkgs := countPackages(idx.Files)
	sb.WriteString(fmt.Sprintf("%s project · %d files · %d packages · %d symbols\n\n",
		primary, total, pkgs, len(idx.Symbols)))

	// Key files (entry points + configs)
	keyFiles := pickKeyFiles(idx.Files)
	if len(keyFiles) > 0 {
		sb.WriteString("### Key files\n")
		sb.WriteString(strings.Join(keyFiles, " · ") + "\n\n")
	}

	// Package list (Go)
	if pkgs > 0 {
		sb.WriteString("### Packages\n")
		sb.WriteString(formatPackages(idx.Files) + "\n\n")
	}

	sb.WriteString("</codebase_index>\n")
	return sb.String()
}

func countLangs(files []FileEntry) map[string]int {
	m := make(map[string]int)
	for _, f := range files {
		if f.Language != "" {
			m[f.Language]++
		}
	}
	return m
}

func primaryLang(langs map[string]int) (string, int) {
	total := 0
	best, bestCount := "unknown", 0
	for lang, count := range langs {
		total += count
		if count > bestCount {
			best, bestCount = lang, count
		}
	}
	return best, total
}

func countPackages(files []FileEntry) int {
	seen := make(map[string]bool)
	for _, f := range files {
		if f.Package != "" {
			seen[f.Package] = true
		}
	}
	return len(seen)
}

func pickKeyFiles(files []FileEntry) []string {
	priority := map[string]int{
		"main.go": 10, "go.mod": 9, "Makefile": 8, "README.md": 7,
		"package.json": 9, "Cargo.toml": 9, "pyproject.toml": 9,
		"setup.py": 8, "Dockerfile": 7, "docker-compose.yml": 7,
	}
	type scored struct {
		path  string
		score int
	}
	var hits []scored
	for _, f := range files {
		base := f.Path[strings.LastIndex(f.Path, "/")+1:]
		if s, ok := priority[base]; ok {
			hits = append(hits, scored{f.Path, s})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	result := make([]string, 0, len(hits))
	for _, h := range hits {
		result = append(result, h.path)
	}
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

func formatPackages(files []FileEntry) string {
	// group files by package
	pkgFiles := make(map[string][]string)
	for _, f := range files {
		if f.Package != "" {
			base := f.Path[strings.LastIndex(f.Path, "/")+1:]
			pkgFiles[f.Package] = append(pkgFiles[f.Package], base)
		}
	}
	var parts []string
	pkgs := make([]string, 0, len(pkgFiles))
	for p := range pkgFiles {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, pkg := range pkgs {
		fnames := pkgFiles[pkg]
		if len(fnames) > 4 {
			fnames = append(fnames[:4], fmt.Sprintf("…+%d", len(fnames)-4))
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", pkg, strings.Join(fnames, ", ")))
	}
	return strings.Join(parts, " · ")
}

