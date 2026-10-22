// Package grid arranges programs in cells.
//
// The arrangement is a tree rather than a table: a column that holds two rows is
// the shape people actually draw, and a table would need empty cells to say it.
// Splitting is what nests, and a leaf is a program.
package grid

import (
	"strconv"
	"strings"
)

// Spec is a cell or a split of them, as the config writes it.
type Spec struct {
	// Run is the command, for a leaf. A spec with one is a cell and holds
	// nothing else.
	Run []string `json:"run"`
	Dir string   `json:"dir"`
	// Name is what the cell is called where it has to be named: the epitaph a
	// panel wears once its program has left. Empty takes the first word of the
	// command.
	Name string `json:"name"`

	// Split is "rows" or "columns", for a branch.
	Split string `json:"split"`
	Cells []Spec `json:"cells"`

	// Size is how much of the parent this takes: "40%", or empty to share what
	// is left equally with the others that said nothing.
	Size string `json:"size"`
}

// Leaf reports whether this is a program rather than a division.
func (s Spec) Leaf() bool { return len(s.Run) > 0 }

// Rect is where a cell sits, in cells of the terminal.
type Rect struct{ X, Y, W, H int }

// Placed is one leaf and the room it was given.
type Placed struct {
	Spec Spec
	Rect Rect
}

// Place works out where every program goes.
//
// Rounding is dealt with by giving the remainder to the last share rather than
// spreading it: a column one cell narrower than its neighbour is invisible, and
// a gap at the edge of the screen is not.
func Place(s Spec, area Rect) []Placed {
	if area.W <= 0 || area.H <= 0 {
		return nil
	}
	if s.Leaf() {
		return []Placed{{Spec: s, Rect: area}}
	}
	if len(s.Cells) == 0 {
		return nil
	}

	along := area.W
	if rows(s) {
		along = area.H
	}
	shares := divide(s.Cells, along)

	var out []Placed
	at := 0
	for i, cell := range s.Cells {
		size := shares[i]
		sub := area
		if rows(s) {
			sub.Y, sub.H = area.Y+at, size
		} else {
			sub.X, sub.W = area.X+at, size
		}
		at += size
		out = append(out, Place(cell, sub)...)
	}
	return out
}

// rows reports whether a split stacks its cells rather than standing them side
// by side. Columns are the default because a wide screen is the usual one.
func rows(s Spec) bool {
	switch strings.ToLower(s.Split) {
	case "rows", "row", "horizontal":
		return true
	default:
		return false
	}
}

// divide shares a length out. What asked for a size gets it; what did not
// splits the rest.
func divide(cells []Spec, total int) []int {
	sizes := make([]int, len(cells))
	asked, free := 0, 0
	for i, c := range cells {
		if n, ok := size(c.Size, total); ok {
			sizes[i] = n
			asked += n
		} else {
			free++
		}
	}

	left := total - asked
	if left < 0 {
		left = 0
	}
	if free > 0 {
		each := left / free
		seen := 0
		for i := range cells {
			if sizes[i] != 0 {
				continue
			}
			seen++
			sizes[i] = each
			if seen == free {
				// The last of them takes the remainder, so nothing is lost to
				// rounding and the far edge lands where the screen ends.
				sizes[i] = left - each*(free-1)
			}
		}
	}
	// Every cell gets at least a line, or a program is started that can never
	// draw and never says why.
	for i := range sizes {
		if sizes[i] < 1 {
			sizes[i] = 1
		}
	}
	return sizes
}

// size reads "40%" or "20" against the room there is.
func size(text string, total int) (int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	if strings.HasSuffix(text, "%") {
		n, err := strconv.Atoi(strings.TrimSuffix(text, "%"))
		if err != nil || n <= 0 {
			return 0, false
		}
		return total * n / 100, true
	}
	n, err := strconv.Atoi(text)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// Title is what a cell is called. Nothing is drawn around a running program to
// carry it — the app has a header of its own that says what it is, and a second
// label over the top of that is the same word twice — so this is read only
// where the app is no longer there to say it.
func (s Spec) Title() string {
	if s.Name != "" {
		return s.Name
	}
	if len(s.Run) > 0 {
		parts := strings.Fields(strings.Join(s.Run, " "))
		if len(parts) > 1 {
			return parts[0] + " " + parts[1]
		}
		return parts[0]
	}
	return ""
}
