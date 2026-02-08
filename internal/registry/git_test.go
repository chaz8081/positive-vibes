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
