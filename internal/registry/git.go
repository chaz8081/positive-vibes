package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/chaz8081/positive-vibes/pkg/schema"
	git "github.com/go-git/go-git/v5"
)

// GitRegistry fetches skills from a remote (or local) git repository.
// It clones the repo on first use into CachePath and reads skills from
// the SkillsPath subdirectory within the cloned worktree.
type GitRegistry struct {
	RegistryName string
	URL          string
	CachePath    string // directory to clone into; e.g., ~/.positive-vibes/cache/<name>/
	SkillsPath   string // subdirectory inside the repo where skills live; defaults to "."
}

func (r *GitRegistry) Name() string { return r.RegistryName }

// ensureCache clones the repository into CachePath if it does not already exist.
// If the clone fails but a cached copy already exists, it silently returns nil
// so callers can continue with stale data.
func (r *GitRegistry) ensureCache() error {
	if _, err := os.Stat(filepath.Join(r.CachePath, ".git")); err == nil {
		// Cache already populated.
		return nil
	}

	_, err := git.PlainClone(r.CachePath, false, &git.CloneOptions{
		URL: r.URL,
	})
	if err != nil {
		// If we somehow have a partial cache, allow fallback.
		if _, statErr := os.Stat(r.CachePath); statErr == nil {
			return nil
		}
		return fmt.Errorf("git clone %s: %w", r.URL, err)
	}
	return nil
}

// skillsDir returns the absolute path to the directory containing skills.
func (r *GitRegistry) skillsDir() string {
	sp := r.SkillsPath
	if sp == "" || sp == "." {
		return r.CachePath
	}
	return filepath.Join(r.CachePath, sp)
}

// Fetch retrieves a skill by name.
// It returns the parsed Skill and the path to the skill's source directory on disk.
func (r *GitRegistry) Fetch(name string) (*schema.Skill, string, error) {
	if err := r.ensureCache(); err != nil {
		return nil, "", err
	}

	srcDir := filepath.Join(r.skillsDir(), name)
	skillFile := filepath.Join(srcDir, "SKILL.md")

	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, "", fmt.Errorf("skill not found: %s (registry %s)", name, r.RegistryName)
	}

	sk, err := schema.ParseSkillFile(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse skill %s: %w", name, err)
	}

	return sk, srcDir, nil
}

// List returns all available skill names (directories containing a SKILL.md).
func (r *GitRegistry) List() ([]string, error) {
	if err := r.ensureCache(); err != nil {
		return nil, err
	}

	base := r.skillsDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	var names []string
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(base, ent.Name(), "SKILL.md")); err == nil {
			names = append(names, ent.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Refresh pulls the latest changes from the remote into the cached worktree.
// If the cache does not exist yet, it clones instead.
func (r *GitRegistry) Refresh() error {
	// If no cache yet, just clone.
	if _, err := os.Stat(filepath.Join(r.CachePath, ".git")); err != nil {
		return r.ensureCache()
	}

	repo, err := git.PlainOpen(r.CachePath)
	if err != nil {
		return fmt.Errorf("open cached repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	err = wt.Pull(&git.PullOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("git pull: %w", err)
	}

	return nil
}
