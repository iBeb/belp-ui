package pane

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// settle waits for the pane to say it changed, so a test never guesses at a
// duration it cannot know.
func settle(t *testing.T, p *Pane, want string, within time.Duration) string {
	t.Helper()
	deadline := time.After(within)
	for {
		screen := p.Render()
		if want == "" || strings.Contains(screen, want) {
			return screen
		}
		select {
		case <-p.Changed():
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatalf("waited %s for %q; the screen was:\n%s", within, want, screen)
		}
	}
}

func sh(t *testing.T, script string, w, h int) *Pane {
	t.Helper()
	p, err := Start("probe", "", []string{"/bin/sh", "-c", script}, w, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// What the program drew is what the cell shows.
func TestACellShowsWhatTheProgramDrew(t *testing.T) {
	p := sh(t, `printf 'hello from a pane\r\n'; sleep 30`, 40, 6)
	settle(t, p, "hello from a pane", 5*time.Second)
}

// Styling survives, or every app in the grid loses the colour it uses to mean
// something.
func TestStylingSurvives(t *testing.T) {
	p := sh(t, `printf '\033[1;32mgreen\033[0m\r\n'; sleep 30`, 40, 6)
	screen := settle(t, p, "green", 5*time.Second)
	if !strings.Contains(screen, "\x1b[") {
		t.Errorf("the styling was flattened:\n%q", screen)
	}
}

// Typing reaches the program as the bytes a terminal would have sent.
func TestTypingReachesTheProgram(t *testing.T) {
	p := sh(t, `read line; printf 'you said %s\r\n' "$line"; sleep 30`, 40, 6)
	settle(t, p, "", time.Second)

	p.Type([]byte("marco\r"))
	settle(t, p, "you said marco", 5*time.Second)
}

// The half that is easy to leave out: a program asks the terminal questions and
// waits for the answers before drawing anything. A host that only listens
// leaves it waiting for ever — which is exactly what happened the first time.
func TestTheHostAnswersWhatTheProgramAsks(t *testing.T) {
	// Ask what the terminal is (a primary device attributes request), then wait
	// for a reply before saying anything.
	p := sh(t, `printf '\033[c'; read -t 5 -r reply; printf 'answered\r\n'; sleep 30`, 40, 6)
	settle(t, p, "answered", 8*time.Second)
}

// Both halves get told: the emulator, which is what gets drawn, and the
// program, which is what decides what to draw.
func TestResizeReachesTheProgram(t *testing.T) {
	p := sh(t, `while :; do stty size; sleep 0.2; done`, 40, 6)
	settle(t, p, "6 40", 5*time.Second)

	p.Resize(100, 20)
	settle(t, p, "20 100", 5*time.Second)
}

// A program that has finished says so, rather than leaving a cell that looks
// alive and answers nothing.
func TestAFinishedProgramSaysSo(t *testing.T) {
	p := sh(t, `printf 'bye\r\n'`, 40, 6)
	settle(t, p, "bye", 5*time.Second)

	deadline := time.After(5 * time.Second)
	for {
		if ended, _ := p.Ended(); ended {
			return
		}
		select {
		case <-p.Changed():
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatal("a finished program still reads as running")
		}
	}
}

// Nothing to run is a mistake in the config, not a crash.
func TestNothingToRunIsRefused(t *testing.T) {
	if _, err := Start("empty", "", nil, 10, 10); err == nil {
		t.Error("a cell with no command started anyway")
	}
}

// The wheel has to go back through what has scrolled off the top, because the
// program in the cell never asked the terminal for the mouse and what it has
// already written is not its to remember.
func TestTheViewGoesBackThroughWhatScrolledOff(t *testing.T) {
	p := sh(t, `i=1; while [ $i -le 60 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done; sleep 30`, 40, 6)
	settle(t, p, "line 60", 5*time.Second)

	if got := p.Back(20); got != 20 {
		t.Fatalf("the view went back %d lines, not 20", got)
	}
	screen := p.Render()
	if strings.Contains(screen, "line 60") {
		t.Errorf("the view did not move:\n%s", screen)
	}
	if !strings.Contains(screen, "line 40") {
		t.Errorf("twenty lines up is not where it went:\n%s", screen)
	}
	if lines := strings.Count(screen, "\n") + 1; lines != 6 {
		t.Errorf("the view is %d lines in a cell six high", lines)
	}

	p.Home()
	if screen := p.Render(); !strings.Contains(screen, "line 60") {
		t.Errorf("it did not come back to the bottom:\n%s", screen)
	}
}

// Going back stops at the oldest line there is, and coming down stops at the
// bottom: a wheel spun hard is the ordinary case, not a mistake.
func TestTheViewStopsAtBothEnds(t *testing.T) {
	p := sh(t, `i=1; while [ $i -le 60 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done; sleep 30`, 40, 6)
	settle(t, p, "line 60", 5*time.Second)

	if got := p.Back(10000); got > 60 || got == 0 {
		t.Errorf("it went back %d lines through sixty", got)
	}
	if !strings.Contains(p.Render(), "line 1") {
		t.Errorf("the oldest line is not on it:\n%s", p.Render())
	}
	if got := p.Back(-10000); got != 0 {
		t.Errorf("it came down to %d, not the bottom", got)
	}
}

// What arrives while somebody is reading further up must not drag the text out
// from under them: the view is held on its lines, not at a distance from an end
// that keeps moving.
func TestWhatArrivesDoesNotMoveWhatIsBeingRead(t *testing.T) {
	p := sh(t, `i=1; while [ $i -le 60 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done
read go; i=61; while [ $i -le 90 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done; sleep 30`, 40, 6)
	settle(t, p, "line 60", 5*time.Second)

	p.Back(20)
	held := p.Render()

	p.Type([]byte("go\r"))
	settle(t, p, "", 2*time.Second)
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if now := p.Render(); now != held {
		t.Errorf("thirty lines of output moved the reading:\nwas:\n%s\nnow:\n%s", held, now)
	}
	// And the output did arrive, or the test above passed on nothing having
	// happened at all.
	p.Home()
	settle(t, p, "line 90", 5*time.Second)
}

// The cursor is the host's to draw, so the cell says where it is, whether the
// program wants it seen, and nothing at all while the view is held up the
// history — the line it is on is not the one being read.
func TestTheCellSaysWhereTheCursorIs(t *testing.T) {
	p := sh(t, `printf 'ready\r\n'; read go; printf '\033[?25l'; sleep 30`, 40, 6)
	settle(t, p, "ready", 5*time.Second)
	if _, y, shown := p.Caret(); !shown || y != 1 {
		t.Errorf("the caret is at row %d, shown %v, after one line", y, shown)
	}

	// And gone once the program says to hide it, which is what it does while
	// it is drawing.
	p.Type([]byte("go\r"))
	for i := 0; i < 40; i++ {
		if _, _, shown := p.Caret(); !shown {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, _, shown := p.Caret(); shown {
		t.Error("the caret is still shown after the program hid it")
	}
}

func TestNoCaretWhileTheViewIsHeldBack(t *testing.T) {
	p := sh(t, `i=1; while [ $i -le 60 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done; sleep 30`, 40, 6)
	settle(t, p, "line 60", 5*time.Second)
	if _, _, shown := p.Caret(); !shown {
		t.Fatal("no caret at the bottom")
	}
	p.Back(20)
	if _, _, shown := p.Caret(); shown {
		t.Error("a caret was reported for a screen that is not being shown")
	}
}

// hold sets how much history the cell keeps, so a test can fill it in a second
// rather than in an hour of use.
func hold(p *Pane, lines int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.term.SetScrollbackSize(lines)
}

// The history holds so many lines and no more: after that the oldest is dropped
// for each new one, so counting it stops working — which is what it did, a line
// at a time, towards whatever had just been written. Every session anybody
// wants to scroll is long enough to be in this state.
func TestTheViewHoldsOnAHistoryThatIsFull(t *testing.T) {
	p := sh(t, `i=1; while [ $i -le 120 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done
read go; i=121; while [ $i -le 160 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done; sleep 30`, 40, 6)
	hold(p, 100)
	settle(t, p, "line 120", 5*time.Second)

	p.Back(40)
	held := p.Render()

	p.Type([]byte("go\r"))
	settle(t, p, "", time.Second)
	time.Sleep(time.Second)
	if now := p.Render(); now != held {
		t.Errorf("forty lines onto a full history moved the reading:\nwas:\n%s\nnow:\n%s", held, now)
	}
	// And it was the history rolling that it held through, not a quiet minute.
	if _, above, _ := p.Seen(); above != 100 {
		t.Errorf("the history holds %d lines, so nothing was dropped", above)
	}
}

// What is being read can be dropped off the top of the history altogether,
// which is the one thing nothing can hold. The oldest line there is is as far
// back as anyone can go, and that is where the view is left — not sliding on
// through the middle of what is arriving.
func TestTheViewStopsAtTheOldestLineThereIs(t *testing.T) {
	p := sh(t, `i=1; while [ $i -le 60 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done
read go; i=61; while [ $i -le 400 ]; do printf 'line %d\r\n' $i; i=$((i+1)); done; sleep 30`, 40, 6)
	hold(p, 50)
	settle(t, p, "line 60", 5*time.Second)

	p.Back(30)
	p.Type([]byte("go\r"))
	settle(t, p, "", time.Second)
	time.Sleep(time.Second)

	if back, above, _ := p.Seen(); back != above {
		t.Errorf("the view is %d lines up a history of %d, want the top of it", back, above)
	}
}

// Who the wheel belongs to is the program's to say. One that asks the terminal
// for the mouse draws its own screen and scrolls itself; one that never asked
// would only be typed at by a wheel.
func TestTheCellSaysWhetherTheProgramTookTheMouse(t *testing.T) {
	p := sh(t, `printf 'plain\r\n'; read a; printf '\033[?1002h'; read b; printf '\033[?1002l'; sleep 30`, 40, 6)
	settle(t, p, "plain", 5*time.Second)
	if p.TakesTheMouse() {
		t.Error("a program that asked for nothing was given the mouse")
	}

	p.Type([]byte("go\r"))
	waitFor(t, p, func() bool { return p.TakesTheMouse() }, "the program to take the mouse")
	p.Type([]byte("go\r"))
	waitFor(t, p, func() bool { return !p.TakesTheMouse() }, "the program to give it back")
}

func waitFor(t *testing.T, p *Pane, ready func() bool, what string) {
	t.Helper()
	for i := 0; i < 60; i++ {
		if ready() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("waited three seconds for %s", what)
}

// What the cell is told about its terminal is the host's to decide: a session
// that takes the screen and the mouse away leaves the host nothing to draw a
// scrollbar from.
func TestTheHostSetsTheEnvironment(t *testing.T) {
	p, err := Start("probe", "", []string{"/bin/sh", "-c", `printf 'said %s\r\n' "$BELP_PROBE"; sleep 30`},
		40, 6, "BELP_PROBE=inside")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	settle(t, p, "said inside", 5*time.Second)
}

// A screen built again at a new size wraps where the new margin is.
//
// Nothing in the screen it was kept in says which rows broke because the line
// hit the old margin and which because the program sent a newline, so the rows
// cannot be unwrapped after the fact: the stream that drew them is the only
// thing that knows, and Reflow plays it again.
func TestAScreenIsBuiltAgainAtTheNewWidth(t *testing.T) {
	sh := os.Getenv("BELP_TEST_SH")
	if sh == "" {
		sh = "/bin/sh"
	}
	// One line of 90 characters, drawn on a screen 100 wide: it fits.
	said := strings.Repeat("abcdefghij", 9)
	p, err := Start("test", t.TempDir(), []string{sh, "-c", "printf '%s\\n' " + said}, 100, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	until(t, func() bool { return strings.Contains(p.Render(), said[:40]) })

	if lines := wrapped(p, said[:40]); lines != 1 {
		t.Fatalf("at 100 wide it took %d lines", lines)
	}

	// Half as wide: the line has to break in two, and nothing it is kept in
	// can work that out — only the stream can.
	p.Resize(50, 10)
	p.Reflow()
	until(t, func() bool { return wrapped(p, said[:40]) == 2 })
	if n := wrapped(p, said[:40]); n != 2 {
		t.Errorf("at 50 wide the 90-character line takes %d lines, want 2", n)
	}
	// And the text itself survived the rebuild.
	if flat := strings.ReplaceAll(ansi.Strip(p.Render()), "\n", ""); !strings.Contains(flat, said) {
		t.Error("the line did not come through the rebuild whole")
	}
}

// wrapped is how many lines of the screen the text starting with head runs over.
func wrapped(p *Pane, head string) int {
	rows := strings.Split(ansi.Strip(p.Render()), "\n")
	for i, row := range rows {
		if !strings.HasPrefix(strings.TrimSpace(row), head) {
			continue
		}
		n := 1
		for j := i + 1; j < len(rows) && strings.TrimSpace(rows[j]) != ""; j++ {
			n++
		}
		return n
	}
	return 0
}

func until(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// What the program asked the terminal for survives the rebuild.
//
// The reset that precedes it turns every mode off, and the tape is only the
// last few megabytes: bracketed paste is asked for once, when a session starts,
// so a resize an hour later would have quietly stopped marking pastes — which
// is the whole of how a session knows a return inside pasted text is a newline
// rather than the key that sends.
func TestTheModesSurviveTheRebuild(t *testing.T) {
	sh := os.Getenv("BELP_TEST_SH")
	if sh == "" {
		sh = "/bin/sh"
	}
	// The mode first, then more output than the tape holds, so the sequence
	// that asked for it is long gone by the time the window is resized — which
	// is the case this exists for, and a test whose tape still carries it
	// proves nothing.
	flood := "awk 'BEGIN{s=sprintf(\"%070d\",0); for(i=0;i<40000;i++) print i s}'"
	p, err := Start("test", t.TempDir(),
		[]string{sh, "-c", "printf '\\033[?2004h'; " + flood + "; sleep 5"}, 80, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	until(t, p.TakesPastes)
	if !p.TakesPastes() {
		t.Fatal("the program asked for bracketed paste and the pane did not hear it")
	}
	// Wait until the flood has pushed it off the tape.
	until(t, func() bool { return p.taped2() })
	if !p.taped2() {
		t.Skip("the tape never filled, so there is nothing to lose")
	}

	p.Resize(60, 10)
	p.Reflow()
	if !p.TakesPastes() {
		t.Error("the rebuild lost it, so a paste would arrive as keystrokes again")
	}
}

// taped2 reports whether the tape has wrapped round, so that what was written
// before it is gone.
func (p *Pane) taped2() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.tape) >= tape && !strings.Contains(string(p.tape[:4096]), "\x1b[?2004h")
}
