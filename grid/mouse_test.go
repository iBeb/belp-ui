package grid

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The point a program is told about is the point in its own screen, counted
// from its own corner and from one — not where it happened to land on yours.
func TestTheProgramIsToldWhereTheClickWasInItsOwnScreen(t *testing.T) {
	cell := Rect{X: 80, Y: 25, W: 40, H: 10}

	got := string(SGR(tea.MouseMsg{X: 80, Y: 25, Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress}, cell))
	if want := "\x1b[<0;1;1M"; got != want {
		t.Errorf("the top-left corner became %q, want %q", got, want)
	}

	got = string(SGR(tea.MouseMsg{X: 95, Y: 30, Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress}, cell))
	if want := "\x1b[<0;16;6M"; got != want {
		t.Errorf("a click inside became %q, want %q", got, want)
	}
}

// A press and a release are different events, and a program that only ever sees
// presses never learns that a drag has ended.
func TestAReleaseIsNotAPress(t *testing.T) {
	cell := Rect{X: 0, Y: 0, W: 20, H: 5}
	press := string(SGR(tea.MouseMsg{X: 3, Y: 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}, cell))
	release := string(SGR(tea.MouseMsg{X: 3, Y: 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}, cell))

	if press[len(press)-1] != 'M' || release[len(release)-1] != 'm' {
		t.Errorf("press ended %q and release ended %q", press[len(press)-1:], release[len(release)-1:])
	}
}

// The wheel is the event the apps use for scrolling, so it has to survive the
// journey with the number terminals give it.
func TestTheWheelKeepsItsNumbers(t *testing.T) {
	cell := Rect{X: 0, Y: 0, W: 20, H: 5}
	for _, c := range []struct {
		button tea.MouseButton
		want   string
	}{
		{tea.MouseButtonWheelUp, "\x1b[<64;3;2M"},
		{tea.MouseButtonWheelDown, "\x1b[<65;3;2M"},
	} {
		got := string(SGR(tea.MouseMsg{X: 2, Y: 1, Button: c.button, Action: tea.MouseActionPress}, cell))
		if got != c.want {
			t.Errorf("%v became %q, want %q", c.button, got, c.want)
		}
		if !Wheel(tea.MouseMsg{Button: c.button}) {
			t.Errorf("%v is not recognised as a scroll", c.button)
		}
	}
	if Wheel(tea.MouseMsg{Button: tea.MouseButtonLeft}) {
		t.Error("a left click is recognised as a scroll")
	}
}

// Modifiers travel too: shift-click means something different from a click, and
// a program that never hears the difference cannot act on it.
func TestModifiersTravel(t *testing.T) {
	cell := Rect{X: 0, Y: 0, W: 20, H: 5}
	plain := string(SGR(tea.MouseMsg{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}, cell))
	shifted := string(SGR(tea.MouseMsg{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, Shift: true}, cell))
	if plain == shifted {
		t.Error("shift made no difference to what the program is told")
	}
}

// A point outside the cell is nothing to do with the program in it.
func TestAPointOutsideTheCellIsNotSentIn(t *testing.T) {
	cell := Rect{X: 10, Y: 10, W: 5, H: 5}
	for _, p := range [][2]int{{9, 12}, {15, 12}, {12, 9}, {12, 15}} {
		if got := SGR(tea.MouseMsg{X: p[0], Y: p[1], Button: tea.MouseButtonLeft,
			Action: tea.MouseActionPress}, cell); got != nil {
			t.Errorf("(%d,%d) was sent in as %q", p[0], p[1], got)
		}
	}
}
