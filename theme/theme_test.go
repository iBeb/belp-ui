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
