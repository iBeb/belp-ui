package grid

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Bytes is a key as the terminal would have sent it.
//
// The grid is given keys already parsed, and the program in a cell expects the
// bytes that parsing was done on — so they have to be written back. Lossy in
// principle and exact in practice for everything these apps read: text, the
// control characters, and the escape sequences for the keys that have no
// character of their own.
func Bytes(k tea.KeyMsg) []byte {
	// The keys that move about in a line of text carry their modifiers inside
	// the sequence rather than in front of it: ⌥← is CSI 1;3D, and a reader
	// given an escape and then a plain arrow takes it for two keys — which is
	// how word-by-word movement went missing in a session hosted here.
	if b := moving(k); b != nil {
		return b
	}
	if k.Alt {
		// Meta is an escape and then the key, which is how terminals have sent
		// it for decades and what every reader still expects.
		plain := k
		plain.Alt = false
		if b := Bytes(plain); len(b) > 0 {
			return append([]byte{0x1b}, b...)
		}
		return nil
	}

	switch k.Type {
	case tea.KeyRunes:
		return []byte(string(k.Runes))
	case tea.KeySpace:
		return []byte{' '}
	case tea.KeyEnter:
		// Carriage return, not newline: Enter is what the key sends, and a
		// newline is what a program writes.
		return []byte{'\r'}
	case tea.KeyTab:
		return []byte{'\t'}
	case tea.KeyShiftTab:
		return []byte("\x1b[Z")
	case tea.KeyBackspace:
		return []byte{0x7f}
	case tea.KeyEsc:
		return []byte{0x1b}
	}

	// The control keys are the letter with the top bits cleared, which is what
	// ^C has meant since teletypes and what every one of them still reads.
	if b, ok := ctrl[k.Type]; ok {
		return []byte{b}
	}
	return nil
}

// moving is one of the keys that move about, with its modifiers written the
// way a terminal writes them, or nil for anything else.
//
// Every one of these has a plain form and a modified form, and the modified
// form is the plain one with a number in it: 2 is shift, 3 is alt, 5 is ctrl,
// and they add — 7 is ctrl and alt together. Bubbletea gives some of them their
// own names (ctrl+left) and carries alt in a flag, so both are read back here.
func moving(k tea.KeyMsg) []byte {
	at, ok := about[k.Type]
	if !ok {
		return nil
	}
	mod := 1 + at.mod
	if k.Alt {
		mod += 2
	}
	if mod == 1 {
		if at.tilde {
			return fmt.Appendf(nil, "\x1b[%s~", at.say)
		}
		return fmt.Appendf(nil, "\x1b[%s", at.say)
	}
	if at.tilde {
		return fmt.Appendf(nil, "\x1b[%s;%d~", at.say, mod)
	}
	return fmt.Appendf(nil, "\x1b[1;%d%s", mod, at.say)
}

// The keys that move about, what a terminal calls each one, and which
// modifiers bubbletea has already folded into the name.
type moves struct {
	say string
	// tilde is the other shape a sequence comes in: CSI 5~ rather than CSI A.
	tilde bool
	mod   int
}

const (
	shift = 1
	meta  = 2
	ctrl0 = 4
)

var about = map[tea.KeyType]moves{
	tea.KeyUp:    {say: "A"},
	tea.KeyDown:  {say: "B"},
	tea.KeyRight: {say: "C"},
	tea.KeyLeft:  {say: "D"},
	tea.KeyHome:  {say: "H"},
	tea.KeyEnd:   {say: "F"},

	tea.KeyShiftUp:    {say: "A", mod: shift},
	tea.KeyShiftDown:  {say: "B", mod: shift},
	tea.KeyShiftRight: {say: "C", mod: shift},
	tea.KeyShiftLeft:  {say: "D", mod: shift},
	tea.KeyShiftHome:  {say: "H", mod: shift},
	tea.KeyShiftEnd:   {say: "F", mod: shift},

	tea.KeyCtrlUp:    {say: "A", mod: ctrl0},
	tea.KeyCtrlDown:  {say: "B", mod: ctrl0},
	tea.KeyCtrlRight: {say: "C", mod: ctrl0},
	tea.KeyCtrlLeft:  {say: "D", mod: ctrl0},
	tea.KeyCtrlHome:  {say: "H", mod: ctrl0},
	tea.KeyCtrlEnd:   {say: "F", mod: ctrl0},

	tea.KeyCtrlShiftUp:    {say: "A", mod: ctrl0 + shift},
	tea.KeyCtrlShiftDown:  {say: "B", mod: ctrl0 + shift},
	tea.KeyCtrlShiftRight: {say: "C", mod: ctrl0 + shift},
	tea.KeyCtrlShiftLeft:  {say: "D", mod: ctrl0 + shift},
	tea.KeyCtrlShiftHome:  {say: "H", mod: ctrl0 + shift},
	tea.KeyCtrlShiftEnd:   {say: "F", mod: ctrl0 + shift},

	tea.KeyPgUp:   {say: "5", tilde: true},
	tea.KeyPgDown: {say: "6", tilde: true},
	tea.KeyDelete: {say: "3", tilde: true},
	tea.KeyInsert: {say: "2", tilde: true},
}

// ctrl is ^A to ^Z, less the four that already have names above: ^I is Tab,
// ^M is Enter, ^H is Backspace and ^[ is Escape.
var ctrl = map[tea.KeyType]byte{
	tea.KeyCtrlA: 0x01, tea.KeyCtrlB: 0x02, tea.KeyCtrlC: 0x03, tea.KeyCtrlD: 0x04,
	tea.KeyCtrlE: 0x05, tea.KeyCtrlF: 0x06, tea.KeyCtrlG: 0x07,
	tea.KeyCtrlJ: 0x0a, tea.KeyCtrlK: 0x0b, tea.KeyCtrlL: 0x0c,
	tea.KeyCtrlN: 0x0e, tea.KeyCtrlO: 0x0f, tea.KeyCtrlP: 0x10,
	tea.KeyCtrlQ: 0x11, tea.KeyCtrlR: 0x12, tea.KeyCtrlS: 0x13, tea.KeyCtrlT: 0x14,
	tea.KeyCtrlU: 0x15, tea.KeyCtrlV: 0x16, tea.KeyCtrlW: 0x17, tea.KeyCtrlX: 0x18,
	tea.KeyCtrlY: 0x19, tea.KeyCtrlZ: 0x1a,
}
