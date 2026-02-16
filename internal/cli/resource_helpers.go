package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

func findRegistryByName(m *manifest.Manifest, name string) (manifest.RegistryRef, bool) {
	if m == nil {
		return manifest.RegistryRef{}, false
	}
	for _, r := range m.Registries {
		if r.Name == name {
			return r, true
		}
	}
	return manifest.RegistryRef{}, false
}

func registrySourceState(name string, local, global *manifest.Manifest) (string, string) {
	lr, hasLocal := findRegistryByName(local, name)
	gr, hasGlobal := findRegistryByName(global, name)

	if hasLocal {
		if strings.TrimSpace(lr.URL) == "" {
			return "local_issue", "local registry is invalid: missing url"
		}
		if strings.TrimSpace(lr.Ref) == "" {
			return "local_issue", "local registry is invalid: missing ref"
		}
	}

	if hasLocal && hasGlobal {
		if strings.TrimSpace(lr.URL) != "" && strings.TrimSpace(gr.URL) != "" && lr.URL != gr.URL {
			return "local_conflict", "local registry conflicts with global URL"
		}
		if lr.Ref != gr.Ref || !reflect.DeepEqual(lr.Paths, gr.Paths) {
			return "local_override", "local registry overrides global definition"
		}
		return "both", "registry is defined in both local and global"
	}
	if hasLocal {
		return "local_only", "registry exists only in local config"
	}
	if hasGlobal {
		return "global_only", "registry is defined in global catalog"
	}
	return "unknown", "registry source could not be determined"
}

func formatRegistryDetailContent(reg manifest.RegistryRef) string {
	skillsRoot := reg.Paths["skills"]
	if skillsRoot == "" {
		skillsRoot = "skills/"
	}
	instructionsRoot := reg.Paths["instructions"]
	if instructionsRoot == "" {
		instructionsRoot = "instructions/"
	}
	agentsRoot := reg.Paths["agents"]
	if agentsRoot == "" {
		agentsRoot = "agents/"
	}
	promptsRoot := reg.Paths["prompts"]
	if promptsRoot == "" {
		promptsRoot = "prompts/"
	}

	return fmt.Sprintf("URL: %s\nRef: %s\n\npaths:\n- skills: %s\n- instructions: %s\n- agents: %s\n- prompts: %s", reg.URL, reg.Ref, skillsRoot, instructionsRoot, agentsRoot, promptsRoot)
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
		skill, err := fetchSkillForRead(src, name)
		if err == nil {
			return skill, src.Name(), nil
		}
	}
	return nil, "", fmt.Errorf("skill not found: %s", name)
}

func fetchSkillForRead(src registry.SkillSource, name string) (*schema.Skill, error) {
	if cf, ok := src.(registry.CleanableFetcher); ok {
		skill, _, cleanup, err := cf.FetchWithCleanup(name)
		if cleanup != nil {
			cleanup()
		}
		return skill, err
	}

	skill, _, err := src.Fetch(name)
	return skill, err
}

func resourceNameFromPath(resType ResourceType, relPath string) string {
	base := filepath.Base(relPath)
	switch resType {
	case ResourceInstructions:
		if !strings.HasSuffix(base, ".instructions.md") {
			return ""
		}
		return strings.TrimSuffix(base, ".instructions.md")
	case ResourcePrompts:
		if strings.HasSuffix(base, ".prompt.md") {
			return strings.TrimSuffix(base, ".prompt.md")
		}
		if strings.HasSuffix(base, ".md") {
			return strings.TrimSuffix(base, ".md")
		}
		return ""
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

// dedup returns a new slice with duplicate and empty strings removed, preserving order.
func dedup(names []string) []string {
	return manifest.Dedup(names)
}
