package target

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempSkill(t *testing.T, dir string) *schema.Skill {
	t.Helper()
	s := &schema.Skill{
		Name:        "test-skill",
		Description: "a test skill",
		Version:     "0.1.0",
		Author:      "tester",
	}
	content, err := schema.RenderSkillFile(s)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))
	// create an extra file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra"), 0o644))
	return s
}

func TestCopilotTarget_Install(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := CopilotTarget{}
	err := tgt.Install(s, src, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".github", "skills", s.Name, "SKILL.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Contains(t, string(b), "test-skill")
}

func TestOpenCodeTarget_Install(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := OpenCodeTarget{}
	err := tgt.Install(s, src, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".opencode", "skills", s.Name, "SKILL.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Contains(t, string(b), "test-skill")
}

func TestCursorTarget_Install(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := CursorTarget{}
	err := tgt.Install(s, src, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".cursor", "skills", s.Name, "SKILL.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Contains(t, string(b), "test-skill")
}

func TestTarget_Install_NoOverwrite(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := CopilotTarget{}
	require.NoError(t, tgt.Install(s, src, proj, InstallOpts{}))
	// try again without force
	err := tgt.Install(s, src, proj, InstallOpts{})
	require.Error(t, err)
}

func TestTarget_Install_Force(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := CopilotTarget{}
	require.NoError(t, tgt.Install(s, src, proj, InstallOpts{}))
	// force overwrite
	require.NoError(t, tgt.Install(s, src, proj, InstallOpts{Force: true}))
}

func TestTarget_Install_Link(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := OpenCodeTarget{}
	require.NoError(t, tgt.Install(s, src, proj, InstallOpts{Link: true}))

	installed := filepath.Join(proj, ".opencode", "skills", s.Name)
	fi, err := os.Lstat(installed)
	require.NoError(t, err)
	assert.True(t, fi.Mode()&os.ModeSymlink != 0)
}

func TestTarget_SkillExists(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	s := writeTempSkill(t, src)

	proj := filepath.Join(tmp, "proj")
	require.NoError(t, os.MkdirAll(proj, 0o755))

	tgt := CursorTarget{}
	require.False(t, tgt.SkillExists(s.Name, proj))
	require.NoError(t, tgt.Install(s, src, proj, InstallOpts{}))
	require.True(t, tgt.SkillExists(s.Name, proj))
}

func TestResolveTargets_Valid(t *testing.T) {
	names := []string{"vscode-copilot", "opencode", "cursor"}
	ts, err := ResolveTargets(names)
	require.NoError(t, err)
	assert.Len(t, ts, 3)
}

func TestResolveTargets_Invalid(t *testing.T) {
	_, err := ResolveTargets([]string{"notepad"})
	require.Error(t, err)
}
