package tty

import (
	"io"
	"os"
	"testing"
)

func folded(in string) string {
	b := []byte(in)
	return string(b[:fold(b)])
}

func TestKeypadKeysArriveAsTheCharactersTheyArePrintedWith(t *testing.T) {
	cases := map[string]string{
		"\x1bOp":                   "0",
		"\x1bOq\x1bOr\x1bOs":       "123",
		"\x1bOn":                   ".",
		"\x1bOM":                   "\r",
		"\x1bOk\x1bOm\x1bOj\x1bOo": "+-*/",
		"v\x1bOp.\x1bOq\x1bOq0":    "v0.110", // v0.110.2 typed on the keypad
	}
	for in, want := range cases {
		if got := folded(in); got != want {
			t.Errorf("fold(%q) = %q, want %q", in, got, want)
		}
	}
}

// The eight sequences that share the prefix and are not the keypad: bubbletea
// reads these correctly, so folding them would break the arrows.
func TestTheArrowsAndTheFunctionKeysPassThrough(t *testing.T) {
	for _, seq := range []string{
		"\x1bOA", "\x1bOB", "\x1bOC", "\x1bOD",
		"\x1bOP", "\x1bOQ", "\x1bOR", "\x1bOS",
		"\x1b[A", "\x1b[1;5C", "\x1b", "\x1b[200~pasted\x1b[201~",
	} {
		if got := folded(seq); got != seq {
			t.Errorf("fold(%q) = %q, want it untouched", seq, got)
		}
	}
}

// A sequence cut in half by a read boundary is left as it is: holding the ESC
// back for its tail would mean the Escape key only registering on the next
// keypress.
func TestAPartialSequenceIsNotHeldBack(t *testing.T) {
	for _, partial := range []string{"\x1b", "\x1bO", "abc\x1bO"} {
		if got := folded(partial); got != partial {
			t.Errorf("fold(%q) = %q, want it passed straight through", partial, got)
		}
	}
}

func TestKeypadReadsThroughTheFile(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var mode discard
	in := Keypad(r, &mode)
	if mode.String() != "\x1b>" {
		t.Errorf("wrote %q to the terminal, want the numeric keypad request", mode.String())
	}
	// The reader has to stay something bubbletea can cancel, which means keeping
	// the file descriptor visible.
	if _, ok := in.(interface{ Fd() uintptr }); !ok {
		t.Error("the wrapped input no longer carries a file descriptor")
	}

	go func() {
		_, _ = w.Write([]byte("\x1bOq\x1bOn\x1bOr"))
		_ = w.Close()
	}()
	got, err := io.ReadAll(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1.2" {
		t.Errorf("read %q, want %q", got, "1.2")
	}
}

// discard collects what was written to the terminal.
type discard struct{ b []byte }

func (d *discard) Write(p []byte) (int, error) {
	d.b = append(d.b, p...)
	return len(p), nil
}
func (d *discard) String() string { return string(d.b) }
