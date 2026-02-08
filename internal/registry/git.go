package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/chaz8081/positive-vibes/pkg/schema"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitRegistry fetches skills from a remote (or local) git repository.
// It clones the repo on first use into CachePath and reads skills from
// the SkillsPath subdirectory within the cloned worktree.
type GitRegistry struct {
	RegistryName string
	URL          string
	CachePath    string // directory to clone into; e.g., ~/.positive-vibes/cache/<name>/
	SkillsPath   string // subdirectory inside the repo where skills live; defaults to "."
	Ref          string // "latest", branch name, tag name, or commit SHA
}

const RefLatest = "latest"

var shaPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

func isSHA(ref string) bool {
	return shaPattern.MatchString(ref)
}

func (r *GitRegistry) isPinned() bool {
	return r.Ref != "" && r.Ref != RefLatest
}

func (r *GitRegistry) Name() string { return r.RegistryName }

// authMethod returns the appropriate transport.AuthMethod for the URL.
// For SSH URLs (git@... or ssh://...), it attempts to use the system SSH agent.
// For HTTPS or local paths, no auth is needed.
func (r *GitRegistry) authMethod() transport.AuthMethod {
	if isSSHURL(r.URL) {
		auth, err := gitssh.NewSSHAgentAuth("git")
		if err == nil {
			return auth
		}
	}
	return nil
}

// isSSHURL returns true if the URL looks like an SSH git URL.
func isSSHURL(url string) bool {
	return strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://")
}

// ensureCache clones the repository into CachePath if it does not already exist.
// If the clone fails but a cached copy already exists, it silently returns nil
// so callers can continue with stale data.
func (r *GitRegistry) ensureCache() error {
	if _, err := os.Stat(filepath.Join(r.CachePath, ".git")); err == nil {
		// Cache already populated.
		return nil
	}

	cloneOpts := &git.CloneOptions{
		URL:  r.URL,
		Auth: r.authMethod(),
	}

	var repo *git.Repository
	var err error

	// For non-SHA pinned refs, attempt branch clone first (single-branch), then tag.
	if r.isPinned() && !isSHA(r.Ref) {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(r.Ref)
		cloneOpts.SingleBranch = true
	}

	repo, err = git.PlainClone(r.CachePath, false, cloneOpts)
	if err != nil {
		if r.isPinned() && !isSHA(r.Ref) {
			// try tag
			_ = os.RemoveAll(r.CachePath)
			cloneOpts.ReferenceName = plumbing.NewTagReferenceName(r.Ref)
			cloneOpts.SingleBranch = true
			repo, err = git.PlainClone(r.CachePath, false, cloneOpts)
		}
		if err != nil {
			// If we somehow have a partial cache, allow fallback.
			if _, statErr := os.Stat(r.CachePath); statErr == nil {
				return nil
			}
			if r.isPinned() && !isSHA(r.Ref) {
				return fmt.Errorf("registry %q: ref %q not found as branch or tag in %s", r.RegistryName, r.Ref, r.URL)
			}
			return fmt.Errorf("git clone %s: %w", r.URL, err)
		}
	}

	// If pinned to a commit SHA, checkout that hash.
	if r.isPinned() && isSHA(r.Ref) {
		// Ensure repo is cloned (full clone)
		if repo == nil {
			repo, err = git.PlainClone(r.CachePath, false, &git.CloneOptions{
				URL:  r.URL,
				Auth: r.authMethod(),
			})
			if err != nil {
				if _, statErr := os.Stat(r.CachePath); statErr == nil {
					return nil
				}
				return fmt.Errorf("git clone %s: %w", r.URL, err)
			}
		}

		wt, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("worktree: %w", err)
		}
		err = wt.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(r.Ref),
		})
		if err != nil {
			return fmt.Errorf("registry %q: commit %s not found: %w", r.RegistryName, r.Ref, err)
		}
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

// FetchFile retrieves raw file bytes from a skill directory in the registry.
// skillName is the skill directory name; relPath is the path relative to the
// skill directory (e.g., "instructions/setup.md").
func (r *GitRegistry) FetchFile(skillName, relPath string) ([]byte, error) {
	if err := r.ensureCache(); err != nil {
		return nil, err
	}

	path := filepath.Join(r.skillsDir(), skillName, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s/%s (registry %s)", skillName, relPath, r.RegistryName)
	}
	return data, nil
}

// ListFiles returns the names of files directly within a subdirectory of a
// skill directory. It does not recurse into nested subdirectories.
// Returns an empty slice (not an error) if the directory does not exist.
func (r *GitRegistry) ListFiles(skillName, relDir string) ([]string, error) {
	if err := r.ensureCache(); err != nil {
		return nil, err
	}

	dir := filepath.Join(r.skillsDir(), skillName, relDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s/%s: %w", skillName, relDir, err)
	}

	var names []string
	for _, ent := range entries {
		if !ent.IsDir() {
			names = append(names, ent.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Refresh pulls the latest changes from the remote into the cached worktree.
// If the cache does not exist yet, it clones instead.
// For pinned refs (anything other than "latest" or empty), refresh is a no-op
// since the cached checkout already has the correct content.
func (r *GitRegistry) Refresh() error {
	// Pinned refs don't need refreshing -- the cached checkout is correct.
	// If the cache is missing, ensureCache will re-clone at the pinned ref.
	if r.isPinned() {
		return r.ensureCache()
	}

	// "latest" (or empty): pull to update.
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

	err = wt.Pull(&git.PullOptions{
		Auth: r.authMethod(),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("git pull: %w", err)
	}

	return nil
}
