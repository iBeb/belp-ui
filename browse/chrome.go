package browse

import (
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
}

// Key is one footer hint.
type Key struct {
	Name string // "↵", "^R"
	Desc string // "open", "review"
}

// Prompt is a question being asked of whatever the cursor is on.
//
// It is drawn in the footer band rather than in one of its own: a question is
// transient, and a row given up permanently to something that appears twice a
// day is a row the list wanted. The key hints are what it replaces, which is
// also honest — while a question is open, none of them apply.
type Prompt struct {
	Label string // "title: ", "move to: "
	Text  string
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

	// Prompt is drawn instead of Keys while Focus is FocusPrompt.
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
	if c.Focus == FocusPrompt {
		put(l.Footer, c.Ask(l.Width))
	} else {
		put(l.Footer, c.Footer(l.Width))
	}

	// Last, and over the list: a modal that something else can draw on top of is
	// not modal. The list stays visible around it, because the question is about
	// the row still highlighted underneath.
	if c.Focus == FocusConfirm && !c.Confirm.Empty() {
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
		if labels && g.Label != "" {
			b.WriteString(s.Label.Render(g.Label) + " ")
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

	// Focused, the field carries one cell more than it has characters: the one
	// the cursor sits on when it is past the end of the query, which is where it
	// spends most of its life.
	cells := []rune(c.Query)
	caret := -1
	if focused {
		cells = append(cells, ' ')
		caret = clamp(c.Caret, 0, len(cells)-1)
	}

	// The window has to hold the cursor when there is one. With none, it holds
	// the end: an unfocused field is showing what the query is, and a query is
	// most recently true at its end.
	follow := caret
	if !focused {
		follow = len(cells) - 1
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
	return fit(prompt+" "+b.String(), width)
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

// Ask draws the open question and what has been typed into it.
//
// Elided from the left as it outgrows the width, like the search field: you are
// typing at the end, so the end is the part that has to stay visible. The caret
// is the search field's, because the two are the same line editor and a question
// that marked where you are typing differently would read as a different kind of
// field. It is always drawn: a question is only ever open while it has the focus.
func (c Chrome) Ask(width int) string {
	s := c.Styles
	label := s.KeyName.Render(c.Prompt.Label)
	room := width - lipgloss.Width(label) - 1 // the caret

	text := c.Prompt.Text
	if room > 0 && lipgloss.Width(text) > room {
		runes := []rune(text)
		if n := len(runes) - room + 1; n > 0 && n < len(runes) {
			text = ellipsis + string(runes[n:])
		}
	}
	return fit(label+s.Value.Render(text)+s.Cursor.Render(theme.Caret), width)
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

// Window is the question as a bordered box.
//
// The answers are drawn as the keys that give them, because a window with no
// visible way out is a window people quit the program to escape.
func (c Chrome) Window(width int) []string {
	if c.Confirm.Empty() || width <= 4 {
		return nil
	}
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
	body = append(body, "",
		s.KeyName.Render("y")+" "+s.KeyDesc.Render(yes)+keyGap+
			s.KeyName.Render("n")+" "+s.KeyDesc.Render(no)+"   "+s.KeyDesc.Render("␛"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Palette.Accent).
		Padding(0, 1).
		Width(width - 2). // the border takes a cell either side
		Render(strings.Join(body, "\n"))
	return strings.Split(box, "\n")
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
