package cli

import (
	"sort"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
)

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
	case ResourcePrompts:
		return completePromptNames(merged, mode)
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

func completePromptNames(merged *manifest.Manifest, mode string) []string {
	switch mode {
	case "available":
		items := collectAvailablePrompts(merged)
		var names []string
		for _, item := range items {
			if !item.Installed {
				names = append(names, item.Name)
			}
		}
		return names
	case "installed":
		return resourceNamesFromItems(collectPrompts(merged))
	default:
		names := resourceNamesFromItems(collectAvailablePrompts(merged))
		for _, n := range resourceNamesFromItems(collectPrompts(merged)) {
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
