package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// ManifestFilenames lists the manifest filenames in priority order.
// vibes.yaml is preferred; vibes.yml is the legacy fallback.
var ManifestFilenames = []string{"vibes.yaml", "vibes.yml"}

// ValidTargets are the supported target tool identifiers.
var ValidTargets = []string{"vscode-copilot", "opencode", "cursor"}

// Manifest represents a vibes.yaml file.
type Manifest struct {
	Registries   []RegistryRef    `yaml:"registries,omitempty"`
	Skills       []SkillRef       `yaml:"skills"`
	Instructions []InstructionRef `yaml:"instructions,omitempty"`
	Agents       []AgentRef       `yaml:"agents,omitempty"`
	Targets      []string         `yaml:"targets"`
}

// OverrideDiagnostics describes names where local config overrides global config.
type OverrideDiagnostics struct {
	Registries   []string
	Skills       []string
	Instructions []string
	Agents       []string
}

// ComputeOverrideDiagnostics returns resource names defined in both global and local manifests.
func ComputeOverrideDiagnostics(global, local *Manifest) OverrideDiagnostics {
	if global == nil || local == nil {
		return OverrideDiagnostics{}
	}

	globalRegs := make(map[string]bool)
	for _, r := range global.Registries {
		globalRegs[r.Name] = true
	}
	globalSkills := make(map[string]bool)
	for _, s := range global.Skills {
		globalSkills[s.Name] = true
	}
	globalInst := make(map[string]bool)
	for _, i := range global.Instructions {
		globalInst[i.Name] = true
	}
	globalAgents := make(map[string]bool)
	for _, a := range global.Agents {
		globalAgents[a.Name] = true
	}

	d := OverrideDiagnostics{}
	for _, r := range local.Registries {
		if globalRegs[r.Name] {
			d.Registries = append(d.Registries, r.Name)
		}
	}
	for _, s := range local.Skills {
		if globalSkills[s.Name] {
			d.Skills = append(d.Skills, s.Name)
		}
	}
	for _, i := range local.Instructions {
		if globalInst[i.Name] {
			d.Instructions = append(d.Instructions, i.Name)
		}
	}
	for _, a := range local.Agents {
		if globalAgents[a.Name] {
			d.Agents = append(d.Agents, a.Name)
		}
	}

	sort.Strings(d.Registries)
	sort.Strings(d.Skills)
	sort.Strings(d.Instructions)
	sort.Strings(d.Agents)
	return d
}

// SkillRef is a reference to a skill in the manifest.
type SkillRef struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path,omitempty"`
	Version string `yaml:"version,omitempty"`
}

// InstructionRef is a reference to an instruction in the manifest.
type InstructionRef struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content,omitempty"`
	Path    string `yaml:"path,omitempty"`
	ApplyTo string `yaml:"apply_to,omitempty"`
}

// AgentRef is a reference to an agent in the manifest.
type AgentRef struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path,omitempty"`
	Registry string `yaml:"registry,omitempty"`
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
// Returns error if: no resources defined, invalid target name, or invalid instruction/agent refs.
func (m *Manifest) Validate() error {
	resourceCount := len(m.Skills) + len(m.Instructions) + len(m.Agents)
	if resourceCount == 0 {
		return fmt.Errorf("manifest must define at least one resource (skill, instruction, or agent)")
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
	for i, inst := range m.Instructions {
		if inst.Name == "" {
			return fmt.Errorf("instruction[%d]: name is required", i)
		}
		if inst.Content != "" && inst.Path != "" {
			return fmt.Errorf("instruction %q: content and path are mutually exclusive", inst.Name)
		}
		if inst.Content == "" && inst.Path == "" {
			return fmt.Errorf("instruction %q: one of content or path is required", inst.Name)
		}
	}
	for i, agent := range m.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agent[%d]: name is required", i)
		}
		if agent.Path != "" && agent.Registry != "" {
			return fmt.Errorf("agent %q: path and registry are mutually exclusive", agent.Name)
		}
		if agent.Path == "" && agent.Registry == "" {
			return fmt.Errorf("agent %q: one of path or registry is required", agent.Name)
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

// LoadManifestFromProject searches a project directory for vibes.yaml (preferred)
// or vibes.yml (legacy fallback). Returns the parsed manifest and the path that was loaded.
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
//   - Instructions: merged by Name; project overrides global for same name
//   - Agents: merged by Name; project overrides global for same name
//   - Targets: project targets override global (no merge)
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
		resolveManifestPaths(g, filepath.Dir(globalPath))
		global = g
	}

	// Load project manifest (optional)
	if p, pPath, err := LoadManifestFromProject(projectDir); err == nil {
		resolveManifestPaths(p, filepath.Dir(pPath))
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

	// Instructions: merge by Name, project wins
	instMap := make(map[string]InstructionRef)
	var instOrder []string
	for _, inst := range global.Instructions {
		instMap[inst.Name] = inst
		instOrder = append(instOrder, inst.Name)
	}
	for _, inst := range project.Instructions {
		if _, exists := instMap[inst.Name]; !exists {
			instOrder = append(instOrder, inst.Name)
		}
		instMap[inst.Name] = inst // project overrides
	}
	for _, name := range instOrder {
		merged.Instructions = append(merged.Instructions, instMap[name])
	}
	if len(merged.Instructions) == 0 {
		merged.Instructions = nil
	}

	// Agents: merge by Name, project wins
	agentMap := make(map[string]AgentRef)
	var agentOrder []string
	for _, a := range global.Agents {
		agentMap[a.Name] = a
		agentOrder = append(agentOrder, a.Name)
	}
	for _, a := range project.Agents {
		if _, exists := agentMap[a.Name]; !exists {
			agentOrder = append(agentOrder, a.Name)
		}
		agentMap[a.Name] = a // project overrides
	}
	for _, name := range agentOrder {
		merged.Agents = append(merged.Agents, agentMap[name])
	}
	if len(merged.Agents) == 0 {
		merged.Agents = nil
	}

	return merged, nil
}

// resolveManifestPaths converts relative paths inside a manifest to absolute
// paths using baseDir as the source root.
func resolveManifestPaths(m *Manifest, baseDir string) {
	if m == nil || baseDir == "" {
		return
	}
	for i := range m.Skills {
		if m.Skills[i].Path != "" && !filepath.IsAbs(m.Skills[i].Path) {
			m.Skills[i].Path = filepath.Join(baseDir, m.Skills[i].Path)
		}
	}
	for i := range m.Instructions {
		if m.Instructions[i].Path != "" && !filepath.IsAbs(m.Instructions[i].Path) {
			m.Instructions[i].Path = filepath.Join(baseDir, m.Instructions[i].Path)
		}
	}
	for i := range m.Agents {
		if m.Agents[i].Path != "" && !filepath.IsAbs(m.Agents[i].Path) {
			m.Agents[i].Path = filepath.Join(baseDir, m.Agents[i].Path)
		}
	}
}
