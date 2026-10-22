package grid

import (
	"fmt"
	"testing"
)

// the arrangement drawn in the design: a board down the left, sillage over
// vigie on the right.
func control() Spec {
	return Spec{Split: "columns", Cells: []Spec{
		{Run: []string{"belp", "board"}, Size: "40%"},
		{Split: "rows", Cells: []Spec{
			{Run: []string{"sillage"}},
			{Run: []string{"vigie"}},
		}},
	}}
}

func at(t *testing.T, placed []Placed, name string) Rect {
	t.Helper()
	for _, p := range placed {
		if p.Spec.Title() == name {
			return p.Rect
		}
	}
	t.Fatalf("nothing called %q was placed", name)
	return Rect{}
}

func TestTheArrangementIsWhatWasDrawn(t *testing.T) {
	placed := Place(control(), Rect{W: 200, H: 50})
	if len(placed) != 3 {
		t.Fatalf("placed %d cells, want 3", len(placed))
	}

	board, mine, shipped := at(t, placed, "belp board"), at(t, placed, "sillage"), at(t, placed, "vigie")

	if board.X != 0 || board.W != 80 {
		t.Errorf("board at x=%d w=%d, want the left 40%%", board.X, board.W)
	}
	if board.H != 50 {
		t.Errorf("board is %d high, want the full height", board.H)
	}
	if mine.X != 80 || mine.W != 120 {
		t.Errorf("sillage at x=%d w=%d, want the rest of the width", mine.X, mine.W)
	}
	if mine.Y != 0 || shipped.Y != mine.H {
		t.Errorf("sillage y=%d h=%d, vigie y=%d — they should stack without a gap",
			mine.Y, mine.H, shipped.Y)
	}
	if mine.H+shipped.H != 50 {
		t.Errorf("the two rows come to %d, want 50", mine.H+shipped.H)
	}
}

// A column one cell narrower than its neighbour is invisible; a gap at the edge
// of the screen is not. So the remainder goes to the last share.
func TestNothingIsLostToRounding(t *testing.T) {
	for _, total := range []int{80, 81, 99, 100, 101, 137, 200} {
		s := Spec{Split: "columns", Cells: []Spec{
			{Run: []string{"a"}}, {Run: []string{"b"}}, {Run: []string{"c"}},
		}}
		placed := Place(s, Rect{W: total, H: 20})
		last := placed[len(placed)-1]
		if got := last.Rect.X + last.Rect.W; got != total {
			t.Errorf("at width %d the far edge lands at %d", total, got)
		}
		for i := 1; i < len(placed); i++ {
			if placed[i].Rect.X != placed[i-1].Rect.X+placed[i-1].Rect.W {
				t.Errorf("at width %d there is a gap between cell %d and %d", total, i-1, i)
			}
		}
	}
}

// What asked for a size gets it; what did not splits what is left.
func TestAskingForASizeGetsIt(t *testing.T) {
	s := Spec{Split: "columns", Cells: []Spec{
		{Run: []string{"fixed"}, Size: "30"},
		{Run: []string{"a"}}, {Run: []string{"b"}},
	}}
	placed := Place(s, Rect{W: 130, H: 10})
	if w := at(t, placed, "fixed").W; w != 30 {
		t.Errorf("the fixed cell got %d, want 30", w)
	}
	if w := at(t, placed, "a").W; w != 50 {
		t.Errorf("a shared cell got %d, want half of what was left", w)
	}
}

// A cell with no room is still a cell: a program started into nothing can never
// draw and never says why, so every one keeps at least a line.
func TestEveryCellKeepsALine(t *testing.T) {
	s := Spec{Split: "rows", Cells: []Spec{
		{Run: []string{"a"}}, {Run: []string{"b"}}, {Run: []string{"c"}}, {Run: []string{"d"}},
	}}
	for _, p := range Place(s, Rect{W: 40, H: 2}) {
		if p.Rect.H < 1 || p.Rect.W < 1 {
			t.Errorf("%s got %dx%d", p.Spec.Title(), p.Rect.W, p.Rect.H)
		}
	}
}

// Nesting is what lets a column hold rows, which is the shape people draw.
func TestSplitsNest(t *testing.T) {
	s := Spec{Split: "columns", Cells: []Spec{
		{Run: []string{"left"}},
		{Split: "rows", Cells: []Spec{
			{Run: []string{"top"}},
			{Split: "columns", Cells: []Spec{{Run: []string{"bl"}}, {Run: []string{"br"}}}},
		}},
	}}
	placed := Place(s, Rect{W: 100, H: 40})
	if len(placed) != 4 {
		t.Fatalf("placed %d, want 4", len(placed))
	}
	bl, br := at(t, placed, "bl"), at(t, placed, "br")
	if bl.Y != 20 || br.Y != 20 {
		t.Errorf("the nested pair is at y=%d,%d; want the lower half", bl.Y, br.Y)
	}
	if bl.X != 50 || br.X != 75 {
		t.Errorf("the nested pair is at x=%d,%d; want the right half split again", bl.X, br.X)
	}
}

// Nothing to draw into is nothing placed, rather than a panic or a cell of
// negative size.
func TestNoRoomPlacesNothing(t *testing.T) {
	for _, area := range []Rect{{W: 0, H: 10}, {W: 10, H: 0}, {W: -5, H: -5}} {
		if got := Place(control(), area); len(got) != 0 {
			t.Errorf("%v placed %d cells", area, len(got))
		}
	}
}

// The border has to say which app it is without being told twice.
func TestACellNamesItself(t *testing.T) {
	for _, c := range []struct {
		spec Spec
		want string
	}{
		{Spec{Run: []string{"sillage"}}, "sillage"},
		{Spec{Run: []string{"belp", "board"}}, "belp board"},
		{Spec{Run: []string{"belp", "logs", "web"}}, "belp logs"},
		{Spec{Run: []string{"vigie"}, Name: "shipped"}, "shipped"},
	} {
		if got := c.spec.Title(); got != c.want {
			t.Errorf("%v named itself %q, want %q", c.spec.Run, got, c.want)
		}
	}
}

func ExamplePlace() {
	for _, p := range Place(control(), Rect{W: 200, H: 50}) {
		fmt.Printf("%-12s %3d,%-3d %3dx%d\n", p.Spec.Title(), p.Rect.X, p.Rect.Y, p.Rect.W, p.Rect.H)
	}
	// Output:
	// belp board     0,0    80x50
	// sillage       80,0   120x25
	// vigie         80,25  120x25
}
