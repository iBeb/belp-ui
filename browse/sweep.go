package browse

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Selecting anything on the screen, not only what is typed in the field.
//
// Mouse reporting is on, so the terminal draws no selection of its own over
// this window, and a list whose rows cannot be copied — a session's title, a
// path, a ticket's name — is a list you retype from. So a drag anywhere picks
// out what it passes over, as it would in any other window, and letting go
// puts it on the clipboard. A press that does not move is still the click it
// always was.

// sweep is a drag across the screen: where it started and where it has got to,
// in the window's own cells.
type sweep struct {
	x0, y0, x1, y1 int
	moved, done    bool
}

// ends is the sweep in reading order.
func (s sweep) ends() (int, int, int, int) {
	if s.y0 < s.y1 || (s.y0 == s.y1 && s.x0 <= s.x1) {
		return s.x0, s.y0, s.x1, s.y1
	}
	return s.x1, s.y1, s.x0, s.y0
}

// holds is which columns of row y are selected, if any.
func (s sweep) holds(y, width int) (int, int, bool) {
	fx, fy, tx, ty := s.ends()
	if !s.moved || y < fy || y > ty {
		return 0, 0, false
	}
	from, to := 0, width
	if y == fy {
		from = fx
	}
	if y == ty {
		to = tx + 1
	}
	to = min(to, width)
	return from, to, from < to
}

// sweeping takes a mouse event as part of a drag across the screen, and says
// whether it was one. A press is noted and left for the click it also is.
func (m Model) sweeping(msg tea.MouseMsg) (Model, tea.Cmd, bool) {
	if msg.Button != tea.MouseButtonLeft {
		return m, nil, false
	}
	switch msg.Action {
	case tea.MouseActionPress:
		m.sw = &sweep{x0: msg.X, y0: msg.Y, x1: msg.X, y1: msg.Y}
		return m, nil, false
	case tea.MouseActionMotion:
		if m.sw == nil || m.sw.done {
			return m, nil, false
		}
		s := *m.sw
		s.x1, s.y1, s.moved = msg.X, msg.Y, true
		m.sw = &s
		return m, nil, true
	case tea.MouseActionRelease:
		if m.sw == nil || m.sw.done || !m.sw.moved {
			m.sw = nil
			return m, nil, false
		}
		s := *m.sw
		s.done = true
		m.sw = &s
		return m, copyText(m.Swept()), true
	}
	return m, nil, false
}

// Swept is the text a drag across the screen has picked out, as it reads:
// without the colours, and without the blank the end of each row is padded to.
func (m Model) Swept() string {
	if m.sw == nil || !m.sw.moved {
		return ""
	}
	l := m.Layout()
	rows := strings.Split(m.drawn().Render(l, m.rows(l), m.previewLines(l)), "\n")
	var out []string
	for y, row := range rows {
		from, to, ok := m.sw.holds(y, lipgloss.Width(row))
		if !ok {
			continue
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(row, from, to)), " "))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// banded draws the sweep over what was drawn.
func (m Model) banded(view string) string {
	if m.sw == nil || !m.sw.moved {
		return view
	}
	rows := strings.Split(view, "\n")
	s := m.chrome.Styles.Selection
	for y, row := range rows {
		from, to, ok := m.sw.holds(y, lipgloss.Width(row))
		if !ok {
			continue
		}
		said := ansi.Strip(ansi.Cut(row, from, to))
		rows[y] = ansi.Truncate(row, from, "") + s.Render(said) + ansi.TruncateLeft(row, to, "")
	}
	return strings.Join(rows, "\n")
}
