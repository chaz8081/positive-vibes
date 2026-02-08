package schema

import (
	"bytes"
	"errors"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Skill represents an Agent Skill per the open standard.
type Skill struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Version      string   `yaml:"version,omitempty"`
	Author       string   `yaml:"author,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
	Globs        []string `yaml:"globs,omitempty"`
	Instructions string   `yaml:"-"` // markdown body, not in frontmatter
}

// ParseSkillFile parses a SKILL.md file content into a Skill struct.
// It splits YAML frontmatter (between --- delimiters) from the markdown body.
func ParseSkillFile(content []byte) (*Skill, error) {
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, errors.New("empty content")
	}

	s := &Skill{}
	text := string(content)

	// Look for frontmatter delimiters at the start
	if strings.HasPrefix(text, "---\n") {
		// find second delimiter
		parts := strings.SplitN(text, "---\n", 3)
		// parts[0] is "" before first, parts[1] is yaml, parts[2] is rest
		if len(parts) >= 3 {
			yamlPart := parts[1]
			body := parts[2]
			if err := yaml.Unmarshal([]byte(yamlPart), s); err != nil {
				return nil, err
			}
			s.Instructions = strings.TrimSpace(body)
			return s, nil
		}
	}

	// No frontmatter: whole content is instructions
	s.Instructions = strings.TrimSpace(text)
	return s, nil
}

// RenderSkillFile renders a Skill struct back to SKILL.md format.
func RenderSkillFile(skill *Skill) ([]byte, error) {
	// Marshal frontmatter
	fm := map[string]interface{}{
		"name":        skill.Name,
		"description": skill.Description,
	}
	if skill.Version != "" {
		fm["version"] = skill.Version
	}
	if skill.Author != "" {
		fm["author"] = skill.Author
	}
	if len(skill.Tags) > 0 {
		fm["tags"] = skill.Tags
	}
	if len(skill.Globs) > 0 {
		fm["globs"] = skill.Globs
	}

	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(string(yamlBytes))
	b.WriteString("---\n\n")
	if skill.Instructions != "" {
		b.WriteString(strings.TrimSpace(skill.Instructions))
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}
