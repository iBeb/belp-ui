// Package browse is the shape of a belp app that shows a list of things and the
// detail of whichever one you are pointing at: a bar of filters, a search field,
// the list, and a preview pane.
//
// It exists because that is not one app's shape but every app's. Written once,
// two apps agree on where the search field is, what ␛ does and how the list
// behaves on a short terminal — which is the difference between a suite and a
// pile of programs that each had their own idea.
//
// The package owns the chrome and the geometry. What a row says, and which rows
// survive a filter, stay with the app: those are the only parts that are
// genuinely about pull requests, or sessions, or whatever is being listed.
package browse

// Region is a horizontal band of the screen: the row it starts on and how many
// rows it has. A Region with no height is a band that did not fit, which is a
// state the renderer has to expect rather than an error.
type Region struct {
	Y      int
	Height int
}

// Empty reports whether the band was left out.
func (r Region) Empty() bool { return r.Height <= 0 }

// Bottom is the last row the band covers, or Y-1 when it is empty.
func (r Region) Bottom() int { return r.Y + r.Height - 1 }

// Layout is where every band of a browse screen goes, for one terminal size.
//
// The separators are bands of their own rather than something the renderer
// squeezes in afterwards. A rule has to occupy a row, and a rule drawn on a row
// that the list also believes it owns is how the bottom row of a list goes
// missing.
type Layout struct {
	Width  int
	Height int

	Header     Region // the app's name and the counts
	HeaderRule Region
	Filters    Region // the chips: what kind of thing, in what state, from when
	Search     Region // the query line
	SearchRule Region

	List Region

	PreviewRule Region
	Preview     Region // empty when the terminal is too short to spare the rows

	Footer Region // the key hints
}

// stack is every band in the order it appears down the screen. Both Compute and
// the tests walk this rather than naming the fields again: a band added to
// Layout and forgotten in one of the two is the mistake worth designing out.
func (l *Layout) stack() []band {
	return []band{
		{"header", &l.Header},
		{"header rule", &l.HeaderRule},
		{"filters", &l.Filters},
		{"search", &l.Search},
		{"search rule", &l.SearchRule},
		{"list", &l.List},
		{"preview rule", &l.PreviewRule},
		{"preview", &l.Preview},
		{"footer", &l.Footer},
	}
}

type band struct {
	name string
	r    *Region
}

// The rows above the list that are not the list: header, its rule, the filter
// bar, the search line and its rule.
const chromeRows = 5

// minList is how short the list may get before the preview is given up. A single
// row of list beside a fourteen-row preview is not a list you can browse — and
// browsing is the point, the preview only ever confirms what the list found.
const minList = 3

// Preview bounds. A share of the height rather than fixed steps: a threshold
// tuned for one terminal size means every other size gets the wrong pane, and
// the tall pane in particular ends up never appearing at all.
const (
	previewMin = 6
	previewMax = 14
)

// Compute divides a terminal of this size into bands.
//
// The footer is pinned to the last row and the list absorbs whatever is left, so
// the layout always fills the terminal exactly. When there is not enough room
// the bands are given up in a fixed order — the preview first, then the filter
// bar, then the header and its rule — so that shrinking a window degrades the
// same way every time instead of rearranging the screen.
func Compute(width, height int) Layout {
	l := Layout{Width: max(0, width), Height: max(0, height)}
	if l.Height <= 0 || l.Width <= 0 {
		return l
	}

	// A row for the footer wherever there are two to divide: a screen with no
	// key hints is a screen nobody can leave. On a single row the list wins —
	// the list is the app, and hints for an app you cannot see are no help.
	rest := l.Height
	if l.Height >= 2 {
		l.Footer = Region{Height: 1}
		rest--
	}

	wantHeader, wantFilters := true, true
	for {
		fixed := chromeRows
		if !wantHeader {
			fixed -= 2 // the header and its rule go together
		}
		if !wantFilters {
			fixed--
		}

		preview := clamp(l.Height/3, previewMin, previewMax)
		list := rest - fixed - (preview + 1)
		if list < minList {
			// No preview: the list is what the screen is for.
			preview = 0
			list = rest - fixed
		}
		if list >= 1 {
			l.Search = Region{Height: 1}
			l.SearchRule = Region{Height: 1}
			if wantHeader {
				l.Header = Region{Height: 1}
				l.HeaderRule = Region{Height: 1}
			}
			if wantFilters {
				l.Filters = Region{Height: 1}
			}
			l.List = Region{Height: list}
			if preview > 0 {
				l.PreviewRule = Region{Height: 1}
				l.Preview = Region{Height: preview}
			}
			break
		}

		// Still no room. Give something up, in the order that costs least.
		switch {
		case wantFilters:
			wantFilters = false
		case wantHeader:
			wantHeader = false
		default:
			// Nothing left to drop: the list takes what there is, even if that
			// is a single row with no search field above it.
			l.Search, l.SearchRule = Region{}, Region{}
			l.List = Region{Height: rest}
			l.assign()
			return l
		}
	}

	l.assign()
	return l
}

// assign walks the stack and gives each non-empty band its row.
func (l *Layout) assign() {
	y := 0
	for _, b := range l.stack() {
		if b.r.Empty() {
			// Parked on the row it would have started at, so a renderer that
			// forgets to check Empty draws over itself rather than off-screen.
			b.r.Y = y
			b.r.Height = 0
			continue
		}
		b.r.Y = y
		y += b.r.Height
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
