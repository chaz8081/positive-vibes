package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <resource-type> [name...]",
	Short: "Install resources into the manifest",
	Long: `Install one or more resources into your manifest.

If no names are given, an interactive picker is shown.

Resource types: skills, agents, instructions

Examples:
  positive-vibes install skills                     # interactive picker
  positive-vibes install skills code-review          # install by name
  positive-vibes install skills code-review tdd      # install multiple
  positive-vibes install agents reviewer             # add agent by name
  positive-vibes install instructions standards      # add instruction by name`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: makeValidArgsFunction("available"),
	Run: func(cmd *cobra.Command, args []string) {
		resType, err := ParseResourceType(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		names := args[1:]

		switch resType {
		case ResourceSkills:
			installSkillsRun(names)
		case ResourceAgents:
			installAgentsRun(names)
		case ResourceInstructions:
			installInstructionsRun(names)
		}
	},
}

var installResourcesCommandAction = InstallResourcesCommandAction

// InstallResourcesCommandAction applies install mutations for command flows.
func InstallResourcesCommandAction(projectDir, globalPath, kind string, names []string) (ResourceMutationReport, error) {
	return InstallResourceItemsWithReport(projectDir, globalPath, kind, names)
}

func installSkillsRun(names []string) {
	project := ProjectDir()
	globalPath := defaultGlobalManifestPath()

	// Find existing manifest
	_, manifestPath, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		manifestPath = filepath.Join(project, "vibes.yaml")
	}

	// If no names provided, show interactive picker
	if len(names) == 0 {
		merged, _ := manifest.LoadMergedManifest(project, globalPath)
		available := collectAvailableSkills(merged)

		// Filter out already-installed skills
		var options []huh.Option[string]
		for _, item := range available {
			if !item.Installed {
				options = append(options, huh.NewOption(item.Name, item.Name))
			}
		}

		if len(options) == 0 {
			fmt.Println("All available skills are already installed.")
			return
		}

		var selected []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select skills to install").
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

	report, err := installResourcesCommandAction(project, globalPath, string(ResourceSkills), names)
	for _, name := range report.MutatedNames {
		fmt.Printf("Added '%s' to %s\n", name, filepath.Base(manifestPath))
	}
	for _, name := range report.SkippedDuplicateNames {
		fmt.Fprintf(os.Stderr, "warning: skill '%s' already exists in manifest, skipping\n", name)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}

	fmt.Println("\nRun 'positive-vibes apply' to install everywhere!")
}

func installAgentsRun(names []string) {
	project := ProjectDir()
	globalPath := defaultGlobalManifestPath()

	m, manifestPath, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		manifestPath = filepath.Join(project, "vibes.yaml")
		m = &manifest.Manifest{}
	}

	// Build a set of existing agent names for duplicate detection
	existing := make(map[string]bool)
	for _, a := range m.Agents {
		existing[a.Name] = true
	}

	merged, _ := manifest.LoadMergedManifest(project, globalPath)
	availableRefs := collectRegistryResourceItems(merged, ResourceAgents)
	availableByName := make(map[string]registryResourceItem, len(availableRefs))
	for _, ref := range availableRefs {
		availableByName[ref.Name] = ref
	}

	if len(names) == 0 {
		var options []huh.Option[string]
		for _, ref := range availableRefs {
			if !existing[ref.Name] {
				options = append(options, huh.NewOption(ref.Name, ref.Name))
			}
		}
		if len(options) > 0 {
			var selected []string
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Select agents to install").
						Description("Use space to toggle, enter to confirm").
						Options(options...).
						Value(&selected),
				),
			)
			if err := form.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
			if len(selected) == 0 {
				fmt.Println("No agents selected.")
				return
			}
			names = selected
		}
	}

	// If no names provided, prompt for agent details interactively
	if len(names) == 0 {
		var name, source, value, regName string

		nameInput := huh.NewInput().
			Title("Agent name").
			Description("A unique name for this agent").
			Value(&name)

		sourceSelect := huh.NewSelect[string]().
			Title("Agent source").
			Description("Where is the agent definition?").
			Options(
				huh.NewOption("Local path (file in this project)", "path"),
				huh.NewOption("Registry file (registry + path)", "registry"),
			).
			Value(&source)

		form := huh.NewForm(
			huh.NewGroup(nameInput, sourceSelect),
		)

		if err := form.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if name == "" {
			fmt.Fprintln(os.Stderr, "error: agent name is required")
			return
		}

		if existing[name] {
			fmt.Fprintf(os.Stderr, "error: agent '%s' already exists in manifest\n", name)
			return
		}

		var pathPrompt string
		if source == "path" {
			pathPrompt = "Path to agent file (e.g. ./agents/reviewer.md)"
		} else {
			pathPrompt = "Registry path (e.g. my-skill/agents/reviewer.md)"
		}

		if source == "registry" {
			regInput := huh.NewInput().
				Title("Registry name").
				Description("Must match a name in registries.").
				Value(&regName)
			regForm := huh.NewForm(huh.NewGroup(regInput))
			if err := regForm.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
			if regName == "" {
				fmt.Fprintln(os.Stderr, "error: registry name is required")
				return
			}
		}

		valueInput := huh.NewInput().
			Title(pathPrompt).
			Value(&value)

		valueForm := huh.NewForm(huh.NewGroup(valueInput))
		if err := valueForm.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if value == "" {
			fmt.Fprintln(os.Stderr, "error: agent source value is required")
			return
		}

		agent := manifest.AgentRef{Name: name}
		if source == "path" {
			agent.Path = value
		} else {
			agent.Registry = regName
			agent.Path = value
		}

		m.Agents = append(m.Agents, agent)
		if err := manifest.SaveManifest(m, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving manifest: %v\n", err)
			return
		}
		fmt.Printf("Added agent '%s' to %s\n", name, filepath.Base(manifestPath))
		fmt.Println("\nRun 'positive-vibes apply' to install everywhere!")
		return
	}

	report, err := installResourcesCommandAction(project, globalPath, string(ResourceAgents), names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}

	for _, name := range report.SkippedDuplicateNames {
		fmt.Fprintf(os.Stderr, "warning: agent '%s' already exists in manifest, skipping\n", name)
	}
	fmt.Printf("Saved %d agent(s) to %s\n", len(report.MutatedNames), filepath.Base(manifestPath))
	fmt.Println("Run 'positive-vibes apply' to install everywhere!")
}

func installInstructionsRun(names []string) {
	project := ProjectDir()
	globalPath := defaultGlobalManifestPath()

	m, manifestPath, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		manifestPath = filepath.Join(project, "vibes.yaml")
		m = &manifest.Manifest{}
	}

	// Build a set of existing instruction names for duplicate detection
	existing := make(map[string]bool)
	for _, inst := range m.Instructions {
		existing[inst.Name] = true
	}

	merged, _ := manifest.LoadMergedManifest(project, globalPath)
	availableRefs := collectRegistryResourceItems(merged, ResourceInstructions)
	availableByName := make(map[string]registryResourceItem, len(availableRefs))
	for _, ref := range availableRefs {
		availableByName[ref.Name] = ref
	}

	if len(names) == 0 {
		var options []huh.Option[string]
		for _, ref := range availableRefs {
			if !existing[ref.Name] {
				options = append(options, huh.NewOption(ref.Name, ref.Name))
			}
		}
		if len(options) > 0 {
			var selected []string
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Select instructions to install").
						Description("Use space to toggle, enter to confirm").
						Options(options...).
						Value(&selected),
				),
			)
			if err := form.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
			if len(selected) == 0 {
				fmt.Println("No instructions selected.")
				return
			}
			names = selected
		}
	}

	// If no names provided, prompt for instruction details interactively
	if len(names) == 0 {
		var name, source, value string

		nameInput := huh.NewInput().
			Title("Instruction name").
			Description("A unique name for this instruction").
			Value(&name)

		sourceSelect := huh.NewSelect[string]().
			Title("Instruction source").
			Description("How is the instruction content provided?").
			Options(
				huh.NewOption("Inline content (entered directly)", "content"),
				huh.NewOption("File path (reference a local file)", "path"),
			).
			Value(&source)

		form := huh.NewForm(
			huh.NewGroup(nameInput, sourceSelect),
		)

		if err := form.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if name == "" {
			fmt.Fprintln(os.Stderr, "error: instruction name is required")
			return
		}

		if existing[name] {
			fmt.Fprintf(os.Stderr, "error: instruction '%s' already exists in manifest\n", name)
			return
		}

		if source == "content" {
			contentInput := huh.NewText().
				Title("Instruction content").
				Description("Enter the instruction text").
				Value(&value)
			contentForm := huh.NewForm(huh.NewGroup(contentInput))
			if err := contentForm.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
		} else {
			pathInput := huh.NewInput().
				Title("Path to instruction file").
				Description("e.g. ./instructions/standards.md").
				Value(&value)
			pathForm := huh.NewForm(huh.NewGroup(pathInput))
			if err := pathForm.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return
			}
		}

		if value == "" {
			fmt.Fprintln(os.Stderr, "error: instruction content or path is required")
			return
		}

		inst := manifest.InstructionRef{Name: name}
		if source == "content" {
			inst.Content = value
		} else {
			inst.Path = value
		}

		m.Instructions = append(m.Instructions, inst)
		if err := manifest.SaveManifest(m, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving manifest: %v\n", err)
			return
		}
		fmt.Printf("Added instruction '%s' to %s\n", name, filepath.Base(manifestPath))
		fmt.Println("\nRun 'positive-vibes apply' to install everywhere!")
		return
	}

	report, err := installResourcesCommandAction(project, globalPath, string(ResourceInstructions), names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}

	for _, name := range report.SkippedDuplicateNames {
		fmt.Fprintf(os.Stderr, "warning: instruction '%s' already exists in manifest, skipping\n", name)
	}
	fmt.Printf("Saved %d instruction(s) to %s\n", len(report.MutatedNames), filepath.Base(manifestPath))
	fmt.Println("Run 'positive-vibes apply' to install everywhere!")
}

func init() {
	rootCmd.AddCommand(installCmd)
}
