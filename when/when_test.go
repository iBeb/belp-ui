package when

import (
	"testing"
	"time"
)

// Paris, because the bugs this package exists to prevent are all about the
// difference between local midnight and UTC midnight.
func paris(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("no zone database")
	}
	return loc
}

func TestWhen(t *testing.T) {
	loc := paris(t)
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, loc)

	cases := map[string]time.Time{
		// Before 02:00 is the case Truncate(24h) got wrong: it rounds the instant
		// to midnight UTC, which is 02:00 here.
		"today at 00:05":     time.Date(2026, 8, 25, 0, 5, 0, 0, loc),
		"today at 01:00":     time.Date(2026, 8, 25, 1, 0, 0, 0, loc),
		"today at 12:00":     now,
		"yesterday at 23:55": time.Date(2026, 8, 24, 23, 55, 0, 0, loc),
		"yesterday at 00:30": time.Date(2026, 8, 24, 0, 30, 0, 0, loc),
		"23.08.2026 16:05":   time.Date(2026, 8, 23, 16, 5, 0, 0, loc),
	}
	for want, at := range cases {
		if got := When(at, now); got != want {
			t.Errorf("When(%v) = %q, want %q", at, got, want)
		}
	}
	if got := When(time.Time{}, now); got != "" {
		t.Errorf("When(zero) = %q, want empty", got)
	}
}

// The clock shown is the event's own, in local time, never shifted.
func TestWhenShowsTheLocalClock(t *testing.T) {
	loc := paris(t)
	utc := time.Date(2026, 8, 25, 10, 14, 5, 0, time.UTC) // 12:14 in Paris
	now := time.Date(2026, 8, 25, 12, 20, 0, 0, loc)
	if got := When(utc, now); got != "today at 12:14" {
		t.Errorf("When() = %q, want %q", got, "today at 12:14")
	}
}

// The bug a four-day-old screen showed: a row keeps calling itself today for as
// long as the clock it is compared against stands still.
func TestWhenStopsSayingTodayOnceTheDayTurns(t *testing.T) {
	loc := paris(t)
	event := time.Date(2026, 8, 23, 16, 5, 0, 0, loc)

	sameDay := time.Date(2026, 8, 23, 17, 0, 0, 0, loc)
	if got := When(event, sameDay); got != "today at 16:05" {
		t.Errorf("on the day: %q", got)
	}
	twoDaysOn := time.Date(2026, 8, 25, 12, 3, 0, 0, loc)
	if got := When(event, twoDaysOn); got != "23.08.2026 16:05" {
		t.Errorf("two days on: %q, want the date", got)
	}
}

// Width has to fit every label, or a column padded to it still shifts.
func TestWidthFitsEveryLabel(t *testing.T) {
	loc := paris(t)
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, loc)
	for _, at := range []time.Time{
		now,
		now.AddDate(0, 0, -1),
		time.Date(2026, 12, 31, 23, 59, 0, 0, loc),
		time.Date(1999, 1, 1, 0, 0, 0, 0, loc),
	} {
		if got := When(at, now); len(got) > Width {
			t.Errorf("When(%v) = %q, %d chars, wider than Width %d", at, got, len(got), Width)
		}
	}
}

func TestAgo(t *testing.T) {
	cases := map[string]time.Duration{
		"just now": 30 * time.Second,
		"3m":       3 * time.Minute,
		"90m":      90 * time.Minute,
		"5h":       5 * time.Hour,
		"47h":      47 * time.Hour,
		"4d":       4 * 24 * time.Hour,
	}
	for want, d := range cases {
		if got := Ago(d); got != want {
			t.Errorf("Ago(%v) = %q, want %q", d, got, want)
		}
	}
	// A clock that has gone backwards is not an age.
	if got := Ago(-time.Hour); got != "just now" {
		t.Errorf("Ago(negative) = %q", got)
	}
}
