package dash

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

func plainTabs(t Tabs, focused bool, width int) string {
	return plain(t.Render(theme.Default(), focused, width))
}

// The current tab is bracketed and the rest are padded to match, so the row
// under the cursor does not shift as the cursor moves along it.
func TestTheCurrentTabIsBracketedAndTheRestKeepItsWidth(t *testing.T) {
	tabs := Tabs{Labels: []string{"staging", "demo"}, Active: 0}
	first := plainTabs(tabs, false, 60)
	if !strings.Contains(first, "[ staging ]") {
		t.Errorf("Render() = %q, want the current tab bracketed", first)
	}
	if strings.Contains(first, "[ demo ]") {
		t.Errorf("Render() = %q, want only one tab bracketed", first)
	}

	tabs.Active = 1
	second := plainTabs(tabs, false, 60)
	if !strings.Contains(second, "[ demo ]") {
		t.Errorf("Render() = %q, want the current tab bracketed", second)
	}
	if a, b := lipgloss.Width(first), lipgloss.Width(second); a != b {
		t.Errorf("the strip is %d cells then %d: moving along it must not shift the row", a, b)
	}
}

// A tab hides its band, so a tab with something running under it says so.
func TestAMarkedTabSaysSomethingIsLiveUnderIt(t *testing.T) {
	tabs := Tabs{Labels: []string{"staging", "demo"}, Marked: []bool{false, true}, Active: 0}
	got := plainTabs(tabs, false, 60)
	if !strings.Contains(got, "demo "+tabLive) {
		t.Errorf("Render() = %q, want the live tab marked", got)
	}
	if strings.Contains(got, "staging "+tabLive) {
		t.Errorf("Render() = %q, want the quiet tab unmarked", got)
	}
	// Marked may be shorter than Labels, and the tabs it does not reach are
	// simply unmarked rather than a panic.
	short := Tabs{Labels: []string{"a", "b", "c"}, Marked: []bool{true}}
	if got := plainTabs(short, false, 40); strings.Count(got, tabLive) != 1 {
		t.Errorf("Render() = %q, want one mark", got)
	}
}

// The rule runs out to the width, which is what makes the strip the top of the
// band rather than a row of words floating over it.
func TestTheRuleFillsTheWidth(t *testing.T) {
	tabs := Tabs{Labels: []string{"staging", "demo"}}
	for _, width := range []int{40, 80, 120} {
		if got := lipgloss.Width(plainTabs(tabs, false, width)); got != width {
			t.Errorf("Render(width %d) is %d cells", width, got)
		}
	}
	// Too narrow for a rule: the tabs are what matters and they stay whole.
	narrow := plainTabs(tabs, false, tabs.Wide())
	if !strings.Contains(narrow, "staging") || !strings.Contains(narrow, "demo") {
		t.Errorf("Render(narrow) = %q, want both tabs", narrow)
	}
	if got := (Tabs{}).Render(theme.Default(), false, 80); got != "" {
		t.Errorf("Render() with no tabs = %q, want nothing drawn", got)
	}
}

// Wide is what the strip takes before the rule, so a caller can tell whether
// there is room for one.
func TestWideCountsTheTabsAndTheirMarks(t *testing.T) {
	tabs := Tabs{Labels: []string{"staging", "demo"}}
	if got, want := tabs.Wide(), lipgloss.Width(plainTabs(tabs, false, 0)); got != want {
		t.Errorf("Wide() = %d, want the %d cells it draws", got, want)
	}
	marked := Tabs{Labels: []string{"staging", "demo"}, Marked: []bool{true, true}}
	if marked.Wide() <= tabs.Wide() {
		t.Error("Wide() ignores the marks, which are drawn")
	}
}

// Every tab is its own click target, so the spans have to be where the tabs
// actually are: a strip whose offsets are a guess is a strip that selects the
// tab beside the one aimed at.
func TestSpansLandOnTheTabsThemselves(t *testing.T) {
	tabs := Tabs{Labels: []string{"staging", "demo"}, Marked: []bool{false, true}}
	drawn := []rune(plainTabs(tabs, false, 80))
	spans := tabs.Spans()
	if len(spans) != 2 {
		t.Fatalf("Spans() = %v, want one per tab", spans)
	}
	for i, label := range tabs.Labels {
		at, w := spans[i][0], spans[i][1]
		got := strings.TrimSpace(string(drawn[at : at+w]))
		if !strings.Contains(got, label) {
			t.Errorf("span %d covers %q, want the tab saying %q", i, got, label)
		}
	}
	// The last span ends where the tabs end, before the rule.
	last := spans[len(spans)-1]
	if got := last[0] + last[1]; got != tabs.Wide() {
		t.Errorf("the spans run to %d, want the %d cells the tabs take", got, tabs.Wide())
	}
}
