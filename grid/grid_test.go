package grid

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iBeb/belp-ui/theme"
)

// placed is a grid laid out but with nothing started, so the arrangement can be
// asked about without running three programs.
func placed(w, h int) Model {
	m := New(theme.Default(), control())
	m.width, m.height = w, h
	for _, p := range Place(m.spec, Rect{W: w, H: h}) {
		m.rects = append(m.rects, inner(p.Rect, w, h))
		m.areas = append(m.areas, p.Rect)
		m.specs = append(m.specs, p.Spec)
		m.panes = append(m.panes, nil)
	}
	return m
}

// A click picks the panel it landed in, which is the whole of what a click is
// for here.
func TestAClickPicksThePanelItLandedIn(t *testing.T) {
	m := placed(200, 50)

	for _, c := range []struct {
		name string
		x, y int
		want int
	}{
		{"the middle of the board", 40, 25, 0},
		{"the top right", 150, 5, 1},
		{"the bottom right", 150, 40, 2},
		{"the corner of the screen", 0, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := m.cellAt(c.x, c.y)
			if !ok {
				t.Fatalf("(%d,%d) landed in no cell", c.x, c.y)
			}
			if got != c.want {
				t.Errorf("(%d,%d) picked %q, want %q",
					c.x, c.y, m.specs[got].Title(), m.specs[c.want].Title())
			}
		})
	}
}

// The gap between two panels belongs to the one that gave it up, so a click a
// cell wide of the last character still lands somewhere.
func TestTheGapBelongsToThePanelThatGaveItUp(t *testing.T) {
	m := placed(200, 50)

	// The board is the left column, so the column it gave up is its own.
	r := m.rects[0]
	if r.W != m.areas[0].W-1 {
		t.Fatalf("the board kept %d of the %d columns it was placed with",
			r.W, m.areas[0].W)
	}
	if got, _ := m.cellAt(r.X+r.W, r.Y); got != 0 {
		t.Errorf("the gap beside the board picked cell %d, want the board", got)
	}

	// Sillage is stacked over vigie, so the row it gave up is sillage's.
	r = m.rects[1]
	if r.H != m.areas[1].H-1 {
		t.Fatalf("sillage kept %d of the %d rows it was placed with",
			r.H, m.areas[1].H)
	}
	if got, _ := m.cellAt(r.X, r.Y+r.H); got != 1 {
		t.Errorf("the gap under sillage picked cell %d, want sillage", got)
	}
}

// And a cell against the edge of the screen gives up nothing: the gap is
// between two panels, and there is no panel past the edge.
func TestACellAgainstTheEdgeKeepsEveryColumnItWasGiven(t *testing.T) {
	m := placed(200, 50)
	for i, r := range m.rects {
		a := m.areas[i]
		if a.X+a.W == 200 && r.W != a.W {
			t.Errorf("%s is against the right edge and kept %d of %d columns",
				m.specs[i].Title(), r.W, a.W)
		}
		if a.Y+a.H == 50 && r.H != a.H {
			t.Errorf("%s is against the bottom and kept %d of %d rows",
				m.specs[i].Title(), r.H, a.H)
		}
	}
}

// A press moves the focus; a release must not move it back somewhere else.
func TestOnlyAPressMovesTheFocus(t *testing.T) {
	m := placed(200, 50)
	next, _ := m.Update(tea.MouseMsg{X: 150, Y: 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next.(Model)
	if m.focus != 1 {
		t.Fatalf("a press left the focus at %d", m.focus)
	}

	next, _ = m.Update(tea.MouseMsg{X: 40, Y: 25, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	if got := next.(Model).focus; got != 1 {
		t.Errorf("a release moved the focus to %d", got)
	}
}

// Tab is the grid's and everything else is the app's, or a search field in a
// cell could never be typed into.
func TestOnlyTheGridsOwnKeysAreTakenFromTheApp(t *testing.T) {
	m := placed(200, 50)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if m.focus != 1 {
		t.Errorf("⇥ left the focus at %d", m.focus)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := next.(Model).focus; got != 0 {
		t.Errorf("⇧⇥ left the focus at %d", got)
	}
}

func stripANSI(s string) string {
	out := make([]rune, 0, len(s))
	skip := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			skip = true
		case skip && r == 'm':
			skip = false
		case !skip:
			out = append(out, r)
		}
	}
	return string(out)
}

// running is a grid with real programs in it, so a cell can be watched dying.
// shell is the shell these tests drive their panels with.
//
// Nameable from the outside, because /bin/sh is bash-ish on a Mac and dash on
// the machines CI runs on — and a test written against one silently passes
// there and fails here. `make check` runs this package a second time against
// dash for exactly that reason.
func shell() string {
	if sh := os.Getenv("BELP_TEST_SH"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

func running(t *testing.T, specs ...Spec) Model {
	t.Helper()
	m := New(theme.Default(), Spec{Split: "columns", Cells: specs})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	m = next.(Model)
	t.Cleanup(m.Close)
	return m
}

// waitScreen polls until the screen says something, so a test asserts on output
// that has arrived rather than on output that was sent.
func waitScreen(t *testing.T, m Model, want string) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for !strings.Contains(stripANSI(m.View()), want) {
		select {
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			t.Fatalf("%q never reached the screen:\n%s", want, stripANSI(m.View()))
		}
	}
}

func waitGone(t *testing.T, m Model, i int) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for !m.gone(i) {
		select {
		case <-m.panes[i].Changed():
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatalf("cell %d never finished", i)
		}
	}
}

// Nothing is drawn around a program, and two panels do not touch: a row is the
// left program's output, one blank column, and the right program's.
//
// Asserted on the row itself rather than on the absence of box characters,
// which would pass just as happily if a cell were drawn in the wrong place or
// a column wide.
func TestNothingIsDrawnAroundAPanelAndTwoDoNotTouch(t *testing.T) {
	line := func(c string) string {
		return strings.Repeat(c, 200)
	}
	m := running(t,
		Spec{Run: []string{shell(), "-c", "printf '" + line("a") + "\\r\\n'; sleep 30"}, Name: "a"},
		Spec{Run: []string{shell(), "-c", "printf '" + line("b") + "\\r\\n'; sleep 30"}, Name: "b"})

	// 80 columns split in two: the left panel gives up its last column, the
	// right is against the edge of the screen and keeps all of its own.
	left, right := m.rects[0], m.rects[1]
	if left.W != 39 || right.W != 40 {
		t.Fatalf("the panels are %d and %d columns wide, want 39 and 40", left.W, right.W)
	}

	waitScreen(t, m, strings.Repeat("b", right.W))
	want := strings.Repeat("a", left.W) + " " + strings.Repeat("b", right.W)
	if got := strings.Split(stripANSI(m.View()), "\n")[0]; got != want {
		t.Errorf("the first row is\n%q\nwant\n%q", got, want)
	}
}

// ^C is how you stop a thing in a terminal, and these apps take it as leave. So
// the panel has to say it has gone rather than sit there looking alive.
func TestACellWhoseProgramLeftSaysSo(t *testing.T) {
	m := running(t,
		Spec{Run: []string{shell(), "-c", "printf 'staying\\r\\n'; sleep 30"}, Name: "stays"},
		Spec{Run: []string{shell(), "-c", "printf 'drewthis\\r\\n'; exit 0"}, Name: "goes"})

	waitGone(t, m, 1)
	if m.gone(0) {
		t.Error("the cell that was told to stay has gone too")
	}
	// Waiting for the cell that left says nothing about the cell that stayed:
	// one exits at once and the other has to get its line through a pty first,
	// so asserting on that line without waiting for it is a race that a busy
	// machine loses.
	waitScreen(t, m, "staying")

	screen := stripANSI(m.View())
	if !strings.Contains(screen, "goes ended") {
		t.Errorf("the panel does not say its program left:\n%s", screen)
	}
	if !strings.Contains(screen, "run it again") {
		t.Errorf("the panel does not say how to bring it back:\n%s", screen)
	}
	if strings.Contains(screen, "drewthis") {
		t.Errorf("the panel is still showing the screen it drew before it died:\n%s", screen)
	}
	if !strings.Contains(screen, "staying") {
		t.Errorf("the cell that lived stopped drawing:\n%s", screen)
	}
}

// A cell that has gone belongs to the grid again, so the key that brings it
// back is not swallowed by a program that is no longer there.
func TestAnEndedCellCanBeStartedAgain(t *testing.T) {
	m := running(t, Spec{Run: []string{shell(), "-c", "printf 'alive\\r\\n'; sleep 30"}, Name: "one"},
		Spec{Run: []string{shell(), "-c", "exit 3"}, Name: "two"})

	waitGone(t, m, 1)
	m.focus = 1
	before := m.panes[1]

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)

	// That a new one was started, not that it is still running: this cell's
	// program is `exit 3`, so the thing ↵ starts is gone again within
	// milliseconds. Asking whether it is alive is a race the test wins only by
	// being slower than the process it started.
	if m.panes[1] == before {
		t.Fatal("↵ on a dead cell did not start it again")
	}
	if m.panes[1] == nil {
		t.Fatal("↵ on a dead cell left it with nothing at all")
	}
	// And the cell that was told to stay is untouched.
	if m.gone(0) {
		t.Error("restarting one cell took the other with it")
	}
	m.Close()
}

// Keys meant for the grid must not be posted to a program that has finished:
// there is nothing there to read them.
func TestKeysAreNotSentToAProgramThatHasGone(t *testing.T) {
	m := running(t, Spec{Run: []string{shell(), "-c", "exit 0"}, Name: "gone"})
	waitGone(t, m, 0)

	// Any key other than the one that restarts is simply ignored, and nothing
	// panics for want of a process.
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("x")}, {Type: tea.KeyUp}} {
		next, _ := m.Update(k)
		m = next.(Model)
		if !m.gone(0) {
			t.Errorf("%v started something", k)
		}
	}
}

// ^C leaves belp rather than closing a panel.
//
// On its own in a terminal it means stop this thing, and these apps take that as
// leave — which in a grid means closing panels one at a time until nothing is
// left and there is no way out of the grid at all.
func TestControlCLeavesBelpRatherThanAPanel(t *testing.T) {
	m := running(t, Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "one"},
		Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "two"})

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("^C did nothing")
	}
	if _, quits := cmd().(tea.QuitMsg); !quits {
		t.Error("^C did not leave belp")
	}
	// And it takes the programs with it, rather than leaving them running
	// against a terminal nobody is reading.
	deadline := time.After(5 * time.Second)
	for i := range m.panes {
		for !m.gone(i) {
			select {
			case <-time.After(50 * time.Millisecond):
			case <-deadline:
				t.Fatalf("cell %d was left running", i)
			}
		}
	}
}

// Ordinary keys still reach the program: taking ^C for the grid must not mean
// taking everything, or a search field in a cell could never be typed into.
func TestOrdinaryKeysStillReachTheProgram(t *testing.T) {
	m := running(t, Spec{Run: []string{shell(), "-c",
		`read -r k; printf 'got %s\r\n' "$k"; sleep 30`}, Name: "listens"})

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)

	deadline := time.After(5 * time.Second)
	for {
		if strings.Contains(stripANSI(m.View()), "got hello") {
			return
		}
		select {
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatalf("the app never saw the keys:\n%s", stripANSI(m.View()))
		}
	}
}

// Inside belp nothing closes an app: ^C is the grid's, and Esc was never a quit
// in the first place. A panel that cannot be closed is one that cannot be lost.
func TestNothingInsideBelpClosesAnApp(t *testing.T) {
	m := running(t, Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "one"})

	for _, k := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyRunes, Runes: []rune("q")}} {
		next, _ := m.Update(k)
		m = next.(Model)
	}
	// Give anything that was going to die the chance to.
	deadline := time.After(time.Second)
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			if m.gone(0) {
				t.Fatal("a key closed the app inside the grid")
			}
		case <-deadline:
			return
		}
	}
}

// A click in another panel chooses it and goes no further: acting on a row in a
// panel you had not been looking at is not what pointing at it meant.
// echoesEveryByte prints a line for each byte that reaches it, and stops when
// its input ends.
//
// dd and od rather than `read -n1`, which is bash's: /bin/sh is dash on the
// machines this runs on in CI, where -n is an illegal option — so the read
// failed instantly, the loop spun, and the marker appeared whether or not
// anything had been sent. A test that reports a fault with no input at all
// cannot say anything about input.
//
// stty first, because a pty starts in canonical mode and a byte-at-a-time read
// would sit there until a newline arrived. A mouse press contains none, so
// without this the marker could never appear and the test would pass by being
// unable to fail.
const echoesEveryByte = `stty -icanon -echo min 1 time 0 2>/dev/null; ` +
	`while :; do ` +
	`c=$(dd bs=1 count=1 2>/dev/null | od -An -tx1 | tr -d " \n"); ` +
	`[ -n "$c" ] || break; printf 'saw:%s\r\n' "$c"; done`

func TestAClickElsewhereChoosesAndDoesNotActThere(t *testing.T) {
	m := running(t,
		Spec{Run: []string{shell(), "-c", echoesEveryByte}, Name: "a"},
		Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "b"})

	// The second panel is not the focused one; pressing in it should only move
	// the focus.
	r := m.rects[1]
	next, _ := m.Update(tea.MouseMsg{X: r.X + 2, Y: r.Y + 2,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = next.(Model)
	if m.focus != 1 {
		t.Fatalf("the click did not choose the panel; focus is %d", m.focus)
	}

	// Given time to arrive, and then asserted absent. A negative check made the
	// instant after the click passes whether or not anything was sent — it just
	// beats the write through the pty — which is how a panel that really was
	// echoing the click went unnoticed here and failed in CI instead.
	//
	// The wait is the time its opposite takes to show something, and that is
	// asserted next door: if the marker can appear in this long, not appearing
	// in this long means something.
	deadline := time.After(time.Second)
	for {
		if screen := stripANSI(m.View()); strings.Contains(screen, "saw:") {
			t.Fatalf("the click was also acted on inside a panel that had not the keys:\n%s",
				screen)
		}
		select {
		case <-deadline:
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// And a click in the panel that does have the keys is passed on, which is what
// makes the test above worth anything: a marker that never appears proves
// nothing about a click that was supposed not to produce one.
func TestAClickInTheFocusedPanelIsPassedOn(t *testing.T) {
	m := running(t,
		Spec{Run: []string{shell(), "-c", echoesEveryByte}, Name: "a"},
		Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "b"})

	r := m.rects[0] // the focused one
	next, _ := m.Update(tea.MouseMsg{X: r.X + 2, Y: r.Y + 2,
		Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = next.(Model)

	deadline := time.After(5 * time.Second)
	for {
		if strings.Contains(stripANSI(m.View()), "saw:") {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("the click never reached the panel that had the keys:\n%s",
				stripANSI(m.View()))
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// The panel that has the keys is told so, and the others are told they have
// not — which is what lets them draw themselves quiet.
func TestEveryPanelIsToldWhetherItHasTheKeys(t *testing.T) {
	m := running(t,
		Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "a"},
		Spec{Run: []string{shell(), "-c", "sleep 30"}, Name: "b"})

	// Nothing observable from outside a shell, so this asserts the wiring: the
	// focus moves and neither call panics on a pane that has gone.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if m.focus != 1 {
		t.Fatalf("⇥ left the focus at %d", m.focus)
	}
	m.panes[0].Close()
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := next.(Model).focus; got != 0 {
		t.Errorf("⇧⇥ onto a closed panel left the focus at %d", got)
	}
}
