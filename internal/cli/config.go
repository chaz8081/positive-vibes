package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

type statusKind int

const (
	statusOK statusKind = iota
	statusWarn
	statusFail
)

func shouldUseColor(mode string) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	switch strings.ToLower(mode) {
	case "always":
		return true
	case "never":
		return false
	default:
		if strings.EqualFold(os.Getenv("TERM"), "dumb") {
			return false
		}
		fi, err := os.Stdout.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
}

func colorizeStatus(label string, kind statusKind, enabled bool) string {
	if !enabled {
		return label
	}
	code := "32"
	switch kind {
	case statusWarn:
		code = "33"
	case statusFail:
		code = "31"
	}
	return "\x1b[" + code + "m" + label + "\x1b[0m"
}

func colorizeSourceAnnotations(s string, enabled bool) string {
	if !enabled {
		return s
	}
	repl := strings.NewReplacer(
		"# [global]", "\x1b[34m# [global]\x1b[0m",
		"# [local]", "\x1b[32m# [local]\x1b[0m",
		"# [local, overrides global]", "\x1b[33m# [local, overrides global]\x1b[0m",
	)
	return repl.Replace(s)
}

func relativePathsNoEffectNote(m *manifest.Manifest) string {
	if m == nil {
		return ""
	}
	for _, s := range m.Skills {
		if s.Path != "" {
			return ""
		}
	}
	for _, i := range m.Instructions {
		if i.Path != "" {
			return ""
		}
	}
	for _, a := range m.Agents {
		if a.Path != "" {
			return ""
		}
	}
	return "# note: no path entries are present, so --relative-paths has no visible effect"
}

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
	return annotateManifestWithOptions(global, local, merged, annotateRenderOptions{})
}

type annotateRenderOptions struct {
	RelativePaths bool
	ProjectDir    string
	GlobalPath    string
}

func pathForDisplay(path string, root string, relative bool) string {
	if path == "" || !relative || root == "" || !filepath.IsAbs(path) {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	if rel == "." {
		return "./"
	}
	if strings.HasPrefix(rel, "..") {
		return path
	}
	return "./" + filepath.ToSlash(rel)
}

func annotateManifestWithOptions(global, local, merged *manifest.Manifest, opts annotateRenderOptions) string {
	var b strings.Builder
	globalRoot := filepath.Dir(opts.GlobalPath)
	localRoot := opts.ProjectDir

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
			b.WriteString(fmt.Sprintf("  - name: %s  %s\n", s.Name, tag))
			if s.Registry != "" {
				b.WriteString(fmt.Sprintf("    registry: %s\n", s.Registry))
			}
			if s.Path != "" {
				pathRoot := globalRoot
				if localSkills[s.Name] {
					pathRoot = localRoot
				}
				if s.Registry != "" {
					pathRoot = ""
				}
				displayPath := pathForDisplay(s.Path, pathRoot, opts.RelativePaths)
				b.WriteString(fmt.Sprintf("    path: %s\n", displayPath))
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
			if inst.Registry != "" {
				b.WriteString(fmt.Sprintf("    registry: %s\n", inst.Registry))
			}
			if inst.Content != "" {
				b.WriteString(fmt.Sprintf("    content: %q\n", inst.Content))
			} else if inst.Path != "" {
				pathRoot := globalRoot
				if localInstructions[inst.Name] {
					pathRoot = localRoot
				}
				if inst.Registry != "" {
					pathRoot = ""
				}
				displayPath := pathForDisplay(inst.Path, pathRoot, opts.RelativePaths)
				b.WriteString(fmt.Sprintf("    path: %s\n", displayPath))
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
			if a.Registry != "" {
				b.WriteString(fmt.Sprintf("    registry: %s\n", a.Registry))
			}
			if a.Path != "" {
				pathRoot := globalRoot
				if localAgents[a.Name] {
					pathRoot = localRoot
				}
				if a.Registry != "" {
					pathRoot = ""
				}
				displayPath := pathForDisplay(a.Path, pathRoot, opts.RelativePaths)
				b.WriteString(fmt.Sprintf("    path: %s\n", displayPath))
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
	warnings []configProblem
}

func (r *configValidationResult) ok() bool {
	return len(r.problems) == 0
}

func (r *configValidationResult) add(field, message string) {
	r.problems = append(r.problems, configProblem{field: field, message: message})
}

func (r *configValidationResult) warn(field, message string) {
	r.warnings = append(r.warnings, configProblem{field: field, message: message})
}

// validateConfig runs offline checks on a merged manifest.
// embeddedSkills is the list of skill names available in the embedded registry.
// hasLocalConfig indicates whether a local project config was found; when false
// (global-only), empty skills/targets are not flagged as problems since a global
// config is just a base layer.
func validateConfig(m *manifest.Manifest, embeddedSkills []string, hasLocalConfig ...bool) *configValidationResult {
	hasLocal := true
	if len(hasLocalConfig) > 0 {
		hasLocal = hasLocalConfig[0]
	}
	return validateConfigWithContext(m, embeddedSkills, hasLocal, nil, nil)
}

func validateConfigWithContext(m *manifest.Manifest, embeddedSkills []string, hasLocalConfig bool, global, local *manifest.Manifest, unresolvedRegistries ...string) *configValidationResult {
	result := &configValidationResult{}

	// Determine if we should require skills/targets.
	// Default to true (backwards-compatible with existing callers).
	requireSkillsAndTargets := hasLocalConfig

	// Check resource/target presence (only when local config is present)
	if requireSkillsAndTargets {
		resourceCount := len(m.Skills) + len(m.Instructions) + len(m.Agents)
		if resourceCount == 0 {
			result.add("resources", "no resources defined (skills, instructions, or agents)")
		}
		if resourceCount > 0 && len(m.Targets) == 0 {
			result.add("targets", "no targets defined")
		}
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
		if s.Registry == "" && s.Path != "" {
			// Local path skill: check directory exists
			if _, err := os.Stat(s.Path); err != nil {
				result.add(s.Name, "path not found: "+s.Path)
			}
		} else if s.Registry != "" {
			if !registryNameExists(s.Registry, m.Registries) && s.Registry != "embedded" {
				result.add(s.Name, "registry not found: "+s.Registry)
			}
		} else if !embeddedSet[s.Name] {
			if len(unresolvedRegistries) > 0 {
				result.warn(s.Name, fmt.Sprintf("could not verify skill due to registry lookup failures: %s", strings.Join(unresolvedRegistries, ", ")))
			} else {
				result.add(s.Name, "not found in any registry")
			}
		}
	}

	// Check each instruction with a path
	for _, inst := range m.Instructions {
		if inst.Registry != "" {
			if !registryNameExists(inst.Registry, m.Registries) {
				result.add(inst.Name, "registry not found: "+inst.Registry)
			}
		} else if inst.Path != "" {
			if _, err := os.Stat(inst.Path); err != nil {
				result.add(inst.Name, "path not found: "+inst.Path)
			}
		}
	}

	// Check each agent with a path
	for _, a := range m.Agents {
		if a.Registry != "" {
			if !registryNameExists(a.Registry, m.Registries) {
				result.add(a.Name, "registry not found: "+a.Registry)
			}
		} else if a.Path != "" {
			if _, err := os.Stat(a.Path); err != nil {
				result.add(a.Name, "path not found: "+a.Path)
			}
		}
	}

	for _, p := range localGlobalRegistryDependencyProblems(global, local) {
		result.add(p.field, p.message)
	}

	if global != nil && local != nil {
		d := manifest.ComputeRiskyOverrideDiagnostics(global, local)
		for _, name := range d.Skills {
			result.warn(name, "local skill switches source type (path vs registry/embedded)")
		}
		for _, name := range d.Instructions {
			result.warn(name, "local instruction switches source type (content vs path vs registry)")
		}
		for _, name := range d.Agents {
			result.warn(name, "local agent switches source type (path vs registry)")
		}
	}

	return result
}

func registryNameExists(name string, regs []manifest.RegistryRef) bool {
	for _, r := range regs {
		if r.Name == name {
			return true
		}
	}
	return false
}

func localGlobalRegistryDependencyProblems(global, local *manifest.Manifest) []configProblem {
	if local == nil {
		return nil
	}
	localRegs := make(map[string]bool)
	for _, r := range local.Registries {
		localRegs[r.Name] = true
	}
	globalRegs := make(map[string]bool)
	if global != nil {
		for _, r := range global.Registries {
			globalRegs[r.Name] = true
		}
	}
	var out []configProblem
	appendIfGlobalOnly := func(field, reg, kind string) {
		if reg == "" || reg == "embedded" {
			return
		}
		if !localRegs[reg] && globalRegs[reg] {
			out = append(out, configProblem{
				field:   field,
				message: fmt.Sprintf("%s references registry %q defined only in global config; add it to project registries for portability", kind, reg),
			})
		}
	}
	for _, s := range local.Skills {
		appendIfGlobalOnly(s.Name, s.Registry, "skill")
	}
	for _, i := range local.Instructions {
		appendIfGlobalOnly(i.Name, i.Registry, "instruction")
	}
	for _, a := range local.Agents {
		appendIfGlobalOnly(a.Name, a.Registry, "agent")
	}
	return out
}

func collectAvailableSkillsFromSources(sources []registry.SkillSource) (map[string]bool, []configProblem) {
	available := make(map[string]bool)
	var warnings []configProblem

	for _, src := range sources {
		names, err := src.List()
		if err != nil {
			warnings = append(warnings, configProblem{
				field:   "registry/" + src.Name(),
				message: "could not list skills: " + err.Error(),
			})
			continue
		}
		for _, n := range names {
			available[n] = true
		}
	}

	return available, warnings
}

func namesFromSkills(items []manifest.SkillRef) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it.Name] = true
	}
	return m
}

func namesFromInstructions(items []manifest.InstructionRef) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it.Name] = true
	}
	return m
}

func namesFromAgents(items []manifest.AgentRef) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it.Name] = true
	}
	return m
}

func setDiff(a, b map[string]bool) []string {
	var out []string
	for name := range a {
		if !b[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func formatConfigDiff(global, local, merged *manifest.Manifest) string {
	if global == nil {
		global = &manifest.Manifest{}
	}
	if local == nil {
		local = &manifest.Manifest{}
	}
	if merged == nil {
		merged = &manifest.Manifest{}
	}

	globalSkills := namesFromSkills(global.Skills)
	localSkills := namesFromSkills(local.Skills)
	globalInst := namesFromInstructions(global.Instructions)
	localInst := namesFromInstructions(local.Instructions)
	globalAgents := namesFromAgents(global.Agents)
	localAgents := namesFromAgents(local.Agents)

	d := manifest.ComputeOverrideDiagnostics(global, local)
	var b strings.Builder

	b.WriteString("Global-only:\n")
	if items := setDiff(globalSkills, localSkills); len(items) > 0 {
		b.WriteString("  skills: " + strings.Join(items, ", ") + "\n")
	}
	if items := setDiff(globalInst, localInst); len(items) > 0 {
		b.WriteString("  instructions: " + strings.Join(items, ", ") + "\n")
	}
	if items := setDiff(globalAgents, localAgents); len(items) > 0 {
		b.WriteString("  agents: " + strings.Join(items, ", ") + "\n")
	}

	b.WriteString("\nLocal-only:\n")
	if items := setDiff(localSkills, globalSkills); len(items) > 0 {
		b.WriteString("  skills: " + strings.Join(items, ", ") + "\n")
	}
	if items := setDiff(localInst, globalInst); len(items) > 0 {
		b.WriteString("  instructions: " + strings.Join(items, ", ") + "\n")
	}
	if items := setDiff(localAgents, globalAgents); len(items) > 0 {
		b.WriteString("  agents: " + strings.Join(items, ", ") + "\n")
	}

	b.WriteString("\nOverrides:\n")
	if len(d.Skills) > 0 {
		b.WriteString("  skills: " + strings.Join(d.Skills, ", ") + "\n")
	}
	if len(d.Instructions) > 0 {
		b.WriteString("  instructions: " + strings.Join(d.Instructions, ", ") + "\n")
	}
	if len(d.Agents) > 0 {
		b.WriteString("  agents: " + strings.Join(d.Agents, ", ") + "\n")
	}
	if len(d.Registries) > 0 {
		b.WriteString("  registries: " + strings.Join(d.Registries, ", ") + "\n")
	}

	b.WriteString("\nEffective config summary:\n")
	b.WriteString(fmt.Sprintf("  registries: %d\n", len(merged.Registries)))
	b.WriteString(fmt.Sprintf("  skills: %d\n", len(merged.Skills)))
	b.WriteString(fmt.Sprintf("  instructions: %d\n", len(merged.Instructions)))
	b.WriteString(fmt.Sprintf("  agents: %d\n", len(merged.Agents)))
	b.WriteString(fmt.Sprintf("  targets: %d\n", len(merged.Targets)))

	return b.String()
}

func formatConfigDiffJSON(global, local, merged *manifest.Manifest) (string, error) {
	if global == nil {
		global = &manifest.Manifest{}
	}
	if local == nil {
		local = &manifest.Manifest{}
	}
	if merged == nil {
		merged = &manifest.Manifest{}
	}

	globalSkills := namesFromSkills(global.Skills)
	localSkills := namesFromSkills(local.Skills)
	globalInst := namesFromInstructions(global.Instructions)
	localInst := namesFromInstructions(local.Instructions)
	globalAgents := namesFromAgents(global.Agents)
	localAgents := namesFromAgents(local.Agents)

	payload := map[string]any{
		"global_only": map[string]any{
			"skills":       setDiff(globalSkills, localSkills),
			"instructions": setDiff(globalInst, localInst),
			"agents":       setDiff(globalAgents, localAgents),
		},
		"local_only": map[string]any{
			"skills":       setDiff(localSkills, globalSkills),
			"instructions": setDiff(localInst, globalInst),
			"agents":       setDiff(localAgents, globalAgents),
		},
		"overrides": map[string]any{
			"all":   manifest.ComputeOverrideDiagnostics(global, local),
			"risky": manifest.ComputeRiskyOverrideDiagnostics(global, local),
		},
		"effective_summary": map[string]any{
			"registries":   len(merged.Registries),
			"skills":       len(merged.Skills),
			"instructions": len(merged.Instructions),
			"agents":       len(merged.Agents),
			"targets":      len(merged.Targets),
		},
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func buildConfigDiffOutput(project, globalPath string, asJSON bool) (string, error) {
	var global, local *manifest.Manifest
	if data, err := os.ReadFile(globalPath); err == nil {
		if g, err := manifest.LoadManifestFromBytes(data); err == nil {
			global = g
		}
	}
	if p, _, err := manifest.LoadManifestFromProject(project); err == nil {
		local = p
	}

	merged, err := manifest.LoadMergedManifest(project, globalPath)
	if err != nil {
		return "", fmt.Errorf("no config found (checked %s and %s)", globalPath, project)
	}

	if asJSON {
		return formatConfigDiffJSON(global, local, merged)
	}
	return formatConfigDiff(global, local, merged), nil
}

// --- Cobra commands ---

var configShowSources bool
var configShowRelativePaths bool
var configDiffJSON bool
var configColor string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and validate your vibes configuration",
	Long: `View the effective merged configuration, check file locations,
or validate your setup for problems.

Subcommands:
  show       Print the effective merged config as YAML
  paths      Show resolved config file locations
  diff       Show differences between global/local/effective config
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
		colorEnabled := shouldUseColor(configColor)

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
			if configShowRelativePaths {
				if note := relativePathsNoEffectNote(merged); note != "" {
					fmt.Println(note)
				}
			}
			out := annotateManifestWithOptions(global, local, merged, annotateRenderOptions{
				RelativePaths: configShowRelativePaths,
				ProjectDir:    project,
				GlobalPath:    globalPath,
			})
			fmt.Print(colorizeSourceAnnotations(out, colorEnabled))
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

var configDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show global, local, and effective config differences",
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()
		out, err := buildConfigDiffOutput(project, globalPath, configDiffJSON)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Print(out)
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
		colorEnabled := shouldUseColor(configColor)

		// Report config file status
		globalStatus := "ok"
		if _, err := os.Stat(globalPath); err != nil {
			globalStatus = "not found"
		}
		statusLabel := colorizeStatus(globalStatus, statusOK, colorEnabled)
		if globalStatus != "ok" {
			statusLabel = colorizeStatus(globalStatus, statusWarn, colorEnabled)
		}
		fmt.Fprintf(os.Stdout, "Loading global config:  %s  %s\n", globalPath, statusLabel)

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
		localStatusLabel := colorizeStatus(localStatus, statusOK, colorEnabled)
		if localStatus != "ok" {
			localStatusLabel = colorizeStatus(localStatus, statusWarn, colorEnabled)
		}
		fmt.Fprintf(os.Stdout, "Loading local config:   %s  %s\n\n", localPath, localStatusLabel)

		// Load merged manifest
		merged, err := manifest.LoadMergedManifest(project, globalPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var globalM, localM *manifest.Manifest
		if data, err := os.ReadFile(globalPath); err == nil {
			globalM, _ = manifest.LoadManifestFromBytes(data)
		}
		if m, _, err := manifest.LoadManifestFromProject(project); err == nil {
			localM = m
		}

		// Get available skill names from embedded + configured registries
		sources := []registry.SkillSource{registry.NewEmbeddedRegistry()}
		sources = append(sources, gitRegistriesFromManifest(merged)...)
		availableSkills, sourceWarnings := collectAvailableSkillsFromSources(sources)
		var skillNames []string
		for name := range availableSkills {
			skillNames = append(skillNames, name)
		}
		sort.Strings(skillNames)
		var unresolved []string
		for _, w := range sourceWarnings {
			resultField := strings.TrimPrefix(w.field, "registry/")
			unresolved = append(unresolved, resultField)
		}

		// Run validation -- pass whether local config was found
		hasLocal := localStatus == "ok"
		result := validateConfigWithContext(merged, skillNames, hasLocal, globalM, localM, unresolved...)
		result.warnings = append(sourceWarnings, result.warnings...)

		// Print registries
		fmt.Fprintf(os.Stdout, "Registries (%d):\n", len(merged.Registries))
		for _, r := range merged.Registries {
			fmt.Fprintf(os.Stdout, "  %s  %s  %s\n", colorizeStatus("ok", statusOK, colorEnabled), r.Name, r.URL)
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
				fmt.Fprintf(os.Stdout, "  %s  %s  %s\n", colorizeStatus("FAIL", statusFail, colorEnabled), s.Name, msg)
			} else {
				source := "(embedded)"
				if s.Path != "" {
					source = "(local: " + s.Path + ")"
				}
				fmt.Fprintf(os.Stdout, "  %s  %s  %s\n", colorizeStatus("ok", statusOK, colorEnabled), s.Name, source)
			}
		}
		fmt.Println()

		// Print targets
		fmt.Fprintf(os.Stdout, "Targets (%d):\n", len(merged.Targets))
		for _, t := range merged.Targets {
			if msg, bad := problemSkills[t]; bad {
				fmt.Fprintf(os.Stdout, "  %s  %s  %s\n", colorizeStatus("FAIL", statusFail, colorEnabled), t, msg)
			} else {
				fmt.Fprintf(os.Stdout, "  %s  %s\n", colorizeStatus("ok", statusOK, colorEnabled), t)
			}
		}
		fmt.Println()

		if len(result.warnings) > 0 {
			fmt.Println("Warnings:")
			for _, w := range result.warnings {
				fmt.Fprintf(os.Stdout, "  %s  %s  %s\n", colorizeStatus("WARN", statusWarn, colorEnabled), w.field, w.message)
			}
			fmt.Println()
		}

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
	configShowCmd.Flags().BoolVar(&configShowRelativePaths, "relative-paths", false, "show source-annotated paths relative to their config root")
	configDiffCmd.Flags().BoolVar(&configDiffJSON, "json", false, "emit config diff as JSON")
	configCmd.PersistentFlags().StringVar(&configColor, "color", "auto", "color output: auto, always, never")
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathsCmd)
	configCmd.AddCommand(configDiffCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
