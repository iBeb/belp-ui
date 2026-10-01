package browse

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// One line, the way every list in these apps draws one: a mark in the margin,
// the name taking whatever is left, and the columns given up widest-Give first
// as the terminal narrows — the name being the last thing to go.
func TestALineGivesUpItsColumnsBeforeItsName(t *testing.T) {
	s := theme.Default()
	cells := []Cell{
		{Text: "today at 11:27", Wide: 14, Weight: Quiet, Give: 2},
		{Text: "ES#725 the work itself, which is what tells one row from another"},
		{Text: "review", Wide: 6, Weight: Quiet, Right: true, Give: 1},
	}

	wide := plain(Line(s, false, 100, "●", cells...))
	for _, want := range []string{"●", "today at 11:27", "ES#725 the work", "review"} {
		if !strings.Contains(wide, want) {
			t.Errorf("a wide line lost %q: %q", want, wide)
		}
	}
	if got := lipgloss.Width(wide); got > 100 {
		t.Errorf("it ran to %d cells of 100", got)
	}

	// Narrow enough that the name would be squeezed: the date goes first,
	// because its Give is the largest.
	narrow := plain(Line(s, false, 40, "●", cells...))
	if strings.Contains(narrow, "today at") {
		t.Errorf("the date stayed on a line with no room: %q", narrow)
	}
	if !strings.Contains(narrow, "ES#725") {
		t.Errorf("the name went before a column did: %q", narrow)
	}
	// And the loop stays, because by then the name has its twenty cells and
	// giving up more would buy nothing.
	if strings.Contains(narrow, "review") == false {
		t.Errorf("a column was given up that was not in the way: %q", narrow)
	}
	// Narrower still, and it goes too — the name outlasts everything.
	tight := plain(Line(s, false, 24, "●", cells...))
	if strings.Contains(tight, "review") || !strings.Contains(tight, "ES#725") {
		t.Errorf("at 24 cells it says %q", tight)
	}
}

func plain(said string) string {
	var b strings.Builder
	for i := 0; i < len(said); i++ {
		if said[i] == 0x1b {
			for i < len(said) && said[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(said[i])
	}
	return b.String()
}
