package browse

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var codesRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Backspace takes the character behind the cursor, Delete the one under it.
// Only backspace was bound, so the key marked Delete on a full keyboard — and
// fn+Delete on this one — did nothing at all in the field.
func TestDeleteTakesTheCharacterUnderTheCursor(t *testing.T) {
	for _, tc := range []struct {
		name      string
		text      string
		caret     int
		wantText  string
		wantCaret int
	}{
		{"mid-word", "banana", 2, "baana", 2},
		{"first character", "banana", 0, "anana", 0},
		{"last character", "banana", 5, "banan", 5},
		{"at the end there is nothing under it", "banana", 6, "banana", 6},
		{"empty", "", 0, "", 0},
	} {
		gotText, gotCaret := deleteAt(tc.text, tc.caret)
		if gotText != tc.wantText || gotCaret != tc.wantCaret {
			t.Errorf("%s: deleteAt(%q, %d) = (%q, %d), want (%q, %d)",
				tc.name, tc.text, tc.caret, gotText, gotCaret, tc.wantText, tc.wantCaret)
		}
	}
}

// And it is bound, in the field and in a window's answer: the helper existing
// is not the same as the key reaching it.
func TestDeleteIsBoundInBothFields(t *testing.T) {
	m := launched(5)
	m, _ = press(m, "ctrl+u")
	for _, r := range []string{"a", "b", "c"} {
		m, _ = press(m, r)
	}
	m, _ = press(m, "left", "left") // between a and b
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if got := m.Query(); got != "ac" {
		t.Errorf("query = %q, want %q — Delete did not reach the search field", got, "ac")
	}

	p := launched(5)
	p.chrome.Focus, p.chrome.Prompt = FocusPrompt, Prompt{Label: "rename", Text: "abc", Caret: 1}
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if got := p.chrome.Prompt.Text; got != "ac" {
		t.Errorf("answer = %q, want %q — Delete did not reach the window's field", got, "ac")
	}
}

// A click in the field puts the cursor where it was clicked.
//
// Checked against the drawn line rather than against the arithmetic: the column
// a character is on is a fact about the rendering, and a mapping that agrees
// only with itself is how a caret lands one cell off.
func TestClickingTheFieldPutsTheCursorThere(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	const width = 80
	m := launched(5)
	m.SetSize(width, 30)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("cherry", "") {
		m, _ = press(m, r)
	}

	// In cells, not bytes: the border and the magnifier are three bytes each,
	// so a byte offset into the line is four columns adrift of where the
	// character is drawn.
	line := []rune(plainLine(m, m.Layout().Search.Y+1))
	at := indexRunes(line, "cherry")
	if at < 0 {
		t.Fatalf("the query is not on the line: %q", string(line))
	}
	for i, r := range "cherry" {
		col := at + i // where that character is actually drawn
		clicked := m.chrome.CaretAt(col, width)
		if clicked != i {
			t.Errorf("a click on %q at column %d gives caret %d, want %d\n  %q",
				string(r), col, clicked, i, string(line))
		}
	}
}

// The same, with the query longer than the field: the text scrolls and an
// ellipsis stands where the hidden characters would be, so the column a
// character sits on is no longer its index.
func TestClickingAScrolledFieldPutsTheCursorThere(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	const width = 30 // narrow enough that the query cannot fit
	m := launched(5)
	m.SetSize(width, 30)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("abcdefghijklmnopqrstuvwxyz", "") {
		m, _ = press(m, r)
	}

	line := []rune(plainLine(m, m.Layout().Search.Y+1))
	if indexRunes(line, ellipsis) < 0 {
		t.Fatalf("the field is not scrolled at width %d: %q", width, string(line))
	}
	// Every drawn letter, and only those: the border, the magnifier, the
	// ellipsis and the caret block are not characters of the query.
	query := []rune(m.Query())
	checked := 0
	for col := 0; col < len(line); col++ {
		r := line[col]
		if r < 'a' || r > 'z' {
			continue
		}
		at := m.chrome.CaretAt(col, width)
		if at < 0 || at > len(query) {
			t.Fatalf("caret %d is outside the query", at)
		}
		if at >= len(query) || query[at] != r {
			t.Errorf("clicking column %d, which draws %q, gives caret %d\n  %q",
				col, string(r), at, string(line))
			continue
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("no drawn character was checked: %q", string(line))
	}
}

// indexRunes is strings.Index in cells rather than bytes.
func indexRunes(line []rune, want string) int {
	w := []rune(want)
	for i := 0; i+len(w) <= len(line); i++ {
		if string(line[i:i+len(w)]) == string(w) {
			return i
		}
	}
	return -1
}

// plainLine is one line of the rendered screen with the colour taken out.
func plainLine(m Model, y int) string {
	lines := strings.Split(m.View(), "\n")
	if y < 0 || y >= len(lines) {
		return ""
	}
	return codesRE.ReplaceAllString(lines[y], "")
}

// The click has to be wired to the mapping, not merely agree with it.
//
// CaretAt being right is half of it: until the mouse handler calls it, clicking
// the field takes the focus and leaves the cursor wherever it was, which is the
// behaviour being fixed.
func TestClickingTheFieldMovesTheCaret(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	const width = 80
	m := launched(5)
	m.SetSize(width, 30)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("cherry", "") {
		m, _ = press(m, r)
	}
	m, _ = press(m, "down") // into the list, so the click has to take the focus back

	line := []rune(plainLine(m, m.Layout().Search.Y+1))
	at := indexRunes(line, "cherry")
	if at < 0 {
		t.Fatalf("the query is not on the line: %q", string(line))
	}

	// On the "e", which is index 2.
	m, _ = m.Update(click(at+2, m.Layout().Search.Y+1))
	if m.Focus() != FocusSearch {
		t.Errorf("Focus() = %v, want the click to take the field", m.Focus())
	}
	if m.chrome.Caret != 2 {
		t.Errorf("Caret = %d, want 2 — the click did not move the cursor", m.chrome.Caret)
	}

	// Typing goes in where it was clicked, which is the point of moving it.
	m, _ = press(m, "X")
	if got := m.Query(); got != "chXerry" {
		t.Errorf("query = %q, want %q", got, "chXerry")
	}
}

// A click on the border takes the focus without moving the cursor: there is no
// character under it to point at.
func TestClickingTheBorderLeavesTheCaretAlone(t *testing.T) {
	const width = 80
	m := launched(5)
	m.SetSize(width, 30)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("cherry", "") {
		m, _ = press(m, r)
	}
	before := m.chrome.Caret
	m, _ = press(m, "down")

	m, _ = m.Update(click(4, m.Layout().Search.Y)) // the top border
	if m.Focus() != FocusSearch {
		t.Errorf("Focus() = %v, want the click to take the field", m.Focus())
	}
	if m.chrome.Caret != before {
		t.Errorf("Caret = %d, want it left at %d", m.chrome.Caret, before)
	}
}

// A word skip lands at the edge of a word, crossing whatever gap it starts in.
//
// Punctuation separates as much as a space does, so a skip through a path or a
// branch name stops at its parts. Treating them as one word makes the key
// useless on exactly the text these apps are full of.
func TestWordSkipStopsAtTheEdgesOfWords(t *testing.T) {
	for _, tc := range []struct {
		text        string
		caret       int
		left, right int
	}{
		{"one two three", 13, 8, 13},       // from the end, back over "three"
		{"one two three", 8, 4, 13},        // at "three", back to the start of "two"
		{"one two three", 10, 8, 13},       // inside a word, back to its own start
		{"one two three", 0, 0, 3},         // at the start, forward over "one"
		{"one two three", 4, 0, 7},         // from a word start, over the space behind it
		{"feat/add-the-thing", 0, 0, 4},    // punctuation is a boundary
		{"feat/add-the-thing", 18, 13, 18}, // back over "thing", not the whole branch
		{"  ", 2, 0, 2},                    // nothing but gap
		{"", 0, 0, 0},
	} {
		f := field{Text: tc.text, Caret: tc.caret}
		if got := f.wordLeft(); got != tc.left {
			t.Errorf("wordLeft(%q, %d) = %d, want %d", tc.text, tc.caret, got, tc.left)
		}
		if got := f.wordRight(); got != tc.right {
			t.Errorf("wordRight(%q, %d) = %d, want %d", tc.text, tc.caret, got, tc.right)
		}
	}
}

// And the keys that ask for it reach both fields. Terminal.app sends ⌥← as a
// meta escape and ⌃← as a CSI sequence, and which one a keyboard produces is
// not something an app gets to choose, so both are bound.
func TestWordKeysAreBoundInBothFields(t *testing.T) {
	for _, k := range []string{"alt+left", "ctrl+left", "alt+b"} {
		m := launched(5)
		m, _ = press(m, "ctrl+u")
		for _, r := range strings.Split("one two", "") {
			m, _ = press(m, r)
		}
		m, _ = press(m, k, "X")
		if got := m.Query(); got != "one Xtwo" {
			t.Errorf("%s in the search field gave %q, want %q", k, got, "one Xtwo")
		}
	}

	p := launched(5)
	p.chrome.Focus = FocusPrompt
	p.chrome.Prompt = Prompt{Label: "rename", Text: "one two", Caret: 7}
	p, _ = press(p, "alt+right") // already at the end: nowhere to go
	p, _ = press(p, "alt+left", "X")
	if got := p.chrome.Prompt.Text; got != "one Xtwo" {
		t.Errorf("the window's answer is %q, want %q", got, "one Xtwo")
	}
}

// Deleting by word, forwards and back.
func TestDeletingByWord(t *testing.T) {
	m := launched(5)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("one two three", "") {
		m, _ = press(m, r)
	}
	m, _ = press(m, "alt+backspace")
	if got := m.Query(); got != "one two " {
		t.Errorf("after alt+backspace the query is %q, want %q", got, "one two ")
	}
	m, _ = press(m, "home", "alt+delete")
	if got := m.Query(); got != " two " {
		t.Errorf("after alt+delete the query is %q, want %q", got, " two ")
	}
}

// And it crosses what a word skip crosses, rather than running to the nearest
// space: the key that moves and the key that deletes have to agree, or the
// second undoes more than the first said it would.
func TestDeletingByWordStopsWhereASkipDoes(t *testing.T) {
	m := launched(5)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("feat/add-thing", "") {
		m, _ = press(m, r)
	}
	m, _ = press(m, "alt+backspace")
	if got := m.Query(); got != "feat/add-" {
		t.Errorf("alt+backspace left %q, want %q", got, "feat/add-")
	}

	// ^W is the shell's key and keeps the shell's word, which ends at a space.
	m, _ = press(m, "ctrl+w")
	if got := m.Query(); got != "" {
		t.Errorf("^W left %q, want the whole word gone", got)
	}
}

// Shift and an arrow select, and typing replaces what was selected.
func TestSelectingAndTypingOverIt(t *testing.T) {
	m := launched(5)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("one two", "") {
		m, _ = press(m, r)
	}

	m, _ = press(m, "home", "alt+shift+right") // "one"
	if lo, hi, ok := m.chrome.field().sel(); !ok || lo != 0 || hi != 3 {
		t.Fatalf("selection = (%d, %d, %v), want (0, 3, true)", lo, hi, ok)
	}
	m, _ = press(m, "X")
	if got := m.Query(); got != "X two" {
		t.Errorf("typing over the selection gave %q, want %q", got, "X two")
	}
	if _, _, ok := m.chrome.field().sel(); ok {
		t.Errorf("the selection survived being typed over")
	}
}

// A key that deletes takes the whole selection, rather than one character out
// of the middle of it and the selection quietly with it.
func TestDeleteTakesTheWholeSelection(t *testing.T) {
	for _, k := range []string{"backspace", "delete", "ctrl+w"} {
		m := launched(5)
		m, _ = press(m, "ctrl+u")
		for _, r := range strings.Split("one two", "") {
			m, _ = press(m, r)
		}
		m, _ = press(m, "shift+home", k)
		if got := m.Query(); got != "" {
			t.Errorf("%s with everything selected left %q", k, got)
		}
	}
}

// An arrow with no shift is the way out of a selection, and leaves the cursor
// where the arrow put it rather than where the selection ended.
func TestAnArrowDropsTheSelection(t *testing.T) {
	m := launched(5)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("one", "") {
		m, _ = press(m, r)
	}
	m, _ = press(m, "shift+home", "right")
	if _, _, ok := m.chrome.field().sel(); ok {
		t.Errorf("the selection survived a plain arrow")
	}
	m, _ = press(m, "X")
	if got := m.Query(); got != "oXne" {
		t.Errorf("query = %q, want %q — the arrow moved to the wrong place", got, "oXne")
	}
}

// A Chrome or a Prompt built as a literal has nothing selected.
//
// The selection was once an anchor with -1 for "none", which made the zero
// value a selection of the first character: every app that built a Prompt
// without knowing about anchors had its first letter silently eaten by the
// next keystroke. The bool is what prevents that, and this is the test that
// says so.
func TestALiteralFieldHasNothingSelected(t *testing.T) {
	if _, _, ok := (Prompt{Text: "abc", Caret: 1}).field().sel(); ok {
		t.Errorf("a Prompt built without a selection has one")
	}
	if _, _, ok := (Chrome{Query: "abc", Caret: 1}).field().sel(); ok {
		t.Errorf("a Chrome built without a selection has one")
	}
}

// The selection is drawn, and drawn differently from both the plain text and
// the caret. A selection that Delete acts on but the screen does not show is
// worse than no selection at all.
func TestTheSelectionIsDrawn(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	m := launched(5)
	m.SetSize(80, 30)
	m, _ = press(m, "ctrl+u")
	for _, r := range strings.Split("abcd", "") {
		m, _ = press(m, r)
	}
	m, _ = press(m, "home", "shift+right", "shift+right") // "ab", cursor on "c"

	line := strings.Split(m.View(), "\n")[m.Layout().Search.Y+1]
	s := m.chrome.Styles
	for _, tc := range []struct {
		what string
		want string
	}{
		{"the selected characters", s.Selection.Render("a") + s.Selection.Render("b")},
		{"the character under the caret", caretCell(s, 'c')},
		{"the text beyond the selection", s.Value.Render("d")},
	} {
		if !strings.Contains(line, tc.want) {
			t.Errorf("%s are not drawn as expected\n  line %q\n  want %q", tc.what, line, tc.want)
		}
	}
}
