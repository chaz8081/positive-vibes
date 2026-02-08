package registry

import "github.com/chaz8081/positive-vibes/pkg/schema"

// SkillSource abstracts where skills come from.
type SkillSource interface {
	// Name returns the registry name.
	Name() string
	// Fetch retrieves a skill by name.
	// Returns the parsed skill and the path to the skill's source directory.
	Fetch(name string) (*schema.Skill, string, error)
	// List returns all available skill names.
	List() ([]string, error)
}
