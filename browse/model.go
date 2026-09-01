package browse

import (
	tea "github.com/charmbracelet/bubbletea"
)

// RowFunc renders row i to at most width cells, told whether the cursor is on it
// — restyling a line that already carries escape sequences is not reliable.
//
// Called only for rows on screen, so ten thousand rows cost what a screenful does.
type RowFunc func(i, width int, selected bool) string

// PreviewFunc renders the detail of row i into a band of this size.
//
// Called only for the row under the cursor. That is what makes it safe for the
// detail to be expensive — a fetch, a file read — because it happens once per
// row you actually stop on rather than once per row that goes past.
type PreviewFunc func(i, width, height int) []string

// ActivateMsg is Enter on a row: the app is being asked to do the obvious thing
// with it — open it, resume it, whatever the app is for.
type ActivateMsg struct{ Index int }

// SubmitMsg is Enter in the search field.
//
// For a list filtered in memory the query is live and this is noise, so ignoring
// it costs nothing. It exists for the app whose query is a question worth asking
// out loud — one that costs a network round trip, and so wants saying when.
type SubmitMsg struct{ Query string }

// FiltersChangedMsg says a chip was toggled, so whatever the app derives from
// the filters is now stale.
//
// The app reads the new state from Selected; this only says when to look. Sending
// the state itself would mean deciding here what a selection means, and a chip
// labelled "30 days" means something only the app knows.
type FiltersChangedMsg struct{}

// PreviewClickMsg is a click on one line of the preview, counted from the top of
// the band.
//
// A message rather than an activation: the preview is text the app rendered, and
// only the app knows whether the fourth line points at a commit. The band knows
// which line was hit and nothing else.
type PreviewClickMsg struct{ Line int }

// AnsweredMsg is Enter on an open prompt: the answer, and the label it answers.
//
// The label comes back so that an app with more than one question does not have
// to remember which one it asked — the screen already knows.
type AnsweredMsg struct {
	Label string
	Text  string
}

// CancelledMsg is Esc on an open prompt.
//
// A message rather than silence: what an abandoned question undoes is the app's
// business, and something is usually mid-flight by the time it is asked.
type CancelledMsg struct{ Label string }

// ConfirmedMsg is y on an open confirmation window.
type ConfirmedMsg struct{ Label string }

// DismissedMsg is n, Esc or Enter on one.
//
// Enter dismisses rather than confirms: the questions worth a window are the ones
// where a stray keypress must not be consent, and Enter is the key most likely to
// be struck by reflex.
type DismissedMsg struct{ Label string }

// QuitMsg is Esc or ^C.
//
// A message rather than tea.Quit: whether those keys end the program is the
// app's decision, and a component that quits on its own cannot be embedded in
// anything that has more than one screen.
type QuitMsg struct{}

// Model is an interactive browse screen: a filter bar, a search field, a list
// and a preview, with one way of moving between them.
type Model struct {
	// Row and Preview are how the app draws itself. A nil Row draws nothing,
	// which is what an app with no results looks like.
	Row     RowFunc
	Preview PreviewFunc

	chrome Chrome

	count  int // how many rows there are
	cursor int // which one the cursor is on
	top    int // the first row on screen

	// The label of the open confirmation window, so the answer can say which
	// question it answers.
	confirming string

	// Where the focus goes when an open prompt closes, either way. Kept so that
	// answering a question puts you back on the row you asked it of.
	returnTo Focus

	// Where the cursor sits on the filter bar. Kept here rather than as flags
	// on the options so that there is one cursor, not one per group that has to
	// be kept in agreement with the others.
	group  int
	option int

	width  int
	height int
}

// New builds a model that draws with this chrome.
//
// The cursor starts after whatever query the app pre-filled: a field handed to
// you with text already in it is one you are meant to add to, not overwrite.
func New(c Chrome) Model {
	c.Caret = len([]rune(c.Query))
	return Model{chrome: c}
}

// SetRowCount tells the model how many rows there are now.
//
// The cursor is kept in range but not reset: re-filtering a list while pointing
// at something is normal, and jumping back to the top on every keystroke is what
// makes a search field feel like it is fighting you.
func (m *Model) SetRowCount(n int) {
	if n < 0 {
		n = 0
	}
	m.count = n
	if m.cursor >= n {
		m.cursor = max(0, n-1)
	}
	m.clamp()
}

// SetStatus replaces the right-hand side of the header.
func (m *Model) SetStatus(s string) { m.chrome.Status = s }

// SetColumns is the header naming what the columns of the list hold, rendered by
// the app because it has to line up with rows the app draws. An empty line gives
// the row back to the list.
func (m *Model) SetColumns(line string) { m.chrome.Columns = line }

// SetKeys replaces the footer hints.
//
// For a screen that has changed mode: a list showing folders to move a session
// into is still a list, but "↵ resume" is then a lie about what Enter does.
func (m *Model) SetKeys(keys []Key) { m.chrome.Keys = keys }

// Ask opens a question in the footer, with initial already typed into it.
//
// The answer arrives as an AnsweredMsg and an abandoned question as a
// CancelledMsg. Modal while it is open: the arrows stop moving the list, because
// the question is about the row it was asked of and a list that moved underneath
// would answer it about something else.
func (m *Model) Ask(label, initial string) {
	if m.chrome.Focus != FocusPrompt {
		m.returnTo = m.chrome.Focus
	}
	m.chrome.Focus = FocusPrompt
	m.chrome.Prompt = Prompt{Label: label, Text: initial, Caret: len([]rune(initial))}
}

// AskConfirm opens a confirmation window over the list, and answers it with a
// ConfirmedMsg or a DismissedMsg carrying the label.
//
// Modal like a prompt: the arrows stop moving the list, because the question is
// about the row it was asked of.
func (m *Model) AskConfirm(label string, c Confirm) {
	if m.chrome.Focus != FocusConfirm && m.chrome.Focus != FocusPrompt {
		m.returnTo = m.chrome.Focus
	}
	m.confirming = label
	m.chrome.Focus = FocusConfirm
	m.chrome.Confirm = c
}

// Confirming is the label of the open window, or "" when there is none.
func (m Model) Confirming() string {
	if m.chrome.Focus != FocusConfirm {
		return ""
	}
	return m.confirming
}

// Asking is whether a question is open, for an app deciding whether a key of its
// own is a key or a character.
func (m Model) Asking() bool { return m.chrome.Focus == FocusPrompt }

// close puts the question away and the focus back, returning what was asked and
// what had been typed.
func (m *Model) close() (label, text string) {
	label, text = m.chrome.Prompt.Label, m.chrome.Prompt.Text
	m.chrome.Prompt = Prompt{}
	m.chrome.Confirm = Confirm{}
	m.chrome.Focus = m.returnTo
	return label, text
}

// SetSize is what a tea.WindowSizeMsg would set, for a caller that handles the
// message itself.
func (m *Model) SetSize(width, height int) {
	m.width, m.height = width, height
	m.clamp()
}

// Query is what has been typed into the search field.
func (m Model) Query() string { return m.chrome.Query }

// Cursor is the row the cursor is on, or -1 when there are no rows.
func (m Model) Cursor() int {
	if m.count == 0 {
		return -1
	}
	return m.cursor
}

// Focus is the band the keyboard is talking to.
func (m Model) Focus() Focus { return m.chrome.Focus }

// Layout is where the bands are, for an app that wants to know how much room its
// preview has before it goes and fetches anything.
func (m Model) Layout() Layout { return Compute(m.width, m.height, m.chrome.Columns != "") }

// Update handles a message and returns the model to keep.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		return m.key(msg)
	case tea.MouseMsg:
		return m.mouse(msg)
	}
	return m, nil
}

// wheelRows is how far one wheel notch moves the list. Three is the usual
// terminal step; one is sluggish and a whole page loses your place.
const wheelRows = 3

// mouse handles the wheel and clicks.
//
// The app has to ask for mouse events for these to arrive at all — until it
// does, the terminal keeps the wheel and scrolls its own scrollback instead of
// the list. The cost is that the terminal's text selection stops working while
// the app is up, unless you hold the key it reserves for that (Option on macOS).
func (m Model) mouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	l := m.Layout()

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scroll(-wheelRows)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.scroll(wheelRows)
		return m, nil
	}

	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil // releases, motion and other buttons are not ours
	}

	// Clicking a band is the same as arrowing to it.
	switch y := msg.Y; {
	case !l.Filters.Empty() && y == l.Filters.Y:
		m.chrome.Focus = FocusFilters
	// Anywhere on the box, border included: the border is part of what looks
	// like the field, and a click on it that did nothing would read as the field
	// refusing the focus.
	case !l.Search.Empty() && y >= l.Search.Y && y <= l.Search.Bottom():
		m.chrome.Focus = FocusSearch
	case y >= l.List.Y && y <= l.List.Bottom():
		if i := m.top + (y - l.List.Y); i < m.count {
			m.chrome.Focus = FocusList
			m.cursor = i
			m.clamp()
		}
	case !l.Preview.Empty() && y >= l.Preview.Y && y <= l.Preview.Bottom():
		// The preview has no cursor to move, so a click is the only way to act on
		// a line of it. The focus stays where it was: the click was about that
		// line, not about which band the keyboard is talking to.
		line := y - l.Preview.Y
		return m, func() tea.Msg { return PreviewClickMsg{Line: line} }
	}
	return m, nil
}

// scroll moves the viewport and takes the cursor along only as far as it must.
//
// The wheel moves the view, not the selection: tying it to the cursor makes the
// list lurch, because the view then only moves once the highlight is pushed off
// an edge — and does nothing at all at either end.
func (m *Model) scroll(delta int) {
	rows := m.Layout().List.Height
	if m.count == 0 || rows <= 0 {
		return
	}
	m.top = clamp(m.top+delta, 0, max(0, m.count-rows))
	m.cursor = clamp(m.cursor, m.top, min(m.top+rows-1, m.count-1))
}

func (m Model) key(msg tea.KeyMsg) (Model, tea.Cmd) {
	rows := m.Layout().List.Height

	// An open question owns the keyboard, apart from the two keys that end the
	// program: a modal you cannot quit out of is a trap.
	if m.chrome.Focus == FocusConfirm {
		return m.confirm(msg)
	}
	if m.chrome.Focus == FocusPrompt {
		return m.prompt(msg)
	}

	switch msg.String() {
	// q from the list, the way every other full-screen program in a terminal
	// quits. Only from the list: in the search field it is a letter, and on the
	// filter bar it is nothing.
	//
	// It exists because ^Q does not arrive. A multiplexer binds it to quit the
	// whole session and takes it before an app sees it, so the key this used to
	// advertise killed the wrong thing — and every app in this family runs inside
	// one. ^C still arrives, since a multiplexer only claims that inside its own
	// modes.
	case "q":
		switch m.chrome.Focus {
		case FocusList:
			return m, func() tea.Msg { return QuitMsg{} }
		case FocusSearch:
			m.chrome.Query += "q"
		}

	// ^C and ^Q, never Esc. Esc is too useful inside an app — backing out of a
	// field, cancelling an overlay — to spend on the one action you cannot undo,
	// and it is one stray keypress away at all times.
	//
	// ^Q is XON, which a cooked terminal eats before an app sees it. Bubbletea
	// puts the terminal in raw mode, which clears IXON, so it arrives — where a
	// multiplexer has not taken it first.
	case "ctrl+q", "ctrl+c":
		return m, func() tea.Msg { return QuitMsg{} }

	// The bands are one selection model, not three: the search field is the row
	// above the first row and the filter bar the row above that, so ↑ and ↓ are
	// the only thing to learn.
	case "up", "ctrl+p":
		switch {
		case m.chrome.Focus == FocusFilters:
			// Already at the top.
		case m.chrome.Focus == FocusSearch:
			m.chrome.Focus = FocusFilters
		case m.cursor > 0:
			m.cursor--
		default:
			m.chrome.Focus = FocusSearch
		}
	case "down", "ctrl+n":
		switch {
		case m.chrome.Focus == FocusFilters:
			m.chrome.Focus = FocusSearch
		case m.chrome.Focus == FocusSearch:
			m.chrome.Focus = FocusList
		case m.cursor < m.count-1:
			m.cursor++
		}

	// Along the chips on the filter bar, and through the text in the search
	// field. Same keys, because in both bands they mean "the next thing to the
	// left", and the band already says which thing that is.
	case "left":
		switch m.chrome.Focus {
		case FocusFilters:
			m.moveChip(-1)
		case FocusSearch:
			m.chrome.Caret = moveAt(m.chrome.Query, m.chrome.Caret, -1)
		}
	case "right":
		switch m.chrome.Focus {
		case FocusFilters:
			m.moveChip(1)
		case FocusSearch:
			m.chrome.Caret = moveAt(m.chrome.Query, m.chrome.Caret, 1)
		}

	case "enter", " ":
		switch m.chrome.Focus {
		case FocusFilters:
			m.toggleChip()
			return m, func() tea.Msg { return FiltersChangedMsg{} }
		case FocusSearch:
			if msg.String() == " " {
				m.chrome.Query, m.chrome.Caret = insertAt(m.chrome.Query, m.chrome.Caret, " ")
				break
			}
			// Down into the list, which is what Enter means in a search field —
			// and a submission, for the app whose query costs something to ask.
			m.chrome.Focus = FocusList
			q := m.chrome.Query
			return m, func() tea.Msg { return SubmitMsg{Query: q} }
		case FocusList:
			if m.count > 0 {
				i := m.cursor
				m.clamp()
				return m, func() tea.Msg { return ActivateMsg{Index: i} }
			}
		}

	case "pgup":
		if m.chrome.Focus == FocusList {
			m.cursor = max(0, m.cursor-rows)
		}
	case "pgdown":
		if m.chrome.Focus == FocusList {
			m.cursor = min(m.count-1, m.cursor+rows)
		}
	case "home":
		switch m.chrome.Focus {
		case FocusList:
			m.cursor = 0
		case FocusSearch:
			m.chrome.Caret = 0
		}
	case "end":
		switch m.chrome.Focus {
		case FocusList:
			m.cursor = max(0, m.count-1)
		case FocusSearch:
			m.chrome.Caret = len([]rune(m.chrome.Query))
		}

	// ^U clears from anywhere, because it is advertised in the footer and a key
	// in the footer that only works in one band is a key that looks broken.
	case "ctrl+u":
		m.chrome.Query, m.chrome.Caret = "", 0

	case "backspace":
		if m.chrome.Focus == FocusSearch {
			m.chrome.Query, m.chrome.Caret = backspaceAt(m.chrome.Query, m.chrome.Caret)
		}
	case "ctrl+w":
		if m.chrome.Focus == FocusSearch {
			m.chrome.Query, m.chrome.Caret = dropWordAt(m.chrome.Query, m.chrome.Caret)
		}

	default:
		// Typing reaches the query only while the search field has the focus.
		// Otherwise every letter would be both a search term and a shortcut,
		// and the list could never have single-key bindings of its own.
		if m.chrome.Focus == FocusSearch && msg.Type == tea.KeyRunes {
			m.chrome.Query, m.chrome.Caret = insertAt(m.chrome.Query, m.chrome.Caret, string(msg.Runes))
		}
	}

	m.clamp()
	return m, nil
}

// confirm is the keyboard while a confirmation window is open.
//
// Only three keys mean anything: y, n and Esc. Everything else is swallowed
// rather than passed through, because a window is a question and typing at it is
// not an answer — but it must not be answered by accident either.
func (m Model) confirm(msg tea.KeyMsg) (Model, tea.Cmd) {
	label := m.confirming
	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return m, func() tea.Msg { return QuitMsg{} }
	case "y", "Y":
		m.close()
		m.confirming = ""
		return m, func() tea.Msg { return ConfirmedMsg{Label: label} }
	case "n", "N", "esc", "enter":
		m.close()
		m.confirming = ""
		return m, func() tea.Msg { return DismissedMsg{Label: label} }
	}
	return m, nil
}

// prompt is the keyboard while a typed question is open.
//
// The same line editor as the search field, down to the arrows walking the
// cursor through the answer: one editor to learn, and ^U clearing the answer
// rather than the query it is drawn over.
func (m Model) prompt(msg tea.KeyMsg) (Model, tea.Cmd) {
	text, caret := m.chrome.Prompt.Text, m.chrome.Prompt.Caret

	switch msg.String() {
	case "ctrl+q", "ctrl+c":
		return m, func() tea.Msg { return QuitMsg{} }

	case "enter":
		label, text := m.close()
		return m, func() tea.Msg { return AnsweredMsg{Label: label, Text: text} }

	// Esc, which is free precisely because quitting is not spent on it.
	case "esc":
		label, _ := m.close()
		return m, func() tea.Msg { return CancelledMsg{Label: label} }

	case "left":
		caret = moveAt(text, caret, -1)
	case "right":
		caret = moveAt(text, caret, 1)
	case "home":
		caret = 0
	case "end":
		caret = len([]rune(text))

	case "ctrl+u":
		text, caret = "", 0
	case "ctrl+w":
		text, caret = dropWordAt(text, caret)
	case "backspace":
		text, caret = backspaceAt(text, caret)

	default:
		// Space arrives as a rune here rather than as the list's Enter-alike:
		// inside an answer it is a character like any other.
		switch msg.Type {
		case tea.KeySpace:
			text, caret = insertAt(text, caret, " ")
		case tea.KeyRunes:
			text, caret = insertAt(text, caret, string(msg.Runes))
		}
	}

	m.chrome.Prompt.Text, m.chrome.Prompt.Caret = text, caret
	return m, nil
}

// moveChip walks the cursor along the whole bar, crossing group boundaries, so
// the bar behaves like one row of chips rather than a set of separate widgets.
func (m *Model) moveChip(delta int) {
	flat := m.flatChips()
	if len(flat) == 0 {
		return
	}
	at := 0
	for i, c := range flat {
		if c.group == m.group && c.option == m.option {
			at = i
			break
		}
	}
	at = clamp(at+delta, 0, len(flat)-1)
	m.group, m.option = flat[at].group, flat[at].option
}

// toggleChip flips the chip under the cursor.
//
// Copy on write, never in place: a Model is passed around by value, so several
// copies share one Groups backing array and mutating it would change the filters
// on every copy — including the one the caller still holds.
func (m *Model) toggleChip() {
	if m.group >= len(m.chrome.Groups) {
		return
	}
	groups := make([]Group, len(m.chrome.Groups))
	copy(groups, m.chrome.Groups)

	g := groups[m.group]
	if m.option >= len(g.Options) {
		return
	}
	options := make([]Option, len(g.Options))
	copy(options, g.Options)

	if g.Exclusive {
		for i := range options {
			options[i].Selected = i == m.option
		}
	} else {
		options[m.option].Selected = !options[m.option].Selected
	}

	g.Options = options
	groups[m.group] = g
	m.chrome.Groups = groups
}

// Groups is the filter state, for the app to read after a FiltersChangedMsg.
//
// Deep: copying the groups alone leaves every Options slice pointing at the
// model's own, so writing to what looked like a copy would change the filters.
func (m Model) Groups() []Group {
	out := make([]Group, len(m.chrome.Groups))
	for i, g := range m.chrome.Groups {
		out[i] = g
		out[i].Options = make([]Option, len(g.Options))
		copy(out[i].Options, g.Options)
	}
	return out
}

// SelectedIn is the labels selected in one group, in the order they are drawn.
// Empty means the group constrains nothing.
func (m Model) SelectedIn(group int) []string {
	if group < 0 || group >= len(m.chrome.Groups) {
		return nil
	}
	var out []string
	for _, o := range m.chrome.Groups[group].Options {
		if o.Selected {
			out = append(out, o.Label)
		}
	}
	return out
}

type chipAt struct{ group, option int }

func (m Model) flatChips() []chipAt {
	var out []chipAt
	for g, group := range m.chrome.Groups {
		for o := range group.Options {
			out = append(out, chipAt{g, o})
		}
	}
	return out
}

// clamp keeps the cursor in range and the viewport around it.
func (m *Model) clamp() {
	if m.count == 0 {
		m.cursor, m.top = 0, 0
		return
	}
	m.cursor = clamp(m.cursor, 0, m.count-1)

	rows := m.Layout().List.Height
	if rows <= 0 {
		m.top = m.cursor
		return
	}
	// Scroll only as far as it takes to bring the cursor back on screen, so the
	// list moves under a held arrow key rather than jumping a page at a time.
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+rows {
		m.top = m.cursor - rows + 1
	}
	// Never leave blank rows below a list that could fill them.
	m.top = clamp(m.top, 0, max(0, m.count-rows))
}

// View draws the screen.
func (m Model) View() string {
	l := m.Layout()
	c := m.chrome

	// The chip the cursor is on is a property of the model, stamped onto a copy
	// of the groups at draw time. Keeping it on the options themselves would
	// mean two places that have to agree about where the cursor is.
	if c.Focus == FocusFilters {
		c.Groups = m.groupsWithCursor()
	}

	return c.Render(l, m.rows(l), m.previewLines(l))
}

func (m Model) groupsWithCursor() []Group {
	groups := make([]Group, len(m.chrome.Groups))
	for g, group := range m.chrome.Groups {
		groups[g] = group
		groups[g].Options = make([]Option, len(group.Options))
		for o, opt := range group.Options {
			opt.Focused = g == m.group && o == m.option
			groups[g].Options[o] = opt
		}
	}
	return groups
}

func (m Model) rows(l Layout) []string {
	if m.Row == nil || l.List.Height <= 0 {
		return nil
	}
	out := make([]string, 0, l.List.Height)
	for i := m.top; i < m.count && len(out) < l.List.Height; i++ {
		// Nothing is selected while the focus is elsewhere: a highlighted row
		// under a focused search field claims a cursor that is not there.
		selected := i == m.cursor && m.chrome.Focus == FocusList
		out = append(out, m.Row(i, l.Width, selected))
	}
	return out
}

func (m Model) previewLines(l Layout) []string {
	if m.Preview == nil || l.Preview.Empty() || m.count == 0 {
		return nil
	}
	return m.Preview(m.cursor, l.Width, l.Preview.Height)
}

// A line editor, expressed as functions over the text and where the cursor is in
// it, so the search field and the answer in a window are edited by the same code
// rather than by two copies that drift.
//
// Every edit is one of the two halves either side of the cursor changing, which
// is what keeps insert and delete from each needing their own idea of where the
// cursor is and what being at the end means.
func split(text string, caret int) (head, tail string) {
	r := []rune(text)
	at := clamp(caret, 0, len(r))
	return string(r[:at]), string(r[at:])
}

// insertAt puts s in at the cursor and leaves the cursor after it.
func insertAt(text string, caret int, s string) (string, int) {
	head, tail := split(text, caret)
	return head + s + tail, len([]rune(head)) + len([]rune(s))
}

// backspaceAt removes the character before the cursor, which is the one the
// cursor is not sitting on: it deletes what you have just typed, not what you
// have just walked onto.
func backspaceAt(text string, caret int) (string, int) {
	head, tail := split(text, caret)
	r := []rune(head)
	if len(r) == 0 {
		return text, caret
	}
	return string(r[:len(r)-1]) + tail, len(r) - 1
}

// dropWordAt removes the word before the cursor and keeps what follows it.
func dropWordAt(text string, caret int) (string, int) {
	head, tail := split(text, caret)
	head = dropWord(head)
	return head + tail, len([]rune(head))
}

// moveAt walks the cursor along the text, stopping at either end rather than
// wrapping: a cursor that reappears at the far end has lost the one thing it was
// telling you.
func moveAt(text string, caret, delta int) int {
	return clamp(caret+delta, 0, len([]rune(text)))
}

// dropWord removes the last word of a query, and the trailing space with it.
func dropWord(q string) string {
	i := len(q)
	for i > 0 && q[i-1] == ' ' {
		i--
	}
	for i > 0 && q[i-1] != ' ' {
		i--
	}
	return q[:i]
}
