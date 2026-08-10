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
}

// DefaultPalette is deliberately restrained: one accent, three greys, three
// signals. A launcher is glanced at rather than read, so colour is spent on
// the thing you are about to act on and on warning you off the rest.
func DefaultPalette() Palette {
	return Palette{
		Text:    lipgloss.AdaptiveColor{Light: "#1c1c1c", Dark: "#dcdcdc"},
		Dim:     lipgloss.AdaptiveColor{Light: "#6c6c6c", Dark: "#8a8a8a"},
		Faint:   lipgloss.AdaptiveColor{Light: "#c6c6c6", Dark: "#3a3a3a"},
		Accent:  lipgloss.AdaptiveColor{Light: "#0057d8", Dark: "#7aa2f7"},
		Key:     lipgloss.AdaptiveColor{Light: "#8f4700", Dark: "#e0af68"},
		Success: lipgloss.AdaptiveColor{Light: "#006600", Dark: "#9ece6a"},
		Warn:    lipgloss.AdaptiveColor{Light: "#8f6a00", Dark: "#e0af68"},
		Danger:  lipgloss.AdaptiveColor{Light: "#a00000", Dark: "#f7768e"},
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
	Success  lipgloss.Style
	Warn     lipgloss.Style
	Danger   lipgloss.Style
}

// Chevron is the separator between crumbs: a plain one, not a powerline
// triangle. That glyph exists only in a patched font, and an app that renders
// as tofu on a stock terminal is worse than one that renders plainly anywhere.
const Chevron = "›"

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
		Success:  fg(p.Success),
		Warn:     fg(p.Warn),
		Danger:   fg(p.Danger),
	}
}

// Default is what an app uses when it has no reason to customise.
func Default() Styles { return New(DefaultPalette()) }
