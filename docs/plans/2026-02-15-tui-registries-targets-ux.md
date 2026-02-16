# TUI Registries + Targets UX Improvement Plan

> **Status: COMPLETED** -- Implemented in `feature/tui-ux-overhaul` and merged to `main`.

**Goal:** Make registries and targets management in the TUI intuitive, safe, and portable, with global-first registry management, local override awareness, and guided remediation for invalid/conflicting local registry states.

**Architecture:** Extend existing resource/TUI scope model with source-state metadata for registries and targets. Add registry rail entry hooks for auto-promotion, safe delete preflight checks, and actionable row states (`L!`). Keep local manual overrides authoritative in merge behavior while making global catalog management explicit.

**Tech Stack:** Go, Cobra CLI, Bubble Tea TUI, manifest merge/validation model, existing resource actions and UI service bridge.

---

## Locked Product Decisions

- Registry CRUD in TUI is **global-only**.
- Manual local config overrides continue to win in merged/effective behavior.
- Entering Registries rail auto-promotes valid local-only registries into global.
- If same registry name exists globally with different URL, promotion is skipped with warning (safe behavior).
- Global registry delete is blocked when any local resources reference that registry.
- `L!` row state indicates unresolved local-only issue (invalid or conflict).
- Pressing `space` on `L!` opens guided fix dialog.
- Include Targets UX improvements in this same increment.

## UX Spec

### Registries

- Row source markers:
  - `G` = global only
  - `B` = global + local
  - `L!` = local-only unresolved (invalid/conflict), requires user action
- On entering Registries rail:
  - detect local-only registries
  - auto-promote valid/no-conflict entries to global
  - show summary status (`Promoted X, skipped Y`)
- Conflict handling:
  - same name + different URL => no auto-promotion, row marked `L!`, reason shown
- Delete flow:
  - preflight dependency graph across local resources (skills/instructions/agents/prompts)
  - if any local refs exist => hard block with clear reason
- `space` behavior:
  - normal rows => existing scope cycle behavior
  - `L!` rows => guided fix dialog

### Targets

- Show source-state in row/detail:
  - inherited from global vs local override
- First local target mutation in project context:
  - one-time confirm: create local target override from effective set
- Add explicit action: reset to inherited targets (clear local targets override)
- Keep current list toggling ergonomics consistent with other resource rails

## Guided Fix Dialog for `L!` Registry Rows

Trigger: `space` on `L!` row

Options:
1. Promote now (only when valid and conflict-free)
2. Open local config hint (show exact file + field to fix)
3. Rename local registry (safe conflict resolution workflow)
4. Cancel

Dialog body always includes:
- reason code (`missing ref`, `missing url`, `name conflict: URL differs`, etc.)
- exact remediation hint

## Implementation Tasks

### Task 1: Registry source-state model + diagnostics surface
- Modify: `internal/cli/resource.go`
- Modify: `internal/manifest/manifest.go`
- Test: `internal/cli/resource_test.go`
- Test: `internal/manifest/manifest_test.go`

### Task 2: Auto-promotion engine on Registries rail entry
- Modify: `internal/cli/resource_actions.go`
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/service.go`
- Test: `internal/cli/resource_actions_test.go`
- Test: `internal/cli/ui/model_test.go`

### Task 3: Conflict-safe promotion rules
- Modify: `internal/cli/resource_actions.go`
- Test: `internal/cli/resource_actions_test.go`

### Task 4: Delete block preflight for registry dependencies
- Modify: `internal/cli/resource_actions.go`
- Modify: `internal/cli/resource.go`
- Test: `internal/cli/resource_actions_test.go`
- Test: `internal/cli/resource_test.go`

### Task 5: `L!` row rendering + guided fix dialog
- Modify: `internal/cli/ui/view.go`
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/styles.go`
- Test: `internal/cli/ui/model_test.go`
- Test: `internal/cli/ui/show_modal_test.go`

### Task 6: Targets source-awareness + override UX
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/view.go`
- Modify: `internal/cli/resource_actions.go`
- Test: `internal/cli/ui/model_test.go`
- Test: `internal/cli/resource_actions_test.go`

### Task 7: Help text, docs, and behavior docs sync
- Modify: `README.md`
- Modify: `internal/cli/ui/view.go`
- Test: `internal/cli/ui/model_test.go`

### Task 8: Verification and regression sweep
- `go test ./internal/manifest ./internal/cli ./internal/cli/ui ./internal/engine ./internal/registry ./internal/target`
- `go test ./...`
- `go vet ./...`

## Execution Tracker

- [ ] Task 1: Registry source-state model + diagnostics
- [ ] Task 2: Auto-promotion on Registries rail entry
- [ ] Task 3: Conflict-safe promotion rules
- [ ] Task 4: Delete block preflight by local dependencies
- [ ] Task 5: `L!` rendering + guided fix dialog
- [ ] Task 6: Targets inherited/override/reset UX
- [ ] Task 7: Docs/help/hint text updates
- [ ] Task 8: Full verification suite
