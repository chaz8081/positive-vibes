package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/spf13/cobra"
)

// ResourceType identifies which manifest resource type a command operates on.
type ResourceType string

const (
	ResourceSkills       ResourceType = "skills"
	ResourceAgents       ResourceType = "agents"
	ResourceInstructions ResourceType = "instructions"
	ResourceTargets      ResourceType = "targets"
	ResourceRegistries   ResourceType = "registries"
)

// ValidResourceTypes returns the list of supported resource type strings.
func ValidResourceTypes() []string {
	return []string{string(ResourceSkills), string(ResourceAgents), string(ResourceInstructions), string(ResourceTargets), string(ResourceRegistries)}
}

// ParseResourceType validates and returns a ResourceType from a string.
func ParseResourceType(s string) (ResourceType, error) {
	switch ResourceType(s) {
	case ResourceSkills, ResourceAgents, ResourceInstructions, ResourceTargets, ResourceRegistries:
		return ResourceType(s), nil
	default:
		return "", fmt.Errorf("unknown resource type %q (valid: %s)", s, strings.Join(ValidResourceTypes(), ", "))
	}
}

// --- Resource item abstraction ---

// ResourceItem is a generic item with a name and optional metadata,
// used to unify skills, agents, and instructions for list/show/install/remove.
type ResourceItem struct {
	Name         string
	Installed    bool
	InstallScope string
}

const (
	installScopeNone   = "none"
	installScopeLocal  = "local"
	installScopeGlobal = "global"
	installScopeBoth   = "both"
)

// ResourceDetailResult describes a fully-resolved resource for show operations.
type ResourceDetailResult struct {
	Kind        ResourceType
	Name        string
	Installed   bool
	Registry    string
	RegistryURL string
	Path        string
	Payload     any
}

// MergeResourceItems merges available and installed resources into a deduplicated
// list keyed by Name. Installed entries always win for the Installed flag.
func MergeResourceItems(available, installed []ResourceItem) []ResourceItem {
	byName := make(map[string]ResourceItem, len(available)+len(installed))
	for _, item := range available {
		if item.Name == "" {
			continue
		}
		item.Installed = false
		if item.InstallScope == "" {
			item.InstallScope = installScopeNone
		}
		byName[item.Name] = item
	}
	for _, item := range installed {
		if item.Name == "" {
			continue
		}
		item.Installed = true
		if item.InstallScope == "" {
			item.InstallScope = installScopeLocal
		}
		byName[item.Name] = item
	}

	merged := make([]ResourceItem, 0, len(byName))
	for _, item := range byName {
		merged = append(merged, item)
	}
	return merged
}

// ListAvailableResourceItems returns discoverable resources for a type.
func ListAvailableResourceItems(projectDir, globalPath, kind string) ([]ResourceItem, error) {
	resType, err := ParseResourceType(kind)
	if err != nil {
		return nil, err
	}
	merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)

	switch resType {
	case ResourceSkills:
		return collectAvailableSkills(merged), nil
	case ResourceAgents:
		return collectAvailableAgents(merged), nil
	case ResourceInstructions:
		return collectAvailableInstructions(merged), nil
	case ResourceTargets:
		return collectAvailableTargets(merged), nil
	case ResourceRegistries:
		return collectAvailableRegistries(merged), nil
	default:
		return nil, fmt.Errorf("unknown resource type %q", kind)
	}
}

// ListInstalledResourceItems returns installed resources for a type.
func ListInstalledResourceItems(projectDir, globalPath, kind string) ([]ResourceItem, error) {
	resType, err := ParseResourceType(kind)
	if err != nil {
		return nil, err
	}
	merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
	local, _, _ := manifest.LoadManifestFromProject(projectDir)
	global, _ := manifest.LoadManifest(globalPath)

	switch resType {
	case ResourceSkills:
		return collectInstalledSkillsWithScope(local, global, merged), nil
	case ResourceAgents:
		return collectAgentsWithScope(local, global, merged), nil
	case ResourceInstructions:
		return collectInstructionsWithScope(local, global, merged), nil
	case ResourceTargets:
		return collectTargetsWithScope(local, global, merged), nil
	case ResourceRegistries:
		return collectInstalledRegistriesWithScope(local, global, merged), nil
	default:
		return nil, fmt.Errorf("unknown resource type %q", kind)
	}
}

// ShowResourceDetail resolves a resource detail using merged manifest + registries.
func ShowResourceDetail(projectDir, globalPath, kind, name string) (ResourceDetailResult, error) {
	resType, err := ParseResourceType(kind)
	if err != nil {
		return ResourceDetailResult{}, err
	}
	if name == "" {
		return ResourceDetailResult{}, fmt.Errorf("resource name is required")
	}

	merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)

	switch resType {
	case ResourceSkills:
		skill, regName, err := resolveSkillFromSources(name, buildAllSources(merged))
		if err != nil {
			return ResourceDetailResult{}, err
		}
		files := collectSkillPreviewFiles(projectDir, merged, name, regName)
		payload := map[string]any{
			"skill":       skill,
			"description": skill.Description,
			"content":     skill.Instructions,
			"files":       files,
		}
		return ResourceDetailResult{
			Kind:        resType,
			Name:        name,
			Installed:   hasInstalledSkill(merged, name),
			Registry:    regName,
			RegistryURL: registryURLByName(merged, regName),
			Payload:     payload,
		}, nil
	case ResourceAgents:
		if merged != nil {
			for _, a := range merged.Agents {
				if a.Name == name {
					content := loadResourceContent(projectDir, merged, string(resType), a.Registry, a.Path)
					return ResourceDetailResult{
						Kind:      resType,
						Name:      name,
						Installed: true,
						Registry:  a.Registry,
						Path:      a.Path,
						Payload: map[string]any{
							"name":        a.Name,
							"path":        a.Path,
							"registry":    a.Registry,
							"description": "Agent instructions",
							"content":     content,
						},
					}, nil
				}
			}
		}
		for _, ref := range collectRegistryResourceItems(merged, resType) {
			if ref.Name == name {
				content := loadResourceContent(projectDir, merged, string(resType), ref.Registry, ref.Path)
				return ResourceDetailResult{
					Kind:      resType,
					Name:      name,
					Installed: false,
					Registry:  ref.Registry,
					Path:      ref.Path,
					Payload: map[string]any{
						"name":        name,
						"path":        ref.Path,
						"registry":    ref.Registry,
						"description": "Agent instructions",
						"content":     content,
					},
				}, nil
			}
		}
		return ResourceDetailResult{}, fmt.Errorf("agent not found: %s", name)
	case ResourceInstructions:
		if merged != nil {
			for _, inst := range merged.Instructions {
				if inst.Name == name {
					content := inst.Content
					if content == "" {
						content = loadResourceContent(projectDir, merged, string(resType), inst.Registry, inst.Path)
					}
					return ResourceDetailResult{
						Kind:      resType,
						Name:      name,
						Installed: true,
						Registry:  inst.Registry,
						Path:      inst.Path,
						Payload: map[string]any{
							"name":        inst.Name,
							"path":        inst.Path,
							"registry":    inst.Registry,
							"description": "Instruction content",
							"content":     content,
						},
					}, nil
				}
			}
		}
		for _, ref := range collectRegistryResourceItems(merged, resType) {
			if ref.Name == name {
				content := loadResourceContent(projectDir, merged, string(resType), ref.Registry, ref.Path)
				return ResourceDetailResult{
					Kind:      resType,
					Name:      name,
					Installed: false,
					Registry:  ref.Registry,
					Path:      ref.Path,
					Payload: map[string]any{
						"name":        name,
						"path":        ref.Path,
						"registry":    ref.Registry,
						"description": "Instruction content",
						"content":     content,
					},
				}, nil
			}
		}
		return ResourceDetailResult{}, fmt.Errorf("instruction not found: %s", name)
	case ResourceTargets:
		if !contains(manifest.ValidTargets, name) {
			return ResourceDetailResult{}, fmt.Errorf("target not found: %s", name)
		}
		installed := false
		if merged != nil {
			for _, t := range merged.Targets {
				if t == name {
					installed = true
					break
				}
			}
		}
		return ResourceDetailResult{
			Kind:      resType,
			Name:      name,
			Installed: installed,
			Payload: map[string]any{
				"name":        name,
				"description": "Target platform",
				"content":     fmt.Sprintf("Target: %s\nUse apply to sync skills/instructions/agents to this platform.", name),
			},
		}, nil
	case ResourceRegistries:
		if merged == nil {
			return ResourceDetailResult{}, fmt.Errorf("registry not found: %s", name)
		}
		for _, reg := range merged.Registries {
			if reg.Name == name {
				return ResourceDetailResult{
					Kind:      resType,
					Name:      name,
					Installed: true,
					Registry:  reg.Name,
					Payload: map[string]any{
						"name":        reg.Name,
						"description": "Registry source",
						"content":     formatRegistryDetailContent(reg),
					},
				}, nil
			}
		}
		return ResourceDetailResult{}, fmt.Errorf("registry not found: %s", name)
	default:
		return ResourceDetailResult{}, fmt.Errorf("unknown resource type %q", kind)
	}
}

func formatRegistryDetailContent(reg manifest.RegistryRef) string {
	skillsRoot := reg.Paths["skills"]
	if skillsRoot == "" {
		skillsRoot = "skills/"
	}
	instructionsRoot, instructionsExplicit := reg.Paths["instructions"]
	if instructionsRoot == "" {
		instructionsRoot = skillsRoot
	}
	agentsRoot, agentsExplicit := reg.Paths["agents"]
	if agentsRoot == "" {
		agentsRoot = skillsRoot
	}

	instructionsSuffix := ""
	if !instructionsExplicit {
		instructionsSuffix = " (inherited)"
	}
	agentsSuffix := ""
	if !agentsExplicit {
		agentsSuffix = " (inherited)"
	}

	return fmt.Sprintf("URL: %s\nRef: %s\n\npaths:\n- skills: %s\n- instructions: %s%s\n- agents: %s%s", reg.URL, reg.Ref, skillsRoot, instructionsRoot, instructionsSuffix, agentsRoot, agentsSuffix)
}

func loadResourceContent(projectDir string, merged *manifest.Manifest, kind, registryName, path string) string {
	if path == "" {
		return ""
	}
	if registryName == "" {
		resolved := path
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(projectDir, resolved)
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
	for _, src := range gitRegistriesFromManifest(merged) {
		if src.Name() != registryName {
			continue
		}
		resourceSource, ok := src.(registry.ResourceSource)
		if !ok {
			return ""
		}
		data, err := resourceSource.FetchResourceFile(kind, path)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
	return ""
}

func collectSkillPreviewFiles(projectDir string, merged *manifest.Manifest, skillName, registryName string) []map[string]any {
	skillRef := findSkillRefByName(merged, skillName)
	skillDir := ""
	if skillRef != nil && skillRef.Path != "" {
		skillDir = skillRef.Path
	} else {
		skillDir = skillName
	}

	if registryName == "" {
		resolved := skillDir
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(projectDir, resolved)
		}
		if strings.EqualFold(filepath.Base(resolved), "SKILL.md") {
			resolved = filepath.Dir(resolved)
		}
		return collectLocalFiles(resolved)
	}
	if strings.EqualFold(filepath.Base(skillDir), "SKILL.md") {
		skillDir = filepath.Dir(skillDir)
	}

	for _, src := range gitRegistriesFromManifest(merged) {
		if src.Name() != registryName {
			continue
		}
		return collectSkillPreviewFilesFromRegistrySource(src, skillDir)
	}
	return nil
}

func collectSkillPreviewFilesFromRegistrySource(src registry.SkillSource, skillDir string) []map[string]any {
	if rs, ok := src.(registry.ResourceSource); ok {
		all, err := rs.ListResourceFiles("skills")
		if err != nil {
			return nil
		}
		prefix := filepath.ToSlash(strings.TrimSuffix(skillDir, "/")) + "/"
		var names []string
		for _, p := range all {
			normalized := filepath.ToSlash(p)
			if strings.HasPrefix(normalized, prefix) {
				names = append(names, strings.TrimPrefix(normalized, prefix))
			}
		}
		sort.Strings(names)
		files := make([]map[string]any, 0, len(names))
		for _, n := range names {
			data, err := rs.FetchResourceFile("skills", filepath.ToSlash(filepath.Join(skillDir, n)))
			if err != nil {
				continue
			}
			binary := isBinary(data)
			content := ""
			if !binary {
				content = strings.TrimSpace(string(data))
			}
			files = append(files, map[string]any{"name": n, "content": content, "binary": binary})
		}
		if len(files) > 0 {
			return files
		}
	}

	fs, ok := src.(registry.FileSource)
	if !ok {
		return nil
	}
	names, err := fs.ListFiles(skillDir, "")
	if err != nil {
		return nil
	}
	sort.Strings(names)
	files := make([]map[string]any, 0, len(names))
	for _, n := range names {
		data, err := fs.FetchFile(skillDir, n)
		if err != nil {
			continue
		}
		binary := isBinary(data)
		content := ""
		if !binary {
			content = strings.TrimSpace(string(data))
		}
		files = append(files, map[string]any{"name": n, "content": content, "binary": binary})
	}
	return files
}

func collectLocalFiles(root string) []map[string]any {
	if strings.EqualFold(filepath.Base(root), "SKILL.md") {
		root = filepath.Dir(root)
	}
	stat, err := os.Stat(root)
	if err != nil || !stat.IsDir() {
		return nil
	}
	var files []map[string]any
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		binary := isBinary(data)
		content := ""
		if !binary {
			content = strings.TrimSpace(string(data))
		}
		files = append(files, map[string]any{"name": filepath.ToSlash(rel), "content": content, "binary": binary})
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		left, _ := files[i]["name"].(string)
		right, _ := files[j]["name"].(string)
		return left < right
	})
	return files
}

func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if strings.IndexByte(string(data), 0) >= 0 {
		return true
	}
	for _, b := range data {
		if b < 9 {
			return true
		}
	}
	return false
}

func findSkillRefByName(m *manifest.Manifest, name string) *manifest.SkillRef {
	if m == nil {
		return nil
	}
	for i := range m.Skills {
		if m.Skills[i].Name == name {
			return &m.Skills[i]
		}
	}
	return nil
}

type registryResourceItem struct {
	Name     string
	Registry string
	Path     string
}

// --- Registry-based skill sets (reused from skills.go) ---

// registrySkillSet holds the skills available from a single registry source.
type registrySkillSet struct {
	RegistryName string
	URL          string // empty for embedded
	Skills       []string
	Error        string // non-empty if listing failed
}

// collectSkillSets gathers available skills from the embedded registry and
// any git registries defined in the merged manifest. If merged is nil, only
// the embedded registry is consulted.
func collectSkillSets(merged *manifest.Manifest) []registrySkillSet {
	var sets []registrySkillSet

	// Always include embedded registry first.
	embedded := registry.NewEmbeddedRegistry()
	if names, err := embedded.List(); err == nil && len(names) > 0 {
		sets = append(sets, registrySkillSet{
			RegistryName: "embedded",
			Skills:       names,
		})
	}

	// Add git registries from merged manifest.
	if merged != nil {
		for _, src := range gitRegistriesFromManifest(merged) {
			names, err := src.List()
			if err != nil {
				sets = append(sets, registrySkillSet{
					RegistryName: src.Name(),
					Error:        err.Error(),
				})
				continue
			}
			if len(names) > 0 {
				var url string
				for _, r := range merged.Registries {
					if r.Name == src.Name() {
						url = r.URL
						break
					}
				}
				sets = append(sets, registrySkillSet{
					RegistryName: src.Name(),
					URL:          url,
					Skills:       names,
				})
			}
		}
	}

	return sets
}

// --- List formatting ---

// listFormatOptions controls filtering for the list output.
type listFormatOptions struct {
	Registry      string // filter to a specific registry name
	InstalledOnly bool   // show only installed skills
}

// formatSkillsList renders the skill sets into a human-readable string,
// grouped by registry. Skills present in the installed map are marked.
func formatSkillsList(sets []registrySkillSet, installed map[string]bool) string {
	return formatSkillsListFiltered(sets, installed, listFormatOptions{})
}

// formatSkillsListFiltered renders skill sets with optional filters applied.
func formatSkillsListFiltered(sets []registrySkillSet, installed map[string]bool, opts listFormatOptions) string {
	var filtered []registrySkillSet
	for _, s := range sets {
		if opts.Registry != "" && s.RegistryName != opts.Registry {
			continue
		}
		if opts.InstalledOnly {
			var kept []string
			for _, name := range s.Skills {
				if installed[name] {
					kept = append(kept, name)
				}
			}
			if len(kept) == 0 && s.Error == "" {
				continue
			}
			s.Skills = kept
		}
		filtered = append(filtered, s)
	}

	if len(filtered) == 0 {
		return "No skills found matching the given filters.\n"
	}

	var b strings.Builder
	totalAvailable := 0
	totalInstalled := 0
	printedAny := false

	for _, s := range filtered {
		if len(s.Skills) == 0 && s.Error == "" {
			continue
		}
		if printedAny {
			b.WriteString("\n")
		}
		printedAny = true

		if s.RegistryName == "embedded" {
			b.WriteString("Embedded:\n")
		} else if s.URL != "" {
			fmt.Fprintf(&b, "%s (%s):\n", s.RegistryName, s.URL)
		} else {
			fmt.Fprintf(&b, "%s:\n", s.RegistryName)
		}

		if s.Error != "" {
			fmt.Fprintf(&b, "  (error: %s)\n", s.Error)
			continue
		}

		for _, name := range s.Skills {
			totalAvailable++
			if installed[name] {
				totalInstalled++
				fmt.Fprintf(&b, "  %s  [installed]\n", name)
			} else {
				fmt.Fprintf(&b, "  %s\n", name)
			}
		}
	}

	if !printedAny {
		return "No skills found matching the given filters.\n"
	}

	fmt.Fprintf(&b, "\n%d installed, %d available\n", totalInstalled, totalAvailable)
	return b.String()
}

// --- Agents/Instructions list formatting ---

// formatResourceList renders agents or instructions from the manifest as a
// human-readable list.
func formatResourceList(resType ResourceType, items []ResourceItem) string {
	if len(items) == 0 {
		return fmt.Sprintf("No %s found.\n", resType)
	}

	var b strings.Builder
	label := string(resType)
	if len(label) > 0 {
		label = strings.ToUpper(label[:1]) + label[1:]
	}
	fmt.Fprintf(&b, "%s:\n", label)

	installed := 0
	for _, item := range items {
		if item.Installed {
			installed++
			fmt.Fprintf(&b, "  %s  [installed]\n", item.Name)
		} else {
			fmt.Fprintf(&b, "  %s\n", item.Name)
		}
	}
	fmt.Fprintf(&b, "\n%d installed, %d available\n", installed, len(items))
	return b.String()
}

// --- JSON output types ---

type skillsListJSON struct {
	Registries     []registryJSON `json:"registries"`
	TotalAvailable int            `json:"total_available"`
	TotalInstalled int            `json:"total_installed"`
}

type registryJSON struct {
	Name   string      `json:"name"`
	URL    string      `json:"url,omitempty"`
	Error  string      `json:"error,omitempty"`
	Skills []skillJSON `json:"skills,omitempty"`
}

type skillJSON struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// formatSkillsListJSON renders the skill sets as a JSON string.
func formatSkillsListJSON(sets []registrySkillSet, installed map[string]bool) string {
	result := skillsListJSON{}
	for _, s := range sets {
		reg := registryJSON{
			Name:  s.RegistryName,
			URL:   s.URL,
			Error: s.Error,
		}
		for _, name := range s.Skills {
			isInstalled := installed[name]
			reg.Skills = append(reg.Skills, skillJSON{Name: name, Installed: isInstalled})
			result.TotalAvailable++
			if isInstalled {
				result.TotalInstalled++
			}
		}
		result.Registries = append(result.Registries, reg)
	}
	if result.Registries == nil {
		result.Registries = []registryJSON{}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data)
}

// --- Resource list JSON for agents/instructions ---

type resourceListJSON struct {
	Type  string             `json:"type"`
	Items []resourceItemJSON `json:"items"`
	Total int                `json:"total"`
}

type resourceItemJSON struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

func formatResourceListJSON(resType ResourceType, items []ResourceItem) string {
	result := resourceListJSON{
		Type:  string(resType),
		Items: make([]resourceItemJSON, 0, len(items)),
		Total: len(items),
	}
	for _, item := range items {
		result.Items = append(result.Items, resourceItemJSON{
			Name:      item.Name,
			Installed: item.Installed,
		})
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data)
}

// --- Show helpers ---

// buildAllSources returns the embedded registry plus any git registries
// from the merged manifest as a unified slice of SkillSource.
func buildAllSources(merged *manifest.Manifest) []registry.SkillSource {
	sources := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	if merged != nil {
		sources = append(sources, gitRegistriesFromManifest(merged)...)
	}
	return sources
}

// resolveSkillFromSources searches registries in order and returns the first
// matching skill along with the registry name that provided it.
func resolveSkillFromSources(name string, sources []registry.SkillSource) (*schema.Skill, string, error) {
	for _, src := range sources {
		skill, _, err := src.Fetch(name)
		if err == nil {
			return skill, src.Name(), nil
		}
	}
	return nil, "", fmt.Errorf("skill not found: %s", name)
}

// formatSkillShow renders a single skill's details as a human-readable string.
func formatSkillShow(skill *schema.Skill, registryName, registryURL string, installed bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Name: %s\n", skill.Name)
	if skill.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", skill.Description)
	}
	if skill.Version != "" {
		fmt.Fprintf(&b, "Version: %s\n", skill.Version)
	}
	if skill.Author != "" {
		fmt.Fprintf(&b, "Author: %s\n", skill.Author)
	}
	if len(skill.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(skill.Tags, ", "))
	}

	if registryURL != "" {
		fmt.Fprintf(&b, "Registry: %s (%s)\n", registryName, registryURL)
	} else {
		fmt.Fprintf(&b, "Registry: %s\n", registryName)
	}

	if installed {
		b.WriteString("Status: installed\n")
	} else {
		b.WriteString("Status: not installed\n")
	}

	if skill.Instructions != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(skill.Instructions)
		b.WriteString("\n")
	}

	return b.String()
}

// formatAgentShow renders an agent's details.
func formatAgentShow(agent manifest.AgentRef, installed bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\n", agent.Name)
	if agent.Path != "" {
		fmt.Fprintf(&b, "Path: %s\n", agent.Path)
	}
	if agent.Registry != "" {
		fmt.Fprintf(&b, "Registry: %s\n", agent.Registry)
	}
	if installed {
		b.WriteString("Status: installed\n")
	} else {
		b.WriteString("Status: available\n")
	}
	return b.String()
}

// formatInstructionShow renders an instruction's details.
func formatInstructionShow(inst manifest.InstructionRef, installed bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\n", inst.Name)
	if inst.Path != "" {
		fmt.Fprintf(&b, "Path: %s\n", inst.Path)
	}
	if inst.Registry != "" {
		fmt.Fprintf(&b, "Registry: %s\n", inst.Registry)
	}
	if inst.ApplyTo != "" {
		fmt.Fprintf(&b, "ApplyTo: %s\n", inst.ApplyTo)
	}
	if installed {
		b.WriteString("Status: installed\n")
	} else {
		b.WriteString("Status: available\n")
	}
	if inst.Content != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(inst.Content)
		b.WriteString("\n")
	}
	return b.String()
}

// --- Collecting items for interactive pickers ---

// collectAvailableSkills returns all skill names from all registries, with
// install status, suitable for a picker.
func collectAvailableSkills(merged *manifest.Manifest) []ResourceItem {
	sets := collectSkillSets(merged)
	installed := buildInstalledSkillsMap(merged)

	seen := make(map[string]bool)
	var items []ResourceItem
	for _, s := range sets {
		for _, name := range s.Skills {
			if seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, ResourceItem{
				Name:         name,
				Installed:    installed[name],
				InstallScope: installScopeNone,
			})
		}
	}
	return items
}

// collectInstalledSkills returns only installed skill names.
func collectInstalledSkills(merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	var items []ResourceItem
	for _, s := range merged.Skills {
		items = append(items, ResourceItem{
			Name:         s.Name,
			Installed:    true,
			InstallScope: installScopeLocal,
		})
	}
	return items
}

func collectInstalledSkillsWithScope(local, global, merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	localSet := make(map[string]bool)
	if local != nil {
		for _, s := range local.Skills {
			localSet[s.Name] = true
		}
	}
	globalSet := make(map[string]bool)
	if global != nil {
		for _, s := range global.Skills {
			globalSet[s.Name] = true
		}
	}
	items := make([]ResourceItem, 0, len(merged.Skills))
	for _, s := range merged.Skills {
		items = append(items, ResourceItem{Name: s.Name, Installed: true, InstallScope: resolveInstallScope(localSet[s.Name], globalSet[s.Name])})
	}
	return items
}

// collectAgents returns agents from the merged manifest.
func collectAgents(merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	var items []ResourceItem
	for _, a := range merged.Agents {
		items = append(items, ResourceItem{
			Name:         a.Name,
			Installed:    true,
			InstallScope: installScopeLocal,
		})
	}
	return items
}

func collectAgentsWithScope(local, global, merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	localSet := make(map[string]bool)
	if local != nil {
		for _, a := range local.Agents {
			localSet[a.Name] = true
		}
	}
	globalSet := make(map[string]bool)
	if global != nil {
		for _, a := range global.Agents {
			globalSet[a.Name] = true
		}
	}
	items := make([]ResourceItem, 0, len(merged.Agents))
	for _, a := range merged.Agents {
		items = append(items, ResourceItem{Name: a.Name, Installed: true, InstallScope: resolveInstallScope(localSet[a.Name], globalSet[a.Name])})
	}
	return items
}

func collectAvailableAgents(merged *manifest.Manifest) []ResourceItem {
	installed := make(map[string]bool)
	if merged != nil {
		for _, a := range merged.Agents {
			installed[a.Name] = true
		}
	}
	refs := collectRegistryResourceItems(merged, ResourceAgents)
	var items []ResourceItem
	for _, ref := range refs {
		items = append(items, ResourceItem{Name: ref.Name, Installed: installed[ref.Name], InstallScope: installScopeNone})
	}
	return items
}

// collectInstructions returns instructions from the merged manifest.
func collectInstructions(merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	var items []ResourceItem
	for _, inst := range merged.Instructions {
		items = append(items, ResourceItem{
			Name:         inst.Name,
			Installed:    true,
			InstallScope: installScopeLocal,
		})
	}
	return items
}

func collectInstructionsWithScope(local, global, merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	localSet := make(map[string]bool)
	if local != nil {
		for _, i := range local.Instructions {
			localSet[i.Name] = true
		}
	}
	globalSet := make(map[string]bool)
	if global != nil {
		for _, i := range global.Instructions {
			globalSet[i.Name] = true
		}
	}
	items := make([]ResourceItem, 0, len(merged.Instructions))
	for _, i := range merged.Instructions {
		items = append(items, ResourceItem{Name: i.Name, Installed: true, InstallScope: resolveInstallScope(localSet[i.Name], globalSet[i.Name])})
	}
	return items
}

func collectAvailableInstructions(merged *manifest.Manifest) []ResourceItem {
	installed := make(map[string]bool)
	if merged != nil {
		for _, i := range merged.Instructions {
			installed[i.Name] = true
		}
	}
	refs := collectRegistryResourceItems(merged, ResourceInstructions)
	var items []ResourceItem
	for _, ref := range refs {
		items = append(items, ResourceItem{Name: ref.Name, Installed: installed[ref.Name], InstallScope: installScopeNone})
	}
	return items
}

func collectTargets(merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	items := make([]ResourceItem, 0, len(merged.Targets))
	for _, t := range merged.Targets {
		items = append(items, ResourceItem{Name: t, Installed: true, InstallScope: installScopeLocal})
	}
	return items
}

func collectTargetsWithScope(local, global, merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	localSet := make(map[string]bool)
	if local != nil {
		for _, t := range local.Targets {
			localSet[t] = true
		}
	}
	globalSet := make(map[string]bool)
	if global != nil {
		for _, t := range global.Targets {
			globalSet[t] = true
		}
	}
	items := make([]ResourceItem, 0, len(merged.Targets))
	for _, t := range merged.Targets {
		items = append(items, ResourceItem{Name: t, Installed: true, InstallScope: resolveInstallScope(localSet[t], globalSet[t])})
	}
	return items
}

func collectAvailableTargets(merged *manifest.Manifest) []ResourceItem {
	installed := make(map[string]bool)
	if merged != nil {
		for _, t := range merged.Targets {
			installed[t] = true
		}
	}
	items := make([]ResourceItem, 0, len(manifest.ValidTargets))
	for _, t := range manifest.ValidTargets {
		items = append(items, ResourceItem{Name: t, Installed: installed[t], InstallScope: installScopeNone})
	}
	return items
}

func collectInstalledRegistries(local *manifest.Manifest) []ResourceItem {
	if local == nil {
		return nil
	}
	items := make([]ResourceItem, 0, len(local.Registries))
	for _, r := range local.Registries {
		items = append(items, ResourceItem{Name: r.Name, Installed: true, InstallScope: installScopeLocal})
	}
	return items
}

func collectInstalledRegistriesWithScope(local, global, merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	localSet := make(map[string]bool)
	if local != nil {
		for _, r := range local.Registries {
			localSet[r.Name] = true
		}
	}
	globalSet := make(map[string]bool)
	if global != nil {
		for _, r := range global.Registries {
			globalSet[r.Name] = true
		}
	}
	items := make([]ResourceItem, 0, len(merged.Registries))
	for _, r := range merged.Registries {
		items = append(items, ResourceItem{Name: r.Name, Installed: true, InstallScope: resolveInstallScope(localSet[r.Name], globalSet[r.Name])})
	}
	return items
}

func collectAvailableRegistries(merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return []ResourceItem{{Name: "awesome-copilot", Installed: false, InstallScope: installScopeNone}}
	}
	items := make([]ResourceItem, 0, len(merged.Registries))
	for _, r := range merged.Registries {
		items = append(items, ResourceItem{Name: r.Name, Installed: false, InstallScope: installScopeNone})
	}
	if len(items) == 0 {
		items = append(items, ResourceItem{Name: "awesome-copilot", Installed: false, InstallScope: installScopeNone})
	}
	return items
}

func resolveInstallScope(localInstalled, globalInstalled bool) string {
	if localInstalled && globalInstalled {
		return installScopeBoth
	}
	if localInstalled {
		return installScopeLocal
	}
	if globalInstalled {
		return installScopeGlobal
	}
	return installScopeNone
}

func collectRegistryResourceItems(merged *manifest.Manifest, resType ResourceType) []registryResourceItem {
	if merged == nil {
		return nil
	}
	kind := string(resType)
	seen := make(map[string]bool)
	var items []registryResourceItem
	for _, src := range gitRegistriesFromManifest(merged) {
		fs, ok := src.(registry.ResourceSource)
		if !ok {
			continue
		}
		files, err := fs.ListResourceFiles(kind)
		if err != nil {
			continue
		}
		for _, rel := range files {
			name := resourceNameFromPath(resType, rel)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, registryResourceItem{Name: name, Registry: src.Name(), Path: rel})
		}
	}
	return items
}

func resourceNameFromPath(resType ResourceType, relPath string) string {
	base := filepath.Base(relPath)
	switch resType {
	case ResourceInstructions:
		if !strings.HasSuffix(base, ".instructions.md") {
			return ""
		}
		return strings.TrimSuffix(base, ".instructions.md")
	case ResourceAgents:
		if !strings.HasSuffix(base, ".agent.md") {
			return ""
		}
		return strings.TrimSuffix(base, ".agent.md")
	}
	if !strings.HasSuffix(base, ".md") {
		return ""
	}
	base = strings.TrimSuffix(base, ".md")
	return base
}

// buildInstalledSkillsMap builds a map of installed skill names from a manifest.
func buildInstalledSkillsMap(merged *manifest.Manifest) map[string]bool {
	installed := make(map[string]bool)
	if merged != nil {
		for _, s := range merged.Skills {
			installed[s.Name] = true
		}
	}
	return installed
}

func hasInstalledSkill(merged *manifest.Manifest, name string) bool {
	if merged == nil {
		return false
	}
	for _, s := range merged.Skills {
		if s.Name == name {
			return true
		}
	}
	return false
}

func registryURLByName(merged *manifest.Manifest, name string) string {
	if merged == nil {
		return ""
	}
	for _, r := range merged.Registries {
		if r.Name == name {
			return r.URL
		}
	}
	return ""
}

// --- Shell completion helpers ---

// resourceTypeCompletions returns the valid resource type strings as
// cobra.ShellCompDirective completions.
func resourceTypeCompletions() []string {
	return ValidResourceTypes()
}

// completeResourceNames returns name suggestions for the given resource type
// and command context. The mode parameter controls which names are returned:
//
//   - "available" — all available names (e.g. for install: skills from registries)
//   - "installed" — only installed names (e.g. for remove: skills in manifest)
//   - "all"       — both installed and available (e.g. for show)
func completeResourceNames(resType ResourceType, mode string) []string {
	project := ProjectDir()
	globalPath := defaultGlobalManifestPath()
	merged, _ := manifest.LoadMergedManifest(project, globalPath)

	switch resType {
	case ResourceSkills:
		return completeSkillNames(merged, mode)
	case ResourceAgents:
		return completeAgentNames(merged, mode)
	case ResourceInstructions:
		return completeInstructionNames(merged, mode)
	case ResourceTargets:
		return completeTargetNames(merged, mode)
	case ResourceRegistries:
		return completeRegistryNames(merged, mode)
	default:
		return nil
	}
}

func completeSkillNames(merged *manifest.Manifest, mode string) []string {
	switch mode {
	case "available":
		items := collectAvailableSkills(merged)
		var names []string
		for _, item := range items {
			if !item.Installed {
				names = append(names, item.Name)
			}
		}
		return names
	case "installed":
		items := collectInstalledSkills(merged)
		var names []string
		for _, item := range items {
			names = append(names, item.Name)
		}
		return names
	default: // "all"
		items := collectAvailableSkills(merged)
		var names []string
		for _, item := range items {
			names = append(names, item.Name)
		}
		return names
	}
}

func completeAgentNames(merged *manifest.Manifest, mode string) []string {
	switch mode {
	case "available":
		items := collectAvailableAgents(merged)
		var names []string
		for _, item := range items {
			if !item.Installed {
				names = append(names, item.Name)
			}
		}
		return names
	case "installed":
		return resourceNamesFromItems(collectAgents(merged))
	default:
		names := resourceNamesFromItems(collectAvailableAgents(merged))
		for _, n := range resourceNamesFromItems(collectAgents(merged)) {
			if !contains(names, n) {
				names = append(names, n)
			}
		}
		return names
	}
}

func completeInstructionNames(merged *manifest.Manifest, mode string) []string {
	switch mode {
	case "available":
		items := collectAvailableInstructions(merged)
		var names []string
		for _, item := range items {
			if !item.Installed {
				names = append(names, item.Name)
			}
		}
		return names
	case "installed":
		return resourceNamesFromItems(collectInstructions(merged))
	default:
		names := resourceNamesFromItems(collectAvailableInstructions(merged))
		for _, n := range resourceNamesFromItems(collectInstructions(merged)) {
			if !contains(names, n) {
				names = append(names, n)
			}
		}
		return names
	}
}

func completeTargetNames(merged *manifest.Manifest, mode string) []string {
	switch mode {
	case "available":
		items := collectAvailableTargets(merged)
		var names []string
		for _, item := range items {
			if !item.Installed {
				names = append(names, item.Name)
			}
		}
		return names
	case "installed":
		return resourceNamesFromItems(collectTargets(merged))
	default:
		return resourceNamesFromItems(collectAvailableTargets(merged))
	}
}

func completeRegistryNames(merged *manifest.Manifest, mode string) []string {
	if merged == nil {
		if mode == "installed" {
			return nil
		}
		return []string{"awesome-copilot"}
	}
	names := make([]string, 0, len(merged.Registries))
	for _, r := range merged.Registries {
		names = append(names, r.Name)
	}
	if mode == "installed" {
		return names
	}
	if !contains(names, "awesome-copilot") {
		names = append(names, "awesome-copilot")
	}
	sort.Strings(names)
	return names
}

func contains(items []string, v string) bool {
	for _, item := range items {
		if item == v {
			return true
		}
	}
	return false
}

// resourceNamesFromItems extracts name strings from a slice of ResourceItem.
func resourceNamesFromItems(items []ResourceItem) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

// makeValidArgsFunction builds a ValidArgsFunction for commands that take
// <resource-type> as the first positional arg and optional [name...] after.
// The nameMode parameter controls which names are suggested for arg positions
// after the resource type:
//
//   - "available" — not-yet-installed resources (install command)
//   - "installed" — currently installed resources (remove command)
//   - "all"       — all known resources (show command)
//   - ""          — no name suggestions (list command)
func makeValidArgsFunction(nameMode string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// First positional arg: resource type
			return resourceTypeCompletions(), cobra.ShellCompDirectiveNoFileComp
		}

		if nameMode == "" {
			// Command doesn't accept names (e.g. list)
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Validate the resource type to decide what names to suggest
		resType, err := ParseResourceType(args[0])
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Filter out names already provided on the command line
		existing := make(map[string]bool, len(args)-1)
		for _, a := range args[1:] {
			existing[a] = true
		}

		all := completeResourceNames(resType, nameMode)
		var suggestions []string
		for _, name := range all {
			if !existing[name] {
				suggestions = append(suggestions, name)
			}
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	}
}

// dedup returns a new slice with duplicate strings removed, preserving order.
func dedup(names []string) []string {
	seen := make(map[string]bool, len(names))
	result := make([]string, 0, len(names))
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	return result
}
