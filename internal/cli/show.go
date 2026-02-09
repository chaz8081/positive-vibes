package cli

import (
	"fmt"
	"os"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <resource-type> <name>",
	Short: "Show details for a specific resource",
	Long: `Display detailed information about a resource.

Resource types: skills, agents, instructions

Examples:
  positive-vibes show skills code-review
  positive-vibes show agents reviewer
  positive-vibes show instructions coding-standards`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: makeValidArgsFunction("all"),
	Run: func(cmd *cobra.Command, args []string) {
		resType, err := ParseResourceType(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		name := args[1]

		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()
		merged, _ := manifest.LoadMergedManifest(project, globalPath)

		switch resType {
		case ResourceSkills:
			showSkillRun(name, merged)
		case ResourceAgents:
			showAgentRun(name, merged)
		case ResourceInstructions:
			showInstructionRun(name, merged)
		}
	},
}

func showSkillRun(name string, merged *manifest.Manifest) {
	sources := buildAllSources(merged)
	skill, regName, err := resolveSkillFromSources(name, sources)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}

	var regURL string
	if merged != nil {
		for _, r := range merged.Registries {
			if r.Name == regName {
				regURL = r.URL
				break
			}
		}
	}

	installed := false
	if merged != nil {
		for _, s := range merged.Skills {
			if s.Name == name {
				installed = true
				break
			}
		}
	}

	fmt.Print(formatSkillShow(skill, regName, regURL, installed))
}

func showAgentRun(name string, merged *manifest.Manifest) {
	if merged == nil {
		fmt.Fprintln(os.Stderr, "error: no manifest found")
		return
	}
	for _, a := range merged.Agents {
		if a.Name == name {
			fmt.Print(formatAgentShow(a))
			return
		}
	}
	fmt.Fprintf(os.Stderr, "error: agent not found: %s\n", name)
}

func showInstructionRun(name string, merged *manifest.Manifest) {
	if merged == nil {
		fmt.Fprintln(os.Stderr, "error: no manifest found")
		return
	}
	for _, inst := range merged.Instructions {
		if inst.Name == name {
			fmt.Print(formatInstructionShow(inst))
			return
		}
	}
	fmt.Fprintf(os.Stderr, "error: instruction not found: %s\n", name)
}

func init() {
	rootCmd.AddCommand(showCmd)
}
