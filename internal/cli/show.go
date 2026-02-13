package cli

import (
	"fmt"
	"os"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
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

		switch resType {
		case ResourceSkills:
			showSkillRun(name)
		case ResourceAgents:
			showAgentRun(name)
		case ResourceInstructions:
			showInstructionRun(name)
		}
	},
}

// ShowResourceCommandAction resolves details for command show flows.
func ShowResourceCommandAction(projectDir, globalPath, kind, name string) (ResourceDetailResult, error) {
	return ShowResourceDetail(projectDir, globalPath, kind, name)
}

func showSkillRun(name string) {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourceSkills), name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	skill, ok := detail.Payload.(*schema.Skill)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: unexpected skill payload for %s\n", name)
		return
	}
	fmt.Print(formatSkillShow(skill, detail.Registry, detail.RegistryURL, detail.Installed))
}

func showAgentRun(name string) {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourceAgents), name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	agent, ok := detail.Payload.(manifest.AgentRef)
	if !ok {
		agent = manifest.AgentRef{Name: detail.Name, Registry: detail.Registry, Path: detail.Path}
	}
	fmt.Print(formatAgentShow(agent, detail.Installed))
}

func showInstructionRun(name string) {
	detail, err := ShowResourceCommandAction(ProjectDir(), defaultGlobalManifestPath(), string(ResourceInstructions), name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	inst, ok := detail.Payload.(manifest.InstructionRef)
	if !ok {
		inst = manifest.InstructionRef{Name: detail.Name, Registry: detail.Registry, Path: detail.Path}
	}
	fmt.Print(formatInstructionShow(inst, detail.Installed))
}

func init() {
	rootCmd.AddCommand(showCmd)
}
