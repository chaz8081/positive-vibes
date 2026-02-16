package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

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

// formatPromptShow renders a prompt's details.
func formatPromptShow(prompt manifest.PromptRef, installed bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\n", prompt.Name)
	if prompt.Path != "" {
		fmt.Fprintf(&b, "Path: %s\n", prompt.Path)
	}
	if prompt.Registry != "" {
		fmt.Fprintf(&b, "Registry: %s\n", prompt.Registry)
	}
	if installed {
		b.WriteString("Status: installed\n")
	} else {
		b.WriteString("Status: available\n")
	}
	return b.String()
}
