package dash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// Popup is a box drawn over the dashboard: the detail behind a control, or the
// choice a control needs before it can act.
//
// Over rather than under. A panel that opens beneath a card pushes everything
// below it down the screen, so reading one card rearranges the others — and on
// a dashboard the arrangement is half of what you are reading.
type Popup struct {
	Title string
	// Body is already-styled lines. The popup pads and frames them and takes
	// nothing else on trust: what a fact is called and how it is coloured
	// belongs to whoever knows what the fact means.
	Body []string
	// Buttons are the ways out. A popup with none is closed with escape; one
	// with some is a question, and the focused button is the answer.
	Buttons []Button
	Focus   int
	// Note is one dim line under the buttons, for how to leave.
	Note string
}

// Wide is how much room the popup wants, given what is in it.
func (p Popup) Wide() int {
	want := lipgloss.Width(p.Title) + 4
	for _, line := range p.Body {
		want = max(want, lipgloss.Width(line)+4)
	}
	row := 0
	for i, b := range p.Buttons {
		if i > 0 {
			row++
		}
		row += b.Wide()
	}
	return max(want, row+4)
}

// Render draws the popup at a width.
func (p Popup) Render(s theme.Styles, width int) []string {
	inner := width - 2
	if inner < 8 {
		inner = 8
	}
	edge := s.Selected

	head := " " + s.Heading.Render(p.Title) + " "
	rule := inner - lipgloss.Width(head)
	if rule < 0 {
		head, rule = " ", inner-1
	}
	out := []string{edge.Render("╭") + head + edge.Render(strings.Repeat("─", rule)+"╮")}

	// Cut as well as padded: a body line longer than the box runs straight
	// through the right-hand border, and a popup that cannot hold its own frame
	// is worse than one that says less.
	line := func(text string) {
		out = append(out, edge.Render("│")+" "+padTo(cutTo(text, inner-2), inner-2)+" "+
			edge.Render("│"))
	}
	line("")
	for _, b := range p.Body {
		line(b)
	}
	if len(p.Buttons) > 0 {
		line("")
		for _, row := range Buttons(s, p.Buttons, p.Focus, 0) {
			line(row)
		}
	}
	if p.Note != "" {
		line("")
		line(s.Desc.Render(cutTo(p.Note, inner-2)))
	}
	line("")
	out = append(out, edge.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return out
}

// Spot is where a popup was drawn, so a click can be tested against what it
// covers rather than against where it was last time.
type Spot struct{ X, Y, W, H int }

// Over places a box over a screen, centred, and says where it landed.
//
// The lines under it are replaced rather than blended: a terminal has no alpha,
// and a box you can read the dashboard through is a box nobody can read.
func Over(lines []string, box []string, width, height int) ([]string, Spot) {
	if len(box) == 0 {
		return lines, Spot{}
	}
	w := 0
	for _, line := range box {
		w = max(w, lipgloss.Width(line))
	}
	left := max(0, (width-w)/2)
	top := max(0, (height-len(box))/2)

	out := make([]string, len(lines))
	copy(out, lines)
	for i, line := range box {
		y := top + i
		if y < 0 || y >= len(out) {
			continue
		}
		out[y] = strings.Repeat(" ", left) + line
	}
	return out, Spot{X: left, Y: top, W: w, H: len(box)}
}

// Holds reports whether a point is inside the popup.
func (s Spot) Holds(x, y int) bool {
	return s.H > 0 && x >= s.X && x < s.X+s.W && y >= s.Y && y < s.Y+s.H
}

// ButtonAt is which of the popup's buttons covers a point, for a click.
//
// Worked out from the same numbers that drew them: the buttons sit on the three
// lines above the note and the bottom padding, and each is as wide as it asked
// to be.
func (s Spot) ButtonAt(p Popup, st theme.Styles, x, y int) (int, bool) {
	if len(p.Buttons) == 0 {
		return 0, false
	}
	rows := len(p.Render(st, s.W))
	// The buttons are three lines, sitting above the note and the last padding
	// line and the bottom edge.
	bottom := rows - 2
	if p.Note != "" {
		bottom -= 2
	}
	top := bottom - 3
	if y < s.Y+top || y >= s.Y+bottom {
		return 0, false
	}
	at := x - (s.X + 2)
	for i, b := range p.Buttons {
		if at < 0 {
			return 0, false
		}
		if at < b.Wide() {
			return i, true
		}
		at -= b.Wide() + 1
	}
	return 0, false
}
