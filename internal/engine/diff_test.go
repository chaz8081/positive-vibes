package engine

import (
	"fmt"
	"strings"
	"testing"
)

func TestUnifiedDiff_BothEmpty(t *testing.T) {
	result := unifiedDiff("a.txt", "b.txt", "", "")
	if result != "" {
		t.Fatalf("expected empty diff for identical empty files, got:\n%s", result)
	}
}

func TestUnifiedDiff_Identical(t *testing.T) {
	text := "line1\nline2\nline3\n"
	result := unifiedDiff("a.txt", "b.txt", text, text)
	if result != "" {
		t.Fatalf("expected empty diff for identical files, got:\n%s", result)
	}
}

func TestUnifiedDiff_Addition(t *testing.T) {
	a := "line1\nline2\n"
	b := "line1\nline2\nline3\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(result, "+line3") {
		t.Fatalf("expected +line3 in diff, got:\n%s", result)
	}
	if !strings.Contains(result, "--- a.txt") {
		t.Fatalf("expected --- a.txt header, got:\n%s", result)
	}
	if !strings.Contains(result, "+++ b.txt") {
		t.Fatalf("expected +++ b.txt header, got:\n%s", result)
	}
}

func TestUnifiedDiff_Deletion(t *testing.T) {
	a := "line1\nline2\nline3\n"
	b := "line1\nline2\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(result, "-line3") {
		t.Fatalf("expected -line3 in diff, got:\n%s", result)
	}
}

func TestUnifiedDiff_Modification(t *testing.T) {
	a := "line1\nold line\nline3\n"
	b := "line1\nnew line\nline3\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	if !strings.Contains(result, "-old line") {
		t.Fatalf("expected -old line in diff, got:\n%s", result)
	}
	if !strings.Contains(result, "+new line") {
		t.Fatalf("expected +new line in diff, got:\n%s", result)
	}
}

func TestUnifiedDiff_ContextLines(t *testing.T) {
	a := "a\nb\nc\nd\ne\nf\ng\n"
	b := "a\nb\nc\nD\ne\nf\ng\n"
	result := unifiedDiff("a.txt", "b.txt", a, b)
	// Should have context lines around the change
	if !strings.Contains(result, " c") {
		t.Fatalf("expected context line ' c' in diff, got:\n%s", result)
	}
	if !strings.Contains(result, " e") {
		t.Fatalf("expected context line ' e' in diff, got:\n%s", result)
	}
}

func TestUnifiedDiff_LargeInputsFallback(t *testing.T) {
	t.Parallel()

	var a, b strings.Builder
	for i := 0; i < 5000; i++ {
		fmt.Fprintf(&a, "line-%d\n", i)
		fmt.Fprintf(&b, "line-%d-changed\n", i)
	}

	diff := unifiedDiff("a.txt", "b.txt", a.String(), b.String())
	if diff == "" {
		t.Fatalf("expected non-empty diff for large inputs")
	}
	if !strings.Contains(diff, "diff omitted") {
		t.Fatalf("expected fallback message, got:\n%s", diff)
	}
}
