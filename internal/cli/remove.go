package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/chaz8081/positive-vibes/internal/engine"
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

	inst := engine.NewInstaller(nil)
	for _, name := range names {
		if err := inst.Remove(name, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error removing '%s': %v\n", name, err)
			continue
		}
		fmt.Printf("Removed '%s' from %s\n", name, filepath.Base(manifestPath))
	}
}

func removeAgentsRun(names []string) {
	project := ProjectDir()

	m, manifestPath, findErr := manifest.LoadManifestFromProject(project)
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

	for _, name := range names {
		found := -1
		for i, a := range m.Agents {
			if a.Name == name {
				found = i
				break
			}
		}
		if found < 0 {
			fmt.Fprintf(os.Stderr, "error: agent not found in manifest: %s\n", name)
			continue
		}
		m.Agents = append(m.Agents[:found], m.Agents[found+1:]...)
		fmt.Printf("Removed agent '%s'\n", name)
	}

	if err := manifest.SaveManifest(m, manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "error saving manifest: %v\n", err)
		return
	}
}

func removeInstructionsRun(names []string) {
	project := ProjectDir()

	m, manifestPath, findErr := manifest.LoadManifestFromProject(project)
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

	for _, name := range names {
		found := -1
		for i, inst := range m.Instructions {
			if inst.Name == name {
				found = i
				break
			}
		}
		if found < 0 {
			fmt.Fprintf(os.Stderr, "error: instruction not found in manifest: %s\n", name)
			continue
		}
		m.Instructions = append(m.Instructions[:found], m.Instructions[found+1:]...)
		fmt.Printf("Removed instruction '%s'\n", name)
	}

	if err := manifest.SaveManifest(m, manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "error saving manifest: %v\n", err)
		return
	}
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
