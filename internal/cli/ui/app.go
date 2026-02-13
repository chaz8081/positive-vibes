package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

var newResourceService = NewService

func newRuntimeModel(projectDir string) model {
	m := newModel()

	svc, err := newResourceService(projectDir)
	if err != nil {
		m.statusMessage = fmt.Sprintf("resource service unavailable: %v", err)
		return m
	}

	m.listResources = svc.ListResources
	m.showResource = svc.ShowResource
	m.installResources = svc.InstallResources
	m.removeResources = svc.RemoveResources

	if !m.refreshRowsForActiveRail() {
		m.statusMessage = fmt.Sprintf("resource service unavailable: %v", m.statusMessage)
	}

	return m
}

func Run(projectDir string) error {
	p := tea.NewProgram(newRuntimeModel(projectDir))
	_, err := p.Run()
	return err
}
