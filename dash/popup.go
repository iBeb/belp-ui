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
	// Rows are things with a switch each, one to a line: a list of what a service
	// is made of, with a control per item rather than one control for the lot.
	// Drawn between the body and the buttons.
	Rows []Row
	// Buttons are the ways out. A popup with none is closed with escape; one
	// with some is a question, and the focused button is the answer.
	Buttons []Button
	// Focus runs over the rows first and then the buttons, because that is the
	// order they are drawn and the order the arrows walk them.
	Focus int
	// Note is one dim line under the buttons, for how to leave.
	Note string
	// Fixed is a popup that cannot be closed, and so is drawn without the mark
	// in its corner. The zero value is closable, because nearly everything is.
	Fixed bool
	// Shut overrides the mark in the corner. The default reads in any terminal;
	// an app that knows its own font may pass something rounder.
	Shut string
}

// Shut is the mark in the top-left corner: the thing a pointer goes to when a
// window is in the way.
//
// Square brackets rather than a glyph — ✕ and ⊗ are emoji in half the fonts and
// two cells wide in a border that has room for three.
const Shut = "[x]"

// Row is one line of a list with its own switch, and whatever is worth saying
// about it to the right.
type Row struct {
	Toggle Toggle
	// Say is the state, drawn after the label. Note is quieter still, for what
	// is true of the row rather than what it is doing.
	Say  string
	Note string
	// Busy replaces Say while the switch is being thrown, because a row that
	// says nothing for the several seconds a container takes looks broken.
	Busy string
}

// Stops is how many things in this popup the focus can land on.
func (p Popup) Stops() int { return len(p.Rows) + len(p.Buttons) }

// Wide is how much room the popup wants, given what is in it.
func (p Popup) Wide() int {
	// The mark takes room out of the top line, so a box sized without it is one
	// whose title no longer fits on it.
	want := lipgloss.Width(p.Title) + 4
	if !p.Fixed {
		// The mark, a space either side of it, and the border it hands back to.
		want += lipgloss.Width(p.shut()) + 3
	}
	for _, line := range p.Body {
		want = max(want, lipgloss.Width(line)+4)
	}
	if len(p.Rows) > 0 {
		want = max(want, p.rowsWide()+6)
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

	// A title is padded away from the corners; no title leaves the edge unbroken.
	// Spaces either side of nothing is a two-cell gap at the top left, which
	// reads as a dent in the border rather than as room for a word.
	// The corner, then the way out, then the border carries on with the title.
	// A window you can close says so where a pointer already goes to look. In
	// the colour of the frame rather than in red: the mark is part of the
	// window's chrome, and red on a border that is blue everywhere else reads
	// as a warning about the window rather than as the way out of it.
	shut := ""
	if !p.Fixed {
		// Held off the corner and off the border either side, so the mark is its
		// own thing: crowded against the line it reads as part of the frame, and
		// crowded against the words as a bullet belonging to the title.
		shut = " " + edge.Render(p.shut()) + " " + edge.Render("─")
	}
	head := ""
	if p.Title != "" {
		head = " " + s.Heading.Render(p.Title) + " "
	}
	rule := inner - lipgloss.Width(shut) - lipgloss.Width(head)
	if rule < 0 {
		head, rule = " ", max(0, inner-lipgloss.Width(shut)-1)
	}
	out := []string{edge.Render("╭") + shut + head +
		edge.Render(strings.Repeat("─", rule)+"╮")}

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
	if len(p.Rows) > 0 {
		if len(p.Body) > 0 {
			line("")
		}
		for i, r := range p.Rows {
			line(" " + p.row(s, r, i == p.Focus, inner-4))
		}
	}
	if len(p.Buttons) > 0 {
		line("")
		// Centred in the box rather than pushed against its left edge: the row
		// is the one thing in a popup that is not a list of facts, and a row of
		// buttons hard against one margin reads as the start of another column.
		line(centre(Buttons(s, p.Buttons, p.Focus-len(p.Rows), 0), inner-2))
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
// Worked out from the same numbers that drew them: one line, centred in the
// inner width, counted back from the bottom edge past the padding and the note.
// A button is a single filled line now rather than three lines in a box, and
// counting three put every click a row above the row it was aimed at.
func (s Spot) ButtonAt(p Popup, st theme.Styles, x, y int) (int, bool) {
	if len(p.Buttons) == 0 {
		return 0, false
	}
	rows := len(p.Render(st, s.W))
	// From the end: the bottom edge, the last pad, then the note and its pad
	// where there is one, and the buttons above that.
	line := rows - 3
	if p.Note != "" {
		line = rows - 5
	}
	if y != s.Y+line {
		return 0, false
	}

	at := x - (s.X + 2 + p.buttonsLeft(st, s.W))
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

// buttonsLeft is how far the row of buttons is inset from the content edge.
//
// The renderer centres the row, so the click has to know the same offset. Asked
// of one place rather than worked out twice: a click that lands on the button
// next to the one it was aimed at is what two copies of this arithmetic drifting
// apart looks like.
//
// The width is the one the box was drawn at, not the one it asked for. A popup
// rendered narrower than Wide() still centres its row inside what it got.
func (p Popup) buttonsLeft(st theme.Styles, width int) int {
	room := width - 4 - lipgloss.Width(Buttons(st, p.Buttons, -1, 0))
	if room <= 0 {
		return 0
	}
	return room / 2
}

// row draws one line of the list: the switch, then what it is doing.
func (p Popup) row(s theme.Styles, r Row, focused bool, width int) string {
	const gap = 2
	label := r.Toggle.Render(s, focused)
	say, style := r.Say, s.Value
	if r.Busy != "" {
		say, style = r.Busy, s.Warn
	}
	out := padTo(label, p.labelWidth()) + strings.Repeat(" ", gap) + style.Render(say)
	if r.Note != "" {
		out = padTo(out, p.labelWidth()+gap+p.sayWidth()) + strings.Repeat(" ", gap) +
			s.Desc.Render(r.Note)
	}
	return cutTo(out, width)
}

// The three columns of the list, each as wide as its widest entry, so the states
// line up down the block.
func (p Popup) labelWidth() int {
	n := 0
	for _, r := range p.Rows {
		n = max(n, r.Toggle.Wide())
	}
	return n
}

func (p Popup) sayWidth() int {
	n := 0
	for _, r := range p.Rows {
		n = max(n, lipgloss.Width(r.Say), lipgloss.Width(r.Busy))
	}
	return n
}

// rowsWide is how wide the list needs to be, worked out from the same column
// widths the rows are drawn at — one row's own measurements are not enough,
// because every row is padded to the widest.
func (p Popup) rowsWide() int {
	n := p.labelWidth() + 2 + p.sayWidth()
	note := 0
	for _, r := range p.Rows {
		note = max(note, lipgloss.Width(r.Note))
	}
	if note > 0 {
		n += 2 + note
	}
	return n
}

// RowAt is which of the popup's rows covers a point, for a click.
//
// Counted from the top, where the rows are: the body above them is a known
// number of lines and so is the padding, which is the only way a click and a
// drawing can be made to agree without one of them guessing.
func (s Spot) RowAt(p Popup, x, y int) (int, bool) {
	if len(p.Rows) == 0 {
		return 0, false
	}
	top := 2 + len(p.Body) // the border and the pad line, then the body
	if len(p.Body) > 0 {
		top++ // the blank between body and rows
	}
	i := y - (s.Y + top)
	if i < 0 || i >= len(p.Rows) {
		return 0, false
	}
	if x < s.X+2 || x >= s.X+s.W-2 {
		return 0, false
	}
	return i, true
}

// ShutAt reports whether a point is on the mark that closes the window.
//
// The top line, immediately after the corner: the same three cells the drawing
// puts it in, so the pointer and the picture cannot disagree.
func (s Spot) ShutAt(p Popup, x, y int) bool {
	if p.Fixed || s.H == 0 {
		return false
	}
	// The corner counts as part of it. A glyph is one cell wide, which is a hard
	// thing to hit with a pointer, and the corner beside it means nothing else.
	// The corner, the space before the mark, and the mark: a pointer aimed at a
	// small round thing lands around it as often as on it. Not the border after
	// the second space, which is border like any other.
	return y == s.Y && x >= s.X && x < s.X+2+lipgloss.Width(p.shut())
}

// shut is the mark this popup draws in its corner.
func (p Popup) shut() string {
	if p.Shut != "" {
		return p.Shut
	}
	return Shut
}
