package registry

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestGitRepo creates a local git repo with skills committed.
// Returns the path to the repo (usable as a URL for go-git).
func setupTestGitRepo(t *testing.T, skillsDir string, skills map[string]string) string {
	t.Helper()
	repoDir := t.TempDir()

	run := func(args ...string) {
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

	run("init")
	run("checkout", "-b", "main")

	// Write skill files
	for name, content := range skills {
		dir := filepath.Join(repoDir, skillsDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	run("add", ".")
	run("commit", "-m", "initial commit")

	return repoDir
}

func makeGitRunner(t *testing.T, repoDir string) func(args ...string) string {
	t.Helper()
	return func(args ...string) string {
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
		return strings.TrimSpace(string(out))
	}
}

func TestGitRegistry_Fetch(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"test-skill": "---\nname: test-skill\ndescription: A test skill\nversion: \"1.0\"\nauthor: test\ntags: [test]\n---\n# Test Skill\n\nDo the thing.\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "test-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "test-reg"),
		SkillsPath:   ".",
	}

	sk, srcDir, err := reg.Fetch("test-skill")
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if sk.Name != "test-skill" {
		t.Fatalf("expected name 'test-skill', got %q", sk.Name)
	}
	if sk.Description != "A test skill" {
		t.Fatalf("expected description 'A test skill', got %q", sk.Description)
	}
	if srcDir == "" {
		t.Fatalf("expected non-empty srcDir")
	}
	// Verify SKILL.md exists at srcDir
	if _, err := os.Stat(filepath.Join(srcDir, "SKILL.md")); err != nil {
		t.Fatalf("SKILL.md not found at srcDir: %v", err)
	}
}

func TestGitRegistry_Fetch_NotFound(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"real-skill": "---\nname: real-skill\n---\n# Real\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "test-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "test-reg"),
		SkillsPath:   ".",
	}

	_, _, err := reg.Fetch("nonexistent-skill")
	if err == nil {
		t.Fatalf("expected error for nonexistent skill")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestGitRegistry_List(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"skill-alpha": "---\nname: skill-alpha\n---\n# Alpha\n",
		"skill-beta":  "---\nname: skill-beta\n---\n# Beta\n",
		"skill-gamma": "---\nname: skill-gamma\n---\n# Gamma\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "test-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "test-reg"),
		SkillsPath:   ".",
	}

	names, err := reg.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 skills, got %d: %v", len(names), names)
	}
	for _, want := range []string{"skill-alpha", "skill-beta", "skill-gamma"} {
		if !containsSlice(names, want) {
			t.Fatalf("missing %q in %v", want, names)
		}
	}
}

func TestGitRegistry_Fetch_CustomSkillsPath(t *testing.T) {
	// Skills are in a subdirectory, not repo root
	repoDir := setupTestGitRepo(t, "resources/skills", map[string]string{
		"nested-skill": "---\nname: nested-skill\ndescription: Lives in a subdir\nversion: \"1.0\"\nauthor: test\ntags: [nested]\n---\n# Nested\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "nested-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "nested-reg"),
		SkillsPath:   "resources/skills",
	}

	sk, _, err := reg.Fetch("nested-skill")
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if sk.Name != "nested-skill" {
		t.Fatalf("expected 'nested-skill', got %q", sk.Name)
	}

	// List should also work with custom path
	names, err := reg.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(names) != 1 || names[0] != "nested-skill" {
		t.Fatalf("expected [nested-skill], got %v", names)
	}
}

func TestGitRegistry_CachePersistence(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"cached-skill": "---\nname: cached-skill\n---\n# Cached\n",
	})

	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "persist-reg")

	reg := &GitRegistry{
		RegistryName: "persist-reg",
		URL:          repoDir,
		CachePath:    cachePath,
		SkillsPath:   ".",
	}

	// First fetch -- triggers clone
	_, _, err := reg.Fetch("cached-skill")
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}

	// Cache dir should now exist
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache dir should exist after first fetch: %v", err)
	}

	// Second fetch -- should use cache (no network needed)
	// Change URL to something invalid to prove cache is used
	reg2 := &GitRegistry{
		RegistryName: "persist-reg",
		URL:          "/nonexistent/repo/path",
		CachePath:    cachePath,
		SkillsPath:   ".",
	}

	sk, _, err := reg2.Fetch("cached-skill")
	if err != nil {
		t.Fatalf("second fetch from cache: %v", err)
	}
	if sk.Name != "cached-skill" {
		t.Fatalf("expected 'cached-skill', got %q", sk.Name)
	}
}

func TestGitRegistry_Refresh(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"original-skill": "---\nname: original-skill\n---\n# Original\n",
	})

	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "refresh-reg")

	reg := &GitRegistry{
		RegistryName: "refresh-reg",
		URL:          repoDir,
		CachePath:    cachePath,
		SkillsPath:   ".",
	}

	// Initial fetch
	_, _, err := reg.Fetch("original-skill")
	if err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	// Add a new skill to the "remote" repo
	newSkillDir := filepath.Join(repoDir, "new-skill")
	if err := os.MkdirAll(newSkillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newSkillDir, "SKILL.md"), []byte("---\nname: new-skill\n---\n# New\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run := func(args ...string) {
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
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", ".")
	run("commit", "-m", "add new-skill")

	// Before refresh, new-skill should NOT be available
	_, _, err = reg.Fetch("new-skill")
	if err == nil {
		t.Fatalf("expected new-skill to not exist before refresh")
	}

	// Refresh
	if err := reg.Refresh(); err != nil {
		t.Fatalf("refresh error: %v", err)
	}

	// After refresh, new-skill should be available
	sk, _, err := reg.Fetch("new-skill")
	if err != nil {
		t.Fatalf("fetch after refresh: %v", err)
	}
	if sk.Name != "new-skill" {
		t.Fatalf("expected 'new-skill', got %q", sk.Name)
	}
}

func TestGitRegistry_Fetch_WithTagRef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"v1-skill": "---\nname: v1-skill\n---\n# V1 Skill\n",
	})

	run := makeGitRunner(t, repoDir)
	run("tag", "v1.0.0")

	// Add a second skill after the tag
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

	run := makeGitRunner(t, repoDir)
	run("checkout", "-b", "feature")

	featureDir := filepath.Join(repoDir, "feature-skill")
	require.NoError(t, os.MkdirAll(featureDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "SKILL.md"),
		[]byte("---\nname: feature-skill\n---\n# Feature\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add feature-skill")
	run("checkout", "main") // switch source repo back

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

	// Should also find main-skill (feature branch has it too, inherited from main)
	sk2, _, err := reg.Fetch("main-skill")
	require.NoError(t, err)
	assert.Equal(t, "main-skill", sk2.Name)
}

func TestGitRegistry_Fetch_WithSHARef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"original": "---\nname: original\n---\n# Original\n",
	})

	// Grab the SHA of the first commit
	run := makeGitRunner(t, repoDir)
	sha := run("rev-parse", "HEAD")

	// Add another commit on top
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

	// Initial fetch (clones at tag)
	_, _, err := reg.Fetch("pinned-skill")
	require.NoError(t, err)

	// Refresh should be a no-op for pinned ref
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

// --- Edge case tests ---

func TestGitRegistry_Fetch_NonexistentBranchOrTagRef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"skill-a": "---\nname: skill-a\n---\n# A\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "bad-ref-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "bad-ref-reg"),
		SkillsPath:   ".",
		Ref:          "nonexistent-branch-or-tag",
	}

	_, _, err := reg.Fetch("skill-a")
	require.Error(t, err)
	// Error should mention the ref and registry name so users know what to fix
	assert.Contains(t, err.Error(), "nonexistent-branch-or-tag")
	assert.Contains(t, err.Error(), "bad-ref-reg")
}

func TestGitRegistry_Fetch_NonexistentSHARef(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"skill-a": "---\nname: skill-a\n---\n# A\n",
	})

	cacheDir := t.TempDir()
	// A valid hex string that doesn't correspond to any commit
	fakeSHA := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	reg := &GitRegistry{
		RegistryName: "bad-sha-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "bad-sha-reg"),
		SkillsPath:   ".",
		Ref:          fakeSHA,
	}

	_, _, err := reg.Fetch("skill-a")
	require.Error(t, err)
	// Error should mention the SHA and registry name
	assert.Contains(t, err.Error(), fakeSHA)
	assert.Contains(t, err.Error(), "bad-sha-reg")
}

// setupTestGitRepoWithFiles creates a local git repo with arbitrary files committed.
// files is a map of relative paths (e.g., "skill-a/instructions/setup.md") to content.
// Returns the path to the repo (usable as a URL for go-git).
func setupTestGitRepoWithFiles(t *testing.T, baseDir string, files map[string]string) string {
	t.Helper()
	repoDir := t.TempDir()

	run := func(args ...string) {
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

	run("init")
	run("checkout", "-b", "main")

	for relPath, content := range files {
		fullPath := filepath.Join(repoDir, baseDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	run("add", ".")
	run("commit", "-m", "initial commit")

	return repoDir
}

// --- FetchFile tests ---

func TestGitRegistry_FetchFile(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md":                   "---\nname: my-skill\n---\n# My Skill\n",
		"my-skill/instructions/setup.md":      "# Setup Instructions\nDo these steps.",
		"my-skill/agents/code-reviewer.md":    "# Code Reviewer Agent\nReview code carefully.",
		"my-skill/instructions/guidelines.md": "# Guidelines\nFollow these rules.",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "file-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "file-reg"),
		SkillsPath:   ".",
	}

	// FetchFile should return exact bytes for a known file
	data, err := reg.FetchFile("my-skill", "instructions/setup.md")
	require.NoError(t, err)
	assert.Equal(t, "# Setup Instructions\nDo these steps.", string(data))

	// FetchFile for agent file
	data, err = reg.FetchFile("my-skill", "agents/code-reviewer.md")
	require.NoError(t, err)
	assert.Equal(t, "# Code Reviewer Agent\nReview code carefully.", string(data))
}

func TestGitRegistry_FetchFile_NotFound(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md": "---\nname: my-skill\n---\n# My Skill\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "file-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "file-reg"),
		SkillsPath:   ".",
	}

	_, err := reg.FetchFile("my-skill", "nonexistent.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGitRegistry_FetchFile_NonexistentSkill(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md": "---\nname: my-skill\n---\n# My Skill\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "file-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "file-reg"),
		SkillsPath:   ".",
	}

	_, err := reg.FetchFile("no-such-skill", "instructions/setup.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGitRegistry_FetchFile_CustomSkillsPath(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, "resources/skills", map[string]string{
		"nested-skill/SKILL.md":            "---\nname: nested-skill\n---\n# Nested\n",
		"nested-skill/instructions/run.md": "# Run Instructions\nJust run it.",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "nested-file-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "nested-file-reg"),
		SkillsPath:   "resources/skills",
	}

	data, err := reg.FetchFile("nested-skill", "instructions/run.md")
	require.NoError(t, err)
	assert.Equal(t, "# Run Instructions\nJust run it.", string(data))
}

// --- ListFiles tests ---

func TestGitRegistry_ListFiles(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md":                   "---\nname: my-skill\n---\n# My Skill\n",
		"my-skill/instructions/setup.md":      "# Setup",
		"my-skill/instructions/guidelines.md": "# Guidelines",
		"my-skill/instructions/advanced.md":   "# Advanced",
		"my-skill/agents/reviewer.md":         "# Reviewer",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "list-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "list-reg"),
		SkillsPath:   ".",
	}

	// ListFiles for instructions directory -- should return sorted relative paths
	files, err := reg.ListFiles("my-skill", "instructions")
	require.NoError(t, err)
	assert.Equal(t, []string{"advanced.md", "guidelines.md", "setup.md"}, files)

	// ListFiles for agents directory
	files, err = reg.ListFiles("my-skill", "agents")
	require.NoError(t, err)
	assert.Equal(t, []string{"reviewer.md"}, files)
}

func TestGitRegistry_ListFiles_EmptyDir(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md": "---\nname: my-skill\n---\n# My Skill\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "list-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "list-reg"),
		SkillsPath:   ".",
	}

	// ListFiles for a directory that doesn't exist should return empty list, no error
	files, err := reg.ListFiles("my-skill", "instructions")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGitRegistry_ListFiles_NonexistentSkill(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md": "---\nname: my-skill\n---\n# My Skill\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "list-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "list-reg"),
		SkillsPath:   ".",
	}

	// ListFiles for nonexistent skill should return empty list, no error
	files, err := reg.ListFiles("no-such-skill", "instructions")
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGitRegistry_ListFiles_CustomSkillsPath(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, "resources/skills", map[string]string{
		"nested/SKILL.md":              "---\nname: nested\n---\n# Nested\n",
		"nested/agents/helper.md":      "# Helper Agent",
		"nested/agents/code-review.md": "# Code Review Agent",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "nested-list-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "nested-list-reg"),
		SkillsPath:   "resources/skills",
	}

	files, err := reg.ListFiles("nested", "agents")
	require.NoError(t, err)
	assert.Equal(t, []string{"code-review.md", "helper.md"}, files)
}

func TestGitRegistry_ListFiles_NestedSubdirs(t *testing.T) {
	// ListFiles should only return files directly in the specified directory,
	// not recurse into subdirectories.
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"my-skill/SKILL.md":                        "---\nname: my-skill\n---\n# My Skill\n",
		"my-skill/instructions/top-level.md":       "# Top Level",
		"my-skill/instructions/sub/nested-file.md": "# Nested",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "nested-dir-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "nested-dir-reg"),
		SkillsPath:   ".",
	}

	// Should only get files at the top level of instructions/, not in sub/
	files, err := reg.ListFiles("my-skill", "instructions")
	require.NoError(t, err)
	assert.Equal(t, []string{"top-level.md"}, files)
}

func TestGitRegistry_ListResourceFiles_DefaultRoot(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"instructions/markdown.instructions.md": "# Markdown",
		"agents/debug.agent.md":                 "# Debug Agent",
		"skills/sample/SKILL.md":                "---\nname: sample\n---\n# Sample\n",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName: "resource-reg",
		URL:          repoDir,
		CachePath:    filepath.Join(cacheDir, "resource-reg"),
		SkillsPath:   ".",
	}

	instFiles, err := reg.ListResourceFiles("instructions")
	require.NoError(t, err)
	assert.Contains(t, instFiles, "instructions/markdown.instructions.md")

	agentFiles, err := reg.ListResourceFiles("agents")
	require.NoError(t, err)
	assert.Contains(t, agentFiles, "agents/debug.agent.md")
}

func TestGitRegistry_FetchResourceFile_WithCustomPaths(t *testing.T) {
	repoDir := setupTestGitRepoWithFiles(t, ".", map[string]string{
		"repo-skills/skill-a/SKILL.md":            "---\nname: skill-a\n---\n# A\n",
		"repo-instructions/setup.instructions.md": "Use setup checklist.",
		"repo-agents/reviewer.agent.md":           "# Reviewer",
	})

	cacheDir := t.TempDir()
	reg := &GitRegistry{
		RegistryName:     "resource-reg",
		URL:              repoDir,
		CachePath:        filepath.Join(cacheDir, "resource-reg"),
		SkillsPath:       "repo-skills",
		InstructionsPath: "repo-instructions",
		AgentsPath:       "repo-agents",
	}

	inst, err := reg.FetchResourceFile("instructions", "setup.instructions.md")
	require.NoError(t, err)
	assert.Equal(t, "Use setup checklist.", string(inst))

	agent, err := reg.FetchResourceFile("agents", "reviewer.agent.md")
	require.NoError(t, err)
	assert.Equal(t, "# Reviewer", string(agent))
}

func TestGitRegistry_Fetch_PinnedRef_FallsBackToCache(t *testing.T) {
	repoDir := setupTestGitRepo(t, ".", map[string]string{
		"cached-skill": "---\nname: cached-skill\n---\n# Cached\n",
	})

	run := makeGitRunner(t, repoDir)
	run("tag", "v1.0.0")

	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "fallback-reg")

	// First: clone successfully with the real URL
	reg := &GitRegistry{
		RegistryName: "fallback-reg",
		URL:          repoDir,
		CachePath:    cachePath,
		SkillsPath:   ".",
		Ref:          "v1.0.0",
	}
	sk, _, err := reg.Fetch("cached-skill")
	require.NoError(t, err)
	assert.Equal(t, "cached-skill", sk.Name)

	// Now: simulate network failure by pointing at invalid URL but keeping cache
	reg2 := &GitRegistry{
		RegistryName: "fallback-reg",
		URL:          "/nonexistent/broken/repo",
		CachePath:    cachePath,
		SkillsPath:   ".",
		Ref:          "v1.0.0",
	}

	// Should succeed using the existing cache
	sk2, _, err := reg2.Fetch("cached-skill")
	require.NoError(t, err)
	assert.Equal(t, "cached-skill", sk2.Name)
}
