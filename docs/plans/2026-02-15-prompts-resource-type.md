# Prompts Resource Type Implementation Plan

> **Status: COMPLETED** -- Implemented in `feature/tui-ux-overhaul` and merged to `main`.

**Goal:** Add a first-class `prompts` resource type that is file-based, works in CLI/TUI/list/show/install/remove/apply flows, installs to Copilot and OpenCode target locations, and skips Cursor with warning.

**Architecture:** Extend the existing resource-family pattern used by `instructions` and `agents` to include `prompts`, while preserving prompt-specific semantics (path-based only and manual macro files). Reuse manifest merge/override logic, registry path roots, and UI scope toggling primitives. Keep apply behavior target-aware: install for `vscode-copilot` and `opencode`, record skipped ops for `cursor`.

**Tech Stack:** Go, Cobra CLI, Bubble Tea TUI, YAML manifest model, existing registry/target abstractions, Go test framework.

---

### Task 1: Manifest model + validation + merge for prompts

**Files:**
- Modify: `internal/manifest/manifest.go`
- Test: `internal/manifest/manifest_test.go`

**Step 1: Write the failing test**

```go
func TestManifestValidate_PromptsRequirePath(t *testing.T) {
	m := &Manifest{
		Prompts: []PromptRef{{Name: "release-checklist"}},
		Targets: []string{"opencode"},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected validate error when prompt path missing")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/manifest -run TestManifestValidate_PromptsRequirePath`
Expected: FAIL because `PromptRef` / prompt validation does not exist yet.

**Step 3: Write minimal implementation**

```go
type PromptRef struct {
	Name     string `yaml:"name"`
	Registry string `yaml:"registry,omitempty"`
	Path     string `yaml:"path"`
}

type Manifest struct {
	// ...existing...
	Prompts []PromptRef `yaml:"prompts,omitempty"`
}
```

Also extend:
- `Validate()` for prompt constraints (`name` + `path` required)
- merge rules in `LoadMergedManifest()`
- `ResolveManifestPaths()` for non-registry prompt paths
- override diagnostics structs/maps/sorting for prompts

**Step 4: Run test to verify it passes**

Run: `go test ./internal/manifest`
Expected: PASS, including prompt-specific and existing merge/validation tests.

**Step 5: Commit**

```bash
git add internal/manifest/manifest.go internal/manifest/manifest_test.go
git commit -m "feat(manifest): add first-class prompts resource model"
```

### Task 2: Registry prompt path support (`paths.prompts`) and file enumeration

**Files:**
- Modify: `internal/registry/registry.go`
- Modify: `internal/registry/git.go`
- Test: `internal/registry/git_test.go`

**Step 1: Write the failing test**

```go
func TestGitRegistry_ListResourceFiles_PromptsKind(t *testing.T) {
	// repo fixtures include prompts/checklist.prompt.md and prompts/refactor.md
	// expect both returned under kind="prompts"
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/registry -run PromptsKind`
Expected: FAIL because `prompts` is not a supported resource kind.

**Step 3: Write minimal implementation**

```go
// ResourceSource kind includes: "skills", "instructions", "agents", "prompts"
case "prompts":
	p = r.PromptsPath
```

Add `PromptsPath` field to `GitRegistry` and wire default behavior like existing kind paths.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/registry`
Expected: PASS for prompt file listing/fetching and legacy resource behavior.

**Step 5: Commit**

```bash
git add internal/registry/registry.go internal/registry/git.go internal/registry/git_test.go
git commit -m "feat(registry): support prompts resource roots and file access"
```

### Task 3: Target interface + per-target prompt install behavior

**Files:**
- Modify: `internal/target/target.go`
- Modify: `internal/target/copilot.go`
- Modify: `internal/target/opencode.go`
- Modify: `internal/target/cursor.go`
- Test: `internal/target/target_test.go`

**Step 1: Write the failing test**

```go
func TestTargets_InstallPrompt_Destinations(t *testing.T) {
	// copilot => .github/prompts/<name>.prompt.md
	// opencode => .opencode/commands/<name>.md
	// cursor => skipped/unsupported error marker (handled by applier)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/target -run InstallPrompt`
Expected: FAIL because target interface lacks prompt methods.

**Step 3: Write minimal implementation**

```go
type Target interface {
	PromptDir() string
	InstallPrompt(name, sourcePath, projectRoot string, opts InstallOpts) error
}
```

Implement prompt destination mapping:
- Copilot: `.github/prompts` and `<name>.prompt.md`
- OpenCode: `.opencode/commands` and `<name>.md`
- Cursor: return sentinel unsupported error consumed by applier

**Step 4: Run test to verify it passes**

Run: `go test ./internal/target`
Expected: PASS for prompt path assertions and existing skill/instruction/agent tests.

**Step 5: Commit**

```bash
git add internal/target/target.go internal/target/copilot.go internal/target/opencode.go internal/target/cursor.go internal/target/target_test.go
git commit -m "feat(target): add prompt install support for copilot and opencode"
```

### Task 4: Apply engine prompt operations and cursor skip warnings

**Files:**
- Modify: `internal/engine/applier.go`
- Test: `internal/engine/applier_test.go`

**Step 1: Write the failing test**

```go
func TestApplierApply_PromptsInstallAndCursorSkip(t *testing.T) {
	// manifest has one prompt and targets: copilot/opencode/cursor
	// expect 2 installed ops, 1 skipped op(kind=prompt), no hard error
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -run PromptsInstallAndCursorSkip`
Expected: FAIL because prompt op kind/install path does not exist.

**Step 3: Write minimal implementation**

```go
const KindPrompt ApplyOpKind = "prompt"
```

Add prompt iteration block in `ApplyManifest()`:
- resolve local/registry source file
- install on targets supporting prompts
- cursor creates `OpSkipped` with clear reason

**Step 4: Run test to verify it passes**

Run: `go test ./internal/engine`
Expected: PASS with correct prompt op counts and statuses.

**Step 5: Commit**

```bash
git add internal/engine/applier.go internal/engine/applier_test.go
git commit -m "feat(engine): apply prompts across targets with cursor skip"
```

### Task 5: CLI resource type + install/list/show/remove/completion for prompts

**Files:**
- Modify: `internal/cli/resource.go`
- Modify: `internal/cli/install.go`
- Modify: `internal/cli/list.go`
- Modify: `internal/cli/show.go`
- Modify: `internal/cli/remove.go`
- Modify: `internal/cli/resource_actions.go`
- Test: `internal/cli/resource_test.go`
- Test: `internal/cli/resource_actions_test.go`

**Step 1: Write the failing test**

```go
func TestParseResourceType_Prompts(t *testing.T) {
	typeVal, err := ParseResourceType("prompts")
	if err != nil || typeVal != ResourcePrompts {
		t.Fatalf("unexpected parse result: %v %v", typeVal, err)
	}
}
```

Add tests for:
- prompt install/remove mutation reports
- list/show formatting and shell completions
- registry prompt discovery filter (`*.prompt.md` and `*.md`)

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run Prompts`
Expected: FAIL due to missing prompts resource type branches.

**Step 3: Write minimal implementation**

```go
const ResourcePrompts ResourceType = "prompts"
```

Extend resource collection/helpers to include prompts and prompt-specific filtering.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli`
Expected: PASS for prompts command flows + existing resource tests.

**Step 5: Commit**

```bash
git add internal/cli/resource.go internal/cli/install.go internal/cli/list.go internal/cli/show.go internal/cli/remove.go internal/cli/resource_actions.go internal/cli/resource_test.go internal/cli/resource_actions_test.go
git commit -m "feat(cli): add prompts resource commands and completions"
```

### Task 6: TUI prompts rail + scope handling via existing cycle

**Files:**
- Modify: `internal/cli/ui/service.go`
- Modify: `internal/cli/ui/model.go`
- Modify: `internal/cli/ui/view.go`
- Modify: `internal/cli/ui/ui_bridge.go`
- Test: `internal/cli/ui/model_test.go`
- Test: `internal/cli/ui/service_test.go`
- Test: `internal/cli/ui/integration_test.go`

**Step 1: Write the failing test**

```go
func TestModel_HomeCategoriesIncludePrompts(t *testing.T) {
	m := newModel()
	if !contains(m.homeCategories(), "prompts") {
		t.Fatal("prompts category missing")
	}
}
```

Add tests for prompt rows in browser + scope cycling (`none/local/global/both`).

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ui -run Prompts`
Expected: FAIL since UI kinds and service validation exclude prompts.

**Step 3: Write minimal implementation**

```go
const resourceKindPrompts = "prompts"
```

Extend service kind validation and model/view category wiring.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ui`
Expected: PASS with prompts listed and toggled like other resources.

**Step 5: Commit**

```bash
git add internal/cli/ui/service.go internal/cli/ui/model.go internal/cli/ui/view.go internal/cli/ui_bridge.go internal/cli/ui/model_test.go internal/cli/ui/service_test.go internal/cli/ui/integration_test.go
git commit -m "feat(ui): add prompts category and scoped toggling"
```

### Task 7: Config diagnostics + init template + docs

**Files:**
- Modify: `internal/cli/config.go`
- Modify: `internal/cli/init.go`
- Modify: `README.md`
- Test: `internal/cli/config_test.go` (or nearest existing config test file)

**Step 1: Write the failing test**

```go
func TestValidateConfig_PromptsRegistryReference(t *testing.T) {
	// prompt with registry should validate registry name existence
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run ValidateConfig_Prompts`
Expected: FAIL due to missing prompt config checks.

**Step 3: Write minimal implementation**

Add prompts into config summary/validation output and init template comments/examples.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli`
Expected: PASS with prompt diagnostics reflected.

**Step 5: Commit**

```bash
git add internal/cli/config.go internal/cli/init.go README.md internal/cli/config_test.go
git commit -m "docs(cli): document prompts resources and config diagnostics"
```

### Task 8: Full verification + cleanup pass

**Files:**
- Modify: any touched files from prior tasks (only if fixes are required)

**Step 1: Write failing regression tests first (if any bug is found)**

```go
// Add package-specific regression test only for discovered breakage
```

**Step 2: Run package suites**

Run: `go test ./internal/manifest ./internal/registry ./internal/target ./internal/engine ./internal/cli ./internal/cli/ui`
Expected: all PASS.

**Step 3: Run full verification**

Run: `go test ./... && go vet ./...`
Expected: all PASS, no vet issues.

**Step 4: Final doc/readability pass**

Confirm `README.md` matches actual key behavior and supported types.

**Step 5: Final commit (if verification fixes were needed)**

```bash
git add <changed-files>
git commit -m "test: close prompts integration gaps found in verification"
```
