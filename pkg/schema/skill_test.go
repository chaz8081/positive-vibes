package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSkillFile_ValidFrontmatter(t *testing.T) {
	content := `---
name: conventional-commits
description: Enforces conventional commit message format
version: "1.0"
author: positive-vibes
tags:
  - git
  - commits
globs:
  - "*.go"
  - "*.ts"
---

# Conventional Commits

Always use the conventional commits format for commit messages...
`

	s, err := ParseSkillFile([]byte(content))
	assert.NoError(t, err)
	assert.Equal(t, "conventional-commits", s.Name)
	assert.Equal(t, "Enforces conventional commit message format", s.Description)
	assert.Equal(t, "1.0", s.Version)
	assert.Equal(t, "positive-vibes", s.Author)
	assert.Equal(t, []string{"git", "commits"}, s.Tags)
	assert.Equal(t, []string{"*.go", "*.ts"}, s.Globs)
	assert.Contains(t, s.Instructions, "# Conventional Commits")
}

func TestParseSkillFile_NoFrontmatter(t *testing.T) {
	content := `# Hello

This is a skill without frontmatter.`

	s, err := ParseSkillFile([]byte(content))
	assert.NoError(t, err)
	assert.Equal(t, "", s.Name)
	assert.Equal(t, "", s.Description)
	assert.Equal(t, "# Hello\n\nThis is a skill without frontmatter.", s.Instructions)
}

func TestParseSkillFile_EmptyFile(t *testing.T) {
	_, err := ParseSkillFile([]byte(""))
	assert.Error(t, err)
}

func TestRenderSkillFile(t *testing.T) {
	s := &Skill{
		Name:        "conventional-commits",
		Description: "Enforces conventional commit message format",
		Version:     "1.0",
		Author:      "positive-vibes",
		Tags:        []string{"git", "commits"},
		Globs:       []string{"*.go", "*.ts"},
		Instructions: `# Conventional Commits

Always use the conventional commits format for commit messages...`,
	}

	out, err := RenderSkillFile(s)
	assert.NoError(t, err)
	outStr := string(out)
	assert.Contains(t, outStr, "---")
	assert.Contains(t, outStr, "name: conventional-commits")
	assert.Contains(t, outStr, "description: Enforces conventional commit message format")
	assert.Contains(t, outStr, "# Conventional Commits")
}

func TestRenderThenParse_Roundtrip(t *testing.T) {
	original := &Skill{
		Name:        "example-skill",
		Description: "An example skill",
		Version:     "0.1",
		Author:      "tester",
		Tags:        []string{"example", "test"},
		Globs:       []string{"*.md"},
		Instructions: `# Example

Do something useful.`,
	}

	out, err := RenderSkillFile(original)
	assert.NoError(t, err)

	parsed, err := ParseSkillFile(out)
	assert.NoError(t, err)

	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.Author, parsed.Author)
	assert.Equal(t, original.Tags, parsed.Tags)
	assert.Equal(t, original.Globs, parsed.Globs)
	assert.Equal(t, original.Instructions, parsed.Instructions)
}
