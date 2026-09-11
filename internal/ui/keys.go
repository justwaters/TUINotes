package ui

import "charm.land/bubbles/v2/key"

var (
	keyTab      = key.NewBinding(key.WithKeys("tab"))
	keyShiftTab = key.NewBinding(key.WithKeys("shift+tab"))
	keyEnter    = key.NewBinding(key.WithKeys("enter"))
	keyNew      = key.NewBinding(key.WithKeys("n"))
	keyEdit     = key.NewBinding(key.WithKeys("e"))
	keySave     = key.NewBinding(key.WithKeys("ctrl+s"))
	keyCancel   = key.NewBinding(key.WithKeys("esc"))
	keyDelete   = key.NewBinding(key.WithKeys("d"))
	keySearch   = key.NewBinding(key.WithKeys("S"))
	keyRefresh  = key.NewBinding(key.WithKeys("r"))
	keyQuit     = key.NewBinding(key.WithKeys("ctrl+c", "q"))
	keyConfirm  = key.NewBinding(key.WithKeys("y"))
)
