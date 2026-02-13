package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <resource-type> [name...]",
	Short: "Remove resources from the manifest",
	Long: `Remove one or more resources from your manifest.

If no names are given, an interactive picker is shown.

Resource types: skills, agents, instructions

Examples:
  positive-vibes remove skills                      # interactive picker
  positive-vibes remove skills code-review           # remove by name
  positive-vibes remove skills code-review tdd       # remove multiple`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: makeValidArgsFunction("installed"),
	Run: func(cmd *cobra.Command, args []string) {
		resType, err := ParseResourceType(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		names := dedup(args[1:])

		switch resType {
		case ResourceSkills:
			removeSkillsRun(names)
		case ResourceAgents:
			removeAgentsRun(names)
		case ResourceInstructions:
			removeInstructionsRun(names)
		}
	},
}

// RemoveResourcesCommandAction applies remove mutations for command flows.
func RemoveResourcesCommandAction(projectDir, kind string, names []string) error {
	return RemoveResourceItems(projectDir, kind, names)
}

func removeSkillsRun(names []string) {
	project := ProjectDir()

	_, manifestPath, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		fmt.Fprintf(os.Stderr, "error: no manifest found in %s\n", project)
		return
	}

	// If no names, show interactive picker
	if len(names) == 0 {
		globalPath := defaultGlobalManifestPath()
		merged, _ := manifest.LoadMergedManifest(project, globalPath)
		installed := collectInstalledSkills(merged)

		if len(installed) == 0 {
			fmt.Println("No skills installed to remove.")
			return
		}

		var options []huh.Option[string]
		for _, item := range installed {
			options = append(options, huh.NewOption(item.Name, item.Name))
		}

		var selected []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select skills to remove").
					Description("Use space to toggle, enter to confirm").
					Options(options...).
					Value(&selected),
			),
		)

		err := form.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if len(selected) == 0 {
			fmt.Println("No skills selected.")
			return
		}

		names = selected
	}

	if err := RemoveResourcesCommandAction(project, string(ResourceSkills), names); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	for _, name := range dedup(names) {
		fmt.Printf("Removed '%s' from %s\n", name, filepath.Base(manifestPath))
	}
}

func removeAgentsRun(names []string) {
	project := ProjectDir()

	m, _, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		fmt.Fprintf(os.Stderr, "error: no manifest found in %s\n", project)
		return
	}

	// If no names, show interactive picker
	if len(names) == 0 {
		if len(m.Agents) == 0 {
			fmt.Println("No agents configured to remove.")
			return
		}

		var options []huh.Option[string]
		for _, a := range m.Agents {
			options = append(options, huh.NewOption(a.Name, a.Name))
		}

		var selected []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select agents to remove").
					Description("Use space to toggle, enter to confirm").
					Options(options...).
					Value(&selected),
			),
		)

		err := form.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if len(selected) == 0 {
			fmt.Println("No agents selected.")
			return
		}

		names = selected
	}

	if err := RemoveResourcesCommandAction(project, string(ResourceAgents), names); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	for _, name := range dedup(names) {
		fmt.Printf("Removed agent '%s'\n", name)
	}
}

func removeInstructionsRun(names []string) {
	project := ProjectDir()

	m, _, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		fmt.Fprintf(os.Stderr, "error: no manifest found in %s\n", project)
		return
	}

	// If no names, show interactive picker
	if len(names) == 0 {
		if len(m.Instructions) == 0 {
			fmt.Println("No instructions configured to remove.")
			return
		}

		var options []huh.Option[string]
		for _, inst := range m.Instructions {
			options = append(options, huh.NewOption(inst.Name, inst.Name))
		}

		var selected []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select instructions to remove").
					Description("Use space to toggle, enter to confirm").
					Options(options...).
					Value(&selected),
			),
		)

		err := form.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if len(selected) == 0 {
			fmt.Println("No instructions selected.")
			return
		}

		names = selected
	}

	if err := RemoveResourcesCommandAction(project, string(ResourceInstructions), names); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	for _, name := range dedup(names) {
		fmt.Printf("Removed instruction '%s'\n", name)
	}
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
