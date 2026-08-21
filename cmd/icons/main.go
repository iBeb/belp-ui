// Command icons shows candidate glyphs so they can be chosen by eye.
//
// Nobody can pick an icon from a codepoint, and a missing glyph renders as a box
// that no runtime check can tell from a real one — so the only honest test is to
// print them and look. Every candidate here was read out of the installed Nerd
// Font's own name table, so the codepoints are the font's, not a guess.
//
// If you see boxes rather than pictures: make font.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iBeb/belp-ui/theme"
)

// candidate is one glyph on offer.
//
// The rune is written as an escape, never as the character: a private-use glyph
// does not survive every editor, clipboard and diff it passes through.
type candidate struct {
	r    rune
	name string
}

func (c candidate) glyph() string { return string(c.r) }

func (c candidate) label() string { return fmt.Sprintf("%s %04X", c.name, c.r) }

// choices are the alternatives for each kind of activity, best guess first.
var choices = []struct {
	activity string
	options  []candidate
}{
	{"commit", []candidate{
		{0xF417, "oct-git_commit"},
		{0xF4B6, "oct-commit"},
		{0xEAFC, "cod-git_commit"},
		{0xE729, "dev-git_commit"},
		{0xF172, "fa-code_commit"},
	}},
	{"push", []candidate{
		{0xF403, "oct-repo_push"},
		{0xEB41, "cod-repo_push"},
		{0xF431, "oct-arrow_up"},
		{0xF40A, "oct-upload"},
		{0xF093, "fa-upload"},
	}},
	{"branch created", []candidate{
		{0xF418, "oct-git_branch"},
		{0xEC6F, "cod-git_branch"},
		{0xE725, "dev-git_branch"},
	}},
	{"branch deleted", []candidate{
		{0xF48E, "oct-trash"},
		{0xEA81, "cod-trash"},
		{0xF52F, "oct-x_circle"},
		{0xF014, "fa-trash_o"},
	}},
	{"pr opened", []candidate{
		{0xF407, "oct-git_pull_request"},
		{0xEA64, "cod-git_pull_request"},
		{0xEBBC, "cod-git_pull_request_create"},
		{0xE726, "dev-git_pull_request"},
	}},
	{"pr merged", []candidate{
		{0xF419, "oct-git_merge"},
		{0xEAFE, "cod-git_merge"},
		{0xE727, "dev-git_merge"},
		{0xF17F, "fa-code_merge"},
	}},
	{"pr closed", []candidate{
		{0xF4DC, "oct-git_pull_request_closed"},
		{0xEBDA, "cod-git_pull_request_closed"},
		{0xF52F, "oct-x_circle"},
		{0xF41D, "oct-issue_closed"},
	}},
	{"reviewed", []candidate{
		{0xF441, "oct-eye"},
		{0xEA70, "cod-eye"},
		{0xF49E, "oct-check_circle"},
		{0xF45E, "oct-checklist"},
		{0xEAB3, "cod-checklist"},
	}},
	{"commented", []candidate{
		{0xF41F, "oct-comment"},
		{0xF442, "oct-comment_discussion"},
		{0xEA6B, "cod-comment"},
		{0xEAC7, "cod-comment_discussion"},
		{0xF075, "fa-comment"},
	}},
}

func main() {
	light := flag.Bool("light", false, "render as if the terminal were light")
	flag.Parse()

	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(!*light)
	s := theme.Default()

	fmt.Println(s.App.Render("belp") + " " + s.Chevron.Render(theme.Chevron) + " " +
		s.Crumb.Render("Icons") + "  " +
		s.Desc.Render("boxes instead of pictures? make font"))
	fmt.Println(s.Rule.Render(strings.Repeat("─", 78)))
	fmt.Println()

	for _, c := range choices {
		fmt.Print(s.Label.Render(fmt.Sprintf("%16s", c.activity)) + "  ")
		for _, o := range c.options {
			fmt.Print(s.Value.Render(o.glyph()) + " " +
				s.Desc.Render(fmt.Sprintf("%-33s", o.label())))
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println(s.Heading.Render("the chip bar, first choice of each"))
	fmt.Println(chipBar(s))
	fmt.Println()
	fmt.Println(s.Heading.Render("the kind column, three ways"))
	for _, line := range rows(s) {
		fmt.Println(line)
	}
}

// first is the best-guess glyph for an activity, which is what the mocks draw.
func first(activity string) string {
	for _, c := range choices {
		if c.activity == activity {
			return c.options[0].glyph()
		}
	}
	return "?"
}

// chipBar mocks the filter bar with icons: brackets still mark the cursor, colour
// still marks what is chosen, and a one-cell glyph keeps the padding trick that
// stops the bar shifting.
func chipBar(s theme.Styles) string {
	type chip struct {
		glyph    string
		selected bool
		focused  bool
	}
	code := []chip{
		{first("commit"), true, false}, {first("push"), false, true},
		{first("branch created"), false, false}, {first("branch deleted"), false, false},
	}
	pr := []chip{
		{first("pr opened"), false, false}, {first("pr merged"), true, false},
		{first("pr closed"), false, false}, {first("reviewed"), false, false},
		{first("commented"), false, false},
	}

	draw := func(chips []chip) string {
		var b strings.Builder
		for _, c := range chips {
			style := s.Desc
			if c.selected {
				style = s.Selected
			}
			body := " " + c.glyph + " "
			if c.focused {
				body = "[" + c.glyph + "]"
			}
			b.WriteString(style.Render(body))
		}
		return b.String()
	}
	return "  " + s.Label.Render("code") + " " + draw(code) +
		s.Rule.Render("  ·  ") + s.Label.Render("pr") + " " + draw(pr) +
		s.Rule.Render("  ·  ") + s.Desc.Render(" 7d ") + s.Selected.Render(" 30d ") +
		s.Desc.Render(" 90d  all ")
}

// rows shows the same feed line with the kind spelled out, as an icon and a word,
// and as an icon alone — which is the choice the column has to make.
func rows(s theme.Styles) []string {
	type sample struct {
		icon, word, ticket, where, title string
		style                            lipgloss.Style
	}
	samples := []sample{
		{first("pr merged"), "pr merged", "ABC-5242", "SV#561", "split the sub-record type", s.Success},
		{first("push"), "pushed", "ABC-6455", "SV", "ABC-6455_Cache-Resolver", s.Value},
		{first("branch deleted"), "branch deleted", "ABC-5242", "SV", "ABC-5242-tag-list", s.Danger},
		{first("reviewed"), "reviewed", "ABC-3154", "SV#845", "legacy concept retirement", s.Desc},
	}

	var out []string
	for _, label := range []string{"words only", "icon and word", "icon only"} {
		out = append(out, "", s.Label.Render("  "+label))
		for _, x := range samples {
			var kind string
			switch label {
			case "words only":
				kind = x.style.Render(fmt.Sprintf("%-17s", x.word))
			case "icon and word":
				kind = x.style.Render(x.icon + " " + fmt.Sprintf("%-15s", x.word))
			default:
				kind = x.style.Render(x.icon) + "  "
			}
			out = append(out, "  "+s.Desc.Render("today 13:58  ")+kind+" "+
				s.Value.Render(fmt.Sprintf("%-10s", x.ticket))+" "+
				s.Desc.Render(fmt.Sprintf("%-8s", x.where))+" "+
				s.Item.Render(x.title))
		}
	}
	return out
}
