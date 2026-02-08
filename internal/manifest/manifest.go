package manifest

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// ValidTargets are the supported target tool identifiers.
var ValidTargets = []string{"vscode-copilot", "opencode", "cursor"}

// Manifest represents a vibes.yaml file.
type Manifest struct {
	Registries   []RegistryRef `yaml:"registries,omitempty"`
	Skills       []SkillRef    `yaml:"skills"`
	Instructions []string      `yaml:"instructions,omitempty"`
	Targets      []string      `yaml:"targets"`
}

// SkillRef is a reference to a skill in the manifest.
type SkillRef struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path,omitempty"`
	Version string `yaml:"version,omitempty"`
}

// RegistryRef points to a remote git repository of skills.
type RegistryRef struct {
	Name  string            `yaml:"name"`
	URL   string            `yaml:"url"`
	Paths map[string]string `yaml:"paths,omitempty"` // e.g. {"skills": "skills/", "prompts": "prompts/"}
}

// SkillsPath returns the configured path for skills in this registry,
// defaulting to "." (repo root) if not set.
func (r RegistryRef) SkillsPath() string {
	if p, ok := r.Paths["skills"]; ok && p != "" {
		return p
	}
	return "."
}

// LoadManifest reads and parses a vibes.yaml file from the given path.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return LoadManifestFromBytes(data)
}

// LoadManifestFromBytes parses vibes.yaml content from bytes.
func LoadManifestFromBytes(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// SaveManifest writes the manifest to the given path as YAML.
func SaveManifest(m *Manifest, path string) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// Validate checks the manifest for correctness.
// Returns error if: no skills defined, or invalid target name.
func (m *Manifest) Validate() error {
	if len(m.Skills) == 0 {
		return fmt.Errorf("manifest must define at least one skill")
	}
	if len(m.Targets) == 0 {
		return fmt.Errorf("manifest must define at least one target")
	}
	for _, t := range m.Targets {
		if !isValidTarget(t) {
			return fmt.Errorf("invalid target: %s", t)
		}
	}
	return nil
}

func isValidTarget(t string) bool {
	for _, v := range ValidTargets {
		if t == v {
			return true
		}
	}
	return false
}
