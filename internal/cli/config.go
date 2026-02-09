package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

// --- Pure helper functions (tested independently) ---

// formatPaths returns a human-readable summary of config file locations and their status.
func formatPaths(globalPath, projectDir, cacheDir string) string {
	var b strings.Builder

	// Global config status
	globalStatus := "[not found]"
	if _, err := os.Stat(globalPath); err == nil {
		globalStatus = "[found]"
	}
	fmt.Fprintf(&b, "Global config:  %s  %s\n", globalPath, globalStatus)

	// Local config status -- check vibes.yaml then vibes.yml
	localStatus := "[not found]"
	localPath := "(none)"
	for _, name := range manifest.ManifestFilenames {
		p := filepath.Join(projectDir, name)
		if _, err := os.Stat(p); err == nil {
			localPath = p
			if name == "vibes.yml" {
				localStatus = "[found, legacy name]"
			} else {
				localStatus = "[found]"
			}
			break
		}
	}
	if localStatus == "[not found]" {
		localPath = filepath.Join(projectDir, "vibes.yaml")
	}
	fmt.Fprintf(&b, "Local config:   %s  %s\n", localPath, localStatus)

	// Project dir and cache
	absProject, err := filepath.Abs(projectDir)
	if err != nil {
		absProject = projectDir
	}
	fmt.Fprintf(&b, "Project dir:    %s\n", absProject)
	fmt.Fprintf(&b, "Cache dir:      %s\n", cacheDir)

	return b.String()
}

// renderMergedYAML marshals a manifest to YAML for display.
func renderMergedYAML(m *manifest.Manifest) string {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Sprintf("# error marshaling manifest: %v\n", err)
	}
	return string(data)
}

// annotateManifest renders the merged manifest with [global]/[local] source annotations.
// Either global or local may be nil (but not both).
func annotateManifest(global, local, merged *manifest.Manifest) string {
	var b strings.Builder

	// Build lookup sets for skills
	globalSkills := make(map[string]bool)
	localSkills := make(map[string]bool)
	if global != nil {
		for _, s := range global.Skills {
			globalSkills[s.Name] = true
		}
	}
	if local != nil {
		for _, s := range local.Skills {
			localSkills[s.Name] = true
		}
	}

	// Build lookup sets for registries
	globalRegs := make(map[string]bool)
	localRegs := make(map[string]bool)
	if global != nil {
		for _, r := range global.Registries {
			globalRegs[r.Name] = true
		}
	}
	if local != nil {
		for _, r := range local.Registries {
			localRegs[r.Name] = true
		}
	}

	// Build lookup sets for instructions
	globalInstructions := make(map[string]bool)
	localInstructions := make(map[string]bool)
	if global != nil {
		for _, i := range global.Instructions {
			globalInstructions[i.Name] = true
		}
	}
	if local != nil {
		for _, i := range local.Instructions {
			localInstructions[i.Name] = true
		}
	}

	// Build lookup sets for agents
	globalAgents := make(map[string]bool)
	localAgents := make(map[string]bool)
	if global != nil {
		for _, a := range global.Agents {
			globalAgents[a.Name] = true
		}
	}
	if local != nil {
		for _, a := range local.Agents {
			localAgents[a.Name] = true
		}
	}

	// Registries
	if len(merged.Registries) > 0 {
		b.WriteString("registries:\n")
		for _, r := range merged.Registries {
			tag := sourceTag(globalRegs[r.Name], localRegs[r.Name])
			b.WriteString(fmt.Sprintf("  - name: %s  %s\n", r.Name, tag))
			b.WriteString(fmt.Sprintf("    url: %s\n", r.URL))
		}
	}

	// Skills
	if len(merged.Skills) > 0 {
		b.WriteString("skills:\n")
		for _, s := range merged.Skills {
			tag := sourceTag(globalSkills[s.Name], localSkills[s.Name])
			if s.Path != "" {
				b.WriteString(fmt.Sprintf("  - name: %s  %s\n", s.Name, tag))
				b.WriteString(fmt.Sprintf("    path: %s\n", s.Path))
			} else {
				b.WriteString(fmt.Sprintf("  - name: %s  %s\n", s.Name, tag))
			}
		}
	}

	// Targets
	if len(merged.Targets) > 0 {
		// Determine if targets come from local or global
		targetsFromLocal := local != nil && len(local.Targets) > 0
		targetsSource := "# [global]"
		if targetsFromLocal {
			targetsSource = "# [local]"
		}
		b.WriteString(fmt.Sprintf("targets: %s\n", targetsSource))
		for _, t := range merged.Targets {
			b.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}

	// Instructions
	if len(merged.Instructions) > 0 {
		b.WriteString("instructions:\n")
		for _, inst := range merged.Instructions {
			tag := sourceTag(globalInstructions[inst.Name], localInstructions[inst.Name])
			b.WriteString(fmt.Sprintf("  - name: %s  %s\n", inst.Name, tag))
			if inst.Content != "" {
				b.WriteString(fmt.Sprintf("    content: %q\n", inst.Content))
			} else if inst.Path != "" {
				b.WriteString(fmt.Sprintf("    path: %s\n", inst.Path))
			}
			if inst.ApplyTo != "" {
				b.WriteString(fmt.Sprintf("    apply_to: %q\n", inst.ApplyTo))
			}
		}
	}

	// Agents
	if len(merged.Agents) > 0 {
		b.WriteString("agents:\n")
		for _, a := range merged.Agents {
			tag := sourceTag(globalAgents[a.Name], localAgents[a.Name])
			b.WriteString(fmt.Sprintf("  - name: %s  %s\n", a.Name, tag))
			if a.Path != "" {
				b.WriteString(fmt.Sprintf("    path: %s\n", a.Path))
			} else if a.Registry != "" {
				b.WriteString(fmt.Sprintf("    registry: %s\n", a.Registry))
			}
		}
	}

	return b.String()
}

// sourceTag returns the appropriate annotation based on origin.
func sourceTag(inGlobal, inLocal bool) string {
	switch {
	case inGlobal && inLocal:
		return "# [local, overrides global]"
	case inGlobal:
		return "# [global]"
	case inLocal:
		return "# [local]"
	default:
		return ""
	}
}

// --- Validation ---

// configProblem represents a single validation issue.
type configProblem struct {
	field   string
	message string
}

// configValidationResult holds the results of a config validation run.
type configValidationResult struct {
	problems []configProblem
}

func (r *configValidationResult) ok() bool {
	return len(r.problems) == 0
}

func (r *configValidationResult) add(field, message string) {
	r.problems = append(r.problems, configProblem{field: field, message: message})
}

// validateConfig runs offline checks on a merged manifest.
// embeddedSkills is the list of skill names available in the embedded registry.
// hasLocalConfig indicates whether a local project config was found; when false
// (global-only), empty skills/targets are not flagged as problems since a global
// config is just a base layer.
func validateConfig(m *manifest.Manifest, embeddedSkills []string, hasLocalConfig ...bool) *configValidationResult {
	result := &configValidationResult{}

	// Determine if we should require skills/targets.
	// Default to true (backwards-compatible with existing callers).
	requireSkillsAndTargets := true
	if len(hasLocalConfig) > 0 {
		requireSkillsAndTargets = hasLocalConfig[0]
	}

	// Check skills defined (only when local config is present)
	if requireSkillsAndTargets && len(m.Skills) == 0 {
		result.add("skills", "no skills defined")
	}

	// Check targets defined (only when local config is present)
	if requireSkillsAndTargets && len(m.Targets) == 0 {
		result.add("targets", "no targets defined")
	}

	// Check each target is valid
	validTargets := make(map[string]bool)
	for _, t := range manifest.ValidTargets {
		validTargets[t] = true
	}
	for _, t := range m.Targets {
		if !validTargets[t] {
			result.add(t, fmt.Sprintf("invalid target (valid: %s)", strings.Join(manifest.ValidTargets, ", ")))
		}
	}

	// Check each skill is resolvable
	embeddedSet := make(map[string]bool)
	for _, s := range embeddedSkills {
		embeddedSet[s] = true
	}
	for _, s := range m.Skills {
		if s.Path != "" {
			// Local path skill: check directory exists
			if _, err := os.Stat(s.Path); err != nil {
				result.add(s.Name, "path not found: "+s.Path)
			}
		} else if !embeddedSet[s.Name] {
			result.add(s.Name, "not found in any registry")
		}
	}

	// Check each instruction with a path
	for _, inst := range m.Instructions {
		if inst.Path != "" {
			if _, err := os.Stat(inst.Path); err != nil {
				result.add(inst.Name, "path not found: "+inst.Path)
			}
		}
	}

	// Check each agent with a path
	for _, a := range m.Agents {
		if a.Path != "" {
			if _, err := os.Stat(a.Path); err != nil {
				result.add(a.Name, "path not found: "+a.Path)
			}
		}
	}

	return result
}

// --- Cobra commands ---

var configShowSources bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and validate your vibes configuration",
	Long: `View the effective merged configuration, check file locations,
or validate your setup for problems.

Subcommands:
  show       Print the effective merged config as YAML
  paths      Show resolved config file locations
  validate   Check config for problems (offline)`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the effective merged configuration",
	Long: `Loads global and project manifests, merges them, and prints the
effective configuration as YAML.

Use --sources to annotate each value with [global], [local], or
[local, overrides global] to show where each value comes from.`,
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()

		if configShowSources {
			// Load global and local separately for annotation
			var global, local *manifest.Manifest
			if data, err := os.ReadFile(globalPath); err == nil {
				if g, err := manifest.LoadManifestFromBytes(data); err == nil {
					global = g
				}
			}
			if p, _, err := manifest.LoadManifestFromProject(project); err == nil {
				local = p
			}
			if global == nil && local == nil {
				fmt.Fprintf(os.Stderr, "No config found (checked %s and %s)\n", globalPath, project)
				os.Exit(1)
			}

			merged, err := manifest.LoadMergedManifest(project, globalPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(annotateManifest(global, local, merged))
		} else {
			merged, err := manifest.LoadMergedManifest(project, globalPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "No config found (checked %s and %s)\n", globalPath, project)
				os.Exit(1)
			}
			fmt.Print(renderMergedYAML(merged))
		}
	},
}

var configPathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Show resolved config file locations",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()
		home, _ := os.UserHomeDir()
		cacheDir := filepath.Join(home, ".positive-vibes", "cache")
		fmt.Print(formatPaths(globalPath, project, cacheDir))
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check configuration for problems",
	Long: `Loads the merged configuration and runs offline checks:
- Config files exist and parse correctly
- All targets are valid
- All skills are resolvable (embedded or local path)

Exits with code 1 if any problems are found.`,
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()

		// Report config file status
		globalStatus := "ok"
		if _, err := os.Stat(globalPath); err != nil {
			globalStatus = "not found"
		}
		fmt.Fprintf(os.Stdout, "Loading global config:  %s  %s\n", globalPath, globalStatus)

		localStatus := "ok"
		localPath := ""
		for _, name := range manifest.ManifestFilenames {
			p := filepath.Join(project, name)
			if _, err := os.Stat(p); err == nil {
				localPath = p
				break
			}
		}
		if localPath == "" {
			localStatus = "not found"
			localPath = filepath.Join(project, "vibes.yaml")
		}
		fmt.Fprintf(os.Stdout, "Loading local config:   %s  %s\n\n", localPath, localStatus)

		// Load merged manifest
		merged, err := manifest.LoadMergedManifest(project, globalPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Get embedded skill names
		embedded := registry.NewEmbeddedRegistry()
		embeddedSkills, _ := embedded.List()

		// Run validation -- pass whether local config was found
		hasLocal := localStatus == "ok"
		result := validateConfig(merged, embeddedSkills, hasLocal)

		// Print registries
		fmt.Fprintf(os.Stdout, "Registries (%d):\n", len(merged.Registries))
		for _, r := range merged.Registries {
			fmt.Fprintf(os.Stdout, "  ok  %s  %s\n", r.Name, r.URL)
		}
		fmt.Println()

		// Print skills
		fmt.Fprintf(os.Stdout, "Skills (%d):\n", len(merged.Skills))
		problemSkills := make(map[string]string)
		for _, p := range result.problems {
			problemSkills[p.field] = p.message
		}
		for _, s := range merged.Skills {
			if msg, bad := problemSkills[s.Name]; bad {
				fmt.Fprintf(os.Stdout, "  FAIL  %s  %s\n", s.Name, msg)
			} else {
				source := "(embedded)"
				if s.Path != "" {
					source = "(local: " + s.Path + ")"
				}
				fmt.Fprintf(os.Stdout, "  ok  %s  %s\n", s.Name, source)
			}
		}
		fmt.Println()

		// Print targets
		fmt.Fprintf(os.Stdout, "Targets (%d):\n", len(merged.Targets))
		for _, t := range merged.Targets {
			if msg, bad := problemSkills[t]; bad {
				fmt.Fprintf(os.Stdout, "  FAIL  %s  %s\n", t, msg)
			} else {
				fmt.Fprintf(os.Stdout, "  ok  %s\n", t)
			}
		}
		fmt.Println()

		if result.ok() {
			if !hasLocal {
				fmt.Println("No local vibes detected. Run 'positive-vibes init' to spread some good vibes here.")
			} else {
				fmt.Println("All checks passed.")
			}
		} else {
			fmt.Fprintf(os.Stdout, "%d problem(s) found.\n", len(result.problems))
			os.Exit(1)
		}
	},
}

func init() {
	configShowCmd.Flags().BoolVar(&configShowSources, "sources", false, "annotate values with their source (global/local)")
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathsCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
