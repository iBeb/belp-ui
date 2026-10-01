package browse

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iBeb/belp-ui/theme"
)

// One line of a list, as every list in these apps draws one.
//
// Here rather than in each of them because three apps draw the same line and
// they had drifted: a cursor glyph in one where another lets the colour say
// which row the keys are on, a flat style down the whole line in a third, and
// three different answers to what a narrow terminal gives up first. A row is a
// convention, and a convention kept in three places is three conventions.

// Weight is what a column is for, which is what decides how it is drawn.
type Weight int

const (
	// Name is what tells one row from another: the one column that is never
	// given up, and the one that takes whatever width is left over.
	Name Weight = iota
	// Ref is an identifier — a reference, a short code — which is read as a
	// label rather than as prose.
	Ref
	// Quiet is a category or a date: present, and not read twice.
	Quiet
)

// Cell is one column.
type Cell struct {
	Text string
	// Wide is its width. Zero takes whatever is left over, which is what a name
	// does; the first such cell is the one that gets it.
	Wide   int
	Weight Weight
	// Right pushes the text to the right of its column, where a date belongs
	// when the one above it is a different length.
	Right bool
	// Give is the order columns are given up in as the line narrows: the
	// largest goes first, and zero never goes.
	Give int
	// Tint is a colour of its own for a column whose colour is part of what it
	// says — which kind of work a row is, how far along something got. Taken
	// from the styles the row is drawn with, so a selected row is still the
	// accent throughout: what it is told is the row's styles, and those are
	// already the accent when the keys are on it.
	Tint func(theme.Styles) lipgloss.Style
}

// Line draws a row: a mark in the margin, then the columns.
//
// The mark is drawn already styled, because only the app knows what its states
// are — a session that is answering, a service that is up. The margin carries
// it rather than a cursor glyph: the colour says which line the keys are on,
// and a column that is empty half the time reads as a drawing fault.
func Line(s theme.Styles, selected bool, width int, mark string, cells ...Cell) string {
	if width <= 0 {
		return ""
	}
	// A selected row is the accent throughout, so the colour says "this line"
	// rather than "this word".
	s = s.ForRow(selected)

	margin := " "
	if mark != "" {
		margin = " " + mark + " "
	}

	shown := make([]bool, len(cells))
	flex := -1
	for i, c := range cells {
		shown[i] = true
		if c.Wide == 0 && flex < 0 {
			flex = i
		}
	}
	room := func() int {
		w := width - lipgloss.Width(margin)
		for i, c := range cells {
			if !shown[i] || i == flex {
				continue
			}
			w -= c.Wide + 1
		}
		return w
	}
	// Give up columns until the name has room to be read, largest Give first.
	for flex >= 0 && room() < minName {
		at, give := -1, 0
		for i, c := range cells {
			if shown[i] && c.Give > give {
				at, give = i, c.Give
			}
		}
		if at < 0 {
			break
		}
		shown[at] = false
	}

	var b strings.Builder
	b.WriteString(margin)
	for i, c := range cells {
		if !shown[i] {
			continue
		}
		style := weighed(s, c.Weight)
		if c.Tint != nil {
			style = c.Tint(s)
		}
		if i == flex {
			said := elide(c.Text, max(room(), 0))
			b.WriteString(style.Render(said))
			// What follows the name is pushed to the far end, so the columns
			// after it line up down the list however long the names are.
			if gap := room() - lipgloss.Width(said); gap > 0 {
				b.WriteString(strings.Repeat(" ", gap))
			}
			continue
		}
		b.WriteString(style.Render(column(c.Text, c.Wide, c.Right)) + " ")
	}
	return elide(strings.TrimRight(b.String(), " "), width)
}

// minName is how much of a name has to be readable before a line gives a
// column up for it.
const minName = 20

func weighed(s theme.Styles, w Weight) lipgloss.Style {
	switch w {
	case Ref:
		return s.Label
	case Quiet:
		return s.Desc
	default:
		return s.Item
	}
}

// column is a cell's text in its column, cut where it is too long and padded on
// whichever side keeps the column lined up.
func column(said string, wide int, right bool) string {
	said = elide(said, wide)
	n := wide - lipgloss.Width(said)
	if n <= 0 {
		return said
	}
	if right {
		return strings.Repeat(" ", n) + said
	}
	return said + strings.Repeat(" ", n)
}
