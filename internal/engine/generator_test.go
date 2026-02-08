package engine

import (
	"strings"
	"testing"
)

func TestMockGenerator(t *testing.T) {
	mg := NewMockGenerator()
	sk, err := mg.Generate("accessibility checker")
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	if sk.Name != "accessibility-checker" {
		t.Fatalf("expected slug name, got %s", sk.Name)
	}
	if sk.Description != "accessibility checker" {
		t.Fatalf("description mismatch")
	}
	if !strings.Contains(sk.Instructions, "accessibility checker") {
		t.Fatalf("instructions do not contain description")
	}

	// empty
	sk2, err := mg.Generate("")
	if err != nil {
		t.Fatalf("generate empty error: %v", err)
	}
	if sk2 == nil {
		t.Fatalf("expected non-nil skill for empty description")
	}
}
