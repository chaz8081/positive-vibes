package cli

import (
	"fmt"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
)

// --- Resource item abstraction ---

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
	case ResourcePrompts:
		return collectAvailablePrompts(merged), nil
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
	var global *manifest.Manifest
	if globalPath != "" {
		global, _ = manifest.LoadManifest(globalPath)
	}

	switch resType {
	case ResourceSkills:
		return collectInstalledSkillsWithScope(local, global, merged), nil
	case ResourceAgents:
		return collectAgentsWithScope(local, global, merged), nil
	case ResourceInstructions:
		return collectInstructionsWithScope(local, global, merged), nil
	case ResourcePrompts:
		return collectPromptsWithScope(local, global, merged), nil
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
	local, _, _ := manifest.LoadManifestFromProject(projectDir)
	var global *manifest.Manifest
	if globalPath != "" {
		global, _ = manifest.LoadManifest(globalPath)
	}

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
	case ResourcePrompts:
		if merged != nil {
			for _, p := range merged.Prompts {
				if p.Name == name {
					content := loadResourceContent(projectDir, merged, string(resType), p.Registry, p.Path)
					return ResourceDetailResult{
						Kind:      resType,
						Name:      name,
						Installed: true,
						Registry:  p.Registry,
						Path:      p.Path,
						Payload: map[string]any{
							"name":        p.Name,
							"path":        p.Path,
							"registry":    p.Registry,
							"description": "Prompt template",
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
						"description": "Prompt template",
						"content":     content,
					},
				}, nil
			}
		}
		return ResourceDetailResult{}, fmt.Errorf("prompt not found: %s", name)
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
				state, reason := registrySourceState(name, local, global)
				return ResourceDetailResult{
					Kind:      resType,
					Name:      name,
					Installed: true,
					Registry:  reg.Name,
					Payload: map[string]any{
						"name":          reg.Name,
						"description":   "Registry source",
						"content":       formatRegistryDetailContent(reg),
						"source_state":  state,
						"source_reason": reason,
					},
				}, nil
			}
		}
		return ResourceDetailResult{}, fmt.Errorf("registry not found: %s", name)
	default:
		return ResourceDetailResult{}, fmt.Errorf("unknown resource type %q", kind)
	}
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

func collectPrompts(merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	items := make([]ResourceItem, 0, len(merged.Prompts))
	for _, p := range merged.Prompts {
		items = append(items, ResourceItem{Name: p.Name, Installed: true, InstallScope: installScopeLocal})
	}
	return items
}

func collectPromptsWithScope(local, global, merged *manifest.Manifest) []ResourceItem {
	if merged == nil {
		return nil
	}
	localSet := make(map[string]bool)
	if local != nil {
		for _, p := range local.Prompts {
			localSet[p.Name] = true
		}
	}
	globalSet := make(map[string]bool)
	if global != nil {
		for _, p := range global.Prompts {
			globalSet[p.Name] = true
		}
	}
	items := make([]ResourceItem, 0, len(merged.Prompts))
	for _, p := range merged.Prompts {
		items = append(items, ResourceItem{Name: p.Name, Installed: true, InstallScope: resolveInstallScope(localSet[p.Name], globalSet[p.Name])})
	}
	return items
}

func collectAvailablePrompts(merged *manifest.Manifest) []ResourceItem {
	installed := make(map[string]bool)
	if merged != nil {
		for _, p := range merged.Prompts {
			installed[p.Name] = true
		}
	}
	refs := collectRegistryResourceItems(merged, ResourcePrompts)
	items := make([]ResourceItem, 0, len(refs))
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
		state, _ := registrySourceState(r.Name, local, global)
		rowState := ""
		if state == "local_issue" || state == "local_conflict" {
			rowState = "registry_attention"
		}
		items = append(items, ResourceItem{Name: r.Name, Installed: true, InstallScope: resolveInstallScope(localSet[r.Name], globalSet[r.Name]), State: rowState})
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
