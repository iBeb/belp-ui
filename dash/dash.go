// Package dash is the parts a dashboard is made of: buttons, switches, meters,
// key-and-value blocks and the box that opens over them.
//
// A dashboard is not a list. Where browse draws rows and a cursor, this draws
// controls — things with a border round them that look pressed when they have
// the keys, and say what pressing them would do rather than what state they are
// already in. An app arranges them; this only knows how one is drawn.
//
// Nothing here holds state or handles a key. A control is a value the app owns,
// rendered on demand, so which button has the focus is the app's business and
// stays in one place rather than being spread across a dozen little models.
package dash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// Tone is what a control means, which is the only thing that changes its
// colour. Kept to three, because a palette of button colours is a decoration
// budget rather than a vocabulary.
type Tone int

const (
	// Plain is an ordinary control.
	Plain Tone = iota
	// Good is the safe setting of a choice: the one you would rather find
	// selected when you come back to it.
	Good
	// Care is one whose effect reaches past this machine, or costs real time.
	Care
	// Grave is destructive.
	Grave
)

func (t Tone) style(s theme.Styles) lipgloss.Style {
	switch t {
	case Good:
		return s.Success
	case Care:
		return s.Warn
	case Grave:
		return s.Danger
	default:
		return s.Item
	}
}

func (t Tone) colour(p theme.Palette) lipgloss.AdaptiveColor {
	switch t {
	case Good:
		return p.Success
	case Care:
		return p.Warn
	case Grave:
		return p.Danger
	default:
		return p.Accent
	}
}

// Info is the mark that opens the detail.
//
// U+24D8 rather than ℹ or a Nerd Font glyph: the first carries an emoji
// presentation in most fonts and so takes two cells in some terminals and one in
// others, and the second is tofu anywhere the patched font is not installed.
// This one is an ordinary letterform in a circle, which every font has.
const Info = "ⓘ"

// Button is a thing to press: a label in a thin box.
//
// Three lines, always, so a row of buttons has one line its labels all sit on.
// A button that shrank to one line when it had no border would put its label
// somewhere different from its neighbour's.
type Button struct {
	Label string
	Tone  Tone
	// Off is a button with nothing behind it — a pause on a service that is
	// already down. Drawn, because a row of controls that loses one has moved
	// every button after it, and greyed, because it cannot be pressed.
	Off bool
	// Width forces a width; zero fits the label.
	Width int
}

// Wide is how many cells a button takes.
func (b Button) Wide() int {
	if b.Width > 0 {
		return b.Width
	}
	return lipgloss.Width(b.Label) + 4
}

// Render draws the button, focused or not.
func (b Button) Render(s theme.Styles, focused bool) []string {
	inner := b.Wide() - 2
	edge, text := s.Rule, b.Tone.style(s)
	switch {
	case b.Off:
		edge, text = s.Rule, s.Desc
	case focused:
		// Filled rather than merely outlined: a border in the accent and a label
		// in it is a button that looks selected, where a button that looks
		// pressable is one the eye reads as the next thing to do.
		fill := lipgloss.NewStyle().Foreground(b.Tone.colour(s.Palette)).Reverse(true)
		return []string{
			s.Selected.Render("╭" + strings.Repeat("─", inner) + "╮"),
			s.Selected.Render("│") + fill.Render(centre(b.Label, inner)) + s.Selected.Render("│"),
			s.Selected.Render("╰" + strings.Repeat("─", inner) + "╯"),
		}
	}
	return []string{
		edge.Render("╭" + strings.Repeat("─", inner) + "╮"),
		edge.Render("│") + text.Render(centre(b.Label, inner)) + edge.Render("│"),
		edge.Render("╰" + strings.Repeat("─", inner) + "╯"),
	}
}

// Buttons draws several side by side, with one of them focused. A focus outside
// the row focuses none of them.
//
// Given a width, the row is stretched to fill it and the slack is handed out in
// proportion to what each button asked for, so a row of controls reaches both
// edges of its card and the important one stays the widest. Given none, each is
// as wide as its label.
func Buttons(s theme.Styles, row []Button, focus, width int) []string {
	const gap = 1
	if len(row) == 0 {
		return []string{"", "", ""}
	}
	row = Stretch(row, width, gap)

	drawn := make([][]string, len(row))
	for i, b := range row {
		drawn[i] = b.Render(s, i == focus)
	}
	out := make([]string, 3)
	for line := range out {
		var parts []string
		for _, b := range drawn {
			parts = append(parts, b[line])
		}
		out[line] = strings.Join(parts, strings.Repeat(" ", gap))
	}
	return out
}

// Stretch is the widths a row of buttons will be drawn at, given the room.
//
// Exported because whoever draws the row also has to know where each button
// landed, for a click — and working that out a second time is how a click starts
// landing on the button next to the one it was aimed at.
//
// The slack goes out in proportion, and the rounding to the last button so the
// row lands exactly on the width rather than a cell short.
func Stretch(row []Button, width, gap int) []Button {
	if width <= 0 {
		return row
	}
	want := gap * (len(row) - 1)
	for _, b := range row {
		want += b.Wide()
	}
	if want >= width {
		return row
	}
	room := width - gap*(len(row)-1)
	natural := want - gap*(len(row)-1)

	out := make([]Button, len(row))
	spent := 0
	for i, b := range row {
		out[i] = b
		if i == len(row)-1 {
			out[i].Width = room - spent
			continue
		}
		out[i].Width = b.Wide() * room / natural
		spent += out[i].Width
	}
	return out
}

// Radio is a choice among a few, with every option in view and the chosen one
// filled in.
//
// Drawn where the fact it sets already lives rather than as a control of its
// own: the row that says which access a tunnel has is the row to change it on,
// and a separate switch above it would be the same fact stated twice.
type Radio struct {
	Options []string
	Active  int
	// Tone colours the chosen option. The unchosen ones stay quiet whatever it
	// is: a row of alternatives all shouting is a row that says nothing.
	Tone Tone
}

// The filled and hollow marks of a radio. Geometric shapes, one cell in every
// font, for the same reason as everything else drawn here.
const (
	radioOn  = "◉"
	radioOff = "○"
)

// Render draws the options on one line.
func (r Radio) Render(s theme.Styles, focused bool) string {
	if len(r.Options) == 0 {
		return ""
	}
	chosen := r.Tone.style(s)
	if focused {
		chosen = chosen.Bold(true).Underline(true)
	}
	parts := make([]string, len(r.Options))
	for i, opt := range r.Options {
		if i == r.Active {
			parts[i] = chosen.Render(radioOn + " " + opt)
			continue
		}
		// The mark and the word it belongs to are one thing, drawn in one style.
		// Split between two — a mark at the weight of a border beside a word at
		// the weight of text — the mark reads as part of the frame rather than
		// as the thing you are choosing between.
		parts[i] = s.Desc.Render(radioOff + " " + opt)
	}
	return strings.Join(parts, "   ")
}

// Wide is how many cells a radio takes.
func (r Radio) Wide() int {
	n := 3 * max(0, len(r.Options)-1)
	for _, opt := range r.Options {
		n += lipgloss.Width(opt) + 2
	}
	return n
}

// Toggle is one thing that is on or off, on its own.
//
// A checkbox rather than a radio: these do not exclude each other, and a filled
// dot beside another filled dot reads as a choice that has gone wrong.
type Toggle struct {
	Label string
	On    bool
	Tone  Tone
}

// The box of a toggle, in its two states.
const (
	toggleOn  = "■"
	toggleOff = "□"
)

// Render draws the toggle.
//
// The box and its label in one style, for the same reason a radio's mark is: a
// box drawn at the weight of a border beside a word at the weight of text reads
// as part of the frame rather than as the switch it is.
func (g Toggle) Render(s theme.Styles, focused bool) string {
	box, text := toggleOff, s.Desc
	if g.On {
		box, text = toggleOn, g.Tone.style(s)
	}
	if focused {
		text = text.Bold(true).Underline(true)
	}
	return text.Render(box + " " + g.Label)
}

// Wide is how many cells a toggle takes.
func (g Toggle) Wide() int { return 2 + lipgloss.Width(g.Label) }

// Pair is one fact: what it is, and what it says.
type Pair struct {
	Key   string
	Value string
	Tone  Tone
	// Quiet draws the value as secondary text rather than as a value. For the
	// facts that are there for completeness rather than to be read.
	Quiet bool
}

// KeyWidth is the column the values of a block of pairs start after, so a row
// the caller draws itself can line up with the block around it.
func KeyWidth(pairs []Pair) int {
	n := 0
	for _, p := range pairs {
		n = max(n, lipgloss.Width(p.Key))
	}
	return n
}

// Pairs lays facts out in as many columns of key-and-value as the width allows,
// and stacks them when it does not.
//
// Keys right-aligned against their values: a ragged left edge of labels reads as
// a list of words, where labels ending on one column read as the thing they
// are, which is a gutter beside the facts.
//
// Every column is the same width, taken from the widest key and the widest
// value, so the values line up down the block whichever column they land in —
// which is the only reason to put them in columns at all.
func Pairs(s theme.Styles, pairs []Pair, width int) []string {
	if len(pairs) == 0 {
		return nil
	}
	const gap = 2
	keyW, valW := 0, 0
	for _, p := range pairs {
		keyW = max(keyW, lipgloss.Width(p.Key))
		valW = max(valW, lipgloss.Width(p.Value))
	}
	// A key that will not fit is cut too: a block narrower than its own widest
	// key is still a block that has to be drawn.
	if keyW > width-2 {
		keyW = max(1, width-2)
	}
	pairW := keyW + 1 + valW

	cols := max(1, (width+gap)/(pairW+gap))
	if cols > len(pairs) {
		cols = len(pairs)
	}
	if cols == 1 && pairW > width {
		valW = max(1, width-keyW-1)
		pairW = keyW + 1 + valW
	}
	// Spend whatever is left over on the values, so a single column fills the
	// width rather than leaving a ragged half.
	if slack := width - cols*pairW - (cols-1)*gap; slack > 0 {
		valW += slack / cols
		pairW = keyW + 1 + valW
	}

	rows := (len(pairs) + cols - 1) / cols
	out := make([]string, 0, rows)
	for r := 0; r < rows; r++ {
		var line []string
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if i >= len(pairs) {
				line = append(line, strings.Repeat(" ", pairW))
				continue
			}
			p := pairs[i]
			value := p.Tone.style(s)
			if p.Quiet {
				value = s.Desc
			}
			line = append(line, s.Label.Render(rightTo(cutTo(p.Key, keyW), keyW))+" "+
				value.Render(padTo(cutTo(p.Value, valW), valW)))
		}
		out = append(out, strings.TrimRight(strings.Join(line, strings.Repeat(" ", gap)), " "))
	}
	return out
}

// Gauge is a proportion drawn as a filled bar with its own figure inside it.
//
// Backgrounds rather than block characters, and the label knocked through them
// rather than written beside them. A card gives this about a dozen cells: ten of
// twelve containers cannot be drawn as ten rectangles in that, and a figure
// sitting outside the bar spends a third of what is left saying what the bar is
// already for.
type Gauge struct {
	Done, Total int
	// Label goes inside the bar. Empty draws a bar with nothing in it.
	Label string
	// Tone overrides the colour the proportion would choose, for a bar that is
	// empty because nothing could be read rather than because nothing is up.
	Tone Tone
}

// Render draws the gauge at a width.
func (g Gauge) Render(s theme.Styles, width int) string {
	if width <= 0 {
		return ""
	}
	filled, track := g.colours(s)

	full := 0
	if g.Total > 0 {
		full = g.Done * width / g.Total
		// Anything at all shows as something: a bar that rounds one container out
		// of twelve down to empty says the service is down when it is not.
		if full == 0 && g.Done > 0 {
			full = 1
		}
		full = min(full, width)
	}

	// The label is centred over the whole bar and then cut where the fill ends,
	// so the boundary runs through the text rather than the text choosing where
	// the boundary may fall.
	text := []rune(centre(cutTo(g.Label, width), width))
	return filled.Render(string(text[:full])) + track.Render(string(text[full:]))
}

// colours are the two halves of the bar: what is done, and what is not.
//
// The figure is knocked out of the filled part in the terminal's own background
// and written in ordinary text on the rest, so it stays legible wherever the
// boundary happens to fall across it.
func (g Gauge) colours(s theme.Styles) (filled, track lipgloss.Style) {
	p := s.Palette
	bar := p.Spent
	switch {
	case g.Tone != Plain:
		bar = g.Tone.colour(p)
	case g.Total > 0 && g.Done >= g.Total:
		bar = p.Success
	case g.Done > 0:
		bar = p.Warn
	}
	filled = lipgloss.NewStyle().Background(bar).Foreground(p.Ground)
	track = lipgloss.NewStyle().Background(p.Faint).Foreground(p.Text)
	if g.Tone == Grave {
		// Nothing to fill and something to say: the whole bar takes the colour,
		// so an unreadable count does not look like a count of nothing.
		track = lipgloss.NewStyle().Background(bar).Foreground(p.Ground)
	}
	return filled, track
}

// centre puts a label in the middle of the room it has.
//
// Measured with lipgloss rather than by counting runes, so the escape sequences
// a styled label carries are not mistaken for width — a centred string padded by
// its byte length drifts further off centre the more colour it has in it.
func centre(label string, width int) string {
	room := width - lipgloss.Width(label)
	if room <= 0 {
		return cutTo(label, width)
	}
	left := room / 2
	return strings.Repeat(" ", left) + label + strings.Repeat(" ", room-left)
}

// rightTo pads on the left, for a column that ends where the next begins.
func rightTo(text string, width int) string {
	if n := width - lipgloss.Width(text); n > 0 {
		return strings.Repeat(" ", n) + text
	}
	return text
}

func padTo(text string, width int) string {
	if n := width - lipgloss.Width(text); n > 0 {
		return text + strings.Repeat(" ", n)
	}
	return text
}

func cutTo(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
