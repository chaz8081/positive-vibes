package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedRegistry_Fetch_ConventionalCommits(t *testing.T) {
	r := NewEmbeddedRegistry()
	sk, path, err := r.Fetch("conventional-commits")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "conventional-commits" {
		t.Fatalf("expected name conventional-commits, got %s", sk.Name)
	}
	if sk.Description == "" {
		t.Fatalf("expected description to be set")
	}
	if len(sk.Tags) == 0 {
		t.Fatalf("expected tags to be set")
	}
	if sk.Instructions == "" {
		t.Fatalf("expected instructions to be set")
	}
	if path == "" {
		t.Fatalf("expected a temp path")
	}
}

func TestEmbeddedRegistry_Fetch_CodeReview(t *testing.T) {
	r := NewEmbeddedRegistry()
	sk, _, err := r.Fetch("code-review")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "code-review" {
		t.Fatalf("expected name code-review, got %s", sk.Name)
	}
	if !contains(sk.Instructions, "When reviewing code") {
		t.Fatalf("expected instructions to contain guidance")
	}
}

func TestEmbeddedRegistry_Fetch_NotFound(t *testing.T) {
	r := NewEmbeddedRegistry()
	_, _, err := r.Fetch("nonexistent-skill")
	if err == nil {
		t.Fatalf("expected error for missing skill")
	}
}

func TestEmbeddedRegistry_List(t *testing.T) {
	r := NewEmbeddedRegistry()
	names, err := r.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// expect both skills
	want := []string{"code-review", "conventional-commits"}
	if len(names) != len(want) {
		t.Fatalf("expected %d skills, got %d: %v", len(want), len(names), names)
	}
	// sort-agnostic check
	for _, w := range want {
		if !containsSlice(names, w) {
			t.Fatalf("missing expected skill %s in %v", w, names)
		}
	}
}

func TestGitRegistry_Fetch_InvalidURL(t *testing.T) {
	cacheDir := t.TempDir()
	g := &GitRegistry{
		RegistryName: "git",
		URL:          "/nonexistent/repo",
		CachePath:    filepath.Join(cacheDir, "git"),
	}
	_, _, err := g.Fetch("any")
	if err == nil {
		t.Fatalf("expected error from invalid git URL")
	}
	if !strings.Contains(err.Error(), "git clone") {
		t.Fatalf("expected clone error, got: %v", err)
	}
}

func TestGitRegistry_List_InvalidURL(t *testing.T) {
	cacheDir := t.TempDir()
	g := &GitRegistry{
		RegistryName: "git",
		URL:          "/nonexistent/repo",
		CachePath:    filepath.Join(cacheDir, "git"),
	}
	_, err := g.List()
	if err == nil {
		t.Fatalf("expected error from invalid git URL")
	}
}

func TestEmbeddedRegistry_Fetch_TempDirCleanup(t *testing.T) {
	r := NewEmbeddedRegistry()
	sk, srcDir, cleanup, err := r.FetchWithCleanup("conventional-commits")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk == nil || sk.Name != "conventional-commits" {
		t.Fatalf("expected valid skill, got %v", sk)
	}
	if srcDir == "" {
		t.Fatalf("expected non-empty srcDir")
	}

	// srcDir should exist before cleanup
	if _, err := os.Stat(srcDir); err != nil {
		t.Fatalf("srcDir should exist before cleanup: %v", err)
	}

	// Call cleanup -- temp dir should be removed
	cleanup()

	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Fatalf("srcDir should be removed after cleanup, got err: %v", err)
	}
}

func TestEmbeddedRegistry_Fetch_CleanupIsIdempotent(t *testing.T) {
	r := NewEmbeddedRegistry()
	_, _, cleanup, err := r.FetchWithCleanup("conventional-commits")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Calling cleanup multiple times should not panic
	cleanup()
	cleanup()
}

// helpers
func contains(s, sub string) bool { return strings.Contains(s, sub) }

func containsSlice(slice []string, v string) bool {
	for _, s := range slice {
		if s == v {
			return true
		}
	}
	return false
}
