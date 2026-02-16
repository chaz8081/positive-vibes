# positive-vibes

[![Go](https://github.com/chaz8081/positive-vibes/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/chaz8081/positive-vibes/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/license-MIT-blue?style=for-the-badge)](LICENSE)

> Harmonize your AI tooling. One manifest to rule them all.

positive-vibes is an environment-agnostic configuration manager for AI tooling. It aligns your AI tools -- VS Code Copilot, OpenCode, Cursor, and more -- from a single source of truth.

Every AI coding tool has its own way of configuring resources like skills and instructions. You end up maintaining the same context in `.github/skills/`, `.opencode/skills/`, `.cursor/skills/`... separately.

positive-vibes gives you one `vibes.yaml` to define your resources, then syncs them everywhere.

## Prerequisites

- **Go 1.25+** -- required to build and install
- **git** -- required for registry cloning and cache management

## Quick Start

### Install

```bash
go install github.com/chaz8081/positive-vibes/cmd/positive-vibes@latest
```

#### From source

```bash
git clone https://github.com/chaz8081/positive-vibes.git
cd positive-vibes
go build -o positive-vibes ./cmd/positive-vibes
./positive-vibes --help
```

To install to your `$GOPATH/bin`:

```bash
go install ./cmd/positive-vibes
```

### Initialize

```bash
positive-vibes init
```

This scans your project, detects the language (Go, Node, Python), and creates a starter `vibes.yaml` with recommended skills and a commented header explaining each section.

### Add Skills

```bash
positive-vibes install skills conventional-commits
```

To add an instruction entry by name (uses registry when available, otherwise creates a local path-based entry):

```bash
positive-vibes install instructions coding-standards
```

To add an agent entry by name (uses registry when available, otherwise creates a local path-based entry):

```bash
positive-vibes install agents code-reviewer
```

### Apply

```bash
positive-vibes apply
```

This reads your manifest and installs configured resources (skills, instructions, agents, prompts) into your target tools' directories.

## The Manifest (`vibes.yaml`)

The example below is runnable in this repo. First create local instruction/agent/prompt files:

```bash
mkdir -p instructions agents prompts
printf "Keep responses concise and actionable.\n" > instructions/repo-guidelines.md
printf "# Local Reviewer\nFocus on correctness, readability, and tests.\n" > agents/local-reviewer.md
printf "---\ndescription: Release checklist\n---\nConfirm tests pass and changelog is updated.\n" > prompts/release.prompt.md
```

```yaml
registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
    paths:
      skills: .
      instructions: .
      agents: .
      prompts: .

skills:
  - name: gh-cli
    registry: awesome-copilot
    path: skills/gh-cli
  - name: local-conventional-commits
    path: ./skills/conventional-commits

instructions:
  - name: markdown-guidelines
    registry: awesome-copilot
    path: instructions/markdown.instructions.md
  - name: repo-guidelines
    path: ./instructions/repo-guidelines.md

agents:
  - name: debug-agent
    registry: awesome-copilot
    path: agents/debug.agent.md
  - name: local-reviewer
    path: ./agents/local-reviewer.md

prompts:
  - name: release
    path: ./prompts/release.prompt.md

targets:
  - vscode-copilot
  - opencode
  - cursor
```

Instruction entries are object-based: each item must include `name` and one source: `content` or `path`.
When `registry` is set, use `path` (file path inside that registry).

Agent entries are object-based: each item must include `name` and `path`; add `registry` when the path is inside a registry.

Registry-backed resources use `registry: <name>` + `path`. For skills, `path` is a folder inside the registry. For instructions, agents, and prompts, `path` is a file inside the registry.

Registry paths default to repo root (`.`) for all resource types. You can override each independently with `registries[].paths.skills`, `registries[].paths.instructions`, `registries[].paths.agents`, and `registries[].paths.prompts`.

When adding registries through the current TUI flow, missing registry paths are normalized to resource-specific defaults: `skills/`, `instructions/`, `agents/`, and `prompts/`.

`config validate` returns an error when a project resource references a registry that exists only in global config, to keep project manifests portable.

## Layered Configuration

positive-vibes supports a global + project layered config:

| Level       | Location                              | Purpose                                                     |
| ----------- | ------------------------------------- | ----------------------------------------------------------- |
| **Global**  | `~/.config/positive-vibes/vibes.yaml` | User-level defaults (personal registries, shared resources) |
| **Project** | `./vibes.yaml`                        | Project-specific resources and targets                      |

### Merge behavior

When both exist, they are merged:

- **Registries**: combined by name; project overrides global for same name
- **Skills**: combined by name; project overrides global for same name
- **Instructions**: combined by name; project overrides global for same name
- **Agents**: combined by name; project overrides global for same name
- **Prompts**: combined by name; project overrides global for same name
- **Targets**: project targets override global entirely
- **Paths**: relative `path` entries are resolved from the manifest they came from
- **Warnings**: `config validate` warns on risky overrides that change source type (e.g., `content` -> `path`, or registry -> path)

The global config path respects `$XDG_CONFIG_HOME` if set.

## Registry Versioning

Every registry entry requires a `ref` field that controls which version of the registry is used. This makes your setup reproducible and explicit.

### Ref types

| Ref value                            | Behavior                                                                                 |
| ------------------------------------ | ---------------------------------------------------------------------------------------- |
| `latest`                             | Track the registry's default branch. `positive-vibes apply --refresh` pulls new changes. |
| Branch name (e.g. `main`, `develop`) | Pin to a specific branch. Refresh is a no-op.                                            |
| Tag name (e.g. `v1.2.0`)             | Pin to a tagged release. Refresh is a no-op.                                             |
| Commit SHA (7-40 hex chars)          | Pin to an exact commit. Refresh is a no-op.                                              |

### Examples

```yaml
registries:
  # Track the latest skills (auto-updates on refresh)
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
    paths:
      skills: skills/

  # Example with custom roots per resource type
  - name: team-content
    url: https://github.com/myorg/team-content
    ref: main
    paths:
      skills: packages/skills
      instructions: docs/instructions
      agents: docs/agents
      prompts: docs/prompts

  # Pin to a stable release
  - name: team-skills
    url: https://github.com/myorg/team-skills
    ref: v2.1.0

  # Pin to an exact commit for reproducibility
  - name: audited-skills
    url: https://github.com/myorg/audited-skills
    ref: a1b2c3d4e5f6
```

### How pinning works

- **`latest`**: Clones the default branch. Running `positive-vibes apply --refresh` pulls new commits, so you always get the newest skills.
- **Pinned refs** (branch, tag, or SHA): The registry is cloned once at that ref and cached. Refresh does nothing -- to update, change the `ref` value in your manifest.
- If a clone fails but a previous cache exists, the cached copy is used as a fallback.

## Commands

### Interactive mode (no args)

Running `positive-vibes` with no subcommand launches an interactive TUI when `stdout` is a TTY.

- In a terminal session, `positive-vibes` opens the resource browser UI.
- In non-interactive contexts (CI, redirected output, scripts), `positive-vibes` prints help instead of launching the TUI.

This keeps automation predictable while giving humans a fast default experience.

### TUI keybindings

| Key           | Action                                                                                             |
| ------------- | -------------------------------------------------------------------------------------------------- |
| `left` / `h`  | Focus list pane                                                                                    |
| `right` / `l` | Focus preview pane                                                                                 |
| `up` / `down` | Move cursor up/down or scroll preview (by focus)                                                   |
| `j` / `k`     | Vim-style vertical movement/scroll                                                                 |
| `tab`         | Toggle list/preview focus                                                                          |
| `enter`       | Open detail view for multi-file skills (no-op for single-file skills, instructions/prompts/agents) |
| `space`       | Cycle install scope for selected row (`none -> local -> global -> both -> none`)                   |
| `r`           | In `targets`, reset local override back to inherited global targets                                |
| `/`           | Open search                                                                                        |
| `?`           | Open help overlay                                                                                  |
| `esc`         | Universal back/close (`detail -> browser -> home -> quit`)                                         |

Current top-level categories in the TUI: `skills`, `instructions`, `prompts`, `agents`, `targets`, `registries`.

When entering the `registries` rail, the TUI auto-promotes valid local-only registries into global config. If promotion is skipped (for example, name collision with different URL, or invalid local registry), the row is marked `L!` and `space` opens a guided fix dialog.

### Install scope markers

Browser rows show install scope markers:

- `L` = installed in local `./vibes.yaml`
- `G` = installed in global `~/.config/positive-vibes/vibes.yaml`
- `B` = installed in both local and global
- blank = not installed

For registries, an additional marker can appear:

- `L!` = local registry needs attention (invalid local config or safe conflict that blocked auto-promotion)

`space` cycles install scope in this order:

- blank -> `L`
- `L` -> `G`
- `G` -> `B`
- `B` -> blank

### Detail view

- **Show**: for **multi-file** skills, highlight and press `enter` to open full detail view (kind/name/status/paths/payload preview).
- **Single-file skills**: preview inline in browser right pane; `enter` does not switch screens.
- **Single-file types**: `instructions`, `prompts`, and `agents` preview inline in the browser right pane; `enter` does not switch screens.
- **Skills**: file list on left and preview on right; use `j/k` for file selection and preview scrolling (with focus).
- **Registries**: detail includes effective roots for `skills`, `instructions`, `agents`, and `prompts`, with inherited values noted.

### Script-safe classic subcommands

For scripts and CI, use explicit subcommands instead of relying on no-args behavior:

| Command                                                 | Description                                                                                                  |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `positive-vibes init`                                   | Scan project and create `vibes.yaml`                                                                         |
| `positive-vibes install <resource-type> [name...]`      | Add resources to your manifest (`skills`, `agents`, `instructions`, `prompts`, `targets`, `registries`)      |
| `positive-vibes install agents <name>`                  | Add an agent by name (registry-backed when available, else local path convention)                            |
| `positive-vibes list <resource-type>`                   | List available resources (`skills`, `agents`, `instructions`, `prompts`, `targets`, `registries`)            |
| `positive-vibes list agents`                            | List configured agents                                                                                       |
| `positive-vibes show <resource-type> <name>`            | Show detailed info for one resource                                                                          |
| `positive-vibes show agents <name>`                     | Show details for a configured agent                                                                          |
| `positive-vibes remove <resource-type> [name...]`       | Remove resources from your manifest (`skills`, `agents`, `instructions`, `prompts`, `targets`, `registries`) |
| `positive-vibes remove agents <name>`                   | Remove one or more agents from your manifest                                                                 |
| `positive-vibes apply`                                  | Sync resources to all configured target tool directories                                                     |
| `positive-vibes apply --force`                          | Overwrite existing installed resources                                                                       |
| `positive-vibes apply --link`                           | Use symlinks instead of copies                                                                               |
| `positive-vibes apply --refresh`                        | Pull latest from git registries before applying                                                              |
| `positive-vibes apply --global`                         | Apply only global config into current project targets                                                        |
| `positive-vibes config paths`                           | Show resolved config file locations                                                                          |
| `positive-vibes config show`                            | Show merged config                                                                                           |
| `positive-vibes config show --sources --relative-paths` | Show source-annotated paths relative to each config root                                                     |
| `positive-vibes config diff`                            | Show global-only, local-only, overrides, and effective summary                                               |
| `positive-vibes config diff --json`                     | Emit the same config diff as machine-readable JSON                                                           |
| `positive-vibes config validate`                        | Validate config and check for issues                                                                         |
| `positive-vibes config --color always validate`         | Control color output for config commands (`auto`, `always`, `never`)                                         |
| `positive-vibes completion install`                     | Install shell completion for your current shell                                                              |
| `positive-vibes completion uninstall`                   | Remove installed shell completion for your current shell                                                     |
| `positive-vibes generate <desc>`                        | Generate a custom skill from a description                                                                   |

## How Skills Work

A skill follows the [Agent Skills open standard](https://agentskills.io/specification). Each skill is a directory containing a `SKILL.md` with YAML frontmatter:

```markdown
---
name: conventional-commits
description: Enforces conventional commit format
version: "1.0"
tags:
  - git
  - standards
---

# Conventional Commits

Always use conventional commit format...
```

When you run `positive-vibes apply`, each configured skill is installed to the right place for each tool:

| Target          | Location                           |
| --------------- | ---------------------------------- |
| VS Code Copilot | `.github/skills/<name>/SKILL.md`   |
| OpenCode        | `.opencode/skills/<name>/SKILL.md` |
| Cursor          | `.cursor/skills/<name>/SKILL.md`   |

Instructions, agents, and prompts are also applied when configured, using each target's conventions (`.github/prompts/*.prompt.md` for Copilot, `.opencode/commands/*.md` for OpenCode; Cursor currently skips prompts with a warning).

## Bundled Skills

positive-vibes ships with a curated set of skills:

- **conventional-commits** -- Enforces conventional commit format
- **code-review** -- Thorough, constructive code review feedback

More coming soon. PRs welcome.

## Generating Custom Skills

```bash
positive-vibes generate "accessibility checker for JSX components"
```

This creates a starter `SKILL.md` you can customize. (Currently uses a template; LLM-powered generation coming soon.)

## Project Structure

```
cmd/positive-vibes/    Entry point
internal/
  cli/                 Cobra commands
  engine/              Business logic (scanner, applier, installer, generator)
  manifest/            vibes.yaml parsing and layered config
  registry/            Skill sources (embedded, git)
  target/              Tool adapters (Copilot, OpenCode, Cursor)
pkg/schema/            Skill struct and SKILL.md parser
skills/                Bundled skill templates
```

## Contributing

Contributions welcome:

1. Fork it
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (use conventional commits)
4. Push and open a PR

## License

[MIT License](https://opensource.org/license/mit)
