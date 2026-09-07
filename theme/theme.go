// Package theme is the palette and the text styles every belp app draws with.
//
// It holds the look and nothing else: no widgets, no layout, no behaviour.
// An app builds its own screens out of bubbles and lipgloss, but takes every
// colour and every text style from here, so that belp and the apps it launches
// read as one product rather than several programs that share a terminal.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette is the colours, and only the colours.
//
// Each is adaptive: it carries a value for light terminals and one for dark,
// and lipgloss chooses per the terminal's reported background. A single fixed
// colour is always wrong on one of the two.
type Palette struct {
	Text   lipgloss.AdaptiveColor // body text
	Dim    lipgloss.AdaptiveColor // labels, metadata, anything secondary
	Faint  lipgloss.AdaptiveColor // rules and separators
	Accent lipgloss.AdaptiveColor // the selected or focused thing
	Key    lipgloss.AdaptiveColor // key names in a status bar

	Success lipgloss.AdaptiveColor
	Warn    lipgloss.AdaptiveColor
	Danger  lipgloss.AdaptiveColor // destructive and irreversible

	// A life cycle, in the order things travel through it: begun, under way,
	// landed, and spent. Success is the landed step — a thing that arrived is
	// the same green as a thing that went well — and Danger is the step a cycle
	// takes when it ends badly instead.
	//
	// A ramp rather than four unrelated hues, because the question a feed asks
	// of every line is not "what sort of act was this" but "how far along is
	// it": amber has not landed, teal is moving, green arrived, grey is over.
	Begun  lipgloss.AdaptiveColor
	Flight lipgloss.AdaptiveColor
	Spent  lipgloss.AdaptiveColor

	// Two neighbouring hues for acts on somebody else's work, which sit outside
	// either cycle: they move nothing along, and colouring them as a stage
	// would put them somewhere on a ramp they are not on.
	//
	// Kept clear of Accent, which means "the thing you are about to act on": a
	// row drawn in it competes with the cursor.
	Voice lipgloss.AdaptiveColor // something said — a comment
	Judge lipgloss.AdaptiveColor // something weighed — a review
}

// DefaultPalette is deliberately restrained: one accent, three greys, three
// signals, a four-step life cycle, and two voices for other people. A launcher is glanced at rather than read, so colour is spent on
// the thing you are about to act on and on warning you off the rest.
func DefaultPalette() Palette {
	return Palette{
		Text:    lipgloss.AdaptiveColor{Light: "#1c1c1c", Dark: "#ffffff"},
		Dim:     lipgloss.AdaptiveColor{Light: "#6c6c6c", Dark: "#dcdcdc"},
		Faint:   lipgloss.AdaptiveColor{Light: "#c6c6c6", Dark: "#3a3a3a"},
		Accent:  lipgloss.AdaptiveColor{Light: "#0057d8", Dark: "#7aa2f7"},
		Key:     lipgloss.AdaptiveColor{Light: "#8f4700", Dark: "#e0af68"},
		Success: lipgloss.AdaptiveColor{Light: "#006600", Dark: "#9ece6a"},
		Warn:    lipgloss.AdaptiveColor{Light: "#8f6a00", Dark: "#e0af68"},
		Danger:  lipgloss.AdaptiveColor{Light: "#a00000", Dark: "#f7768e"},
		Begun:   lipgloss.AdaptiveColor{Light: "#8f6a00", Dark: "#e0af68"},
		Flight:  lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#73daca"},
		Spent:   lipgloss.AdaptiveColor{Light: "#767676", Dark: "#8a92b2"},
		Voice:   lipgloss.AdaptiveColor{Light: "#9d174d", Dark: "#ff9ac1"},
		Judge:   lipgloss.AdaptiveColor{Light: "#6b21a8", Dark: "#bb9af7"},
	}
}

// Styles are the text styles built from a Palette. Apps use these rather than
// calling lipgloss themselves, so a change of look lands everywhere at once.
//
// They carry no width, no padding and no borders: those belong to whichever
// screen is being laid out, and baking them in here would make every style
// wrong somewhere.
type Styles struct {
	Palette Palette

	App      lipgloss.Style // the app's own name, in a header
	Crumb    lipgloss.Style // where you are within it
	Chevron  lipgloss.Style // between crumbs
	Heading  lipgloss.Style // a group heading
	Item     lipgloss.Style
	Selected lipgloss.Style
	Desc     lipgloss.Style // secondary text beside an item
	Label    lipgloss.Style // a field name in a detail view
	Value    lipgloss.Style
	Rule     lipgloss.Style
	KeyName  lipgloss.Style // "^G"
	KeyDesc  lipgloss.Style // "grep"
	Cursor   lipgloss.Style // the caret of a text field being typed into
	Success  lipgloss.Style
	Warn     lipgloss.Style
	Danger   lipgloss.Style
	Begun    lipgloss.Style
	Flight   lipgloss.Style
	Spent    lipgloss.Style
	Voice    lipgloss.Style
	Judge    lipgloss.Style
}

// Chevron is the separator between crumbs: a plain one, not a powerline
// triangle. That glyph exists only in a patched font, and an app that renders
// as tofu on a stock terminal is worse than one that renders plainly anywhere.
const Chevron = "›"

// Bullet marks a filter chip that is set.
//
// Colour alone cannot carry it. Which chip the cursor is on and which chip is set
// are two things, and if only one of them has a glyph, a filter set in passing
// stays invisible until you notice something missing from the screen.
const Bullet = "•"

// A group's checkbox, in its three states: every option in the group, some of
// them, none of them.
//
// Three glyphs rather than two, because "some" is the state a group spends most
// of its life in, and drawing it as either of the others makes one click do
// something different from what the box appeared to promise.
const (
	BoxAll  = "■"
	BoxSome = "▪"
	BoxNone = "□"
)

// Magnifier marks the search field, in place of a label saying "search".
//
// U+2315 rather than the 🔍 emoji or a nerd-font glyph: the emoji is two cells
// wide and coloured by the font rather than by the palette, and the patched-font
// one renders as tofu anywhere the font is not installed — the same reason
// [Chevron] is not a powerline triangle.
const Magnifier = "⌕"

// Caret is the block a text field draws where the next character will land.
//
// Drawn rather than left to the terminal: an app that repaints whole frames
// parks the real cursor wherever the last line ended, and bubbletea hides it
// besides, so a field that wants a cursor has to paint one. A block, because
// that is what a terminal cursor looks like — anything thinner reads as
// punctuation someone typed.
const Caret = "█"

// New builds the styles for a palette.
func New(p Palette) Styles {
	fg := func(c lipgloss.AdaptiveColor) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(c)
	}
	return Styles{
		Palette:  p,
		App:      fg(p.Accent).Bold(true),
		Crumb:    fg(p.Text),
		Chevron:  fg(p.Faint),
		Heading:  fg(p.Dim).Bold(true),
		Item:     fg(p.Text),
		Selected: fg(p.Accent).Bold(true),
		Desc:     fg(p.Dim),
		Label:    fg(p.Dim),
		Value:    fg(p.Text),
		Rule:     fg(p.Faint),
		KeyName:  fg(p.Key).Bold(true),
		KeyDesc:  fg(p.Dim),
		Cursor:   fg(p.Dim),
		Success:  fg(p.Success),
		Warn:     fg(p.Warn),
		Danger:   fg(p.Danger),
		Begun:    fg(p.Begun),
		Flight:   fg(p.Flight),
		Spent:    fg(p.Spent),
		Voice:    fg(p.Voice),
		Judge:    fg(p.Judge),
	}
}

// Default is what an app uses when it has no reason to customise.
func Default() Styles { return New(DefaultPalette()) }
