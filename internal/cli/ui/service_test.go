package ui

import (
	"testing"
)

func TestService_ListAndShow(t *testing.T) {
	availableByKind := map[string][]resourceRow{
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

	installedByKind := map[string][]resourceRow{
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

	details := map[string]ResourceDetail{
		"skills/code-review": {
			Kind:      "skills",
			Name:      "code-review",
			Installed: true,
			Payload:   "skill-payload",
		},
		"agents/reviewer": {
			Kind:      "agents",
			Name:      "reviewer",
			Installed: false,
			Payload:   "agent-payload",
		},
		"instructions/style": {
			Kind:      "instructions",
			Name:      "style",
			Installed: true,
			Payload:   "instruction-payload",
		},
	}

	svc := newServiceWithDeps(serviceDeps{
		listAvailable: func(kind string) ([]resourceRow, error) {
			return availableByKind[kind], nil
		},
		listInstalled: func(kind string) ([]resourceRow, error) {
			return installedByKind[kind], nil
		},
		showDetail: func(kind, name string) (ResourceDetail, error) {
			return details[kind+"/"+name], nil
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
	if skillDetail.Payload != "skill-payload" {
		t.Fatalf("ShowResource(skills) payload = %v, want skill-payload", skillDetail.Payload)
	}

	agentDetail, err := svc.ShowResource("agents", "reviewer")
	if err != nil {
		t.Fatalf("ShowResource(agents) returned error: %v", err)
	}
	if agentDetail.Payload != "agent-payload" {
		t.Fatalf("ShowResource(agents) payload = %v, want agent-payload", agentDetail.Payload)
	}

	instructionDetail, err := svc.ShowResource("instructions", "style")
	if err != nil {
		t.Fatalf("ShowResource(instructions) returned error: %v", err)
	}
	if instructionDetail.Payload != "instruction-payload" {
		t.Fatalf("ShowResource(instructions) payload = %v, want instruction-payload", instructionDetail.Payload)
	}
}
