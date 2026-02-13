package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
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
		installSkillsRun([]string{"alpha", "beta", "broken"})
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
		removeSkillsRun([]string{"alpha", "ghost", "broken"})
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
