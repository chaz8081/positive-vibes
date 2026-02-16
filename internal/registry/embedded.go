package registry

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/chaz8081/positive-vibes/pkg/schema"
	skills "github.com/chaz8081/positive-vibes/skills"
)

// EmbeddedRegistry serves skills embedded into the binary.
type EmbeddedRegistry struct {
	RegistryName string
	FS           embed.FS
}

// NewEmbeddedRegistry constructs an EmbeddedRegistry using the embedded skills FS.
func NewEmbeddedRegistry() *EmbeddedRegistry {
	return &EmbeddedRegistry{
		RegistryName: "embedded",
		FS:           skills.SkillsFS,
	}
}

func (e *EmbeddedRegistry) Name() string { return e.RegistryName }

// Fetch reads the SKILL.md from the embedded FS, parses it and writes it to a temp dir.
// DEPRECATED: Use FetchWithCleanup instead to ensure temp directories are cleaned up.
func (e *EmbeddedRegistry) Fetch(name string) (*schema.Skill, string, error) {
	sk, srcDir, _, err := e.FetchWithCleanup(name)
	return sk, srcDir, err
}

// FetchWithCleanup reads the SKILL.md from the embedded FS, parses it, writes
// it to a temp dir, and returns a cleanup function that removes the temp dir.
// Callers MUST call the cleanup function when they are done with srcDir.
func (e *EmbeddedRegistry) FetchWithCleanup(name string) (*schema.Skill, string, func(), error) {
	noop := func() {}

	rel := filepath.Join(name, "SKILL.md")
	b, err := e.FS.ReadFile(rel)
	if err != nil {
		return nil, "", noop, fmt.Errorf("skill %s not found: %w", name, err)
	}

	sk, err := schema.ParseSkillFile(b)
	if err != nil {
		return nil, "", noop, err
	}

	// write to temp dir
	tmp, err := os.MkdirTemp("", "pv-skill-")
	if err != nil {
		return nil, "", noop, err
	}

	// create dir for skill
	skillDir := filepath.Join(tmp, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		os.RemoveAll(tmp)
		return nil, "", noop, err
	}

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), b, 0o644); err != nil {
		os.RemoveAll(tmp)
		return nil, "", noop, err
	}

	cleanup := func() {
		os.RemoveAll(tmp)
	}

	return sk, skillDir, cleanup, nil
}

// List returns all skill names embedded.
func (e *EmbeddedRegistry) List() ([]string, error) {
	var names []string
	// Walk the embedded FS root
	entries, err := fs.ReadDir(e.FS, ".")
	if err != nil {
		return nil, err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			// ensure SKILL.md exists
			p := filepath.Join(ent.Name(), "SKILL.md")
			if _, err := e.FS.ReadFile(p); err == nil {
				names = append(names, ent.Name())
			}
		}
	}
	sort.Strings(names)
	return names, nil
}
