package cli

import (
	"io"
	"os"
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
	if strings.Contains(stderr, "error:") {
		t.Fatalf("stderr = %q, expected no error line (top-level prints fatal errors)", stderr)
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
	if strings.Contains(stderr, "error:") {
		t.Fatalf("stderr = %q, expected no error line (top-level prints fatal errors)", stderr)
	}
}

func TestListCommand_RuntimeErrorsSilenceUsage(t *testing.T) {
	originalSilenceUsage := listCmd.SilenceUsage
	originalSilenceErrors := listCmd.SilenceErrors
	t.Cleanup(func() {
		listCmd.SilenceUsage = originalSilenceUsage
		listCmd.SilenceErrors = originalSilenceErrors
	})

	err := listCmd.RunE(listCmd, []string{"widgets"})
	if err == nil {
		t.Fatalf("expected error for invalid resource type")
	}
	if !listCmd.SilenceUsage || !listCmd.SilenceErrors {
		t.Fatalf("expected list command to silence usage/errors on runtime failure")
	}
}

func TestShowCommand_RuntimeErrorsSilenceUsage(t *testing.T) {
	originalSilenceUsage := showCmd.SilenceUsage
	originalSilenceErrors := showCmd.SilenceErrors
	t.Cleanup(func() {
		showCmd.SilenceUsage = originalSilenceUsage
		showCmd.SilenceErrors = originalSilenceErrors
	})

	err := showCmd.RunE(showCmd, []string{"widgets", "anything"})
	if err == nil {
		t.Fatalf("expected error for invalid resource type")
	}
	if !showCmd.SilenceUsage || !showCmd.SilenceErrors {
		t.Fatalf("expected show command to silence usage/errors on runtime failure")
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
