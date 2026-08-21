package icons

import "testing"

// Every glyph is one cell and one rune. A two-cell glyph would push every column
// after it out of place, and the whole point of an icon here is that it costs one.
func TestEveryGlyphIsASingleRune(t *testing.T) {
	for _, g := range All {
		r := []rune(g.Glyph)
		if len(r) != 1 {
			t.Errorf("%s = %q, want one rune, got %d", g.Name, g.Glyph, len(r))
		}
		if len(r) == 1 && !(r[0] >= 0xE000 && r[0] <= 0xF8FF) {
			t.Errorf("%s = U+%04X, outside the private use area a Nerd Font patches",
				g.Name, r[0])
		}
	}
}

// A repeated glyph means two activities look identical on screen, which is the
// one failure an icon set cannot recover from.
func TestNoGlyphIsUsedTwice(t *testing.T) {
	seen := map[string]string{}
	for _, g := range All {
		if first, ok := seen[g.Glyph]; ok {
			t.Errorf("%s and %s are the same glyph %q", first, g.Name, g.Glyph)
		}
		seen[g.Glyph] = g.Name
	}
}

// All has to hold every constant, or the preview draws a set the apps do not use.
func TestAllCoversEveryConstant(t *testing.T) {
	want := []string{
		Commit, Push, Branch, Trash,
		PullRequest, Merge, PullRequestClosed, Eye, Comment,
	}
	if len(All) != len(want) {
		t.Fatalf("All has %d entries, want %d", len(All), len(want))
	}
	have := map[string]bool{}
	for _, g := range All {
		have[g.Glyph] = true
		if g.Name == "" {
			t.Errorf("%q has no name", g.Glyph)
		}
	}
	for _, g := range want {
		if !have[g] {
			t.Errorf("%q is a constant but not in All", g)
		}
	}
}
