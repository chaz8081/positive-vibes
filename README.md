# positive-vibes

> Harmonize your AI tooling. One manifest to rule them all.

positive-vibes is an environment-agnostic configuration manager for AI tooling. It aligns your AI tools -- VS Code Copilot, OpenCode, Cursor, and more -- from a single source of truth.

Every AI coding tool has its own way of configuring "skills" or "instructions." You end up maintaining the same context in `.github/skills/`, `.opencode/skills/`, `.cursor/skills/`... separately.

positive-vibes gives you one `vibes.yaml` to define your skills and instructions, then syncs them everywhere.

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
positive-vibes install conventional-commits
```

### Apply

```bash
positive-vibes apply
```

This reads your manifest and installs skills into all your target tools' directories.

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
  - "Always use TypeScript for frontend code"
  - "Prefer functional components"

targets:
  - vscode-copilot
  - opencode
  - cursor
```

> **Note:** `vibes.yml` is still supported for backwards compatibility. If both `vibes.yaml` and `vibes.yml` exist, `vibes.yaml` takes precedence.

## Layered Configuration

positive-vibes supports a global + project layered config:

| Level       | Location                             | Purpose                                                        |
| ----------- | ------------------------------------ | -------------------------------------------------------------- |
| **Global**  | `~/.config/positive-vibes/vibes.yaml` | User-level defaults (personal registries, shared instructions) |
| **Project** | `./vibes.yaml`                        | Project-specific skills and targets                            |

### Merge behavior

When both exist, they are merged:

- **Registries**: combined by name; project overrides global for same name
- **Skills**: combined by name; project overrides global for same name
- **Targets**: project targets override global entirely
- **Instructions**: concatenated (global first, then project)

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

### Migrating existing manifests

If you have a `vibes.yaml` without `ref` on a registry, validation will fail with a helpful message:

```
registry "awesome-copilot" must specify a ref (use "latest" to track the default branch)
```

Add `ref: latest` to preserve the previous behavior.

## Commands

| Command                 | Description                                     |
| ----------------------- | ----------------------------------------------- |
| `positive-vibes init`            | Scan project and create `vibes.yaml`             |
| `positive-vibes install <skill>` | Add a skill to your manifest                     |
| `positive-vibes apply`           | Sync skills to all target tool directories       |
| `positive-vibes apply --force`   | Overwrite existing skills                        |
| `positive-vibes apply --link`    | Use symlinks instead of copies                   |
| `positive-vibes apply --refresh` | Pull latest from git registries before applying  |
| `positive-vibes config show`     | Show merged config with source annotations       |
| `positive-vibes config validate` | Validate manifest and check for issues           |
| `positive-vibes generate <desc>` | Generate a custom skill from a description       |

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

When you run `positive-vibes apply`, each skill gets installed to the right place for each tool:

| Target          | Location                           |
| --------------- | ---------------------------------- |
| VS Code Copilot | `.github/skills/<name>/SKILL.md`   |
| OpenCode        | `.opencode/skills/<name>/SKILL.md` |
| Cursor          | `.cursor/skills/<name>/SKILL.md`   |

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

MIT
