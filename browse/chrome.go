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

	// Prompt is drawn instead of Keys while Focus is FocusPrompt.
	Prompt Prompt

	// Placeholder shows in the search field while it is empty.
	Placeholder string
}

// The gap between filter groups, and between footer hints. Two spaces read as
// one group; this reads as a boundary.
const (
	groupGap = "  ·  "
	keyGap   = "   "
)

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
	put(l.Filters, c.Filters(l.Width))
	put(l.Search, c.Search(l.Width))
	put(l.SearchRule, c.Rule(l.Width))
	put(l.List, fitAll(rows, l.List.Height, l.Width)...)
	put(l.PreviewRule, c.Rule(l.Width))
	put(l.Preview, fitAll(preview, l.Preview.Height, l.Width)...)
	if c.Focus == FocusPrompt {
		put(l.Footer, c.Ask(l.Width))
	} else {
		put(l.Footer, c.Footer(l.Width))
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
			style := s.Desc
			if o.Selected {
				style = s.Selected
			}
			label := " " + o.Label + " "
			if o.Focused && c.Focus == FocusFilters {
				label = "[" + o.Label + "]"
			}
			b.WriteString(style.Render(label))
		}
		groups = append(groups, b.String())
	}
	return strings.Join(groups, s.Rule.Render(groupGap))
}

// Search is the query line.
//
// Elided from the left when it outgrows the width: you are typing at the end, so
// the end is the part that has to stay visible.
func (c Chrome) Search(width int) string {
	s := c.Styles
	prompt := s.Desc.Render("›")
	if c.Focus == FocusSearch {
		prompt = s.Selected.Render("›")
	}

	body := s.Value.Render(c.Query)
	if c.Query == "" {
		body = s.Desc.Render(c.Placeholder)
	}

	room := width - 2 // "› "
	if room > 0 && lipgloss.Width(body) > room {
		runes := []rune(c.Query)
		if n := len(runes) - room + 1; n > 0 && n < len(runes) {
			body = s.Desc.Render("…") + s.Value.Render(string(runes[n:]))
		}
	}
	return fit(prompt+" "+body, width)
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
// is drawn rather than left to the terminal, because a full-frame redraw parks
// the real cursor wherever the last line ended.
func (c Chrome) Ask(width int) string {
	s := c.Styles
	label := s.KeyName.Render(c.Prompt.Label)
	room := width - lipgloss.Width(label) - 1 // the caret

	text := c.Prompt.Text
	if room > 0 && lipgloss.Width(text) > room {
		runes := []rune(text)
		if n := len(runes) - room + 1; n > 0 && n < len(runes) {
			text = "…" + string(runes[n:])
		}
	}
	return fit(label+s.Value.Render(text)+s.Selected.Render("▏"), width)
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
		return "…"
	}
	return lipgloss.NewStyle().MaxWidth(width-1).Render(line) + "…"
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
