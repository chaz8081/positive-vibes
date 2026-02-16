package cli

import (
	"fmt"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <resource-type> <name>",
	Short: "Show details for a specific resource",
	Long: `Display detailed information about a resource.

Resource types: skills, agents, instructions, prompts

Examples:
  positive-vibes show skills code-review
  positive-vibes show agents reviewer
  positive-vibes show instructions coding-standards`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: makeValidArgsFunction("all"),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		resType, err := ParseResourceType(args[0])
		if err != nil {
			return err
		}
		name := args[1]

		switch resType {
		case ResourceSkills:
			return showSkillRun(name)
		case ResourceAgents:
			return showAgentRun(name)
		case ResourceInstructions:
			return showInstructionRun(name)
		case ResourcePrompts:
			return showPromptRun(name)
		}
		return nil
	},
}

// ShowResourceCommandAction resolves details for command show flows.
func ShowResourceCommandAction(projectDir, globalPath, kind, name string) (ResourceDetailResult, error) {
	return ShowResourceDetail(projectDir, globalPath, kind, name)
}

func showSkillRun(name string) error {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourceSkills), name)
	if err != nil {
		return err
	}
	skill, ok := detail.Payload.(*schema.Skill)
	if !ok {
		return fmt.Errorf("unexpected skill payload for %s", name)
	}
	fmt.Print(formatSkillShow(skill, detail.Registry, detail.RegistryURL, detail.Installed))
	return nil
}

func showAgentRun(name string) error {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourceAgents), name)
	if err != nil {
		return err
	}
	agent, ok := detail.Payload.(manifest.AgentRef)
	if !ok {
		agent = manifest.AgentRef{Name: detail.Name, Registry: detail.Registry, Path: detail.Path}
	}
	fmt.Print(formatAgentShow(agent, detail.Installed))
	return nil
}

func showInstructionRun(name string) error {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourceInstructions), name)
	if err != nil {
		return err
	}
	inst, ok := detail.Payload.(manifest.InstructionRef)
	if !ok {
		inst = manifest.InstructionRef{Name: detail.Name, Registry: detail.Registry, Path: detail.Path}
	}
	fmt.Print(formatInstructionShow(inst, detail.Installed))
	return nil
}

func showPromptRun(name string) error {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourcePrompts), name)
	if err != nil {
		return err
	}
	prompt, ok := detail.Payload.(manifest.PromptRef)
	if !ok {
		prompt = manifest.PromptRef{Name: detail.Name, Registry: detail.Registry, Path: detail.Path}
	}
	fmt.Print(formatPromptShow(prompt, detail.Installed))
	return nil
}

func init() {
	rootCmd.AddCommand(showCmd)
}
