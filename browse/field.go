package browse

import (
	"strings"
	"unicode"

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

// wordLeft and wordRight are where the cursor lands on a word skip.
//
// Leading separators are crossed before the word is, so a skip from the space
// after a word lands at the start of that word rather than stopping on the gap.
func (f field) wordLeft() int {
	r := []rune(f.Text)
	i := clamp(f.Caret, 0, len(r))
	for i > 0 && isSep(r[i-1]) {
		i--
	}
	for i > 0 && !isSep(r[i-1]) {
		i--
	}
	return i
}

func (f field) wordRight() int {
	r := []rune(f.Text)
	i := clamp(f.Caret, 0, len(r))
	for i < len(r) && isSep(r[i]) {
		i++
	}
	for i < len(r) && !isSep(r[i]) {
		i++
	}
	return i
}

// isSep is what separates one word from the next. Punctuation counts, so a skip
// through a path or a branch name stops at its parts — which is the text these
// apps are full of, and where a word ending only at a space would put every key
// at the far end of the line.
func isSep(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("/-_.,:;=+()[]{}'\"`~!?@#$%^&*|\\<>", r)
}

// drop removes the text between the cursor and to.
func (f field) drop(to int) field {
	r := []rune(f.Text)
	lo, hi := min(f.Caret, to), max(f.Caret, to)
	lo, hi = clamp(lo, 0, len(r)), clamp(hi, 0, len(r))
	f.Text, f.Caret = string(r[:lo])+string(r[hi:]), lo
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
	// Both spellings of a word skip: Terminal.app sends ⌥← as a meta escape and
	// ⌃← as a CSI sequence, and which one a keyboard produces is not something
	// an app gets to choose.
	case "alt+left", "ctrl+left":
		return f.move(f.wordLeft()), true
	case "alt+right", "ctrl+right":
		return f.move(f.wordRight()), true

	case "home", "ctrl+a":
		return f.move(0), true
	case "end", "ctrl+e":
		return f.move(len([]rune(f.Text))), true

	case "backspace":
		f.Text, f.Caret = backspaceAt(f.Text, f.Caret)
		return f, true
	case "delete":
		f.Text, f.Caret = deleteAt(f.Text, f.Caret)
		return f, true
	// Deleting by word mirrors moving by word, so what a skip crosses is what
	// the same key with a delete on it removes. ^W is left as it was: it is the
	// shell's key and means the shell's word, which ends at a space.
	case "alt+backspace":
		return f.drop(f.wordLeft()), true
	case "alt+delete":
		return f.drop(f.wordRight()), true
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
