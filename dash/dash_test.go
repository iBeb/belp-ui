package dash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iBeb/belp-ui/theme"
)

// A test has no terminal, so lipgloss would render every style as bare text and
// every assertion about colour would pass by saying nothing. These controls
// carry their meaning in the drawing, so the drawing has to happen.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m.Run()
}

func plain(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// A button is three lines whatever it says, so a row of them has one line its
// labels all sit on.
func TestAButtonIsAlwaysThreeLinesAndAsWideAsItSaid(t *testing.T) {
	s := theme.Default()
	for _, b := range []Button{
		{Label: "pause"},
		{Label: "▸ start"},
		{Label: "repair", Tone: Care},
		{Label: "x", Off: true},
		{Label: "wide one", Width: 30},
	} {
		for _, focused := range []bool{false, true} {
			lines := b.Render(s, focused)
			if len(lines) != 3 {
				t.Errorf("%q drew %d lines, want 3", b.Label, len(lines))
			}
			for _, l := range lines {
				if got := lipgloss.Width(l); got != b.Wide() {
					t.Errorf("%q line is %d wide, Wide() says %d: %q",
						b.Label, got, b.Wide(), plain(l))
				}
			}
			if !strings.Contains(plain(lines[1]), b.Label) {
				t.Errorf("%q does not carry its label: %q", b.Label, plain(lines[1]))
			}
		}
	}
}

// A row of buttons keeps every button's box intact, whichever one has the keys.
func TestARowOfButtonsLinesUp(t *testing.T) {
	s := theme.Default()
	row := []Button{{Label: "▸ start"}, {Label: "repair"}, {Label: Info}}
	for focus := -1; focus < len(row); focus++ {
		lines := Buttons(s, row, focus)
		if len(lines) != 3 {
			t.Fatalf("a row of buttons drew %d lines", len(lines))
		}
		want := lipgloss.Width(lines[0])
		for _, l := range lines {
			if got := lipgloss.Width(l); got != want {
				t.Errorf("focus %d: line is %d wide, the first was %d", focus, got, want)
			}
		}
	}
}

// A switch shows the alternatives rather than hiding them behind a press, and
// says which one is set even when the keys are somewhere else.
func TestASwitchAlwaysShowsWhichOptionIsSet(t *testing.T) {
	s := theme.Default()
	w := Switch{Options: []string{"RO", "RW"}, Active: 1}
	for _, focused := range []bool{false, true} {
		line := w.Render(s, focused)
		if got := lipgloss.Width(line); got != w.Wide() {
			t.Errorf("focused=%v: switch is %d wide, Wide() says %d", focused, got, w.Wide())
		}
		flat := plain(line)
		if !strings.Contains(flat, "RO") || !strings.Contains(flat, "RW") {
			t.Errorf("the switch hides an option: %q", flat)
		}
	}
	// Which option is set is carried by the drawing, not by the text: moving the
	// setting must change what is on screen without moving anything.
	for _, focused := range []bool{false, true} {
		ro := Switch{Options: []string{"RO", "RW"}, Active: 0}.Render(s, focused)
		rw := Switch{Options: []string{"RO", "RW"}, Active: 1}.Render(s, focused)
		if ro == rw {
			t.Errorf("focused=%v: the setting does not show", focused)
		}
		if plain(ro) != plain(rw) {
			t.Errorf("focused=%v: setting the switch moved the text: %q vs %q",
				focused, plain(ro), plain(rw))
		}
	}
}

// Facts go in as many columns as fit and stack when they do not, and the values
// line up down the block either way — which is the only reason for columns.
func TestPairsReflowAndStillLineUp(t *testing.T) {
	s := theme.Default()
	pairs := []Pair{
		{Key: "state", Value: "live"},
		{Key: "clients", Value: "2"},
		{Key: "port", Value: ":15442"},
		{Key: "latency", Value: "137ms"},
	}

	wide := Pairs(s, pairs, 60)
	if len(wide) != 2 {
		t.Errorf("four facts in 60 cells drew %d lines, want two of two", len(wide))
	}
	narrow := Pairs(s, pairs, 18)
	if len(narrow) != 4 {
		t.Errorf("four facts in 18 cells drew %d lines, want one each", len(narrow))
	}
	for _, width := range []int{12, 18, 24, 40, 60, 100} {
		for _, line := range Pairs(s, pairs, width) {
			if got := lipgloss.Width(line); got > width {
				t.Errorf("a line is %d wide at width %d: %q", got, width, plain(line))
			}
		}
	}
	// The values start at the same column in every row of a stacked block.
	at := -1
	for _, line := range narrow {
		i := strings.Index(plain(line), " ")
		flat := plain(line)
		for i < len(flat) && flat[i] == ' ' {
			i++
		}
		if at < 0 {
			at = i
		} else if i != at {
			t.Errorf("a value starts at %d where another started at %d", i, at)
		}
	}
}

// A bar answers "nearly all, or hardly any" at a glance, and never rounds
// something down to nothing.
func TestTheMeterFillsInProportionAndNeverRoundsSomethingToNothing(t *testing.T) {
	s := theme.Default()
	for _, c := range []struct{ done, total, width, full int }{
		{0, 10, 10, 0},
		{10, 10, 10, 10},
		{5, 10, 10, 5},
		{1, 12, 10, 1}, // would round to nothing
		{0, 0, 10, 0},
		{12, 10, 10, 10}, // more than all of it is still all of it
	} {
		bar := plain(Meter(s, c.done, c.total, c.width))
		if got := lipgloss.Width(bar); got != c.width {
			t.Errorf("%d/%d: bar is %d wide, want %d", c.done, c.total, got, c.width)
		}
		if got := strings.Count(bar, "█"); got != c.full {
			t.Errorf("%d/%d in %d: %d blocks filled, want %d (%q)",
				c.done, c.total, c.width, got, c.full, bar)
		}
	}
}

// A popup replaces what it covers: a terminal has no alpha, and a box you can
// read the dashboard through is a box nobody can read.
func TestAPopupCoversWhatIsUnderItAndSaysWhere(t *testing.T) {
	s := theme.Default()
	screen := make([]string, 20)
	for i := range screen {
		screen[i] = strings.Repeat("x", 60)
	}
	p := Popup{Title: "es · staging", Body: []string{"one", "two"},
		Buttons: []Button{{Label: "close"}}, Note: "esc to leave"}

	box := p.Render(s, p.Wide())
	out, spot := Over(screen, box, 60, 20)
	if spot.H != len(box) {
		t.Errorf("the popup says it is %d lines, it drew %d", spot.H, len(box))
	}
	if strings.Contains(plain(out[spot.Y+1]), "xxx") {
		t.Errorf("the dashboard shows through the popup: %q", plain(out[spot.Y+1]))
	}
	if !strings.Contains(plain(out[spot.Y]), "es · staging") {
		t.Errorf("the popup lost its title: %q", plain(out[spot.Y]))
	}
	if !spot.Holds(spot.X, spot.Y) || spot.Holds(spot.X-1, spot.Y) {
		t.Error("the popup does not know its own edges")
	}
}

// A click on a popup's button has to land on the button that was drawn there.
func TestAClickFindsThePopupButtonItWasDrawnOn(t *testing.T) {
	s := theme.Default()
	p := Popup{Title: "repair web", Body: []string{"gems changed"},
		Buttons: []Button{{Label: "rebuild"}, {Label: "migrate"}, {Label: "leave it"}},
		Note:    "esc to leave"}
	box := p.Render(s, p.Wide())
	screen := make([]string, 24)
	_, spot := Over(screen, box, 80, 24)

	// The middle of each button, on the line its label sits on.
	x := spot.X + 2
	for want, b := range p.Buttons {
		rows := len(box)
		y := spot.Y + rows - 2 - 2 - 3 + 1 // label line of the button row
		got, ok := spot.ButtonAt(p, s, x+b.Wide()/2, y)
		if !ok || got != want {
			t.Errorf("a click on %q found button %d (ok=%v), want %d", b.Label, got, ok, want)
		}
		x += b.Wide() + 1
	}
	// And a click on the body is not a click on a button.
	if _, ok := spot.ButtonAt(p, s, spot.X+3, spot.Y+2); ok {
		t.Error("a click on the popup's text pressed a button")
	}
}
