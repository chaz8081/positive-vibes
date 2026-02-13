package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	LeftRail   key.Binding
	RightRail  key.Binding
	CursorUp   key.Binding
	CursorDown key.Binding
	Install    key.Binding
	Remove     key.Binding
	Quit       key.Binding
	Help       key.Binding
	CloseHelp  key.Binding
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
		Install: key.NewBinding(
			key.WithKeys("i"),
		),
		Remove: key.NewBinding(
			key.WithKeys("r"),
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
