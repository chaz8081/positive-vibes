package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
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

		names := dedup(args[1:])

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

func installSkillsRun(names []string) {
	project := ProjectDir()
	globalPath := defaultGlobalManifestPath()

	// Find existing manifest
	_, manifestPath, findErr := manifest.LoadManifestFromProject(project)
	if findErr != nil {
		manifestPath = filepath.Join(project, "vibes.yaml")
	}

	// Build registries
	regs := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	if merged, err := manifest.LoadMergedManifest(project, globalPath); err == nil {
		regs = append(regs, gitRegistriesFromManifest(merged)...)
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

	// Install each skill
	inst := engine.NewInstaller(regs)
	for _, name := range names {
		fmt.Printf("Installing '%s'...\n", name)
		if err := inst.Install(name, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}

		// Check what was installed
		m, err := manifest.LoadManifest(manifestPath)
		if err != nil {
			debugf("warning: could not reload manifest after install: %v", err)
		} else {
			for _, s := range m.Skills {
				if s.Name == name && s.Path != "" {
					fmt.Printf("  Found local skill at %s\n", s.Path)
					break
				}
			}
		}
		fmt.Printf("  Added '%s' to %s\n", name, filepath.Base(manifestPath))
	}

	fmt.Println("\nRun 'positive-vibes apply' to install everywhere!")
}

func installAgentsRun(names []string) {
	project := ProjectDir()

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

	// If no names provided, prompt for agent details interactively
	if len(names) == 0 {
		var name, source, value string

		nameInput := huh.NewInput().
			Title("Agent name").
			Description("A unique name for this agent").
			Value(&name)

		sourceSelect := huh.NewSelect[string]().
			Title("Agent source").
			Description("Where is the agent definition?").
			Options(
				huh.NewOption("Local path (file in this project)", "path"),
				huh.NewOption("Registry (registry/skill:path format)", "registry"),
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
			pathPrompt = "Registry reference (e.g. awesome-copilot/my-skill:agents/reviewer.md)"
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
			agent.Registry = value
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

	// Non-interactive: add named agents with path source (convention: ./agents/<name>.md)
	added := 0
	for _, name := range names {
		if existing[name] {
			fmt.Fprintf(os.Stderr, "warning: agent '%s' already exists in manifest, skipping\n", name)
			continue
		}

		agent := manifest.AgentRef{
			Name: name,
			Path: fmt.Sprintf("./agents/%s.md", name),
		}
		m.Agents = append(m.Agents, agent)
		existing[name] = true
		added++
		fmt.Printf("Added agent '%s' (path: %s)\n", name, agent.Path)
	}

	if added > 0 {
		if err := manifest.SaveManifest(m, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving manifest: %v\n", err)
			return
		}
		fmt.Printf("\nSaved %d agent(s) to %s\n", added, filepath.Base(manifestPath))
		fmt.Println("Run 'positive-vibes apply' to install everywhere!")
	}
}

func installInstructionsRun(names []string) {
	project := ProjectDir()

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

	// Non-interactive: add named instructions with path source (convention: ./instructions/<name>.md)
	added := 0
	for _, name := range names {
		if existing[name] {
			fmt.Fprintf(os.Stderr, "warning: instruction '%s' already exists in manifest, skipping\n", name)
			continue
		}

		inst := manifest.InstructionRef{
			Name: name,
			Path: fmt.Sprintf("./instructions/%s.md", name),
		}
		m.Instructions = append(m.Instructions, inst)
		existing[name] = true
		added++
		fmt.Printf("Added instruction '%s' (path: %s)\n", name, inst.Path)
	}

	if added > 0 {
		if err := manifest.SaveManifest(m, manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving manifest: %v\n", err)
			return
		}
		fmt.Printf("\nSaved %d instruction(s) to %s\n", added, filepath.Base(manifestPath))
		fmt.Println("Run 'positive-vibes apply' to install everywhere!")
	}
}

func init() {
	rootCmd.AddCommand(installCmd)
}
