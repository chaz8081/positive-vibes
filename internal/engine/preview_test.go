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

	// Verify no files created on disk
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

func TestPreviewSkillInstall_AdditionalFiles(t *testing.T) {
	projectRoot := t.TempDir()
	skill := &schema.Skill{Name: "multi-file", Description: "test", Instructions: "# Test"}
	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Test"), 0o644)
	os.MkdirAll(filepath.Join(sourceDir, "reference"), 0o755)
	os.WriteFile(filepath.Join(sourceDir, "reference", "examples.md"), []byte("# Examples"), 0o644)

	tgt := target.OpenCodeTarget{}
	ops, err := previewSkillInstall(skill, sourceDir, projectRoot, tgt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have at least 2 ops: SKILL.md + reference/examples.md
	if len(ops) < 2 {
		t.Fatalf("expected at least 2 ops, got %d: %+v", len(ops), ops)
	}
}
