// Package theme is the palette and the text styles every belp app draws with.
//
// It holds the look and nothing else: no widgets, no layout, no behaviour.
// An app builds its own screens out of bubbles and lipgloss, but takes every
// colour and every text style from here, so that belp and the apps it launches
// read as one product rather than several programs that share a terminal.
package theme

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
)

// Palette is the colours, and only the colours.
//
// Each is adaptive: it carries a value for light terminals and one for dark,
// and lipgloss chooses per the terminal's reported background. A single fixed
// colour is always wrong on one of the two.
type Palette struct {
	Text  lipgloss.AdaptiveColor // body text
	Dim   lipgloss.AdaptiveColor // metadata, anything secondary
	Faint lipgloss.AdaptiveColor // rules and separators
	// Quiet is the name of a thing rather than the thing: the label beside a
	// value, the key beside a fact. A step below Dim, because a column of labels
	// is read once and then skipped over, and one drawn at the weight of the
	// values makes a block of facts read as twice as much text as it is.
	Quiet lipgloss.AdaptiveColor
	// Primary and Secondary are the two neutral tones a control takes: the
	// ordinary button, and the one beside it that wants telling apart. Neutral
	// rather than coloured, so the coloured tones keep meaning something.
	//
	// Secondary is cooler rather than merely darker. A neutral has nothing but
	// lightness to vary, and lightness is what says how live a control is, so
	// two greys end up arguing with the state they are drawn in; a difference
	// in temperature survives being dimmed, where a difference in weight does
	// not.
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor

	Accent lipgloss.AdaptiveColor // the selected or focused thing
	Key    lipgloss.AdaptiveColor // key names in a status bar

	// Ground is the terminal's own background, near enough to write on: the
	// foreground for text drawn on a band of another colour, where Text would
	// be the one thing guaranteed not to read.
	//
	// An approximation, since a terminal's real background is whatever the user
	// set. It only ever appears on top of a colour this palette chose, so what
	// it has to contrast with is known even when the screen behind it is not.
	Ground lipgloss.AdaptiveColor

	// Resting is Accent for an app that is not the one being typed into: the
	// same blue with the life taken out of it. A selected row still has to read
	// as selected — you want to see where you left it — without competing with
	// the panel that actually has the keys.
	Resting lipgloss.AdaptiveColor

	Success lipgloss.AdaptiveColor
	Warn    lipgloss.AdaptiveColor
	Danger  lipgloss.AdaptiveColor // destructive and irreversible
	// Cycle is an act that ends a thing and begins it again — a restart. Its own
	// colour because it is none of the three above: not the safe one, not a
	// warning, and not destructive, but not nothing either.
	Cycle lipgloss.AdaptiveColor

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
		Text:      lipgloss.AdaptiveColor{Light: "#1c1c1c", Dark: "#ffffff"},
		Dim:       lipgloss.AdaptiveColor{Light: "#6c6c6c", Dark: "#dcdcdc"},
		Faint:     lipgloss.AdaptiveColor{Light: "#c6c6c6", Dark: "#3a3a3a"},
		Quiet:     lipgloss.AdaptiveColor{Light: "#8a8a8a", Dark: "#7d7d7d"},
		Primary:   lipgloss.AdaptiveColor{Light: "#4a4a4a", Dark: "#c8c8c8"},
		Secondary: lipgloss.AdaptiveColor{Light: "#5a6b7d", Dark: "#8fa3b8"},
		Accent:    lipgloss.AdaptiveColor{Light: "#0057d8", Dark: "#7aa2f7"},
		Resting:   lipgloss.AdaptiveColor{Light: "#7a8ba6", Dark: "#4d5a78"},
		Key:       lipgloss.AdaptiveColor{Light: "#8f4700", Dark: "#e0af68"},
		Ground:    lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#16161e"},
		Success:   lipgloss.AdaptiveColor{Light: "#006600", Dark: "#9ece6a"},
		Warn:      lipgloss.AdaptiveColor{Light: "#8f6a00", Dark: "#e0af68"},
		Danger:    lipgloss.AdaptiveColor{Light: "#a00000", Dark: "#f7768e"},
		Cycle:     lipgloss.AdaptiveColor{Light: "#6b21a8", Dark: "#bb9af7"},
		Begun:     lipgloss.AdaptiveColor{Light: "#8f6a00", Dark: "#e0af68"},
		Flight:    lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#73daca"},
		Spent:     lipgloss.AdaptiveColor{Light: "#767676", Dark: "#8a92b2"},
		Voice:     lipgloss.AdaptiveColor{Light: "#9d174d", Dark: "#ff9ac1"},
		Judge:     lipgloss.AdaptiveColor{Light: "#6b21a8", Dark: "#bb9af7"},
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

	App   lipgloss.Style // the app's own name, in a header
	Crumb lipgloss.Style // where you are within it
	// Chevron separates the crumbs. Drawn as the words are, not fainter: it is
	// part of the phrase rather than a rule between two things, and a separator
	// pale enough to disappear leaves two words sitting next to each other for
	// no reason anyone can see.
	Chevron  lipgloss.Style
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
	// Selection is text picked out in a field, drawn as a band rather than
	// reversed: reversed is what the caret is, and a selection that looks like
	// the caret leaves you unable to tell where typing would land.
	Selection lipgloss.Style
	Success   lipgloss.Style
	Warn      lipgloss.Style
	Danger    lipgloss.Style
	Cycle     lipgloss.Style
	Begun     lipgloss.Style
	Flight    lipgloss.Style
	Spent     lipgloss.Style
	Voice     lipgloss.Style
	Judge     lipgloss.Style
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
		Chevron:  fg(p.Text),
		Heading:  fg(p.Dim).Bold(true),
		Item:     fg(p.Text),
		Selected: fg(p.Accent).Bold(true),
		Desc:     fg(p.Dim),
		Label:    fg(p.Quiet),
		Value:    fg(p.Text),
		Rule:     fg(p.Faint),
		KeyName:  fg(p.Key).Bold(true),
		KeyDesc:  fg(p.Dim),
		Cursor:   fg(p.Dim),
		Selection: lipgloss.NewStyle().
			Foreground(p.Ground).Background(p.Accent),
		Success: fg(p.Success),
		Warn:    fg(p.Warn),
		Danger:  fg(p.Danger),
		Cycle:   fg(p.Cycle),
		Begun:   fg(p.Begun),
		Flight:  fg(p.Flight),
		Spent:   fg(p.Spent),
		Voice:   fg(p.Voice),
		Judge:   fg(p.Judge),
	}
}

// Default is what an app uses when it has no reason to customise.
func Default() Styles { return New(DefaultPalette()) }

// Rest is how far a resting palette moves toward the background: a light,
// even hand.
//
// One fraction for every colour, because the eye reads a panel as resting from
// the fact that all of it went quiet together. Moving some colours a long way
// and others hardly at all reads as a panel with something wrong on it rather
// than one that is merely not in use.
const Rest = 0.3

// AtRest is this palette as an app draws itself when the keys are somewhere
// else: every colour a fixed step toward the background.
//
// Toward the background rather than toward grey, so that what a colour means
// survives the trip: a red at seven tenths is still plainly red, and a panel you
// are not typing into is still one whose broken service you can see.
func (p Palette) AtRest() Palette {
	q := p
	for _, c := range []*AdaptiveColor{
		&q.Text, &q.Dim, &q.Faint, &q.Quiet, &q.Primary, &q.Secondary,
		&q.Accent, &q.Key, &q.Resting,
		&q.Success, &q.Warn, &q.Danger, &q.Cycle,
		&q.Begun, &q.Flight, &q.Spent, &q.Voice, &q.Judge,
	} {
		*c = fade(*c, p.Ground, Rest)
	}
	return q
}

// Fill is a colour shown at part of its strength: the hue blended that far from
// the background toward itself, at Level 0 the background and at 1 the hue.
//
// This is how a control says how live it is. A terminal has no alpha — there is
// no such escape as "red at a fifth" — but the background is known, so the same
// picture can be computed and sent as one solid colour. What a browser would do
// with opacity, this does with arithmetic.
func (p Palette) Fill(c AdaptiveColor, level float64) AdaptiveColor {
	// fade measures the distance travelled toward the ground; a level measures
	// how much of the hue is left, which is the other end of the same journey.
	return fade(c, p.Ground, 1-clampF(level, 0, 1))
}

// On is the colour to write on a fill: Ground or Text, whichever can actually
// be read against it.
//
// Chosen by measuring rather than by a rule of thumb, and per variant, because
// the answer flips partway up the ramp and flips at a different point for every
// hue. A mid-tone fill is where a guess goes wrong: too dark for the ground and
// too light for the text, and the label that looked fine on one tone is illegible
// on the next.
func (p Palette) On(fill AdaptiveColor) AdaptiveColor {
	pick := func(bg, a, b string) string {
		if contrast(bg, a) >= contrast(bg, b) {
			return a
		}
		return b
	}
	return AdaptiveColor{
		Light: pick(fill.Light, p.Ground.Light, p.Text.Light),
		Dark:  pick(fill.Dark, p.Ground.Dark, p.Text.Dark),
	}
}

// contrast is the WCAG ratio between two hexes, 1 for identical and 21 for
// black against white.
func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// luminance is WCAG relative luminance. An unreadable colour comes back as 0,
// which makes it the darkest thing there is rather than a panic.
func luminance(hex string) float64 {
	r, g, b, ok := split(hex)
	if !ok {
		return 0
	}
	lin := func(v int) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// AdaptiveColor is lipgloss's, named here so the list above reads as one thing.
type AdaptiveColor = lipgloss.AdaptiveColor

// fade moves a colour that far toward the ground it is drawn on — the terminal's
// own background, not pure black, or a dark theme's text would fade past where
// the screen actually ends.
//
// A colour it cannot read is returned as it came. Every colour in this file is a
// six-digit hex, so that is a guard rather than a case.
func fade(c, ground AdaptiveColor, by float64) AdaptiveColor {
	return AdaptiveColor{
		Light: toward(c.Light, ground.Light, by),
		Dark:  toward(c.Dark, ground.Dark, by),
	}
}

func toward(hex, ground string, by float64) string {
	r, g, b, ok := split(hex)
	gr, gg, gb, okg := split(ground)
	if !ok || !okg {
		return hex
	}
	mix := func(v, to int) int { return int(float64(v) + (float64(to)-float64(v))*by) }
	return fmt.Sprintf("#%02x%02x%02x", mix(r, gr), mix(g, gg), mix(b, gb))
}

func split(hex string) (r, g, b int, ok bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	n, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b, n == 3 && err == nil
}

// AtRest is these styles, drawn for an app whose keys are elsewhere.
func (s Styles) AtRest() Styles { return New(s.Palette.AtRest()) }

// ForRow returns the styles a list row draws with.
//
// Selected, every part of it takes the accent. A row picked out in one place
// and not another reads as two rows: the eye follows the colour, so a blue
// title beside a grey date and a green dot says the title is the selection
// rather than the line. Nothing in a row is worth more than knowing which row
// you are on, so while it is selected the row says only that.
//
// The semantic colours go with it, which is the cost: a failing thing on the
// selected row is blue like everything else on that row, and says what it is
// through its own text and through the preview below.
func (s Styles) ForRow(selected bool) Styles {
	if !selected {
		return s
	}
	for _, f := range []*lipgloss.Style{
		&s.Item, &s.Desc, &s.Label, &s.Value,
		&s.Success, &s.Warn, &s.Danger, &s.Cycle,
		&s.Begun, &s.Flight, &s.Spent, &s.Voice, &s.Judge,
	} {
		*f = s.Selected
	}
	return s
}
