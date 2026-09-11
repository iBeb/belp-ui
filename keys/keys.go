// Package keys is what each key does, and whether the footer says so.
//
// An app names its actions once — "refresh", "open", "trash" — and everything
// else follows from a table: which key performs one, how the footer draws it,
// and whether it is drawn at all. That is what makes a binding something a
// person can change rather than something a programmer chose.
//
// The names are bubbletea's own, as KeyMsg.String returns them: "ctrl+r",
// "enter", "alt+enter", "?" — so a config says the key the way the terminal
// says it, and nothing has to keep a second vocabulary in step.
package keys

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iBeb/belp-ui/browse"
)

// Binding is one action.
type Binding struct {
	// Key is what performs it, as bubbletea names it. Several may, which is how
	// a key that some terminals eat can have a second spelling.
	Key []string `json:"key"`
	// Desc is what the footer says it does. Empty keeps it out of the footer
	// whatever Show says: a key with nothing to say about itself is not a hint.
	Desc string `json:"desc"`
	// Name is how the key is drawn — "^R", "⌥↵". Empty draws it from Key.
	Name string `json:"name"`
	// Hide keeps a binding working and out of the footer. For the ones everybody
	// already knows, and for the ones an app offers without advertising.
	Hide bool `json:"hide"`
}

// Map is an app's actions, by name.
type Map map[string]Binding

// Does reports whether this key performs that action.
//
// An action nobody has bound does nothing, rather than falling back to a
// default that the config was trying to take away.
func (m Map) Does(action string, msg tea.KeyMsg) bool {
	b, ok := m[action]
	if !ok {
		return false
	}
	pressed := msg.String()
	for _, k := range b.Key {
		if k == pressed {
			return true
		}
	}
	return false
}

// Footer is the hints, in the order the actions are named.
//
// Named rather than taken whole, because the order is the app's: the key you
// reach for most goes first, and a map has no order to offer.
func (m Map) Footer(actions ...string) []browse.Key {
	var out []browse.Key
	for _, action := range actions {
		b, ok := m[action]
		if !ok || b.Hide || b.Desc == "" || len(b.Key) == 0 {
			continue
		}
		out = append(out, browse.Key{Name: b.Label(), Desc: b.Desc})
	}
	return out
}

// All is every action that would show, for an app that has no order to impose.
func (m Map) All() []browse.Key {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return m.Footer(names...)
}

// Label is how the key is drawn.
func (b Binding) Label() string {
	if b.Name != "" {
		return b.Name
	}
	if len(b.Key) == 0 {
		return ""
	}
	return Draw(b.Key[0])
}

// Draw turns a key's name into the way a footer writes it.
func Draw(key string) string {
	switch key {
	case "enter":
		return "↵"
	case "tab":
		return "⇥"
	case "shift+tab":
		return "⇧⇥"
	case "esc":
		return "esc"
	case "space":
		return "␣"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	}
	if rest, ok := strings.CutPrefix(key, "alt+"); ok {
		return "⌥" + Draw(rest)
	}
	if rest, ok := strings.CutPrefix(key, "ctrl+"); ok {
		return "^" + strings.ToUpper(Draw(rest))
	}
	return key
}

// With is this map, with those bindings replacing the ones of the same name.
//
// Replacing rather than merging field by field: a config that gives an action
// one key means that key and not that key as well, and a half-overridden
// binding is the kind of state nobody can read off the file in front of them.
// An override with no keys at all takes the action away, which is how a person
// says they do not want it.
func (m Map) With(over map[string]Binding) Map {
	out := make(Map, len(m))
	for name, b := range m {
		out[name] = b
	}
	for name, b := range over {
		if b.Desc == "" {
			// Keep what the app called it: a config saying which key, not what
			// the key is for, should not have to repeat the description.
			if old, ok := out[name]; ok {
				b.Desc = old.Desc
			}
		}
		out[name] = b
	}
	return out
}
