package browse

import (
	"regexp"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
