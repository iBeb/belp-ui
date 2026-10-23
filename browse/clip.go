package browse

import (
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CopiedMsg says text reached the clipboard, or did not.
type CopiedMsg struct {
	Text string
	Err  error
}

// copyText puts text on the system clipboard.
//
// A subprocess rather than the OSC 52 escape, which would be the portable
// answer and the one that survives ssh: the escape has to be written to the
// same stream bubbletea is painting frames on, and a write that lands between
// two frames corrupts the one being drawn. OSC 52 belongs behind this function
// the day an app of ours runs anywhere but a local terminal.
func copyText(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	return func() tea.Msg {
		name, args := clipCommand()
		if name == "" {
			return CopiedMsg{Text: text, Err: errNoClipboard}
		}
		cmd := exec.Command(name, args...)
		cmd.Stdin = stringReader(text)
		return CopiedMsg{Text: text, Err: cmd.Run()}
	}
}

func clipCommand() (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil
	case "linux":
		if p, err := exec.LookPath("wl-copy"); err == nil {
			return p, nil
		}
		if p, err := exec.LookPath("xclip"); err == nil {
			return p, []string{"-selection", "clipboard"}
		}
	}
	return "", nil
}

type clipErr string

func (e clipErr) Error() string { return string(e) }

const errNoClipboard = clipErr("no clipboard program on this system")

func stringReader(s string) *strings.Reader { return strings.NewReader(s) }
