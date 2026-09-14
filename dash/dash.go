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
	// Care is one whose effect reaches past this machine, or costs real time.
	Care
	// Grave is destructive.
	Grave
)

func (t Tone) style(s theme.Styles) lipgloss.Style {
	switch t {
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
	case Care:
		return p.Warn
	case Grave:
		return p.Danger
	default:
		return p.Accent
	}
}

// Info is the mark that opens the detail. Three ASCII cells rather than ⓘ,
// which half the terminal fonts render as an emoji and so two cells wide.
const Info = "(i)"

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
func Buttons(s theme.Styles, row []Button, focus int) []string {
	const gap = 1
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

// Switch is a segmented choice: exactly one of a few, side by side, so that the
// alternatives are visible rather than hidden behind a press.
type Switch struct {
	Options []string
	Active  int
	Tone    Tone
}

// Render draws the switch on one line.
func (w Switch) Render(s theme.Styles, focused bool) string {
	if len(w.Options) == 0 {
		return ""
	}
	on := lipgloss.NewStyle().Foreground(w.Tone.colour(s.Palette)).Reverse(true)
	if !focused {
		// Chosen but not focused still has to read as chosen: a switch drawn
		// flat when the keys are elsewhere is a switch whose setting you have to
		// go and check.
		on = lipgloss.NewStyle().Foreground(w.Tone.colour(s.Palette)).Bold(true)
	}
	edge := s.Rule
	if focused {
		edge = s.Selected
	}

	parts := make([]string, len(w.Options))
	for i, opt := range w.Options {
		label := " " + opt + " "
		if i == w.Active {
			parts[i] = on.Render(label)
		} else {
			parts[i] = s.Desc.Render(label)
		}
	}
	return edge.Render("▏") + strings.Join(parts, edge.Render("│")) + edge.Render("▕")
}

// Wide is how many cells a switch takes.
func (w Switch) Wide() int {
	n := 2 + max(0, len(w.Options)-1)
	for _, opt := range w.Options {
		n += lipgloss.Width(opt) + 2
	}
	return n
}

// Pair is one fact: what it is, and what it says.
type Pair struct {
	Key   string
	Value string
	Tone  Tone
	// Quiet draws the value as secondary text rather than as a value. For the
	// facts that are there for completeness rather than to be read.
	Quiet bool
}

// Pairs lays facts out in as many columns of key-and-value as the width allows,
// and stacks them when it does not.
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
			line = append(line, s.Label.Render(padTo(cutTo(p.Key, keyW), keyW))+" "+
				value.Render(padTo(cutTo(p.Value, valW), valW)))
		}
		out = append(out, strings.TrimRight(strings.Join(line, strings.Repeat(" ", gap)), " "))
	}
	return out
}

// Meter is a proportion drawn as a bar.
//
// Filled and unfilled blocks rather than a percentage: the question a bar
// answers is "nearly all, or hardly any", and a number makes the eye read
// before it can tell.
func Meter(s theme.Styles, done, total, width int) string {
	if width <= 0 {
		return ""
	}
	style := s.Desc
	switch {
	case total > 0 && done >= total:
		style = s.Success
	case done > 0:
		style = s.Warn
	}
	full := 0
	if total > 0 {
		full = done * width / total
		// Anything at all shows as something: a bar that rounds one container
		// out of twelve down to empty says the service is down when it is not.
		if full == 0 && done > 0 {
			full = 1
		}
		if full > width {
			full = width
		}
	}
	return style.Render(strings.Repeat("█", full)) +
		s.Rule.Render(strings.Repeat("░", width-full))
}

func centre(label string, width int) string {
	room := width - lipgloss.Width(label)
	if room <= 0 {
		return cutTo(label, width)
	}
	left := room / 2
	return strings.Repeat(" ", left) + label + strings.Repeat(" ", room-left)
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
