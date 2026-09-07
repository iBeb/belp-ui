package browse

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Two things about the right edge, and they are not the same thing.
//
// Nothing may run past it: a band that reaches the last column while the others
// stop short reads as overflowing the screen. And a band that is meant to span
// the width has to actually reach it, which is the half that truncation hides —
// a band handed too little width looks fine until you notice it ends early.
//
// Measured by filling the app-drawn bands to whatever width they are handed, so
// the rendered line reports the width the band was really allotted.
func TestNoBandRunsPastTheRightEdge(t *testing.T) {
	forEachBand(t, func(t *testing.T, width int, name string, y int, line string) {
		want := width - margin // one column of air, as on the left
		if got := lipgloss.Width(line); got > want {
			t.Errorf("width %d: %s row %d ends at column %d, past %d\n  %q",
				width, name, y, got, want, line)
		}
	})
}

// The bands that span: the list and the preview are filled to the width they are
// given, and the search box is a border drawn to it.
func TestSpanningBandsReachTheRightEdge(t *testing.T) {
	spans := map[string]bool{"list": true, "preview": true, "search": true}
	forEachBand(t, func(t *testing.T, width int, name string, y int, line string) {
		if !spans[name] {
			return
		}
		want := width - margin
		if got := lipgloss.Width(line); got != want {
			t.Errorf("width %d: %s row %d ends at column %d, want %d\n  %q",
				width, name, y, got, want, line)
		}
	})
}

// forEachBand renders a screen at several widths and hands each non-empty band
// line to check.
func forEachBand(t *testing.T, check func(t *testing.T, width int, name string, y int, line string)) {
	t.Helper()
	for _, width := range []int{40, 60, 80, 120, 200} {
		m := New(sample())
		m.Row = func(_, w int, _ bool) string { return strings.Repeat("r", w) }
		m.Preview = func(_, w, h int) []string {
			out := make([]string, h)
			for i := range out {
				out[i] = strings.Repeat("p", w)
			}
			return out
		}
		m.SetSize(width, 30)
		m.SetRowCount(5)

		l := m.Layout()
		lines := strings.Split(m.View(), "\n")
		for _, band := range []struct {
			name string
			r    Region
		}{
			{"filters", l.Filters},
			{"search", l.Search},
			{"list", l.List},
			{"preview", l.Preview},
		} {
			if band.r.Empty() {
				continue
			}
			for y := band.r.Y; y <= band.r.Bottom() && y < len(lines); y++ {
				line := strings.TrimRight(lines[y], " ")
				if line == "" {
					continue // a band with nothing to draw on that row
				}
				check(t, width, band.name, y, line)
			}
		}
	}
}

// A band keeps the width it is handed: nothing is laid out and then trimmed.
//
// The visible result of getting this wrong is nothing at all — the trimming
// makes the edge look right — so it is checked against the width the callback
// was given rather than against the picture. An app that lays out columns to a
// width two wider than it keeps loses the end of the last one, and the only
// symptom is content that quietly is not there.
func TestABandKeepsTheWidthItIsHanded(t *testing.T) {
	for _, width := range []int{40, 80, 160} {
		var rowWidth, previewWidth int
		m := New(sample())
		m.Row = func(_, w int, _ bool) string {
			rowWidth = w
			return strings.Repeat("r", w)
		}
		m.Preview = func(_, w, h int) []string {
			previewWidth = w
			out := make([]string, h)
			for i := range out {
				out[i] = strings.Repeat("p", w)
			}
			return out
		}
		m.SetSize(width, 30)
		m.SetRowCount(5)

		l := m.Layout()
		lines := strings.Split(m.View(), "\n")

		// A row is not inset, so what it drew is what shows.
		if got := lipgloss.Width(strings.TrimRight(lines[l.List.Y], " ")); got != rowWidth {
			t.Errorf("width %d: row was handed %d columns and %d survived",
				width, rowWidth, got)
		}
		// The preview is inset by a margin, so that much is added to what it drew.
		if !l.Preview.Empty() {
			got := lipgloss.Width(strings.TrimRight(lines[l.Preview.Y], " "))
			if got != previewWidth+margin {
				t.Errorf("width %d: preview was handed %d columns and %d survived",
					width, previewWidth, got-margin)
			}
		}
	}
}
