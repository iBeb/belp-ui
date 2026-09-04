package browse

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// Focus is which band the keyboard is talking to.
//
// The list by default. The search field is the row above the first row and the
// filter bar the row above that, so ↑ and ↓ are the only way to move.
type Focus int

const (
	FocusList Focus = iota
	FocusSearch
	FocusFilters

	// FocusPrompt is a question the app asked — a title to type, a name to
	// confirm. It is modal on purpose: while one is open the arrows must not
	// move a list underneath it, because the answer is about the row the
	// question was asked of.
	FocusPrompt

	// FocusConfirm is a question with two answers, drawn over the middle of the
	// screen rather than in a band. A question that has to be answered before
	// anything else happens should look like one: a line in the footer reads as
	// one more hint, and the row it is about is still sitting there under the
	// cursor as though nothing were pending.
	FocusConfirm

	// FocusMenu is an open Menu group. Modal like the others: the arrows are
	// walking the list of members, so nothing underneath may move.
	FocusMenu
)

// Option is one filter chip.
//
// Selected and Focused are different things and both have to be visible at once:
// what the filter is set to, and where the cursor is sitting. Conflating them
// means you cannot see what you are about to toggle.
type Option struct {
	Label    string
	Selected bool
	Focused  bool
}

// Group is a set of chips that belong together — the kinds of activity, the
// states, the date range.
//
// A group with nothing selected is a group that constrains nothing. That is the
// only sensible reading: the alternative is that clearing a filter hides
// everything, so the empty bar would show an empty list.
type Group struct {
	// Label is drawn before the chips, dimmed. Empty for a group that speaks
	// for itself.
	Label   string
	Options []Option

	// Exclusive groups hold one answer at a time — a date range is one window,
	// not several. Selecting a chip in one clears its siblings, and selecting
	// the chip that is already on leaves it on: there is no useful state where
	// an exclusive group has no answer.
	Exclusive bool

	// Menu collapses the group to a single chip that opens a list.
	//
	// For a set whose members are data rather than design: colleagues, projects,
	// repositories. Laid out as chips they would push every group after them off
	// a narrow terminal, and the bar would be a different width for each person
	// using it. One chip says how many are on; the list says which.
	Menu bool

	// Focused is the bar's cursor sitting on this menu's chip. Set by the model
	// as it draws, exactly as Option.Focused is, and meaningless on a group that
	// is not a Menu — its own chips carry the cursor themselves.
	Focused bool
}

// Key is one footer hint.
type Key struct {
	Name string // "↵", "^R"
	Desc string // "open", "review"
}

// Prompt is a question being asked of whatever the cursor is on.
//
// Drawn over the list, in the same frame as the yes-or-no question. A question
// in the footer borrows the row the key hints were using and still leaves the
// screen looking like nothing is pending: the list sits there under the cursor
// as though the next arrow would move it, when the arrow is going into an answer.
type Prompt struct {
	Label string // "title: ", "move to: "
	Text  string

	// Caret is where the cursor sits in Text, counted in runes from its start,
	// exactly as Chrome.Caret does in the search field.
	Caret int
}

// Chrome is everything a browse screen draws except the rows themselves.
//
// The rows and the preview text stay with the app: they are the only part that
// is about pull requests, or sessions, or whatever is being listed. Everything
// here is the same in every app, which is the point.
type Chrome struct {
	Styles theme.Styles

	App    string   // "belp"
	Crumbs []string // "Sillage", and deeper if an app has anywhere to go
	Status string   // right-aligned in the header: counts, or what was truncated

	Groups []Group
	Query  string
	Focus  Focus
	Keys   []Key

	// Caret is where the cursor sits in Query, counted in runes from its start.
	// Query's length means the cell after the last character, which is where a
	// field that has only been typed into keeps it.
	Caret int

	// Columns is a header naming what the columns of the list hold, rendered by
	// the app: it has to line up with the rows, and only the app knows where its
	// columns start. Empty for a list whose rows are not a table.
	Columns string

	// Prompt is the question drawn over the list while Focus is FocusPrompt.
	Prompt Prompt

	// Confirm is drawn over the list while Focus is FocusConfirm.
	Confirm Confirm

	// Placeholder shows in the search field while it is empty.
	Placeholder string
}

// Confirm is a yes-or-no question and what turns on the answer.
//
// Yes and No are labelled rather than assumed: "resume it anyway" and "leave it"
// say what will happen, where "OK" and "Cancel" leave the reader to work out
// which way round the question was.
type Confirm struct {
	Question string
	Detail   []string // a line or two on why it is being asked
	Yes      string
	No       string
}

// Empty reports whether there is no question to draw.
func (c Confirm) Empty() bool { return c.Question == "" }

// The gap between filter groups, and between footer hints. Two spaces read as
// one group; this reads as a boundary.
const (
	groupGap = "  ·  "
	keyGap   = "   "
)

// ellipsis says a line was cut rather than simply ending there.
const ellipsis = "…"

// margin is the cell the filter bar and the preview keep either side of them.
//
// Those two are the panels with nothing of their own standing in for it: a row
// has its marker and the header its app name, but a chip or a field label
// beginning in column 0 reads as having been cut off rather than placed.
const margin = 1

// inset moves lines in by the margin. Empty lines are left empty: a line of one
// space is still a line, and a band that ran out of content should look like it.
func inset(lines ...string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if line != "" {
			out[i] = strings.Repeat(" ", margin) + line
		}
	}
	return out
}

// Render draws a whole screen: chrome in its bands, rows in the list, preview in
// the preview.
//
// rows and preview come already rendered by the app; this places them and drops
// what does not fit. Returns exactly l.Height lines, each within l.Width.
func (c Chrome) Render(l Layout, rows, preview []string) string {
	if l.Height <= 0 || l.Width <= 0 {
		return ""
	}

	out := make([]string, l.Height)
	put := func(r Region, lines ...string) {
		for i := 0; i < r.Height && i < len(lines); i++ {
			if y := r.Y + i; y >= 0 && y < len(out) {
				out[y] = lines[i]
			}
		}
	}

	put(l.Header, c.Header(l.Width))
	put(l.HeaderRule, c.Rule(l.Width))
	put(l.Filters, inset(c.Filters(l.Width-2*margin))...)
	put(l.Search, c.SearchBox(l.Width)...)
	put(l.Columns, fit(c.Columns, l.Width))
	put(l.List, fitAll(rows, l.List.Height, l.Width)...)
	put(l.PreviewRule, c.Rule(l.Width))
	put(l.Preview, inset(fitAll(preview, l.Preview.Height, l.Width-2*margin)...)...)
	// The hints are the list's, and while a question is open none of them are
	// true: ↵ answers the question rather than opening the row. The window says
	// what its own keys do, so the row goes blank rather than lying.
	if !c.asking() {
		put(l.Footer, c.Footer(l.Width))
	}

	// Last, and over the list: a modal that something else can draw on top of is
	// not modal. The list stays visible around it, because the question is about
	// the row still highlighted underneath.
	if c.asking() {
		c.putWindow(out, l)
	}

	return strings.Join(out, "\n")
}

// Header is the app's name, where you are in it, and the counts on the right.
func (c Chrome) Header(width int) string {
	s := c.Styles
	crumbs := s.App.Render(c.App)
	for _, crumb := range c.Crumbs {
		crumbs += " " + s.Chevron.Render(theme.Chevron) + " " + s.Crumb.Render(crumb)
	}

	// The status is the first thing to go: it is a count, and the app's name is
	// how you know which app you are looking at.
	status := s.Desc.Render(c.Status)
	if c.Status == "" || lipgloss.Width(crumbs)+2+lipgloss.Width(status) > width {
		return fit(crumbs, width)
	}
	gap := width - lipgloss.Width(crumbs) - lipgloss.Width(status)
	return crumbs + strings.Repeat(" ", gap) + status
}

// Filters is the chip bar.
//
// A focused chip is bracketed, an unfocused one padded to the same width, so the
// bar does not shift under the cursor. Too narrow and the group labels go first,
// then the rest is elided: a chip that is set and off the end means a list
// filtered in a way you cannot see.
func (c Chrome) Filters(width int) string {
	if full := c.filterBar(true); lipgloss.Width(full) <= width {
		return full
	}
	if short := c.filterBar(false); lipgloss.Width(short) <= width {
		return short
	}
	return elide(c.filterBar(false), width)
}

func (c Chrome) filterBar(labels bool) string {
	s := c.Styles
	groups := make([]string, 0, len(c.Groups))
	for _, g := range c.Groups {
		var b strings.Builder
		if labels && g.Label != "" && !g.Menu {
			// Bold: dimmed alone, the name of a filter read as one of its values.
			b.WriteString(s.Heading.Render(g.Label) + " ")
		}
		if g.Menu {
			b.WriteString(c.menuChip(g))
			groups = append(groups, b.String())
			continue
		}
		for i, o := range g.Options {
			if i > 0 {
				b.WriteString(" ")
			}
			// A chip carries two things at once and needs a glyph for each: the
			// brackets are where the cursor is, the bullet is what is set. Every
			// chip is the same width whichever it has, so the bar does not shift
			// under the cursor and a set chip does not grow.
			style, mark, open, close := s.Desc, " ", " ", " "
			if o.Selected {
				style, mark = s.Selected, theme.Bullet
			}
			if o.Focused && c.Focus == FocusFilters {
				open, close = "[", "]"
			}
			b.WriteString(style.Render(open + mark + o.Label + close))
		}
		groups = append(groups, b.String())
	}
	return strings.Join(groups, s.Rule.Render(groupGap))
}

// menuChip is a whole group in one chip: the marker, and how many of it are on.
//
// The count rather than the names. Names would make the bar's width a property
// of who is in the list, which is the thing a menu exists to avoid.
func (c Chrome) menuChip(g Group) string {
	s := c.Styles
	on := 0
	for _, o := range g.Options {
		if o.Selected {
			on++
		}
	}

	style, mark, open, close := s.Desc, " ", " ", " "
	if on > 0 {
		style, mark = s.Selected, theme.Bullet
	}
	if g.Focused && c.Focus == FocusFilters {
		open, close = "[", "]"
	}
	label := g.Label
	if label == "" {
		label = "select"
	}
	return style.Render(fmt.Sprintf("%s%s%s %d/%d ⌄%s",
		open, mark, label, on, len(g.Options), close))
}

// menuBody is the open list: every member, with what is on marked and the
// cursor where it is.
//
// The same two glyphs the chips use, for the same reason — where the cursor is
// and what is set are different questions, and answering them with one mark
// means you cannot see what you are about to toggle.
func (c Chrome) menuBody(room int) []string {
	s := c.Styles
	g, ok := c.openMenu()
	if !ok {
		return nil
	}

	label := g.Label
	if label == "" {
		label = "select"
	}
	body := []string{s.Selected.Render(label)}
	for _, o := range g.Options {
		mark := " "
		if o.Selected {
			mark = theme.Bullet
		}
		line := " " + mark + " " + o.Label
		if o.Focused {
			line = s.Selected.Render("▸" + mark + " " + o.Label)
		} else {
			line = s.Desc.Render(line)
		}
		body = append(body, elide(line, room))
	}
	return append(body, "",
		s.KeyName.Render("␣")+" "+s.KeyDesc.Render("toggle")+keyGap+
			s.KeyName.Render("↵")+" "+s.KeyDesc.Render("done"))
}

// openMenu is the group the open menu belongs to.
func (c Chrome) openMenu() (Group, bool) {
	for _, g := range c.Groups {
		if g.Menu && g.Focused {
			return g, true
		}
	}
	return Group{}, false
}

// Search is the query line: the magnifier, what has been typed, and the cursor
// sitting somewhere inside it.
//
// The visible window follows the cursor rather than the end of the text. A
// cursor walked back to the start of a long query is the one thing that has to
// stay on screen — eliding to the tail would hide the very character about to
// change — and an ellipsis marks whichever side was cut.
//
// No placeholder while the field has the focus. The hint answers "what is this
// field for", which is a question you have stopped asking by the time you are
// typing into it, and prose to the right of a cursor reads as text somebody else
// left there.
func (c Chrome) Search(width int) string {
	s := c.Styles
	focused := c.Focus == FocusSearch

	prompt := s.Desc.Render(theme.Magnifier)
	if focused {
		prompt = s.Selected.Render(theme.Magnifier)
	}

	room := width - 2 // the magnifier and the space after it
	if room < 1 {
		return fit(prompt, width)
	}
	if !focused && c.Query == "" {
		return fit(prompt+" "+s.Desc.Render(c.Placeholder), width)
	}

	caret := -1
	if focused {
		caret = c.Caret
	}
	return fit(prompt+" "+typed(s, c.Query, caret, room), width)
}

// typed draws text into room cells with the cursor sitting at caret, or with no
// cursor at all when caret is negative.
//
// One editor for both fields that have one: the search band and the question in
// a window. Two would drift, and the second to drift would be the one nobody
// looks at until they are typing into it.
func typed(s theme.Styles, text string, caret, room int) string {
	cells := []rune(text)
	follow := len(cells) - 1
	if caret >= 0 {
		// One cell more than there are characters: the one the cursor sits on
		// when it is past the end, which is where it spends most of its life.
		cells = append(cells, ' ')
		caret = clamp(caret, 0, len(cells)-1)
		follow = caret
	}
	if len(cells) == 0 || room < 1 {
		return ""
	}

	start, end, cutLeft, cutRight := window(len(cells), follow, room)

	var b strings.Builder
	if cutLeft {
		b.WriteString(s.Desc.Render(ellipsis))
	}
	for i := start; i < end; i++ {
		if i == caret {
			b.WriteString(caretCell(s, cells[i]))
			continue
		}
		b.WriteString(s.Value.Render(string(cells[i])))
	}
	if cutRight {
		b.WriteString(s.Desc.Render(ellipsis))
	}
	return b.String()
}

// caretCell draws the cursor over one cell of the field.
//
// The character under it is inverted rather than covered: a cursor that hides
// what it is on makes you move it away to read what you are about to change.
// Where there is no character — the cell past the end of the query — there is
// nothing to invert and the block is drawn outright.
func caretCell(s theme.Styles, r rune) string {
	if r == ' ' {
		return s.Cursor.Render(theme.Caret)
	}
	return s.Cursor.Reverse(true).Render(string(r))
}

// window is the run of cells to draw: which slice of n fits in room, chosen so
// that follow falls inside it, and whether each end had to be cut.
//
// The ellipses cost a cell each, and giving one up moves the window, which can
// change whether the other end is cut at all — so it settles rather than
// calculates. Three passes at the outside: each one can only take cells away.
func window(n, follow, room int) (start, end int, cutLeft, cutRight bool) {
	if n <= room {
		return 0, n, false, false
	}
	for {
		room := room
		if cutLeft {
			room--
		}
		if cutRight {
			room--
		}
		if room < 1 {
			// Narrower than its own ellipses. The cursor is what the field is
			// for, so the cursor is what the one cell shows.
			return max(0, follow), max(0, follow) + 1, false, false
		}
		start = 0
		if follow >= room {
			start = follow - room + 1
		}
		if start+room > n {
			start = n - room
		}
		end = start + room

		left, right := start > 0, end < n
		if left == cutLeft && right == cutRight {
			return start, end, cutLeft, cutRight
		}
		cutLeft, cutRight = left, right
	}
}

// SearchBox is the field inside its border: exactly searchRows lines, each
// within width.
//
// The border is what says the field is a field. Rounded, because a query is
// typed into rather than acted on, and a square box in a screen of plain rules
// reads as one more panel; blue while it has the focus, which is the same blue
// the selected row uses, so the eye only ever has to learn one colour for "this
// is where the keyboard is pointing".
//
// The margin either side matches the filter bar above it, so the two left edges
// line up rather than stepping.
func (c Chrome) SearchBox(width int) []string {
	// The border's own two columns, and a cell of padding inside it: a magnifier
	// touching the border reads as part of the frame rather than as the field's.
	inner := width - 2*margin - 2 - 2*margin
	if inner < 1 {
		// No room to be a box. The field alone is still worth having: it says
		// what has been typed, which is the part nothing else on screen says.
		return []string{fit(c.Search(width), width), "", ""}
	}

	edge := c.Styles.Palette.Faint
	if c.Focus == FocusSearch {
		edge = c.Styles.Palette.Accent
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(edge).
		Padding(0, margin).
		Width(inner).
		Render(c.Search(inner))

	return inset(strings.Split(box, "\n")...)
}

// Footer is the key hints.
//
// A hint that does not fit is left out whole. Half a binding is worse than a
// missing one: it reads as a key that does something unnameable.
func (c Chrome) Footer(width int) string {
	s := c.Styles
	var b strings.Builder
	for _, k := range c.Keys {
		hint := s.KeyName.Render(k.Name) + " " + s.KeyDesc.Render(k.Desc)
		gap := ""
		if b.Len() > 0 {
			gap = keyGap
		}
		if lipgloss.Width(b.String())+lipgloss.Width(gap)+lipgloss.Width(hint) > width {
			break
		}
		b.WriteString(gap + hint)
	}
	return fit(b.String(), width)
}

// windowMax is as wide as a question gets. A box the width of the terminal is not
// a window, it is another band; this keeps it plainly a thing on top.
const windowMax = 64

// putWindow centres the question over the list.
func (c Chrome) putWindow(out []string, l Layout) {
	box := c.Window(min(l.Width-2*margin, windowMax))
	if len(box) == 0 {
		return
	}

	// Centred in the list, or wherever there is room for it: on a short terminal
	// the list is three rows and the box is five, so it starts at the top of the
	// band and takes what it needs rather than vanishing.
	top := l.List.Y + max(0, (l.List.Height-len(box))/2)
	left := max(0, (l.Width-lipgloss.Width(box[0]))/2)
	pad := strings.Repeat(" ", left)
	for i, line := range box {
		if y := top + i; y >= 0 && y < len(out) {
			out[y] = fit(pad+line, l.Width)
		}
	}
}

// Window is the question as a bordered box: exactly the lines it needs, each
// within width.
//
// Both kinds go in the same frame. A question with two answers and a question
// with a typed one are the same interruption, and giving them different shapes
// would say they are not. The answers are drawn as the keys that give them,
// because a window with no visible way out is a window people quit the program
// to escape.
func (c Chrome) Window(width int) []string {
	if !c.asking() || width <= 4 {
		return nil
	}

	// The border, and the cell of padding inside it.
	room := width - 4
	body := c.confirmBody()
	switch c.Focus {
	case FocusPrompt:
		body = c.promptBody(room)
	case FocusMenu:
		body = c.menuBody(room)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Styles.Palette.Accent).
		Padding(0, 1).
		Width(width - 2). // the border takes a cell either side
		Render(strings.Join(body, "\n"))
	return strings.Split(box, "\n")
}

// asking reports whether there is a question to draw over the list.
func (c Chrome) asking() bool {
	switch c.Focus {
	case FocusPrompt:
		return c.Prompt.Label != ""
	case FocusConfirm:
		return !c.Confirm.Empty()
	case FocusMenu:
		_, ok := c.openMenu()
		return ok
	}
	return false
}

// promptBody is a question with an answer being typed into it: what was asked,
// the answer with the cursor in it, and the two keys that end it.
//
// The line has no border of its own. The window is already a frame, and a frame
// inside a frame reads as two things rather than one field — the cursor is what
// says the line is being typed into.
func (c Chrome) promptBody(room int) []string {
	s := c.Styles

	// The label was written to sit inline in a footer ("title: "), so its colon
	// is punctuation for a sentence that no longer exists. Trimmed for the
	// heading only: the app gets its label back untouched in the answer.
	label := strings.TrimSuffix(strings.TrimSpace(c.Prompt.Label), ":")

	return []string{
		s.Selected.Render(label),
		typed(s, c.Prompt.Text, c.Prompt.Caret, room),
		"",
		s.KeyName.Render("↵") + " " + s.KeyDesc.Render("confirm") + keyGap +
			s.KeyName.Render("␛") + " " + s.KeyDesc.Render("cancel"),
	}
}

// confirmBody is a question with two answers.
func (c Chrome) confirmBody() []string {
	s := c.Styles

	body := []string{s.Selected.Render(c.Confirm.Question)}
	for _, line := range c.Confirm.Detail {
		body = append(body, s.Desc.Render(line))
	}
	yes, no := c.Confirm.Yes, c.Confirm.No
	if yes == "" {
		yes = "yes"
	}
	if no == "" {
		no = "no"
	}
	return append(body, "",
		s.KeyName.Render("y")+" "+s.KeyDesc.Render(yes)+keyGap+
			s.KeyName.Render("n")+" "+s.KeyDesc.Render(no)+"   "+s.KeyDesc.Render("␛"))
}

// Rule is a separator across the full width.
func (c Chrome) Rule(width int) string {
	if width <= 0 {
		return ""
	}
	return c.Styles.Rule.Render(strings.Repeat("─", width))
}

// fit truncates a rendered line to width cells, ANSI sequences and wide runes
// accounted for. It does not pad: nothing here paints a background, so trailing
// spaces would only be spaces.
func fit(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) <= width {
		return line
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

// elide truncates to width cells with a trailing ellipsis, so a line that was
// cut says so. Used where what falls off the end is state rather than prose.
func elide(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) <= width {
		return line
	}
	if width == 1 {
		return ellipsis
	}
	return lipgloss.NewStyle().MaxWidth(width-1).Render(line) + ellipsis
}

// fitAll takes the caller's lines to exactly n of them, each within width.
func fitAll(lines []string, n, width int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		if i < len(lines) {
			out[i] = fit(lines[i], width)
		}
	}
	return out
}
