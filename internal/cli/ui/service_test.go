package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chaz8081/positive-vibes/internal/manifest"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

func TestService_ListAndShow(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(projectDir) error: %v", err)
	}

	manifestYAML := `skills:
  - name: code-review
instructions:
  - name: style
    path: ./instructions/style.instructions.md
agents:
  - name: reviewer
    path: ./agents/reviewer.agent.md
targets:
  - opencode
`
	if err := os.WriteFile(filepath.Join(projectDir, "vibes.yaml"), []byte(manifestYAML), 0o644); err != nil {
		t.Fatalf("WriteFile(vibes.yaml) error: %v", err)
	}

	globalPath := filepath.Join(tmpDir, "global", "vibes.yaml")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(globalPath dir) error: %v", err)
	}

	availableByKind := map[string][]ResourceRow{
		"skills": {
			{Name: "code-review"},
			{Name: "tdd"},
		},
		"agents": {
			{Name: "reviewer"},
			{Name: "planner"},
		},
		"instructions": {
			{Name: "style"},
			{Name: "security"},
		},
	}

	installedByKind := map[string][]ResourceRow{
		"skills": {
			{Name: "code-review", Installed: true},
		},
		"agents": {
			{Name: "planner", Installed: true},
		},
		"instructions": {
			{Name: "style", Installed: true},
		},
	}

	svc := newServiceWithDeps(serviceDeps{
		listAvailable: func(kind string) ([]ResourceRow, error) {
			return availableByKind[kind], nil
		},
		listInstalled: func(kind string) ([]ResourceRow, error) {
			return installedByKind[kind], nil
		},
		showDetail: func(kind, name string) (ResourceDetail, error) {
			return showResourceDetail(projectDir, globalPath, kind, name)
		},
		install: func(string, []string) error { return nil },
		remove:  func(string, []string) error { return nil },
	})

	for _, kind := range []string{"skills", "agents", "instructions"} {
		rows, err := svc.ListResources(kind)
		if err != nil {
			t.Fatalf("ListResources(%s) returned error: %v", kind, err)
		}
		if len(rows) != 2 {
			t.Fatalf("ListResources(%s) returned %d rows, want 2", kind, len(rows))
		}

		installedSeen := false
		for _, row := range rows {
			if row.Installed {
				installedSeen = true
			}
		}
		if !installedSeen {
			t.Fatalf("ListResources(%s) did not mark installed rows", kind)
		}
	}

	skillDetail, err := svc.ShowResource("skills", "code-review")
	if err != nil {
		t.Fatalf("ShowResource(skills) returned error: %v", err)
	}
	skillPayload, ok := skillDetail.Payload.(*schema.Skill)
	if !ok {
		t.Fatalf("ShowResource(skills) payload type = %T, want *schema.Skill", skillDetail.Payload)
	}
	if skillPayload.Name != "code-review" {
		t.Fatalf("ShowResource(skills) payload name = %q, want code-review", skillPayload.Name)
	}

	agentDetail, err := svc.ShowResource("agents", "reviewer")
	if err != nil {
		t.Fatalf("ShowResource(agents) returned error: %v", err)
	}
	agentPayload, ok := agentDetail.Payload.(manifest.AgentRef)
	if !ok {
		t.Fatalf("ShowResource(agents) payload type = %T, want manifest.AgentRef", agentDetail.Payload)
	}
	if agentPayload.Name != "reviewer" {
		t.Fatalf("ShowResource(agents) payload name = %q, want reviewer", agentPayload.Name)
	}

	instructionDetail, err := svc.ShowResource("instructions", "style")
	if err != nil {
		t.Fatalf("ShowResource(instructions) returned error: %v", err)
	}
	instructionPayload, ok := instructionDetail.Payload.(manifest.InstructionRef)
	if !ok {
		t.Fatalf("ShowResource(instructions) payload type = %T, want manifest.InstructionRef", instructionDetail.Payload)
	}
	if instructionPayload.Name != "style" {
		t.Fatalf("ShowResource(instructions) payload name = %q, want style", instructionPayload.Name)
	}
}
