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
