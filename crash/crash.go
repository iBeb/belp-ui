// Package crash writes down what went wrong.
//
// A full-screen terminal app that dies takes its own last words with it: the
// alternate screen is torn down, whatever was printed goes with it, and inside a
// grid the pane is redrawn over the top. So the words go to a file, and the one
// line left on the terminal says where.
package crash

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

// Log is where crashes are written.
type Log struct {
	// Dir is the directory holding the file. Empty writes nothing, which is what
	// a test gets unless it says otherwise.
	Dir string
	// Now is the clock, so a test can pin what a line says.
	Now func() time.Time
}

// Default is where a person's crashes are kept.
func Default(home string) Log {
	dir := filepath.Join(home, ".local", "state", "belp")
	if s := os.Getenv("XDG_STATE_HOME"); s != "" {
		dir = filepath.Join(s, "belp")
	}
	return Log{Dir: dir, Now: time.Now}
}

// File is the path written to.
func (l Log) File() string { return filepath.Join(l.Dir, "crash.log") }

// Write records one failure and returns the path, or empty when there is
// nowhere to write.
//
// Appended rather than replaced: the second crash is usually the one that tells
// you what the first was about, and a file that keeps only the latest throws
// that away.
func (l Log) Write(what string, why any, stack []byte) string {
	if l.Dir == "" {
		return ""
	}
	if err := os.MkdirAll(l.Dir, 0o755); err != nil {
		return ""
	}
	f, err := os.OpenFile(l.File(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return ""
	}
	defer f.Close()

	var b strings.Builder
	fmt.Fprintf(&b, "\n%s  %s\n", l.now().Format(time.RFC3339), what)
	fmt.Fprintf(&b, "%v\n", why)
	if len(stack) > 0 {
		b.Write(stack)
	}
	if _, err := f.WriteString(b.String()); err != nil {
		return ""
	}
	return l.File()
}

func (l Log) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

// Caught turns a recovered panic into a line for the terminal and a file with
// the rest of it.
//
// The stack is taken here rather than passed in so that it is the stack of the
// panic and not of whatever tidied up after it.
func (l Log) Caught(what string, why any) string {
	path := l.Write(what, why, debug.Stack())
	if path == "" {
		return fmt.Sprintf("%s: %v", what, why)
	}
	return fmt.Sprintf("%s: %v — written to %s", what, why, path)
}
