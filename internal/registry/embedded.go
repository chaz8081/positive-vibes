package registry

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
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
func (e *EmbeddedRegistry) Fetch(name string) (*schema.Skill, string, error) {
	rel := filepath.Join(name, "SKILL.md")
	b, err := e.FS.ReadFile(rel)
	if err != nil {
		return nil, "", fmt.Errorf("skill %s not found: %w", name, err)
	}

	sk, err := schema.ParseSkillFile(b)
	if err != nil {
		return nil, "", err
	}

	// write to temp dir
	tmp, err := ioutil.TempDir("", "pv-skill-")
	if err != nil {
		return nil, "", err
	}

	// create dir for skill
	skillDir := filepath.Join(tmp, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return nil, "", err
	}

	if err := ioutil.WriteFile(filepath.Join(skillDir, "SKILL.md"), b, 0o644); err != nil {
		return nil, "", err
	}

	return sk, skillDir, nil
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
