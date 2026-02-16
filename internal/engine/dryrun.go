package engine

import (
	"fmt"
	"strings"
)

// DryRunAction describes what would happen to a file during apply.
type DryRunAction string

const (
	DryRunCreate DryRunAction = "create"
	DryRunUpdate DryRunAction = "update"
	DryRunSkip   DryRunAction = "skip"
)

// DryRunOp represents a single file operation that would be performed.
type DryRunOp struct {
	Action   DryRunAction
	RelPath  string      // path relative to project root
	Target   string      // target name (e.g. "vscode-copilot")
	Kind     ApplyOpKind // skill, instruction, agent, prompt
	Resource string      // resource name (e.g. "tdd")
	Diff     string      // unified diff (for updates)
	Reason   string      // reason (for skips)
}

// String returns a plain-text representation of the operation.
func (op DryRunOp) String() string {
	switch op.Action {
	case DryRunCreate:
		return fmt.Sprintf("[create]  %s", op.RelPath)
	case DryRunUpdate:
		return fmt.Sprintf("[update]  %s", op.RelPath)
	case DryRunSkip:
		if op.Reason != "" {
			return fmt.Sprintf("[skip]    %s (%s)", op.RelPath, op.Reason)
		}
		return fmt.Sprintf("[skip]    %s", op.RelPath)
	default:
		return fmt.Sprintf("[%s]  %s", op.Action, op.RelPath)
	}
}

// ANSI escape codes for colored terminal output.
const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiCyan   = "\033[36m"
	ansiDim    = "\033[2m"
)

// ColoredString returns an ANSI-colored representation of the operation.
func (op DryRunOp) ColoredString() string {
	switch op.Action {
	case DryRunCreate:
		return fmt.Sprintf("%s[create]%s  %s", ansiGreen, ansiReset, op.RelPath)
	case DryRunUpdate:
		return fmt.Sprintf("%s[update]%s  %s", ansiYellow, ansiReset, op.RelPath)
	case DryRunSkip:
		if op.Reason != "" {
			return fmt.Sprintf("%s[skip]    %s (%s)%s", ansiDim, op.RelPath, op.Reason, ansiReset)
		}
		return fmt.Sprintf("%s[skip]    %s%s", ansiDim, op.RelPath, ansiReset)
	default:
		return fmt.Sprintf("[%s]  %s", op.Action, op.RelPath)
	}
}

// FormatDryRunSummary returns a human-readable summary of operations.
// Returns "Nothing to apply." for an empty list.
func FormatDryRunSummary(ops []DryRunOp) string {
	if len(ops) == 0 {
		return "Nothing to apply."
	}

	var created, updated, skipped int
	for _, op := range ops {
		switch op.Action {
		case DryRunCreate:
			created++
		case DryRunUpdate:
			updated++
		case DryRunSkip:
			skipped++
		}
	}

	var parts []string
	if created > 0 {
		parts = append(parts, fmt.Sprintf("%d would be created", created))
	}
	if updated > 0 {
		parts = append(parts, fmt.Sprintf("%d would be updated", updated))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d would be skipped", skipped))
	}

	return strings.Join(parts, ", ")
}

// ColorDiff colorizes a unified diff string for terminal output.
// Lines starting with '+' are green, '-' are red, '@@' are cyan.
func ColorDiff(diff string) string {
	if diff == "" {
		return ""
	}

	lines := splitLines(diff)
	var buf strings.Builder
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "@@"):
			fmt.Fprintf(&buf, "%s%s%s\n", ansiCyan, line, ansiReset)
		case strings.HasPrefix(line, "+"):
			fmt.Fprintf(&buf, "%s%s%s\n", ansiGreen, line, ansiReset)
		case strings.HasPrefix(line, "-"):
			fmt.Fprintf(&buf, "%s%s%s\n", ansiRed, line, ansiReset)
		default:
			fmt.Fprintf(&buf, "%s\n", line)
		}
	}
	return buf.String()
}
