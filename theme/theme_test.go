package theme

import (
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
