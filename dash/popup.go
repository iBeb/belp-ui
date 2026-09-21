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
	// Heads names those columns — the switch, the state, the note — drawn
	// quietly above them. For a list where what the box means is not obvious
	// from the box: "included" is a different claim from "running", and a
	// column of ticks with no heading is read as whichever the reader expects.
	Heads [3]string
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
	// Fixed is a row that states something rather than switching it: the box is
	// drawn set and quiet, and the focus steps over it. For a list where some
	// entries are a choice and the rest are not — a switch you cannot throw
	// still has to be visible, or the list reads as shorter than it is.
	Fixed bool
	// Acts are controls at the end of the row: things done to what the row
	// names rather than to the list. Glyphs and not bands, because a row is a
	// line of text and a filled button in the middle of one reads as a heading.
	Acts []Button
}

// Stops is how many things in this popup the focus can land on.
func (p Popup) Stops() int {
	n := len(p.Buttons)
	for _, r := range p.Rows {
		if !r.Fixed {
			n++
		}
		n += len(r.Acts)
	}
	return n
}

// Stop is one landing place: a row's switch, the action at the end of a row, or
// a button.
//
// A type rather than arithmetic at the call site. A row used to be worth one
// stop, so the button under the focus was focus-len(Rows) everywhere — and the
// day a row grew a second control every one of those sums was wrong by a
// different amount.
type Stop struct {
	// Row is which row it belongs to, or -1 for a button.
	Row int
	// Act is which of that row's actions it is, or -1 for the row's own switch.
	Act int
	// Button is which button it is, or -1 for a row.
	Button int
}

// At is what landing i is. Out of range comes back as nothing at all, which is
// what a focus past the end of a shrinking list must not act on.
func (p Popup) At(i int) Stop {
	if i < 0 {
		return nowhere
	}
	for row, r := range p.Rows {
		if !r.Fixed {
			if i == 0 {
				return Stop{Row: row, Act: -1, Button: -1}
			}
			i--
		}
		for act := range r.Acts {
			if i == 0 {
				return Stop{Row: row, Act: act, Button: -1}
			}
			i--
		}
	}
	if i < len(p.Buttons) {
		return Stop{Row: -1, Act: -1, Button: i}
	}
	return nowhere
}

// nowhere is the stop a focus past the end of a shrinking list lands on.
var nowhere = Stop{Row: -1, Act: -1, Button: -1}

// Landed reports whether a stop is anything at all.
func (s Stop) Landed() bool { return s.Row >= 0 || s.Button >= 0 }

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
		if head := p.heads(s); head != "" {
			line(" " + head)
		}
		for i, r := range p.Rows {
			line(" " + p.row(s, r, p.focused(i), inner-4))
		}
	}
	if len(p.Buttons) > 0 {
		line("")
		// Centred in the box rather than pushed against its left edge: the row
		// is the one thing in a popup that is not a list of facts, and a row of
		// buttons hard against one margin reads as the start of another column.
		line(centre(Buttons(s, p.Buttons, p.At(p.Focus).Button, 0), inner-2))
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
func (p Popup) row(s theme.Styles, r Row, on rowFocus, width int) string {
	const gap = 2
	label := r.Toggle.Render(s, on == onSwitch)
	if r.Fixed {
		// Set, and quiet: it says what is so rather than offering to change it.
		label = s.Label.Render(toggleOn + " " + r.Toggle.Label)
	}
	say, style := r.Say, s.Value
	if r.Busy != "" {
		say, style = r.Busy, s.Warn
	}
	out := padTo(label, p.labelWidth()) + strings.Repeat(" ", gap) + style.Render(say)
	if note := p.noteWidth(); note > 0 {
		out = padTo(out, p.labelWidth()+gap+p.sayWidth()) + strings.Repeat(" ", gap) +
			padTo(s.Desc.Render(r.Note), note)
	}
	if len(r.Acts) > 0 {
		// Against the right-hand end of the block, not the left: the controls
		// line up down the list whatever the rows in between are called, and
		// the last cell of every row is the last control on it.
		out = padTo(out, p.labelWidth()+gap+p.sayWidth()+noteRoom(p, gap)+
			p.actWidth()-rowActWidth(r))
		for i, a := range r.Acts {
			out += strings.Repeat(" ", gap) + act(s, a, on == rowFocus(i))
		}
	}
	return cutTo(out, width)
}

// rowActWidth is what one row's controls take, gaps included.
func rowActWidth(r Row) int {
	const gap = 2
	n := 0
	for _, a := range r.Acts {
		n += gap + lipgloss.Width(a.Label)
	}
	return n
}

// noteRoom is what the note column and its gap take, or nothing where no row
// has one.
func noteRoom(p Popup, gap int) int {
	if note := p.noteWidth(); note > 0 {
		return gap + note
	}
	return 0
}

// act draws the control at the end of a row: the glyph in its own tone, or
// reversed out while the keys are on it, the way a selected thing is drawn
// everywhere else.
func act(s theme.Styles, b Button, focused bool) string {
	switch {
	case b.Off:
		return s.Label.Render(b.Label)
	case focused:
		return s.Selection.Render(b.Label)
	default:
		return b.Tone.style(s).Render(b.Label)
	}
}

// rowFocus is which control of a row the keys are on: onSwitch, one of its
// actions by index, or onNothing.
type rowFocus int

const (
	onNothing rowFocus = -2
	onSwitch  rowFocus = -1
)

// focused is which control of row i the keys are on, if any.
func (p Popup) focused(i int) rowFocus {
	at := p.At(p.Focus)
	if at.Row != i {
		return onNothing
	}
	return rowFocus(at.Act)
}

// heads is the column names, in the columns the rows use, or nothing where the
// caller named none.
func (p Popup) heads(s theme.Styles) string {
	if p.Heads == [3]string{} {
		return ""
	}
	const gap = 2
	out := padTo(s.Label.Render(p.Heads[0]), p.labelWidth()) +
		strings.Repeat(" ", gap) + s.Label.Render(p.Heads[1])
	if p.Heads[2] != "" {
		out = padTo(out, p.labelWidth()+gap+p.sayWidth()) +
			strings.Repeat(" ", gap) + s.Label.Render(p.Heads[2])
	}
	return out
}

// The three columns of the list, each as wide as its widest entry, so the states
// line up down the block.
func (p Popup) labelWidth() int {
	n := lipgloss.Width(p.Heads[0])
	for _, r := range p.Rows {
		n = max(n, r.Toggle.Wide())
	}
	return n
}

func (p Popup) sayWidth() int {
	n := lipgloss.Width(p.Heads[1])
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
	if note := p.noteWidth(); note > 0 {
		n += 2 + note
	}
	n += p.actWidth()
	return n
}

func (p Popup) noteWidth() int {
	n := lipgloss.Width(p.Heads[2])
	for _, r := range p.Rows {
		n = max(n, lipgloss.Width(r.Note))
	}
	return n
}

// actWidth is what the controls at the end of a row take, gaps included: every
// row is padded to the widest so that they line up down the block.
func (p Popup) actWidth() int {
	n := 0
	for _, r := range p.Rows {
		n = max(n, rowActWidth(r))
	}
	return n
}

// RowAt is which stop a click inside the list landed on: a row's switch, or the
// action at the end of that row, by where along the line the pointer was.
//
// A stop rather than a row index, because a row is worth one stop or two and
// only the popup knows which: a caller counting rows would press the switch of
// the row below the action it was aimed at.
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
	if p.Heads != [3]string{} {
		top++ // the line naming the columns
	}
	row := y - (s.Y + top)
	if row < 0 || row >= len(p.Rows) {
		return 0, false
	}
	if x < s.X+2 || x >= s.X+s.W-2 {
		return 0, false
	}

	// Which of the row's own controls, by where along it the pointer landed:
	// the action sits at the end of the line, everything before it is the
	// switch. A row with no switch answers only over its action.
	// Which of the row's own controls, by where along it the pointer landed:
	// the actions sit at the end of the line in order, everything before them
	// is the switch. A row with no switch answers only over its actions.
	r := p.Rows[row]
	want := -1 // the switch
	if at := x - (s.X + 2 + p.rowsWide() - rowActWidth(r)); at >= 0 {
		const gap = 2
		for i, a := range r.Acts {
			if at < gap+lipgloss.Width(a.Label) {
				want = i
				break
			}
			at -= gap + lipgloss.Width(a.Label)
		}
	}
	if r.Fixed && want < 0 {
		return 0, false
	}
	for i := 0; i < p.Stops(); i++ {
		if at := p.At(i); at.Row == row && at.Act == want {
			return i, true
		}
	}
	return 0, false
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
