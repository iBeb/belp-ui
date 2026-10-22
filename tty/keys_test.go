package tty

import (
	"strings"
	"testing"
)

func through(in string) string {
	b := []byte(in)
	return string(b[:read(b)])
}

// The whole point of asking: ⌥↵ and ⌥⇥ are a Return and a Tab otherwise, and a
// session hosted here cannot tell them apart — so a prompt gets no newline and
// the keys never leave the session.
func TestTheModifiedKeysArriveAsThemselves(t *testing.T) {
	for _, c := range []struct{ say, in, want string }{
		{"⌥↵", "\x1b[13;3u", "\x1b\r"},
		{"⇧↵", "\x1b[13;2u", "\x1b\r"},
		{"⌥↵, the older way", "\x1b[27;3;13~", "\x1b\r"},
		{"⇧↵, the older way", "\x1b[27;2;13~", "\x1b\r"},
		{"⌥b, the older way", "\x1b[27;3;98~", "\x1bb"},
		{"⌥⇥", "\x1b[9;3u", "\x1b\t"},
		{"↵", "\x1b[13u", "\r"},
		{"⇥", "\x1b[9u", "\t"},
		{"⇧⇥", "\x1b[9;2u", "\x1b[Z"},
		{"escape", "\x1b[27u", "\x1b"},
		{"backspace", "\x1b[127u", "\x7f"},
		{"space", "\x1b[32u", " "},
		{"⌥space", "\x1b[32;3u", "\x1b "},
		{"^a", "\x1b[97;5u", "\x01"},
		{"⌥b", "\x1b[98;3u", "\x1bb"},
		{"⇧a", "\x1b[97;2u", "A"},
		{"a with its alternates", "\x1b[97:65;2u", "A"},
		{"a letter on its own", "hello", "hello"},
	} {
		if got := through(c.in); got != c.want {
			t.Errorf("%s: %q arrived as %q, want %q", c.say, c.in, got, c.want)
		}
	}
}

// The keypad keeps arriving as the characters its keys are printed with.
func TestTheKeypadStillFolds(t *testing.T) {
	if got := through("\x1bOq\x1bOr\x1bOs"); got != "123" {
		t.Errorf("the keypad typed %q", got)
	}
	// And the eight that share the prefix are the arrows and F1 to F4.
	for _, seq := range []string{"\x1bOA", "\x1bOD", "\x1bOP", "\x1bOS", "\x1b[1;5C"} {
		if got := through(seq); got != seq {
			t.Errorf("%q was rewritten as %q", seq, got)
		}
	}
}

// Anything that is not one of those is left exactly as it came, including a
// sequence cut in half by a read boundary.
func TestEverythingElseIsUntouched(t *testing.T) {
	for _, seq := range []string{
		"\x1b", "\x1bO", "\x1b[", "\x1b[13", "\x1b[13;", "\x1b[A", "\x1b[200~pasted\x1b[201~",
		"\x1b[3~", "\x1b[5~", "\x1b[1;3~",
		"\x1b[<64;10;10M", "plain words", "",
	} {
		if got := through(seq); got != seq {
			t.Errorf("%q arrived as %q", seq, got)
		}
	}
	// A run of them in one read, in any order.
	in := "ab\x1b[13;3ucd\x1bOq\x1b[A"
	if got, want := through(in), "ab\x1b\rcd1\x1b[A"; got != want {
		if !strings.Contains(got, "\x1b\r") {
			t.Errorf("a run came through as %q, want %q", got, want)
		}
	}
}

// The terminal's answer about the keys is written down rather than passed on:
// it is a reply to a question this asked, and nothing downstream knows what to
// make of it.
func TestTheAnswerAboutTheKeysIsTakenOut(t *testing.T) {
	Named = false
	if got := through("\x1b[?1u"); got != "" {
		t.Errorf("the answer came through as %q", got)
	}
	if !Named {
		t.Error("a terminal that names the keys was not taken at its word")
	}
	Named = true
	if got := through("ab\x1b[?0ucd"); got != "abcd" {
		t.Errorf("an answer in the middle of a read left %q", got)
	}
	if Named {
		t.Error("a terminal that names nothing was taken for one that does")
	}
	// And something that only looks like one is left alone.
	Named = false
	for _, seq := range []string{"\x1b[?", "\x1b[?1", "\x1b[?1h", "\x1b[?2004h"} {
		if got := through(seq); got != seq {
			t.Errorf("%q was taken for an answer: %q", seq, got)
		}
	}
}
