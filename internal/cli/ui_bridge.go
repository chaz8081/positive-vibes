package cli

import (
	"fmt"

	"github.com/chaz8081/positive-vibes/internal/cli/ui"
)

func init() {
	if err := ui.ConfigureResourceServiceBridge(ui.ResourceServiceBridge{
		ListAvailableRows: func(projectDir, globalPath, kind string) ([]ui.ResourceRow, error) {
			items, err := ListAvailableResourceItems(projectDir, globalPath, kind)
			if err != nil {
				return nil, err
			}
			return toUIRows(items), nil
		},
		ListInstalledRows: func(projectDir, globalPath, kind string) ([]ui.ResourceRow, error) {
			items, err := ListInstalledResourceItems(projectDir, globalPath, kind)
			if err != nil {
				return nil, err
			}
			return toUIRows(items), nil
		},
		ShowResource: func(projectDir, globalPath, kind, name string) (ui.ResourceDetail, error) {
			detail, err := ShowResourceDetail(projectDir, globalPath, kind, name)
			if err != nil {
				return ui.ResourceDetail{}, err
			}
			return ui.ResourceDetail{
				Kind:        string(detail.Kind),
				Name:        detail.Name,
				Installed:   detail.Installed,
				Registry:    detail.Registry,
				RegistryURL: detail.RegistryURL,
				Path:        detail.Path,
				Payload:     detail.Payload,
			}, nil
		},
		MergeRows: func(available, installed []ui.ResourceRow) []ui.ResourceRow {
			merged := MergeResourceItems(fromUIRows(available), fromUIRows(installed))
			return toUIRows(merged)
		},
		InstallResources: func(projectDir, globalPath, kind string, names []string) error {
			return InstallResourceItems(projectDir, globalPath, kind, names)
		},
		RemoveResources: func(projectDir, kind string, names []string) error {
			return RemoveResourceItems(projectDir, kind, names)
		},
	}); err != nil {
		panic(fmt.Sprintf("configure resource service bridge: %v", err))
	}
}

func toUIRows(items []ResourceItem) []ui.ResourceRow {
	rows := make([]ui.ResourceRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ui.ResourceRow{Name: item.Name, Installed: item.Installed})
	}
	return rows
}

func fromUIRows(rows []ui.ResourceRow) []ResourceItem {
	items := make([]ResourceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ResourceItem{Name: row.Name, Installed: row.Installed})
	}
	return items
}
