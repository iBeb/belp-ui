package dash

import (
	"fmt"
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
		lines := Buttons(s, row, focus, 0)
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

// A radio shows every option with the chosen one filled in, and says which is
// set even when the keys are somewhere else.
func TestARadioAlwaysShowsWhichOptionIsSet(t *testing.T) {
	s := theme.Default()
	r := Radio{Options: []string{"read only", "read · write"}, Active: 1}
	for _, focused := range []bool{false, true} {
		line := r.Render(s, focused)
		if got := lipgloss.Width(line); got != r.Wide() {
			t.Errorf("focused=%v: radio is %d wide, Wide() says %d", focused, got, r.Wide())
		}
		flat := plain(line)
		for _, opt := range r.Options {
			if !strings.Contains(flat, opt) {
				t.Errorf("the radio hides %q: %q", opt, flat)
			}
		}
		if strings.Count(flat, radioOn) != 1 {
			t.Errorf("the radio fills %d marks, want exactly one: %q",
				strings.Count(flat, radioOn), flat)
		}
	}
	// Which option is set is carried by the mark, not by the order.
	first := plain(Radio{Options: r.Options, Active: 0}.Render(s, false))
	if !strings.HasPrefix(first, radioOn) {
		t.Errorf("setting the first option did not fill its mark: %q", first)
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
	// The values start at the same column in every row of a stacked block, and
	// the keys end there — which is the whole point of aligning them right.
	keyW := KeyWidth(pairs)
	for i, line := range narrow {
		want := rightTo(pairs[i].Key, keyW) + " " + pairs[i].Value
		if got := strings.TrimRight(plain(line), " "); got != want {
			t.Errorf("row %d drew %q, want %q", i, got, want)
		}
	}
}

// A bar answers "nearly all, or hardly any" at a glance, and never rounds
// something down to nothing.
func TestTheGaugeFillsInProportionAndNeverRoundsSomethingToNothing(t *testing.T) {
	s := theme.Default()
	for _, c := range []struct{ done, total, width, full int }{
		{0, 10, 10, 0},
		{10, 10, 10, 10},
		{5, 10, 10, 5},
		{1, 12, 10, 1}, // would round to nothing
		{0, 0, 10, 0},
		{12, 10, 10, 10}, // more than all of it is still all of it
	} {
		g := Gauge{Done: c.done, Total: c.total}
		bar := g.Render(s, c.width)
		if got := lipgloss.Width(bar); got != c.width {
			t.Errorf("%d/%d: bar is %d wide, want %d", c.done, c.total, got, c.width)
		}
		if got := filledCells(g, s, c.width); got != c.full {
			t.Errorf("%d/%d in %d: %d cells filled, want %d",
				c.done, c.total, c.width, got, c.full)
		}
	}
}

// The figure goes inside the bar, and stays there whole however far the fill has
// reached across it — the boundary runs through the text, not around it.
func TestTheGaugeCarriesItsFigureInside(t *testing.T) {
	s := theme.Default()
	for _, done := range []int{0, 1, 5, 9, 10} {
		g := Gauge{Done: done, Total: 10, Label: "10/10"}
		if got := plain(g.Render(s, 14)); !strings.Contains(got, "10/10") {
			t.Errorf("%d/10: the figure is not in the bar: %q", done, got)
		}
		if got := lipgloss.Width(g.Render(s, 14)); got != 14 {
			t.Errorf("%d/10: bar is %d wide, want 14", done, got)
		}
	}
	// A bar too narrow for its figure says as much of it as it can rather than
	// overflowing the card.
	if got := lipgloss.Width(Gauge{Done: 1, Total: 2, Label: "100/100"}.Render(s, 4)); got != 4 {
		t.Errorf("a narrow bar is %d wide, want 4", got)
	}
}

// Both halves of the bar are painted: a track drawn as nothing is a bar whose
// length you cannot see, which is half of what a bar says.
func TestTheGaugePaintsItsTrackAsWellAsItsFill(t *testing.T) {
	s := theme.Default()
	empty := Gauge{Done: 0, Total: 10}.Render(s, 8)
	if !strings.Contains(empty, background(s.Palette.Faint.Dark)) {
		t.Errorf("an empty bar paints no track: %q", empty)
	}
	// An unreadable count takes the colour across the whole bar, so it does not
	// look like a count of nothing.
	grave := Gauge{Total: 0, Label: "unknown", Tone: Grave}.Render(s, 10)
	if strings.Contains(grave, background(s.Palette.Faint.Dark)) {
		t.Errorf("an unreadable bar draws an ordinary track: %q", grave)
	}
	if !strings.Contains(grave, background(s.Palette.Danger.Dark)) {
		t.Errorf("an unreadable bar is not drawn in the colour that says so: %q", grave)
	}
}

// filledCells is how many cells the fill covers, counted from where the style
// changes rather than from the arithmetic under test.
func filledCells(g Gauge, s theme.Styles, width int) int {
	filled, _ := g.colours(s)
	head := colourOf(filled)
	bar := g.Render(s, width)
	if !strings.HasPrefix(bar, head) {
		return 0
	}
	rest := bar[len(head):]
	end := strings.Index(rest, "\x1b")
	if end < 0 {
		end = len(rest)
	}
	return lipgloss.Width(rest[:end])
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

// Given a width, a row of buttons fills it exactly: a row that stops short of
// the card's edge reads as something half drawn.
func TestButtonsStretchToFillTheirRow(t *testing.T) {
	s := theme.Default()
	for _, row := range [][]Button{
		{{Label: "▮▮ pause"}, {Label: "⚒ repair"}, {Label: Info}},
		{{Label: "▮▮ stop"}, {Label: Info}},
		{{Label: "only one"}},
	} {
		natural := len(row) - 1
		for _, b := range row {
			natural += b.Wide()
		}
		for _, width := range []int{natural, natural + 1, 44, 60, 100} {
			lines := Buttons(s, row, 0, width)
			for _, l := range lines {
				if got := lipgloss.Width(l); got != width {
					t.Errorf("%d buttons in %d cells drew %d: %q",
						len(row), width, got, plain(l))
				}
			}
		}
	}
}

// The slack goes out in proportion, so the button that asked for most still has
// most — a row where a one-letter button ends up as wide as the main action is a
// row that has lost its emphasis.
func TestTheStretchKeepsTheProportions(t *testing.T) {
	s := theme.Default()
	row := []Button{{Label: "▮▮ pause"}, {Label: Info}}
	wide := Stretch(row, 60, 1)
	if wide[0].Width <= wide[1].Width {
		t.Errorf("the main button is %d wide and the mark %d", wide[0].Width, wide[1].Width)
	}
	if got := wide[0].Width + wide[1].Width + 1; got != 60 {
		t.Errorf("the stretched row is %d wide, want 60", got)
	}
	// A row already wider than the space is left alone rather than squeezed.
	if tight := Stretch(row, 4, 1); tight[0].Width != 0 {
		t.Error("a row with no room was stretched anyway")
	}
	_ = s
}

// Keys end where the values begin: a ragged left edge of labels reads as a list
// of words rather than as a gutter.
func TestKeysAreRightAlignedAgainstTheirValues(t *testing.T) {
	s := theme.Default()
	pairs := []Pair{{Key: "port", Value: ":15432"}, {Key: "traffic", Value: "idle"}}
	lines := Pairs(s, pairs, 20)
	if len(lines) != 2 {
		t.Fatalf("drew %d lines", len(lines))
	}
	if got := plain(lines[0]); !strings.HasPrefix(got, "   port ") {
		t.Errorf("the short key is not padded on the left: %q", got)
	}
	if got := KeyWidth(pairs); got != len("traffic") {
		t.Errorf("KeyWidth = %d, want the widest key", got)
	}
}

// The option not taken is still an option, and has to be readable to be one.
// Drawn at the weight of a border it is a word nobody can read.
func TestTheUnchosenOptionIsReadable(t *testing.T) {
	s := theme.Default()
	r := Radio{Options: []string{"read only", "read · write"}, Active: 1}

	for _, focused := range []bool{false, true} {
		line := r.Render(s, focused)
		at := strings.Index(line, "read only")
		if at < 0 {
			t.Fatalf("focused=%v: the option is not there", focused)
		}
		// Whatever style opens the word, it is not the one the borders use.
		opened := line[:at]
		if strings.HasSuffix(opened, render(s.Rule, "")) {
			t.Errorf("focused=%v: the unchosen word is drawn in the rule colour", focused)
		}
		if !strings.Contains(opened, colourOf(s.Desc)) {
			t.Errorf("focused=%v: the unchosen word is not drawn as secondary text: %q",
				focused, opened)
		}
	}

	// And it reads the same whether or not the radio has the keys: which option
	// is set is what the drawing says, not where the cursor happens to be.
	before := plain(r.Render(s, false))
	after := plain(r.Render(s, true))
	if before != after {
		t.Errorf("focus changed the text: %q vs %q", before, after)
	}
}

func render(st lipgloss.Style, text string) string { return st.Render(text) }

// colourOf is the escape a style opens with, so a test can say which colour a
// run of text was drawn in rather than guessing from how it looks.
func colourOf(st lipgloss.Style) string {
	out := st.Render("x")
	if i := strings.Index(out, "x"); i > 0 {
		return out[:i]
	}
	return ""
}

// background is the escape lipgloss writes for a background colour, so a test
// can say which colour a run of cells was painted rather than how it looks.
func background(hex string) string {
	var r, g, b int
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

// A body line longer than the box runs straight through the right-hand border.
// A popup that cannot hold its own frame is worse than one that says less.
func TestAPopupKeepsItsFrameWhateverIsPutInIt(t *testing.T) {
	s := theme.Default()
	long := strings.Repeat("the engine did not answer ", 6)
	p := Popup{Title: "cannot start web", Body: []string{long, "short"},
		Buttons: []Button{{Label: "close"}}, Note: long}

	for _, width := range []int{20, 30, 44, 60} {
		for _, line := range p.Render(s, width) {
			if got := lipgloss.Width(line); got != width {
				t.Errorf("at width %d a line is %d wide: %q", width, got, plain(line))
			}
		}
	}
}

// A toggle is its own thing, not one of a set: two of them on at once is an
// ordinary state, where two radios filled is a choice that has gone wrong.
func TestAToggleSaysWhetherItIsOnWithoutExcludingAnything(t *testing.T) {
	s := theme.Default()
	on := Toggle{Label: "clickhouse", On: true, Tone: Good}
	off := Toggle{Label: "clickhouse"}

	for _, c := range []struct {
		what string
		g    Toggle
		box  string
	}{{"on", on, toggleOn}, {"off", off, toggleOff}} {
		for _, focused := range []bool{false, true} {
			line := c.g.Render(s, focused)
			if got := lipgloss.Width(line); got != c.g.Wide() {
				t.Errorf("%s: %d wide, Wide() says %d", c.what, got, c.g.Wide())
			}
			if !strings.HasPrefix(plain(line), c.box) {
				t.Errorf("%s: drawn with %q", c.what, plain(line))
			}
			if !strings.Contains(plain(line), "clickhouse") {
				t.Errorf("%s: lost its label", c.what)
			}
		}
	}
	// The state is in the drawing, and the text does not move with it.
	after := func(line string) string { return string([]rune(plain(line))[1:]) }
	if after(on.Render(s, false)) != after(off.Render(s, false)) {
		t.Errorf("turning a toggle on moved its label: %q vs %q",
			after(on.Render(s, false)), after(off.Render(s, false)))
	}
}

// A popup that lists things gives each its own switch, and the focus runs over
// the rows before the buttons because that is the order they are drawn.
func TestAPopupListsRowsWithTheirOwnSwitches(t *testing.T) {
	s := theme.Default()
	p := Popup{Title: "web", Body: []string{"branch CCC-4456"},
		Rows: []Row{
			{Toggle: Toggle{Label: "db", On: true, Tone: Good}, Say: "running"},
			{Toggle: Toggle{Label: "clickhouse"}, Say: "exited", Note: "optional"},
		},
		Buttons: []Button{{Label: "close"}}}

	if got := p.Stops(); got != 3 {
		t.Errorf("Stops() = %d, want two rows and a button", got)
	}
	box := p.Render(s, p.Wide())
	flat := strings.Join(box, "\n")
	for _, want := range []string{"db", "running", "clickhouse", "exited", "optional"} {
		if !strings.Contains(plain(flat), want) {
			t.Errorf("the list does not carry %q:\n%s", want, plain(flat))
		}
	}
	for _, line := range box {
		if got := lipgloss.Width(line); got != p.Wide() {
			t.Errorf("a line is %d wide, want %d: %q", got, p.Wide(), plain(line))
		}
	}

	// A row being switched says so, because a row that says nothing for the
	// several seconds a container takes looks broken.
	p.Rows[1].Busy = "starting…"
	if !strings.Contains(plain(strings.Join(p.Render(s, p.Wide()), "\n")), "starting…") {
		t.Error("a row being switched does not say so")
	}
}

// A click on a row has to land on the row that was drawn there.
func TestAClickFindsThePopupRowItWasDrawnOn(t *testing.T) {
	s := theme.Default()
	p := Popup{Title: "web", Body: []string{"branch CCC-4456"},
		Rows: []Row{
			{Toggle: Toggle{Label: "db", On: true}, Say: "running"},
			{Toggle: Toggle{Label: "web", On: true}, Say: "running"},
			{Toggle: Toggle{Label: "clickhouse"}, Say: "exited"},
		},
		Buttons: []Button{{Label: "close"}}}

	box := p.Render(s, p.Wide())
	screen := make([]string, 30)
	_, at := Over(screen, box, 80, 30)

	for want := range p.Rows {
		// The line each row was drawn on, found in the rendering rather than
		// assumed: if these two ever disagree the click lands on the wrong one.
		line := -1
		for i, l := range box {
			if strings.Contains(plain(l), p.Rows[want].Toggle.Label+" ") &&
				strings.Contains(plain(l), p.Rows[want].Say) {
				line = i
			}
		}
		if line < 0 {
			t.Fatalf("row %d was not drawn", want)
		}
		got, ok := at.RowAt(p, at.X+4, at.Y+line)
		if !ok || got != want {
			t.Errorf("a click on row %d found %d (ok=%v)", want, got, ok)
		}
	}
	// The body above them is not a row.
	if _, ok := at.RowAt(p, at.X+4, at.Y+2); ok {
		t.Error("a click on the body found a row")
	}
}

// The button row is the one thing in a popup that is not a list of facts, and a
// line of boxes hard against the left margin reads as the start of a column.
func TestAPopupCentresItsButtonRow(t *testing.T) {
	s := theme.Default()
	p := Popup{Title: "web", Body: []string{strings.Repeat("x", 50)},
		Buttons: []Button{{Label: "▸ start"}, {Label: "close"}}}

	box := p.Render(s, p.Wide())
	var row string
	for _, line := range box {
		if strings.Contains(plain(line), "▸ start") {
			row = plain(line)
		}
	}
	if row == "" {
		t.Fatal("the buttons were not drawn")
	}
	// The gap inside the border is the same on both sides, give or take the odd
	// cell that cannot be split.
	body := strings.TrimSuffix(strings.TrimPrefix(row, "│"), "│")
	left := len(body) - len(strings.TrimLeft(body, " "))
	right := len(body) - len(strings.TrimRight(body, " "))
	if left-right > 1 || right-left > 1 {
		t.Errorf("the row sits %d from the left and %d from the right: %q", left, right, row)
	}
}

// A mark and the word it belongs to are one thing. Drawn in two styles — the
// mark at the weight of a border, the word at the weight of text — the mark
// reads as part of the frame rather than as the thing being chosen.
func TestAMarkIsDrawnLikeTheWordItBelongsTo(t *testing.T) {
	s := theme.Default()

	for _, c := range []struct {
		what string
		line string
		mark string
	}{
		{"an unchosen radio", Radio{Options: []string{"read only", "read · write"},
			Active: 1}.Render(s, false), radioOff},
		{"a chosen radio", Radio{Options: []string{"read only"}, Active: 0,
			Tone: Good}.Render(s, false), radioOn},
		{"a toggle that is off", Toggle{Label: "clickhouse"}.Render(s, false), toggleOff},
		{"a toggle that is on", Toggle{Label: "clickhouse", On: true,
			Tone: Good}.Render(s, false), toggleOn},
	} {
		at := strings.Index(c.line, c.mark)
		if at < 0 {
			t.Errorf("%s: no mark drawn", c.what)
			continue
		}
		// Whatever run the mark is in reaches the word without an escape between
		// them: one style, one run.
		word := c.line[at+len(c.mark):]
		if i := strings.Index(word, "\x1b"); i >= 0 && strings.TrimSpace(plain(word[:i])) == "" {
			t.Errorf("%s: the style changes between the mark and its word: %q", c.what, c.line)
		}
	}
}
