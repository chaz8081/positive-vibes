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

// FileSource extends SkillSource with raw file access into skill directories.
// Registries that support fetching arbitrary files (e.g., agent definitions)
// should implement this interface.
type FileSource interface {
	SkillSource
	// FetchFile retrieves raw file bytes from a skill directory.
	// skillName is the skill directory name; relPath is the path relative to
	// the skill directory (e.g., "agents/reviewer.md").
	FetchFile(skillName, relPath string) ([]byte, error)
	// ListFiles returns the names of files directly within a subdirectory of
	// a skill directory. Returns an empty slice if the directory does not exist.
	ListFiles(skillName, relDir string) ([]string, error)
}
