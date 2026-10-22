// What the terminal is asked for, and what its answers are turned back into.
//
// Two things, both in the same place because both are one pass over the bytes
// on their way in. The keypad's application-mode sequences are folded back into
// the characters the keys are printed with, as Keypad does. And the
// terminal is asked to say which key was pressed rather than leave it to be
// guessed — because ⌥↵ and ⌥⇥ arrive as a plain Return and a plain Tab
// otherwise, and a session hosted here then has no way to put a newline in a
// prompt or to hand the keys back.

package tty

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Reading is the terminal's input with both of those done to it, and the two
// requests that make them possible.
//
// A File rather than a plain io.Reader: bubbletea can only cancel a blocking
// read on something carrying a file descriptor, and wrapping the descriptor
// away would cost that — the app would quit one keystroke late.
func Reading(in *os.File, out io.Writer) io.Reader {
	fmt.Fprint(out, "\x1b>") // DECKPNM: the keypad, in numbers
	Ask(out)
	return &keys{File: in}
}

// Ask is the request to name the keys, and the question of whether it was
// heard.
//
// Made again once the app is on the alternate screen, because the two screens
// keep their own keyboard settings: asked for only on the way in, the request
// lands on the screen the app is about to leave and nothing is named on the one
// it works in.
func Ask(out io.Writer) {
	fmt.Fprint(out, "\x1b[>1u")   // push: disambiguate the keys
	fmt.Fprint(out, "\x1b[>4;2m") // and the older way of asking the same thing
	fmt.Fprint(out, "\x1b[?u")    // then say what you are doing now
}

// Done gives back what was asked for, so that whatever runs next in this
// terminal is not left with a keyboard it did not ask for.
func Done(out io.Writer) {
	fmt.Fprint(out, "\x1b[<u")
	fmt.Fprint(out, "\x1b[>4;0m")
}

type keys struct{ *os.File }

func (k *keys) Read(p []byte) (int, error) {
	got, err := k.File.Read(p)
	return read(p[:got]), err
}

// read rewrites what arrived, in place where it can and into the same buffer
// where it cannot: nothing here is ever longer than what it replaces.
//
// A sequence split across two reads is left alone rather than held back for its
// tail: a lone ESC is the Escape key, and an app that waits for a second byte
// before reporting it is an app where Escape only works once you press
// something else.
func read(b []byte) int {
	w := 0
	for i := 0; i < len(b); {
		if n, out, ok := keypadAt(b, i); ok {
			b[w] = out
			w, i = w+1, i+n
			continue
		}
		if n, ok := answerAt(b, i); ok {
			// The terminal saying what it is doing about the keys. Written
			// down rather than passed on: it is an answer to a question this
			// asked, and nothing downstream knows what to make of it.
			i += n
			continue
		}
		if n, out, ok := otherAt(b, i); ok {
			w += copy(b[w:], out)
			i += n
			continue
		}
		if n, out, ok := pressedAt(b, i); ok {
			w += copy(b[w:], out)
			i += n
			continue
		}
		b[w] = b[i]
		w, i = w+1, i+1
	}
	return w
}

// keypadAt is one of the keypad's own sequences, as the character it is
// printed with.
func keypadAt(b []byte, i int) (int, byte, bool) {
	if i+2 >= len(b) || b[i] != esc || b[i+1] != 'O' {
		return 0, 0, false
	}
	c, ok := keypad[b[i+2]]
	return 3, c, ok
}

// answerAt is the terminal's answer about what it is doing with the keys:
// CSI ? <flags> u, where anything but zero means it will name them.
func answerAt(b []byte, i int) (int, bool) {
	if i+3 >= len(b) || b[i] != esc || b[i+1] != '[' || b[i+2] != '?' {
		return 0, false
	}
	end := i + 3
	for end < len(b) && b[end] >= '0' && b[end] <= '9' {
		end++
	}
	if end >= len(b) || b[end] != 'u' {
		return 0, false
	}
	flags, _ := strconv.Atoi(string(b[i+3 : end]))
	Named = flags > 0
	slog.Info("the terminal answered about the keys", "flags", flags, "names them", Named)
	return end - i + 1, true
}

// Named is whether the terminal said it would name the keys. False until it
// says so, which includes every terminal that does not answer at all.
var Named bool

// pressedAt is a key reported as itself — CSI <key>;<modifiers> u — written
// back as the bytes a terminal sends when nobody asked it to be precise.
//
// The asking is what makes ⌥↵ and ⌥⇥ arrive at all: with the modifier left out
// of the report, one is a Return and the other a Tab, and nothing downstream
// can tell them from the unmodified keys.
func pressedAt(b []byte, i int) (int, []byte, bool) {
	if i+2 >= len(b) || b[i] != esc || b[i+1] != '[' {
		return 0, nil, false
	}
	end := i + 2
	for end < len(b) && (b[end] == ';' || b[end] == ':' || (b[end] >= '0' && b[end] <= '9')) {
		end++
	}
	if end >= len(b) || b[end] != 'u' {
		return 0, nil, false
	}
	key, mod, ok := parts(string(b[i+2 : end]))
	if !ok {
		return 0, nil, false
	}
	said := legacy(key, mod)
	if said == nil {
		return 0, nil, false
	}
	return end - i + 1, said, true
}

// otherAt is the older way a terminal names a key: CSI 27 ; <modifiers> ;
// <key> ~, which is what xterm does and what iTerm2 has done for years. Asked
// for alongside the newer one, because a keyboard that only answers to one of
// them is still a keyboard.
func otherAt(b []byte, i int) (int, []byte, bool) {
	if i+2 >= len(b) || b[i] != esc || b[i+1] != '[' {
		return 0, nil, false
	}
	end := i + 2
	for end < len(b) && (b[end] == ';' || (b[end] >= '0' && b[end] <= '9')) {
		end++
	}
	if end >= len(b) || b[end] != '~' {
		return 0, nil, false
	}
	said := strings.Split(string(b[i+2:end]), ";")
	if len(said) != 3 || said[0] != "27" {
		return 0, nil, false
	}
	mod, err := strconv.Atoi(said[1])
	if err != nil || mod < 1 {
		return 0, nil, false
	}
	key, err := strconv.Atoi(said[2])
	if err != nil {
		return 0, nil, false
	}
	out := legacy(key, mod)
	if out == nil {
		return 0, nil, false
	}
	return end - i + 1, out, true
}

// parts is the key and the modifiers out of the middle of the sequence. The
// key can carry alternates after a colon — the shifted character, the one the
// key would type on a US layout — and the first of them is the one meant.
func parts(said string) (key, mod int, ok bool) {
	first, rest, _ := strings.Cut(said, ";")
	first, _, _ = strings.Cut(first, ":")
	key, err := strconv.Atoi(first)
	if err != nil {
		return 0, 0, false
	}
	mod = 1
	if rest != "" {
		rest, _, _ = strings.Cut(rest, ":")
		if n, err := strconv.Atoi(rest); err == nil && n >= 1 {
			mod = n
		}
	}
	return key, mod, true
}

// legacy is what a terminal that was never asked would have sent, or nothing
// for a key that has no such form and is better left alone.
func legacy(key, mod int) []byte {
	// The modifiers are one more than the bits that say which they are.
	bits := mod - 1
	shift := bits&1 != 0
	alt := bits&2 != 0
	ctrl := bits&4 != 0

	var out []byte
	switch key {
	case 13: // Return
		// Any Return but the plain one is the one that puts a newline in what
		// is being typed rather than sending it. Written as meta-Return, which
		// is what a program reading this has always taken for that.
		if shift || alt || ctrl {
			return []byte{esc, '\r'}
		}
		out = []byte{'\r'}
	case 9: // Tab
		if shift {
			out = []byte("\x1b[Z")
		} else {
			out = []byte{'\t'}
		}
	case 27: // Escape
		out = []byte{esc}
	case 127, 8: // Backspace
		out = []byte{0x7f}
	case 32: // Space
		out = []byte{' '}
	default:
		if key < ' ' || key > 0x10ffff {
			return nil
		}
		r := rune(key)
		switch {
		case ctrl && r >= 'a' && r <= 'z':
			return []byte{byte(r-'a') + 1}
		case shift && r >= 'a' && r <= 'z':
			out = []byte(strings.ToUpper(string(r)))
		default:
			out = []byte(string(r))
		}
	}
	if alt {
		return append([]byte{esc}, out...)
	}
	return out
}

// Clear gives the screen back empty, the way the clear command does — the same
// three sequences, in the same order: drop the scrollback, home the cursor,
// erase what is on it.
//
// Quitting otherwise hands back the screen as it was on the way in, which for a
// session started by chantier is the picker, the line saying which session was
// started, and whatever was above them. The session itself was on the alternate
// screen and is gone; leaving its preamble under the prompt is remains.
func Clear(out io.Writer) {
	fmt.Fprint(out, "\x1b[3J\x1b[H\x1b[2J")
}
