# Agent Instructions

This repository is a Go project (module: `github.com/chaz8081/positive-vibes`, Go 1.24+).
Use these notes for build/test and code style. If you need more context, read
the README and package-level docs, then follow existing patterns.

## Issue Tracking (bd)

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

Quick reference:

- `bd ready` - Find unblocked work
- `bd create "Title" --type task --priority 2` - Create issue
- `bd close <id>` - Complete work
- `bd sync` - Sync with git (run at session end)

For full workflow details: `bd prime`.

## Build, Test, Lint

### Build

```bash
go build ./...
go build -o positive-vibes ./cmd/positive-vibes
```

### Test (full suite)

```bash
go test ./...
```

### Run a single package

```bash
go test ./internal/engine -v
```

### Run a single test

Use `-run` with a regex:

```bash
go test ./internal/engine -run ApplyManifest_ -v
go test ./internal/cli -run Resource -v
```

### Race + Vet (when doing deeper changes)

```bash
go test -race ./internal/...
go vet ./...
```

### Smoke

```bash
go run ./cmd/positive-vibes --help
go run ./cmd/positive-vibes config paths
```

## Code Style & Conventions

### Formatting

- Use `gofmt` formatting (default for all Go files).
- Keep code paths short and readable; prefer early returns on error.

### Imports

- Use standard Go import grouping (std, third-party, local) as gofmt formats it.
- Avoid unused imports; tests should compile cleanly.

### Packages and file layout

- Keep package boundaries clear: `internal/cli`, `internal/engine`,
  `internal/registry`, `internal/target`, `internal/manifest`, `pkg/schema`.
- Add new helpers in the most specific package; avoid cross-package cycles.

### Naming

- Use clear, domain-specific names (e.g., `ApplyManifest`, `ResourceItem`).
- Prefer `Resource*` naming when representing skills/instructions/agents/prompts.
- Avoid introducing new abbreviations unless already common in the codebase.

### Error handling

- Return errors instead of printing from deep helpers.
- Wrap errors with context using `fmt.Errorf("...: %w", err)` when helpful.
- For CLI commands, prefer Cobra `RunE` and return errors; use stderr for
  warnings and failures.

### Tests

- Follow existing patterns: `*_test.go` near the package.
- Use table tests when data-driven and avoid brittle string assertions.
- For isolated helpers, add unit tests; for CLI flows, use existing test helpers.

### CLI behavior

- Use `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` when printing from command handlers.
- Keep user-facing strings consistent with existing commands.

### Paths and files

- Use `filepath.Join` for filesystem paths.
- Avoid hardcoding separators or assuming OS-specific paths.

## Repository Notes

- There are no Cursor rules in `.cursor/rules/` or `.cursorrules`.
- No Copilot instructions file found at `.github/copilot-instructions.md`.

## Landing the Plane (Session Completion)

When ending a work session, complete all steps below. Work is NOT complete
until `git push` succeeds.

1. File issues for remaining work (bd)
2. Run quality gates (tests/lint/build)
3. Update issue status (close/claim)
4. Push:

```bash
git pull --rebase
bd sync
git push
git status
```

5. Clean up (stashes, old branches)
6. Hand off context for next session
