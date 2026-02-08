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

func TestApplierApply_OpsTracking(t *testing.T) {
	tmp := t.TempDir()

	// Create a local skill
	skillDir := filepath.Join(tmp, "skills", "local-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: local-skill\n---\n# Local\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Manifest: 1 registry skill, 1 local skill, 1 nonexistent skill, 2 targets
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode","cursor"]
skills:
- name: conventional-commits
- name: local-skill
  path: ./skills/local-skill
- name: does-not-exist
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

	// Should have per-operation records
	if len(res.Ops) == 0 {
		t.Fatalf("expected Ops to be populated, got empty slice")
	}

	// 2 skills * 2 targets = 4 installs + 1 not-found = 5 ops total
	if len(res.Ops) != 5 {
		t.Fatalf("expected 5 ops, got %d: %+v", len(res.Ops), res.Ops)
	}

	// Count by status
	counts := map[ApplyOpStatus]int{}
	for _, op := range res.Ops {
		counts[op.Status]++
	}
	if counts[OpInstalled] != 4 {
		t.Fatalf("expected 4 installed ops, got %d", counts[OpInstalled])
	}
	if counts[OpNotFound] != 1 {
		t.Fatalf("expected 1 not_found op, got %d", counts[OpNotFound])
	}

	// Verify specific op fields
	for _, op := range res.Ops {
		if op.SkillName == "" {
			t.Fatalf("op has empty SkillName: %+v", op)
		}
		if op.Status == OpInstalled && op.TargetName == "" {
			t.Fatalf("installed op has empty TargetName: %+v", op)
		}
		if op.Status == OpNotFound && op.SkillName != "does-not-exist" {
			t.Fatalf("not_found op for wrong skill: %+v", op)
		}
	}
}

func TestApplierApply_OpsSkipped(t *testing.T) {
	tmp := t.TempDir()
	mfile := filepath.Join(tmp, "vibes.yaml")
	content := `targets: ["opencode"]
skills:
- name: conventional-commits
`
	if err := os.WriteFile(mfile, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	a := NewApplier(regs)

	// First apply with force to install
	opts := target.InstallOpts{Force: true}
	_, err := a.Apply(mfile, opts)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}

	// Second apply WITHOUT force -- should skip
	opts = target.InstallOpts{Force: false}
	res, err := a.Apply(mfile, opts)
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}

	if res.Skipped != 1 {
		t.Fatalf("expected 1 skipped, got %d", res.Skipped)
	}

	// Verify ops contain a skipped record
	skippedCount := 0
	for _, op := range res.Ops {
		if op.Status == OpSkipped {
			skippedCount++
		}
	}
	if skippedCount != 1 {
		t.Fatalf("expected 1 skipped op, got %d", skippedCount)
	}
}
