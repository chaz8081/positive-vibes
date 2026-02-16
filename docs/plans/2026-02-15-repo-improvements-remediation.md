# Repository Improvements Remediation Implementation Plan

> **Status: COMPLETED** -- All 9 tasks complete.

**Goal:** Resolve repo audit findings end-to-end, prioritizing security/correctness, then CI/testability, then cleanup and docs hygiene.

**Architecture:** Apply small boundary-focused fixes in `internal/target`, `internal/registry`, and `internal/engine` first, then harden CI and reduce debt in CLI/TUI. Keep changes incremental and test-first to reduce regression risk.

**Tech Stack:** Go 1.24+, Cobra, Bubble Tea, go-git, GitHub Actions.

---

### Task 1: Path safety foundation

**Files:**
- Create: `internal/fsutil/path_safety.go`
- Create: `internal/fsutil/path_safety_test.go`

**Steps:**
1. Write failing tests for empty/absolute/traversal/safe-relative paths.
2. Run: `go test ./internal/fsutil -run ResolveWithinRoot -v` (expect FAIL).
3. Implement `ResolveWithinRoot(root, rel)`.
4. Re-run same test (expect PASS).
5. Commit.

### Task 2: Apply path safety to targets and registry reads

**Files:**
- Modify: `internal/target/target.go`
- Modify: `internal/target/{copilot.go,opencode.go,cursor.go}`
- Modify: `internal/registry/git.go`
- Test: `internal/target/target_test.go`
- Test: `internal/registry/git_test.go`

**Steps:**
1. Write failing traversal tests in target/registry tests.
2. Run: `go test ./internal/target ./internal/registry -run "Traversal|UnknownKind" -v` (expect FAIL).
3. Implement bounded-path resolution and reject unknown resource kinds.
4. Re-run command (expect PASS).
5. Commit.

### Task 3: Stop local-path fallback masking in applier

**Files:**
- Modify: `internal/engine/applier.go`
- Test: `internal/engine/applier_test.go`

**Steps:**
1. Add failing test: invalid local `SKILL.md` does not fallback to registry same-name skill.
2. Run: `go test ./internal/engine -run LocalPathSkillParseErrorDoesNotFallbackToRegistry -v` (expect FAIL).
3. Return explicit local parse/read error and skip fallback.
4. Re-run command (expect PASS).
5. Commit.

### Task 4: Embedded registry temp lifecycle

**Files:**
- Modify: `internal/registry/embedded.go`
- Modify: `internal/registry/registry.go`
- Modify: `internal/engine/applier.go`
- Test: `internal/registry/registry_test.go`

**Steps:**
1. Add failing test asserting embedded fetch temp path is cleaned up after use.
2. Run: `go test ./internal/registry -run Embedded -v` (expect FAIL).
3. Add cleanup contract and invoke it from applier after install.
4. Re-run test (expect PASS).
5. Commit.

### Task 5: Git registry hardening

**Files:**
- Modify: `internal/registry/git.go`
- Test: `internal/registry/git_test.go`

**Steps:**
1. Add failing tests for invalid cache-without-`.git` and unsafe URL prefix.
2. Run: `go test ./internal/registry -run "Cache|URL" -v` (expect FAIL).
3. Validate cache integrity and reject unsafe URL forms.
4. Re-run command (expect PASS).
5. Commit.

### Task 6: CI and release readiness

**Files:**
- Modify: `.github/workflows/go.yml`
- Modify: `.github/workflows/go-ossf-slsa3-publish.yml`
- Create: `.slsa-goreleaser.yml`
- Modify: `README.md`

**Steps:**
1. Align Go versions and add vet/race checks in CI.
2. Add minimal SLSA builder config file.
3. Verify local parity: `go vet ./... && go test ./...`.
4. Commit.

### Task 7: Hygiene and docs cleanup

**Files:**
- Modify: `.gitignore`
- Modify: `README.md`
- Modify: `docs/plans/*.md` (status cleanup)

**Steps:**
1. Add `.opencode/` ignore rule.
2. Update README prerequisites and audit-driven notes.
3. Mark stale plan docs as completed/archived style.
4. Commit.

### Task 8: CLI/TUI correctness and debt reduction

**Files:**
- Modify: `internal/cli/ui/{view.go,model.go}`
- Modify: `internal/cli/{config.go,list.go,install.go,remove.go,root.go,init.go,resource.go}`
- Test: related `*_test.go`

**Steps:**
1. Add failing tests for View purity and error propagation.
2. Implement fixes (no state mutation in `View`, no swallowed errors, stderr usage in main).
3. Remove dead code and deduplicate helpers.
4. Re-run package tests.
5. Commit.

### Task 9: Final verification

**Files:**
- Modify: any touched files as needed

**Steps:**
1. Run: `go test ./...`
2. Run: `go vet ./...`
3. Run: `go test -race ./internal/...`
4. Run smoke: `go run ./cmd/positive-vibes --help` and `go run ./cmd/positive-vibes config paths`
5. Commit final polish.
