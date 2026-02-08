package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

// ManifestFilenames lists the manifest filenames in priority order.
// vibes.yml is preferred; vibes.yaml is the legacy fallback.
var ManifestFilenames = []string{"vibes.yml", "vibes.yaml"}

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
	Ref   string            `yaml:"ref"`
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
	for _, r := range m.Registries {
		if r.Ref == "" {
			return fmt.Errorf("registry %q must specify a ref (use \"latest\" to track the default branch)", r.Name)
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

// LoadManifestFromProject searches a project directory for vibes.yml (preferred)
// or vibes.yaml (fallback). Returns the parsed manifest and the path that was loaded.
func LoadManifestFromProject(projectDir string) (*Manifest, string, error) {
	for _, name := range ManifestFilenames {
		p := filepath.Join(projectDir, name)
		if _, err := os.Stat(p); err == nil {
			m, err := LoadManifest(p)
			if err != nil {
				return nil, "", err
			}
			return m, p, nil
		}
	}
	return nil, "", fmt.Errorf("no manifest found in %s (looked for %v)", projectDir, ManifestFilenames)
}

// SaveManifestWithComments writes a manifest to the given path, prepending
// a comment header string. The header should already contain '#' prefixed lines.
// An empty header is allowed (equivalent to SaveManifest).
func SaveManifestWithComments(m *Manifest, path string, header string) error {
	yamlData, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	content := header + string(yamlData)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// LoadMergedManifest loads a global manifest (from globalPath) and a project
// manifest (from projectDir), merging them with project values taking priority.
//
// Merge rules:
//   - Registries: merged by Name; project overrides global for same name
//   - Skills: merged by Name; project overrides global for same name
//   - Targets: project targets override global (no merge)
//   - Instructions: concatenated (global first, then project)
//
// Returns error only if neither global nor project manifest exists.
func LoadMergedManifest(projectDir string, globalPath string) (*Manifest, error) {
	var global, project *Manifest

	// Load global manifest (optional)
	if data, err := os.ReadFile(globalPath); err == nil {
		g, err := LoadManifestFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("parse global manifest: %w", err)
		}
		global = g
	}

	// Load project manifest (optional)
	if p, _, err := LoadManifestFromProject(projectDir); err == nil {
		project = p
	}

	if global == nil && project == nil {
		return nil, fmt.Errorf("no manifest found: checked %s and %s", globalPath, projectDir)
	}

	// If only one exists, return it directly
	if global == nil {
		return project, nil
	}
	if project == nil {
		return global, nil
	}

	// Merge: start from global, overlay project
	merged := &Manifest{}

	// Registries: merge by Name, project wins
	regMap := make(map[string]RegistryRef)
	var regOrder []string
	for _, r := range global.Registries {
		regMap[r.Name] = r
		regOrder = append(regOrder, r.Name)
	}
	for _, r := range project.Registries {
		if _, exists := regMap[r.Name]; !exists {
			regOrder = append(regOrder, r.Name)
		}
		regMap[r.Name] = r // project overrides
	}
	for _, name := range regOrder {
		merged.Registries = append(merged.Registries, regMap[name])
	}

	// Skills: merge by Name, project wins
	skillMap := make(map[string]SkillRef)
	var skillOrder []string
	for _, s := range global.Skills {
		skillMap[s.Name] = s
		skillOrder = append(skillOrder, s.Name)
	}
	for _, s := range project.Skills {
		if _, exists := skillMap[s.Name]; !exists {
			skillOrder = append(skillOrder, s.Name)
		}
		skillMap[s.Name] = s // project overrides
	}
	for _, name := range skillOrder {
		merged.Skills = append(merged.Skills, skillMap[name])
	}

	// Targets: project overrides entirely
	if len(project.Targets) > 0 {
		merged.Targets = project.Targets
	} else {
		merged.Targets = global.Targets
	}

	// Instructions: concatenate (global first, then project)
	merged.Instructions = append(merged.Instructions, global.Instructions...)
	merged.Instructions = append(merged.Instructions, project.Instructions...)
	if len(merged.Instructions) == 0 {
		merged.Instructions = nil
	}

	return merged, nil
}
