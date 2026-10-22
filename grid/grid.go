package grid

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/pane"
	"github.com/iBeb/belp-ui/theme"
)

// frame is how often the grid redraws.
//
// A clock rather than a wake per change, because several programs redrawing at
// once would otherwise each cost a frame. Twenty a second is smooth to read and
// costs a render of the cells that actually changed.
var frame = 50 * time.Millisecond

type frameMsg struct{}

// Model is the grid: the arrangement, the programs in it, and which one has the
// keys.
type Model struct {
	styles theme.Styles
	spec   Spec
	panes  []*pane.Pane
	rects  []Rect
	// areas is what each cell was placed with, rects what is left for the
	// program once the gap to its neighbour is taken off. Both are kept: a
	// click is about which panel and so is answered by the area, a click
	// handed on is about where in the program and so is answered by the rect.
	areas []Rect
	specs []Spec

	focus         int
	width, height int
	started       bool
	// told is whether each program has been told where the keys are. It waits
	// until the program has drawn: a program says which modes it wants — focus
	// reporting among them — before it draws anything, so an empty screen means
	// it is not yet listening.
	told []bool
}

// New builds the grid over an arrangement. Nothing is started until the first
// size arrives: a program started into a cell of no known size draws itself for
// a terminal it is not in.
func New(s theme.Styles, spec Spec) Model {
	return Model{styles: s, spec: spec}
}

func (m Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(frame, func(time.Time) tea.Msg { return frameMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case frameMsg:
		m.announce()
		return m, tick()

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.arrange()
		return m, nil

	case tea.MouseMsg:
		i, ok := m.cellAt(msg.X, msg.Y)
		if !ok {
			return m, nil
		}
		// A press in another panel chooses it and goes no further: the click was
		// about which panel, and passing it on as well would act on a row in a
		// panel you had not been looking at.
		if !Wheel(msg) && msg.Action == tea.MouseActionPress && i != m.focus {
			m.refocus(i)
			return m, nil
		}
		// The wheel belongs to the panel under the pointer whatever has the
		// keys, which is how scrolling works everywhere else.
		if p := m.panes[i]; p != nil && !m.gone(i) {
			p.Type(SGR(msg, m.rects[i]))
		}
		return m, nil

	case tea.KeyMsg:
		// Two keys belong to the grid and nothing else: one to move between
		// cells and one to leave. Everything else is the program's, or the
		// search field in a cell could never receive a letter.
		switch msg.Type {
		case tea.KeyTab:
			m.refocus((m.focus + 1) % max(1, len(m.panes)))
			return m, nil
		case tea.KeyShiftTab:
			m.refocus((m.focus - 1 + max(1, len(m.panes))) % max(1, len(m.panes)))
			return m, nil
		}
		// ^C leaves belp, and is not passed on.
		//
		// In a terminal on its own it means stop this thing, and these apps take
		// that as leave — which in a grid means closing panels one at a time
		// until nothing is left and there is no way out at all. Taken here it is
		// the one key that always works, which is what it is for.
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyCtrlQ {
			m.Close()
			return m, tea.Quit
		}
		// A cell whose program has finished belongs to the grid again: ^C is how
		// you stop a thing in a terminal and these apps take it as leave, so a
		// panel that has been left has to be startable without leaving the grid.
		if m.gone(m.focus) {
			if msg.Type == tea.KeyEnter {
				m.restart(m.focus)
			}
			return m, nil
		}
		if p := m.focused(); p != nil {
			p.Type(Bytes(msg))
		}
		return m, nil
	}
	return m, nil
}

// arrange works out the cells and either starts the programs or tells them
// their new size.
func (m *Model) arrange() {
	placed := Place(m.spec, Rect{W: m.width, H: m.height})
	m.rects, m.areas, m.specs = nil, nil, nil
	for _, p := range placed {
		m.rects = append(m.rects, inner(p.Rect, m.width, m.height))
		m.areas = append(m.areas, p.Rect)
		m.specs = append(m.specs, p.Spec)
	}

	if !m.started {
		for i, s := range m.specs {
			r := m.rects[i]
			p, err := pane.Start(s.Title(), s.Dir, s.Run, r.W, r.H)
			if err != nil {
				p = nil
			}
			m.panes = append(m.panes, p)
		}
		m.started = true
		m.told = make([]bool, len(m.panes))
		return
	}
	for i, p := range m.panes {
		if p != nil && i < len(m.rects) {
			p.Resize(m.rects[i].W, m.rects[i].H)
		}
	}
}

// inner is the room a cell gives its program: all of it, less a blank column
// where it has a neighbour to the right and a blank row where it has one below.
//
// Nothing is drawn around a program here, the way nothing is drawn around the
// session in a chantier. What a border was still doing past the look of it was
// saying where one panel stopped and the next began, and two apps that each
// rule a line under their own header meet in one line across the terminal. The
// gap says it instead, and says it with cells nobody has to style.
//
// Given up on the near side rather than the far one, so a cell flush against
// the edge of the screen keeps every column it was placed with: the gap is
// between two panels, and there is no panel past the edge.
func inner(r Rect, width, height int) Rect {
	w, h := r.W, r.H
	if r.X+r.W < width {
		w--
	}
	if r.Y+r.H < height {
		h--
	}
	return Rect{X: r.X, Y: r.Y, W: max(w, 1), H: max(h, 1)}
}

// refocus moves the keys, and tells both programs so.
//
// Told rather than left to guess: a panel that knows the keys have gone can draw
// itself quieter, and only the one being typed into should look like it.
func (m *Model) refocus(to int) {
	if to == m.focus || to < 0 || to >= len(m.panes) {
		return
	}
	if p := m.panes[m.focus]; p != nil {
		p.Tell(false)
	}
	m.focus = to
	if p := m.panes[to]; p != nil {
		p.Tell(true)
	}
}

// announce tells a program where the keys are, once it is up enough to hear.
func (m *Model) announce() {
	for i, p := range m.panes {
		if p == nil || i >= len(m.told) || m.told[i] || !p.Ready() {
			continue
		}
		p.Tell(i == m.focus)
		m.told[i] = true
	}
}

// gone reports whether a cell's program has finished.
func (m Model) gone(i int) bool {
	if i < 0 || i >= len(m.panes) {
		return true
	}
	if m.panes[i] == nil {
		return true
	}
	ended, _ := m.panes[i].Ended()
	return ended
}

// restart runs a cell's program again, at whatever size the cell is now.
func (m *Model) restart(i int) {
	if i < 0 || i >= len(m.specs) {
		return
	}
	if m.panes[i] != nil {
		m.panes[i].Close()
	}
	r := m.rects[i]
	p, err := pane.Start(m.specs[i].Title(), m.specs[i].Dir, m.specs[i].Run, r.W, r.H)
	if err != nil {
		p = nil
	}
	m.panes[i] = p
}

// cellAt is the cell a point is in, measured against what it was placed with
// rather than what it draws in: the blank column beside a panel is the panel's
// own, so a click that lands a cell wide of its last character still chooses
// it rather than nothing. The cells tile the screen, so every point is in one.
func (m Model) cellAt(x, y int) (int, bool) {
	for i, r := range m.areas {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return i, true
		}
	}
	return 0, false
}

func (m Model) focused() *pane.Pane {
	if m.focus < 0 || m.focus >= len(m.panes) {
		return nil
	}
	return m.panes[m.focus]
}

// Close ends every program. A grid that goes away without doing this leaves
// them running against a terminal nobody is reading.
func (m Model) Close() {
	for _, p := range m.panes {
		if p != nil {
			p.Close()
		}
	}
}

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 || len(m.panes) == 0 {
		return ""
	}
	// Rows written into rather than panels joined: the cells are placed by
	// coordinate, and joining them would mean rebuilding the arrangement a
	// second time in a different way.
	rows := make([]string, m.height)

	for i, p := range m.panes {
		r := m.rects[i]
		var body []string
		switch {
		case m.gone(i):
			// Not the frozen last screen: that one looks alive and answers
			// nothing, which is the worst of both.
			body = m.epitaph(i, r)
		case p != nil:
			body = strings.Split(p.Render(), "\n")
		}
		for j := 0; j < r.H; j++ {
			line := ""
			if j < len(body) {
				// Cut rather than padded: what follows on the row is the next
				// panel, and place pads up to wherever that starts.
				line = cut(body[j], r.W)
			}
			if y := r.Y + j; y >= 0 && y < m.height {
				rows[y] = place(rows[y], r.X, line, m.width)
			}
		}
	}
	return strings.Join(rows, "\n")
}

// epitaph is what a cell says once its program has finished.
func (m Model) epitaph(i int, r Rect) []string {
	s := m.styles
	what := m.specs[i].Title()
	why := "ended"
	if m.panes[i] != nil {
		if _, err := m.panes[i].Ended(); err != nil {
			why = "ended — " + err.Error()
		}
	}
	how := "↵ to run it again"
	if i != m.focus {
		how = "⇥ to it, then ↵ to run it again"
	}

	lines := make([]string, r.H)
	at := r.H / 2
	if at >= r.H {
		at = r.H - 1
	}
	if at >= 0 {
		lines[at] = centre(s.Desc.Render(what+" "+why), r.W)
	}
	if at+1 < r.H {
		lines[at+1] = centre(s.KeyName.Render(how), r.W)
	}
	return lines
}

// centre puts a line in the middle of the room there is.
func centre(text string, width int) string {
	gap := (width - lipgloss.Width(text)) / 2
	if gap < 0 {
		gap = 0
	}
	return strings.Repeat(" ", gap) + text
}

// place writes a line into a row at a column, keeping whatever was there.
func place(row string, x int, line string, width int) string {
	if row == "" {
		row = strings.Repeat(" ", width)
	}
	before := pad(cut(row, x), x)
	after := ""
	if end := x + lipgloss.Width(line); end < width {
		after = strings.Repeat(" ", width-end)
	}
	return before + line + after
}

func pad(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func cut(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
