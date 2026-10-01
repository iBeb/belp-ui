package browse

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// How often a screen asks again what is going on.
//
// One clock for the apps that watch the same thing. They all draw sessions and
// services that start and stop while you are looking at them, and a screen whose
// first column is a live state is wrong the moment it stops asking — a session
// that has finished still turning its spinner, one you closed still lit.
//
// Five seconds: fast enough that a state is never wrong for long, slow enough
// that the answer costs nothing anybody notices. Shared because three apps
// choosing their own meant three answers to the same question, and a list in one
// window disagreeing with the list in the next.

// Beat is that clock.
const Beat = 5 * time.Second

// BeatMsg is one tick of it.
type BeatMsg time.Time

// Tick asks for the next one. A screen that wants the clock sends this from
// Init and again whenever it has a beat, which is what keeps one clock per
// screen however many times it is asked for.
func Tick() tea.Cmd {
	return tea.Tick(Beat, func(t time.Time) tea.Msg { return BeatMsg(t) })
}
