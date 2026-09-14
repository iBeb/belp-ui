// Package spin is the loader every belp app shows while it waits.
//
// Four circles filling left to right, then empty again. Not a rotating glyph:
// this one says a fixed amount of work is being done in steps, which is what a
// fetch of a handful of repositories is.
package spin

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Dots is how many circles there are, and so how wide the loader is.
const Dots = 4

// Frames is one full cycle: empty, filling one at a time, then empty again.
const Frames = Dots + 1

// Tick is how long a frame lasts. Fast enough to read as motion, slow enough
// that a fetch of a second or two shows most of a cycle.
const Tick = 120 * time.Millisecond

const (
	full  = "●"
	empty = "○"
)

// Width is the cells a frame takes, so a caller can pad a column to it.
const Width = Dots

// Frame is the nth frame, counting from any number: callers keep a counter and
// never have to wrap it themselves.
func Frame(n int) string {
	if n < 0 {
		n = -n
	}
	on := n % Frames
	return strings.Repeat(full, on) + strings.Repeat(empty, Dots-on)
}

// FrameMsg says a frame has elapsed.
type FrameMsg time.Time

// Next waits one frame. A caller that is still waiting asks for another; one
// that has finished simply stops, and the last message is ignored.
func Next() tea.Cmd {
	return tea.Tick(Tick, func(t time.Time) tea.Msg { return FrameMsg(t) })
}
