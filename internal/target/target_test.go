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

// --- InstallInstruction tests ---

func TestCopilotTarget_InstallInstruction_Content(t *testing.T) {
	proj := t.TempDir()

	tgt := CopilotTarget{}
	err := tgt.InstallInstruction("go-style", "Always use gofmt", "", proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".github", "instructions", "go-style.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "Always use gofmt", string(b))
}

func TestOpenCodeTarget_InstallInstruction_Content(t *testing.T) {
	proj := t.TempDir()

	tgt := OpenCodeTarget{}
	err := tgt.InstallInstruction("ts-style", "Use TypeScript strict mode", "", proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".opencode", "instructions", "ts-style.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "Use TypeScript strict mode", string(b))
}

func TestCursorTarget_InstallInstruction_Content(t *testing.T) {
	proj := t.TempDir()

	tgt := CursorTarget{}
	err := tgt.InstallInstruction("py-style", "Use type hints", "", proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".cursor", "instructions", "py-style.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "Use type hints", string(b))
}

func TestInstallInstruction_FromPath(t *testing.T) {
	proj := t.TempDir()

	// Create a source instruction file
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "guide.md")
	require.NoError(t, os.WriteFile(srcFile, []byte("# Detailed guide\nDo things this way."), 0o644))

	tgt := OpenCodeTarget{}
	err := tgt.InstallInstruction("project-guide", "", srcFile, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".opencode", "instructions", "project-guide.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "# Detailed guide\nDo things this way.", string(b))
}

func TestInstallInstruction_Force(t *testing.T) {
	proj := t.TempDir()

	tgt := CopilotTarget{}
	require.NoError(t, tgt.InstallInstruction("style", "old content", "", proj, InstallOpts{}))
	// Without force, should error
	err := tgt.InstallInstruction("style", "new content", "", proj, InstallOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// With force, should overwrite
	require.NoError(t, tgt.InstallInstruction("style", "new content", "", proj, InstallOpts{Force: true}))
	b, err := os.ReadFile(filepath.Join(proj, ".github", "instructions", "style.md"))
	require.NoError(t, err)
	assert.Equal(t, "new content", string(b))
}

// --- InstallAgent tests ---

func TestCopilotTarget_InstallAgent_FromPath(t *testing.T) {
	proj := t.TempDir()

	// Create a source agent file
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "reviewer.md")
	require.NoError(t, os.WriteFile(srcFile, []byte("# Code Reviewer Agent\nReview code for bugs."), 0o644))

	tgt := CopilotTarget{}
	err := tgt.InstallAgent("reviewer", srcFile, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".github", "agents", "reviewer.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "# Code Reviewer Agent\nReview code for bugs.", string(b))
}

func TestOpenCodeTarget_InstallAgent_FromPath(t *testing.T) {
	proj := t.TempDir()

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "helper.md")
	require.NoError(t, os.WriteFile(srcFile, []byte("# Helper Agent"), 0o644))

	tgt := OpenCodeTarget{}
	err := tgt.InstallAgent("helper", srcFile, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".opencode", "agents", "helper.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "# Helper Agent", string(b))
}

func TestCursorTarget_InstallAgent_FromPath(t *testing.T) {
	proj := t.TempDir()

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "analyst.md")
	require.NoError(t, os.WriteFile(srcFile, []byte("# Analyst Agent"), 0o644))

	tgt := CursorTarget{}
	err := tgt.InstallAgent("analyst", srcFile, proj, InstallOpts{})
	require.NoError(t, err)

	got := filepath.Join(proj, ".cursor", "agents", "analyst.md")
	b, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "# Analyst Agent", string(b))
}

func TestInstallAgent_Force(t *testing.T) {
	proj := t.TempDir()

	srcDir := t.TempDir()
	oldFile := filepath.Join(srcDir, "old.md")
	newFile := filepath.Join(srcDir, "new.md")
	require.NoError(t, os.WriteFile(oldFile, []byte("old agent"), 0o644))
	require.NoError(t, os.WriteFile(newFile, []byte("new agent"), 0o644))

	tgt := OpenCodeTarget{}
	require.NoError(t, tgt.InstallAgent("my-agent", oldFile, proj, InstallOpts{}))

	// Without force, should error
	err := tgt.InstallAgent("my-agent", newFile, proj, InstallOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// With force, should overwrite
	require.NoError(t, tgt.InstallAgent("my-agent", newFile, proj, InstallOpts{Force: true}))
	b, err := os.ReadFile(filepath.Join(proj, ".opencode", "agents", "my-agent.md"))
	require.NoError(t, err)
	assert.Equal(t, "new agent", string(b))
}
