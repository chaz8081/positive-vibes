Here is the definitive "Vibe Check" Cheat Sheet based on the latest standards (as of early 2026).

It turns out the ecosystem is converging on a file-based standard for "Agents" (often just Markdown with YAML frontmatter), but each tool has a slightly different folder convention.

```csv
Tool,Scope,Instructions (System Prompt),Agent Skills (Tools/Capabilities),Custom Agents (Personas/Sub-agents)
GitHub Copilot,Project,.github/copilot-instructions.md,.github/skills/**/SKILL.md,.github/agents/*.agent.md
,Global,(VSCode Settings),~/.copilot/skills/**/SKILL.md,(Defined via Extensions)
OpenCode,Project,AGENTS.md (or .opencode/AGENTS.md),.opencode/skills/**/SKILL.md,.opencode/agents/*.md
,Global,~/.config/opencode/AGENTS.md,~/.config/opencode/skills/**,~/.config/opencode/agents/*.md
Cursor,Project,.cursor/rules/*.mdc,(Integrated into Rules or MCP),.cursor/agents/*.md (Sub-agents)
,Global,(Settings UI),~/.cursor/rules/,~/.cursor/agents/*.md
Claude (Desktop),Project,CLAUDE.md,.claude/skills/**/SKILL.md,claude_desktop_config.json (MCP)
,Global,(Account Settings),~/.claude/skills/**,~/Library/.../Claude/config.json
```

The Workflow Support Table

Here is how each tool currently handles "doing a complex task" (e.g., "Plan, Code, Test, and Refactor"):

```csv
Feature,GitHub Copilot,OpenCode,Cursor,Claude (Desktop/Code)
Workflow Concept,Prompt Files & Workspace,Agent Modes,Composer (Composer Stories),Chain of Thought
File Standard,.github/prompts/*.prompt.md,opencode.json (Agent Config),.cursor/rules/*.mdc,CLAUDE.md (Project Rules)
Orchestration,"Task-Based: You define a ""Prompt"" that acts as a mini-agent with specific tools attached.","Role-Based: You switch between a ""Plan"" agent (read-only) and a ""Build"" agent (write-access).","Context-Based: You write a ""Story"" or ""Spec"" in Markdown, and Composer iteratively implements it.","Prompt-Based: You rely on a large context window and manual steering via ""Projects""."
Execution,"User triggers via / command (e.g., /refactor).",User switches modes (Tab) or invokes sub-agents (@general).,User hits Cmd+I (Composer) and references the spec file.,"User types request; Claude plans via ""Chain of Thought""."
Best For...,"Repeatable Tasks (e.g., ""Generate Unit Tests"", ""Review PR"").","Autonomous Loops (e.g., ""Fix this bug and don't stop until tests pass"").","Feature Development (e.g., ""Build this entire page based on this image"").","Deep Reasoning (e.g., ""Architectural analysis of the whole repo"")."
```
