package grid

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// What a program reads is bytes, so what the grid forwards has to be the bytes
// the terminal would have sent — not a description of them.
func TestAKeyBecomesWhatTheTerminalWouldHaveSent(t *testing.T) {
	for _, c := range []struct {
		name string
		key  tea.KeyMsg
		want string
	}{
		{"a letter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")}, "w"},
		{"several", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("web")}, "web"},
		{"a space", tea.KeyMsg{Type: tea.KeySpace}, " "},
		{"enter is a carriage return", tea.KeyMsg{Type: tea.KeyEnter}, "\r"},
		{"backspace is delete", tea.KeyMsg{Type: tea.KeyBackspace}, "\x7f"},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, "\x1b[A"},
		{"left", tea.KeyMsg{Type: tea.KeyLeft}, "\x1b[D"},
		{"page up", tea.KeyMsg{Type: tea.KeyPgUp}, "\x1b[5~"},
		{"shift tab", tea.KeyMsg{Type: tea.KeyShiftTab}, "\x1b[Z"},
		{"^C", tea.KeyMsg{Type: tea.KeyCtrlC}, "\x03"},
		{"^R", tea.KeyMsg{Type: tea.KeyCtrlR}, "\x12"},
		{"escape", tea.KeyMsg{Type: tea.KeyEsc}, "\x1b"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := string(Bytes(c.key)); got != c.want {
				t.Errorf("Bytes(%v) = %q, want %q", c.key, got, c.want)
			}
		})
	}
}

// Meta is an escape and then the key, which is how terminals have sent it for
// decades and what portulan's own ⌥↵ is written to read.
func TestAltIsAnEscapeAndThenTheKey(t *testing.T) {
	if got := string(Bytes(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})); got != "\x1b\r" {
		t.Errorf("⌥↵ = %q, want an escape then a carriage return", got)
	}
	if got := string(Bytes(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f"), Alt: true})); got != "\x1bf" {
		t.Errorf("⌥f = %q", got)
	}
}

// A key with no bytes is forwarded as nothing rather than as an empty write
// that a program would read as end of input.
func TestAKeyWithNoBytesSendsNothing(t *testing.T) {
	if got := Bytes(tea.KeyMsg{Type: tea.KeyType(-999)}); got != nil {
		t.Errorf("an unknown key sent %q", got)
	}
}

// The keys that move about in a line of text carry their modifiers inside the
// sequence, not in front of it. Written the other way, ⌥← reached the program
// as an escape and then a plain arrow — two keys, and word-by-word movement
// gone from every session hosted here.
func TestMovingKeysCarryTheirModifiers(t *testing.T) {
	for _, c := range []struct {
		say  string
		key  tea.KeyMsg
		want string
	}{
		{"left", tea.KeyMsg{Type: tea.KeyLeft}, "\x1b[D"},
		{"alt+left", tea.KeyMsg{Type: tea.KeyLeft, Alt: true}, "\x1b[1;3D"},
		{"alt+right", tea.KeyMsg{Type: tea.KeyRight, Alt: true}, "\x1b[1;3C"},
		{"shift+left", tea.KeyMsg{Type: tea.KeyShiftLeft}, "\x1b[1;2D"},
		{"ctrl+left", tea.KeyMsg{Type: tea.KeyCtrlLeft}, "\x1b[1;5D"},
		{"ctrl+shift+left", tea.KeyMsg{Type: tea.KeyCtrlShiftLeft}, "\x1b[1;6D"},
		{"ctrl+alt+left", tea.KeyMsg{Type: tea.KeyCtrlLeft, Alt: true}, "\x1b[1;7D"},
		{"home", tea.KeyMsg{Type: tea.KeyHome}, "\x1b[H"},
		{"ctrl+home", tea.KeyMsg{Type: tea.KeyCtrlHome}, "\x1b[1;5H"},
		{"shift+end", tea.KeyMsg{Type: tea.KeyShiftEnd}, "\x1b[1;2F"},
		{"page up", tea.KeyMsg{Type: tea.KeyPgUp}, "\x1b[5~"},
		{"alt+page up", tea.KeyMsg{Type: tea.KeyPgUp, Alt: true}, "\x1b[5;3~"},
		{"delete", tea.KeyMsg{Type: tea.KeyDelete}, "\x1b[3~"},
	} {
		if got := string(Bytes(c.key)); got != c.want {
			t.Errorf("%s arrives as %q, want %q", c.say, got, c.want)
		}
	}

	// A letter with alt is still an escape and then the letter, which is what
	// meta has meant since teletypes.
	if got := string(Bytes(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true})); got != "\x1bb" {
		t.Errorf("alt+b arrives as %q", got)
	}
}
