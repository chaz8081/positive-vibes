package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/internal/target"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

func TestApplyManifest_DryRun_NoFilesCreated(t *testing.T) {
	tmp := t.TempDir()

	// Create local source files for agent and prompt
	agentSrc := filepath.Join(tmp, "sources", "my-agent.md")
	promptSrc := filepath.Join(tmp, "sources", "my-prompt.md")
	if err := os.MkdirAll(filepath.Dir(agentSrc), 0o755); err != nil {
		t.Fatalf("mkdir sources: %v", err)
	}
	if err := os.WriteFile(agentSrc, []byte("# Agent\nDo things."), 0o644); err != nil {
		t.Fatalf("write agent: %v", err)
	}
	if err := os.WriteFile(promptSrc, []byte("---\ndescription: Prompt\n---\nDo stuff."), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	m := &manifest.Manifest{
		Targets: []string{"opencode", "cursor", "vscode-copilot"},
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}},
		Instructions: []manifest.InstructionRef{{
			Name:    "test-inst",
			Content: "Some inline instruction content.",
		}},
		Agents: []manifest.AgentRef{{
			Name: "my-agent",
			Path: "./sources/my-agent.md",
		}},
		Prompts: []manifest.PromptRef{{
			Name: "my-prompt",
			Path: "./sources/my-prompt.md",
		}},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	res, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run apply error: %v", err)
	}

	// DryRunOps should be populated
	if len(res.DryRunOps) == 0 {
		t.Fatalf("expected DryRunOps to be non-empty, got 0")
	}

	// Installed counter should NOT be incremented in dry-run
	if res.Installed != 0 {
		t.Fatalf("expected Installed == 0 in dry-run, got %d", res.Installed)
	}

	// No files or directories should have been created on disk for any target.
	// Check all target root directories.
	for _, dir := range []string{
		filepath.Join(tmp, ".opencode"),
		filepath.Join(tmp, ".cursor"),
		filepath.Join(tmp, ".github"),
	} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("dry-run created directory %s (should not exist)", dir)
		}
	}
}

func TestApplyManifest_DryRun_DetectsUpdates(t *testing.T) {
	tmp := t.TempDir()

	m := &manifest.Manifest{
		Targets: []string{"opencode"},
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)

	// First: actually install the skill with Force
	_, err := a.ApplyManifest(m, tmp, target.InstallOpts{Force: true})
	if err != nil {
		t.Fatalf("real apply error: %v", err)
	}

	// Verify it was really installed
	skillFile := filepath.Join(tmp, ".opencode", "skills", "conventional-commits", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Fatalf("expected skill to be installed: %v", err)
	}

	// Now dry-run the same manifest again
	res, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run apply error: %v", err)
	}

	if len(res.DryRunOps) == 0 {
		t.Fatalf("expected DryRunOps to be non-empty")
	}

	// At least one op should be an update (since skill already exists)
	hasUpdate := false
	for _, op := range res.DryRunOps {
		if op.Action == DryRunUpdate {
			hasUpdate = true
			break
		}
	}
	if !hasUpdate {
		t.Fatalf("expected at least one DryRunUpdate op, got: %+v", res.DryRunOps)
	}
}

func TestApplyManifest_DryRun_PromptsSkipCursor(t *testing.T) {
	tmp := t.TempDir()

	// Create a local prompt source file
	promptSrc := filepath.Join(tmp, "prompts", "my-prompt.md")
	if err := os.MkdirAll(filepath.Dir(promptSrc), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(promptSrc, []byte("---\ndescription: My Prompt\n---\nDo something"), 0o644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	m := &manifest.Manifest{
		Targets: []string{"opencode", "cursor"},
		Prompts: []manifest.PromptRef{{
			Name: "my-prompt",
			Path: "./prompts/my-prompt.md",
		}},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	res, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run apply error: %v", err)
	}

	// Should have a DryRunSkip op for cursor with KindPrompt
	hasSkipCursor := false
	for _, op := range res.DryRunOps {
		if op.Action == DryRunSkip && op.Kind == KindPrompt && op.Target == "cursor" {
			hasSkipCursor = true
			break
		}
	}
	if !hasSkipCursor {
		t.Fatalf("expected a DryRunSkip op for cursor prompt, got: %+v", res.DryRunOps)
	}

	// Should also have a create/update op for opencode's prompt
	hasOpencode := false
	for _, op := range res.DryRunOps {
		if op.Kind == KindPrompt && op.Target == "opencode" && (op.Action == DryRunCreate || op.Action == DryRunUpdate) {
			hasOpencode = true
			break
		}
	}
	if !hasOpencode {
		t.Fatalf("expected a create/update op for opencode prompt, got: %+v", res.DryRunOps)
	}

	// Installed counter should be 0 in dry-run
	if res.Installed != 0 {
		t.Fatalf("expected Installed == 0 in dry-run, got %d", res.Installed)
	}

	// Verify RelPath for cursor skip op is a proper relative path, not just the prompt name
	for _, op := range res.DryRunOps {
		if op.Action == DryRunSkip && op.Kind == KindPrompt && op.Target == "cursor" {
			expected := filepath.Join(".cursor", "prompts", "my-prompt.md")
			if op.RelPath != expected {
				t.Fatalf("expected skip op RelPath=%q, got %q", expected, op.RelPath)
			}
			break
		}
	}

	// CRITICAL: dry-run must not create any directories as side effect.
	// The old probe-call approach caused os.MkdirAll to create prompt dirs.
	cursorPromptDir := filepath.Join(tmp, ".cursor", "prompts")
	if _, err := os.Stat(cursorPromptDir); !os.IsNotExist(err) {
		t.Fatalf("dry-run created cursor prompts directory: %s (should not exist)", cursorPromptDir)
	}
	opencodeCommandsDir := filepath.Join(tmp, ".opencode", "commands")
	if _, err := os.Stat(opencodeCommandsDir); !os.IsNotExist(err) {
		t.Fatalf("dry-run created opencode commands directory: %s (should not exist)", opencodeCommandsDir)
	}
}

func TestApplyManifest_DryRun_EmptyManifest(t *testing.T) {
	tmp := t.TempDir()

	// A manifest with no resources is a validation error (not a dry-run concern).
	// Verify dry-run surfaces the validation error properly.
	m := &manifest.Manifest{
		Targets: []string{"opencode"},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	_, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true})
	if err == nil {
		t.Fatal("expected validation error for empty manifest")
	}
}

func TestApplyManifest_DryRun_MissingSourceFile(t *testing.T) {
	tmp := t.TempDir()

	m := &manifest.Manifest{
		Targets: []string{"opencode"},
		Instructions: []manifest.InstructionRef{{
			Name: "missing-inst",
			Path: "./nonexistent/file.md",
		}},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	res, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run should not return top-level error for missing source, got: %v", err)
	}

	// Should report an error in res.Errors, not panic
	if len(res.Errors) == 0 {
		t.Fatalf("expected error for missing source file, got none; DryRunOps: %+v", res.DryRunOps)
	}
}

func TestApplyManifest_DryRun_LinkMode(t *testing.T) {
	tmp := t.TempDir()

	m := &manifest.Manifest{
		Targets: []string{"opencode"},
		Skills:  []manifest.SkillRef{{Name: "conventional-commits"}},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)

	// Dry-run with Link mode should still produce preview ops (not actually symlink)
	res, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true, Link: true})
	if err != nil {
		t.Fatalf("dry-run+link apply error: %v", err)
	}

	if len(res.DryRunOps) == 0 {
		t.Fatalf("expected DryRunOps for link mode, got 0")
	}

	// Should all be create (new install)
	for _, op := range res.DryRunOps {
		if op.Action != DryRunCreate {
			t.Fatalf("expected create for new install with link, got %s for %s", op.Action, op.RelPath)
		}
	}

	// No files or symlinks should have been created
	if _, err := os.Stat(filepath.Join(tmp, ".opencode")); !os.IsNotExist(err) {
		t.Fatalf("dry-run+link created directory (should not exist)")
	}
}

func TestApplyManifest_DryRun_InstructionApplyTo(t *testing.T) {
	tmp := t.TempDir()

	m := &manifest.Manifest{
		Targets: []string{"opencode", "vscode-copilot"},
		Instructions: []manifest.InstructionRef{{
			Name:    "opencode-only",
			Content: "This is only for opencode.",
			ApplyTo: "opencode",
		}},
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	res, err := a.ApplyManifest(m, tmp, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run apply error: %v", err)
	}

	// Should only have ops for opencode, not vscode-copilot
	for _, op := range res.DryRunOps {
		if op.Target == "vscode-copilot" {
			t.Fatalf("instruction with apply_to=opencode should not appear for vscode-copilot: %+v", op)
		}
	}

	if len(res.DryRunOps) != 1 {
		t.Fatalf("expected exactly 1 DryRunOp, got %d: %+v", len(res.DryRunOps), res.DryRunOps)
	}
	if res.DryRunOps[0].Target != "opencode" {
		t.Fatalf("expected target opencode, got %s", res.DryRunOps[0].Target)
	}
}

func TestApplyManifest_DryRun_RegistryDoesNotWriteTempFiles(t *testing.T) {
	t.Parallel()

	reg := &fakeResourceRegistry{
		name: "test-reg",
		data: map[string][]byte{
			"instructions/hello.md": []byte("inst content\n"),
			"agents/bot.md":         []byte("agent content\n"),
			"prompts/prompt.md":     []byte("prompt content\n"),
		},
	}

	dir := t.TempDir()
	m := &manifest.Manifest{
		Instructions: []manifest.InstructionRef{{
			Name:     "hello",
			Registry: "test-reg",
			Path:     "hello.md",
		}},
		Agents: []manifest.AgentRef{{
			Name:     "bot",
			Registry: "test-reg",
			Path:     "bot.md",
		}},
		Prompts: []manifest.PromptRef{{
			Name:     "prompt",
			Registry: "test-reg",
			Path:     "prompt.md",
		}},
		Targets: []string{"opencode"},
	}

	a := NewApplier([]registry.SkillSource{reg})
	res, err := a.ApplyManifest(m, dir, target.InstallOpts{DryRun: true})
	if err != nil {
		t.Fatalf("apply manifest: %v", err)
	}

	if len(res.Errors) > 0 {
		t.Fatalf("expected no errors, got: %v", res.Errors)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read project dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files in project dir, got %d entries", len(entries))
	}
}

type fakeResourceRegistry struct {
	name string
	data map[string][]byte
}

func (f *fakeResourceRegistry) Name() string { return f.name }
func (f *fakeResourceRegistry) Fetch(name string) (*schema.Skill, string, error) {
	return nil, "", fmt.Errorf("skill not found: %s", name)
}
func (f *fakeResourceRegistry) List() ([]string, error) {
	return nil, nil
}
func (f *fakeResourceRegistry) FetchResourceFile(category, name string) ([]byte, error) {
	key := filepath.Join(category, name)
	if data, ok := f.data[key]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("resource not found: %s/%s", category, name)
}
func (f *fakeResourceRegistry) ListResourceFiles(kind string) ([]string, error) {
	return nil, nil
}
