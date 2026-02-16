package cli

import (
	"fmt"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
)

var (
	listRegistry      string
	listInstalledOnly bool
	listJSON          bool
)

var listCmd = &cobra.Command{
	Use:   "list <resource-type>",
	Short: "List available resources",
	Long: `List available resources of a given type.

Resource types: skills, agents, instructions, prompts, targets, registries

Examples:
  positive-vibes list skills
  positive-vibes list skills --registry=embedded
  positive-vibes list skills --installed-only
  positive-vibes list skills --json
  positive-vibes list agents
  positive-vibes list instructions
  positive-vibes list targets
  positive-vibes list registries`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: makeValidArgsFunction(""),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		resType, err := ParseResourceType(args[0])
		if err != nil {
			return err
		}

		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()
		merged, err := manifest.LoadMergedManifest(project, globalPath)
		if err != nil {
			return err
		}

		switch resType {
		case ResourceSkills:
			listSkillsRun(merged)
		case ResourceAgents:
			listAgentsRun(merged)
		case ResourceInstructions:
			listInstructionsRun(merged)
		case ResourcePrompts:
			listPromptsRun(merged)
		case ResourceTargets:
			listTargetsRun(merged)
		case ResourceRegistries:
			listRegistriesRun(merged)
		}
		return nil
	},
}

func listSkillsRun(merged *manifest.Manifest) {
	sets := collectSkillSets(merged)
	installed := buildInstalledSkillsMap(merged)

	if listJSON {
		fmt.Println(formatSkillsListJSON(sets, installed))
		return
	}

	opts := listFormatOptions{
		Registry:      listRegistry,
		InstalledOnly: listInstalledOnly,
	}
	fmt.Print(formatSkillsListFiltered(sets, installed, opts))
}

func listAgentsRun(merged *manifest.Manifest) {
	items := collectAvailableAgents(merged)

	if listJSON {
		fmt.Println(formatResourceListJSON(ResourceAgents, items))
		return
	}

	fmt.Print(formatResourceList(ResourceAgents, items))
}

func listInstructionsRun(merged *manifest.Manifest) {
	items := collectAvailableInstructions(merged)

	if listJSON {
		fmt.Println(formatResourceListJSON(ResourceInstructions, items))
		return
	}

	fmt.Print(formatResourceList(ResourceInstructions, items))
}

func listPromptsRun(merged *manifest.Manifest) {
	items := collectAvailablePrompts(merged)

	if listJSON {
		fmt.Println(formatResourceListJSON(ResourcePrompts, items))
		return
	}

	fmt.Print(formatResourceList(ResourcePrompts, items))
}

func listTargetsRun(merged *manifest.Manifest) {
	items := collectAvailableTargets(merged)

	if listJSON {
		fmt.Println(formatResourceListJSON(ResourceTargets, items))
		return
	}

	fmt.Print(formatResourceList(ResourceTargets, items))
}

func listRegistriesRun(merged *manifest.Manifest) {
	items := collectAvailableRegistries(merged)

	if listJSON {
		fmt.Println(formatResourceListJSON(ResourceRegistries, items))
		return
	}

	fmt.Print(formatResourceList(ResourceRegistries, items))
}

func init() {
	listCmd.Flags().StringVar(&listRegistry, "registry", "", "filter by registry name (skills only)")
	listCmd.Flags().BoolVar(&listInstalledOnly, "installed-only", false, "show only installed resources (skills only)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}
