package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
)

// ValidResourceTypes returns the list of supported resource type strings.
func ValidResourceTypes() []string {
	return []string{string(ResourceSkills), string(ResourceAgents), string(ResourceInstructions)}
}

// ParseResourceType validates and returns a ResourceType from a string.
func ParseResourceType(s string) (ResourceType, error) {
	switch ResourceType(s) {
	case ResourceSkills, ResourceAgents, ResourceInstructions:
		return ResourceType(s), nil
	default:
		return "", fmt.Errorf("unknown resource type %q (valid: %s)", s, strings.Join(ValidResourceTypes(), ", "))
	}
}

// --- Resource item abstraction ---

// ResourceItem is a generic item with a name and optional metadata,
// used to unify skills, agents, and instructions for list/show/install/remove.
type ResourceItem struct {
	Name      string
	Installed bool
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
				Name:      name,
				Installed: installed[name],
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
			Name:      s.Name,
			Installed: true,
		})
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
			Name:      a.Name,
			Installed: true,
		})
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
		items = append(items, ResourceItem{Name: ref.Name, Installed: installed[ref.Name]})
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
			Name:      inst.Name,
			Installed: true,
		})
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
		items = append(items, ResourceItem{Name: ref.Name, Installed: installed[ref.Name]})
	}
	return items
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
