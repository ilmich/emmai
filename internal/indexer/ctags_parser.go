package indexer

import (
	"regexp"
	"strings"
)

type langPattern struct {
	kind    string
	re      *regexp.Regexp
	nameIdx int // capture group for symbol name
}

// langPatterns maps file extension to ordered list of patterns.
var langPatterns = map[string][]langPattern{
	".py": {
		{kind: "func", re: regexp.MustCompile(`^def\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^class\s+(\w+)`), nameIdx: 1},
	},
	".js": jsPatterns(),
	".ts": jsPatterns(),
	".jsx": jsPatterns(),
	".tsx": jsPatterns(),
	".rs": {
		{kind: "func", re: regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`), nameIdx: 1},
		{kind: "struct", re: regexp.MustCompile(`^(?:pub\s+)?struct\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^(?:pub\s+)?enum\s+(\w+)`), nameIdx: 1},
		{kind: "interface", re: regexp.MustCompile(`^(?:pub\s+)?trait\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^(?:pub\s+)?impl\s+(\w+)`), nameIdx: 1},
	},
	".c": cPatterns(),
	".h": cPatterns(),
	".cpp": cPatterns(),
	".cc": cPatterns(),
	".hpp": cPatterns(),
	".java": {
		{kind: "type", re: regexp.MustCompile(`\bclass\s+(\w+)`), nameIdx: 1},
		{kind: "interface", re: regexp.MustCompile(`\binterface\s+(\w+)`), nameIdx: 1},
		{kind: "func", re: regexp.MustCompile(`^\s+(?:public|private|protected|static|final|\s)*[\w<>\[\]]+\s+(\w+)\s*\(`), nameIdx: 1},
	},
	".rb": {
		{kind: "func", re: regexp.MustCompile(`^\s*def\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^\s*class\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^\s*module\s+(\w+)`), nameIdx: 1},
	},
	".php": {
		{kind: "func", re: regexp.MustCompile(`^\s*(?:public|private|protected|static|\s)*function\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^\s*class\s+(\w+)`), nameIdx: 1},
		{kind: "interface", re: regexp.MustCompile(`^\s*interface\s+(\w+)`), nameIdx: 1},
	},
	".lua": {
		{kind: "func", re: regexp.MustCompile(`^(?:local\s+)?function\s+(\w+)`), nameIdx: 1},
		{kind: "func", re: regexp.MustCompile(`^(?:local\s+)?(\w+)\s*=\s*function`), nameIdx: 1},
	},
	".swift": {
		{kind: "func", re: regexp.MustCompile(`^\s*(?:public|private|internal|open|fileprivate|\s)*func\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^\s*(?:public|private|internal|open|\s)*class\s+(\w+)`), nameIdx: 1},
		{kind: "struct", re: regexp.MustCompile(`^\s*(?:public|private|internal|\s)*struct\s+(\w+)`), nameIdx: 1},
		{kind: "interface", re: regexp.MustCompile(`^\s*(?:public|private|internal|\s)*protocol\s+(\w+)`), nameIdx: 1},
	},
	".kt": {
		{kind: "func", re: regexp.MustCompile(`^\s*(?:fun)\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^\s*(?:data\s+)?class\s+(\w+)`), nameIdx: 1},
		{kind: "interface", re: regexp.MustCompile(`^\s*interface\s+(\w+)`), nameIdx: 1},
	},
}

func jsPatterns() []langPattern {
	return []langPattern{
		{kind: "func", re: regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?class\s+(\w+)`), nameIdx: 1},
		{kind: "type", re: regexp.MustCompile(`^(?:export\s+)?(?:type|interface)\s+(\w+)`), nameIdx: 1},
		// arrow functions and const functions: export const foo = () => or export const foo = async () =>
		{kind: "func", re: regexp.MustCompile(`^(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s*)?\(`), nameIdx: 1},
	}
}

func cPatterns() []langPattern {
	return []langPattern{
		{kind: "type", re: regexp.MustCompile(`^(?:typedef\s+)?(?:struct|enum|union)\s+(\w+)`), nameIdx: 1},
		// simple function definition: return_type name(
		{kind: "func", re: regexp.MustCompile(`^[a-zA-Z_][\w\s\*]*\b(\w+)\s*\([^;]`), nameIdx: 1},
	}
}

// parseFileCtags extracts symbols from non-Go files using regex patterns.
func parseFileCtags(relPath string, lines []string) []Symbol {
	ext := fileExt(relPath)
	patterns, ok := langPatterns[ext]
	if !ok {
		return nil
	}

	var symbols []Symbol
	for lineNum, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		for _, p := range patterns {
			m := p.re.FindStringSubmatch(trimmed)
			if m == nil || p.nameIdx >= len(m) {
				continue
			}
			name := m[p.nameIdx]
			if name == "" {
				continue
			}
			symbols = append(symbols, Symbol{
				Name: name,
				Kind: p.kind,
				File: relPath,
				Line: lineNum + 1,
			})
			break // only first matching pattern per line
		}
	}
	return symbols
}

func fileExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return ""
	}
	return strings.ToLower(path[idx:])
}
