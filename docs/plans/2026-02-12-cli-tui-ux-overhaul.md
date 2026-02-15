# CLI TUI UX Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a keyboard-first full-screen TUI that launches on no-args (TTY only) and supports install/remove/show flows for skills, instructions, and agents with modal overlays.

**Architecture:** Introduce a dedicated Bubble Tea app under `internal/cli/ui` with one persistent screen (resource rail, table, preview pane, footer) and modal overlays for install/remove/show. Keep existing command handlers intact for scripting, and route TUI actions through a small service layer that wraps existing manifest/registry logic. Add root command no-arg behavior to start TUI only in interactive terminals and fallback to help in non-interactive contexts.

**Tech Stack:** Go, Cobra, Bubble Tea/Bubbles/Lip Gloss, existing `internal/manifest`, `internal/registry`, `internal/engine` packages.

---

### Task 1: Add TUI dependencies and command entry

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/cli/ui/app.go`
- Modify: `internal/cli/root.go`
- Test: `internal/cli/apply_test.go`

**Step 1: Write the failing test**

Add a root behavior test asserting no-args on a TTY path calls the UI launcher function instead of only help fallback (inject launcher function for testability).

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestRootNoArgs_LaunchesUIOnTTY`
Expected: FAIL because UI launch hook does not exist.

**Step 3: Write minimal implementation**

- Add Bubble Tea dependencies.
- Create `internal/cli/ui/app.go` with `Run() error` placeholder.
- Add root no-args dispatch in `internal/cli/root.go`:
  - if interactive TTY: call injected `launchUI` (default `ui.Run`)
  - else: show help.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestRootNoArgs_LaunchesUIOnTTY`
Expected: PASS.

**Step 5: Commit**

```bash
git add go.mod go.sum internal/cli/ui/app.go internal/cli/root.go internal/cli/apply_test.go
git commit -m "feat(ui): launch tui on no-args in interactive terminals"
```

### Task 2: Build base TUI layout and keyboard model

**Files:**
- Create: `internal/cli/ui/model.go`
- Create: `internal/cli/ui/view.go`
- Create: `internal/cli/ui/keys.go`
- Create: `internal/cli/ui/styles.go`
- Create: `internal/cli/ui/model_test.go`

**Step 1: Write the failing test**

Add tests for keyboard navigation state:
- switching resource rail tabs
- table cursor movement
- opening/closing help modal.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run TestModel_NavigationAndHelpModal`
Expected: FAIL because model does not exist.

**Step 3: Write minimal implementation**

Implement Bubble Tea model with:
- left rail (`skills`, `instructions`, `agents`)
- center list cursor
- right details preview panel
- footer key hints
- help modal (`?` open, `esc` close).

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui -run TestModel_NavigationAndHelpModal`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/model.go internal/cli/ui/view.go internal/cli/ui/keys.go internal/cli/ui/styles.go internal/cli/ui/model_test.go
git commit -m "feat(ui): add base tui layout and keyboard navigation"
```

### Task 3: Add service layer for resource discovery/install/remove/show

**Files:**
- Create: `internal/cli/ui/service.go`
- Create: `internal/cli/ui/service_test.go`
- Modify: `internal/cli/resource.go`

**Step 1: Write the failing test**

Add tests that service returns merged available+installed rows for each resource type and can resolve detail payload for show.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run TestService_ListAndShow`
Expected: FAIL because service methods are missing.

**Step 3: Write minimal implementation**

Implement service methods:
- `ListResources(kind)`
- `ShowResource(kind, name)`
- `InstallResources(kind, names)`
- `RemoveResources(kind, names)`

Use existing manifest/registry helpers from `internal/cli/resource.go` and existing install/remove behavior.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui -run TestService_ListAndShow`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/service.go internal/cli/ui/service_test.go internal/cli/resource.go
git commit -m "feat(ui): add resource service layer for list/show/install/remove"
```

### Task 4: Implement install modal overlay

**Files:**
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/view.go`
- Create: `internal/cli/ui/install_modal_test.go`

**Step 1: Write the failing test**

Add tests that `i` opens install modal, toggles selections, confirms install action, and refreshes list state.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run TestInstallModal_Flow`
Expected: FAIL because install modal flow is missing.

**Step 3: Write minimal implementation**

Implement install modal overlay:
- opens with not-installed rows
- supports multi-select (`space`)
- confirm (`enter`) calls service install
- close (`esc`) with no mutation.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui -run TestInstallModal_Flow`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/model.go internal/cli/ui/view.go internal/cli/ui/install_modal_test.go
git commit -m "feat(ui): add install modal with multi-select"
```

### Task 5: Implement remove modal overlay

**Files:**
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/view.go`
- Create: `internal/cli/ui/remove_modal_test.go`

**Step 1: Write the failing test**

Add tests that `r` opens remove modal for installed items, confirms deletion, and list refreshes.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run TestRemoveModal_Flow`
Expected: FAIL because remove modal flow is missing.

**Step 3: Write minimal implementation**

Implement remove modal overlay:
- prefilled with installed rows
- multi-select and confirm removal
- safe cancel path.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui -run TestRemoveModal_Flow`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/model.go internal/cli/ui/view.go internal/cli/ui/remove_modal_test.go
git commit -m "feat(ui): add remove modal for installed resources"
```

### Task 6: Implement show detail modal overlay

**Files:**
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/view.go`
- Create: `internal/cli/ui/show_modal_test.go`

**Step 1: Write the failing test**

Add tests that `enter` opens show modal with full detail for current selection, and closes with `esc`.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run TestShowModal_Flow`
Expected: FAIL because show modal flow is missing.

**Step 3: Write minimal implementation**

Implement show modal overlay:
- uses service `ShowResource`
- displays source metadata, path/registry, and content preview.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui -run TestShowModal_Flow`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/ui/model.go internal/cli/ui/view.go internal/cli/ui/show_modal_test.go
git commit -m "feat(ui): add show modal with detailed resource preview"
```

### Task 7: Wire command parity hooks for milestone-2 commands

**Files:**
- Modify: `internal/cli/install.go`
- Modify: `internal/cli/remove.go`
- Modify: `internal/cli/show.go`
- Create: `internal/cli/ui/integration_test.go`

**Step 1: Write the failing test**

Add integration tests to ensure TUI install/remove/show actions mutate manifest consistently with existing commands.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run TestUI_InstallRemoveShowParity`
Expected: FAIL due to missing parity wiring.

**Step 3: Write minimal implementation**

Centralize shared mutation logic callable by both command handlers and TUI service where necessary.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui -run TestUI_InstallRemoveShowParity`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/cli/install.go internal/cli/remove.go internal/cli/show.go internal/cli/ui/integration_test.go
git commit -m "refactor(cli): share install/remove/show logic with tui service"
```

### Task 8: Update docs and verify full repo

**Files:**
- Modify: `README.md`
- Modify: `internal/cli/root.go`

**Step 1: Write/update failing docs expectation test (if applicable)**

If no doc tests exist, skip to Step 2.

**Step 2: Run verification commands**

Run: `go test ./... && go vet ./...`
Expected: PASS.

**Step 3: Update docs**

Document:
- no-args TUI behavior (TTY only)
- keybindings
- install/remove/show modal flows
- how to use classic subcommands in scripts.

**Step 4: Re-run verification**

Run: `go test ./... && go vet ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add README.md internal/cli/root.go
git commit -m "docs(ui): document interactive tui mode and keybindings"
```
