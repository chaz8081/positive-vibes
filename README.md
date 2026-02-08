# positive-vibes ✨

> Harmonize your AI tooling. One manifest to rule them all.

positive-vibes is an environment-agnostic configuration manager for AI tooling. It checks the vibes of your project and aligns your AI tools — VS Code Copilot, OpenCode, Cursor, and more — from a single source of truth. 🌈✨

Why? Every AI coding tool has its own way of configuring "skills" or "instructions." You end up maintaining the same context in `.github/skills/`, `.opencode/skills/`, `.cursor/skills/`... separately. That's bad vibes. 😅

positive-vibes gives you one `vibes.yaml` to define your skills and instructions, then syncs them everywhere. Good vibes only. 😎

## Quick Start

### Install

```bash
go install github.com/chaz8081/positive-vibes/cmd/positive-vibes@latest
```

### Initialize

```bash
vibes init
```

This scans your project, detects the language (Go, Node, Python), and creates a starter `vibes.yaml` with recommended skills.

### Add Skills

```bash
vibes install conventional-commits
```

### Apply

```bash
vibes apply
```

This reads your `vibes.yaml` and installs skills into all your target tools' directories. Vibe check passed! ✅

## The Manifest (`vibes.yaml`)

```yaml
registries:
  - name: awesome-copilot
    url: https://github.com/github/awesome-copilot

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

## Commands

| Command | Description |
|---------|-------------|
| `vibes init` | Scan project and create `vibes.yaml` |
| `vibes install <skill>` | Add a skill to your manifest |
| `vibes apply` | Sync skills to all target tool directories |
| `vibes apply --force` | Overwrite existing skills |
| `vibes apply --link` | Use symlinks instead of copies |
| `vibes generate <desc>` | Generate a custom skill from a description |

## How Skills Work

A skill follows the [Agent Skills open standard](https://github.com/github/copilot-skills). Each skill is a directory containing a `SKILL.md` with YAML frontmatter:

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

When you run `vibes apply`, each skill gets installed to the right place for each tool:

| Target | Location |
|--------|----------|
| VS Code Copilot | `.github/skills/<name>/SKILL.md` |
| OpenCode | `.opencode/skills/<name>/SKILL.md` |
| Cursor | `.cursor/skills/<name>/SKILL.md` |

## Bundled Skills

positive-vibes ships with a curated set of skills:

- **conventional-commits** — Enforces conventional commit format
- **code-review** — Thorough, constructive code review feedback

More coming soon. PRs welcome! 🎉

## Generating Custom Skills

```bash
vibes generate "accessibility checker for JSX components"
```

This creates a starter `SKILL.md` you can customize. (Currently uses a template; LLM-powered generation coming soon.) 🤖✨

## Project Structure

```
cmd/positive-vibes/    Entry point
internal/
  cli/                 Cobra commands
  engine/              Business logic (scanner, applier, installer, generator)
  manifest/            vibes.yaml parsing
  registry/            Skill sources (embedded, git)
  target/              Tool adapters (Copilot, OpenCode, Cursor)
pkg/schema/            Skill struct and SKILL.md parser
skills/                Bundled skill templates
```

## Contributing

Contributions welcome! This project runs on good vibes:

1. Fork it
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (use conventional commits, naturally 😄)
4. Push and open a PR

## License

MIT
