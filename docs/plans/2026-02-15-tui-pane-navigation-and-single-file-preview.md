# TUI Pane Navigation and Single-File Preview Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align browser navigation to Vim/arrow expectations (`h/j/k/l` and arrow equivalents), keep single-file resource previews inline, and remove unnecessary drill-in for instructions/prompts/agents.

**Architecture:** Keep the current two-pane browser layout, but make horizontal keys switch focus between panes and vertical keys operate within the focused pane. Preserve skills detail drill-in, while treating instructions/prompts/agents as inline-preview-only in browser mode. Update help/footer/docs and tests in lockstep to prevent regressions.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Cobra CLI docs, Go test.

---

### Task 1: Lock key behavior with failing UI tests first

**Files:**
- Modify: `internal/cli/ui/model_test.go`

**Step 1: Write failing tests for focus semantics**

Add tests covering:
- `h` and `left` switch focus to list pane
- `l` and `right` switch focus to preview pane
- `j/k` and `down/up` move selection when list focused
- `j/k` and `down/up` scroll preview when preview focused

**Step 2: Run tests to verify RED**

Run: `go test ./internal/cli/ui -run 'TestModel_.*PaneFocus.*|TestModel_.*VerticalMovement.*'`
Expected: FAIL due to current key-routing conflicts.

**Step 3: Write minimal test scaffolding helpers (if needed)**

Add helper setup in test file only if existing helpers are insufficient.

**Step 4: Re-run RED tests**

Run: `go test ./internal/cli/ui -run 'TestModel_.*PaneFocus.*|TestModel_.*VerticalMovement.*'`
Expected: still FAIL (correctly).

**Step 5: Commit (tests-only checkpoint)**

```bash
git add internal/cli/ui/model_test.go
git commit -m "test(ui): define pane-focus and vertical navigation expectations"
```

### Task 2: Implement browser key mapping alignment (GREEN)

**Files:**
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/keys.go`

**Step 1: Implement minimal key routing changes**

Implement behavior:
- Browser mode: `h/left` => list focus, `l/right` => preview focus
- Vertical movement remains `j/k` + `up/down`
- Remove/avoid conflicting category-switch behavior in browser mode

**Step 2: Run targeted tests (GREEN)**

Run: `go test ./internal/cli/ui -run 'TestModel_.*PaneFocus.*|TestModel_.*VerticalMovement.*'`
Expected: PASS.

**Step 3: Run broader UI tests**

Run: `go test ./internal/cli/ui`
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/cli/ui/model.go internal/cli/ui/keys.go internal/cli/ui/model_test.go
git commit -m "feat(ui): align browser navigation with vim and arrow motions"
```

### Task 3: Make enter a no-op for instructions/prompts/agents

**Files:**
- Modify: `internal/cli/ui/model_test.go`
- Modify: `internal/cli/ui/model.go`

**Step 1: Write failing tests**

Add tests verifying:
- `enter` on instructions/prompts/agents in browser does not transition to detail screen
- `enter` on skills still drills into detail

**Step 2: Run targeted tests (RED)**

Run: `go test ./internal/cli/ui -run 'TestModel_.*Enter.*(Instructions|Prompts|Agents|Skills)'`
Expected: FAIL for non-skill no-op behavior.

**Step 3: Implement minimal behavior change**

In browser key handling:
- if active kind is `skills`, keep existing `openShowModal()` behavior
- if active kind is `instructions/prompts/agents`, return without state transition

**Step 4: Run targeted tests (GREEN)**

Run: `go test ./internal/cli/ui -run 'TestModel_.*Enter.*(Instructions|Prompts|Agents|Skills)'`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/model.go internal/cli/ui/model_test.go
git commit -m "feat(ui): keep single-file resources in inline preview mode"
```

### Task 4: Update help/footer text to match new controls

**Files:**
- Modify: `internal/cli/ui/view.go`
- Modify: `internal/cli/ui/model_test.go`

**Step 1: Add/adjust failing tests for copy expectations**

Add tests for help/footer including:
- `h/l` and `left/right` focus semantics
- `enter` drills into skills only

**Step 2: Run tests to verify RED**

Run: `go test ./internal/cli/ui -run 'TestView_.*Help.*|TestView_.*Footer.*'`
Expected: FAIL due to stale hints.

**Step 3: Implement minimal copy updates**

Update help and footer strings only (no extra behavior changes).

**Step 4: Run targeted + full UI tests**

Run:
- `go test ./internal/cli/ui -run 'TestView_.*Help.*|TestView_.*Footer.*'`
- `go test ./internal/cli/ui`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/view.go internal/cli/ui/model_test.go
git commit -m "docs(ui): refresh key hints for pane focus and enter behavior"
```

### Task 5: README behavior sync

**Files:**
- Modify: `README.md`

**Step 1: Update keybinding section**

Document:
- `h/j/k/l` and arrow equivalence
- browser pane focus semantics
- `enter` drill-in scope (skills only)

**Step 2: Verify docs accuracy against implementation**

Run: `go test ./internal/cli/ui`
Expected: PASS (sanity check after docs sync).

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: clarify browser pane navigation and single-file preview behavior"
```

### Task 6: Final verification sweep

**Files:**
- Modify: any files requiring final small regression fixes

**Step 1: Run focused packages**

Run: `go test ./internal/cli ./internal/cli/ui`
Expected: PASS.

**Step 2: Run full suite + vet**

Run: `go test ./... && go vet ./...`
Expected: PASS and no vet issues.

**Step 3: If failures found, add regression test first**

Fix only after reproducing with a failing test.

**Step 4: Commit final stabilization fixes (if any)**

```bash
git add <changed-files>
git commit -m "test(ui): stabilize pane-navigation preview workflow"
```
