package engine

import (
	"os"
	"path/filepath"
	"testing"

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
