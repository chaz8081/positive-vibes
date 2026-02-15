package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	LeftRail        key.Binding
	RightRail       key.Binding
	CursorUp        key.Binding
	CursorDown      key.Binding
	PreviewUp       key.Binding
	PreviewDown     key.Binding
	PreviewHalfUp   key.Binding
	PreviewHalfDown key.Binding
	PreviewTop      key.Binding
	PreviewBottom   key.Binding
	ToggleFocus     key.Binding
	ResetTargets    key.Binding
	Search          key.Binding
	Back            key.Binding
	Quit            key.Binding
	Help            key.Binding
	CloseHelp       key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		LeftRail: key.NewBinding(
			key.WithKeys("left", "h"),
		),
		RightRail: key.NewBinding(
			key.WithKeys("right", "l"),
		),
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		PreviewUp: key.NewBinding(
			key.WithKeys("k", "K"),
		),
		PreviewDown: key.NewBinding(
			key.WithKeys("j", "J"),
		),
		PreviewHalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
		),
		PreviewHalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
		),
		PreviewTop: key.NewBinding(
			key.WithKeys("g"),
		),
		PreviewBottom: key.NewBinding(
			key.WithKeys("G"),
		),
		ToggleFocus: key.NewBinding(
			key.WithKeys("tab"),
		),
		ResetTargets: key.NewBinding(
			key.WithKeys("r"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
		),
		Back: key.NewBinding(
			key.WithKeys("b"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
		),
		CloseHelp: key.NewBinding(
			key.WithKeys("esc"),
		),
	}
}
