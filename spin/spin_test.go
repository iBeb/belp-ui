package spin

import (
	"strings"
	"testing"
)

// The cycle is what the loader promises: empty, filling one at a time, empty
// again — and every frame the same width, so the column it sits in never moves.
func TestTheCycleFillsAndEmpties(t *testing.T) {
	want := []string{"○○○○", "●○○○", "●●○○", "●●●○", "●●●●"}
	for i, w := range want {
		if got := Frame(i); got != w {
			t.Errorf("Frame(%d) = %q, want %q", i, got, w)
		}
	}
	if Frame(Frames) != want[0] {
		t.Errorf("Frame(%d) = %q, want the cycle to start again", Frames, Frame(Frames))
	}
}

// A caller keeps a counter and never wraps it, so any number is a frame.
func TestAnyCounterIsAFrame(t *testing.T) {
	for _, n := range []int{-7, -1, 0, 1, 97, 1 << 20} {
		got := Frame(n)
		if len([]rune(got)) != Dots {
			t.Errorf("Frame(%d) = %q, want %d circles", n, got, Dots)
		}
		if strings.Trim(got, full+empty) != "" {
			t.Errorf("Frame(%d) = %q, want circles only", n, got)
		}
	}
}
