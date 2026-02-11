package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeOverrideDiagnostics(t *testing.T) {
	global := &Manifest{
		Registries: []RegistryRef{{Name: "shared-reg"}, {Name: "global-reg"}},
		Skills:     []SkillRef{{Name: "shared-skill"}, {Name: "global-skill"}},
		Instructions: []InstructionRef{
			{Name: "shared-inst", Content: "g"},
			{Name: "global-inst", Content: "g"},
		},
		Agents: []AgentRef{{Name: "shared-agent", Registry: "g"}, {Name: "global-agent", Registry: "g"}},
	}
	local := &Manifest{
		Registries:   []RegistryRef{{Name: "shared-reg"}},
		Skills:       []SkillRef{{Name: "shared-skill"}},
		Instructions: []InstructionRef{{Name: "shared-inst", Content: "l"}},
		Agents:       []AgentRef{{Name: "shared-agent", Path: "./a.md"}},
	}

	d := ComputeOverrideDiagnostics(global, local)

	assert.Equal(t, []string{"shared-reg"}, d.Registries)
	assert.Equal(t, []string{"shared-skill"}, d.Skills)
	assert.Equal(t, []string{"shared-inst"}, d.Instructions)
	assert.Equal(t, []string{"shared-agent"}, d.Agents)
}
