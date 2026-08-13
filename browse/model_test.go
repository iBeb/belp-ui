package browse

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func model(rows int) Model {
	m := New(sample())
	m.Row = func(i, _ int, selected bool) string {
		if selected {
			return fmt.Sprintf("▸row%d", i)
		}
		return fmt.Sprintf(" row%d", i)
	}
	m.Preview = func(i, _, _ int) []string {
		return []string{fmt.Sprintf("preview of row%d", i)}
	}
	m.SetSize(100, 30)
	m.SetRowCount(rows)
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+w":
		return tea.KeyMsg{Type: tea.KeyCtrlW}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// press feeds keys in order and returns the model and the last command.
func press(m Model, keys ...string) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		m, cmd = m.Update(key(k))
	}
	return m, cmd
}

// The list has the focus to begin with. A list you have to click into before the
// arrows work is a list that feels broken.
func TestFocusStartsOnTheList(t *testing.T) {
	m := model(10)
	if m.Focus() != FocusList {
		t.Errorf("Focus() = %v, want FocusList", m.Focus())
	}
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want 0", m.Cursor())
	}
}

// One selection model for the whole screen: the search field is the row above
// the first row, and the filter bar the row above that.
func TestUpAndDownWalkOutOfTheListAndBack(t *testing.T) {
	m := model(10)

	m, _ = press(m, "down", "down") // rows 1, 2
	if m.Cursor() != 2 {
		t.Fatalf("Cursor() = %d, want 2", m.Cursor())
	}

	m, _ = press(m, "up", "up") // back to row 0
	if m.Cursor() != 0 || m.Focus() != FocusList {
		t.Fatalf("Cursor() = %d, Focus() = %v, want 0 and the list", m.Cursor(), m.Focus())
	}

	m, _ = press(m, "up") // out of the list, into the search field
	if m.Focus() != FocusSearch {
		t.Fatalf("Focus() = %v, want FocusSearch", m.Focus())
	}
	m, _ = press(m, "up") // and up again onto the filter bar
	if m.Focus() != FocusFilters {
		t.Fatalf("Focus() = %v, want FocusFilters", m.Focus())
	}
	m, _ = press(m, "up") // already at the top
	if m.Focus() != FocusFilters {
		t.Fatalf("Focus() = %v, want to stay on the filters", m.Focus())
	}

	m, _ = press(m, "down", "down")
	if m.Focus() != FocusList {
		t.Fatalf("Focus() = %v, want to come back to the list", m.Focus())
	}
	// Coming back lands where it left, not at the top.
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want the row it left", m.Cursor())
	}
}

func TestCursorStopsAtBothEnds(t *testing.T) {
	m := model(3)
	m, _ = press(m, "down", "down", "down", "down")
	if m.Cursor() != 2 {
		t.Errorf("Cursor() = %d, want the last row 2", m.Cursor())
	}
}

// Typing only reaches the query while the search field has the focus, or every
// letter would be both a search term and a shortcut.
func TestTypingOnlyReachesTheQueryWhenSearchHasFocus(t *testing.T) {
	m := model(10)
	m.chrome.Query = "" // start empty, or "geo" typed over "geo" proves nothing

	for _, focus := range []Focus{FocusList, FocusFilters} {
		m.chrome.Focus = focus
		m, _ = press(m, "g", "e", "o")
		if m.Query() != "" {
			t.Errorf("focus %v: Query() = %q, want typing ignored", focus, m.Query())
		}
	}

	m.chrome.Focus = FocusSearch
	m, _ = press(m, "g", "e", "o")
	if m.Query() != "geo" {
		t.Errorf("Query() = %q, want %q", m.Query(), "geo")
	}
}

func TestBackspaceAndWordDeleteEditTheQuery(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up") // search
	m.chrome.Query = ""
	m, _ = press(m, "g", "e", "o", "backspace")
	if m.Query() != "ge" {
		t.Errorf("Query() = %q, want %q", m.Query(), "ge")
	}

	m.chrome.Query = "geo package "
	m, _ = press(m, "ctrl+w")
	if m.Query() != "geo " {
		t.Errorf("Query() = %q, want %q", m.Query(), "geo ")
	}
	m, _ = press(m, "ctrl+w")
	if m.Query() != "" {
		t.Errorf("Query() = %q, want it emptied", m.Query())
	}
}

// ^U is in the footer, and a footer key that only works in one band looks broken.
func TestClearWorksFromAnyBand(t *testing.T) {
	for _, focus := range []Focus{FocusList, FocusSearch, FocusFilters} {
		m := model(10)
		m.chrome.Focus = focus
		m.chrome.Query = "something"
		m, _ = press(m, "ctrl+u")
		if m.Query() != "" {
			t.Errorf("focus %v: Query() = %q, want it cleared", focus, m.Query())
		}
	}
}

func TestBackspaceIsIgnoredOutsideTheSearchField(t *testing.T) {
	m := model(10)
	before := m.Query()
	m, _ = press(m, "backspace")
	if m.Query() != before {
		t.Errorf("Query() = %q, want %q untouched", m.Query(), before)
	}
}

func TestEnterOnTheSearchFieldDropsIntoTheList(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up") // search
	m, cmd := press(m, "enter")
	if m.Focus() != FocusList {
		t.Errorf("Focus() = %v, want FocusList", m.Focus())
	}
	if cmd != nil {
		t.Error("Enter in the search field should not activate a row")
	}
}

func TestEnterOnARowActivatesIt(t *testing.T) {
	m := model(10)
	m, _ = press(m, "down", "down")
	_, cmd := press(m, "enter")
	if cmd == nil {
		t.Fatal("Enter on a row produced no command")
	}
	msg, ok := cmd().(ActivateMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want ActivateMsg", cmd())
	}
	if msg.Index != 2 {
		t.Errorf("ActivateMsg.Index = %d, want 2", msg.Index)
	}
}

func TestEnterOnAnEmptyListDoesNothing(t *testing.T) {
	m := model(0)
	if m.Cursor() != -1 {
		t.Errorf("Cursor() = %d, want -1 with no rows", m.Cursor())
	}
	_, cmd := press(m, "enter")
	if cmd != nil {
		t.Error("Enter on an empty list produced a command")
	}
}

// Quitting is the app's decision; the component only reports the key.
func TestEscAndCtrlCAskToQuit(t *testing.T) {
	for _, k := range []string{"esc", "ctrl+c"} {
		_, cmd := press(model(10), k)
		if cmd == nil {
			t.Fatalf("%s produced no command", k)
		}
		if _, ok := cmd().(QuitMsg); !ok {
			t.Errorf("%s gave %T, want QuitMsg", k, cmd())
		}
	}
}

// The viewport follows the cursor by as little as it can, so a held arrow key
// scrolls the list rather than jumping a page at a time.
func TestViewportFollowsTheCursorOneRowAtATime(t *testing.T) {
	m := model(100)
	rows := m.Layout().List.Height

	for i := 0; i < rows-1; i++ {
		m, _ = press(m, "down")
	}
	if m.top != 0 {
		t.Errorf("top = %d, want 0 while the cursor is still on screen", m.top)
	}
	m, _ = press(m, "down") // one past the bottom
	if m.top != 1 {
		t.Errorf("top = %d, want 1 after scrolling by one", m.top)
	}

	// And back up the other way.
	for i := 0; i < rows; i++ {
		m, _ = press(m, "up")
	}
	if m.top != 0 {
		t.Errorf("top = %d, want 0 back at the start", m.top)
	}
}

func TestViewportNeverLeavesBlankRowsBelowTheList(t *testing.T) {
	m := model(100)
	m, _ = press(m, "end")
	rows := m.Layout().List.Height
	if want := 100 - rows; m.top != want {
		t.Errorf("top = %d, want %d so the last screenful is full", m.top, want)
	}
	if m.Cursor() != 99 {
		t.Errorf("Cursor() = %d, want the last row", m.Cursor())
	}
}

func TestPageAndHomeEndMoveTheCursor(t *testing.T) {
	m := model(100)
	rows := m.Layout().List.Height

	m, _ = press(m, "pgdown")
	if m.Cursor() != rows {
		t.Errorf("Cursor() = %d after pgdown, want %d", m.Cursor(), rows)
	}
	m, _ = press(m, "pgup")
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d after pgup, want 0", m.Cursor())
	}
	m, _ = press(m, "end")
	if m.Cursor() != 99 {
		t.Errorf("Cursor() = %d after end, want 99", m.Cursor())
	}
	m, _ = press(m, "home")
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d after home, want 0", m.Cursor())
	}
}

// Re-filtering while pointing at something is normal; jumping to the top on
// every keystroke is what makes a search field feel like it is fighting you.
func TestSetRowCountKeepsTheCursorWhereItCan(t *testing.T) {
	m := model(100)
	m, _ = press(m, "pgdown")
	at := m.Cursor()

	m.SetRowCount(100)
	if m.Cursor() != at {
		t.Errorf("Cursor() = %d, want it left at %d", m.Cursor(), at)
	}

	m.SetRowCount(5)
	if m.Cursor() != 4 {
		t.Errorf("Cursor() = %d, want the last of 5 rows", m.Cursor())
	}
	m.SetRowCount(0)
	if m.Cursor() != -1 {
		t.Errorf("Cursor() = %d, want -1 with no rows", m.Cursor())
	}
	if m.top != 0 {
		t.Errorf("top = %d, want 0 with no rows", m.top)
	}
}

// Rows are drawn lazily, so a huge list costs no more than a screenful.
func TestOnlyVisibleRowsAreRendered(t *testing.T) {
	m := model(10000)
	drawn := map[int]bool{}
	m.Row = func(i, _ int, _ bool) string {
		drawn[i] = true
		return "row"
	}
	m.View()

	rows := m.Layout().List.Height
	if len(drawn) != rows {
		t.Errorf("rendered %d row(s), want the %d on screen", len(drawn), rows)
	}
	if drawn[rows+1] {
		t.Error("a row past the bottom of the screen was rendered")
	}
}

// The preview is only ever asked for the row under the cursor, which is what
// makes it safe for an app to fetch something there.
func TestPreviewIsAskedOnlyForTheCursorRow(t *testing.T) {
	m := model(50)
	m, _ = press(m, "down", "down")

	var asked []int
	m.Preview = func(i, _, _ int) []string {
		asked = append(asked, i)
		return []string{"detail"}
	}
	m.View()

	if len(asked) != 1 || asked[0] != 2 {
		t.Errorf("preview asked for %v, want only row 2", asked)
	}
}

func TestNoPreviewOnAnEmptyList(t *testing.T) {
	m := model(0)
	m.Preview = func(int, int, int) []string { return []string{"SHOULD NOT APPEAR"} }
	if strings.Contains(m.View(), "SHOULD NOT APPEAR") {
		t.Error("the preview was drawn although there are no rows")
	}
}

// A highlighted row under a focused search field claims a cursor that is not
// there.
func TestNoRowLooksSelectedWhileAnotherBandHasFocus(t *testing.T) {
	m := model(10)
	if !strings.Contains(m.View(), "▸row0") {
		t.Error("the cursor row is not marked while the list has focus")
	}
	m, _ = press(m, "up") // search
	if strings.Contains(m.View(), "▸") {
		t.Error("a row still looks selected while the search field has focus")
	}
}

// The bar is one row of chips: ← and → cross group boundaries rather than
// stopping at them.
func TestChipCursorCrossesGroups(t *testing.T) {
	m := model(10)
	m.chrome.Focus = FocusFilters

	// sample(): three chips, then two, then one.
	want := []chipAt{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {2, 0}}
	for i, w := range want {
		if m.group != w.group || m.option != w.option {
			t.Fatalf("step %d: chip at (%d,%d), want (%d,%d)", i, m.group, m.option, w.group, w.option)
		}
		m, _ = press(m, "right")
	}
	// Stops at the end rather than wrapping: wrapping in a bar you cannot see
	// the ends of loses you your place.
	last := want[len(want)-1]
	if m.group != last.group || m.option != last.option {
		t.Errorf("chip at (%d,%d), want to stop at (%d,%d)", m.group, m.option, last.group, last.option)
	}

	for i := 0; i < 10; i++ {
		m, _ = press(m, "left")
	}
	if m.group != 0 || m.option != 0 {
		t.Errorf("chip at (%d,%d), want to stop at the first", m.group, m.option)
	}
}

func TestChipCursorIsShownOnlyOnTheFilterBar(t *testing.T) {
	m := model(10)
	if strings.Contains(m.View(), "[") {
		t.Error("a chip is bracketed although the list has the focus")
	}
	m.chrome.Focus = FocusFilters
	if !strings.Contains(m.View(), "[opened]") {
		t.Errorf("View() has no bracketed chip:\n%s", m.View())
	}
}

func TestArrowsAlongTheBarDoNotMoveTheListCursor(t *testing.T) {
	m := model(10)
	m, _ = press(m, "down", "down")
	at := m.Cursor()
	m.chrome.Focus = FocusFilters
	m, _ = press(m, "right", "right", "left")
	if m.Cursor() != at {
		t.Errorf("Cursor() = %d, want it left at %d", m.Cursor(), at)
	}
}

func TestWindowSizeMessageResizes(t *testing.T) {
	m := model(100)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	l := m.Layout()
	if l.Width != 60 || l.Height != 12 {
		t.Errorf("Layout() = %dx%d, want 60x12", l.Width, l.Height)
	}
	if lines := strings.Split(m.View(), "\n"); len(lines) != 12 {
		t.Errorf("View() has %d line(s), want 12", len(lines))
	}
}

// Shrinking a window must not leave the cursor off screen.
func TestResizeKeepsTheCursorVisible(t *testing.T) {
	m := model(100)
	m, _ = press(m, "end")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})

	rows := m.Layout().List.Height
	if m.Cursor() < m.top || m.Cursor() >= m.top+rows {
		t.Errorf("cursor %d is outside the visible rows [%d, %d)", m.Cursor(), m.top, m.top+rows)
	}
}

func TestModelWithNoRowFuncDrawsTheFrameAnyway(t *testing.T) {
	m := New(sample())
	m.SetSize(80, 24)
	m.SetRowCount(5)
	got := m.View()
	if !strings.Contains(got, "belp") {
		t.Errorf("View() = %q, want the chrome drawn without a RowFunc", got)
	}
	if lines := strings.Split(got, "\n"); len(lines) != 24 {
		t.Errorf("View() has %d line(s), want 24", len(lines))
	}
}

func TestSpaceAndEnterToggleTheChipUnderTheCursor(t *testing.T) {
	for _, k := range []string{" ", "enter"} {
		m := model(10)
		m.chrome.Focus = FocusFilters
		m, _ = press(m, "right") // "reviewed", unselected in sample()

		if got := m.SelectedIn(0); len(got) != 1 || got[0] != "opened" {
			t.Fatalf("%s: SelectedIn(0) = %v, want [opened] to begin with", k, got)
		}
		m, cmd := press(m, k)
		if cmd == nil {
			t.Fatalf("%s produced no command", k)
		}
		if _, ok := cmd().(FiltersChangedMsg); !ok {
			t.Errorf("%s gave %T, want FiltersChangedMsg", k, cmd())
		}
		if got := m.SelectedIn(0); len(got) != 2 || got[1] != "reviewed" {
			t.Errorf("%s: SelectedIn(0) = %v, want opened and reviewed", k, got)
		}
		// And off again.
		m, _ = press(m, k)
		if got := m.SelectedIn(0); len(got) != 1 || got[0] != "opened" {
			t.Errorf("%s: SelectedIn(0) = %v, want reviewed toggled back off", k, got)
		}
	}
}

// A date range is one window, not several.
func TestExclusiveGroupHoldsOneAnswer(t *testing.T) {
	m := model(10)
	m.chrome.Groups = []Group{{Exclusive: true, Options: []Option{
		{Label: "7 days"}, {Label: "30 days", Selected: true}, {Label: "all"},
	}}}
	m.chrome.Focus = FocusFilters

	m, _ = press(m, "right", "right", " ") // onto "all"
	if got := m.SelectedIn(0); len(got) != 1 || got[0] != "all" {
		t.Errorf("SelectedIn(0) = %v, want only [all]", got)
	}
	// Choosing the one already chosen leaves it chosen: an exclusive group with
	// no answer is not a state worth having.
	m, _ = press(m, " ")
	if got := m.SelectedIn(0); len(got) != 1 || got[0] != "all" {
		t.Errorf("SelectedIn(0) = %v, want it still [all]", got)
	}
}

// A Model is copied by value, so a toggle must not reach through a shared
// backing array into a copy somebody else is still holding.
func TestTogglingDoesNotMutateOtherCopies(t *testing.T) {
	before := model(10)
	before.chrome.Focus = FocusFilters
	was := before.SelectedIn(0)

	after, _ := press(before, " ") // toggle "opened" off

	if got := before.SelectedIn(0); len(got) != len(was) {
		t.Errorf("the earlier copy changed: SelectedIn(0) = %v, want %v", got, was)
	}
	if len(after.SelectedIn(0)) == len(was) {
		t.Error("the toggle had no effect on the new copy")
	}
}

// Space in the search field is a space, not a toggle.
func TestSpaceTypesWhileSearching(t *testing.T) {
	m := model(10)
	m.chrome.Focus = FocusSearch
	m.chrome.Query = "geo"
	m, cmd := press(m, " ")
	if m.Query() != "geo " {
		t.Errorf("Query() = %q, want %q", m.Query(), "geo ")
	}
	if cmd != nil {
		t.Errorf("cmd = %T, want none: nothing was filtered", cmd())
	}
}

func TestGroupsIsACopy(t *testing.T) {
	m := model(10)
	got := m.Groups()
	got[0].Options[0].Selected = !got[0].Options[0].Selected
	if m.SelectedIn(0)[0] != "opened" {
		t.Error("writing to the returned groups changed the model")
	}
}
