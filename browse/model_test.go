package browse

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
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
	case "ctrl+q":
		return tea.KeyMsg{Type: tea.KeyCtrlQ}
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

	m.chrome.Query, m.chrome.Caret = "geo package ", len("geo package ")
	m, _ = press(m, "ctrl+w")
	if m.Query() != "geo " {
		t.Errorf("Query() = %q, want %q", m.Query(), "geo ")
	}
	m, _ = press(m, "ctrl+w")
	if m.Query() != "" {
		t.Errorf("Query() = %q, want it emptied", m.Query())
	}
}

// A cursor that only ever sits at the end is an append point wearing a cursor's
// clothes. Left and right are what make it one.
func TestArrowsMoveTheCaretInTheSearchField(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up") // search
	m.chrome.Query, m.chrome.Caret = "", 0
	m, _ = press(m, "g", "e", "o")

	m, _ = press(m, "left", "left")
	if m.chrome.Caret != 1 {
		t.Errorf("caret = %d after two lefts from the end of %q, want 1", m.chrome.Caret, "geo")
	}
	m, _ = press(m, "right")
	if m.chrome.Caret != 2 {
		t.Errorf("caret = %d after a right, want 2", m.chrome.Caret)
	}

	// Both ends stop rather than wrap: a cursor that reappears at the far end
	// of the text has lost the one thing it was saying.
	m, _ = press(m, "left", "left", "left", "left")
	if m.chrome.Caret != 0 {
		t.Errorf("caret = %d against the left end, want 0", m.chrome.Caret)
	}
	m, _ = press(m, "right", "right", "right", "right", "right")
	if m.chrome.Caret != 3 {
		t.Errorf("caret = %d against the right end of %q, want 3", m.chrome.Caret, m.Query())
	}
}

// Where the cursor is, is where the typing goes — and where a backspace bites.
func TestEditingHappensAtTheCaret(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up") // search
	m.chrome.Query, m.chrome.Caret = "", 0
	m, _ = press(m, "g", "e", "o")

	m, _ = press(m, "left", "x")
	if m.Query() != "gexo" {
		t.Errorf("Query() = %q, want %q: typing lands at the cursor", m.Query(), "gexo")
	}
	if m.chrome.Caret != 3 {
		t.Errorf("caret = %d after typing, want it after what was typed", m.chrome.Caret)
	}

	// Backspace takes the character before the cursor, not the one under it:
	// it deletes what you just typed, not what you just walked onto.
	m, _ = press(m, "backspace")
	if m.Query() != "geo" {
		t.Errorf("Query() = %q, want %q", m.Query(), "geo")
	}
	m, _ = press(m, "home", "backspace")
	if m.Query() != "geo" {
		t.Errorf("Query() = %q, want the query untouched at the left end", m.Query())
	}
	m, _ = press(m, "end", "backspace")
	if m.Query() != "ge" {
		t.Errorf("Query() = %q, want %q after end then backspace", m.Query(), "ge")
	}
}

// The same two keys walk the chips. The band decides which, and the other must
// not move underneath it.
func TestArrowsInTheFieldLeaveTheChipsAlone(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up", "up") // filters
	m, _ = press(m, "right")
	chip := m.option

	m, _ = press(m, "down") // search
	m.chrome.Query, m.chrome.Caret = "geo", 3
	m, _ = press(m, "left", "left")
	if m.option != chip {
		t.Errorf("chip cursor moved to %d while typing in the field, want %d", m.option, chip)
	}
	if m.chrome.Caret != 1 {
		t.Errorf("caret = %d, want 1", m.chrome.Caret)
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

func TestEnterOnTheSearchFieldDropsIntoTheListAndSubmits(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up") // search
	m.chrome.Query = "geo"
	m, cmd := press(m, "enter")
	if m.Focus() != FocusList {
		t.Errorf("Focus() = %v, want FocusList", m.Focus())
	}
	if cmd == nil {
		t.Fatal("Enter in the search field produced no command")
	}
	msg, ok := cmd().(SubmitMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want SubmitMsg: an app whose query costs something needs to know when", cmd())
	}
	if msg.Query != "geo" {
		t.Errorf("SubmitMsg.Query = %q, want %q", msg.Query, "geo")
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
func TestCtrlQAndCtrlCAskToQuit(t *testing.T) {
	for _, k := range []string{"ctrl+q", "ctrl+c"} {
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
	// Bracketed for the cursor, bulleted for being set: "opened" is both.
	if !strings.Contains(m.View(), "["+theme.Bullet+"opened]") {
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

func wheel(up bool) tea.MouseMsg {
	b := tea.MouseButtonWheelDown
	if up {
		b = tea.MouseButtonWheelUp
	}
	return tea.MouseMsg{Button: b, Action: tea.MouseActionPress}
}

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
}

// The wheel moves the view, not the selection. Tied to the cursor it lurches: the
// view only moves once the highlight is pushed off an edge.
func TestWheelMovesTheViewportNotTheCursor(t *testing.T) {
	m := model(100)
	// Down the list far enough that a notch of scrolling leaves it on screen.
	for i := 0; i < 10; i++ {
		m, _ = press(m, "down")
	}
	at := m.Cursor()

	m, _ = m.Update(wheel(false))
	if m.top != wheelRows {
		t.Errorf("top = %d, want %d after one notch", m.top, wheelRows)
	}
	if m.Cursor() != at {
		t.Errorf("Cursor() = %d, want it left at %d: the wheel moves the view", m.Cursor(), at)
	}

	m, _ = m.Update(wheel(true))
	if m.top != 0 {
		t.Errorf("top = %d, want back to 0", m.top)
	}
	if m.Cursor() != at {
		t.Errorf("Cursor() = %d, want it still at %d", m.Cursor(), at)
	}
}

// Scrolled far enough that the cursor would be off screen, it is dragged to the
// edge it left — but no further.
func TestWheelDragsTheCursorOnlyToTheEdge(t *testing.T) {
	m := model(100)
	m, _ = m.Update(wheel(false))
	if m.Cursor() != wheelRows {
		t.Errorf("Cursor() = %d, want %d: the top row of the new view", m.Cursor(), wheelRows)
	}
}

// Scrolled far enough, the cursor has to come along or it would be off screen.
func TestWheelDragsTheCursorOffTheEdge(t *testing.T) {
	m := model(100)
	rows := m.Layout().List.Height
	for i := 0; i < 20; i++ {
		m, _ = m.Update(wheel(false))
	}
	if m.Cursor() < m.top || m.Cursor() >= m.top+rows {
		t.Errorf("cursor %d outside the visible rows [%d, %d)", m.Cursor(), m.top, m.top+rows)
	}
}

func TestWheelStopsAtBothEnds(t *testing.T) {
	m := model(100)
	rows := m.Layout().List.Height

	for i := 0; i < 200; i++ {
		m, _ = m.Update(wheel(false))
	}
	if want := 100 - rows; m.top != want {
		t.Errorf("top = %d, want %d: never past the last screenful", m.top, want)
	}
	for i := 0; i < 200; i++ {
		m, _ = m.Update(wheel(true))
	}
	if m.top != 0 {
		t.Errorf("top = %d, want 0", m.top)
	}
}

func TestWheelOnAnEmptyListDoesNothing(t *testing.T) {
	m := model(0)
	m, _ = m.Update(wheel(false))
	if m.top != 0 || m.Cursor() != -1 {
		t.Errorf("top = %d, cursor = %d, want 0 and -1", m.top, m.Cursor())
	}
}

// Clicking a band is the same as arrowing to it.
func TestClickSelectsTheRowUnderThePointer(t *testing.T) {
	m := model(100)
	l := m.Layout()

	m, _ = m.Update(click(10, l.List.Y+4))
	if m.Focus() != FocusList {
		t.Errorf("Focus() = %v, want FocusList", m.Focus())
	}
	if m.Cursor() != 4 {
		t.Errorf("Cursor() = %d, want 4", m.Cursor())
	}

	// And after scrolling, the row under the pointer is the one it picks.
	m, _ = m.Update(wheel(false))
	m, _ = m.Update(click(10, l.List.Y+4))
	if m.Cursor() != wheelRows+4 {
		t.Errorf("Cursor() = %d, want %d", m.Cursor(), wheelRows+4)
	}
}

func TestClickOnTheSearchAndFilterBands(t *testing.T) {
	m := model(100)
	l := m.Layout()

	m, _ = m.Update(click(3, l.Search.Y))
	if m.Focus() != FocusSearch {
		t.Errorf("Focus() = %v, want FocusSearch", m.Focus())
	}
	m, _ = m.Update(click(3, l.Filters.Y))
	if m.Focus() != FocusFilters {
		t.Errorf("Focus() = %v, want FocusFilters", m.Focus())
	}
}

func TestClickPastTheLastRowChangesNothing(t *testing.T) {
	m := model(3) // three rows in a much taller band
	l := m.Layout()
	m, _ = m.Update(click(10, l.List.Y+8))
	if m.Cursor() != 0 {
		t.Errorf("Cursor() = %d, want it left at 0", m.Cursor())
	}
}

func TestMouseReleaseAndMotionAreIgnored(t *testing.T) {
	m := model(100)
	l := m.Layout()
	for _, msg := range []tea.MouseMsg{
		{X: 10, Y: l.List.Y + 4, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
		{X: 10, Y: l.List.Y + 4, Button: tea.MouseButtonNone, Action: tea.MouseActionMotion},
	} {
		m, _ = m.Update(msg)
		if m.Cursor() != 0 {
			t.Errorf("%v moved the cursor to %d", msg.Action, m.Cursor())
		}
	}
}

// Esc must not quit. It is one stray keypress away at all times, and an app needs
// it for backing out of things.
func TestEscDoesNotQuit(t *testing.T) {
	m := model(10)
	m, cmd := press(m, "esc")
	if cmd != nil {
		t.Fatalf("esc produced %T, want nothing", cmd())
	}
	if m.Focus() != FocusList || m.Cursor() != 0 {
		t.Errorf("esc changed the state: focus %v, cursor %d", m.Focus(), m.Cursor())
	}
}

// A question is asked of the row the cursor is on, so the answer has to be about
// that row: while one is open the arrows must not move the list underneath it.
func TestAskIsModal(t *testing.T) {
	m := model(10)
	m, _ = press(m, "down", "down")
	before := m.Cursor()

	m.Ask("title: ", "old title")
	if !m.Asking() || m.Focus() != FocusPrompt {
		t.Fatalf("Asking() = %v, Focus() = %v, want a prompt", m.Asking(), m.Focus())
	}

	m, _ = press(m, "down", "down", "up", "pgdown", "home", "end")
	if m.Cursor() != before {
		t.Errorf("the cursor moved to %d under an open question, want it to stay at %d",
			m.Cursor(), before)
	}
	// The query is what the sample chrome came with: typing into a question must
	// not reach the field the question is drawn over.
	if m.Query() != "geo" {
		t.Errorf("Query() = %q, want it untouched at %q", m.Query(), "geo")
	}
}

// The same editing keys as the search field, and ^U clears the answer rather than
// the query it is drawn over.
func TestAskEditsItsOwnText(t *testing.T) {
	m := model(10)
	m, _ = press(m, "down") // focus the list
	m.Ask("title: ", "site ABC-8714 ")
	m, _ = press(m, "s", "p", "l", "i", "t")

	_, cmd := press(m, "enter")
	msg, ok := cmd().(AnsweredMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want AnsweredMsg", cmd())
	}
	if want := "site ABC-8714 split"; msg.Text != want {
		t.Errorf("answer = %q, want %q", msg.Text, want)
	}
	if msg.Label != "title: " {
		t.Errorf("label = %q, want the question back with the answer", msg.Label)
	}

	m.Ask("title: ", "throw this away")
	m, _ = press(m, "ctrl+u")
	m, _ = press(m, "k", "e", "p", "t")
	_, cmd = press(m, "enter")
	if got := cmd().(AnsweredMsg).Text; got != "kept" {
		t.Errorf("after ^U the answer is %q, want %q", got, "kept")
	}

	// ^W drops the last word and leaves the space that separated it, which the
	// backspace then takes.
	m.Ask("title: ", "one two three")
	m, _ = press(m, "ctrl+w", "backspace")
	_, cmd = press(m, "enter")
	if got := cmd().(AnsweredMsg).Text; got != "one two" {
		t.Errorf("after ^W and backspace the answer is %q, want %q", got, "one two")
	}
}

// Esc is free for this precisely because quitting is not spent on it.
func TestEscAbandonsTheQuestionAndCtrlQStillQuits(t *testing.T) {
	m := model(10)
	m, _ = press(m, "up") // the search field, so the focus has somewhere to return to
	m.Ask("move to: ", "")

	after, cmd := press(m, "esc")
	msg, ok := cmd().(CancelledMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want CancelledMsg", cmd())
	}
	if msg.Label != "move to: " {
		t.Errorf("CancelledMsg.Label = %q", msg.Label)
	}
	if after.Asking() {
		t.Error("the question is still open after Esc")
	}
	if after.Focus() != FocusSearch {
		t.Errorf("Focus() = %v after answering, want the band that asked (FocusSearch)", after.Focus())
	}

	m.Ask("title: ", "")
	if _, cmd := press(m, "ctrl+q"); cmd == nil {
		t.Fatal("^Q under an open question produced no command")
	} else if _, ok := cmd().(QuitMsg); !ok {
		t.Errorf("cmd() = %T, want QuitMsg — a modal you cannot quit out of is a trap", cmd())
	}
}

// A question is drawn over the list, not into the footer: the keys go on saying
// what they do, and the list stays visible around the window because the question
// is about the row still highlighted underneath.
func TestTheQuestionIsAWindowOverTheList(t *testing.T) {
	m := model(10)
	l := m.Layout()

	m.Ask("title: ", "SV PR#853")
	view := m.View()
	lines := strings.Split(view, "\n")

	if footer := lines[l.Footer.Y]; footer != "" {
		t.Errorf("footer = %q, want it blank: ↵ is answering the question", footer)
	}
	if !strings.Contains(view, "SV PR#853") || !strings.Contains(view, "title") {
		t.Error("the question and the answer so far are not on the screen")
	}
	if !strings.Contains(view, "╭") {
		t.Error("the question is not in a window")
	}
	// The list is still there around it: a question is not a new screen.
	if !strings.Contains(view, "row0") {
		t.Error("the list is not drawn under the question")
	}
}

// One editor, both fields: the arrows walk the cursor through an answer exactly
// as they do through a query.
func TestTheAnswerHasAMovableCursor(t *testing.T) {
	m := model(10)
	m.Ask("title: ", "geo")
	if m.chrome.Prompt.Caret != 3 {
		t.Errorf("caret = %d on opening, want it after the text it was given", m.chrome.Prompt.Caret)
	}

	m, _ = press(m, "left", "x")
	if m.chrome.Prompt.Text != "gexo" {
		t.Errorf("Text = %q, want %q: typing lands at the cursor", m.chrome.Prompt.Text, "gexo")
	}
	m, _ = press(m, "home", "backspace")
	if m.chrome.Prompt.Text != "gexo" {
		t.Errorf("Text = %q, want it untouched at the left end", m.chrome.Prompt.Text)
	}
	m, _ = press(m, "end", "backspace")
	if m.chrome.Prompt.Text != "gex" {
		t.Errorf("Text = %q, want %q", m.chrome.Prompt.Text, "gex")
	}

	// And the list underneath does not move while it is open.
	before := m.Cursor()
	m, _ = press(m, "down", "down")
	if m.Cursor() != before {
		t.Errorf("cursor moved to %d with a question open, want %d", m.Cursor(), before)
	}
}

// A screen that changes mode has to be able to say so: a list of folders to move
// into is still a list, but "↵ resume" is then a lie about what Enter does.
func TestSetKeysRelabelsTheFooter(t *testing.T) {
	m := model(10)
	m.SetKeys([]Key{{"↵", "move here"}, {"␛", "cancel"}})

	footer := strings.Split(m.View(), "\n")[m.Layout().Footer.Y]
	if !strings.Contains(footer, "move here") || !strings.Contains(footer, "cancel") {
		t.Errorf("footer = %q, want the new hints", footer)
	}
	if strings.Contains(footer, "review") {
		t.Errorf("footer = %q, want the old hints gone", footer)
	}
}

// The preview has no cursor, so clicking a line of it is the only way to act on
// what it says. The band reports which line and leaves the meaning to the app.
func TestClickingThePreviewReportsTheLine(t *testing.T) {
	m := model(10)
	l := m.Layout()
	if l.Preview.Empty() {
		t.Fatal("no preview band to click")
	}

	for _, line := range []int{0, 2, l.Preview.Height - 1} {
		_, cmd := m.Update(tea.MouseMsg{
			Y:      l.Preview.Y + line,
			Button: tea.MouseButtonLeft,
			Action: tea.MouseActionPress,
		})
		if cmd == nil {
			t.Fatalf("clicking preview line %d produced no message", line)
		}
		got, ok := cmd().(PreviewClickMsg)
		if !ok {
			t.Fatalf("clicking preview line %d sent %T", line, cmd())
		}
		if got.Line != line {
			t.Errorf("PreviewClickMsg.Line = %d, want %d", got.Line, line)
		}
	}

	// The focus is not the point of the click, and must not move with it.
	before := m.Focus()
	after, _ := m.Update(tea.MouseMsg{Y: l.Preview.Y, Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress})
	if after.Focus() != before {
		t.Errorf("Focus() = %v, want it left at %v", after.Focus(), before)
	}
}

// A confirmation window is modal and answered by y or n, and nothing else — the
// questions worth a window are the ones a stray keypress must not answer.
func TestConfirmWindowAnswersOnlyToYesAndNo(t *testing.T) {
	m := model(10)
	m, _ = press(m, "down", "down")
	before := m.Cursor()
	m.AskConfirm("trash", Confirm{Question: "Send this session to the trash?", Yes: "remove it", No: "keep it"})

	if m.Focus() != FocusConfirm || m.Confirming() != "trash" {
		t.Fatalf("Focus = %v, Confirming = %q", m.Focus(), m.Confirming())
	}

	// The list must not move under a question about the row it was asked of.
	moved, _ := press(m, "down", "up", "pgdown", "home", "j", "x")
	if moved.Cursor() != before {
		t.Errorf("cursor moved to %d under an open window, want %d", moved.Cursor(), before)
	}
	if moved.Confirming() != "trash" {
		t.Error("a stray key closed the window")
	}

	_, cmd := press(m, "y")
	if msg, ok := cmd().(ConfirmedMsg); !ok || msg.Label != "trash" {
		t.Errorf("y gave %T %+v, want ConfirmedMsg{trash}", cmd(), cmd())
	}
	for _, key := range []string{"n", "esc", "enter"} {
		after, cmd := press(m, key)
		if cmd == nil {
			t.Fatalf("%s produced no command", key)
		}
		if msg, ok := cmd().(DismissedMsg); !ok || msg.Label != "trash" {
			t.Errorf("%s gave %T, want DismissedMsg", key, cmd())
		}
		if after.Confirming() != "" {
			t.Errorf("%s left the window open", key)
		}
		if after.Focus() != FocusList {
			t.Errorf("after %s the focus is %v, want the band that asked", key, after.Focus())
		}
	}
}

// ^Q still works: a modal you cannot quit out of is a trap.
func TestCtrlQClosesTheProgramFromAConfirmWindow(t *testing.T) {
	m := model(10)
	m.AskConfirm("live", Confirm{Question: "Resume a running session?"})
	_, cmd := press(m, "ctrl+q")
	if _, ok := cmd().(QuitMsg); !ok {
		t.Errorf("cmd() = %T, want QuitMsg", cmd())
	}
}

// The window is drawn over the list, centred, with the answers named as the keys
// that give them — and the rest of the screen still standing around it.
func TestConfirmWindowIsDrawnOverTheList(t *testing.T) {
	m := model(30)
	m.AskConfirm("live", Confirm{
		Question: "Resume a session that is already running?",
		Detail:   []string{"Two processes appending to one transcript mangles it."},
		Yes:      "resume it anyway",
		No:       "leave it alone",
	})

	l := m.Layout()
	lines := strings.Split(m.View(), "\n")
	body := strings.Join(lines[l.List.Y:l.List.Y+l.List.Height], "\n")

	for _, want := range []string{"already running", "mangles it", "resume it anyway", "leave it alone", "␛"} {
		if !strings.Contains(body, want) {
			t.Errorf("the window is missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, "╭") || !strings.Contains(body, "╯") {
		t.Errorf("no border drawn:\n%s", body)
	}
	// The chrome around it is untouched: a question is not a new screen. The
	// footer is the exception — its hints are the list's, and none of them apply
	// while the window owns the keyboard, so the row goes blank rather than
	// claiming ↵ still opens a row.
	if !strings.Contains(lines[l.Header.Y], "belp") {
		t.Error("the header went missing")
	}
	if got := lines[l.Footer.Y]; got != "" {
		t.Errorf("footer = %q, want it blank while the window is open", got)
	}
	// And rows are still visible above or below it.
	if !strings.Contains(body, "row") {
		t.Error("the list was replaced rather than drawn over")
	}
	for _, line := range lines {
		if lipgloss.Width(line) > l.Width {
			t.Fatalf("line wider than the screen: %q", line)
		}
	}
}
