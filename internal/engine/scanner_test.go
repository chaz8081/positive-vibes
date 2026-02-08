package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanProject(t *testing.T) {
	tmp := t.TempDir()

	// go
	d1 := filepath.Join(tmp, "goproj")
	os.Mkdir(d1, 0o755)
	os.WriteFile(filepath.Join(d1, "go.mod"), []byte("module example"), 0o644)
	r, _ := ScanProject(d1)
	if r.Language != "go" {
		t.Fatalf("expected go, got %s", r.Language)
	}

	// node
	d2 := filepath.Join(tmp, "nodeproj")
	os.Mkdir(d2, 0o755)
	os.WriteFile(filepath.Join(d2, "package.json"), []byte("{}"), 0o644)
	r2, _ := ScanProject(d2)
	if r2.Language != "node" {
		t.Fatalf("expected node, got %s", r2.Language)
	}

	// python pyproject
	d3 := filepath.Join(tmp, "pyproj")
	os.Mkdir(d3, 0o755)
	os.WriteFile(filepath.Join(d3, "pyproject.toml"), []byte("[tool.poetry]"), 0o644)
	r3, _ := ScanProject(d3)
	if r3.Language != "python" {
		t.Fatalf("expected python, got %s", r3.Language)
	}

	// python requirements
	d4 := filepath.Join(tmp, "pyreq")
	os.Mkdir(d4, 0o755)
	os.WriteFile(filepath.Join(d4, "requirements.txt"), []byte("requests"), 0o644)
	r4, _ := ScanProject(d4)
	if r4.Language != "python" {
		t.Fatalf("expected python, got %s", r4.Language)
	}

	// unknown
	d5 := filepath.Join(tmp, "empty")
	os.Mkdir(d5, 0o755)
	r5, _ := ScanProject(d5)
	if r5.Language != "unknown" {
		t.Fatalf("expected unknown, got %s", r5.Language)
	}

	// check recommendations present
	if len(r5.RecommendedSkills) == 0 || len(r5.SuggestedTargets) == 0 {
		t.Fatalf("recommendations missing")
	}
}
