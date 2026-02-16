package fsutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveWithinRoot resolves rel beneath root and rejects absolute/escaping paths.
func ResolveWithinRoot(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative: %q", rel)
	}

	cleanRoot := filepath.Clean(root)
	resolved := filepath.Clean(filepath.Join(cleanRoot, rel))
	relToRoot, err := filepath.Rel(cleanRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path %q within %q: %w", rel, root, err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root: %q", rel)
	}

	return resolved, nil
}
