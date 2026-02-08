package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/internal/target"
)

func TestApplierApply(t *testing.T) {
	tmp := t.TempDir()
	// create a simple manifest
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["vscode-copilot","opencode","cursor"]
skills:
- name: conventional-commits
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)
	opts := target.InstallOpts{Force: true}
	res, err := a.Apply(mfile, opts)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if res.Installed == 0 {
		t.Fatalf("expected installed > 0, got 0, errors: %v", res.Errors)
	}
}

func TestApplierApply_LocalPathSkill(t *testing.T) {
	tmp := t.TempDir()

	// Create a local skill at skills/my-local/SKILL.md
	skillDir := filepath.Join(tmp, "skills", "my-local")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillContent := `---
name: my-local
description: A locally generated skill
version: "1.0"
author: test
tags: [local]
---
# My Local Skill

Instructions go here.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	// Manifest references the local skill with a relative path
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: my-local
  path: ./skills/my-local
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// No registries needed -- the skill is local
	a := NewApplier(nil)
	opts := target.InstallOpts{Force: true}
	res, err := a.Apply(mfile, opts)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.Installed != 1 {
		t.Fatalf("expected 1 installed, got %d", res.Installed)
	}

	// Verify the skill was actually written to the target directory
	installed := filepath.Join(tmp, ".opencode", "skills", "my-local", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("expected skill file at %s, got error: %v", installed, err)
	}
}
