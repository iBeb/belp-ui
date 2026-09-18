package theme

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var hex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Walking the struct by reflection rather than listing the fields: a colour
// added to Palette and forgotten in DefaultPalette is exactly the mistake a
// hand-written list of assertions would not catch, because nobody updates it.
func TestDefaultPaletteIsComplete(t *testing.T) {
	p := reflect.ValueOf(DefaultPalette())
	for i := 0; i < p.NumField(); i++ {
		name := p.Type().Field(i).Name
		c, ok := p.Field(i).Interface().(lipgloss.AdaptiveColor)
		if !ok {
			t.Fatalf("%s is not an AdaptiveColor; the test needs updating", name)
		}
		if !hex.MatchString(c.Light) {
			t.Errorf("%s has no usable light colour: %q", name, c.Light)
		}
		if !hex.MatchString(c.Dark) {
			t.Errorf("%s has no usable dark colour: %q", name, c.Dark)
		}
		if c.Light == c.Dark {
			t.Errorf("%s is the same in both profiles (%s), so one of them is wrong",
				name, c.Light)
		}
	}
}

// Same reasoning in the other direction: a style added to Styles and left
// unset in New would render as the terminal's default and look like a bug
// somewhere else entirely.
func TestEveryStyleIsBuilt(t *testing.T) {
	s := reflect.ValueOf(Default())
	for i := 0; i < s.NumField(); i++ {
		f := s.Type().Field(i)
		if f.Type != reflect.TypeOf(lipgloss.Style{}) {
			continue
		}
		style, ok := s.Field(i).Interface().(lipgloss.Style)
		if !ok {
			t.Fatalf("%s is not a Style; the test needs updating", f.Name)
		}
		if style.GetForeground() == (lipgloss.NoColor{}) {
			t.Errorf("%s has no foreground: New does not set it", f.Name)
		}
	}
}

// The styles have to follow the palette they were built from, or a theme is
// decoration rather than configuration.
func TestStylesFollowTheirPalette(t *testing.T) {
	p := DefaultPalette()
	p.Accent = lipgloss.AdaptiveColor{Light: "#123456", Dark: "#654321"}
	s := New(p)

	for _, tc := range []struct {
		name  string
		style lipgloss.Style
	}{
		{"App", s.App},
		{"Selected", s.Selected},
	} {
		if got := tc.style.GetForeground(); got != p.Accent {
			t.Errorf("%s uses %v, not the palette's accent %v", tc.name, got, p.Accent)
		}
	}
}

// Layout belongs to the screen being drawn, not to a shared style: a width or
// a border baked in here is wrong on every screen that wanted a different one.
func TestStylesCarryNoLayout(t *testing.T) {
	s := reflect.ValueOf(Default())
	for i := 0; i < s.NumField(); i++ {
		f := s.Type().Field(i)
		if f.Type != reflect.TypeOf(lipgloss.Style{}) {
			continue
		}
		style := s.Field(i).Interface().(lipgloss.Style)
		if w := style.GetWidth(); w != 0 {
			t.Errorf("%s fixes a width of %d", f.Name, w)
		}
		if style.GetBorderStyle() != lipgloss.HiddenBorder() && style.GetBorderTop() {
			t.Errorf("%s carries a border", f.Name)
		}
		if h, v := style.GetHorizontalPadding(), style.GetVerticalPadding(); h != 0 || v != 0 {
			t.Errorf("%s carries padding (%d, %d)", f.Name, h, v)
		}
	}
}

// A selected row is one colour throughout. The eye follows colour, so a row
// coloured in one place and not another reads as two rows.
func TestForRowPaintsTheWholeRow(t *testing.T) {
	s := Default().ForRow(true)
	accent := DefaultPalette().Accent

	v := reflect.ValueOf(s)
	// Everything a row draws with. Chrome styles are not a row's business and
	// are left alone, which is why this is a list rather than every field.
	for _, name := range []string{
		"Item", "Desc", "Label", "Value",
		"Success", "Warn", "Danger",
		"Begun", "Flight", "Spent", "Voice", "Judge",
	} {
		style, ok := v.FieldByName(name).Interface().(lipgloss.Style)
		if !ok {
			t.Fatalf("%s is not a Style; the test needs updating", name)
		}
		if got := style.GetForeground(); got != accent {
			t.Errorf("%s renders in %v on a selected row, want the accent %v", name, got, accent)
		}
	}
}

// And an unselected row is left exactly as it was: ForRow is not a way to
// recolour a screen by accident.
func TestForRowLeavesAnUnselectedRowAlone(t *testing.T) {
	plain, same := Default(), Default().ForRow(false)
	v1, v2 := reflect.ValueOf(plain), reflect.ValueOf(same)
	for i := 0; i < v1.NumField(); i++ {
		f := v1.Type().Field(i)
		if f.Type != reflect.TypeOf(lipgloss.Style{}) {
			continue
		}
		a := v1.Field(i).Interface().(lipgloss.Style)
		b := v2.Field(i).Interface().(lipgloss.Style)
		if a.GetForeground() != b.GetForeground() || a.GetBold() != b.GetBold() {
			t.Errorf("%s changed for an unselected row", f.Name)
		}
	}
}

// A label is the name of a thing rather than the thing, and has to be quieter
// than the value beside it — on both grounds, a block of facts that draws its
// keys at the weight of its values reads as twice as much text as it is.
func TestLabelsAreQuieterThanTheValuesTheyName(t *testing.T) {
	p := DefaultPalette()
	for _, c := range []struct {
		what  string
		quiet lipgloss.AdaptiveColor
		loud  lipgloss.AdaptiveColor
	}{
		{"label against value", p.Quiet, p.Text},
		{"label against secondary text", p.Quiet, p.Dim},
	} {
		for _, dark := range []bool{false, true} {
			if near(pick(c.quiet, dark), pick(c.loud, dark)) {
				t.Errorf("dark=%v: %s — %q and %q are the same shade",
					dark, c.what, pick(c.quiet, dark), pick(c.loud, dark))
			}
		}
	}
	if New(p).Label.GetForeground() == New(p).Value.GetForeground() {
		t.Error("Label and Value draw in the same colour")
	}
}

func pick(c lipgloss.AdaptiveColor, dark bool) string {
	if dark {
		return c.Dark
	}
	return c.Light
}

// near reports whether two hex colours are within a hair of each other, which is
// the failure worth catching: not "wrong colour" but "no visible difference".
func near(a, b string) bool {
	if a == b {
		return true
	}
	ar, ag, ab := rgb(a)
	br, bg, bb := rgb(b)
	return abs(ar-br)+abs(ag-bg)+abs(ab-bb) < 48
}

func rgb(s string) (int, int, int) {
	var r, g, b int
	_, _ = fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// A resting panel goes quiet all at once. Moving some colours a long way and
// others hardly at all is what made it read as a panel with something wrong on
// it rather than one that is simply not in use.
func TestRestingMovesEveryColourByTheSameAmount(t *testing.T) {
	p := DefaultPalette()
	q := p.AtRest()

	var moved []float64
	each(p, q, func(name string, from, to lipgloss.AdaptiveColor) {
		if name == "Ground" {
			return // the background does not move toward itself
		}
		for _, dark := range []bool{false, true} {
			a, b := pick(from, dark), pick(to, dark)
			if a == b {
				t.Errorf("dark=%v: %s did not move at all", dark, name)
				continue
			}
			moved = append(moved, travel(a, b, dark))
		}
	})
	if len(moved) == 0 {
		t.Fatal("nothing was compared")
	}
	lo, hi := moved[0], moved[0]
	for _, d := range moved {
		lo, hi = min(lo, d), max(hi, d)
	}
	if hi-lo > 0.02 {
		t.Errorf("colours moved between %.2f and %.2f of the way; want one step for all",
			lo, hi)
	}
	if hi > 0.5 {
		t.Errorf("resting moves %.2f of the way to the background, which is not a light hand", hi)
	}
}

// What a colour means has to survive resting: a red faded until it reads as grey
// has stopped saying what red says.
func TestRestingKeepsAColourRecognisable(t *testing.T) {
	q := DefaultPalette().AtRest()
	r, g, b := rgb(q.Danger.Dark)
	if r <= g || r <= b {
		t.Errorf("a resting danger is %q, which is no longer red", q.Danger.Dark)
	}
	gr, gg, gb := rgb(q.Success.Dark)
	if gg <= gr || gg <= gb {
		t.Errorf("a resting success is %q, which is no longer green", q.Success.Dark)
	}
}

// each walks the colours of two palettes side by side.
func each(a, b Palette, f func(name string, from, to lipgloss.AdaptiveColor)) {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	for i := 0; i < av.NumField(); i++ {
		from, ok := av.Field(i).Interface().(lipgloss.AdaptiveColor)
		if !ok {
			continue
		}
		to := bv.Field(i).Interface().(lipgloss.AdaptiveColor)
		f(av.Type().Field(i).Name, from, to)
	}
}

// travel is how far a colour moved toward the ground, as a fraction.
func travel(from, to string, dark bool) float64 {
	ground := DefaultPalette().Ground.Light
	if dark {
		ground = DefaultPalette().Ground.Dark
	}
	er, eg, eb := rgb(ground)
	fr, fg, fb := rgb(from)
	tr, tg, tb := rgb(to)
	var sum float64
	n := 0
	for i, c := range [][2]int{{fr, tr}, {fg, tg}, {fb, tb}} {
		gap := float64([]int{er, eg, eb}[i]) - float64(c[0])
		if gap == 0 {
			continue
		}
		sum += (float64(c[1]) - float64(c[0])) / gap
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// A fill runs from the background to the hue itself.
func TestFillWalksFromTheGroundToTheHue(t *testing.T) {
	p := DefaultPalette()
	if got := p.Fill(p.Danger, 0); got != p.Ground {
		t.Errorf("Fill at 0 is %v, want the ground %v", got, p.Ground)
	}
	if got := p.Fill(p.Danger, 1); got != p.Danger {
		t.Errorf("Fill at 1 is %v, want the hue %v", got, p.Danger)
	}
	// And it climbs: every step is lighter than the one below it on a dark
	// terminal, which is what makes a ramp read as a ramp.
	last := -1.0
	for i := 0; i <= 10; i++ {
		l := luminance(p.Fill(p.Danger, float64(i)/10).Dark)
		if l < last {
			t.Errorf("level %d is darker than the one below it", i)
		}
		last = l
	}
}

// Every label the palette picks has to be readable on the fill it sits on.
//
// This is the property the whole ramp rests on, and the one a new tone breaks
// silently: a hue added to the palette that happens to land mid-grey is too
// dark for the ground and too light for the text, and the button looks fine to
// whoever added it and unreadable on somebody else's terminal.
func TestALabelIsReadableOnEveryFill(t *testing.T) {
	const floor = 4.5 // WCAG AA for text of an ordinary size

	p := DefaultPalette()
	for _, tone := range []struct {
		name string
		c    AdaptiveColor
	}{
		{"Primary", p.Primary}, {"Secondary", p.Secondary}, {"Accent", p.Accent},
		{"Success", p.Success}, {"Warn", p.Warn}, {"Danger", p.Danger},
	} {
		// The levels a control is actually drawn at, and the same again for a
		// panel at rest.
		for _, level := range []float64{0.2, 0.5, 1, 0.2 * Rest, 0.5 * Rest, Rest} {
			fill := p.Fill(tone.c, level)
			on := p.On(fill)
			for _, v := range []struct{ what, fg, bg string }{
				{"light", on.Light, fill.Light},
				{"dark", on.Dark, fill.Dark},
			} {
				if got := contrast(v.bg, v.fg); got < floor {
					t.Errorf("%s at %.2f on a %s terminal: %s on %s is %.2f:1, want %.1f",
						tone.name, level, v.what, v.fg, v.bg, got, floor)
				}
			}
		}
	}
}

// A link is where a row hands off, and says so in two ways: an underline, which
// a reader already knows, and a blue quieter than the accent, which means "the
// thing you are about to act on" and would be spent if every link claimed it.
func TestALinkIsUnderlinedAndQuieterThanTheAccent(t *testing.T) {
	s := Default()
	if !s.Link.GetUnderline() {
		t.Error("a link is not underlined")
	}
	if s.Palette.Link == s.Palette.Accent {
		t.Error("a link is drawn in the accent, which is the cursor's colour")
	}
	if s.Link.GetForeground() != lipgloss.Color(s.Palette.Link.Dark) &&
		s.Link.GetForeground() != lipgloss.Color(s.Palette.Link.Light) {
		// Adaptive colours resolve at render time; what matters here is only
		// that the link does not wear the accent.
		if s.Link.GetForeground() == s.Selected.GetForeground() {
			t.Error("a link is the same colour as the cursor")
		}
	}

	// On the selected row it takes the accent with everything else, and keeps
	// the underline: what it is does not change because the cursor is there.
	on := s.ForRow(true).Link
	if !on.GetUnderline() {
		t.Error("a selected link lost its underline")
	}
	if on.GetForeground() != s.Selected.GetForeground() {
		t.Error("a selected link is not the accent")
	}
}

// Every colour fades at rest, the link with them: one that stayed bright in a
// pane nobody is typing into would be the brightest thing on it.
func TestTheLinkColourFadesAtRest(t *testing.T) {
	s := Default()
	if s.AtRest().Palette.Link == s.Palette.Link {
		t.Error("the link colour is the same at rest as in the foreground")
	}
}
