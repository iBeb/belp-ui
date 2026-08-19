// Package tty patches over what a terminal sends, before bubbletea reads it.
//
// One thing lives here so far, and it is the sort of thing that belongs in the
// shared library rather than in an app: every belp app reads the same keyboard
// through the same component, so a key that arrives wrong arrives wrong in all
// of them.
package tty

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const esc = 0x1b

// keypad is what the keypad sends in application mode, against what the keys are
// printed with: ESC O p for 0, ESC O n for the decimal point, and so on down the
// row.
//
// A, B, C, D and P, Q, R, S share the ESC O prefix and are deliberately absent:
// they are the four arrows and F1 to F4, which bubbletea already reads and which
// have to arrive as themselves.
var keypad = map[byte]byte{
	'p': '0', 'q': '1', 'r': '2', 's': '3', 't': '4',
	'u': '5', 'v': '6', 'w': '7', 'x': '8', 'y': '9',
	'n': '.', 'l': ',', 'k': '+', 'm': '-', 'j': '*', 'o': '/',
	'X': '=', 'M': '\r',
}

// Input is the option every belp app passes to bubbletea, in place of letting it
// take the terminal itself.
func Input() tea.ProgramOption { return tea.WithInput(Keypad(os.Stdin, os.Stdout)) }

// Keypad is the terminal's input with the keypad's application-mode sequences
// folded back into the characters the keys are printed with, and a request that
// the terminal stop sending them in the first place.
//
// Both halves are needed. DECKPNM asks for numeric mode and a terminal on its own
// obliges; inside Zellij the keypad stays in application mode whatever the
// program asked for, and then 0 arrives as ESC O p. Bubbletea knows the eight
// sequences that share that prefix and nothing else, so the rest reaches an app
// as alt+O followed by a letter — which is how typing 0 into a search field put
// "Op" in it.
//
// A File rather than a plain io.Reader: bubbletea can only cancel a blocking read
// on something carrying a file descriptor, and wrapping the descriptor away would
// cost that — the app would then quit one keystroke late.
func Keypad(in *os.File, out io.Writer) io.Reader {
	fmt.Fprint(out, "\x1b>") // DECKPNM
	return &numeric{File: in}
}

type numeric struct{ *os.File }

func (n *numeric) Read(p []byte) (int, error) {
	got, err := n.File.Read(p)
	return fold(p[:got]), err
}

// fold rewrites the sequences in place and returns what is left.
//
// In place is safe because the result is never longer than what arrived: three
// bytes become one, and everything else is copied over itself.
//
// A sequence split across two reads is left alone rather than held back for its
// tail: a lone ESC is the Escape key, and an app that waits for a second byte
// before reporting it is an app where Escape only works once you press something
// else.
func fold(b []byte) int {
	w := 0
	for i := 0; i < len(b); {
		if i+2 < len(b) && b[i] == esc && b[i+1] == 'O' {
			if c, ok := keypad[b[i+2]]; ok {
				b[w] = c
				w, i = w+1, i+3
				continue
			}
		}
		b[w] = b[i]
		w, i = w+1, i+1
	}
	return w
}
