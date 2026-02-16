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

// CleanableFetcher extends SkillSource with a Fetch variant that returns a
// cleanup function. Callers should invoke cleanup after they are done with
// the returned srcDir. For sources backed by temp directories (e.g.
// EmbeddedRegistry), cleanup removes the temp directory. For sources backed
// by persistent caches (e.g. GitRegistry), cleanup is a no-op.
type CleanableFetcher interface {
	SkillSource
	// FetchWithCleanup is like Fetch but returns an additional cleanup func.
	// The caller MUST call cleanup when done with srcDir. Cleanup is
	// idempotent and safe to call multiple times.
	FetchWithCleanup(name string) (*schema.Skill, string, func(), error)
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

// ResourceSource exposes generic file operations for resource families that are
// rooted at configurable registry base paths (skills, instructions, agents).
// kind must be one of: "skills", "instructions", "agents", "prompts".
type ResourceSource interface {
	SkillSource
	FetchResourceFile(kind, relPath string) ([]byte, error)
	ListResourceFiles(kind string) ([]string, error)
}
