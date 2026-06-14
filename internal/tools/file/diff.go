package file

import (
	"fmt"
	"strings"
)

const (
	diffContextLines = 3
	maxDiffLines     = 3000
	maxDiffBytes     = 10240 // 10KB
)

type diffOpKind int

const (
	diffOpEqual  diffOpKind = iota
	diffOpInsert
	diffOpDelete
)

type rawOp struct {
	kind diffOpKind
	text string
}

type editLine struct {
	marker rune
	text   string
	origNo int // 1-based line in original (for ' ' and '-')
	newNo  int // 1-based line in modified (for ' ' and '+')
}

// unifiedDiff returns a unified diff between original and modified content.
// Returns empty string when identical or too large to diff.
func unifiedDiff(original, modified, path string) string {
	if original == modified {
		return ""
	}
	if original == "" {
		return newFileDiff(modified, path)
	}

	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	if len(origLines) > maxDiffLines || len(modLines) > maxDiffLines {
		return fmt.Sprintf("(diff omitted — file exceeds %d lines)\n", maxDiffLines)
	}

	script := buildEditLines(origLines, modLines)
	result := renderUnifiedDiff(script, path)
	if len(result) > maxDiffBytes {
		return result[:maxDiffBytes] + "\n... (diff truncated)\n"
	}
	return result
}

func newFileDiff(content, path string) string {
	lines := strings.Split(content, "\n")
	var sb strings.Builder
	sb.WriteString("--- /dev/null\n")
	fmt.Fprintf(&sb, "+++ b/%s\n", path)
	fmt.Fprintf(&sb, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, l := range lines {
		sb.WriteByte('+')
		sb.WriteString(l)
		sb.WriteByte('\n')
	}
	result := sb.String()
	if len(result) > maxDiffBytes {
		return result[:maxDiffBytes] + "\n... (diff truncated)\n"
	}
	return result
}

func buildEditLines(a, b []string) []editLine {
	dp := computeLCS(a, b)
	raw := backtrackLCS(dp, len(b)+1, a, b)

	lines := make([]editLine, 0, len(raw))
	oi, ni := 1, 1
	for _, r := range raw {
		switch r.kind {
		case diffOpEqual:
			lines = append(lines, editLine{' ', r.text, oi, ni})
			oi++
			ni++
		case diffOpDelete:
			lines = append(lines, editLine{'-', r.text, oi, 0})
			oi++
		case diffOpInsert:
			lines = append(lines, editLine{'+', r.text, 0, ni})
			ni++
		}
	}
	return lines
}

func computeLCS(a, b []string) []int32 {
	m, n := len(a), len(b)
	w := n + 1
	dp := make([]int32, (m+1)*w)
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i*w+j] = dp[(i-1)*w+(j-1)] + 1
			} else if dp[(i-1)*w+j] >= dp[i*w+(j-1)] {
				dp[i*w+j] = dp[(i-1)*w+j]
			} else {
				dp[i*w+j] = dp[i*w+(j-1)]
			}
		}
	}
	return dp
}

func backtrackLCS(dp []int32, w int, a, b []string) []rawOp {
	var ops []rawOp
	i, j := len(a), len(b)
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			ops = append(ops, rawOp{diffOpEqual, a[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i*w+(j-1)] >= dp[(i-1)*w+j]) {
			ops = append(ops, rawOp{diffOpInsert, b[j-1]})
			j--
		} else {
			ops = append(ops, rawOp{diffOpDelete, a[i-1]})
			i--
		}
	}
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

func renderUnifiedDiff(lines []editLine, path string) string {
	var changedIdx []int
	for i, l := range lines {
		if l.marker != ' ' {
			changedIdx = append(changedIdx, i)
		}
	}
	if len(changedIdx) == 0 {
		return ""
	}

	type hunkRange struct{ s, e int }
	var hunks []hunkRange
	n := len(lines)
	for i := 0; i < len(changedIdx); {
		s := max(0, changedIdx[i]-diffContextLines)
		j := i
		for j+1 < len(changedIdx) && changedIdx[j+1] <= changedIdx[j]+2*diffContextLines+1 {
			j++
		}
		e := min(n-1, changedIdx[j]+diffContextLines)
		hunks = append(hunks, hunkRange{s, e})
		i = j + 1
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)

	for _, hr := range hunks {
		hlines := lines[hr.s : hr.e+1]
		origStart, newStart, origCount, newCount := 0, 0, 0, 0
		for _, l := range hlines {
			if l.marker == ' ' || l.marker == '-' {
				if origStart == 0 {
					origStart = l.origNo
				}
				origCount++
			}
			if l.marker == ' ' || l.marker == '+' {
				if newStart == 0 {
					newStart = l.newNo
				}
				newCount++
			}
		}
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", origStart, origCount, newStart, newCount)
		for _, l := range hlines {
			sb.WriteByte(byte(l.marker))
			sb.WriteString(l.text)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}
