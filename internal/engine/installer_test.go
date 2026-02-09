package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
)

func TestInstaller(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: conventional-commits
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	inst := NewInstaller([]registry.SkillSource{registry.NewEmbeddedRegistry()})
	if err := inst.Install("code-review", mfile); err != nil {
		t.Fatalf("install error: %v", err)
	}

	// try again -> error
	if err := inst.Install("code-review", mfile); err == nil {
		t.Fatalf("expected error when installing duplicate")
	}

	// nonexistent
	if err := inst.Install("no-such-skill-xyz", mfile); err == nil {
		t.Fatalf("expected error for nonexistent skill")
	}
}

func TestInstaller_LocalSkill(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: conventional-commits
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Create a local skill at skills/my-custom-skill/SKILL.md
	skillDir := filepath.Join(tmp, "skills", "my-custom-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillContent := `---
name: my-custom-skill
description: A custom local skill
version: "1.0"
author: test-user
tags: [custom]
---
# My Custom Skill

Do the thing.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	// Install should find the local skill even without registry match
	inst := NewInstaller([]registry.SkillSource{registry.NewEmbeddedRegistry()})
	if err := inst.Install("my-custom-skill", mfile); err != nil {
		t.Fatalf("install local skill error: %v", err)
	}

	// Verify manifest was updated with path
	m, err := manifest.LoadManifest(mfile)
	if err != nil {
		t.Fatalf("reload manifest: %v", err)
	}

	var found *manifest.SkillRef
	for i, s := range m.Skills {
		if s.Name == "my-custom-skill" {
			found = &m.Skills[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("skill 'my-custom-skill' not found in manifest after install")
	}
	if found.Path != "./skills/my-custom-skill" {
		t.Fatalf("expected path './skills/my-custom-skill', got %q", found.Path)
	}
}

func TestInstaller_LocalSkillDuplicate(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: my-local-skill
  path: ./skills/my-local-skill
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Create local skill dir
	skillDir := filepath.Join(tmp, "skills", "my-local-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-local-skill\n---\n# Skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	inst := NewInstaller(nil)
	err := inst.Install("my-local-skill", mfile)
	if err == nil {
		t.Fatalf("expected error installing duplicate local skill")
	}
	if !strings.Contains(err.Error(), "already in manifest") {
		t.Fatalf("expected 'already in manifest' error, got: %v", err)
	}
}

func TestInstaller_LocalSkillPriority(t *testing.T) {
	// If a skill exists both locally AND in a registry, local should win
	// (path should be set in manifest)
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills: []
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// "code-review" exists in the embedded registry, but also create it locally
	skillDir := filepath.Join(tmp, "skills", "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: code-review\n---\n# Local code review\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	inst := NewInstaller([]registry.SkillSource{registry.NewEmbeddedRegistry()})
	if err := inst.Install("code-review", mfile); err != nil {
		t.Fatalf("install error: %v", err)
	}

	// Should have path set (local takes priority)
	m, err := manifest.LoadManifest(mfile)
	if err != nil {
		t.Fatalf("reload manifest: %v", err)
	}
	for _, s := range m.Skills {
		if s.Name == "code-review" {
			if s.Path != "./skills/code-review" {
				t.Fatalf("local skill should take priority; expected path './skills/code-review', got %q", s.Path)
			}
			return
		}
	}
	t.Fatalf("skill 'code-review' not found in manifest")
}

func TestInstaller_NoManifest_CreatesNew(t *testing.T) {
	// When no manifest file exists, Install should create one with the skill added.
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	// mfile does NOT exist on disk

	inst := NewInstaller([]registry.SkillSource{registry.NewEmbeddedRegistry()})
	if err := inst.Install("code-review", mfile); err != nil {
		t.Fatalf("install into missing manifest should succeed, got: %v", err)
	}

	// Verify manifest was created with the skill
	m, err := manifest.LoadManifest(mfile)
	if err != nil {
		t.Fatalf("should be able to load created manifest: %v", err)
	}
	found := false
	for _, s := range m.Skills {
		if s.Name == "code-review" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'code-review' in created manifest skills")
	}
}

func TestInstaller_NoManifest_SupportsYml(t *testing.T) {
	// When the caller passes a .yml path and it doesn't exist, Install should
	// still create it (both extensions supported).
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yml")

	inst := NewInstaller([]registry.SkillSource{registry.NewEmbeddedRegistry()})
	if err := inst.Install("code-review", mfile); err != nil {
		t.Fatalf("install into missing .yml manifest should succeed, got: %v", err)
	}

	m, err := manifest.LoadManifest(mfile)
	if err != nil {
		t.Fatalf("should be able to load created manifest: %v", err)
	}
	found := false
	for _, s := range m.Skills {
		if s.Name == "code-review" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'code-review' in created manifest skills")
	}
}

func TestInstaller_InvalidLocalSkill(t *testing.T) {
	// A local skill dir exists but SKILL.md is missing -> should fall through to registry
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills: []
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Create dir but no SKILL.md
	skillDir := filepath.Join(tmp, "skills", "ghost-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	inst := NewInstaller([]registry.SkillSource{registry.NewEmbeddedRegistry()})
	err := inst.Install("ghost-skill", mfile)
	// Should fail because not in registry either
	if err == nil {
		t.Fatalf("expected error for skill without SKILL.md and not in registry")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

// --- Remove tests ---

func TestRemove_RemovesExistingSkill(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: code-review
- name: conventional-commits
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	inst := NewInstaller(nil)
	if err := inst.Remove("code-review", mfile); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	// Verify manifest updated
	m, err := manifest.LoadManifest(mfile)
	if err != nil {
		t.Fatalf("reload manifest: %v", err)
	}
	if len(m.Skills) != 1 {
		t.Fatalf("expected 1 skill remaining, got %d", len(m.Skills))
	}
	if m.Skills[0].Name != "conventional-commits" {
		t.Fatalf("expected remaining skill to be 'conventional-commits', got %q", m.Skills[0].Name)
	}
}

func TestRemove_SkillNotInManifest(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: code-review
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	inst := NewInstaller(nil)
	err := inst.Remove("nonexistent-skill", mfile)
	if err == nil {
		t.Fatalf("expected error removing nonexistent skill")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestRemove_NoManifest(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	// File does NOT exist

	inst := NewInstaller(nil)
	err := inst.Remove("code-review", mfile)
	if err == nil {
		t.Fatalf("expected error when manifest doesn't exist")
	}
}

func TestRemove_LastSkill(t *testing.T) {
	// Removing the last skill should work (manifest may become invalid, but that's user's choice)
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: code-review
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	inst := NewInstaller(nil)
	if err := inst.Remove("code-review", mfile); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	m, err := manifest.LoadManifest(mfile)
	if err != nil {
		t.Fatalf("reload manifest: %v", err)
	}
	if len(m.Skills) != 0 {
		t.Fatalf("expected 0 skills remaining, got %d", len(m.Skills))
	}
}
