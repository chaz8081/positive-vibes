package fsutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveWithinRoot_AllowsSimpleRelativePath(t *testing.T) {
	root := filepath.Join("/tmp", "pv-root")
	resolved, err := ResolveWithinRoot(root, "nested/file.md")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "nested", "file.md"), resolved)
}

func TestResolveWithinRoot_RejectsTraversal(t *testing.T) {
	root := filepath.Join("/tmp", "pv-root")
	_, err := ResolveWithinRoot(root, "../outside")
	require.Error(t, err)
}

func TestResolveWithinRoot_RejectsAbsolutePath(t *testing.T) {
	root := filepath.Join("/tmp", "pv-root")
	_, err := ResolveWithinRoot(root, "/etc/passwd")
	require.Error(t, err)
}

func TestResolveWithinRoot_RejectsEmptyPath(t *testing.T) {
	root := filepath.Join("/tmp", "pv-root")
	_, err := ResolveWithinRoot(root, "")
	require.Error(t, err)
}
