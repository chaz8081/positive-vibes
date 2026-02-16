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

Resource types: skills, agents, instructions, prompts

Examples:
  positive-vibes remove skills                      # interactive picker
  positive-vibes remove skills code-review           # remove by name
  positive-vibes remove skills code-review tdd       # remove multiple`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: makeValidArgsFunction("installed"),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		resType, err := ParseResourceType(args[0])
		if err != nil {
			return err
		}

		names := args[1:]

		switch resType {
		case ResourceSkills:
			return removeSkillsRun(cmd, names)
		case ResourceAgents:
			return removeAgentsRun(cmd, names)
		case ResourceInstructions:
			return removeInstructionsRun(cmd, names)
		case ResourcePrompts:
			return removePromptsRun(cmd, names)
		}
		return nil
	},
}

var removeResourcesCommandAction = RemoveResourcesCommandAction

// RemoveResourcesCommandAction applies remove mutations for command flows.
func RemoveResourcesCommandAction(projectDir, kind string, names []string) (ResourceMutationReport, error) {
	return RemoveResourceItemsWithReport(projectDir, kind, names)
}

func removeSkillsRun(cmd *cobra.Command, names []string) error {
	project := ProjectDir()

	_, manifestPath, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		err := fmt.Errorf("no manifest found in %s", project)
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}

	// If no names, show interactive picker
	if len(names) == 0 {
		globalPath := defaultGlobalManifestPath()
		merged, _ := manifest.LoadMergedManifest(project, globalPath)
		installed := collectInstalledSkills(merged)

		if len(installed) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No skills installed to remove.")
			return nil
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
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return err
		}

		if len(selected) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No skills selected.")
			return nil
		}

		names = selected
	}

	report, err := removeResourcesCommandAction(project, string(ResourceSkills), names)
	for _, name := range report.MutatedNames {
		fmt.Fprintf(cmd.OutOrStdout(), "Removed '%s' from %s\n", name, filepath.Base(manifestPath))
	}
	for _, name := range report.SkippedMissingNames {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: skill not found in manifest: %s\n", name)
	}
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}
	return nil
}

func removeAgentsRun(cmd *cobra.Command, names []string) error {
	project := ProjectDir()

	m, _, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		err := fmt.Errorf("no manifest found in %s", project)
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}

	// If no names, show interactive picker
	if len(names) == 0 {
		if len(m.Agents) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No agents configured to remove.")
			return nil
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
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return err
		}

		if len(selected) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No agents selected.")
			return nil
		}

		names = selected
	}

	report, err := removeResourcesCommandAction(project, string(ResourceAgents), names)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}
	for _, name := range report.MutatedNames {
		fmt.Fprintf(cmd.OutOrStdout(), "Removed agent '%s'\n", name)
	}
	for _, name := range report.SkippedMissingNames {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: agent not found in manifest: %s\n", name)
	}
	return nil
}

func removeInstructionsRun(cmd *cobra.Command, names []string) error {
	project := ProjectDir()

	m, _, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		err := fmt.Errorf("no manifest found in %s", project)
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}

	// If no names, show interactive picker
	if len(names) == 0 {
		if len(m.Instructions) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No instructions configured to remove.")
			return nil
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
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return err
		}

		if len(selected) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No instructions selected.")
			return nil
		}

		names = selected
	}

	report, err := removeResourcesCommandAction(project, string(ResourceInstructions), names)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}
	for _, name := range report.MutatedNames {
		fmt.Fprintf(cmd.OutOrStdout(), "Removed instruction '%s'\n", name)
	}
	for _, name := range report.SkippedMissingNames {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: instruction not found in manifest: %s\n", name)
	}
	return nil
}

func removePromptsRun(cmd *cobra.Command, names []string) error {
	project := ProjectDir()

	m, _, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		err := fmt.Errorf("no manifest found in %s", project)
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}

	if len(names) == 0 {
		if len(m.Prompts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No prompts configured to remove.")
			return nil
		}

		var options []huh.Option[string]
		for _, p := range m.Prompts {
			options = append(options, huh.NewOption(p.Name, p.Name))
		}

		var selected []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select prompts to remove").
					Description("Use space to toggle, enter to confirm").
					Options(options...).
					Value(&selected),
			),
		)

		err := form.Run()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			return err
		}

		if len(selected) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No prompts selected.")
			return nil
		}

		names = selected
	}

	report, err := removeResourcesCommandAction(project, string(ResourcePrompts), names)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
		return err
	}
	for _, name := range report.MutatedNames {
		fmt.Fprintf(cmd.OutOrStdout(), "Removed prompt '%s'\n", name)
	}
	for _, name := range report.SkippedMissingNames {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: prompt not found in manifest: %s\n", name)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
