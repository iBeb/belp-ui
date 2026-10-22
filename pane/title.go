package pane

// What a terminal's title costs here.
//
// A program sets it with a string sequence — ESC ] 0 ; text BEL — and Claude
// Code's title is "✳ Claude Code", and on a resumed session the name of the
// session after it. The parser underneath the emulator leaves the string on the
// first byte of a three-byte rune: it prints that rune and everything after it
// on the screen and carries on as though the sequence had ended. What was drawn
// was " Claude Code" across the banner and then the banner over the top of it —
// "Claude CodClaude Code" — and, on a resumed session, the session's own name
// printed into the first line of it.
//
// Checked against x/vt of 2026-09-20 and x/ansi v0.11.8, both the current
// releases: `ESC ] 0 ; ✳ x BEL hello` puts " x" on the screen where the same
// with "café" does not, three bytes being the difference. Nothing here reads
// the title, so the bytes that sequence cannot carry are taken out of it before
// the emulator is told.
//
// The rule is only inside a string sequence — OSC, DCS, APC, PM, SOS — and
// never in what a program draws, where every rune it writes is its own.

// title takes those bytes out, and remembers where it was: a sequence is split
// across reads as often as not.
type title struct {
	in  bool // inside a string sequence
	esc bool // the byte before was ESC
}

// keep is the same bytes less the ones a string sequence cannot carry, written
// over the buffer it was given.
func (t *title) keep(b []byte) []byte {
	out := b[:0]
	for _, c := range b {
		switch {
		case !t.in:
			if t.esc && opens(c) {
				t.in = true
			}
			t.esc = c == esc
			out = append(out, c)
		case t.esc:
			// ESC \ ends it; ESC anything else is the program's business.
			t.in = c != end
			t.esc = c == esc
			out = append(out, c)
		case c == bell:
			t.in = false
			out = append(out, c)
		case c == esc:
			t.esc = true
			out = append(out, c)
		case c < 0x80:
			out = append(out, c)
		}
	}
	return out
}

// opens says whether this is the byte after ESC that begins a string.
func opens(c byte) bool {
	switch c {
	case ']', 'P', '^', '_', 'X': // OSC, DCS, APC, PM, SOS
		return true
	}
	return false
}

const (
	esc  = 0x1b
	bell = 0x07
	end  = '\\' // with the ESC before it, ST
)
