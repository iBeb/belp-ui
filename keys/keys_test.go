package keys

import (
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func app() Map {
	return Map{
		"open":    {Key: []string{"enter"}, Desc: "open"},
		"refresh": {Key: []string{"ctrl+r"}, Desc: "refresh"},
		"tab":     {Key: []string{"alt+enter", "ctrl+f"}, Desc: "in a tab"},
		"quit":    {Key: []string{"ctrl+c"}, Desc: "quit", Hide: true},
	}
}

func press(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "ctrl+f":
		return tea.KeyMsg{Type: tea.KeyCtrlF}
	case "ctrl+l":
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	case "alt+enter":
		return tea.KeyMsg{Type: tea.KeyEnter, Alt: true}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestAKeyPerformsTheActionItIsBoundTo(t *testing.T) {
	m := app()
	if !m.Does("refresh", press("ctrl+r")) {
		t.Error("^R did not refresh")
	}
	if m.Does("refresh", press("ctrl+l")) {
		t.Error("^L refreshed, and nothing binds it")
	}
	if m.Does("nonesuch", press("ctrl+r")) {
		t.Error("an action nobody named was performed")
	}
}

// Several spellings for one action, which is how a key that some terminals eat
// can have a second that they do not.
func TestAnActionMayHaveMoreThanOneKey(t *testing.T) {
	m := app()
	for _, k := range []string{"alt+enter", "ctrl+f"} {
		if !m.Does("tab", press(k)) {
			t.Errorf("%s did not open in a tab", k)
		}
	}
	// And the footer shows the first, which is the one to reach for.
	if got := m.Footer("tab"); len(got) != 1 || got[0].Name != "⌥↵" {
		t.Errorf("footer = %+v, want ⌥↵", got)
	}
}

// The order is the app's: the key reached for most goes first, and a map has no
// order to offer.
func TestTheFooterIsInTheOrderTheAppNames(t *testing.T) {
	got := app().Footer("refresh", "open")
	if len(got) != 2 || got[0].Desc != "refresh" || got[1].Desc != "open" {
		t.Errorf("footer = %+v, want refresh then open", got)
	}
}

// Hidden keeps a binding working and out of the footer — for the ones everybody
// already knows.
func TestHiddenStillWorks(t *testing.T) {
	m := app()
	if !m.Does("quit", press("ctrl+c")) {
		t.Error("a hidden binding stopped working")
	}
	for _, k := range m.Footer("quit", "open") {
		if k.Desc == "quit" {
			t.Error("a hidden binding is in the footer")
		}
	}
}

// A config says which key, not what the key is for, so it should not have to
// repeat the description.
func TestAnOverrideKeepsWhatTheAppCalledIt(t *testing.T) {
	m := app().With(map[string]Binding{"refresh": {Key: []string{"ctrl+l"}}})

	if !m.Does("refresh", press("ctrl+l")) {
		t.Error("the new key does not refresh")
	}
	if m.Does("refresh", press("ctrl+r")) {
		t.Error("the old key still refreshes — an override replaces rather than adds")
	}
	if got := m.Footer("refresh"); len(got) != 1 || got[0].Desc != "refresh" || got[0].Name != "^L" {
		t.Errorf("footer = %+v, want the app's words and the config's key", got)
	}
}

// An override with no keys takes the action away, which is how a person says
// they do not want it.
func TestAnOverrideWithNoKeyTakesTheActionAway(t *testing.T) {
	m := app().With(map[string]Binding{"refresh": {}})
	if m.Does("refresh", press("ctrl+r")) {
		t.Error("an action given no key still answers one")
	}
	if len(m.Footer("refresh")) != 0 {
		t.Error("an action given no key is still in the footer")
	}
}

// Overriding one binding must not disturb the others.
func TestOverridingOneLeavesTheRest(t *testing.T) {
	m := app().With(map[string]Binding{"refresh": {Key: []string{"ctrl+l"}}})
	if !m.Does("open", press("enter")) {
		t.Error("open stopped working when refresh was rebound")
	}
}

// The footer's spelling of a key is the one people read, not the one bubbletea
// writes.
func TestKeysAreDrawnTheWayAFooterWritesThem(t *testing.T) {
	for _, c := range []struct{ key, want string }{
		{"enter", "↵"},
		{"ctrl+r", "^R"},
		{"alt+enter", "⌥↵"},
		{"shift+tab", "⇧⇥"},
		{"up", "↑"},
		{"?", "?"},
	} {
		if got := Draw(c.key); got != c.want {
			t.Errorf("Draw(%q) = %q, want %q", c.key, got, c.want)
		}
	}
	// And an app may say how a key is drawn where the default reads badly.
	b := Binding{Key: []string{"ctrl+x"}, Name: "^X ^X", Desc: "twice"}
	if got := b.Label(); got != "^X ^X" {
		t.Errorf("Label() = %q, want the name the app gave", got)
	}
}

// A binding is what a config file holds, so it has to survive being one.
func TestABindingIsWhatAConfigHolds(t *testing.T) {
	var over map[string]Binding
	if err := json.Unmarshal([]byte(`{
	  "refresh": { "key": ["ctrl+l"], "desc": "reload" },
	  "trash":   { "key": ["ctrl+d"], "hide": true }
	}`), &over); err != nil {
		t.Fatal(err)
	}
	m := app().With(over)
	if !m.Does("refresh", press("ctrl+l")) {
		t.Error("the config's key does not work")
	}
	if got := m.Footer("refresh"); len(got) != 1 || got[0].Desc != "reload" {
		t.Errorf("footer = %+v, want the config's words where it gave some", got)
	}
	if !strings.Contains(strings.Join([]string{m["trash"].Key[0]}, ""), "ctrl+d") {
		t.Error("an action the config added is not there")
	}
}
