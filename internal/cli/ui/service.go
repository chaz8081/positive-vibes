package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

const (
	resourceKindSkills       = "skills"
	resourceKindAgents       = "agents"
	resourceKindInstructions = "instructions"
)

type ResourceDetail struct {
	Kind        string
	Name        string
	Installed   bool
	Registry    string
	RegistryURL string
	Path        string
	Payload     any
}

type ResourceRow struct {
	Name      string
	Installed bool
}

type ResourceServiceBridge struct {
	ListAvailableRows func(projectDir, globalPath, kind string) ([]ResourceRow, error)
	ListInstalledRows func(projectDir, globalPath, kind string) ([]ResourceRow, error)
	ShowResource      func(projectDir, globalPath, kind, name string) (ResourceDetail, error)
	MergeRows         func(available, installed []ResourceRow) []ResourceRow
}

var resourceServiceBridge = ResourceServiceBridge{
	ListAvailableRows: listAvailableRows,
	ListInstalledRows: listInstalledRows,
	ShowResource:      showResourceDetail,
	MergeRows:         mergeResourceRows,
}

func ConfigureResourceServiceBridge(bridge ResourceServiceBridge) {
	if bridge.ListAvailableRows != nil {
		resourceServiceBridge.ListAvailableRows = bridge.ListAvailableRows
	}
	if bridge.ListInstalledRows != nil {
		resourceServiceBridge.ListInstalledRows = bridge.ListInstalledRows
	}
	if bridge.ShowResource != nil {
		resourceServiceBridge.ShowResource = bridge.ShowResource
	}
	if bridge.MergeRows != nil {
		resourceServiceBridge.MergeRows = bridge.MergeRows
	}
}

type Service struct {
	deps serviceDeps
}

type serviceDeps struct {
	listAvailable func(kind string) ([]ResourceRow, error)
	listInstalled func(kind string) ([]ResourceRow, error)
	showDetail    func(kind, name string) (ResourceDetail, error)
	install       func(kind string, names []string) error
	remove        func(kind string, names []string) error
}

func NewService(projectDir string) *Service {
	if projectDir == "" {
		projectDir = "."
	}
	globalPath := defaultGlobalManifestPath()

	return newServiceWithDeps(serviceDeps{
		listAvailable: func(kind string) ([]ResourceRow, error) {
			return resourceServiceBridge.ListAvailableRows(projectDir, globalPath, kind)
		},
		listInstalled: func(kind string) ([]ResourceRow, error) {
			return resourceServiceBridge.ListInstalledRows(projectDir, globalPath, kind)
		},
		showDetail: func(kind, name string) (ResourceDetail, error) {
			return resourceServiceBridge.ShowResource(projectDir, globalPath, kind, name)
		},
		install: func(kind string, names []string) error {
			return installResources(projectDir, globalPath, kind, names)
		},
		remove: func(kind string, names []string) error {
			return removeResources(projectDir, kind, names)
		},
	})
}

func newServiceWithDeps(deps serviceDeps) *Service {
	return &Service{deps: deps}
}

func (s *Service) ListResources(kind string) ([]ResourceRow, error) {
	if err := validateKind(kind); err != nil {
		return nil, err
	}

	available, err := s.deps.listAvailable(kind)
	if err != nil {
		return nil, err
	}
	installed, err := s.deps.listInstalled(kind)
	if err != nil {
		return nil, err
	}

	rows := resourceServiceBridge.MergeRows(available, installed)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	return rows, nil
}

func mergeResourceRows(available, installed []ResourceRow) []ResourceRow {
	byName := make(map[string]ResourceRow, len(available)+len(installed))
	for _, row := range available {
		if row.Name == "" {
			continue
		}
		byName[row.Name] = ResourceRow{Name: row.Name, Installed: false}
	}
	for _, row := range installed {
		if row.Name == "" {
			continue
		}
		byName[row.Name] = ResourceRow{Name: row.Name, Installed: true}
	}
	rows := make([]ResourceRow, 0, len(byName))
	for _, row := range byName {
		rows = append(rows, row)
	}
	return rows
}

func (s *Service) ShowResource(kind, name string) (ResourceDetail, error) {
	if err := validateKind(kind); err != nil {
		return ResourceDetail{}, err
	}
	if name == "" {
		return ResourceDetail{}, fmt.Errorf("resource name is required")
	}
	return s.deps.showDetail(kind, name)
}

func (s *Service) InstallResources(kind string, names []string) error {
	if err := validateKind(kind); err != nil {
		return err
	}
	return s.deps.install(kind, dedup(names))
}

func (s *Service) RemoveResources(kind string, names []string) error {
	if err := validateKind(kind); err != nil {
		return err
	}
	return s.deps.remove(kind, dedup(names))
}

func validateKind(kind string) error {
	switch kind {
	case resourceKindSkills, resourceKindAgents, resourceKindInstructions:
		return nil
	default:
		return fmt.Errorf("unknown resource type %q (valid: skills, agents, instructions)", kind)
	}
}

func listAvailableRows(projectDir, globalPath, kind string) ([]ResourceRow, error) {
	merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)

	switch kind {
	case resourceKindSkills:
		seen := make(map[string]bool)
		var rows []ResourceRow
		for _, src := range buildAllSources(merged) {
			names, err := src.List()
			if err != nil {
				continue
			}
			for _, name := range names {
				if seen[name] {
					continue
				}
				seen[name] = true
				rows = append(rows, ResourceRow{Name: name})
			}
		}
		return rows, nil
	case resourceKindAgents, resourceKindInstructions:
		refs := collectRegistryResourceItems(merged, kind)
		rows := make([]ResourceRow, 0, len(refs))
		for _, ref := range refs {
			rows = append(rows, ResourceRow{Name: ref.Name})
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unknown resource type %q", kind)
	}
}

func listInstalledRows(projectDir, globalPath, kind string) ([]ResourceRow, error) {
	merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
	if merged == nil {
		return nil, nil
	}

	switch kind {
	case resourceKindSkills:
		rows := make([]ResourceRow, 0, len(merged.Skills))
		for _, s := range merged.Skills {
			rows = append(rows, ResourceRow{Name: s.Name, Installed: true})
		}
		return rows, nil
	case resourceKindAgents:
		rows := make([]ResourceRow, 0, len(merged.Agents))
		for _, a := range merged.Agents {
			rows = append(rows, ResourceRow{Name: a.Name, Installed: true})
		}
		return rows, nil
	case resourceKindInstructions:
		rows := make([]ResourceRow, 0, len(merged.Instructions))
		for _, i := range merged.Instructions {
			rows = append(rows, ResourceRow{Name: i.Name, Installed: true})
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unknown resource type %q", kind)
	}
}

func showResourceDetail(projectDir, globalPath, kind, name string) (ResourceDetail, error) {
	merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)

	switch kind {
	case resourceKindSkills:
		skill, regName, err := resolveSkillFromSources(name, buildAllSources(merged))
		if err != nil {
			return ResourceDetail{}, err
		}
		regURL := registryURLByName(merged, regName)
		return ResourceDetail{
			Kind:        kind,
			Name:        name,
			Installed:   hasInstalledSkill(merged, name),
			Registry:    regName,
			RegistryURL: regURL,
			Payload:     skill,
		}, nil
	case resourceKindAgents:
		if merged != nil {
			for _, a := range merged.Agents {
				if a.Name == name {
					return ResourceDetail{
						Kind:      kind,
						Name:      name,
						Installed: true,
						Registry:  a.Registry,
						Path:      a.Path,
						Payload:   a,
					}, nil
				}
			}
		}
		for _, ref := range collectRegistryResourceItems(merged, kind) {
			if ref.Name == name {
				payload := manifest.AgentRef{Name: name, Registry: ref.Registry, Path: ref.Path}
				return ResourceDetail{Kind: kind, Name: name, Installed: false, Registry: ref.Registry, Path: ref.Path, Payload: payload}, nil
			}
		}
		return ResourceDetail{}, fmt.Errorf("agent not found: %s", name)
	case resourceKindInstructions:
		if merged != nil {
			for _, inst := range merged.Instructions {
				if inst.Name == name {
					return ResourceDetail{
						Kind:      kind,
						Name:      name,
						Installed: true,
						Registry:  inst.Registry,
						Path:      inst.Path,
						Payload:   inst,
					}, nil
				}
			}
		}
		for _, ref := range collectRegistryResourceItems(merged, kind) {
			if ref.Name == name {
				payload := manifest.InstructionRef{Name: name, Registry: ref.Registry, Path: ref.Path}
				return ResourceDetail{Kind: kind, Name: name, Installed: false, Registry: ref.Registry, Path: ref.Path, Payload: payload}, nil
			}
		}
		return ResourceDetail{}, fmt.Errorf("instruction not found: %s", name)
	default:
		return ResourceDetail{}, fmt.Errorf("unknown resource type %q", kind)
	}
}

func installResources(projectDir, globalPath, kind string, names []string) error {
	if len(names) == 0 {
		return nil
	}

	switch kind {
	case resourceKindSkills:
		_, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			manifestPath = filepath.Join(projectDir, "vibes.yaml")
		}

		merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
		installer := engine.NewInstaller(buildAllSources(merged))

		var errs []string
		for _, name := range names {
			if err := installer.Install(name, manifestPath); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf(strings.Join(errs, "; "))
		}
		return nil
	case resourceKindAgents, resourceKindInstructions:
		m, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			manifestPath = filepath.Join(projectDir, "vibes.yaml")
			m = &manifest.Manifest{}
		}

		merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
		availableRefs := collectRegistryResourceItems(merged, kind)
		availableByName := make(map[string]registryResourceItem, len(availableRefs))
		for _, ref := range availableRefs {
			availableByName[ref.Name] = ref
		}

		switch kind {
		case resourceKindAgents:
			existing := make(map[string]bool)
			for _, a := range m.Agents {
				existing[a.Name] = true
			}
			for _, name := range names {
				if existing[name] {
					continue
				}
				a := manifest.AgentRef{Name: name}
				if ref, ok := availableByName[name]; ok {
					a.Registry = ref.Registry
					a.Path = ref.Path
				} else {
					a.Path = fmt.Sprintf("./agents/%s.md", name)
				}
				m.Agents = append(m.Agents, a)
				existing[name] = true
			}
		case resourceKindInstructions:
			existing := make(map[string]bool)
			for _, i := range m.Instructions {
				existing[i.Name] = true
			}
			for _, name := range names {
				if existing[name] {
					continue
				}
				i := manifest.InstructionRef{Name: name}
				if ref, ok := availableByName[name]; ok {
					i.Registry = ref.Registry
					i.Path = ref.Path
				} else {
					i.Path = fmt.Sprintf("./instructions/%s.md", name)
				}
				m.Instructions = append(m.Instructions, i)
				existing[name] = true
			}
		}

		return manifest.SaveManifest(m, manifestPath)
	default:
		return fmt.Errorf("unknown resource type %q", kind)
	}
}

func removeResources(projectDir, kind string, names []string) error {
	if len(names) == 0 {
		return nil
	}

	switch kind {
	case resourceKindSkills:
		_, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			return fmt.Errorf("no manifest found in %s", projectDir)
		}

		installer := engine.NewInstaller(nil)
		var errs []string
		for _, name := range names {
			if err := installer.Remove(name, manifestPath); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf(strings.Join(errs, "; "))
		}
		return nil
	case resourceKindAgents, resourceKindInstructions:
		m, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			return fmt.Errorf("no manifest found in %s", projectDir)
		}

		switch kind {
		case resourceKindAgents:
			for _, name := range names {
				idx := -1
				for i, a := range m.Agents {
					if a.Name == name {
						idx = i
						break
					}
				}
				if idx >= 0 {
					m.Agents = append(m.Agents[:idx], m.Agents[idx+1:]...)
				}
			}
		case resourceKindInstructions:
			for _, name := range names {
				idx := -1
				for i, inst := range m.Instructions {
					if inst.Name == name {
						idx = i
						break
					}
				}
				if idx >= 0 {
					m.Instructions = append(m.Instructions[:idx], m.Instructions[idx+1:]...)
				}
			}
		}

		return manifest.SaveManifest(m, manifestPath)
	default:
		return fmt.Errorf("unknown resource type %q", kind)
	}
}

func buildAllSources(merged *manifest.Manifest) []registry.SkillSource {
	sources := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	if merged != nil {
		sources = append(sources, gitRegistriesFromManifest(merged)...)
	}
	return sources
}

func resolveSkillFromSources(name string, sources []registry.SkillSource) (*schema.Skill, string, error) {
	for _, src := range sources {
		skill, _, err := src.Fetch(name)
		if err == nil {
			return skill, src.Name(), nil
		}
	}
	return nil, "", fmt.Errorf("skill not found: %s", name)
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

func collectRegistryResourceItems(merged *manifest.Manifest, kind string) []registryResourceItem {
	if merged == nil {
		return nil
	}
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
			name := resourceNameFromPath(kind, rel)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, registryResourceItem{Name: name, Registry: src.Name(), Path: rel})
		}
	}
	return items
}

type registryResourceItem struct {
	Name     string
	Registry string
	Path     string
}

func resourceNameFromPath(kind, relPath string) string {
	base := filepath.Base(relPath)
	switch kind {
	case resourceKindInstructions:
		if !strings.HasSuffix(base, ".instructions.md") {
			return ""
		}
		return strings.TrimSuffix(base, ".instructions.md")
	case resourceKindAgents:
		if !strings.HasSuffix(base, ".agent.md") {
			return ""
		}
		return strings.TrimSuffix(base, ".agent.md")
	default:
		if !strings.HasSuffix(base, ".md") {
			return ""
		}
		return strings.TrimSuffix(base, ".md")
	}
}

func gitRegistriesFromManifest(m *manifest.Manifest) []registry.SkillSource {
	if m == nil {
		return nil
	}
	var sources []registry.SkillSource
	for _, r := range m.Registries {
		sources = append(sources, &registry.GitRegistry{
			RegistryName:     r.Name,
			URL:              r.URL,
			CachePath:        defaultCachePath(r.Name),
			SkillsPath:       r.SkillsPath(),
			InstructionsPath: r.InstructionsPath(),
			AgentsPath:       r.AgentsPath(),
			Ref:              r.Ref,
		})
	}
	return sources
}

func defaultCachePath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".positive-vibes", "cache", name)
}

func defaultGlobalManifestPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "positive-vibes", "vibes.yaml")
}

func dedup(names []string) []string {
	seen := make(map[string]bool, len(names))
	result := make([]string, 0, len(names))
	for _, n := range names {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		result = append(result, n)
	}
	return result
}
