package dash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

func statuses() Menu {
	return Menu{Current: 1, Focus: 2, Options: []Option{
		{Label: "backlog", Colour: "#87909e"},
		{Label: "in progress", Colour: "#1090e0"},
		{Label: "in review"},
		{Label: "Closed", Colour: "#008844"},
	}}
}

// Every line of the box is the width it said it would be, or the edges of it
// stop lining up and a click is tested against a box that is not the one drawn.
func TestAMenuIsAsWideAsItSaid(t *testing.T) {
	m := statuses()
	lines := m.Render(theme.Default(), m.Wide())
	if len(lines) != len(m.Options)+2 {
		t.Fatalf("%d lines for %d options", len(lines), len(m.Options))
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got != m.Wide() {
			t.Errorf("%q is %d wide, Wide() says %d", plain(line), got, m.Wide())
		}
	}
}

// The present state is in the list, in its place, and says so.
func TestAMenuMarksWhatIsAlreadySo(t *testing.T) {
	m := statuses()
	lines := m.Render(theme.Default(), m.Wide())
	for i, o := range m.Options {
		line := plain(lines[i+1])
		if !strings.Contains(line, o.Label) {
			t.Errorf("row %d does not carry %q: %q", i, o.Label, line)
		}
		if marked := strings.Contains(line, theme.Bullet); marked != (i == m.Current) {
			t.Errorf("row %d marked=%v, current is %d: %q", i, marked, m.Current, line)
		}
	}
}

// An option is drawn in its own colour, and the one the keys are on is drawn
// as the selection instead.
func TestAMenuDrawsTheColourItWasGiven(t *testing.T) {
	s := theme.Default()
	m := statuses()
	lines := m.Render(s, m.Wide())
	blue := lipgloss.NewStyle().Foreground(lipgloss.Color("#1090e0")).Render("in progress")
	if !strings.Contains(lines[2], blue) {
		t.Errorf("in progress is not drawn in its board's colour: %q", lines[2])
	}
	m.Focus = 1
	lines = m.Render(s, m.Wide())
	if strings.Contains(lines[2], blue) {
		t.Errorf("the focused row kept its own colour rather than the selection: %q", lines[2])
	}
}

func TestStepGoesRoundTheEnds(t *testing.T) {
	m := statuses()
	if got := m.Step(2).Focus; got != 0 {
		t.Errorf("two on from the third of four is %d, want 0", got)
	}
	if got := m.Step(-3).Focus; got != 3 {
		t.Errorf("three back from the third of four is %d, want 3", got)
	}
	if got := (Menu{}).Step(1).Focus; got != 0 {
		t.Errorf("an empty menu moved its focus to %d", got)
	}
}

// Dropped from the end of a line, the box is pulled in to fit, and what the line
// said either side of it is still there.
func TestDropKeepsTheLineAroundIt(t *testing.T) {
	screen := []string{"belp › Development · in review", "──────────────", "session text that runs long enough"}
	box := []string{"+--+", "|ab|"}
	got, at := Drop(screen, box, 30, 1, 34)
	if at != (Spot{X: 30, Y: 1, W: 4, H: 2}) {
		t.Fatalf("landed at %+v", at)
	}
	if got[0] != screen[0] {
		t.Errorf("the line above was touched: %q", got[0])
	}
	if want := "session text that runs long en|ab|"; got[2] != want {
		t.Errorf("line under is %q, want %q", got[2], want)
	}

	_, at = Drop(screen, box, 32, 1, 34)
	if at.X != 30 {
		t.Errorf("a box running off the edge was left at %d, want 30", at.X)
	}
	got, _ = Drop(screen, box, 2, 0, 34)
	if !strings.HasPrefix(got[0], "be+--+") || !strings.HasSuffix(got[0], "in review") {
		t.Errorf("the line it covers lost its ends: %q", got[0])
	}
}

// A click on a row is the option drawn on that row, and the edges are nothing.
func TestOptionAtIsTheRowDrawnThere(t *testing.T) {
	m := statuses()
	screen := make([]string, 10)
	got, at := Drop(screen, m.Render(theme.Default(), m.Wide()), 5, 1, 80)
	for i, o := range m.Options {
		y := at.Y + 1 + i
		if !strings.Contains(plain(got[y]), o.Label) {
			t.Fatalf("row %d does not show %q: %q", y, o.Label, plain(got[y]))
		}
		if n, ok := at.OptionAt(m, at.X+3, y); !ok || n != i {
			t.Errorf("a click on %q came back as %d, %v", o.Label, n, ok)
		}
	}
	for _, y := range []int{at.Y, at.Y + at.H - 1} {
		if _, ok := at.OptionAt(m, at.X+3, y); ok {
			t.Errorf("a click on the edge at row %d chose something", y)
		}
	}
	if _, ok := at.OptionAt(m, at.X+at.W, at.Y+1); ok {
		t.Error("a click beside the box chose something")
	}
}
