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
