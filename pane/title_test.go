package pane

import (
	"strings"
	"testing"
	"time"
)

// A title with a rune the parser underneath cannot carry was printed on the
// screen, and whatever the program drew next went over the top of it: the
// banner came out as "Claude CodClaude Code", and a resumed session had its own
// name across the first line of the transcript.
func TestATitleIsNeverDrawnOnTheScreen(t *testing.T) {
	p := sh(t, `printf '\033]0;\342\234\263 a title\007drawn\r\n'; sleep 30`, 40, 6)
	settle(t, p, "drawn", 5*time.Second)
	if screen := p.Render(); strings.Contains(screen, "a title") {
		t.Errorf("the title was drawn on the screen:\n%s", screen)
	}
}

// The same, ended with ST rather than a bell, and split across two reads the
// way a sequence arrives as often as not.
func TestOnlyAStringSequenceLosesThoseBytes(t *testing.T) {
	for _, c := range []struct{ say, in, want string }{
		{"a bell ends it", "\x1b]0;✳ x\x07✳ drawn", "\x1b]0; x\x07✳ drawn"},
		{"ST ends it", "\x1b]0;✳ x\x1b\\✳ drawn", "\x1b]0; x\x1b\\✳ drawn"},
		{"nothing to do with a title", "héllo ✳ drawn", "héllo ✳ drawn"},
		{"a link keeps its url", "\x1b]8;;https://x/é\x07go\x1b]8;;\x07", "\x1b]8;;https://x/\x07go\x1b]8;;\x07"},
	} {
		var keeps title
		if got := string(keeps.keep([]byte(c.in))); got != c.want {
			t.Errorf("%s: %q, want %q", c.say, got, c.want)
		}
	}

	var keeps title
	got := string(keeps.keep([]byte("\x1b]0;✳ Cla")))
	got += string(keeps.keep([]byte("ude\x07✳ drawn")))
	if want := "\x1b]0; Claude\x07✳ drawn"; got != want {
		t.Errorf("split across two reads: %q, want %q", got, want)
	}
}
