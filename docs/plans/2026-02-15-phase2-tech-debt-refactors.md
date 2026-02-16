# Phase 2 Tech-Debt Refactors Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce structural duplication and improve maintainability in the CLI and engine by refactoring ApplyManifest, splitting large files, and standardizing CLI error handling.

**Architecture:** Keep behavior unchanged while reorganizing code: extract shared apply loops, split monolithic CLI resource handlers into focused files, and move all commands to `RunE` with consistent error propagation. Make small cleanup changes that improve clarity without changing external behavior.

**Tech Stack:** Go stdlib, Cobra CLI, existing `internal/cli` and `internal/engine` patterns.

---

### Task 1: Add characterization tests for ApplyManifest duplication areas

**Files:**
- Modify: `internal/engine/applier_test.go`
- Modify: `internal/engine/applier_dryrun_test.go`

**Step 1: Write failing tests to lock behavior**

Add tests that verify each resource type uses consistent behavior for:
- apply_to filtering (instructions)
- prompt unsupported skip (prompts)
- force/skip logic (skills)
- dry-run preview op counts (skills/instructions/agents/prompts)

**Step 2: Run tests to verify failure**

Run: `go test ./internal/engine -run ApplyManifest_ -v`

Expected: FAIL (tests should expose the duplicated behavior and highlight missing coverage).

**Step 3: Commit failing tests**

```bash
git add internal/engine/applier_test.go internal/engine/applier_dryrun_test.go
git commit -m "test(engine): add ApplyManifest characterization tests"
```

---

### Task 2: Refactor ApplyManifest duplicated loops

**Files:**
- Modify: `internal/engine/applier.go`
- Modify: `internal/engine/applier_test.go`

**Step 1: Extract shared apply helper**

Introduce a helper function (or small struct) that encapsulates:
- resolving source path (local or registry)
- dry-run preview vs install
- cleanup of temporary files

Example shape:

```go
type resourceApplier struct {
    kind ApplyOpKind
    name string
    registry string
    path string
    content string
    suffix string
    applyTo string
    targetDir func(target.Target) string
    install func(target.Target, string, target.InstallOpts) error
    preview func(target.Target, string, string) (DryRunOp, error)
}
```

Then loop `[]resourceApplier{...}` for instructions/agents/prompts, and keep skills separate or convert skills into another struct.

**Step 2: Run characterization tests**

Run: `go test ./internal/engine -run ApplyManifest_ -v`

Expected: PASS

**Step 3: Run full engine tests**

Run: `go test ./internal/engine -v`

Expected: PASS

**Step 4: Commit**

```bash
git add internal/engine/applier.go internal/engine/applier_test.go
git commit -m "refactor(engine): dedupe ApplyManifest resource loops"
```

---

### Task 3: Split monolithic `resource.go` into focused files

**Files:**
- Modify: `internal/cli/resource.go`
- Create: `internal/cli/resource_types.go`
- Create: `internal/cli/resource_format.go`
- Create: `internal/cli/resource_completion.go`
- Create: `internal/cli/resource_helpers.go`

**Step 1: Write characterization tests**

Add tests that validate outputs and formatting of list/show/install/remove for each resource type.

Run: `go test ./internal/cli -run Resource -v`

Expected: PASS (if tests already exist) or FAIL (if missing coverage).

**Step 2: Split file**

Move types/structs to `resource_types.go`, formatting helpers to `resource_format.go`, completion helpers to `resource_completion.go`, and small helpers to `resource_helpers.go`.

**Step 3: Run tests**

Run: `go test ./internal/cli -run Resource -v`

Expected: PASS

**Step 4: Commit**

```bash
git add internal/cli/resource.go internal/cli/resource_*.go
git commit -m "refactor(cli): split resource.go into focused files"
```

---

### Task 4: Standardize all CLI commands on RunE

**Files:**
- Modify: `internal/cli/apply.go`
- Modify: `internal/cli/install.go`
- Modify: `internal/cli/remove.go`
- Modify: `internal/cli/list.go`
- Modify: `internal/cli/show.go`
- Modify: `internal/cli/generate.go`
- Modify: `internal/cli/init.go`
- Modify: `internal/cli/root.go`

**Step 1: Convert Run -> RunE and return errors**

For each command, return errors instead of printing and returning.
Example:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if applyDryRun && applyForce {
        return fmt.Errorf("--dry-run cannot be used with --force")
    }
    ...
    return nil
}
```

**Step 2: Run CLI tests**

Run: `go test ./internal/cli -v`

Expected: PASS

**Step 3: Commit**

```bash
git add internal/cli/*.go
git commit -m "refactor(cli): standardize commands on RunE"
```

---

### Task 5: Clean small tech debt items

**Files:**
- Modify: `internal/engine/applier.go` (rename ApplyOp.SkillName)
- Modify: `internal/engine/scanner.go` (filepath.Join)
- Modify: `internal/cli/init.go` (remove blank import)
- Modify: `internal/registry/embedded.go` (remove deprecated Fetch if unused)

**Step 1: Apply changes**

- Rename `SkillName` to `ResourceName` across the codebase (Go rename or search+replace)
- Replace string path concatenation with `filepath.Join`
- Remove unused blank import in `init.go`
- Remove deprecated `Fetch` if no callers remain

**Step 2: Run tests**

Run: `go test ./...`

Expected: PASS

**Step 3: Commit**

```bash
git add internal/engine internal/cli internal/registry
git commit -m "refactor: cleanup naming and dead code"
```

---

### Task 6: Consolidate ANSI color handling

**Files:**
- Create: `internal/term/color.go`
- Modify: `internal/engine/dryrun.go`
- Modify: `internal/cli/config.go`

**Step 1: Add color helper**

Define shared ANSI constants and helper functions in `internal/term/color.go`.

**Step 2: Replace duplicates**

Update `dryrun.go` and `config.go` to use the shared helper.

**Step 3: Run tests**

Run: `go test ./...`

Expected: PASS

**Step 4: Commit**

```bash
git add internal/term internal/engine/dryrun.go internal/cli/config.go
git commit -m "refactor: centralize ANSI color helpers"
```

---

### Task 7: Update docs & plan status

**Files:**
- Modify: `docs/plans/2026-02-15-repo-improvements-remediation.md`

**Step 1: Note completion/next steps**

Add a status note summarizing Phase 2 coverage and outstanding items.

**Step 2: Commit**

```bash
git add docs/plans/2026-02-15-repo-improvements-remediation.md
git commit -m "docs: update remediation plan status"
```

---

### Task 8: Final verification

**Step 1: Run full test suite**

Run: `go test ./...`

Expected: PASS

**Step 2: Summarize changes for review**

Prepare a brief summary and request code review.
