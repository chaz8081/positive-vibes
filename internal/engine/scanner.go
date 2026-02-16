package engine

import (
	"os"
	"path/filepath"
)

// ScanResult describes a detected project type and recommendations.
type ScanResult struct {
	Language          string
	RecommendedSkills []string
	SuggestedTargets  []string
}

// ScanProject inspects a directory to determine probable language and suggestions.
func ScanProject(dir string) (*ScanResult, error) {
	res := &ScanResult{
		Language:          "unknown",
		RecommendedSkills: []string{"conventional-commits", "code-review"},
		SuggestedTargets:  []string{"vscode-copilot", "opencode", "cursor"},
	}

	// check for go.mod
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		res.Language = "go"
		return res, nil
	}

	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		res.Language = "node"
		return res, nil
	}

	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		res.Language = "python"
		return res, nil
	}

	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		res.Language = "python"
		return res, nil
	}

	return res, nil
}
