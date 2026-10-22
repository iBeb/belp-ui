package grid

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// SGR is a mouse event as the terminal would have sent it to a program in that
// cell, with the point moved into the cell's own coordinates.
//
// The grid asked the terminal for mouse events, so the programs inside it stopped
// receiving any: a click that should have put the cursor in a search field
// arrived here instead. Handing it on is what gives them back — the same bytes,
// about the same point, counted from the corner of their own screen.
func SGR(msg tea.MouseMsg, cell Rect) []byte {
	button, ok := code(msg)
	if !ok {
		return nil
	}
	// Terminals count from one, and from the corner of the screen the program
	// believes it has.
	x := msg.X - cell.X + 1
	y := msg.Y - cell.Y + 1
	if x < 1 || y < 1 || x > cell.W || y > cell.H {
		return nil
	}

	end := 'M' // pressed
	if msg.Action == tea.MouseActionRelease {
		end = 'm'
	}
	return []byte(fmt.Sprintf("\x1b[<%d;%d;%d%c", button, x, y, end))
}

// code is the number a terminal uses for this button and modifiers.
func code(msg tea.MouseMsg) (int, bool) {
	var n int
	switch msg.Button {
	case tea.MouseButtonLeft:
		n = 0
	case tea.MouseButtonMiddle:
		n = 1
	case tea.MouseButtonRight:
		n = 2
	case tea.MouseButtonWheelUp:
		n = 64
	case tea.MouseButtonWheelDown:
		n = 65
	case tea.MouseButtonWheelLeft:
		n = 66
	case tea.MouseButtonWheelRight:
		n = 67
	case tea.MouseButtonNone:
		if msg.Action != tea.MouseActionMotion {
			return 0, false
		}
		n = 3 // motion with nothing held
	default:
		return 0, false
	}
	if msg.Action == tea.MouseActionMotion {
		n += 32
	}
	if msg.Shift {
		n += 4
	}
	if msg.Alt {
		n += 8
	}
	if msg.Ctrl {
		n += 16
	}
	return n, true
}

// Wheel reports whether this is a scroll, which is the one mouse event that
// belongs to the cell under the pointer rather than to the one with the keys.
func Wheel(msg tea.MouseMsg) bool {
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown,
		tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		return true
	}
	return false
}
