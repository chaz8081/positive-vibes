package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
)

type ResourceMutationReport struct {
	MutatedNames          []string
	SkippedDuplicateNames []string
	SkippedMissingNames   []string
}

type RegistryPromotionSkip struct {
	Name   string
	Reason string
}

type RegistryPromotionReport struct {
	PromotedNames []string
	Skipped       []RegistryPromotionSkip
}

func PromoteLocalRegistriesToGlobalWithReport(projectDir, globalPath string) (RegistryPromotionReport, error) {
	report := RegistryPromotionReport{}
	if projectDir == "" || globalPath == "" {
		return report, nil
	}

	local, _, err := manifest.LoadManifestFromProject(projectDir)
	if err != nil || local == nil || len(local.Registries) == 0 {
		return report, nil
	}

	global := &manifest.Manifest{}
	if _, err := os.Stat(globalPath); err == nil {
		loaded, loadErr := manifest.LoadManifest(globalPath)
		if loadErr != nil {
			return report, loadErr
		}
		global = loaded
	}

	globalByName := make(map[string]manifest.RegistryRef, len(global.Registries))
	for _, r := range global.Registries {
		globalByName[r.Name] = r
	}

	mutated := false
	for _, r := range local.Registries {
		if r.Name == "" {
			report.Skipped = append(report.Skipped, RegistryPromotionSkip{Name: r.Name, Reason: "missing name"})
			continue
		}
		if strings.TrimSpace(r.URL) == "" {
			report.Skipped = append(report.Skipped, RegistryPromotionSkip{Name: r.Name, Reason: "missing url"})
			continue
		}
		if strings.TrimSpace(r.Ref) == "" {
			report.Skipped = append(report.Skipped, RegistryPromotionSkip{Name: r.Name, Reason: "missing ref"})
			continue
		}

		if gr, exists := globalByName[r.Name]; exists {
			if gr.URL != r.URL {
				report.Skipped = append(report.Skipped, RegistryPromotionSkip{Name: r.Name, Reason: "name exists globally with different url"})
			}
			continue
		}

		normalized := normalizeRegistryRefPaths(r)
		global.Registries = append(global.Registries, normalized)
		globalByName[r.Name] = normalized
		report.PromotedNames = append(report.PromotedNames, r.Name)
		mutated = true
	}

	if mutated {
		if err := manifest.SaveManifest(global, globalPath); err != nil {
			return report, err
		}
	}

	return report, nil
}

// InstallResourceItems installs resources by type without interactive prompts.
func InstallResourceItems(projectDir, globalPath, kind string, names []string) error {
	_, err := InstallResourceItemsWithReport(projectDir, globalPath, kind, names)
	return err
}

func InstallResourceItemsGlobal(globalPath, kind string, names []string) error {
	globalDir := filepath.Dir(globalPath)
	_, err := InstallResourceItemsWithReport(globalDir, globalPath, kind, names)
	return err
}

func InstallResourceItemsWithReport(projectDir, globalPath, kind string, names []string) (ResourceMutationReport, error) {
	uniqueNames, skippedDuplicateNames := uniqueRequestNames(names)
	report := ResourceMutationReport{SkippedDuplicateNames: skippedDuplicateNames}
	if len(uniqueNames) == 0 {
		return report, nil
	}

	resType, err := ParseResourceType(kind)
	if err != nil {
		return report, err
	}

	switch resType {
	case ResourceSkills:
		m, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			manifestPath = filepath.Join(projectDir, "vibes.yaml")
			m = &manifest.Manifest{}
		}

		merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
		installer := engine.NewInstaller(buildAllSources(merged))

		var errs []string
		for _, name := range uniqueNames {
			existing := false
			for _, s := range m.Skills {
				if s.Name == name {
					existing = true
					break
				}
			}
			if existing {
				appendUniqueName(&report.SkippedDuplicateNames, name)
				continue
			}

			localSkillPath := filepath.Join(projectDir, "skills", name, "SKILL.md")
			if _, err := os.Stat(localSkillPath); err == nil {
				m.Skills = append(m.Skills, manifest.SkillRef{Name: name, Path: "./skills/" + name})
				report.MutatedNames = append(report.MutatedNames, name)
				continue
			}

			registryName := ""
			for _, set := range collectSkillSets(merged) {
				for _, skillName := range set.Skills {
					if skillName == name {
						registryName = set.RegistryName
						break
					}
				}
				if registryName != "" {
					break
				}
			}

			if registryName == "" {
				if err := installer.Install(name, manifestPath); err != nil {
					errs = append(errs, err.Error())
					continue
				}
				report.MutatedNames = append(report.MutatedNames, name)
				continue
			}

			ref := manifest.SkillRef{Name: name}
			if registryName != "embedded" {
				ref.Registry = registryName
				ensureRegistryRefInManifest(m, merged, registryName)
			}
			m.Skills = append(m.Skills, ref)
			report.MutatedNames = append(report.MutatedNames, name)
		}
		if len(errs) > 0 {
			return report, fmt.Errorf("%s", strings.Join(errs, "; "))
		}
		return report, manifest.SaveManifest(m, manifestPath)
	case ResourceAgents, ResourceInstructions, ResourcePrompts, ResourceTargets, ResourceRegistries:
		m, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			manifestPath = filepath.Join(projectDir, "vibes.yaml")
			m = &manifest.Manifest{}
		}

		merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
		availableRefs := collectRegistryResourceItems(merged, resType)
		availableByName := make(map[string]registryResourceItem, len(availableRefs))
		for _, ref := range availableRefs {
			availableByName[ref.Name] = ref
		}

		switch resType {
		case ResourceAgents:
			existing := make(map[string]bool)
			for _, a := range m.Agents {
				existing[a.Name] = true
			}
			for _, name := range uniqueNames {
				if existing[name] {
					appendUniqueName(&report.SkippedDuplicateNames, name)
					continue
				}
				a := manifest.AgentRef{Name: name}
				if ref, ok := availableByName[name]; ok {
					a.Registry = ref.Registry
					a.Path = ref.Path
					ensureRegistryRefInManifest(m, merged, ref.Registry)
				} else {
					a.Path = fmt.Sprintf("./agents/%s.md", name)
				}
				m.Agents = append(m.Agents, a)
				existing[name] = true
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourceInstructions:
			existing := make(map[string]bool)
			for _, i := range m.Instructions {
				existing[i.Name] = true
			}
			for _, name := range uniqueNames {
				if existing[name] {
					appendUniqueName(&report.SkippedDuplicateNames, name)
					continue
				}
				i := manifest.InstructionRef{Name: name}
				if ref, ok := availableByName[name]; ok {
					i.Registry = ref.Registry
					i.Path = ref.Path
					ensureRegistryRefInManifest(m, merged, ref.Registry)
				} else {
					i.Path = fmt.Sprintf("./instructions/%s.md", name)
				}
				m.Instructions = append(m.Instructions, i)
				existing[name] = true
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourcePrompts:
			existing := make(map[string]bool)
			for _, p := range m.Prompts {
				existing[p.Name] = true
			}
			for _, name := range uniqueNames {
				if existing[name] {
					appendUniqueName(&report.SkippedDuplicateNames, name)
					continue
				}
				p := manifest.PromptRef{Name: name}
				if ref, ok := availableByName[name]; ok {
					p.Registry = ref.Registry
					p.Path = ref.Path
					ensureRegistryRefInManifest(m, merged, ref.Registry)
				} else {
					p.Path = fmt.Sprintf("./prompts/%s.prompt.md", name)
				}
				m.Prompts = append(m.Prompts, p)
				existing[name] = true
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourceTargets:
			existing := make(map[string]bool)
			for _, t := range m.Targets {
				existing[t] = true
			}
			for _, name := range uniqueNames {
				if !contains(manifest.ValidTargets, name) {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				if existing[name] {
					appendUniqueName(&report.SkippedDuplicateNames, name)
					continue
				}
				m.Targets = append(m.Targets, name)
				existing[name] = true
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourceRegistries:
			existing := make(map[string]bool)
			for _, r := range m.Registries {
				existing[r.Name] = true
			}
			for _, name := range uniqueNames {
				if existing[name] {
					appendUniqueName(&report.SkippedDuplicateNames, name)
					continue
				}
				added := false
				if merged != nil {
					for _, r := range merged.Registries {
						if r.Name == name {
							m.Registries = append(m.Registries, normalizeRegistryRefPaths(r))
							added = true
							break
						}
					}
				}
				if !added && name == "awesome-copilot" {
					m.Registries = append(m.Registries, normalizeRegistryRefPaths(manifest.RegistryRef{
						Name:  "awesome-copilot",
						URL:   "https://github.com/github/awesome-copilot",
						Ref:   "latest",
						Paths: map[string]string{"skills": "skills/"},
					}))
					added = true
				}
				if !added {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				existing[name] = true
				report.MutatedNames = append(report.MutatedNames, name)
			}
		}

		return report, manifest.SaveManifest(m, manifestPath)
	default:
		return report, fmt.Errorf("unknown resource type %q", kind)
	}
}

// RemoveResourceItems removes resources by type without interactive prompts.
func RemoveResourceItems(projectDir, kind string, names []string) error {
	_, err := RemoveResourceItemsWithReport(projectDir, kind, names)
	return err
}

func RemoveResourceItemsGlobal(globalPath, kind string, names []string) error {
	globalDir := filepath.Dir(globalPath)
	_, err := RemoveResourceItemsWithReport(globalDir, kind, names)
	return err
}

func RemoveResourceItemsGlobalWithReport(projectDir, globalPath, kind string, names []string) (ResourceMutationReport, error) {
	resType, err := ParseResourceType(kind)
	if err != nil {
		return ResourceMutationReport{}, err
	}
	if resType != ResourceRegistries {
		globalDir := filepath.Dir(globalPath)
		return RemoveResourceItemsWithReport(globalDir, kind, names)
	}

	local, _, localErr := manifest.LoadManifestFromProject(projectDir)
	if localErr == nil && local != nil {
		for _, name := range names {
			if refs := countLocalRegistryRefs(local, name); refs > 0 {
				return ResourceMutationReport{}, fmt.Errorf("cannot remove registry %q from global: referenced by local resources (%d)", name, refs)
			}
		}
	}

	globalDir := filepath.Dir(globalPath)
	return RemoveResourceItemsWithReport(globalDir, kind, names)
}

func RemoveResourceItemsWithReport(projectDir, kind string, names []string) (ResourceMutationReport, error) {
	uniqueNames, skippedDuplicateNames := uniqueRequestNames(names)
	report := ResourceMutationReport{SkippedDuplicateNames: skippedDuplicateNames}
	if len(uniqueNames) == 0 {
		return report, nil
	}

	resType, err := ParseResourceType(kind)
	if err != nil {
		return report, err
	}

	switch resType {
	case ResourceSkills:
		_, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			return report, fmt.Errorf("no manifest found in %s", projectDir)
		}

		installer := engine.NewInstaller(nil)
		var errs []string
		for _, name := range uniqueNames {
			if err := installer.Remove(name, manifestPath); err != nil {
				if strings.Contains(err.Error(), "not found in manifest") {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				errs = append(errs, err.Error())
				continue
			}
			report.MutatedNames = append(report.MutatedNames, name)
		}
		if len(errs) > 0 {
			return report, fmt.Errorf("%s", strings.Join(errs, "; "))
		}
		return report, nil
	case ResourceAgents, ResourceInstructions, ResourcePrompts, ResourceTargets, ResourceRegistries:
		m, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			return report, fmt.Errorf("no manifest found in %s", projectDir)
		}

		switch resType {
		case ResourceAgents:
			for _, name := range uniqueNames {
				idx := -1
				for i, a := range m.Agents {
					if a.Name == name {
						idx = i
						break
					}
				}
				if idx < 0 {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				m.Agents = append(m.Agents[:idx], m.Agents[idx+1:]...)
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourceInstructions:
			for _, name := range uniqueNames {
				idx := -1
				for i, inst := range m.Instructions {
					if inst.Name == name {
						idx = i
						break
					}
				}
				if idx < 0 {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				m.Instructions = append(m.Instructions[:idx], m.Instructions[idx+1:]...)
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourcePrompts:
			for _, name := range uniqueNames {
				idx := -1
				for i, p := range m.Prompts {
					if p.Name == name {
						idx = i
						break
					}
				}
				if idx < 0 {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				m.Prompts = append(m.Prompts[:idx], m.Prompts[idx+1:]...)
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourceTargets:
			for _, name := range uniqueNames {
				idx := -1
				for i, targetName := range m.Targets {
					if targetName == name {
						idx = i
						break
					}
				}
				if idx < 0 {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				m.Targets = append(m.Targets[:idx], m.Targets[idx+1:]...)
				report.MutatedNames = append(report.MutatedNames, name)
			}
		case ResourceRegistries:
			for _, name := range uniqueNames {
				idx := -1
				for i, r := range m.Registries {
					if r.Name == name {
						idx = i
						break
					}
				}
				if idx < 0 {
					appendUniqueName(&report.SkippedMissingNames, name)
					continue
				}
				m.Registries = append(m.Registries[:idx], m.Registries[idx+1:]...)
				report.MutatedNames = append(report.MutatedNames, name)
			}
		}

		return report, manifest.SaveManifest(m, manifestPath)
	default:
		return report, fmt.Errorf("unknown resource type %q", kind)
	}
}

func uniqueRequestNames(names []string) ([]string, []string) {
	seen := make(map[string]bool, len(names))
	unique := make([]string, 0, len(names))
	duplicates := make([]string, 0)
	for _, name := range names {
		if name == "" {
			continue
		}
		if seen[name] {
			appendUniqueName(&duplicates, name)
			continue
		}
		seen[name] = true
		unique = append(unique, name)
	}
	return unique, duplicates
}

func appendUniqueName(names *[]string, name string) {
	for _, existing := range *names {
		if existing == name {
			return
		}
	}
	*names = append(*names, name)
}

func ensureRegistryRefInManifest(m *manifest.Manifest, merged *manifest.Manifest, registryName string) {
	if m == nil || registryName == "" {
		return
	}
	for _, r := range m.Registries {
		if r.Name == registryName {
			return
		}
	}
	if merged != nil {
		for _, r := range merged.Registries {
			if r.Name == registryName {
				m.Registries = append(m.Registries, normalizeRegistryRefPaths(r))
				return
			}
		}
	}
}

func normalizeRegistryRefPaths(r manifest.RegistryRef) manifest.RegistryRef {
	if r.Paths == nil {
		r.Paths = map[string]string{}
	}
	root := r.Paths["skills"]
	if root == "" {
		root = "skills/"
		r.Paths["skills"] = root
	}
	if r.Paths["instructions"] == "" {
		r.Paths["instructions"] = "instructions/"
	}
	if r.Paths["agents"] == "" {
		r.Paths["agents"] = "agents/"
	}
	if r.Paths["prompts"] == "" {
		r.Paths["prompts"] = "prompts/"
	}
	return r
}

func countLocalRegistryRefs(local *manifest.Manifest, registryName string) int {
	if local == nil || registryName == "" {
		return 0
	}
	count := 0
	for _, s := range local.Skills {
		if s.Registry == registryName {
			count++
		}
	}
	for _, i := range local.Instructions {
		if i.Registry == registryName {
			count++
		}
	}
	for _, a := range local.Agents {
		if a.Registry == registryName {
			count++
		}
	}
	for _, p := range local.Prompts {
		if p.Registry == registryName {
			count++
		}
	}
	return count
}
