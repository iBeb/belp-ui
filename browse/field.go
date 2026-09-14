package browse

import (
	tea "github.com/charmbracelet/bubbletea"
)

// field is one line of editable text: what it holds and where the cursor is.
//
// One type for both fields that take typing — the search band and the answer in
// a window — because they had a switch each and the second to gain a key was
// always going to be the one nobody noticed had not.
type field struct {
	Text  string
	Caret int
}

// move puts the cursor at to, stopping at either end rather than wrapping: a
// cursor that reappears at the far end has lost the one thing it was telling
// you.
func (f field) move(to int) field {
	f.Caret = clamp(to, 0, len([]rune(f.Text)))
	return f
}

// insert puts s in at the cursor.
func (f field) insert(s string) field {
	f.Text, f.Caret = insertAt(f.Text, f.Caret, s)
	return f
}

// key applies a keystroke and reports whether the field took it. Anything it
// does not take belongs to whatever else is listening: the arrows that leave
// the field, Enter, the keys that move a list.
func (f field) key(msg tea.KeyMsg) (field, bool) {
	switch msg.String() {
	case "left":
		return f.move(f.Caret - 1), true
	case "right":
		return f.move(f.Caret + 1), true
	case "home":
		return f.move(0), true
	case "end":
		return f.move(len([]rune(f.Text))), true

	case "backspace":
		f.Text, f.Caret = backspaceAt(f.Text, f.Caret)
		return f, true
	case "delete":
		f.Text, f.Caret = deleteAt(f.Text, f.Caret)
		return f, true
	case "ctrl+w":
		f.Text, f.Caret = dropWordAt(f.Text, f.Caret)
		return f, true
	}

	// Typing. Space arrives as its own type rather than as a rune, and inside a
	// field it is a character like any other.
	switch msg.Type {
	case tea.KeySpace:
		return f.insert(" "), true
	case tea.KeyRunes:
		return f.insert(string(msg.Runes)), true
	}
	return f, false
}
