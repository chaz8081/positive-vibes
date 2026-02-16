# Phase 1 Dry-Run Correctness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix dry-run correctness and output accuracy, eliminate disk writes in dry-run, and add missing tests for registry paths and output formatting.

**Architecture:** Keep current dry-run flow but ensure registry-backed resources pass content directly to preview paths without temp files, and update preview helpers to correctly mark unchanged files as skipped. Extend diff formatting to color headers correctly and guard against huge diffs. Tests cover new behavior at the engine layer.

**Tech Stack:** Go stdlib, existing test patterns in `internal/engine`.

---

### Task 1: Add failing test for registry-backed dry-run with no temp files

**Files:**
- Modify: `internal/engine/applier_dryrun_test.go`

**Step 1: Write the failing test**

Add a new test that uses a fake registry implementing `ResourceSource` to simulate registry-backed instruction/agent/prompt data and asserts no files are written under `projectDir` during dry-run:

```go
func TestApplyManifest_DryRun_RegistryDoesNotWriteTempFiles(t *testing.T) {
    t.Parallel()

    reg := &fakeResourceRegistry{
        name: "test-reg",
        data: map[string][]byte{
            "instructions/hello.md": []byte("inst content\n"),
            "agents/bot.md":          []byte("agent content\n"),
            "prompts/prompt.md":      []byte("prompt content\n"),
        },
    }

    dir := t.TempDir()
    m := &manifest.Manifest{
        Instructions: []manifest.InstructionRef{{
            Name:     "hello",
            Registry: "test-reg",
            Path:     "hello.md",
        }},
        Agents: []manifest.AgentRef{{
            Name:     "bot",
            Registry: "test-reg",
            Path:     "bot.md",
        }},
        Prompts: []manifest.PromptRef{{
            Name:     "prompt",
            Registry: "test-reg",
            Path:     "prompt.md",
        }},
        Targets: []string{"opencode"},
    }

    a := NewApplier([]registry.SkillSource{reg})
    res, err := a.ApplyManifest(m, dir, target.InstallOpts{DryRun: true})
    if err != nil {
        t.Fatalf("apply manifest: %v", err)
    }

    if len(res.Errors) > 0 {
        t.Fatalf("expected no errors, got: %v", res.Errors)
    }

    entries, err := os.ReadDir(dir)
    if err != nil {
        t.Fatalf("read project dir: %v", err)
    }
    if len(entries) != 0 {
        t.Fatalf("expected no files in project dir, got %d entries", len(entries))
    }
}
```

Add a minimal fake registry type at the bottom of the file:

```go
type fakeResourceRegistry struct {
    name string
    data map[string][]byte
}

func (f *fakeResourceRegistry) Name() string { return f.name }
func (f *fakeResourceRegistry) Fetch(name string) (*schema.Skill, string, error) {
    return nil, "", registry.ErrSkillNotFound
}
func (f *fakeResourceRegistry) FetchResourceFile(category, name string) ([]byte, error) {
    key := filepath.Join(category, name)
    if data, ok := f.data[key]; ok {
        return data, nil
    }
    return nil, registry.ErrResourceNotFound
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -run TestApplyManifest_DryRun_RegistryDoesNotWriteTempFiles -v`

Expected: FAIL — currently temp files are created under `projectDir`.

**Step 3: Commit the failing test**

```bash
git add internal/engine/applier_dryrun_test.go
git commit -m "test(engine): add dry-run registry temp file guard"
```

---

### Task 2: Avoid temp file writes in dry-run for registry resources

**Files:**
- Modify: `internal/engine/applier.go:213-451`

**Step 1: Update dry-run branch for registry resources**

Restructure instruction/agent/prompt loops so that when `opts.DryRun` is true and `Registry != ""`, we pass fetched content directly to preview instead of writing a temp file.

Example for instructions:

```go
var previewContent string
if inst.Registry != "" {
    data, fetchErr := a.fetchResourceFileFromRegistry(inst.Registry, "instructions", inst.Path)
    if fetchErr != nil { ... }
    if opts.DryRun {
        previewContent = string(data)
    } else {
        tmp, tmpErr := writeTempResourceFile(projectDir, "pv-inst-*", data)
        ...
        tempFile = tmp
        sourcePath = tempFile
    }
} else {
    ...
}

if opts.DryRun {
    op, previewErr := previewSingleFileInstall(inst.Name, firstNonEmpty(inst.Content, previewContent), sourcePath, ...)
    ...
}
```

Do the same for agents and prompts, using `previewContent` when dry-run.

**Step 2: Run the failing test**

Run: `go test ./internal/engine -run TestApplyManifest_DryRun_RegistryDoesNotWriteTempFiles -v`

Expected: PASS

**Step 3: Run full engine tests**

Run: `go test ./internal/engine -v`

Expected: PASS

**Step 4: Commit**

```bash
git add internal/engine/applier.go
git commit -m "fix(engine): avoid temp files in dry-run registry preview"
```

---

### Task 3: Mark unchanged files as skips in preview helpers

**Files:**
- Modify: `internal/engine/preview.go`
- Modify: `internal/engine/preview_test.go`

**Step 1: Write failing tests**

Add tests asserting that unchanged files emit `DryRunSkip` with reason `"unchanged"` for both skill SKILL.md and single-file resources:

```go
func TestPreviewSkillInstall_UnchangedIsSkip(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    target := target.NewOpenCodeTarget()
    skill := &schema.Skill{Name: "demo"}
    rendered, err := schema.RenderSkillFile(skill)
    if err != nil {
        t.Fatalf("render skill: %v", err)
    }

    skillDir := filepath.Join(dir, target.SkillDir(), "demo")
    if err := os.MkdirAll(skillDir, 0o755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), rendered, 0o644); err != nil {
        t.Fatalf("write: %v", err)
    }

    ops, err := previewSkillInstall(skill, "", dir, target)
    if err != nil {
        t.Fatalf("preview: %v", err)
    }

    if len(ops) != 1 {
        t.Fatalf("expected 1 op, got %d", len(ops))
    }
    if ops[0].Action != DryRunSkip || ops[0].Reason != "unchanged" {
        t.Fatalf("expected skip unchanged, got %v (%s)", ops[0].Action, ops[0].Reason)
    }
}
```

```go
func TestPreviewSingleFileInstall_UnchangedIsSkip(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    target := target.NewOpenCodeTarget()
    name := "hello"
    resDir := target.InstructionDir()
    dest := filepath.Join(dir, resDir, name+".md")
    if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    if err := os.WriteFile(dest, []byte("content\n"), 0o644); err != nil {
        t.Fatalf("write: %v", err)
    }

    op, err := previewSingleFileInstall(name, "content\n", "", dir, resDir, ".md", target, KindInstruction)
    if err != nil {
        t.Fatalf("preview: %v", err)
    }
    if op.Action != DryRunSkip || op.Reason != "unchanged" {
        t.Fatalf("expected skip unchanged, got %v (%s)", op.Action, op.Reason)
    }
}
```

**Step 2: Run tests to verify failure**

Run: `go test ./internal/engine -run UnchangedIsSkip -v`

Expected: FAIL — currently returns `DryRunUpdate`.

**Step 3: Update preview helpers**

In `previewSkillInstall`, after computing `diff`, if `diff == ""`, return a `DryRunSkip` with reason `"unchanged"` instead of `DryRunUpdate`. Do this for SKILL.md and additional files.

In `previewSingleFileInstall`, if `diff == ""`, return `DryRunSkip` with reason `"unchanged"`.

**Step 4: Run tests**

Run: `go test ./internal/engine -run UnchangedIsSkip -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/engine/preview.go internal/engine/preview_test.go
git commit -m "fix(engine): mark unchanged dry-run files as skipped"
```

---

### Task 4: Fail fast on empty preview input

**Files:**
- Modify: `internal/engine/preview.go`
- Modify: `internal/engine/preview_test.go`

**Step 1: Write failing test**

```go
func TestPreviewSingleFileInstall_EmptyInputReturnsError(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    target := target.NewOpenCodeTarget()
    _, err := previewSingleFileInstall("name", "", "", dir, target.InstructionDir(), ".md", target, KindInstruction)
    if err == nil {
        t.Fatalf("expected error for empty preview input")
    }
}
```

**Step 2: Run test to verify failure**

Run: `go test ./internal/engine -run EmptyInputReturnsError -v`

Expected: FAIL — currently returns nil error.

**Step 3: Implement error path**

Change `previewSingleFileInstall` to return an error when both `content` and `sourcePath` are empty:

```go
return DryRunOp{}, fmt.Errorf("resource %q: no content or source path", name)
```

**Step 4: Run test**

Run: `go test ./internal/engine -run EmptyInputReturnsError -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/engine/preview.go internal/engine/preview_test.go
git commit -m "fix(engine): error on empty preview inputs"
```

---

### Task 5: Add ColorDiff header handling and tests

**Files:**
- Modify: `internal/engine/dryrun.go`
- Modify: `internal/engine/dryrun_test.go`

**Step 1: Write failing tests**

```go
func TestColorDiff_HeaderLinesAreNotTreatedAsAddsOrDeletes(t *testing.T) {
    t.Parallel()

    diff := strings.Join([]string{
        "--- a/file.txt",
        "+++ b/file.txt",
        "@@ -1,1 +1,1 @@",
        "-old",
        "+new",
        " context",
        "",
    }, "\n")

    colored := ColorDiff(diff)

    if strings.Contains(colored, ansiRed+"--- a/file.txt") {
        t.Fatalf("expected header line not to be colored as deletion")
    }
    if strings.Contains(colored, ansiGreen+"+++ b/file.txt") {
        t.Fatalf("expected header line not to be colored as addition")
    }
}
```

**Step 2: Run test to verify failure**

Run: `go test ./internal/engine -run HeaderLinesAreNotTreatedAsAddsOrDeletes -v`

Expected: FAIL

**Step 3: Update ColorDiff**

Add header cases before `+`/`-` checks:

```go
case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "):
    fmt.Fprintf(&buf, "%s%s%s\n", ansiCyan, line, ansiReset)
```

**Step 4: Run tests**

Run: `go test ./internal/engine -run HeaderLinesAreNotTreatedAsAddsOrDeletes -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/engine/dryrun.go internal/engine/dryrun_test.go
git commit -m "fix(engine): color diff headers distinctly"
```

---

### Task 6: Add sourcePath coverage for previewSingleFileInstall

**Files:**
- Modify: `internal/engine/preview_test.go`

**Step 1: Write failing test**

```go
func TestPreviewSingleFileInstall_FromSourcePath(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    target := target.NewOpenCodeTarget()
    source := filepath.Join(dir, "source.md")
    if err := os.WriteFile(source, []byte("source content\n"), 0o644); err != nil {
        t.Fatalf("write source: %v", err)
    }

    op, err := previewSingleFileInstall("hello", "", source, dir, target.InstructionDir(), ".md", target, KindInstruction)
    if err != nil {
        t.Fatalf("preview: %v", err)
    }
    if op.Action != DryRunCreate {
        t.Fatalf("expected create, got %v", op.Action)
    }
}
```

**Step 2: Run test**

Run: `go test ./internal/engine -run FromSourcePath -v`

Expected: PASS (if implementation already works) or FAIL if missing coverage.

**Step 3: If failing, fix implementation**

Update `previewSingleFileInstall` accordingly.

**Step 4: Commit**

```bash
git add internal/engine/preview_test.go
git commit -m "test(engine): cover preview sourcePath input"
```

---

### Task 7: Add large diff guard in unifiedDiff

**Files:**
- Modify: `internal/engine/diff.go`
- Modify: `internal/engine/diff_test.go`

**Step 1: Write failing test**

```go
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
```

**Step 2: Run test to verify failure**

Run: `go test ./internal/engine -run LargeInputsFallback -v`

Expected: FAIL

**Step 3: Implement guard**

In `unifiedDiff`, before `diffLines`, add:

```go
const maxDiffCells = 10_000_000
if len(linesA)*len(linesB) > maxDiffCells {
    return fmt.Sprintf("--- %s\n+++ %s\n@@ -0,0 +0,0 @@\n(diff omitted: files too large)\n", nameA, nameB)
}
```

**Step 4: Run test**

Run: `go test ./internal/engine -run LargeInputsFallback -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/engine/diff.go internal/engine/diff_test.go
git commit -m "feat(engine): guard diff for very large files"
```

---

### Task 8: Run full test suite

**Step 1: Run tests**

Run: `go test ./...`

Expected: PASS

**Step 2: Commit if any test adjustments were needed**

```bash
git add internal/engine
git commit -m "test(engine): align dry-run tests after fixes"
```

---

### Task 9: Update dry-run plan doc status

**Files:**
- Modify: `docs/plans/2026-02-15-apply-dry-run.md`

**Step 1: Mark plan complete**

Add a short completion note at the top or in a footer:

```markdown
**Status:** Completed on 2026-02-15 (merged into main).
```

**Step 2: Commit**

```bash
git add docs/plans/2026-02-15-apply-dry-run.md
git commit -m "docs: mark dry-run plan as completed"
```

---

### Task 10: Final verification

**Step 1: Run full test suite**

Run: `go test ./...`

Expected: PASS

**Step 2: Summarize changes for review**

Prepare a brief summary and request code review.
