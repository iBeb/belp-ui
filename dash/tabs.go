package dash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// Tabs is one row of names with one of them current, over a band that shows
// that one's cards.
//
// For a block that would otherwise be several blocks of the same shape — the
// same four ways in, to a second cluster — where stacking them makes a board
// nobody can read at a glance. Not for navigation between unlike things: a tab
// hides what is under it, and a thing worth watching is not worth hiding.
type Tabs struct {
	Labels []string
	// Marked flags a tab whose own band has something live, so that what is
	// hidden still says it is there. Shorter than Labels is fine: the tabs it
	// does not reach are simply unmarked.
	Marked []bool
	Active int
}

// The mark on a tab that has something running under it, and the brackets the
// current tab wears.
const (
	tabLive  = "●"
	tabOpen  = "["
	tabShut  = "]"
	tabSpace = " "
)

// Render draws the strip, with a rule running out to the width so the tabs read
// as the top of the band rather than as a row of words above it.
//
// The current tab is bracketed and the others are padded to the same width, so
// that moving between them does not shift the row under the cursor.
func (t Tabs) Render(s theme.Styles, focused bool, width int) string {
	if len(t.Labels) == 0 {
		return ""
	}
	// The usual blue for the tab that is current, filled where the cursor is on
	// the strip. No underline: an underline under a word in a row of words reads
	// as a link, and the colour is the app's own mark for "this is the one".
	here := s.Selected
	if focused {
		here = s.Selection
	}
	parts := make([]string, len(t.Labels))
	for i, label := range t.Labels {
		text := label + t.mark(i)
		if i == t.Active {
			parts[i] = here.Render(tabOpen + tabSpace + text + tabSpace + tabShut)
			continue
		}
		parts[i] = s.Desc.Render(tabSpace + tabSpace + text + tabSpace + tabSpace)
	}
	bar := strings.Join(parts, tabSpace)
	if room := width - lipgloss.Width(bar) - 1; room > 0 {
		bar += tabSpace + s.Rule.Render(strings.Repeat("─", room))
	}
	return bar
}

// mark is the live dot for one tab, or nothing.
func (t Tabs) mark(i int) string {
	if i < len(t.Marked) && t.Marked[i] {
		return tabSpace + tabLive
	}
	return ""
}

// Spans is where each tab sits along the strip — the offset and the width of
// one — so a caller can register a click target per tab rather than one for the
// row. A tab you can see is a tab you expect to be able to hit.
func (t Tabs) Spans() [][2]int {
	out := make([][2]int, len(t.Labels))
	at := 0
	for i, label := range t.Labels {
		w := lipgloss.Width(label+t.mark(i)) + 4
		out[i] = [2]int{at, w}
		at += w + 1 // the space between one tab and the next
	}
	return out
}

// Wide is how many cells the tabs themselves take, before the rule.
func (t Tabs) Wide() int {
	n := max(0, len(t.Labels)-1)
	for i, label := range t.Labels {
		n += lipgloss.Width(label+t.mark(i)) + 4
	}
	return n
}
