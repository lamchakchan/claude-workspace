package tui

import (
	tea "charm.land/bubbletea/v2"
)

const (
	keyCtrlC    = "ctrl+c"
	keyEnter    = "enter"
	keyEsc      = "esc"
	keyTab      = "tab"
	keyShiftTab = "shift+tab"
	keyDown     = "down"
	keyUp       = "up"
)

// IsQuit returns true if the key message is a quit key (q or ctrl+c).
func IsQuit(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "q", keyCtrlC:
		return true
	}
	return false
}

// IsBack returns true if the key message is a back key (esc).
func IsBack(msg tea.KeyPressMsg) bool {
	return msg.String() == keyEsc
}
