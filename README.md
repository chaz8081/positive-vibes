# positive-vibes

[![Go](https://github.com/chaz8081/positive-vibes/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/chaz8081/positive-vibes/actions/workflows/go.yml)

> Harmonize your AI tooling. One manifest to rule them all.

positive-vibes is an environment-agnostic configuration manager for AI tooling. It aligns your AI tools -- VS Code Copilot, OpenCode, Cursor, and more -- from a single source of truth.

Every AI coding tool has its own way of configuring resources like skills and instructions. You end up maintaining the same context in `.github/skills/`, `.opencode/skills/`, `.cursor/skills/`... separately.

positive-vibes gives you one `vibes.yaml` to define your resources, then syncs them everywhere.

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

To add an instruction entry by name (creates a path-based instruction by convention):

```bash
positive-vibes install instructions coding-standards
```

To add an agent entry by name (creates a path-based agent by convention):

```bash
positive-vibes install agents code-reviewer
```

### Apply

```bash
positive-vibes apply
```

This reads your manifest and installs configured resources (skills, instructions, agents) into your target tools' directories.

## The Manifest (`vibes.yaml`)

```yaml
registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
    paths:
      skills: skills/

skills:
  - name: conventional-commits
  - name: code-review
  - name: my-custom-skill
    path: ./local-skills/my-custom-skill

instructions:
  - name: frontend-typescript
    content: "Always use TypeScript for frontend code"
  - name: frontend-components
    content: "Prefer functional components"
  - name: team-guide
    path: ./instructions/team-guide.md

agents:
  - name: code-reviewer
    path: ./agents/reviewer.md
  - name: registry-reviewer
    registry: awesome-copilot/my-skill:agents/reviewer.md

targets:
  - vscode-copilot
  - opencode
  - cursor
```

Instruction entries are object-based: each item must include `name` and exactly one of `content` or `path`.

Agent entries are object-based: each item must include `name` and exactly one of `path` or `registry`.

## Layered Configuration

positive-vibes supports a global + project layered config:

| Level       | Location                             | Purpose                                                        |
| ----------- | ------------------------------------ | -------------------------------------------------------------- |
| **Global**  | `~/.config/positive-vibes/vibes.yaml` | User-level defaults (personal registries, shared resources) |
| **Project** | `./vibes.yaml`                        | Project-specific resources and targets                         |

### Merge behavior

When both exist, they are merged:

- **Registries**: combined by name; project overrides global for same name
- **Skills**: combined by name; project overrides global for same name
- **Instructions**: combined by name; project overrides global for same name
- **Agents**: combined by name; project overrides global for same name
- **Targets**: project targets override global entirely
- **Paths**: relative `path` entries are resolved from the manifest they came from
- **Warnings**: `config validate` warns on risky overrides that change source type (e.g., `content` -> `path`, or registry -> path)

The global config path respects `$XDG_CONFIG_HOME` if set.

## Registry Versioning

Every registry entry requires a `ref` field that controls which version of the registry is used. This makes your setup reproducible and explicit.

### Ref types

| Ref value | Behavior |
| --------- | -------- |
| `latest`  | Track the registry's default branch. `positive-vibes apply --refresh` pulls new changes. |
| Branch name (e.g. `main`, `develop`) | Pin to a specific branch. Refresh is a no-op. |
| Tag name (e.g. `v1.2.0`) | Pin to a tagged release. Refresh is a no-op. |
| Commit SHA (7-40 hex chars) | Pin to an exact commit. Refresh is a no-op. |

### Examples

```yaml
registries:
  # Track the latest skills (auto-updates on refresh)
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot
    ref: latest
    paths:
      skills: skills/

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

| Command | Description |
| ------- | ----------- |
| `positive-vibes init` | Scan project and create `vibes.yaml` |
| `positive-vibes install <resource-type> [name...]` | Add skills, agents, or instructions to your manifest |
| `positive-vibes install agents <name>` | Add a path-based agent entry (`./agents/<name>.md`) |
| `positive-vibes list <resource-type>` | List available resources (`skills`, `agents`, `instructions`) |
| `positive-vibes list agents` | List configured agents |
| `positive-vibes show <resource-type> <name>` | Show detailed info for one resource |
| `positive-vibes show agents <name>` | Show details for a configured agent |
| `positive-vibes remove <resource-type> [name...]` | Remove resources from your manifest |
| `positive-vibes remove agents <name>` | Remove one or more agents from your manifest |
| `positive-vibes apply` | Sync resources to all configured target tool directories |
| `positive-vibes apply --force` | Overwrite existing installed resources |
| `positive-vibes apply --link` | Use symlinks instead of copies |
| `positive-vibes apply --refresh` | Pull latest from git registries before applying |
| `positive-vibes apply --global` | Apply only global config into current project targets |
| `positive-vibes config paths` | Show resolved config file locations |
| `positive-vibes config show` | Show merged config |
| `positive-vibes config show --sources --relative-paths` | Show source-annotated paths relative to each config root |
| `positive-vibes config diff` | Show global-only, local-only, overrides, and effective summary |
| `positive-vibes config diff --json` | Emit the same config diff as machine-readable JSON |
| `positive-vibes config validate` | Validate config and check for issues |
| `positive-vibes config --color always validate` | Control color output for config commands (`auto`, `always`, `never`) |
| `positive-vibes completion install` | Install shell completion for your current shell |
| `positive-vibes completion uninstall` | Remove installed shell completion for your current shell |
| `positive-vibes generate <desc>` | Generate a custom skill from a description |

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

Instructions and agents are also applied when configured, using each target's instruction/agent conventions.

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
