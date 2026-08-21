package browse

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iBeb/belp-ui/theme"
)

func sample() Chrome {
	return Chrome{
		Styles: theme.Default(),
		App:    "belp",
		Crumbs: []string{"Sillage"},
		Status: "77 pull requests",
		Groups: []Group{
			{Label: "did", Options: []Option{
				{Label: "opened", Selected: true},
				{Label: "reviewed"},
				{Label: "merged", Focused: true},
			}},
			{Label: "is", Options: []Option{
				{Label: "open"},
				{Label: "merged", Selected: true},
			}},
			{Options: []Option{{Label: "30 days", Selected: true}}},
		},
		Query:       "geo",
		Keys:        []Key{{"↵", "open"}, {"^R", "review"}, {"^Q", "quit"}},
		Placeholder: "type to filter",
	}
}

// The contract a caller relies on: exactly as many lines as the terminal has
// rows, none of them wider than it is. Checked across sizes, because that is
// where padding and truncation go wrong.
func TestRenderFillsTheTerminalExactly(t *testing.T) {
	c := sample()
	rows := []string{"one", "two", "three"}
	preview := []string{"a preview", "and more of it"}

	for _, height := range []int{1, 2, 5, 8, 10, 24, 30, 60} {
		for _, width := range []int{10, 40, 80, 200} {
			l := Compute(width, height, false)
			got := c.Render(l, rows, preview)

			lines := strings.Split(got, "\n")
			if len(lines) != height {
				t.Errorf("%dx%d: %d line(s), want %d", width, height, len(lines), height)
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("%dx%d: line %d is %d cells wide: %q", width, height, i, w, line)
				}
			}
		}
	}
}

func TestRenderPlacesEachBand(t *testing.T) {
	c := sample()
	l := Compute(100, 30, false)
	lines := strings.Split(c.Render(l, []string{"the first row"}, []string{"the preview"}), "\n")

	cases := []struct {
		name string
		y    int
		want string
	}{
		{"header", l.Header.Y, "belp"},
		{"filters", l.Filters.Y, "opened"},
		{"search", l.Search.Y + 1, "geo"}, // the row inside the box
		{"list", l.List.Y, "the first row"},
		{"preview", l.Preview.Y, "the preview"},
		{"footer", l.Footer.Y, "open"},
	}
	for _, tc := range cases {
		if !strings.Contains(lines[tc.y], tc.want) {
			t.Errorf("%s (row %d) = %q, want it to contain %q", tc.name, tc.y, lines[tc.y], tc.want)
		}
	}
	for _, y := range []int{l.HeaderRule.Y, l.PreviewRule.Y} {
		if !strings.Contains(lines[y], "─") {
			t.Errorf("row %d = %q, want a rule", y, lines[y])
		}
	}
	// The search box brings its own edges, top and bottom, and the row under it
	// belongs to the list rather than to a rule repeating what the box said.
	for _, y := range []int{l.Search.Y, l.Search.Bottom()} {
		if !strings.Contains(lines[y], "─") {
			t.Errorf("row %d = %q, want a box edge", y, lines[y])
		}
	}
}

// More rows than the band has room for are dropped, not overflowed into the
// preview. Choosing which rows to show is scrolling, and that is the caller's.
func TestRenderDropsRowsThatDoNotFit(t *testing.T) {
	c := sample()
	l := Compute(80, 24, false)

	rows := make([]string, l.List.Height+10)
	for i := range rows {
		rows[i] = "row"
	}
	rows[l.List.Height] = "OVERFLOW"

	got := c.Render(l, rows, nil)
	if strings.Contains(got, "OVERFLOW") {
		t.Error("a row past the end of the list band was drawn")
	}
	if n := strings.Count(got, "row"); n != l.List.Height {
		t.Errorf("drew %d row(s), want %d", n, l.List.Height)
	}
}

// A band the layout left out must not be drawn into.
func TestRenderSkipsAbsentBands(t *testing.T) {
	c := sample()
	l := Compute(80, 9, false) // too short for a preview
	if !l.Preview.Empty() {
		t.Fatalf("expected no preview at 9 rows, got %+v", l.Preview)
	}
	got := c.Render(l, []string{"a row"}, []string{"PREVIEW TEXT"})
	if strings.Contains(got, "PREVIEW TEXT") {
		t.Error("preview text was drawn although there is no preview band")
	}
}

func TestHeaderDropsTheStatusBeforeTheApp(t *testing.T) {
	c := sample()

	wide := c.Header(80)
	if !strings.Contains(wide, "belp") || !strings.Contains(wide, "77 pull requests") {
		t.Errorf("Header(80) = %q, want both the crumbs and the status", wide)
	}
	// The status is right-aligned, so it ends where the header ends.
	if lipgloss.Width(wide) != 80 {
		t.Errorf("Header(80) is %d cells, want the status pushed to the right edge", lipgloss.Width(wide))
	}

	narrow := c.Header(20)
	if strings.Contains(narrow, "77 pull requests") {
		t.Errorf("Header(20) = %q, want the status dropped", narrow)
	}
	if !strings.Contains(narrow, "belp") {
		t.Errorf("Header(20) = %q, want the app name kept", narrow)
	}
}

func TestHeaderShowsTheCrumbTrail(t *testing.T) {
	c := sample()
	c.Crumbs = []string{"Sillage", "Reviews"}
	got := c.Header(120)
	for _, want := range []string{"belp", theme.Chevron, "Sillage", "Reviews"} {
		if !strings.Contains(got, want) {
			t.Errorf("Header() = %q, want it to contain %q", got, want)
		}
	}
}

// The bar must not shift as the cursor moves along it: a focused chip is
// bracketed and an unfocused one is padded to the same width.
func TestFiltersKeepTheirWidthAsFocusMoves(t *testing.T) {
	c := sample()
	c.Focus = FocusFilters

	first := c.Filters(200)
	for i := range c.Groups[0].Options {
		for j := range c.Groups[0].Options {
			c.Groups[0].Options[j].Focused = i == j
		}
		if got := c.Filters(200); lipgloss.Width(got) != lipgloss.Width(first) {
			t.Errorf("focusing chip %d changed the bar from %d to %d cells",
				i, lipgloss.Width(first), lipgloss.Width(got))
		}
	}
}

func TestFiltersBracketOnlyTheFocusedChipAndOnlyWhenFocused(t *testing.T) {
	c := sample()

	c.Focus = FocusFilters
	got := c.Filters(200)
	// The cursor is on a chip that is not set, so it gets the brackets and no
	// bullet: the two say different things and are drawn separately.
	if !strings.Contains(got, "[ merged]") {
		t.Errorf("Filters() = %q, want the focused chip bracketed", got)
	}
	if strings.Contains(got, "[ opened]") || strings.Contains(got, "["+theme.Bullet+"opened]") {
		t.Errorf("Filters() = %q, want only the focused chip bracketed", got)
	}

	// With the focus elsewhere there is no cursor on the bar to show.
	c.Focus = FocusList
	if got := c.Filters(200); strings.Contains(got, "[") {
		t.Errorf("Filters() = %q, want no brackets while the list has focus", got)
	}
}

func TestFiltersShowEveryGroup(t *testing.T) {
	got := sample().Filters(200)
	for _, want := range []string{"did", "opened", "reviewed", "is", "open", "30 days"} {
		if !strings.Contains(got, want) {
			t.Errorf("Filters() = %q, want it to contain %q", got, want)
		}
	}
}

// The labels are decoration and the chips are state, so a bar under pressure
// gives up the labels and keeps the chips.
func TestFiltersDropLabelsBeforeChips(t *testing.T) {
	c := sample()

	full := c.Filters(200)
	width := lipgloss.Width(c.filterBar(false))
	tight := c.Filters(width)

	if strings.Contains(tight, "did") || strings.Contains(tight, "is ") {
		t.Errorf("Filters(%d) = %q, want the labels dropped", width, tight)
	}
	for _, want := range []string{"opened", "reviewed", "merged", "open", "30 days"} {
		if !strings.Contains(tight, want) {
			t.Errorf("Filters(%d) = %q, want the chip %q kept", width, tight, want)
		}
	}
	if lipgloss.Width(full) <= width {
		t.Fatal("the labelled bar was not wider; the test proves nothing")
	}
}

// A chip that is set and off the end of the bar means a list filtered in a way
// you cannot see. Saying it was cut is the least the bar can do.
func TestFiltersElideRatherThanCutSilently(t *testing.T) {
	c := sample()
	for _, width := range []int{1, 2, 10, 30, 50} {
		got := c.Filters(width)
		if w := lipgloss.Width(got); w > width {
			t.Errorf("Filters(%d) is %d cells: %q", width, w, got)
		}
		if !strings.Contains(got, "…") {
			t.Errorf("Filters(%d) = %q, want an ellipsis: chips were cut", width, got)
		}
	}
}

func TestSearchShowsThePlaceholderOnlyWhileEmpty(t *testing.T) {
	c := sample()

	if got := c.Search(40); !strings.Contains(got, "geo") || strings.Contains(got, "type to filter") {
		t.Errorf("Search() = %q, want the query and no placeholder", got)
	}
	c.Query = ""
	if got := c.Search(40); !strings.Contains(got, "type to filter") {
		t.Errorf("Search() = %q, want the placeholder", got)
	}
}

// You type at the end of a query, so the end is what has to stay on screen.
func TestSearchElidesFromTheLeft(t *testing.T) {
	c := sample()
	c.Query = "a very long query that will not fit in a narrow field"

	got := c.Search(20)
	if lipgloss.Width(got) > 20 {
		t.Errorf("Search(20) is %d cells: %q", lipgloss.Width(got), got)
	}
	if !strings.Contains(got, "field") {
		t.Errorf("Search(20) = %q, want the end of the query kept", got)
	}
	if !strings.Contains(got, "…") {
		t.Errorf("Search(20) = %q, want an ellipsis to show it was cut", got)
	}
}

// One cursor on the screen, and it belongs to whatever the keyboard is talking
// to: a caret in an unfocused field points at a place typing would not go.
func TestSearchDrawsACaretOnlyWhileFocused(t *testing.T) {
	c := sample()

	if got := c.Search(40); strings.Contains(got, theme.Caret) {
		t.Errorf("Search() = %q, want no caret while the list has the focus", got)
	}

	c.Focus = FocusSearch
	got := c.Search(40)
	if !strings.Contains(got, theme.Caret) {
		t.Errorf("Search() = %q, want a caret in the focused field", got)
	}
	// After the query: that is where the next character lands.
	if strings.Index(got, theme.Caret) < strings.Index(got, "geo") {
		t.Errorf("Search() = %q, want the caret after the query", got)
	}
}

// An empty field is typed into at the front, so that is where the caret goes,
// with the placeholder trailing it rather than swallowing it.
func TestSearchCaretLeadsThePlaceholder(t *testing.T) {
	c := sample()
	c.Focus = FocusSearch
	c.Query = ""

	got := c.Search(40)
	if !strings.Contains(got, "type to filter") {
		t.Errorf("Search() = %q, want the placeholder", got)
	}
	if strings.Index(got, theme.Caret) > strings.Index(got, "type to filter") {
		t.Errorf("Search() = %q, want the caret before the placeholder", got)
	}
}

// The caret costs a cell, and a field elided to the width has none to spare: an
// elision that does not account for it is a caret pushed off the end, so the
// focused field looks like the unfocused one exactly when it matters.
func TestSearchKeepsTheCaretWhenElided(t *testing.T) {
	c := sample()
	c.Focus = FocusSearch
	c.Query = "a very long query that will not fit in a narrow field"

	for _, width := range []int{20, 30, 40} {
		got := c.Search(width)
		if w := lipgloss.Width(got); w > width {
			t.Errorf("Search(%d) is %d cells: %q", width, w, got)
		}
		if !strings.Contains(got, theme.Caret) {
			t.Errorf("Search(%d) = %q, want the caret kept", width, got)
		}
	}
}

// The question in the footer is the same line editor as the search field, so it
// marks where you are typing the same way.
func TestAskDrawsTheSameCaret(t *testing.T) {
	c := sample()
	c.Focus = FocusPrompt
	c.Prompt = Prompt{Label: "title: ", Text: "site ABC-8714"}

	got := c.Ask(40)
	if !strings.Contains(got, theme.Caret) {
		t.Errorf("Ask() = %q, want a caret", got)
	}
	if strings.Index(got, theme.Caret) < strings.Index(got, "4178") {
		t.Errorf("Ask() = %q, want the caret after the answer", got)
	}
}

// Half a binding reads as a key that does something unnameable.
func TestFooterDropsWholeHintsThatDoNotFit(t *testing.T) {
	c := sample()

	full := c.Footer(80)
	for _, want := range []string{"open", "review", "quit"} {
		if !strings.Contains(full, want) {
			t.Errorf("Footer(80) = %q, want it to contain %q", full, want)
		}
	}

	narrow := c.Footer(12)
	if lipgloss.Width(narrow) > 12 {
		t.Errorf("Footer(12) is %d cells: %q", lipgloss.Width(narrow), narrow)
	}
	if strings.Contains(narrow, "revi") && !strings.Contains(narrow, "review") {
		t.Errorf("Footer(12) = %q, want no half-drawn hint", narrow)
	}
	// The first hint is the one to keep.
	if !strings.Contains(narrow, "open") {
		t.Errorf("Footer(12) = %q, want the first hint kept", narrow)
	}
}

func TestRuleSpansTheWidth(t *testing.T) {
	c := sample()
	if got := lipgloss.Width(c.Rule(37)); got != 37 {
		t.Errorf("Rule(37) is %d cells, want 37", got)
	}
	if got := c.Rule(0); got != "" {
		t.Errorf("Rule(0) = %q, want empty", got)
	}
}

// Truncation has to survive the escape sequences the styles put in, or a cut
// line leaks colour into the rest of the screen.
func TestFitTruncatesStyledText(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(restore)

	s := theme.Default()
	line := s.Selected.Render("selected") + s.Desc.Render(" and dimmed")
	got := fit(line, 5)

	if w := lipgloss.Width(got); w != 5 {
		t.Errorf("fit() is %d cells, want 5: %q", w, got)
	}
	if !strings.Contains(got, "selec") {
		t.Errorf("fit() = %q, want the visible text kept", got)
	}
	// A truncation that drops the reset leaves every later line coloured.
	if strings.Count(got, "\x1b[") > 0 && !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("fit() = %q, want it to end reset", got)
	}
}

// Colour is the theme's, so the selected chip has to actually differ from an
// unselected one once a profile is available to render it with.
func TestSelectionIsVisible(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(restore)

	c := sample()
	c.Groups = []Group{{Options: []Option{{Label: "same", Selected: true}, {Label: "same"}}}}

	got := c.Filters(200)
	parts := strings.SplitN(got, "same", 2)
	if len(parts) < 2 {
		t.Fatalf("Filters() = %q, want both chips", got)
	}
	if parts[0] == "" {
		t.Errorf("Filters() = %q, want the selected chip styled", got)
	}
	// Same label, so anything that distinguishes them is the styling.
	if strings.Count(got, "\x1b[") < 2 {
		t.Errorf("Filters() = %q, want the two chips styled differently", got)
	}
}

func TestRenderOnAnEmptyTerminal(t *testing.T) {
	c := sample()
	for _, size := range [][2]int{{0, 0}, {0, 24}, {80, 0}, {-5, -5}} {
		if got := c.Render(Compute(size[0], size[1], false), []string{"row"}, nil); got != "" {
			t.Errorf("Render(%dx%d) = %q, want empty", size[0], size[1], got)
		}
	}
}

// The bullet is what says a filter is set. Colour says it too, but a chip set in
// passing is otherwise invisible until a column goes missing from the screen.
func TestASetChipIsMarkedAndNotOnlyColoured(t *testing.T) {
	c := sample()
	c.Groups = []Group{{Options: []Option{
		{Label: "on", Selected: true},
		{Label: "off"},
	}}}

	got := c.Filters(200)
	if !strings.Contains(got, theme.Bullet+"on") {
		t.Errorf("Filters() = %q, want the set chip marked", got)
	}
	if strings.Contains(got, theme.Bullet+"off") {
		t.Errorf("Filters() = %q, want no mark on the chip that is not set", got)
	}

	// And it costs the bar no width, whichever chips are set or focused.
	c.Focus = FocusFilters
	wide := lipgloss.Width(c.Filters(200))
	for _, sel := range []bool{true, false} {
		for _, foc := range []bool{true, false} {
			c.Groups[0].Options[1] = Option{Label: "off", Selected: sel, Focused: foc}
			if w := lipgloss.Width(c.Filters(200)); w != wide {
				t.Errorf("selected=%v focused=%v made the bar %d cells, not %d",
					sel, foc, w, wide)
			}
		}
	}
}
