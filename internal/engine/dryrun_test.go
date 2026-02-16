package engine

import (
	"strings"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/term"
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

func TestColorDiff_HeaderLinesAreNotTreatedAsAddsOrDeletes(t *testing.T) {
	t.Parallel()

	diff := strings.Join([]string{
		"--- a/file.txt",
		"+++ b/file.txt",
		"@@ -1,1 +1,1 @@",
		"-old",
		"+new",
		" context",
		"",
	}, "\n")

	colored := ColorDiff(diff)

	if strings.Contains(colored, term.Red()+"--- a/file.txt") {
		t.Fatalf("expected header line not to be colored as deletion")
	}
	if strings.Contains(colored, term.Green()+"+++ b/file.txt") {
		t.Fatalf("expected header line not to be colored as addition")
	}
}
