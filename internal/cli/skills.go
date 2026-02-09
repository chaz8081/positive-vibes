package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chaz8081/positive-vibes/internal/engine"
	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/internal/registry"
	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/spf13/cobra"
)

// --- Pure helper types and functions (tested independently) ---

// registrySkillSet holds the skills available from a single registry source.
type registrySkillSet struct {
	RegistryName string
	URL          string // empty for embedded
	Skills       []string
	Error        string // non-empty if listing failed
}

// collectSkillSets gathers available skills from the embedded registry and
// any git registries defined in the merged manifest. If merged is nil, only
// the embedded registry is consulted.
func collectSkillSets(merged *manifest.Manifest) []registrySkillSet {
	var sets []registrySkillSet

	// Always include embedded registry first.
	embedded := registry.NewEmbeddedRegistry()
	if names, err := embedded.List(); err == nil && len(names) > 0 {
		sets = append(sets, registrySkillSet{
			RegistryName: "embedded",
			Skills:       names,
		})
	}

	// Add git registries from merged manifest.
	if merged != nil {
		for _, src := range gitRegistriesFromManifest(merged) {
			names, err := src.List()
			if err != nil {
				sets = append(sets, registrySkillSet{
					RegistryName: src.Name(),
					Error:        err.Error(),
				})
				continue
			}
			if len(names) > 0 {
				// Find the URL from the manifest registry entry.
				var url string
				for _, r := range merged.Registries {
					if r.Name == src.Name() {
						url = r.URL
						break
					}
				}
				sets = append(sets, registrySkillSet{
					RegistryName: src.Name(),
					URL:          url,
					Skills:       names,
				})
			}
		}
	}

	return sets
}

// formatSkillsList renders the skill sets into a human-readable string,
// grouped by registry. Skills present in the installed map are marked.
func formatSkillsList(sets []registrySkillSet, installed map[string]bool) string {
	return formatSkillsListFiltered(sets, installed, listFormatOptions{})
}

// listFormatOptions controls filtering for the list output.
type listFormatOptions struct {
	Registry      string // filter to a specific registry name
	InstalledOnly bool   // show only installed skills
}

// formatSkillsListFiltered renders skill sets with optional filters applied.
func formatSkillsListFiltered(sets []registrySkillSet, installed map[string]bool, opts listFormatOptions) string {
	// Apply registry filter
	var filtered []registrySkillSet
	for _, s := range sets {
		if opts.Registry != "" && s.RegistryName != opts.Registry {
			continue
		}
		if opts.InstalledOnly {
			var kept []string
			for _, name := range s.Skills {
				if installed[name] {
					kept = append(kept, name)
				}
			}
			if len(kept) == 0 && s.Error == "" {
				continue
			}
			s.Skills = kept
		}
		filtered = append(filtered, s)
	}

	if len(filtered) == 0 {
		return "No skills found matching the given filters.\n"
	}

	var b strings.Builder
	totalAvailable := 0
	totalInstalled := 0
	printedAny := false

	for _, s := range filtered {
		if len(s.Skills) == 0 && s.Error == "" {
			continue
		}
		if printedAny {
			b.WriteString("\n")
		}
		printedAny = true

		// Header
		if s.RegistryName == "embedded" {
			b.WriteString("Embedded:\n")
		} else if s.URL != "" {
			fmt.Fprintf(&b, "%s (%s):\n", s.RegistryName, s.URL)
		} else {
			fmt.Fprintf(&b, "%s:\n", s.RegistryName)
		}

		if s.Error != "" {
			fmt.Fprintf(&b, "  (error: %s)\n", s.Error)
			continue
		}

		for _, name := range s.Skills {
			totalAvailable++
			if installed[name] {
				totalInstalled++
				fmt.Fprintf(&b, "  %s  [installed]\n", name)
			} else {
				fmt.Fprintf(&b, "  %s\n", name)
			}
		}
	}

	if !printedAny {
		return "No skills found matching the given filters.\n"
	}

	fmt.Fprintf(&b, "\n%d installed, %d available\n", totalInstalled, totalAvailable)
	return b.String()
}

// --- JSON output types and functions ---

type skillsListJSON struct {
	Registries     []registryJSON `json:"registries"`
	TotalAvailable int            `json:"total_available"`
	TotalInstalled int            `json:"total_installed"`
}

type registryJSON struct {
	Name   string      `json:"name"`
	URL    string      `json:"url,omitempty"`
	Error  string      `json:"error,omitempty"`
	Skills []skillJSON `json:"skills,omitempty"`
}

type skillJSON struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// formatSkillsListJSON renders the skill sets as a JSON string.
func formatSkillsListJSON(sets []registrySkillSet, installed map[string]bool) string {
	result := skillsListJSON{}
	for _, s := range sets {
		reg := registryJSON{
			Name:  s.RegistryName,
			URL:   s.URL,
			Error: s.Error,
		}
		for _, name := range s.Skills {
			isInstalled := installed[name]
			reg.Skills = append(reg.Skills, skillJSON{Name: name, Installed: isInstalled})
			result.TotalAvailable++
			if isInstalled {
				result.TotalInstalled++
			}
		}
		result.Registries = append(result.Registries, reg)
	}
	if result.Registries == nil {
		result.Registries = []registryJSON{}
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

// --- skills show helpers ---

// buildAllSources returns the embedded registry plus any git registries
// from the merged manifest as a unified slice of SkillSource.
func buildAllSources(merged *manifest.Manifest) []registry.SkillSource {
	sources := []registry.SkillSource{registry.NewEmbeddedRegistry()}
	if merged != nil {
		sources = append(sources, gitRegistriesFromManifest(merged)...)
	}
	return sources
}

// resolveSkillFromSources searches registries in order and returns the first
// matching skill along with the registry name that provided it.
func resolveSkillFromSources(name string, sources []registry.SkillSource) (*schema.Skill, string, error) {
	for _, src := range sources {
		skill, _, err := src.Fetch(name)
		if err == nil {
			return skill, src.Name(), nil
		}
	}
	return nil, "", fmt.Errorf("skill not found: %s", name)
}

// formatSkillShow renders a single skill's details as a human-readable string.
func formatSkillShow(skill *schema.Skill, registryName, registryURL string, installed bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Name: %s\n", skill.Name)
	if skill.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", skill.Description)
	}
	if skill.Version != "" {
		fmt.Fprintf(&b, "Version: %s\n", skill.Version)
	}
	if skill.Author != "" {
		fmt.Fprintf(&b, "Author: %s\n", skill.Author)
	}
	if len(skill.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(skill.Tags, ", "))
	}

	// Registry info
	if registryURL != "" {
		fmt.Fprintf(&b, "Registry: %s (%s)\n", registryName, registryURL)
	} else {
		fmt.Fprintf(&b, "Registry: %s\n", registryName)
	}

	// Install status
	if installed {
		b.WriteString("Status: installed\n")
	} else {
		b.WriteString("Status: not installed\n")
	}

	// Instructions (SKILL.md body)
	if skill.Instructions != "" {
		b.WriteString("\n---\n\n")
		b.WriteString(skill.Instructions)
		b.WriteString("\n")
	}

	return b.String()
}

// --- Cobra commands ---

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage and discover skills",
	Long: `Browse available skills, inspect details, and manage your skill configuration.

Subcommands:
  list    List all available skills across registries
  show    Show details for a specific skill
  remove  Remove a skill from the manifest`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var (
	listRegistry      string
	listInstalledOnly bool
	listJSON          bool
)

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available skills across registries",
	Long: `Lists skills from the embedded registry and any git registries
configured in your global or local vibes.yaml.

Skills already in your manifest are marked with [installed].`,
	Run: func(cmd *cobra.Command, args []string) {
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()

		// Load merged manifest (global + local) for registries and installed skills.
		merged, _ := manifest.LoadMergedManifest(project, globalPath)

		sets := collectSkillSets(merged)

		// Build installed set from merged manifest.
		installed := make(map[string]bool)
		if merged != nil {
			for _, s := range merged.Skills {
				installed[s.Name] = true
			}
		}

		if listJSON {
			fmt.Println(formatSkillsListJSON(sets, installed))
			return
		}

		opts := listFormatOptions{
			Registry:      listRegistry,
			InstalledOnly: listInstalledOnly,
		}
		fmt.Print(formatSkillsListFiltered(sets, installed, opts))
	},
}

var skillsShowCmd = &cobra.Command{
	Use:   "show <skill-name>",
	Short: "Show details for a specific skill",
	Long: `Displays metadata (name, description, version, author, tags),
the source registry, install status, and the full skill content.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		project := ProjectDir()
		globalPath := defaultGlobalManifestPath()

		merged, _ := manifest.LoadMergedManifest(project, globalPath)

		sources := buildAllSources(merged)
		skill, regName, err := resolveSkillFromSources(name, sources)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		// Determine registry URL if available
		var regURL string
		if merged != nil {
			for _, r := range merged.Registries {
				if r.Name == regName {
					regURL = r.URL
					break
				}
			}
		}

		// Check if installed
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
	},
}

var skillsRemoveCmd = &cobra.Command{
	Use:   "remove <skill-name>",
	Short: "Remove a skill from the manifest",
	Long: `Removes the named skill from the project manifest (vibes.yaml).
Does not delete local skill files.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		project := ProjectDir()

		// Find existing manifest
		_, manifestPath, findErr := manifest.LoadManifestFromProject(project)
		if findErr != nil {
			fmt.Printf("error: no manifest found in %s\n", project)
			return
		}

		inst := engine.NewInstaller(nil)
		if err := inst.Remove(name, manifestPath); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		fmt.Printf("Removed '%s' from %s\n", name, filepath.Base(manifestPath))
	},
}

func init() {
	// List flags
	skillsListCmd.Flags().StringVar(&listRegistry, "registry", "", "filter skills by registry name")
	skillsListCmd.Flags().BoolVar(&listInstalledOnly, "installed-only", false, "show only installed skills")
	skillsListCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")

	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsShowCmd)
	skillsCmd.AddCommand(skillsRemoveCmd)
	rootCmd.AddCommand(skillsCmd)
}
