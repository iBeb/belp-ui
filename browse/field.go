package browse

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// field is one line of editable text: what it holds, where the cursor is, and
// what is selected.
//
// One type for both fields that take typing — the search band and the answer in
// a window — because they had a switch each and the second to gain a key was
// always going to be the one nobody noticed had not.
type field struct {
	Text  string
	Caret int

	// Anchor is the fixed end of a selection, the cursor being the moving one,
	// and Sel says whether there is one at all.
	//
	// A bool rather than a sentinel anchor, so that the zero value is "nothing
	// selected". With -1 for none, every Chrome and Prompt an app builds as a
	// literal would come with its first character selected, and the one that
	// forgot would be found by a user rather than by a compiler.
	Anchor int
	Sel    bool
}

// sel is the selected range as rune offsets, low end first.
func (f field) sel() (lo, hi int, ok bool) {
	if !f.Sel || f.Anchor == f.Caret {
		return 0, 0, false
	}
	if f.Anchor < f.Caret {
		return f.Anchor, f.Caret, true
	}
	return f.Caret, f.Anchor, true
}

// cut removes the selection and leaves the cursor where it began.
func (f field) cut() field {
	lo, hi, ok := f.sel()
	if !ok {
		return f
	}
	r := []rune(f.Text)
	f.Text, f.Caret, f.Sel = string(r[:lo])+string(r[hi:]), lo, false
	return f
}

// move puts the cursor at to, stopping at either end rather than wrapping: a
// cursor that reappears at the far end has lost the one thing it was telling
// you.
// Holding shift keeps or starts a selection; without it the selection goes,
// which is what makes an arrow the way out of one.
func (f field) move(to int, keep bool) field {
	switch {
	case !keep:
		f.Sel = false
	case !f.Sel:
		f.Anchor, f.Sel = f.Caret, true // it starts where the cursor was
	}
	f.Caret = clamp(to, 0, len([]rune(f.Text)))
	return f
}

// insert replaces the selection with s, or puts s in at the cursor.
func (f field) insert(s string) field {
	f = f.cut()
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
	f.Text, f.Caret, f.Sel = string(r[:lo])+string(r[hi:]), lo, false
	return f
}

// key applies a keystroke and reports whether the field took it. Anything it
// does not take belongs to whatever else is listening: the arrows that leave
// the field, Enter, the keys that move a list.
func (f field) key(msg tea.KeyMsg) (field, bool) {
	switch msg.String() {
	case "left":
		return f.move(f.Caret-1, false), true
	case "right":
		return f.move(f.Caret+1, false), true
	case "shift+left":
		return f.move(f.Caret-1, true), true
	case "shift+right":
		return f.move(f.Caret+1, true), true
	// Every spelling of a word skip, because the terminal decides which one it
	// sends and the app does not. Terminal.app's shipped key map turns ⌃← into
	// a CSI sequence and ⌥← into ESC b — unless "Use Option as Meta Key" is on,
	// when ⌥← becomes ESC and the plain arrow instead. ESC b and ESC f are also
	// what readline has meant by a word skip since long before any of this.
	case "alt+left", "ctrl+left", "alt+b":
		return f.move(f.wordLeft(), false), true
	case "alt+right", "ctrl+right", "alt+f":
		return f.move(f.wordRight(), false), true
	case "alt+shift+left", "ctrl+shift+left":
		return f.move(f.wordLeft(), true), true
	case "alt+shift+right", "ctrl+shift+right":
		return f.move(f.wordRight(), true), true

	case "home", "ctrl+a":
		return f.move(0, false), true
	case "end", "ctrl+e":
		return f.move(len([]rune(f.Text)), false), true
	case "shift+home":
		return f.move(0, true), true
	case "shift+end":
		return f.move(len([]rune(f.Text)), true), true

	// Each takes the selection where there is one, rather than a character out
	// of the middle of it and the selection silently with it.
	case "backspace":
		if _, _, ok := f.sel(); ok {
			return f.cut(), true
		}
		f.Text, f.Caret = backspaceAt(f.Text, f.Caret)
		return f, true
	case "delete":
		if _, _, ok := f.sel(); ok {
			return f.cut(), true
		}
		f.Text, f.Caret = deleteAt(f.Text, f.Caret)
		return f, true
	// Deleting by word mirrors moving by word, so what a skip crosses is what
	// the same key with a delete on it removes. ^W is left as it was: it is the
	// shell's key and means the shell's word, which ends at a space.
	case "alt+backspace":
		if _, _, ok := f.sel(); ok {
			return f.cut(), true
		}
		return f.drop(f.wordLeft()), true
	case "alt+delete":
		if _, _, ok := f.sel(); ok {
			return f.cut(), true
		}
		return f.drop(f.wordRight()), true
	case "ctrl+w":
		if _, _, ok := f.sel(); ok {
			return f.cut(), true
		}
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
