package cli

import (
	"fmt"
	"strings"
)

// ResourceType identifies which manifest resource type a command operates on.
type ResourceType string

const (
	ResourceSkills       ResourceType = "skills"
	ResourceAgents       ResourceType = "agents"
	ResourceInstructions ResourceType = "instructions"
	ResourcePrompts      ResourceType = "prompts"
	ResourceTargets      ResourceType = "targets"
	ResourceRegistries   ResourceType = "registries"
)

// ValidResourceTypes returns the list of supported resource type strings.
func ValidResourceTypes() []string {
	return []string{string(ResourceSkills), string(ResourceAgents), string(ResourceInstructions), string(ResourcePrompts), string(ResourceTargets), string(ResourceRegistries)}
}

// ParseResourceType validates and returns a ResourceType from a string.
func ParseResourceType(s string) (ResourceType, error) {
	switch ResourceType(s) {
	case ResourceSkills, ResourceAgents, ResourceInstructions, ResourcePrompts, ResourceTargets, ResourceRegistries:
		return ResourceType(s), nil
	default:
		return "", fmt.Errorf("unknown resource type %q (valid: %s)", s, strings.Join(ValidResourceTypes(), ", "))
	}
}

// ResourceItem is a generic item with a name and optional metadata,
// used to unify skills, agents, and instructions for list/show/install/remove.
type ResourceItem struct {
	Name         string
	Installed    bool
	InstallScope string
	State        string
}

const (
	installScopeNone   = "none"
	installScopeLocal  = "local"
	installScopeGlobal = "global"
	installScopeBoth   = "both"
)

// ResourceDetailResult describes a fully-resolved resource for show operations.
type ResourceDetailResult struct {
	Kind        ResourceType
	Name        string
	Installed   bool
	Registry    string
	RegistryURL string
	Path        string
	Payload     any
}

type registryResourceItem struct {
	Name     string
	Registry string
	Path     string
}

// registrySkillSet holds the skills available from a single registry source.
type registrySkillSet struct {
	RegistryName string
	URL          string // empty for embedded
	Skills       []string
	Error        string // non-empty if listing failed
}

// listFormatOptions controls filtering for the list output.
type listFormatOptions struct {
	Registry      string // filter to a specific registry name
	InstalledOnly bool   // show only installed skills
}

// --- JSON output types ---

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

type resourceListJSON struct {
	Type  string             `json:"type"`
	Items []resourceItemJSON `json:"items"`
	Total int                `json:"total"`
}

type resourceItemJSON struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}
