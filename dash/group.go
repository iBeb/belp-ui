package dash

import (
	"strings"

	"github.com/iBeb/belp-ui/theme"
)

// Fit is how a group uses the width it is given.
type Fit int

const (
	// Start is every button as wide as its label, packed against the left. The
	// zero value, because it is the one fit that is never wrong: a group that
	// was never told how to sit still looks like a row of buttons.
	Start Fit = iota
	// Centre puts the natural row in the middle of the width.
	Centre
	// End puts it against the right edge, for the row that ends a form.
	End
	// Fill stretches the row to the whole width, handing the slack out in
	// proportion to what each button asked for, so the main action stays the
	// widest thing on the row.
	Fill
)

// Group is a row of buttons laid out as one thing.
//
// The point of it is that an app says what it wants — these buttons, centred,
// or filling the card — and never works out a width. Wrapping, stretching and
// where each button ended up for a click all come from here, computed once, so
// the row that is drawn and the row that is clicked cannot disagree.
type Group struct {
	Buttons []Button
	Fit     Fit
	// Gap is the cells between two buttons. Zero means one, which is the least
	// that keeps two coloured bands from reading as a single wider one.
	Gap int
}

// Span is where one button was drawn, in the group's own coordinates.
type Span struct {
	X, Y, Width int
}

// Layout is the whole group's geometry: one span per button, in the order they
// were given, and how many lines they took.
type Layout struct {
	Spans []Span
	High  int

	rows  [][]Button // the buttons at their drawn widths
	lefts []int      // where each row starts
}

func (g Group) gap() int {
	if g.Gap > 0 {
		return g.Gap
	}
	return 1
}

// Wide is the width the group would like: every button at its natural size
// with a gap between each.
func (g Group) Wide() int {
	if len(g.Buttons) == 0 {
		return 0
	}
	w := g.gap() * (len(g.Buttons) - 1)
	for _, b := range g.Buttons {
		w += b.Wide()
	}
	return w
}

// High is how many lines the group takes at this width, which every card in a
// row has to agree on before any of them is drawn.
func (g Group) High(width int) int { return g.Layout(width).High }

// Layout works out where every button goes.
//
// Too narrow, the row wraps rather than shrinking or running off the edge: a
// button narrower than its label says nothing, and a row that overruns takes
// the card's border with it.
func (g Group) Layout(width int) Layout {
	var out Layout
	if len(g.Buttons) == 0 {
		return out
	}
	gap := g.gap()
	out.Spans = make([]Span, len(g.Buttons))

	i := 0
	for i < len(g.Buttons) {
		// As many as fit, and always at least one.
		n, wide := 0, 0
		for i+n < len(g.Buttons) {
			w := g.Buttons[i+n].Wide()
			if n > 0 {
				w += gap
			}
			if n > 0 && wide+w > width {
				break
			}
			wide += w
			n++
		}

		row := make([]Button, n)
		copy(row, g.Buttons[i:i+n])
		if g.Fit == Fill {
			row = Stretch(row, width, gap)
			wide = gap * (n - 1)
			for _, b := range row {
				wide += b.Wide()
			}
		}

		left := 0
		switch g.Fit {
		case Centre:
			left = max(0, (width-wide)/2)
		case End:
			left = max(0, width-wide)
		}

		x := left
		y := len(out.rows)
		for k, b := range row {
			out.Spans[i+k] = Span{X: x, Y: y, Width: b.Wide()}
			x += b.Wide() + gap
		}
		out.rows = append(out.rows, row)
		out.lefts = append(out.lefts, left)
		i += n
	}
	out.High = len(out.rows)
	return out
}

// Render draws the group. focus is which of its buttons has the keys, counted
// from the first one, or any number outside the group for none.
func (g Group) Render(s theme.Styles, focus, width int) []string {
	l := g.Layout(width)
	out := make([]string, l.High)
	at := 0
	for y, row := range l.rows {
		parts := make([]string, len(row))
		for k, b := range row {
			parts[k] = b.Render(s, at+k == focus)
		}
		out[y] = strings.Repeat(" ", l.lefts[y]) +
			strings.Join(parts, strings.Repeat(" ", g.gap()))
		at += len(row)
	}
	return out
}

// At is the button drawn at this point, or -1. The coordinates are the group's
// own: x from its left edge, y from its first line.
func (l Layout) At(x, y int) int {
	for i, s := range l.Spans {
		if y == s.Y && x >= s.X && x < s.X+s.Width {
			return i
		}
	}
	return -1
}
