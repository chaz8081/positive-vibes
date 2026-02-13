package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
)

// InstallResourceItems installs resources by type without interactive prompts.
func InstallResourceItems(projectDir, globalPath, kind string, names []string) error {
	names = dedup(names)
	if len(names) == 0 {
		return nil
	}

	resType, err := ParseResourceType(kind)
	if err != nil {
		return err
	}

	switch resType {
	case ResourceSkills:
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
	case ResourceAgents, ResourceInstructions:
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
		case ResourceInstructions:
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

// RemoveResourceItems removes resources by type without interactive prompts.
func RemoveResourceItems(projectDir, kind string, names []string) error {
	names = dedup(names)
	if len(names) == 0 {
		return nil
	}

	resType, err := ParseResourceType(kind)
	if err != nil {
		return err
	}

	switch resType {
	case ResourceSkills:
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
	case ResourceAgents, ResourceInstructions:
		m, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			return fmt.Errorf("no manifest found in %s", projectDir)
		}

		switch resType {
		case ResourceAgents:
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
		case ResourceInstructions:
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
