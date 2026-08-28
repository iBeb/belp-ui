// Package when says a time the way a person would.
//
// One reading of a timestamp for every belp app: the same words, the same widths,
// so two apps side by side do not disagree about what yesterday looks like.
package when

import (
	"fmt"
	"time"
)

// Width is the widest label this can produce, "yesterday at 00:00".
//
// A column padded to it never shifts as the day turns over, which is what happens
// when a run of "today at" rows is followed by the first full date.
const Width = len("yesterday at 00:00")

// When is today and yesterday with the clock, anything older as a date.
//
// Calendar days compared as dates, never as a rounded duration: rounding to a
// 24-hour boundary rounds to midnight UTC, which is the wrong midnight
// everywhere but one zone — it filed anything done before 02:00 in Paris under
// yesterday.
//
// now is passed rather than read so that a test can place itself in time. Callers
// should hand it time.Now() at the moment of drawing and never keep it: a stored
// clock is how a screen left open comes to call last week's work today.
func When(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	t, now = t.Local(), now.Local()
	clock := t.Format("15:04")
	switch {
	case SameDay(t, now):
		return "today at " + clock
	case SameDay(t, now.AddDate(0, 0, -1)):
		return "yesterday at " + clock
	default:
		return t.Format("02.01.2006 15:04")
	}
}

// SameDay reports whether two times fall on the same calendar day, in whichever
// zone they carry.
func SameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Ago is a rough age, for saying how old something on screen is.
//
// Deliberately coarse: the point is to notice that a screen is stale, not to know
// to the second how stale. The bounded readings are the cases and days is the
// default, because days is the one with no upper end — a screen can be a month
// old, and nothing else here can.
func Ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		// Includes a clock that has gone backwards, which is not an age either.
		return "just now"
	case d < 2*time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
}
