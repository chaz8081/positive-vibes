package engine

import (
	"fmt"
	"strings"
)

// diffOp represents the type of edit operation.
type diffOp int

const (
	opEqual diffOp = iota
	opDelete
	opInsert
)

// edit represents a single line-level edit operation.
type edit struct {
	op   diffOp
	text string
	idxA int // 0-based line index in file A (-1 if not applicable)
	idxB int // 0-based line index in file B (-1 if not applicable)
}

// hunk represents a contiguous group of changes with surrounding context.
type hunk struct {
	startA int // 1-based start line in file A
	countA int // number of lines from file A
	startB int // 1-based start line in file B
	countB int // number of lines from file B
	lines  []edit
}

// unifiedDiff produces a unified diff string between two texts.
// Returns an empty string if the texts are identical.
func unifiedDiff(nameA, nameB, a, b string) string {
	if a == b {
		return ""
	}

	linesA := splitLines(a)
	linesB := splitLines(b)
	const maxDiffCells = 10_000_000
	if len(linesA)*len(linesB) > maxDiffCells {
		return fmt.Sprintf("--- %s\n+++ %s\n@@ -0,0 +0,0 @@\n(diff omitted: files too large)\n", nameA, nameB)
	}

	edits := diffLines(linesA, linesB)
	hunks := groupHunks(edits, 3)

	if len(hunks) == 0 {
		return ""
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "--- %s\n", nameA)
	fmt.Fprintf(&buf, "+++ %s\n", nameB)

	for _, h := range hunks {
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", h.startA, h.countA, h.startB, h.countB)
		for _, e := range h.lines {
			switch e.op {
			case opEqual:
				fmt.Fprintf(&buf, " %s\n", e.text)
			case opDelete:
				fmt.Fprintf(&buf, "-%s\n", e.text)
			case opInsert:
				fmt.Fprintf(&buf, "+%s\n", e.text)
			}
		}
	}

	return buf.String()
}

// splitLines splits a string into lines, stripping a trailing empty element
// that results from a trailing newline.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Remove trailing empty string from trailing newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// diffLines computes the edit sequence between two slices of lines using
// an O(mn) LCS dynamic programming approach.
func diffLines(a, b []string) []edit {
	m := len(a)
	n := len(b)

	// Build LCS DP table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to produce edit sequence (in reverse)
	var edits []edit
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			edits = append(edits, edit{op: opEqual, text: a[i-1], idxA: i - 1, idxB: j - 1})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			edits = append(edits, edit{op: opInsert, text: b[j-1], idxA: -1, idxB: j - 1})
			j--
		} else {
			edits = append(edits, edit{op: opDelete, text: a[i-1], idxA: i - 1, idxB: -1})
			i--
		}
	}

	// Reverse to get forward order
	for left, right := 0, len(edits)-1; left < right; left, right = left+1, right-1 {
		edits[left], edits[right] = edits[right], edits[left]
	}

	return edits
}

// groupHunks groups a sequence of edits into hunks with ctx lines of context
// around each change. Adjacent hunks that would overlap are merged.
func groupHunks(edits []edit, ctx int) []hunk {
	// Find indices of changed edits
	var changes []int
	for i, e := range edits {
		if e.op != opEqual {
			changes = append(changes, i)
		}
	}
	if len(changes) == 0 {
		return nil
	}

	// Group changes into ranges, merging when context overlaps
	type span struct{ start, end int } // indices into edits slice [start, end)
	var spans []span

	spanStart := changes[0] - ctx
	if spanStart < 0 {
		spanStart = 0
	}
	spanEnd := changes[0] + 1 + ctx
	if spanEnd > len(edits) {
		spanEnd = len(edits)
	}

	for _, ci := range changes[1:] {
		newStart := ci - ctx
		if newStart < 0 {
			newStart = 0
		}
		newEnd := ci + 1 + ctx
		if newEnd > len(edits) {
			newEnd = len(edits)
		}

		if newStart <= spanEnd {
			// Merge: extend current span
			spanEnd = newEnd
		} else {
			spans = append(spans, span{spanStart, spanEnd})
			spanStart = newStart
			spanEnd = newEnd
		}
	}
	spans = append(spans, span{spanStart, spanEnd})

	// Convert spans to hunks
	var hunks []hunk
	for _, s := range spans {
		h := hunk{}
		h.lines = edits[s.start:s.end]

		// Compute 1-based start lines and counts
		// Find the first edit's position in A and B
		lineA := 0 // 0-based line counter in A
		lineB := 0 // 0-based line counter in B
		for i := 0; i < s.start; i++ {
			switch edits[i].op {
			case opEqual:
				lineA++
				lineB++
			case opDelete:
				lineA++
			case opInsert:
				lineB++
			}
		}

		h.startA = lineA + 1
		h.startB = lineB + 1
		h.countA = 0
		h.countB = 0
		for _, e := range h.lines {
			switch e.op {
			case opEqual:
				h.countA++
				h.countB++
			case opDelete:
				h.countA++
			case opInsert:
				h.countB++
			}
		}

		hunks = append(hunks, h)
	}

	return hunks
}
