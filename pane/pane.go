// Package pane runs a program in a cell.
//
// A cell is a pseudo-terminal of its own, so what runs in it is the real
// program: sillage keeps its search field and its preview because it is
// sillage, not an impression of it. Nothing it runs has to know it is in a
// grid, and nothing here has to know what it is running.
package pane

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// Pane is one running program and the screen it has drawn.
type Pane struct {
	Label string
	Argv  []string
	Dir   string

	mu    sync.Mutex
	term  *vt.Emulator
	tty   *os.File
	cmd   *exec.Cmd
	ended bool
	drawn bool
	// first is when it drew, which is where the wait for the rest of its modes
	// is counted from.
	first time.Time
	err   error
	// back is how far up its own history the view is held, in lines. Zero is
	// the bottom, which is where a terminal normally is.
	back int
	// holding is the two lines the held view starts with, which is what it is
	// held by. Counting is not enough: see follow.
	holding [2]string
	// tape is what the program has drawn lately, kept so that the screen can be
	// built again at another size.
	//
	// Nothing else can re-wrap it. The emulator holds rows of cells and records
	// nothing about which of them ended because the program sent a newline and
	// which because the line hit the right margin, so a row written at one
	// width cannot be unwrapped at another — the only thing that knows is the
	// stream that produced it, played again into a screen of the new size.
	tape []byte
	// wide and high are the size it was last told to be, for building the
	// screen again.
	wide, high int
	// said is where the last read left off inside a string sequence.
	said title
	// shown is whether the program wants its cursor seen. It hides it while it
	// is drawing and asks for it back at the prompt.
	shown bool
	// mouse is whether the program has asked the terminal for the mouse. One
	// that has does its own scrolling and gets the wheel; one that has not
	// would only be typed at by it.
	mouse bool
	// pastes is whether it asked to be told when a run of text is a paste
	// rather than typing.
	pastes bool

	// wake carries one signal per change, coalesced: a program that redraws a
	// hundred times a second must not make the grid draw a hundred times.
	wake chan struct{}
}

// Start runs argv in a cell of the given size, with env — "NAME=value" — set
// over what this process was given.
func Start(label, dir string, argv []string, w, h int, env ...string) (*Pane, error) {
	p := &Pane{Label: label, Argv: argv, Dir: dir, wake: make(chan struct{}, 1)}
	if len(argv) == 0 {
		return nil, errNothing
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	// TERM matters: a program asks what it is talking to before it decides what
	// it may draw, and a name it does not know means the plainest possible
	// output — which is not what any of these apps look like.
	cmd.Env = append(append(os.Environ(), "TERM=xterm-256color"), env...)

	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
	if err != nil {
		return nil, err
	}
	p.cmd, p.tty, p.term = cmd, tty, vt.NewEmulator(w, h)
	p.wide, p.high = w, h
	p.shown = true
	// Told rather than asked: the emulator keeps the cursor's visibility to
	// itself, and this is called from inside the write that changed it, which
	// is where the lock is already held.
	p.term.SetCallbacks(vt.Callbacks{
		CursorVisibility: func(on bool) { p.shown = on },
		EnableMode:       func(m ansi.Mode) { p.noted(m, true) },
		DisableMode:      func(m ansi.Mode) { p.noted(m, false) },
	})

	go p.read()
	go p.answer()
	go p.reap()
	return p, nil
}

// read takes what the program drew into the screen it drew it on.
func (p *Pane) read() {
	buf := make([]byte, 32*1024)
	for {
		n, err := p.tty.Read(buf)
		if n > 0 {
			p.mu.Lock()
			was := p.term.ScrollbackLen()
			said := p.said.keep(buf[:n])
			p.taped(said)
			_, _ = p.term.Write(said)
			if p.back > 0 {
				// Somebody is reading further up. What arrived pushed that many
				// lines off the top, so holding the same distance from the
				// bottom would drag the text they are reading out from under
				// them.
				p.back = clamp(p.back+p.term.ScrollbackLen()-was, 0, p.term.ScrollbackLen())
				p.follow()
			}
			if !p.drawn {
				p.drawn, p.first = true, time.Now()
			}
			p.mu.Unlock()
			p.signal()
		}
		if err != nil {
			return
		}
	}
}

// answer is the half that is easy to leave out and impossible to work without.
//
// A program asks the terminal questions — what colour is the background, what
// do you support — and waits for the replies before it draws anything at all.
// A host that only listens leaves it waiting for ever, having written about a
// dozen bytes and nothing else.
func (p *Pane) answer() {
	buf := make([]byte, 4096)
	for {
		// Deliberately outside the lock: this read blocks until the emulator has
		// something to say, and holding the pane's lock across it means nothing
		// can ever draw. The emulator guards its own buffers.
		n, err := p.term.Read(buf)
		if n > 0 {
			if _, werr := p.tty.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (p *Pane) reap() {
	err := p.cmd.Wait()
	p.mu.Lock()
	p.ended, p.err = true, err
	p.mu.Unlock()
	p.signal()
}

func (p *Pane) signal() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Changed wakes whoever is drawing, once per batch of changes.
func (p *Pane) Changed() <-chan struct{} { return p.wake }

// Render is the screen as it stands, styling and all — or, where the view has
// been moved up, the lines of that height ending where it was left.
func (p *Pane) Render() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.back <= 0 {
		return p.term.Render()
	}
	// The history and the screen read as one column of lines: what scrolled off
	// the top, then what is still on it.
	sb := p.term.Scrollback()
	top := sb.Len() - p.back
	w, h := p.term.Width(), p.term.Height()
	out := make([]string, 0, h)
	for y := top; y < top+h; y++ {
		if y < sb.Len() {
			out = append(out, sb.Line(y).Render())
			continue
		}
		line := make(uv.Line, w)
		for x := 0; x < w; x++ {
			if c := p.term.CellAt(x, y-sb.Len()); c != nil {
				line[x] = *c
			}
		}
		out = append(out, line.Render())
	}
	return strings.Join(out, "\n")
}

// follow puts the view back on the lines it was on, and is the whole of what
// makes it stay put.
//
// Counting the history is not enough. It holds ten thousand lines and no more:
// after that the oldest is dropped for each new one, so its length stops
// growing while the lines in it keep moving down — and the reading slides a
// line at a time towards whatever has just been written. Which is to say it
// worked until the session was long enough for anybody to want to scroll.
//
// So the view is held by what it says rather than by where it was: the two
// lines it starts with are looked for again around where they were, and the
// count is set to wherever they turn out to be now.
func (p *Pane) follow() {
	sb := p.term.Scrollback()
	if p.holding[0] == "" && p.holding[1] == "" {
		p.mark()
		return
	}
	at := sb.Len() - p.back
	if p.says(sb, at) {
		return
	}
	// Near where it was first, which is where it is on any ordinary write.
	for k := 1; k <= chase; k++ {
		if p.says(sb, at-k) {
			p.back += k
			return
		}
		if p.says(sb, at+k) {
			p.back -= k
			return
		}
	}
	// Then anywhere at all, for a program that rewrote half its screen between
	// two frames.
	for i := range sb.Len() {
		if p.says(sb, i) {
			p.back = sb.Len() - i
			return
		}
	}
	if sb.Len() >= sb.MaxLines() {
		// The history is full and what they were reading has been dropped off
		// the top of it. The oldest line there is is as far back as anyone can
		// go, so that is where they are left.
		p.back = sb.Len()
	}
	// And whatever is there now is what is being read.
	p.mark()
}

// chase is how far either way the lines are looked for before the whole history
// is. A write pushes a handful of lines at the outside.
const chase = 256

// says reports whether the history at that line still begins what is being
// read.
func (p *Pane) says(sb *vt.Scrollback, at int) bool {
	if at < 0 || at >= sb.Len() {
		return false
	}
	if sb.Line(at).String() != p.holding[0] {
		return false
	}
	// Two lines rather than one: a screen of a session has blank lines and
	// repeated ones in it, and either would be found in the wrong place.
	if at+1 < sb.Len() {
		return sb.Line(at+1).String() == p.holding[1]
	}
	return true
}

// mark takes down what the view now starts with, for follow to look for.
func (p *Pane) mark() {
	p.holding = [2]string{}
	if p.back <= 0 {
		return
	}
	sb := p.term.Scrollback()
	at := sb.Len() - p.back
	if at >= 0 && at < sb.Len() {
		p.holding[0] = sb.Line(at).String()
	}
	if at+1 >= 0 && at+1 < sb.Len() {
		p.holding[1] = sb.Line(at + 1).String()
	}
}

// Back moves the view n lines up what the program has written, negative to come
// back down, and says how far up it now is.
//
// Scrolling is the host's and not the program's: a wheel is only ever sent on
// to something that asked the terminal for the mouse, and what has gone off the
// top of the screen is not the program's to remember. It is kept here, where
// the lines are.
func (p *Pane) Back(n int) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.back = clamp(p.back+n, 0, p.term.ScrollbackLen())
	p.mark()
	// signal takes no lock of its own, so it is safe from in here.
	p.signal()
	return p.back
}

// Home puts the view back at the bottom, where what is happening is.
func (p *Pane) Home() {
	p.mu.Lock()
	moved := p.back != 0
	p.back = 0
	p.holding = [2]string{}
	p.mu.Unlock()
	if moved {
		p.signal()
	}
}

// Away reports how far up its history the view is being held.
func (p *Pane) Away() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.back
}

// To holds the view that many lines up, wherever it was before, and says where
// it ended up. What a bar dragged down the side of the cell works in.
func (p *Pane) To(back int) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.back = clamp(back, 0, p.term.ScrollbackLen())
	p.mark()
	p.signal()
	return p.back
}

// Seen is what a scrollbar is drawn from: how far up the history the view is
// being held, how many lines are above the screen altogether, and how many the
// screen itself holds.
func (p *Pane) Seen() (back, above, height int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.back, p.term.ScrollbackLen(), p.term.Height()
}

func clamp(n, low, high int) int {
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}

// Type forwards input, as the bytes a terminal would have sent — which is what
// the program is written to read.
func (p *Pane) Type(b []byte) {
	if len(b) == 0 {
		return
	}
	p.mu.Lock()
	tty := p.tty
	p.mu.Unlock()
	if tty != nil {
		_, _ = tty.Write(b)
	}
}

// Wide is how many columns the session has, which is where its lines wrap.
//
// Asked rather than assumed by the caller: a row that reaches this column was
// broken by the terminal and not by the program writing to it, and that is the
// difference between copying a command and copying half of one.
func (p *Pane) Wide() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.wide
}

// Resize tells both halves: the emulator, which is what gets drawn, and the
// program, which is what decides what to draw.
func (p *Pane) Resize(w, h int) {
	if w < 1 || h < 1 {
		return
	}
	p.mu.Lock()
	p.term.Resize(w, h)
	p.wide, p.high = w, h
	tty := p.tty
	p.mu.Unlock()
	if tty != nil {
		_ = pty.Setsize(tty, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
	}
	p.signal()
}

// taped keeps what was drawn, up to what is worth keeping.
//
// A ring rather than everything: a session draws megabytes over an afternoon,
// and what a resize has to put right is what can still be scrolled back to.
func (p *Pane) taped(b []byte) {
	p.tape = append(p.tape, b...)
	if n := len(p.tape) - tape; n > 0 {
		p.tape = append(p.tape[:0], p.tape[n:]...)
	}
}

// tape is how much of what was drawn is kept for that.
const tape = 2 << 20

// Reflow draws everything again into a screen of the size this now is.
//
// Called once the resizing has stopped rather than on every step of it: a
// window being dragged sends a size a frame, and playing the tape back for each
// would be the whole afternoon's output parsed thirty times a second.
//
// What it fixes is the lines the terminal wrapped: at the old width they break
// where the old margin was, and nothing in the screen they are kept in says
// that is why they broke. What it cannot fix is a line the program wrapped
// itself before sending it — that one is wrapped in the stream, and only the
// program redrawing it will put it right, which is what it does with the live
// screen when it hears the window changed.
func (p *Pane) Reflow() {
	p.mu.Lock()
	if len(p.tape) == 0 || p.wide < 1 || p.high < 1 {
		p.mu.Unlock()
		return
	}
	// The same emulator, reset and written to again — not a new one swapped in.
	// The loop that carries the program's answers back reads this one outside
	// the lock, on purpose, because that read blocks: changing what it points
	// at underneath it is a race, and the answers already waiting in it would
	// go missing with the old one.
	mouse, pastes, shown := p.mouse, p.pastes, p.shown
	_, _ = p.term.Write([]byte("\x1bc")) // RIS: start from nothing
	p.term.Resize(p.wide, p.high)
	// What the program had asked for, as the whole history has it. The tape is
	// the last few megabytes, so a mode set before that and never mentioned
	// again — bracketed paste is set once, at startup — would be lost to the
	// reset. Put back first, so anything the tape does say about them still
	// lands on top.
	p.mouse, p.pastes, p.shown = mouse, pastes, shown
	_, _ = p.term.Write(p.tape)
	p.back = clamp(p.back, 0, p.term.ScrollbackLen())
	p.mu.Unlock()
	p.signal()
}

// Cursor is where the program put it, for the one cell that has the focus.
func (p *Pane) Cursor() (x, y int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pos := p.term.CursorPosition()
	return pos.X, pos.Y
}

// noted takes down the modes that change how this cell talks to the program.
// Called from inside the write that carried them, where the lock is held.
func (p *Pane) noted(m ansi.Mode, on bool) {
	switch m {
	case ansi.ModeMouseX10, ansi.ModeMouseNormal, ansi.ModeMouseHighlight,
		ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent:
		p.mouse = on
	case ansi.ModeBracketedPaste:
		p.pastes = on
	}
}

// Paste hands the program a run of text the way a terminal hands it one:
// wrapped in the marks that say where it starts and ends, where the program
// asked to be told. One that did not gets the text as it is.
//
// Which matters for what comes after it. A program that watches how fast
// characters arrive takes a run of them for a paste whatever it is told — and
// a return inside a paste is a newline in the box, not the key that sends it.
// Closing the paste first leaves the return a keystroke again.
func (p *Pane) Paste(text string) {
	if text == "" {
		return
	}
	p.mu.Lock()
	marks := p.pastes
	p.mu.Unlock()
	if marks {
		p.Type([]byte("\x1b[200~" + text + "\x1b[201~"))
		return
	}
	p.Type([]byte(text))
}

// TakesTheMouse reports whether the program asked the terminal for it.
//
// Which decides who the wheel belongs to. A program drawing a whole screen of
// its own — the alternate screen, its own scrolling — asks for the mouse and
// scrolls itself, and there is no history here to scroll instead: what it has
// drawn over is gone, the way it is in any terminal. One that never asked has
// its lines kept here, and the wheel moves this window's view of them.
func (p *Pane) TakesTheMouse() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mouse
}

// TakesPastes reports whether the program asked to be told where a paste
// starts and ends. The counterpart of TakesTheMouse, and the same question: a
// terminal only marks a paste for a program that said it reads the marks.
func (p *Pane) TakesPastes() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pastes
}

// Caret is where the cursor is and whether it should be seen at all: the
// program hides it while it draws, and a view held up the history is not
// showing the line it is on.
func (p *Pane) Caret() (x, y int, shown bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.shown || p.back > 0 {
		return 0, 0, false
	}
	pos := p.term.CursorPosition()
	return pos.X, pos.Y, true
}

// Ended reports whether the program has finished, and why.
func (p *Pane) Ended() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ended, p.err
}

// Close ends the program and lets go of the terminal.
func (p *Pane) Close() {
	p.mu.Lock()
	cmd, tty := p.cmd, p.tty
	p.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if tty != nil {
		_ = tty.Close()
	}
}

type nothingToRun struct{}

func (nothingToRun) Error() string { return "nothing to run" }

var errNothing = nothingToRun{}

// Tell says whether this is the program being typed into.
//
// Reported rather than imposed: what goes to the program is what a terminal
// sends when its window gains or loses the keyboard, and a program that asked to
// hear about it draws itself accordingly. One that did not ask carries on as it
// was, which is the right answer for a program with nothing to say on the matter.
//
// The state is set through its opposite first, because only a change is
// announced. A pane begins unfocused, so blurring one would otherwise be a
// repetition of what it already was and would tell the program nothing.
func (p *Pane) Tell(focused bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if focused {
		p.term.Blur()
		p.term.Focus()
		return
	}
	p.term.Focus()
	p.term.Blur()
}

// Ready reports whether the program is far enough along to be told things.
//
// Drawing is not enough on its own. The first bytes out of a program are the
// modes it wants — the alternate screen, focus reporting — and they need not
// all arrive in one write. Telling it where the keys are in the gap between two
// of them is telling nothing: the emulator has no instruction yet to pass the
// message on, and only a change is ever announced, so the moment is not
// repeated. A breath after the first output is enough, and costs one frame.
func (p *Pane) Ready() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.drawn && time.Since(p.first) > settling
}

// settling is that breath.
const settling = 400 * time.Millisecond
