package registry

import (
	"fmt"
	"github.com/chaz8081/positive-vibes/pkg/schema"
)

// GitRegistry is a stub for remote git-backed registries.
type GitRegistry struct {
	RegistryName string
	URL          string
	CachePath    string // e.g., ~/.positive-vibes/cache/<name>/
}

func (r *GitRegistry) Name() string { return r.RegistryName }

func (r *GitRegistry) Fetch(name string) (*schema.Skill, string, error) {
	return nil, "", fmt.Errorf("remote git registries are coming soon - stay tuned! (%s)", r.URL)
}

func (r *GitRegistry) List() ([]string, error) {
	return nil, fmt.Errorf("remote git registries are coming soon - stay tuned! (%s)", r.URL)
}
