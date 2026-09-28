package dash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// A title too long for its box is cut to what there is room for. Dropped
// instead — which is what happened — the window with the most to say is the
// one drawn with nothing written on it.
func TestATitleTooLongIsCutRatherThanDropped(t *testing.T) {
	p := Popup{Title: strings.Repeat("resume stopped because ", 6)}
	top := plain(p.Render(theme.Default(), 40)[0])
	if !strings.Contains(top, "resume") {
		t.Errorf("the title is not on the window: %q", top)
	}
	if got := lipgloss.Width(top); got != 40 {
		t.Errorf("the border is %d cells wide, not 40:\n%q", got, top)
	}
}
