package cli

import (
	"fmt"
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

// InstallResourceItems installs resources by type without interactive prompts.
func InstallResourceItems(projectDir, globalPath, kind string, names []string) error {
	_, err := InstallResourceItemsWithReport(projectDir, globalPath, kind, names)
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
		_, manifestPath, findErr := manifest.LoadManifestFromProject(projectDir)
		if findErr != nil {
			manifestPath = filepath.Join(projectDir, "vibes.yaml")
		}

		merged, _ := manifest.LoadMergedManifest(projectDir, globalPath)
		installer := engine.NewInstaller(buildAllSources(merged))

		var errs []string
		for _, name := range uniqueNames {
			if err := installer.Install(name, manifestPath); err != nil {
				if strings.Contains(err.Error(), "already in manifest") {
					appendUniqueName(&report.SkippedDuplicateNames, name)
					continue
				}
				errs = append(errs, err.Error())
				continue
			}
			report.MutatedNames = append(report.MutatedNames, name)
		}
		if len(errs) > 0 {
			return report, fmt.Errorf(strings.Join(errs, "; "))
		}
		return report, nil
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
			for _, name := range uniqueNames {
				if existing[name] {
					appendUniqueName(&report.SkippedDuplicateNames, name)
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
				} else {
					i.Path = fmt.Sprintf("./instructions/%s.md", name)
				}
				m.Instructions = append(m.Instructions, i)
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
			return report, fmt.Errorf(strings.Join(errs, "; "))
		}
		return report, nil
	case ResourceAgents, ResourceInstructions:
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
