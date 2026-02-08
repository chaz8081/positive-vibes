# Registry Versioning Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a required `ref` field to registry entries so users can pin registries to a specific branch, tag, or SHA -- or explicitly opt into "latest" (track default branch).

**Architecture:** Add `Ref` field to `RegistryRef` manifest type and `GitRegistry` struct. Clone behavior branches on the ref value: "latest" preserves current behavior (default branch + pull on refresh); any other value clones and checks out that specific ref, with refresh becoming a no-op. Auto-detect whether a non-"latest" ref is a SHA (hex string 7-40 chars) vs. branch/tag name.

**Tech Stack:** Go, go-git/v5 (already in go.mod), cobra, testify

---

### Task 1: Add `Ref` field to manifest `RegistryRef` and validate it

**Files:**
- Modify: `internal/manifest/manifest.go:33-38` (RegistryRef struct)
- Modify: `internal/manifest/manifest.go:80-94` (Validate method)
- Test: `internal/manifest/manifest_test.go`

**Step 1: Write the failing tests**

Add to `internal/manifest/manifest_test.go`:

```go
func TestValidate_RegistryMissingRef(t *testing.T) {
	m := &Manifest{
		Registries: []RegistryRef{{Name: "r", URL: "https://example.com"}},
		Skills:     []SkillRef{{Name: "x"}},
		Targets:    []string{"opencode"},
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ref")
}

func TestValidate_RegistryWithRef(t *testing.T) {
	m := &Manifest{
		Registries: []RegistryRef{{Name: "r", URL: "https://example.com", Ref: "latest"}},
		Skills:     []SkillRef{{Name: "x"}},
		Targets:    []string{"opencode"},
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestLoadManifest_RefField(t *testing.T) {
	yamlStr := `registries:
  - name: pinned
    url: https://example.com/repo
    ref: v1.2.0
skills:
  - name: s
targets:
  - opencode
`
	m, err := LoadManifestFromBytes([]byte(yamlStr))
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", m.Registries[0].Ref)
}

func TestSaveManifest_RefFieldRoundTrip(t *testing.T) {
	m := &Manifest{
		Registries: []RegistryRef{{Name: "r", URL: "https://r", Ref: "abc123"}},
		Skills:     []SkillRef{{Name: "s"}},
		Targets:    []string{"opencode"},
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "vibes.yml")

	require.NoError(t, SaveManifest(m, p))

	m2, err := LoadManifest(p)
	require.NoError(t, err)
	assert.Equal(t, "abc123", m2.Registries[0].Ref)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/manifest/ -run "TestValidate_RegistryMissingRef|TestValidate_RegistryWithRef|TestLoadManifest_RefField|TestSaveManifest_RefFieldRoundTrip" -v`

Expected: compilation error (Ref field doesn't exist on RegistryRef) or test failures.

**Step 3: Implement the changes**

In `internal/manifest/manifest.go`:

1. Add `Ref` field to `RegistryRef`:

```go
type RegistryRef struct {
	Name  string            `yaml:"name"`
	URL   string            `yaml:"url"`
	Ref   string            `yaml:"ref"`
	Paths map[string]string `yaml:"paths,omitempty"`
}
```

2. Add validation in `Validate()` -- after the existing target validation loop, add:

```go
for _, r := range m.Registries {
	if r.Ref == "" {
		return fmt.Errorf("registry %q must specify a ref (use \"latest\" to track the default branch)", r.Name)
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/manifest/ -v`

Expected: all tests pass, including new ones.

**Step 5: Fix existing tests that create RegistryRef without Ref**

Several existing tests create `RegistryRef` values without `Ref`. These still compile (empty string is zero value) but any that call `Validate()` on manifests with registries will now fail. Update:

- `TestSaveManifest` (line 108): add `Ref: "latest"` to the RegistryRef
- `TestLoadMergedManifest_*` tests that include registries: add `ref: latest` to YAML strings
- `exampleYAML` constant (line 12): add `ref: latest` under the registry entry

Run: `go test ./internal/manifest/ -v`

Expected: all pass.

**Step 6: Commit**

```
git add internal/manifest/manifest.go internal/manifest/manifest_test.go
git commit -m "feat(manifest): add required ref field to RegistryRef for version pinning"
```

---

### Task 2: Add `Ref` field to `GitRegistry` and implement ref-aware clone

**Files:**
- Modify: `internal/registry/git.go:16-24` (GitRegistry struct)
- Modify: `internal/registry/git.go:46-67` (ensureCache)
- Test: `internal/registry/git_test.go`

**Step 1: Write the failing tests**

Add to `internal/registry/git_test.go`:

```go
func TestGitRegistry_Fetch_WithTagRef(t *testing.T) {
	// Create repo with two commits; tag the first
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"v1-skill": "---\nname: v1-skill\n---\n# V1 Skill\n",
	})

	// Tag the current commit
	run := makeGitRunner(t, repoDir)
	run("tag", "v1.0.0")

	// Add a second skill on HEAD (after the tag)
	newDir := filepath.Join(repoDir, "v2-skill")
	require.NoError(t, os.MkdirAll(newDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "SKILL.md"),
		[]byte("---\nname: v2-skill\n---\n# V2 Skill\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add v2-skill")

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "tag-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "tag-reg"),
		SkillsPath:   ".",
		Ref:          "v1.0.0",
	}

	// Should find v1-skill (exists at tag)
	sk, _, err := reg.Fetch("v1-skill")
	require.NoError(t, err)
	assert.Equal(t, "v1-skill", sk.Name)

	// Should NOT find v2-skill (added after tag)
	_, _, err = reg.Fetch("v2-skill")
	require.Error(t, err)
}

func TestGitRegistry_Fetch_WithBranchRef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"main-skill": "---\nname: main-skill\n---\n# Main\n",
	})

	// Create a feature branch with an extra skill
	run := makeGitRunner(t, repoDir)
	run("checkout", "-b", "feature")
	featureDir := filepath.Join(repoDir, "feature-skill")
	require.NoError(t, os.MkdirAll(featureDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "SKILL.md"),
		[]byte("---\nname: feature-skill\n---\n# Feature\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add feature-skill")
	run("checkout", "main") // switch source repo back to main

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "branch-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "branch-reg"),
		SkillsPath:   ".",
		Ref:          "feature",
	}

	// Should find feature-skill (on the feature branch)
	sk, _, err := reg.Fetch("feature-skill")
	require.NoError(t, err)
	assert.Equal(t, "feature-skill", sk.Name)
}

func TestGitRegistry_Fetch_WithSHARef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"original": "---\nname: original\n---\n# Original\n",
	})

	// Grab the SHA of the first commit
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	shaBytes, err := cmd.Output()
	require.NoError(t, err)
	sha := strings.TrimSpace(string(shaBytes))

	// Add another commit on top
	run := makeGitRunner(t, repoDir)
	laterDir := filepath.Join(repoDir, "later-skill")
	require.NoError(t, os.MkdirAll(laterDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(laterDir, "SKILL.md"),
		[]byte("---\nname: later-skill\n---\n# Later\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add later-skill")

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "sha-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "sha-reg"),
		SkillsPath:   ".",
		Ref:          sha,
	}

	// Should find original (at pinned SHA)
	sk, _, err := reg.Fetch("original")
	require.NoError(t, err)
	assert.Equal(t, "original", sk.Name)

	// Should NOT find later-skill
	_, _, err = reg.Fetch("later-skill")
	require.Error(t, err)
}

func TestGitRegistry_Fetch_LatestRef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"skill-a": "---\nname: skill-a\n---\n# A\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "latest-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "latest-reg"),
		SkillsPath:   ".",
		Ref:          "latest",
	}

	sk, _, err := reg.Fetch("skill-a")
	require.NoError(t, err)
	assert.Equal(t, "skill-a", sk.Name)
}
```

Also add a `makeGitRunner` helper to replace the inline `run` closures:

```go
func makeGitRunner(t *testing.T, repoDir string) func(args ...string) {
	t.Helper()
	return func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/registry/ -run "TestGitRegistry_Fetch_With|TestGitRegistry_Fetch_LatestRef" -v`

Expected: compilation error (Ref field doesn't exist) or behavioral failures.

**Step 3: Implement ref-aware clone**

In `internal/registry/git.go`:

1. Add `Ref` field and helper constants:

```go
const RefLatest = "latest"

type GitRegistry struct {
	RegistryName string
	URL          string
	CachePath    string
	SkillsPath   string
	Ref          string // "latest", branch name, tag name, or commit SHA
}
```

2. Add SHA detection helper:

```go
import "regexp"

var shaPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

func isSHA(ref string) bool {
	return shaPattern.MatchString(ref)
}

func (r *GitRegistry) isPinned() bool {
	return r.Ref != "" && r.Ref != RefLatest
}
```

3. Update `ensureCache()`:

```go
func (r *GitRegistry) ensureCache() error {
	if _, err := os.Stat(filepath.Join(r.CachePath, ".git")); err == nil {
		return nil
	}

	cloneOpts := &git.CloneOptions{
		URL:  r.URL,
		Auth: r.authMethod(),
	}

	// For non-SHA branch/tag refs, we can set ReferenceName to clone only that ref
	if r.isPinned() && !isSHA(r.Ref) {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(r.Ref)
		cloneOpts.SingleBranch = true
	}

	repo, err := git.PlainClone(r.CachePath, false, cloneOpts)
	if err != nil {
		// If branch ref failed, retry as tag
		if r.isPinned() && !isSHA(r.Ref) {
			_ = os.RemoveAll(r.CachePath)
			cloneOpts.ReferenceName = plumbing.NewTagReferenceName(r.Ref)
			repo, err = git.PlainClone(r.CachePath, false, cloneOpts)
		}
		if err != nil {
			if _, statErr := os.Stat(r.CachePath); statErr == nil {
				return nil
			}
			return fmt.Errorf("git clone %s: %w", r.URL, err)
		}
	}

	// For SHA refs, checkout the specific commit after cloning
	if r.isPinned() && isSHA(r.Ref) {
		wt, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("worktree: %w", err)
		}
		err = wt.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(r.Ref),
		})
		if err != nil {
			return fmt.Errorf("checkout %s: %w", r.Ref, err)
		}
	}

	return nil
}
```

New imports needed: `"github.com/go-git/go-git/v5/plumbing"`, `"regexp"`

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/registry/ -run "TestGitRegistry_Fetch_With|TestGitRegistry_Fetch_LatestRef" -v`

Expected: all pass.

**Step 5: Verify existing tests still pass**

Run: `go test ./internal/registry/ -v`

Expected: all existing tests pass (empty Ref treated as latest).

**Step 6: Commit**

```
git add internal/registry/git.go internal/registry/git_test.go
git commit -m "feat(registry): implement ref-aware clone for branch, tag, and SHA pinning"
```

---

### Task 3: Implement no-op refresh for pinned refs

**Files:**
- Modify: `internal/registry/git.go:126-152` (Refresh method)
- Test: `internal/registry/git_test.go`

**Step 1: Write the failing tests**

Add to `internal/registry/git_test.go`:

```go
func TestGitRegistry_Refresh_PinnedRef_NoOp(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"pinned-skill": "---\nname: pinned-skill\n---\n# Pinned\n",
	})

	run := makeGitRunner(t, repoDir)
	run("tag", "v1.0.0")

	// Add a new skill after the tag
	newDir := filepath.Join(repoDir, "new-skill")
	require.NoError(t, os.MkdirAll(newDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "SKILL.md"),
		[]byte("---\nname: new-skill\n---\n# New\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add new-skill")

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "pinned-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "pinned-reg"),
		SkillsPath:   ".",
		Ref:          "v1.0.0",
	}

	// Initial fetch
	_, _, err := reg.Fetch("pinned-skill")
	require.NoError(t, err)

	// Refresh should be a no-op for pinned
	require.NoError(t, reg.Refresh())

	// Should still NOT see the new skill (pinned to tag)
	_, _, err = reg.Fetch("new-skill")
	require.Error(t, err)

	// Original skill still available
	sk, _, err := reg.Fetch("pinned-skill")
	require.NoError(t, err)
	assert.Equal(t, "pinned-skill", sk.Name)
}

func TestGitRegistry_Refresh_LatestRef_PullsUpdates(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"original-skill": "---\nname: original-skill\n---\n# Original\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "latest-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "latest-reg"),
		SkillsPath:   ".",
		Ref:          "latest",
	}

	// Initial fetch
	_, _, err := reg.Fetch("original-skill")
	require.NoError(t, err)

	// Add new skill to remote
	run := makeGitRunner(t, repoDir)
	newDir := filepath.Join(repoDir, "added-skill")
	require.NoError(t, os.MkdirAll(newDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "SKILL.md"),
		[]byte("---\nname: added-skill\n---\n# Added\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add skill")

	// Before refresh: not available
	_, _, err = reg.Fetch("added-skill")
	require.Error(t, err)

	// Refresh (should pull)
	require.NoError(t, reg.Refresh())

	// After refresh: available
	sk, _, err := reg.Fetch("added-skill")
	require.NoError(t, err)
	assert.Equal(t, "added-skill", sk.Name)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/registry/ -run "TestGitRegistry_Refresh_PinnedRef|TestGitRegistry_Refresh_LatestRef" -v`

Expected: `TestGitRegistry_Refresh_PinnedRef_NoOp` fails because current Refresh() pulls and updates worktree.

**Step 3: Update Refresh()**

In `internal/registry/git.go`, replace the `Refresh` method:

```go
func (r *GitRegistry) Refresh() error {
	// Pinned refs don't need refreshing -- the cached checkout is correct.
	// If the cache is missing, ensureCache will re-clone at the pinned ref.
	if r.isPinned() {
		return r.ensureCache()
	}

	// "latest" (or empty): pull to update
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/registry/ -v`

Expected: all pass, including new refresh tests and all existing tests.

**Step 5: Commit**

```
git add internal/registry/git.go internal/registry/git_test.go
git commit -m "feat(registry): no-op refresh for pinned refs, pull only for latest"
```

---

### Task 4: Wire `Ref` through CLI and update example manifest

**Files:**
- Modify: `internal/cli/root.go:83-95` (gitRegistriesFromManifest)
- Modify: `vibes.yml` (add ref: latest to example)

**Step 1: Update `gitRegistriesFromManifest`**

In `internal/cli/root.go`, pass `Ref` when constructing `GitRegistry`:

```go
func gitRegistriesFromManifest(m *manifest.Manifest) []registry.SkillSource {
	var sources []registry.SkillSource
	for _, r := range m.Registries {
		sources = append(sources, &registry.GitRegistry{
			RegistryName: r.Name,
			URL:          r.URL,
			CachePath:    defaultCachePath(r.Name),
			SkillsPath:   r.SkillsPath(),
			Ref:          r.Ref,
		})
	}
	return sources
}
```

**Step 2: Update `vibes.yml`**

```yaml
registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
    paths:
      skills: skills/
```

**Step 3: Run full test suite**

Run: `go test ./... -v`

Expected: all pass.

**Step 4: Build binary**

Run: `go build ./cmd/positive-vibes/`

Expected: success.

**Step 5: Commit**

```
git add internal/cli/root.go vibes.yml
git commit -m "feat(cli): wire registry ref field through to GitRegistry"
```

---

### Task 5: Update existing tests that construct RegistryRef without Ref

**Files:**
- Modify: `internal/manifest/manifest_test.go` (update exampleYAML, TestSaveManifest, merged manifest tests)
- Modify: `internal/cli/config_test.go` (if it constructs RegistryRef)
- Modify: `internal/registry/registry_test.go` (if tests pass URL-only registries through manifest)

**Step 1: Search for all RegistryRef literals missing Ref**

Look for `RegistryRef{` without `Ref:` in all test files. Update each to include `Ref: "latest"` (or the appropriate value for the test scenario).

Key locations:
- `exampleYAML` in manifest_test.go: add `ref: latest` under the registry
- `TestSaveManifest`: add `Ref: "latest"` to RegistryRef literal
- `TestLoadMergedManifest_*` tests: add `ref: latest` to YAML strings containing registries

**Step 2: Run full suite**

Run: `go test ./... -v`

Expected: all pass.

**Step 3: Commit**

```
git add -u
git commit -m "test: update existing tests to include required ref field on registries"
```

---

### Task 6: Final verification

**Step 1: Run full test suite**

Run: `go test ./... -v`

Expected: all pass.

**Step 2: Build binary**

Run: `go build ./cmd/positive-vibes/`

Expected: success.

**Step 3: Verify with go vet**

Run: `go vet ./...`

Expected: no issues.
