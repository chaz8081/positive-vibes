# Apply --dry-run Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Status:** Completed on 2026-02-15 (merged into main).

**Goal:** Add `--dry-run` / `-n` flag to `apply` that previews what would change without writing files, with optional `--verbose` / `-v` for unified diffs.

**Architecture:** Add a `DryRun bool` field to `InstallOpts` that threads through the applier and target install functions. When set, install functions compute what *would* happen (create vs update) and return structured results instead of writing. A new `internal/engine/dryrun.go` handles diff computation and colored terminal rendering. No third-party dependencies -- unified diff is computed in-house.

**Tech Stack:** Go stdlib only. `os.ReadFile` for reading existing files, custom line-based unified diff, ANSI escape codes for color output.

---

### Task 1: Unified diff utility

Create a standalone line-based unified diff function that produces standard unified diff output.

**Files:**
- Create: `internal/engine/diff.go`
- Create: `internal/engine/diff_test.go`

**Step 1: Write the failing test**

In `internal/engine/diff_test.go`:

```go
package engine

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_BothEmpty(t *testing.T) {
	result := unifiedDiff("a.txt", "b.txt", "", "")
	if result != "" {
		t.Fatalf("expected empty diff for identical empty files, got:\n%s", result)
	}
}

func TestUnifiedDiff_Identical(t *testing.T) {
	text := "line1\nline2\nline3\n"
	result := unifiedDiff("a.txt", "b.txt", text, text)
	if result != "" {
		t.Fatalf("expected empty diff for identical files, got:\n%s", result)
	}
}

func TestUnifiedDiff_Addition(t *testing.T) {
	a := "line1\nline2\n"
	b := "line1\nline2\nline3\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(result, "+line3") {
		t.Fatalf("expected +line3 in diff, got:\n%s", result)
	}
	if !strings.Contains(result, "--- a.txt") {
		t.Fatalf("expected --- a.txt header, got:\n%s", result)
	}
	if !strings.Contains(result, "+++ b.txt") {
		t.Fatalf("expected +++ b.txt header, got:\n%s", result)
	}
}

func TestUnifiedDiff_Deletion(t *testing.T) {
	a := "line1\nline2\nline3\n"
	b := "line1\nline2\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(result, "-line3") {
		t.Fatalf("expected -line3 in diff, got:\n%s", result)
	}
}

func TestUnifiedDiff_Modification(t *testing.T) {
	a := "line1\nold line\nline3\n"
	b := "line1\nnew line\nline3\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(result, "-old line") {
		t.Fatalf("expected -old line in diff, got:\n%s", result)
	}
	if !strings.Contains(result, "+new line") {
		t.Fatalf("expected +new line in diff, got:\n%s", result)
	}
}

func TestUnifiedDiff_ContextLines(t *testing.T) {
	a := "a\nb\nc\nd\ne\nf\ng\n"
	b := "a\nb\nc\nD\ne\nf\ng\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	// Should have context lines around the change
	if !strings.Contains(result, " c") {
		t.Fatalf("expected context line ' c' in diff, got:\n%s", result)
	}
	if !strings.Contains(result, " e") {
		t.Fatalf("expected context line ' e' in diff, got:\n%s", result)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestUnifiedDiff -v`
Expected: FAIL (function not defined)

**Step 3: Write minimal implementation**

In `internal/engine/diff.go`:

```go
package engine

import (
	"fmt"
	"strings"
)

// unifiedDiff computes a unified diff between two strings.
// Returns empty string if a and b are identical.
// Uses 3 lines of context (standard unified diff).
func unifiedDiff(nameA, nameB, a, b string) string {
	if a == b {
		return ""
	}

	linesA := splitLines(a)
	linesB := splitLines(b)

	// Compute LCS-based edit script
	edits := diffLines(linesA, linesB)
	hunks := groupHunks(edits, 3)

	if len(hunks) == 0 {
		return ""
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "--- %s\n", nameA)
	fmt.Fprintf(&buf, "+++ %s\n", nameB)

	for _, h := range hunks {
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", h.startA+1, h.countA, h.startB+1, h.countB)
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

type diffOp int

const (
	opEqual  diffOp = iota
	opDelete
	opInsert
)

type edit struct {
	op   diffOp
	text string
	idxA int // line index in A (-1 for inserts)
	idxB int // line index in B (-1 for deletes)
}

type hunk struct {
	startA, countA int
	startB, countB int
	lines          []edit
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Remove trailing empty string from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// diffLines computes a Myers-like diff producing a sequence of edits.
func diffLines(a, b []string) []edit {
	// Simple LCS-based approach using O(mn) DP
	m, n := len(a), len(b)
	// dp[i][j] = length of LCS of a[:i] and b[:j]
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

	// Backtrack to produce edits
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
	for l, r := 0, len(edits)-1; l < r; l, r = l+1, r-1 {
		edits[l], edits[r] = edits[r], edits[l]
	}

	return edits
}

// groupHunks groups edits into hunks with `ctx` lines of context.
func groupHunks(edits []edit, ctx int) []hunk {
	if len(edits) == 0 {
		return nil
	}

	// Find ranges of changes
	type changeRange struct{ start, end int }
	var changes []changeRange
	for i, e := range edits {
		if e.op != opEqual {
			if len(changes) == 0 || i > changes[len(changes)-1].end+2*ctx {
				changes = append(changes, changeRange{i, i})
			} else {
				changes[len(changes)-1].end = i
			}
		}
	}

	var hunks []hunk
	for _, cr := range changes {
		start := cr.start - ctx
		if start < 0 {
			start = 0
		}
		end := cr.end + ctx + 1
		if end > len(edits) {
			end = len(edits)
		}

		h := hunk{lines: edits[start:end]}

		// Compute line numbers
		for _, e := range h.lines {
			switch e.op {
			case opEqual:
				if h.startA == 0 && e.idxA >= 0 {
					h.startA = e.idxA
				}
				if h.startB == 0 && e.idxB >= 0 {
					h.startB = e.idxB
				}
				h.countA++
				h.countB++
			case opDelete:
				if h.startA == 0 && e.idxA >= 0 {
					h.startA = e.idxA
				}
				h.countA++
			case opInsert:
				if h.startB == 0 && e.idxB >= 0 {
					h.startB = e.idxB
				}
				h.countB++
			}
		}

		// Fix startA/startB from first relevant edit
		for _, e := range h.lines {
			if e.idxA >= 0 {
				h.startA = e.idxA
				break
			}
		}
		for _, e := range h.lines {
			if e.idxB >= 0 {
				h.startB = e.idxB
				break
			}
		}

		hunks = append(hunks, h)
	}

	return hunks
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestUnifiedDiff -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/engine/diff.go internal/engine/diff_test.go
git commit -m "feat(engine): add in-house unified diff utility"
```

---

### Task 2: DryRun types and rendering

Add `DryRun` to `InstallOpts` and create the dry-run result types and colored terminal renderer.

**Files:**
- Create: `internal/engine/dryrun.go`
- Create: `internal/engine/dryrun_test.go`
- Modify: `internal/target/target.go:15-18` (add `DryRun` to `InstallOpts`)

**Step 1: Write the failing test**

In `internal/engine/dryrun_test.go`:

```go
package engine

import (
	"strings"
	"testing"
)

func TestDryRunOp_String_Create(t *testing.T) {
	op := DryRunOp{
		Action:   DryRunCreate,
		RelPath:  ".github/skills/tdd/SKILL.md",
		Target:   "vscode-copilot",
		Kind:     KindSkill,
		Resource: "tdd",
	}
	s := op.String()
	if !strings.Contains(s, "[create]") {
		t.Fatalf("expected [create] in output, got: %s", s)
	}
	if !strings.Contains(s, ".github/skills/tdd/SKILL.md") {
		t.Fatalf("expected path in output, got: %s", s)
	}
}

func TestDryRunOp_String_Update(t *testing.T) {
	op := DryRunOp{
		Action:   DryRunUpdate,
		RelPath:  ".opencode/instructions/style.md",
		Target:   "opencode",
		Kind:     KindInstruction,
		Resource: "style",
	}
	s := op.String()
	if !strings.Contains(s, "[update]") {
		t.Fatalf("expected [update] in output, got: %s", s)
	}
}

func TestDryRunOp_String_Skip(t *testing.T) {
	op := DryRunOp{
		Action:   DryRunSkip,
		RelPath:  ".cursor/prompts/deploy.md",
		Target:   "cursor",
		Kind:     KindPrompt,
		Resource: "deploy",
		Reason:   "unsupported",
	}
	s := op.String()
	if !strings.Contains(s, "[skip]") {
		t.Fatalf("expected [skip] in output, got: %s", s)
	}
	if !strings.Contains(s, "unsupported") {
		t.Fatalf("expected reason in output, got: %s", s)
	}
}

func TestFormatDryRunSummary(t *testing.T) {
	ops := []DryRunOp{
		{Action: DryRunCreate},
		{Action: DryRunCreate},
		{Action: DryRunUpdate},
		{Action: DryRunSkip},
	}
	s := FormatDryRunSummary(ops)
	if !strings.Contains(s, "2 would be created") {
		t.Fatalf("expected '2 would be created', got: %s", s)
	}
	if !strings.Contains(s, "1 would be updated") {
		t.Fatalf("expected '1 would be updated', got: %s", s)
	}
	if !strings.Contains(s, "1 would be skipped") {
		t.Fatalf("expected '1 would be skipped', got: %s", s)
	}
}

func TestFormatDryRunSummary_Empty(t *testing.T) {
	s := FormatDryRunSummary(nil)
	if !strings.Contains(s, "Nothing to apply") {
		t.Fatalf("expected 'Nothing to apply', got: %s", s)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestDryRun -v`
Expected: FAIL (types not defined)

**Step 3: Add `DryRun` to `InstallOpts`**

In `internal/target/target.go`, modify lines 15-18:

```go
// InstallOpts controls how skills are installed.
type InstallOpts struct {
	Force  bool // overwrite existing skills
	Link   bool // create symlinks instead of copies
	DryRun bool // preview changes without writing
}
```

**Step 4: Write dry-run types and rendering**

In `internal/engine/dryrun.go`:

```go
package engine

import "fmt"

// DryRunAction describes what would happen to a file.
type DryRunAction string

const (
	DryRunCreate DryRunAction = "create"
	DryRunUpdate DryRunAction = "update"
	DryRunSkip   DryRunAction = "skip"
)

// DryRunOp describes a single file operation that would be performed.
type DryRunOp struct {
	Action   DryRunAction
	RelPath  string      // path relative to project root
	Target   string      // target name
	Kind     ApplyOpKind // skill, instruction, agent, prompt
	Resource string      // resource name
	Diff     string      // unified diff (only for updates, populated in verbose mode)
	Reason   string      // reason for skip
}

// DryRunResult collects all planned operations.
type DryRunResult struct {
	Ops    []DryRunOp
	Errors []string
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
		return fmt.Sprintf("[???]     %s", op.RelPath)
	}
}

// FormatDryRunSummary returns a summary line like "2 would be created, 1 would be updated".
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
	if len(parts) == 0 {
		return "Nothing to apply."
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// ColoredString returns an ANSI-colored representation of the operation.
func (op DryRunOp) ColoredString() string {
	const (
		green = "\033[32m"
		yellow = "\033[33m"
		dim   = "\033[2m"
		reset = "\033[0m"
		red   = "\033[31m"
	)

	switch op.Action {
	case DryRunCreate:
		return fmt.Sprintf("%s[create]%s  %s", green, reset, op.RelPath)
	case DryRunUpdate:
		return fmt.Sprintf("%s[update]%s  %s", yellow, reset, op.RelPath)
	case DryRunSkip:
		if op.Reason != "" {
			return fmt.Sprintf("%s[skip]    %s (%s)%s", dim, op.RelPath, op.Reason, reset)
		}
		return fmt.Sprintf("%s[skip]    %s%s", dim, op.RelPath, reset)
	default:
		return fmt.Sprintf("[???]     %s", op.RelPath)
	}
}

// ColorDiff colorizes a unified diff string with ANSI codes.
func ColorDiff(diff string) string {
	if diff == "" {
		return ""
	}
	const (
		green = "\033[32m"
		red   = "\033[31m"
		cyan  = "\033[36m"
		reset = "\033[0m"
	)

	var buf []byte
	lines := splitLines(diff)
	for _, line := range lines {
		switch {
		case len(line) > 0 && line[0] == '+':
			buf = append(buf, green...)
			buf = append(buf, line...)
			buf = append(buf, reset...)
		case len(line) > 0 && line[0] == '-':
			buf = append(buf, red...)
			buf = append(buf, line...)
			buf = append(buf, reset...)
		case len(line) > 0 && line[0] == '@':
			buf = append(buf, cyan...)
			buf = append(buf, line...)
			buf = append(buf, reset...)
		default:
			buf = append(buf, line...)
		}
		buf = append(buf, '\n')
	}
	return string(buf)
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/engine/ -run TestDryRun -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/engine/dryrun.go internal/engine/dryrun_test.go internal/target/target.go
git commit -m "feat(engine): add dry-run types, rendering, and DryRun option"
```

---

### Task 3: Wire dry-run into target install functions

Modify the four `install*Generic()` functions in `target.go` to return `DryRunOp` when `opts.DryRun` is true, instead of writing files.

**Files:**
- Modify: `internal/target/target.go` (all 4 generic install functions)
- Create: `internal/target/dryrun_test.go`

**Step 1: Write the failing tests**

In `internal/target/dryrun_test.go`:

```go
package target

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/pkg/schema"
)

func TestInstallGenericDryRun_NewSkill(t *testing.T) {
	projectRoot := t.TempDir()
	skill := &schema.Skill{Name: "test-skill", Description: "test", Instructions: "# Test"}
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("---\nname: test-skill\n---\n# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := InstallOpts{DryRun: true}
	ops, err := dryRunInstallGeneric(skill, sourceDir, projectRoot, ".opencode/skills", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("expected at least one dry-run op")
	}
	if ops[0].Action != "create" {
		t.Fatalf("expected create action, got %s", ops[0].Action)
	}

	// Verify nothing was written
	dest := filepath.Join(projectRoot, ".opencode", "skills", "test-skill")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create files, but dest exists")
	}
}

func TestInstallGenericDryRun_ExistingSkill(t *testing.T) {
	projectRoot := t.TempDir()
	skill := &schema.Skill{Name: "test-skill", Description: "test", Instructions: "# Updated"}

	// Pre-create the skill
	dest := filepath.Join(projectRoot, ".opencode", "skills", "test-skill")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("# Original"), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("---\nname: test-skill\n---\n# Updated"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := InstallOpts{DryRun: true}
	ops, err := dryRunInstallGeneric(skill, sourceDir, projectRoot, ".opencode/skills", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("expected at least one dry-run op")
	}
	if ops[0].Action != "update" {
		t.Fatalf("expected update action, got %s", ops[0].Action)
	}

	// Original content should be unchanged
	data, _ := os.ReadFile(filepath.Join(dest, "SKILL.md"))
	if string(data) != "# Original" {
		t.Fatalf("dry-run should not modify files")
	}
}

func TestDryRunInstallInstructionGeneric_New(t *testing.T) {
	projectRoot := t.TempDir()
	opts := InstallOpts{DryRun: true}
	op, err := dryRunInstallInstructionGeneric("style", "Use tabs.", "", projectRoot, ".opencode/instructions", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Action != "create" {
		t.Fatalf("expected create, got %s", op.Action)
	}

	// Verify nothing written
	if _, err := os.Stat(filepath.Join(projectRoot, ".opencode", "instructions", "style.md")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not write files")
	}
}

func TestDryRunInstallInstructionGeneric_Update(t *testing.T) {
	projectRoot := t.TempDir()
	instDir := filepath.Join(projectRoot, ".opencode", "instructions")
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instDir, "style.md"), []byte("Use spaces."), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := InstallOpts{DryRun: true}
	op, err := dryRunInstallInstructionGeneric("style", "Use tabs.", "", projectRoot, ".opencode/instructions", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Action != "update" {
		t.Fatalf("expected update, got %s", op.Action)
	}
	if op.Diff == "" {
		t.Fatal("expected diff for update")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/target/ -run TestDryRun -v`
Expected: FAIL (functions not defined)

**Step 3: Add dry-run functions to target.go**

Add `dryRunInstallGeneric()`, `dryRunInstallInstructionGeneric()`, `dryRunInstallAgentGeneric()`, `dryRunInstallPromptGeneric()` functions. These mirror the existing install functions but read existing files to compare and return `engine.DryRunOp` slices instead of writing.

Import `engine` package is problematic (circular dependency: engine imports target). Instead, define a `DryRunOp` in the `target` package mirroring the engine type, and have the applier convert. Actually, better approach: define a lightweight `FileOp` in `target` that the `engine` package converts.

Alternative and cleaner approach: put the dry-run preview computation in the `engine` package, using the existing `Target` interface's directory methods (`SkillDir()`, `InstructionDir()`, etc.) plus `fsutil.ResolveWithinRoot()`. The `target` install functions don't need modification at all -- the applier simply skips calling them in dry-run mode and computes the preview itself.

**Revised approach for Task 3:**

The `engine/applier.go` `ApplyManifest()` method will, when `opts.DryRun` is true, compute what each install would do by:
1. Resolving the dest path using the target's directory methods
2. Checking if the file exists
3. If exists: reading existing content, computing incoming content, diffing
4. If not exists: recording a "create" action
5. Returning `*DryRunResult` instead of `*ApplyResult`

This avoids modifying target install functions or introducing circular deps.

**Step 3 (revised): Add preview helpers in engine package**

In `internal/engine/preview.go`:

```go
package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaz8081/positive-vibes/internal/fsutil"
	"github.com/chaz8081/positive-vibes/internal/target"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

// previewSkillInstall computes DryRunOps for installing a skill to a target.
func previewSkillInstall(skill *schema.Skill, sourceDir, projectRoot string, t target.Target) ([]DryRunOp, error) {
	skillDir := t.SkillDir()
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, skillDir), skill.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid skill name %q: %w", skill.Name, err)
	}

	var ops []DryRunOp

	// Check SKILL.md
	skillFile := filepath.Join(dest, "SKILL.md")
	relSkillFile, _ := filepath.Rel(projectRoot, skillFile)

	incoming, err := schema.RenderSkillFile(skill)
	if err != nil {
		return nil, err
	}

	existingData, readErr := os.ReadFile(skillFile)
	if readErr != nil {
		// File doesn't exist -> create
		ops = append(ops, DryRunOp{
			Action:   DryRunCreate,
			RelPath:  relSkillFile,
			Target:   t.Name(),
			Kind:     KindSkill,
			Resource: skill.Name,
		})
	} else {
		// File exists -> update
		diff := unifiedDiff(relSkillFile+" (current)", relSkillFile+" (incoming)", string(existingData), string(incoming))
		ops = append(ops, DryRunOp{
			Action:   DryRunUpdate,
			RelPath:  relSkillFile,
			Target:   t.Name(),
			Kind:     KindSkill,
			Resource: skill.Name,
			Diff:     diff,
		})
	}

	// Check additional files from sourceDir
	if sourceDir != "" {
		filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(sourceDir, path)
			if rel == "SKILL.md" {
				return nil
			}
			targetPath := filepath.Join(dest, rel)
			relTargetPath, _ := filepath.Rel(projectRoot, targetPath)
			srcData, _ := os.ReadFile(path)

			if existingData, err := os.ReadFile(targetPath); err == nil {
				diff := unifiedDiff(relTargetPath+" (current)", relTargetPath+" (incoming)", string(existingData), string(srcData))
				ops = append(ops, DryRunOp{
					Action:   DryRunUpdate,
					RelPath:  relTargetPath,
					Target:   t.Name(),
					Kind:     KindSkill,
					Resource: skill.Name,
					Diff:     diff,
				})
			} else {
				ops = append(ops, DryRunOp{
					Action:   DryRunCreate,
					RelPath:  relTargetPath,
					Target:   t.Name(),
					Kind:     KindSkill,
					Resource: skill.Name,
				})
			}
			return nil
		})
	}

	return ops, nil
}

// previewSingleFileInstall computes a DryRunOp for a single-file resource (instruction, agent, prompt).
func previewSingleFileInstall(name, content, sourcePath, projectRoot, resDir, suffix string, t target.Target, kind ApplyOpKind) (DryRunOp, error) {
	dest, err := fsutil.ResolveWithinRoot(filepath.Join(projectRoot, resDir), name+suffix)
	if err != nil {
		return DryRunOp{}, fmt.Errorf("invalid name %q: %w", name, err)
	}
	relDest, _ := filepath.Rel(projectRoot, dest)

	// Determine incoming content
	var incomingData []byte
	if content != "" {
		incomingData = []byte(content)
	} else if sourcePath != "" {
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return DryRunOp{}, fmt.Errorf("read source: %w", err)
		}
		incomingData = data
	}

	op := DryRunOp{
		RelPath:  relDest,
		Target:   t.Name(),
		Kind:     kind,
		Resource: name,
	}

	existingData, readErr := os.ReadFile(dest)
	if readErr != nil {
		op.Action = DryRunCreate
	} else {
		op.Action = DryRunUpdate
		op.Diff = unifiedDiff(relDest+" (current)", relDest+" (incoming)", string(existingData), string(incomingData))
	}

	return op, nil
}
```

**Step 4: Write tests for preview helpers**

In `internal/engine/preview_test.go`:

```go
package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/target"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

func TestPreviewSkillInstall_NewSkill(t *testing.T) {
	projectRoot := t.TempDir()
	skill := &schema.Skill{Name: "test-skill", Description: "test", Instructions: "# Test"}
	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Test"), 0o644)

	tgt := target.OpenCodeTarget{}
	ops, err := previewSkillInstall(skill, sourceDir, projectRoot, tgt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("expected at least one op")
	}
	if ops[0].Action != DryRunCreate {
		t.Fatalf("expected create, got %s", ops[0].Action)
	}
	if ops[0].Target != "opencode" {
		t.Fatalf("expected target opencode, got %s", ops[0].Target)
	}

	// Verify no files created
	if _, err := os.Stat(filepath.Join(projectRoot, ".opencode", "skills", "test-skill")); !os.IsNotExist(err) {
		t.Fatal("preview should not create files")
	}
}

func TestPreviewSkillInstall_ExistingSkill(t *testing.T) {
	projectRoot := t.TempDir()
	dest := filepath.Join(projectRoot, ".opencode", "skills", "test-skill")
	os.MkdirAll(dest, 0o755)
	os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("# Old content"), 0o644)

	skill := &schema.Skill{Name: "test-skill", Description: "updated", Instructions: "# New content"}
	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# New content"), 0o644)

	tgt := target.OpenCodeTarget{}
	ops, err := previewSkillInstall(skill, sourceDir, projectRoot, tgt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ops[0].Action != DryRunUpdate {
		t.Fatalf("expected update, got %s", ops[0].Action)
	}

	// Content should not have changed
	data, _ := os.ReadFile(filepath.Join(dest, "SKILL.md"))
	if string(data) != "# Old content" {
		t.Fatal("preview should not modify files")
	}
}

func TestPreviewSingleFileInstall_New(t *testing.T) {
	projectRoot := t.TempDir()
	tgt := target.OpenCodeTarget{}
	op, err := previewSingleFileInstall("style", "Use tabs.", "", projectRoot, tgt.InstructionDir(), ".md", tgt, KindInstruction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Action != DryRunCreate {
		t.Fatalf("expected create, got %s", op.Action)
	}
}

func TestPreviewSingleFileInstall_Update(t *testing.T) {
	projectRoot := t.TempDir()
	instDir := filepath.Join(projectRoot, ".opencode", "instructions")
	os.MkdirAll(instDir, 0o755)
	os.WriteFile(filepath.Join(instDir, "style.md"), []byte("Use spaces."), 0o644)

	tgt := target.OpenCodeTarget{}
	op, err := previewSingleFileInstall("style", "Use tabs.", "", projectRoot, tgt.InstructionDir(), ".md", tgt, KindInstruction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Action != DryRunUpdate {
		t.Fatalf("expected update, got %s", op.Action)
	}
	if op.Diff == "" {
		t.Fatal("expected diff for update")
	}
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run TestPreview -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/engine/preview.go internal/engine/preview_test.go
git commit -m "feat(engine): add preview helpers for dry-run skill and resource installs"
```

---

### Task 4: Wire dry-run through ApplyManifest

Modify `ApplyManifest()` to return `*DryRunResult` when `opts.DryRun` is true, using the preview helpers. Add a new method `ApplyManifestDryRun()` or a second return value. Cleanest approach: add `DryRunOps []DryRunOp` to the existing `ApplyResult` struct and populate it when dry-run.

**Files:**
- Modify: `internal/engine/applier.go:44-50` (add `DryRunOps` to `ApplyResult`)
- Modify: `internal/engine/applier.go:72-396` (add dry-run branches in `ApplyManifest`)
- Create: `internal/engine/applier_dryrun_test.go`

**Step 1: Write the failing test**

In `internal/engine/applier_dryrun_test.go`:

```go
package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/internal/target"
)

func TestApplyManifest_DryRun_NoFilesCreated(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: conventional-commits
instructions:
- name: test-inst
  content: "Test instruction"
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	opts := target.InstallOpts{DryRun: true}
	res, err := a.Apply(mfile, opts)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	// Should have dry-run ops
	if len(res.DryRunOps) == 0 {
		t.Fatal("expected dry-run ops, got none")
	}

	// Should NOT have any installed count
	if res.Installed != 0 {
		t.Fatalf("expected 0 installed in dry-run, got %d", res.Installed)
	}

	// Nothing should exist on disk
	if _, err := os.Stat(filepath.Join(tmp, ".opencode", "skills")); !os.IsNotExist(err) {
		t.Fatal("dry-run should not create skill directory")
	}
	if _, err := os.Stat(filepath.Join(tmp, ".opencode", "instructions")); !os.IsNotExist(err) {
		t.Fatal("dry-run should not create instruction directory")
	}
}

func TestApplyManifest_DryRun_DetectsUpdates(t *testing.T) {
	tmp := t.TempDir()

	// Pre-install a skill
	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)

	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: conventional-commits
`
	os.WriteFile(mfile, []byte(content), 0o644)

	// First, actually install
	_, err := a.Apply(mfile, target.InstallOpts{Force: true})
	if err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	// Now dry-run should detect update
	res, err := a.Apply(mfile, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run apply: %v", err)
	}
	if len(res.DryRunOps) == 0 {
		t.Fatal("expected dry-run ops")
	}

	// At least one op should be "update"
	var hasUpdate bool
	for _, op := range res.DryRunOps {
		if op.Action == DryRunUpdate {
			hasUpdate = true
			break
		}
	}
	if !hasUpdate {
		t.Fatalf("expected at least one update op, got: %+v", res.DryRunOps)
	}
}

func TestApplyManifest_DryRun_PromptsSkipCursor(t *testing.T) {
	tmp := t.TempDir()

	promptSrc := filepath.Join(tmp, "prompts", "release.prompt.md")
	os.MkdirAll(filepath.Dir(promptSrc), 0o755)
	os.WriteFile(promptSrc, []byte("---\ndescription: Release\n---\nRelease"), 0o644)

	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode","cursor"]
prompts:
- name: release
  path: ./prompts/release.prompt.md
`
	os.WriteFile(mfile, []byte(content), 0o644)

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	res, err := a.Apply(mfile, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	// Should have a skip for cursor prompts
	var hasSkip bool
	for _, op := range res.DryRunOps {
		if op.Target == "cursor" && op.Kind == KindPrompt && op.Action == DryRunSkip {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Fatalf("expected cursor prompt skip, got: %+v", res.DryRunOps)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestApplyManifest_DryRun -v`
Expected: FAIL (`DryRunOps` field doesn't exist)

**Step 3: Add `DryRunOps` to `ApplyResult` and wire dry-run into `ApplyManifest()`**

In `internal/engine/applier.go`:

1. Add `DryRunOps []DryRunOp` to the `ApplyResult` struct (after line 49)
2. In `ApplyManifest()`, for each resource loop (skills, instructions, agents, prompts), when `opts.DryRun` is true, call the preview helpers instead of the install functions

The key changes in `ApplyManifest`:
- After resolving each skill/resource source, if dry-run: call `previewSkillInstall()` or `previewSingleFileInstall()` and append to `res.DryRunOps`, then `continue` (skip actual install)
- For prompts on cursor: add a skip op
- Source resolution (registry fetches, local reads) still happens so errors surface

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run TestApplyManifest_DryRun -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS (existing tests unaffected since DryRun defaults to false)

**Step 6: Commit**

```bash
git add internal/engine/applier.go internal/engine/applier_dryrun_test.go
git commit -m "feat(engine): wire dry-run through ApplyManifest for all resource types"
```

---

### Task 5: CLI flags and output

Add `--dry-run`/`-n` and `--verbose`/`-v` flags to the apply command. When dry-run, render the preview output instead of the normal install output.

**Files:**
- Modify: `internal/cli/apply.go` (add flags, render dry-run output)

**Step 1: Write the failing test** (manual testing -- CLI integration)

No unit test for this -- the engine-level tests cover correctness. This is a CLI wiring task.

**Step 2: Add flags and output rendering**

In `internal/cli/apply.go`:

1. Add variables: `applyDryRun bool`, `applyVerbose bool`
2. Register flags in `init()`:
   - `applyCmd.Flags().BoolVarP(&applyDryRun, "dry-run", "n", false, "preview changes without writing files")`
   - `applyCmd.Flags().BoolVarP(&applyVerbose, "verbose", "v", false, "show content diffs for existing files (requires --dry-run)")`
3. In the `Run` function, set `opts.DryRun = applyDryRun`
4. After `applier.ApplyManifest()`, if dry-run: render dry-run output and return early
5. If `--verbose` without `--dry-run`, print a warning that `--verbose` only works with `--dry-run`

Dry-run rendering:
```go
if applyDryRun {
    if len(res.DryRunOps) == 0 && len(res.Errors) == 0 {
        fmt.Println("Nothing to apply.")
        return
    }
    for _, op := range res.DryRunOps {
        fmt.Println(op.ColoredString())
        if applyVerbose && op.Diff != "" {
            fmt.Print(engine.ColorDiff(op.Diff))
        }
    }
    for _, e := range res.Errors {
        fmt.Printf("  error: %s\n", e)
    }
    fmt.Println()
    fmt.Println(engine.FormatDryRunSummary(res.DryRunOps))
    return
}
```

**Step 3: Run build and manual test**

Run: `go build ./cmd/positive-vibes && ./positive-vibes apply --dry-run --help`
Expected: Help text shows --dry-run and --verbose flags

**Step 4: Commit**

```bash
git add internal/cli/apply.go
git commit -m "feat(cli): add --dry-run and --verbose flags to apply command"
```

---

### Task 6: Final verification and edge cases

**Files:**
- Modify: `internal/engine/applier_dryrun_test.go` (add edge case tests)

**Step 1: Add edge-case tests**

- Dry-run with `--link` flag (should still show create/update, note symlink mode)
- Dry-run with missing source file (should surface error, not panic)
- Dry-run with empty manifest (should return empty result)

**Step 2: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

**Step 3: Run go vet and race detector**

Run: `go vet ./... && go test -race ./internal/...`
Expected: Clean

**Step 4: Final commit**

```bash
git add .
git commit -m "test(engine): add dry-run edge case tests"
```
