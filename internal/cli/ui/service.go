package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	resourceKindSkills       = "skills"
	resourceKindAgents       = "agents"
	resourceKindInstructions = "instructions"
)

var ErrResourceServiceBridgeNotConfigured = errors.New("resource service bridge is not configured")

type ResourceDetail struct {
	Kind        string
	Name        string
	Installed   bool
	Registry    string
	RegistryURL string
	Path        string
	Payload     any
}

type ResourceRow struct {
	Name      string
	Installed bool
}

type ResourceServiceBridge struct {
	ListAvailableRows func(projectDir, globalPath, kind string) ([]ResourceRow, error)
	ListInstalledRows func(projectDir, globalPath, kind string) ([]ResourceRow, error)
	ShowResource      func(projectDir, globalPath, kind, name string) (ResourceDetail, error)
	MergeRows         func(available, installed []ResourceRow) []ResourceRow
	InstallResources  func(projectDir, globalPath, kind string, names []string) error
	RemoveResources   func(projectDir, kind string, names []string) error
}

var (
	resourceServiceBridgeMu         sync.RWMutex
	configuredResourceServiceBridge ResourceServiceBridge
	resourceServiceBridgeConfigured bool
)

func ConfigureResourceServiceBridge(bridge ResourceServiceBridge) error {
	if err := validateResourceServiceBridge(bridge); err != nil {
		return err
	}

	resourceServiceBridgeMu.Lock()
	configuredResourceServiceBridge = bridge
	resourceServiceBridgeConfigured = true
	resourceServiceBridgeMu.Unlock()

	return nil
}

func currentResourceServiceBridge() (ResourceServiceBridge, error) {
	resourceServiceBridgeMu.RLock()
	bridge := configuredResourceServiceBridge
	configured := resourceServiceBridgeConfigured
	resourceServiceBridgeMu.RUnlock()

	if !configured {
		return ResourceServiceBridge{}, ErrResourceServiceBridgeNotConfigured
	}
	return bridge, nil
}

func validateResourceServiceBridge(bridge ResourceServiceBridge) error {
	if bridge.ListAvailableRows == nil {
		return fmt.Errorf("resource service bridge ListAvailableRows is required")
	}
	if bridge.ListInstalledRows == nil {
		return fmt.Errorf("resource service bridge ListInstalledRows is required")
	}
	if bridge.ShowResource == nil {
		return fmt.Errorf("resource service bridge ShowResource is required")
	}
	if bridge.MergeRows == nil {
		return fmt.Errorf("resource service bridge MergeRows is required")
	}
	if bridge.InstallResources == nil {
		return fmt.Errorf("resource service bridge InstallResources is required")
	}
	if bridge.RemoveResources == nil {
		return fmt.Errorf("resource service bridge RemoveResources is required")
	}
	return nil
}

func resetResourceServiceBridgeForTesting() {
	resourceServiceBridgeMu.Lock()
	configuredResourceServiceBridge = ResourceServiceBridge{}
	resourceServiceBridgeConfigured = false
	resourceServiceBridgeMu.Unlock()
}

type Service struct {
	deps serviceDeps
}

type serviceDeps struct {
	listAvailable func(kind string) ([]ResourceRow, error)
	listInstalled func(kind string) ([]ResourceRow, error)
	showDetail    func(kind, name string) (ResourceDetail, error)
	mergeRows     func(available, installed []ResourceRow) []ResourceRow
	install       func(kind string, names []string) error
	remove        func(kind string, names []string) error
}

func NewService(projectDir string) (*Service, error) {
	bridge, err := currentResourceServiceBridge()
	if err != nil {
		return nil, err
	}
	return NewServiceWithBridge(projectDir, bridge)
}

func NewServiceWithBridge(projectDir string, bridge ResourceServiceBridge) (*Service, error) {
	if err := validateResourceServiceBridge(bridge); err != nil {
		return nil, err
	}
	if projectDir == "" {
		projectDir = "."
	}
	globalPath := defaultGlobalManifestPath()

	return newServiceWithDeps(serviceDeps{
		listAvailable: func(kind string) ([]ResourceRow, error) {
			return bridge.ListAvailableRows(projectDir, globalPath, kind)
		},
		listInstalled: func(kind string) ([]ResourceRow, error) {
			return bridge.ListInstalledRows(projectDir, globalPath, kind)
		},
		showDetail: func(kind, name string) (ResourceDetail, error) {
			return bridge.ShowResource(projectDir, globalPath, kind, name)
		},
		mergeRows: bridge.MergeRows,
		install: func(kind string, names []string) error {
			return bridge.InstallResources(projectDir, globalPath, kind, names)
		},
		remove: func(kind string, names []string) error {
			return bridge.RemoveResources(projectDir, kind, names)
		},
	}), nil
}

func newServiceWithDeps(deps serviceDeps) *Service {
	return &Service{deps: deps}
}

func (s *Service) ListResources(kind string) ([]ResourceRow, error) {
	if err := validateKind(kind); err != nil {
		return nil, err
	}

	available, err := s.deps.listAvailable(kind)
	if err != nil {
		return nil, err
	}
	installed, err := s.deps.listInstalled(kind)
	if err != nil {
		return nil, err
	}

	rows := s.deps.mergeRows(available, installed)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	return rows, nil
}

func (s *Service) ShowResource(kind, name string) (ResourceDetail, error) {
	if err := validateKind(kind); err != nil {
		return ResourceDetail{}, err
	}
	if name == "" {
		return ResourceDetail{}, fmt.Errorf("resource name is required")
	}
	return s.deps.showDetail(kind, name)
}

func (s *Service) InstallResources(kind string, names []string) error {
	if err := validateKind(kind); err != nil {
		return err
	}
	return s.deps.install(kind, dedup(names))
}

func (s *Service) RemoveResources(kind string, names []string) error {
	if err := validateKind(kind); err != nil {
		return err
	}
	return s.deps.remove(kind, dedup(names))
}

func validateKind(kind string) error {
	switch kind {
	case resourceKindSkills, resourceKindAgents, resourceKindInstructions:
		return nil
	default:
		return fmt.Errorf("unknown resource type %q (valid: skills, agents, instructions)", kind)
	}
}

func defaultGlobalManifestPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "positive-vibes", "vibes.yaml")
}

func dedup(names []string) []string {
	seen := make(map[string]bool, len(names))
	result := make([]string, 0, len(names))
	for _, n := range names {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		result = append(result, n)
	}
	return result
}
