// Command preview renders the theme so it can be judged by eye.
//
// Both variants are shown side by side. lipgloss normally picks one by probing
// the terminal's background, which means you only ever see half the palette on
// your own machine — and the half you cannot see is the half that ships broken.
package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iBeb/belp-ui/theme"
)

func main() {
	// Forced, so the output is identical through a pipe, in CI, and on screen.
	lipgloss.SetColorProfile(termenv.TrueColor)

	fmt.Println(swatches())
	fmt.Println()
	fmt.Println(samples())
}

// swatches shows each colour twice: as it renders on a light terminal and on a
// dark one, with the hex values that produced them.
func swatches() string {
	p := theme.DefaultPalette()
	v := reflect.ValueOf(p)

	rows := []string{
		lipgloss.NewStyle().Bold(true).Render(
			pad("colour", 10) + pad("on light", 22) + pad("on dark", 22)),
	}
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		c, ok := v.Field(i).Interface().(lipgloss.AdaptiveColor)
		if !ok {
			continue
		}
		rows = append(rows, pad(name, 10)+
			pad(chip(c.Light, "#f5f5f5")+" "+c.Light, 22)+
			pad(chip(c.Dark, "#101010")+" "+c.Dark, 22))
	}
	return strings.Join(rows, "\n")
}

// chip is a block of the colour on the background it is meant for, which is the
// only way to tell whether a grey is readable or merely present.
func chip(fg, bg string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Render(" ▇▇ Ag ")
}

// samples renders every style against realistic text, again in both variants.
func samples() string {
	light, dark := render(false), render(true)
	head := lipgloss.NewStyle().Bold(true)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(42).Render(head.Render("light terminal")+"\n"+light),
		lipgloss.NewStyle().Render(head.Render("dark terminal")+"\n"+dark),
	)
}

func render(darkBG bool) string {
	lipgloss.SetHasDarkBackground(darkBG)
	s := theme.Default()
	bg := lipgloss.Color("#f5f5f5")
	if darkBG {
		bg = lipgloss.Color("#101010")
	}
	box := lipgloss.NewStyle().Background(bg).Padding(0, 1).Width(40)

	lines := []string{
		s.App.Render("belp") + " " + s.Chevron.Render(theme.Chevron) + " " + s.Crumb.Render("Go to"),
		s.Rule.Render(strings.Repeat("─", 38)),
		s.Heading.Render("Apps"),
		"  " + s.Item.Render("Recall") + "  " + s.Desc.Render("find a session"),
		s.Selected.Render("▸ Stack") + "  " + s.Desc.Render("the docker stack"),
		"",
		s.Label.Render("     Branch ") + s.Value.Render("main"),
		s.Success.Render("moved") + " " + s.Warn.Render("running") + " " + s.Danger.Render("removed"),
		"",
		// The caret is a block of colour rather than a glyph with a shape, so
		// whether it reads as a cursor or as a hole in the line is a thing only
		// an eye can settle, and only against both backgrounds.
		s.Desc.Render(theme.Magnifier) + " " + s.Value.Render("geo") + s.Cursor.Render(theme.Caret),
		"",
		s.KeyName.Render("↵") + " " + s.KeyDesc.Render("open") + "   " +
			s.KeyName.Render("^G") + " " + s.KeyDesc.Render("grep"),
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = box.Render(l)
	}
	return strings.Join(out, "\n")
}

func pad(s string, w int) string {
	return lipgloss.NewStyle().Width(w).Render(s)
}
