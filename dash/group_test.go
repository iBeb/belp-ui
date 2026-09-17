package dash

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

func sample() Group {
	return Group{Buttons: []Button{
		{Label: "▶ start"}, {Label: "restart", Tone: Warning},
		{Label: "logs"}, {Label: Mark, Tone: Info},
	}}
}

// Each fit puts the row where it said it would.
func TestAGroupSitsWhereItsFitSaid(t *testing.T) {
	s := theme.Default()
	const width = 60
	g := sample()
	natural := g.Wide()

	for _, tc := range []struct {
		name     string
		fit      Fit
		wantLeft int
	}{
		{"Start", Start, 0},
		{"Centre", Centre, (width - natural) / 2},
		{"End", End, width - natural},
	} {
		g.Fit = tc.fit
		l := g.Layout(width)
		if got := l.Spans[0].X; got != tc.wantLeft {
			t.Errorf("%s starts at %d, want %d", tc.name, got, tc.wantLeft)
		}
		// And the drawing agrees with the geometry: the first label sits
		// centred inside the span the layout gave it. Measured against the
		// label rather than the leading spaces, because a button's own padding
		// is spaces too once the colour is stripped.
		drawn := []rune(plain(g.Render(s, -1, width)[0]))
		sp := l.Spans[0]
		want := sp.X + (sp.Width-lipgloss.Width(g.Buttons[0].Label))/2
		if got := runeIndex(drawn, g.Buttons[0].Label); got != want {
			t.Errorf("%s draws its first label at %d, the layout put it at %d: %q",
				tc.name, got, want, string(drawn))
		}
	}

	g.Fit = Fill
	l := g.Layout(width)
	last := l.Spans[len(l.Spans)-1]
	if got := last.X + last.Width; got != width {
		t.Errorf("Fill ends at %d, want the full %d", got, width)
	}
	if got := lipgloss.Width(g.Render(s, -1, width)[0]); got != width {
		t.Errorf("a filled row draws %d wide, want %d", got, width)
	}
}

// Too narrow, a group wraps rather than overflowing or shrinking a button
// below its label.
func TestAGroupWrapsWhenItCannotFit(t *testing.T) {
	s := theme.Default()
	g := sample()
	for _, width := range []int{g.Wide(), 20, 12} {
		l := g.Layout(width)
		if l.High != len(g.Render(s, -1, width)) {
			t.Errorf("at %d, High says %d lines and Render drew %d",
				width, l.High, len(g.Render(s, -1, width)))
		}
		for i, sp := range l.Spans {
			if sp.X+sp.Width > width && sp.Width <= width {
				t.Errorf("at %d, button %d runs to %d", width, i, sp.X+sp.Width)
			}
			if sp.Width < lipgloss.Width(g.Buttons[i].Label) {
				t.Errorf("at %d, button %d is %d wide, narrower than its label",
					width, i, sp.Width)
			}
		}
	}
	if g.Layout(g.Wide()).High != 1 {
		t.Error("a group given exactly its width still wrapped")
	}
	if g.Layout(12).High < 2 {
		t.Error("a group given half its width did not wrap")
	}
}

// A click lands on the button that was drawn under it. Checked against the
// drawn line rather than against the layout, so the two cannot agree with each
// other and be wrong together.
func TestAClickFindsTheButtonDrawnThere(t *testing.T) {
	s := theme.Default()
	for _, fit := range []Fit{Start, Centre, End, Fill} {
		g := sample()
		g.Fit = fit
		const width = 50
		lines := g.Render(s, -1, width)
		l := g.Layout(width)

		for i, b := range g.Buttons {
			drawn := []rune(plain(lines[l.Spans[i].Y]))
			col := runeIndex(drawn, b.Label)
			if col < 0 {
				t.Fatalf("fit %d: %q is not on its line: %q", fit, b.Label, string(drawn))
			}
			x := col + lipgloss.Width(b.Label)/2
			if got := l.At(x, l.Spans[i].Y); got != i {
				t.Errorf("fit %d: a click on %q at column %d found %d, want %d",
					fit, b.Label, col, got, i)
			}
		}
		// The edges, exactly. An off-by-one here is a click landing on the
		// button next to the one it was aimed at, which is the whole reason
		// the layout is computed once rather than twice.
		for i, sp := range l.Spans {
			if got := l.At(sp.X, sp.Y); got != i {
				t.Errorf("fit %d: the first cell of button %d found %d", fit, i, got)
			}
			if got := l.At(sp.X+sp.Width-1, sp.Y); got != i {
				t.Errorf("fit %d: the last cell of button %d found %d", fit, i, got)
			}
			if got := l.At(sp.X+sp.Width, sp.Y); got == i {
				t.Errorf("fit %d: the cell past button %d still found it", fit, i)
			}
			if sp.X > 0 {
				if got := l.At(sp.X-1, sp.Y); got == i {
					t.Errorf("fit %d: the cell before button %d already found it", fit, i)
				}
			}
		}
		if got := l.At(-1, 0); got != -1 {
			t.Errorf("fit %d: a click left of the row found button %d", fit, got)
		}
		if got := l.At(width+5, 0); got != -1 {
			t.Errorf("fit %d: a click past the row found button %d", fit, got)
		}
	}
}

// An empty group draws nothing and takes no lines, rather than one blank one.
func TestAnEmptyGroupTakesNoRoom(t *testing.T) {
	var g Group
	if g.High(40) != 0 || len(g.Render(theme.Default(), -1, 40)) != 0 {
		t.Error("an empty group took a line")
	}
	if g.Wide() != 0 {
		t.Errorf("an empty group wants %d cells", g.Wide())
	}
}
