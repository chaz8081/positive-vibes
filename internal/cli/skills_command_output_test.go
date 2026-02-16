package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/spf13/cobra"
)

func TestInstallSkillsRun_PrintsMutationReportOnPartialFailure(t *testing.T) {
	projectDir = t.TempDir()
	t.Cleanup(func() { projectDir = "." })

	original := installResourcesCommandAction
	t.Cleanup(func() { installResourcesCommandAction = original })
	installResourcesCommandAction = func(projectDir, globalPath, kind string, names []string) (ResourceMutationReport, error) {
		return ResourceMutationReport{
				MutatedNames:          []string{"alpha"},
				SkippedDuplicateNames: []string{"beta"},
			},
			os.ErrInvalid
	}

	stdout, stderr := captureStdoutStderr(t, func() {
		cmd := &cobra.Command{}
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		_ = installSkillsRun(cmd, []string{"alpha", "beta", "broken"})
	})

	if !strings.Contains(stdout, "Added 'alpha' to vibes.yaml") {
		t.Fatalf("stdout = %q, want added mutation line", stdout)
	}
	if !strings.Contains(stderr, "warning: skill 'beta' already exists in manifest, skipping") {
		t.Fatalf("stderr = %q, want duplicate warning", stderr)
	}
	if !strings.Contains(stderr, "error: invalid argument") {
		t.Fatalf("stderr = %q, want command error", stderr)
	}
}

func TestRemoveSkillsRun_PrintsMutationReportOnPartialFailure(t *testing.T) {
	projectDir = t.TempDir()
	t.Cleanup(func() { projectDir = "." })
	writeResourceActionManifest(t, projectDir, &manifest.Manifest{})

	original := removeResourcesCommandAction
	t.Cleanup(func() { removeResourcesCommandAction = original })
	removeResourcesCommandAction = func(projectDir, kind string, names []string) (ResourceMutationReport, error) {
		return ResourceMutationReport{
				MutatedNames:        []string{"alpha"},
				SkippedMissingNames: []string{"ghost"},
			},
			os.ErrInvalid
	}

	stdout, stderr := captureStdoutStderr(t, func() {
		cmd := &cobra.Command{}
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		_ = removeSkillsRun(cmd, []string{"alpha", "ghost", "broken"})
	})

	if !strings.Contains(stdout, "Removed 'alpha' from vibes.yaml") {
		t.Fatalf("stdout = %q, want removed mutation line", stdout)
	}
	if !strings.Contains(stderr, "warning: skill not found in manifest: ghost") {
		t.Fatalf("stderr = %q, want missing warning", stderr)
	}
	if !strings.Contains(stderr, "error: invalid argument") {
		t.Fatalf("stderr = %q, want command error", stderr)
	}
}

func TestListCmd_TargetsProducesOutput(t *testing.T) {
	projectDir = t.TempDir()
	t.Cleanup(func() { projectDir = "." })

	stdout, stderr := captureStdoutStderr(t, func() {
		err := listCmd.RunE(listCmd, []string{"targets"})
		if err != nil {
			t.Fatalf("list targets run error: %v", err)
		}
	})

	if !strings.Contains(stdout, "Targets:") {
		t.Fatalf("stdout = %q, want targets list output", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, expected empty stderr", stderr)
	}
}

func TestListCmd_RegistriesProducesOutput(t *testing.T) {
	projectDir = t.TempDir()
	t.Cleanup(func() { projectDir = "." })

	manifestPath := filepath.Join(projectDir, "vibes.yaml")
	content := `targets: ["opencode"]
registries:
  - name: demo
    url: https://example.com/demo.git
    ref: main
`
	if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	stdout, stderr := captureStdoutStderr(t, func() {
		err := listCmd.RunE(listCmd, []string{"registries"})
		if err != nil {
			t.Fatalf("list registries run error: %v", err)
		}
	})

	if !strings.Contains(stdout, "Registries:") {
		t.Fatalf("stdout = %q, want registries list output", stdout)
	}
	if !strings.Contains(stdout, "demo") {
		t.Fatalf("stdout = %q, want registry name", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, expected empty stderr", stderr)
	}
}

func captureStdoutStderr(t *testing.T, run func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() stdout error = %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() stderr error = %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	run()

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("ReadAll(stdout) error = %v", err)
	}
	stderrBytes, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("ReadAll(stderr) error = %v", err)
	}

	return string(stdoutBytes), string(stderrBytes)
}
