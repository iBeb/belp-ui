// Command browse renders a browse screen so it can be judged by eye.
//
// Tests assert widths and row numbers; they cannot say whether it looks right.
// The rows are invented — the frame is the point. -h shows it getting shorter,
// -focus moves the cursor between the bands.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/iBeb/belp-ui/browse"
	"github.com/iBeb/belp-ui/theme"
)

func main() {
	width := flag.Int("w", 104, "terminal width")
	height := flag.Int("h", 30, "terminal height; try 9 to see the preview go")
	light := flag.Bool("light", false, "render as if the terminal were light")
	focus := flag.String("focus", "list", "band with the focus: list, search, filters or prompt")
	flag.Parse()

	// Forced, so the output is the same through a pipe, in CI and on screen.
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(!*light)

	s := theme.Default()
	l := browse.Compute(*width, *height, false)

	fmt.Printf("%dx%d — list %d row(s), preview %d row(s)\n\n",
		l.Width, l.Height, l.List.Height, l.Preview.Height)
	fmt.Println(chrome(s, *focus).Render(l, rows(s, l.List.Height, l.Width), preview(s)))
}

func chrome(s theme.Styles, focus string) browse.Chrome {
	c := browse.Chrome{
		Styles: s,
		App:    "belp",
		Crumbs: []string{"Sillage"},
		Status: "77 pull requests · 50 of 163 reviewed",
		Groups: []browse.Group{
			{Label: "did", Options: []browse.Option{
				{Label: "opened", Selected: true},
				{Label: "reviewed", Selected: true},
				{Label: "merged"},
				{Label: "mentioned"},
			}},
			{Label: "is", Options: []browse.Option{
				{Label: "open"},
				{Label: "merged"},
				{Label: "closed"},
			}},
			{Options: []browse.Option{
				{Label: "7 days"},
				{Label: "30 days", Selected: true},
				{Label: "all"},
			}},
		},
		Query:       "geo",
		Placeholder: "type to filter",
		Focus:       browse.FocusList,
		Keys: []browse.Key{
			{Name: "↵", Desc: "open"},
			{Name: "^R", Desc: "review"},
			{Name: "^L", Desc: "reload"},
			{Name: "^U", Desc: "clear"},
			{Name: "^Q", Desc: "quit"},
		},
	}

	// The focus is what decides where the caret is and which chip is bracketed,
	// so it is the one piece of state worth being able to move from the command
	// line: those are the parts a test can only say are present, not that they
	// landed anywhere sensible.
	switch focus {
	case "search":
		c.Focus = browse.FocusSearch
	case "filters":
		c.Focus = browse.FocusFilters
		c.Groups[0].Options[2].Focused = true
	case "prompt":
		c.Focus = browse.FocusPrompt
		c.Prompt = browse.Prompt{Label: "title: ", Text: "site ABC-8714_Consolidate"}
	}
	return c
}

// rows is what an app supplies: already styled, already its own business.
//
// The title is elided to the room left over rather than left to be cut by the
// frame. An app knows which part of its row matters and should say so; the frame
// only knows where the edge is.
func rows(s theme.Styles, n, width int) []string {
	type row struct {
		when, state, roles, ref, title string
	}
	sample := []row{
		{"11.08 17:38", "merged", "reviewed", "ui#3405", "fix: refresh the summary counters after an edit"},
		{"11.08 16:58", "open", "reviewed", "sv#385", "feat(ontology): register the tag vocabulary"},
		{"11.08 11:14", "merged", "opened", "sv#757", "chore: point the sync hint at the new log-service path"},
		{"11.08 11:14", "merged", "opened, merged", "log#393", "refactor: consolidate geography handling into one package"},
		{"10.08 17:32", "merged", "reviewed, merged", "sv#974", "chore(deps): bump gradio from 6.0.2 to 6.15.1"},
		{"10.08 10:08", "open", "reviewed", "infra#436", "feat: alert on unexpected node count"},
		{"07.08 11:11", "open", "reviewed", "sv#715", "test: run the suite in a container via a Dockerfile stage"},
		{"07.08 10:13", "merged", "opened, merged", "app#11", "docs: file integration reference per integration"},
		{"28.07 18:01", "closed", "reviewed", "log#203", "refactor: scope the plugin list to its only caller"},
		{"14.07 10:51", "closed", "assigned", "ui#7947", "restructure the report tab and its summary section"},
	}

	// 2 for the marker, then the four fixed columns.
	const columns = 2 + 12 + 8 + 17 + 13
	titleW := width - columns

	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		r := sample[i%len(sample)]
		style, marker := s.Item, "  "
		if i == 3 { // the row the cursor is on
			style, marker = s.Selected, "▸ "
		}
		out = append(out,
			marker+
				s.Desc.Render(pad(r.when, 12))+
				s.Desc.Render(pad(r.state, 8))+
				s.Value.Render(pad(r.roles, 17))+
				s.Desc.Render(pad(r.ref, 13))+
				style.Render(cut(r.title, titleW)))
	}
	return out
}

// cut shortens to w cells, saying so when it had to.
func cut(s string, w int) string {
	if w <= 1 {
		return ""
	}
	if r := []rune(s); len(r) > w {
		return string(r[:w-1]) + "…"
	}
	return s
}

// preview is the detail of the row the cursor is on: labels dimmed and
// right-aligned, values against them, which is the shape a field list wants.
func preview(s theme.Styles) []string {
	const labelW = 12
	field := func(label, value string) string {
		return s.Label.Render(fmt.Sprintf("%*s", labelW, label)) + "  " + value
	}
	return []string{
		s.Selected.Render("refactor: consolidate geography handling into one package") +
			"  " + s.Desc.Render("log#393"),
		"",
		field("roles", s.Value.Render("opened, reviewed, commented, merged")),
		field("state", s.Success.Render("merged")+s.Desc.Render(" 11.08 11:14 by you")),
		field("checks", s.Success.Render("success")+s.Desc.Render("   review  APPROVED")),
		field("diff", s.Value.Render("+840 −434")+s.Desc.Render(" over 28 files")),
		field("branch", s.Desc.Render("main ← ")+s.Value.Render("ABC-2471_Consolidate-Regions")),
		field("my review", s.Desc.Render("10.08 15:39 ")+s.Value.Render("COMMENTED")),
	}
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s[:w-1] + " "
	}
	return s + strings.Repeat(" ", w-len(s))
}
