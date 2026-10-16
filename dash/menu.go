package dash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/iBeb/belp-ui/theme"
)

// Menu is a short list of choices that drops from the thing it changes: a
// ticket's status, opened from the status word and drawn under it.
//
// Under its control rather than in the middle of the screen the way a Popup is.
// A popup is a question about the whole window; a menu is one word being
// changed, and the place it opens is what says which word that is.
type Menu struct {
	Options []Option
	// Current is the option that is already so, or -1 where none of them is.
	// Marked rather than left out: a list that skips the present state reads as
	// a list of somewhere else, and the board's order is lost with it.
	Current int
	// Focus is the option the keys are on.
	Focus int
}

// Option is one choice.
type Option struct {
	Label string
	// Colour is the option's own, "#1090e0", where the thing it names has one —
	// a status as its board paints it. Empty draws it in the ordinary colour
	// rather than in one made up for it.
	Colour string
}

// Step moves the focus by n, round the end and back to the start.
func (m Menu) Step(n int) Menu {
	if len(m.Options) == 0 {
		return m
	}
	m.Focus = ((m.Focus+n)%len(m.Options) + len(m.Options)) % len(m.Options)
	return m
}

// Wide is how much room the menu wants: the widest option, the mark before it,
// a space either side and the border.
func (m Menu) Wide() int {
	n := 0
	for _, o := range m.Options {
		n = max(n, lipgloss.Width(o.Label))
	}
	return n + menuChrome
}

// menuChrome is what a row takes besides its label: the border, a space, the
// mark, a space, and a space before the border again.
const menuChrome = 6

// Render draws the menu at a width, one option to a line between a top and a
// bottom edge.
func (m Menu) Render(s theme.Styles, width int) []string {
	inner := max(width, menuChrome+1) - 2
	edge := s.Selected
	out := []string{edge.Render("╭" + strings.Repeat("─", inner) + "╮")}
	for i, o := range m.Options {
		mark := " "
		if i == m.Current {
			mark = theme.Bullet
		}
		label := cutTo(o.Label, inner-4)
		row := ""
		if i == m.Focus {
			// One band over the whole row, mark and all: the row is the thing
			// being chosen, and a band over the word alone is a selection inside
			// it.
			row = s.Selection.Render(padTo(" "+mark+" "+label, inner))
		} else {
			style := s.Item
			if o.Colour != "" {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color(o.Colour))
			}
			row = " " + s.Desc.Render(mark) + " " + padTo(style.Render(label), inner-3)
		}
		out = append(out, edge.Render("│")+row+edge.Render("│"))
	}
	return append(out, edge.Render("╰"+strings.Repeat("─", inner)+"╯"))
}

// Drop draws a box over a screen with its top-left corner at x, y, and says
// where it landed.
//
// Moved left where it would run off the right-hand edge, because a menu opened
// from the end of a line — which is where a status sits — has nowhere else to
// go. The cells either side of it are kept: unlike a popup, a menu covers a
// patch of the screen, and the rest of each line is still what it was.
func Drop(lines []string, box []string, x, y, width int) ([]string, Spot) {
	if len(box) == 0 {
		return lines, Spot{}
	}
	w := 0
	for _, line := range box {
		w = max(w, lipgloss.Width(line))
	}
	x = max(0, min(x, width-w))

	out := make([]string, len(lines))
	copy(out, lines)
	for i, line := range box {
		at := y + i
		if at < 0 || at >= len(out) {
			continue
		}
		left := ansi.Truncate(out[at], x, "")
		left += strings.Repeat(" ", x-lipgloss.Width(left))
		out[at] = left + line + ansi.TruncateLeft(out[at], x+w, "")
	}
	return out, Spot{X: x, Y: y, W: w, H: len(box)}
}

// OptionAt is which option a click landed on: the rows between the two edges,
// across the whole width of the box.
func (s Spot) OptionAt(m Menu, x, y int) (int, bool) {
	if !s.Holds(x, y) {
		return 0, false
	}
	i := y - s.Y - 1
	if i < 0 || i >= len(m.Options) {
		return 0, false
	}
	return i, true
}
