package browse

import "testing"

// The invariants, checked across every size a terminal plausibly is rather than
// at a few chosen ones. Layout arithmetic fails at the edges — one row short, one
// row too many — and a test that only visits 24×80 never sees them.
func TestComputeHoldsItsInvariantsAtEverySize(t *testing.T) {
	for height := 0; height <= 80; height++ {
		for _, width := range []int{0, 1, 20, 80, 200} {
			l := Compute(width, height, false)

			if l.Width != max(0, width) || l.Height != max(0, height) {
				t.Fatalf("%dx%d: Layout reports %dx%d", width, height, l.Width, l.Height)
			}
			if width <= 0 || height <= 0 {
				for _, b := range l.stack() {
					if !b.r.Empty() {
						t.Errorf("%dx%d: %s has height %d on an empty terminal",
							width, height, b.name, b.r.Height)
					}
				}
				continue
			}

			// Bands run down the screen in stack order, never overlapping and
			// never leaving a gap: together they are exactly the terminal.
			y, total := 0, 0
			for _, b := range l.stack() {
				if b.r.Empty() {
					continue
				}
				if b.r.Y != y {
					t.Errorf("%dx%d: %s starts at row %d, want %d",
						width, height, b.name, b.r.Y, y)
				}
				if b.r.Bottom() >= height {
					t.Errorf("%dx%d: %s ends at row %d, past the last row %d",
						width, height, b.name, b.r.Bottom(), height-1)
				}
				y += b.r.Height
				total += b.r.Height
			}
			if total != height {
				t.Errorf("%dx%d: bands cover %d row(s), want the whole %d",
					width, height, total, height)
			}

			// The one band there is never a good reason to lose.
			if l.List.Height < 1 {
				t.Errorf("%dx%d: no list", width, height)
			}
			// The footer is pinned to the last row wherever there are two rows
			// to divide. On a single row the list takes it.
			switch height {
			case 1:
				if !l.Footer.Empty() {
					t.Errorf("1 row: footer = %+v, want the row to go to the list", l.Footer)
				}
			default:
				if l.Footer.Height != 1 {
					t.Errorf("%dx%d: footer has height %d, want 1", width, height, l.Footer.Height)
				}
				if l.Footer.Y != height-1 {
					t.Errorf("%dx%d: footer at row %d, want the last row %d",
						width, height, l.Footer.Y, height-1)
				}
			}

			// A rule with nothing under it is a rule drawn for no reason.
			if !l.HeaderRule.Empty() && l.Header.Empty() {
				t.Errorf("%dx%d: a header rule with no header", width, height)
			}
			if l.PreviewRule.Empty() != l.Preview.Empty() {
				t.Errorf("%dx%d: preview rule and preview disagree (%v, %v)",
					width, height, l.PreviewRule, l.Preview)
			}
			// The search field is a box: it is three rows or it is none.
			if h := l.Search.Height; h != 0 && h != searchRows {
				t.Errorf("%dx%d: search band is %d row(s), want 0 or %d",
					width, height, h, searchRows)
			}
		}
	}
}

// Growing a terminal must never take a band away. Without this, a preview that
// appears at 20 rows and vanishes at 21 is a bug nobody would think to look for.
func TestComputeOnlyGainsBandsAsTheTerminalGrows(t *testing.T) {
	var prev Layout
	for height := 1; height <= 80; height++ {
		l := Compute(80, height, false)
		if height > 1 {
			for i, b := range l.stack() {
				was := prev.stack()[i]
				if was.r.Empty() || !b.r.Empty() {
					continue
				}
				t.Errorf("%s was present at %d rows and is gone at %d",
					b.name, height-1, height)
			}
		}
		prev = l
	}
}

// The preview earns its rows: it may not squeeze the list below what you can
// browse, and it is the first thing given up when it would.
func TestComputeGivesUpThePreviewBeforeTheList(t *testing.T) {
	for height := 1; height <= 80; height++ {
		l := Compute(80, height, false)
		if l.Preview.Empty() {
			continue
		}
		if l.List.Height < minList {
			t.Errorf("%d rows: preview of %d leaves a list of %d, below the %d minimum",
				height, l.Preview.Height, l.List.Height, minList)
		}
		if l.Preview.Height < previewMin || l.Preview.Height > previewMax {
			t.Errorf("%d rows: preview is %d, outside [%d, %d]",
				height, l.Preview.Height, previewMin, previewMax)
		}
	}
}

// A preview at all, on the terminal sizes people actually use.
func TestComputeShowsAPreviewOnAnOrdinaryTerminal(t *testing.T) {
	for _, height := range []int{24, 30, 40, 50} {
		l := Compute(120, height, false)
		if l.Preview.Empty() {
			t.Errorf("%d rows: no preview on a terminal that size", height)
		}
		if l.Filters.Empty() || l.Header.Empty() {
			t.Errorf("%d rows: lost the header or the filter bar with room to spare", height)
		}
	}
}

// The window settles rather than calculates: giving a cell to one ellipsis moves
// the window, which can decide whether the other end is cut at all.
func TestWindowHoldsWhatItFollows(t *testing.T) {
	for _, tc := range []struct {
		name              string
		n, follow, room   int
		start, end        int
		cutLeft, cutRight bool
	}{
		{"it all fits", 5, 4, 10, 0, 5, false, false},
		{"cursor at the end", 20, 19, 10, 11, 20, true, false},
		{"cursor at the start", 20, 0, 10, 0, 9, false, true},
		{"cursor in the middle", 20, 10, 10, 3, 11, true, true},
		{"narrower than its ellipses", 20, 7, 2, 7, 8, false, false},
	} {
		start, end, left, right := window(tc.n, tc.follow, tc.room)
		if start != tc.start || end != tc.end || left != tc.cutLeft || right != tc.cutRight {
			t.Errorf("%s: window(%d, %d, %d) = %d..%d cut(%v, %v), want %d..%d cut(%v, %v)",
				tc.name, tc.n, tc.follow, tc.room, start, end, left, right,
				tc.start, tc.end, tc.cutLeft, tc.cutRight)
		}
		if tc.follow < start || tc.follow >= end {
			t.Errorf("%s: window %d..%d does not hold %d", tc.name, start, end, tc.follow)
		}
		// What is drawn is the slice plus an ellipsis for each end that was cut,
		// and that has to be exactly the room there was.
		cells := end - start
		if left {
			cells++
		}
		if right {
			cells++
		}
		if cells > tc.room {
			t.Errorf("%s: %d cell(s) drawn in room for %d", tc.name, cells, tc.room)
		}
	}
}

// One worked example, so the arithmetic is legible and not only asserted about.
// 30 rows: 6 of chrome at the top — the search box is three of them — a 10-row
// preview (a third of 30) with its rule, the footer on the last row, and the list
// taking the 12 rows left over.
func TestComputeAtThirtyRows(t *testing.T) {
	l := Compute(120, 30, false)

	want := []struct {
		name string
		r    Region
	}{
		{"header", Region{Y: 0, Height: 1}},
		{"header rule", Region{Y: 1, Height: 1}},
		{"filters", Region{Y: 2, Height: 1}},
		{"search", Region{Y: 3, Height: 3}},
		{"columns", Region{Y: 6, Height: 0}}, // no table on this screen
		{"list", Region{Y: 6, Height: 12}},
		{"preview rule", Region{Y: 18, Height: 1}},
		{"preview", Region{Y: 19, Height: 10}},
		{"footer", Region{Y: 29, Height: 1}},
	}
	got := l.stack()
	for i, w := range want {
		if *got[i].r != w.r {
			t.Errorf("%s = %+v, want %+v", w.name, *got[i].r, w.r)
		}
	}
}

// The narrowest useful screen: no room for a preview, but a list, a search field
// and a way out.
func TestComputeOnAShortTerminal(t *testing.T) {
	l := Compute(80, 10, false)
	if !l.Preview.Empty() {
		t.Errorf("preview = %+v, want none at 10 rows", l.Preview)
	}
	if l.Search.Empty() {
		t.Error("no search field at 10 rows; the list is unusable without one")
	}
	if l.List.Height != 3 {
		t.Errorf("list = %d rows, want 3 (10 less the 6 chrome rows and the footer)", l.List.Height)
	}
}

// Absurdly short, but a terminal can be dragged there and must not panic or
// draw off-screen. The list and the footer are the last things standing.
func TestComputeSurvivesATinyTerminal(t *testing.T) {
	for height := 2; height <= 7; height++ {
		l := Compute(40, height, false)
		if l.List.Empty() {
			t.Errorf("%d rows: no list", height)
		}
		if l.Footer.Y != height-1 {
			t.Errorf("%d rows: footer at %d, want %d", height, l.Footer.Y, height-1)
		}
	}
	// One row is the list, with nothing else: whatever is being browsed beats
	// the hints for how to browse it.
	l := Compute(40, 1, false)
	if l.List != (Region{Y: 0, Height: 1}) {
		t.Errorf("1 row: list = %+v, want the whole row", l.List)
	}
	if !l.Footer.Empty() {
		t.Errorf("1 row: footer = %+v, want none", l.Footer)
	}
}

func TestRegionHelpers(t *testing.T) {
	r := Region{Y: 5, Height: 3}
	if r.Empty() {
		t.Error("Empty() = true for a band with rows")
	}
	if r.Bottom() != 7 {
		t.Errorf("Bottom() = %d, want 7", r.Bottom())
	}
	// An empty band's Bottom sits before its Y, so a `for y := r.Y; y <=
	// r.Bottom()` loop runs zero times rather than once.
	e := Region{Y: 5}
	if !e.Empty() {
		t.Error("Empty() = false for a band with no rows")
	}
	if e.Bottom() != 4 {
		t.Errorf("Bottom() = %d, want 4 so that iterating draws nothing", e.Bottom())
	}
}

// The same 30 rows with a table on the screen: the header takes one, and the list
// gives it up rather than anything below.
func TestComputeMakesRoomForAColumnHeader(t *testing.T) {
	plain, table := Compute(120, 30, false), Compute(120, 30, true)

	if table.Columns != (Region{Y: 6, Height: 1}) {
		t.Errorf("columns = %+v, want one row directly above the list", table.Columns)
	}
	if table.List.Y != 7 || table.List.Height != plain.List.Height-1 {
		t.Errorf("list = %+v, want it one row shorter and one row lower than %+v",
			table.List, plain.List)
	}
	for name, pair := range map[string][2]Region{
		"preview rule": {plain.PreviewRule, table.PreviewRule},
		"preview":      {plain.Preview, table.Preview},
		"footer":       {plain.Footer, table.Footer},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s moved to %+v; the header came out of the list, not out of %s",
				name, pair[1], name)
		}
	}
}

// The column header is the first thing given up, because it is the one piece of
// chrome you could work out for yourself from the rows under it.
func TestTheColumnHeaderGoesBeforeTheFilterBar(t *testing.T) {
	// Tall enough for a list and no more: the header cannot be afforded.
	for height := 3; height <= 9; height++ {
		l := Compute(80, height, true)
		if l.List.Empty() {
			t.Errorf("%d rows: no list at all", height)
			continue
		}
		if !l.Columns.Empty() && l.Filters.Empty() {
			t.Errorf("%d rows: kept the column header %+v and gave up the filter bar",
				height, l.Columns)
		}
	}
}
