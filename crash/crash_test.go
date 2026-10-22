package crash

import (
	"os"
	"strings"
	"testing"
	"time"
)

func at(dir string) Log {
	return Log{Dir: dir, Now: func() time.Time {
		return time.Date(2026, 9, 15, 10, 30, 0, 0, time.UTC)
	}}
}

// A terminal app that dies takes its own last words with it, so they go to a
// file — with enough in them to be worth having opened.
func TestACrashIsWrittenDownWithWhatItWas(t *testing.T) {
	l := at(t.TempDir())
	path := l.Write("board", "index out of range [3] with length 3",
		[]byte("goroutine 1 [running]:\nui.hints()\n"))
	if path == "" {
		t.Fatal("nothing was written")
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"2026-09-15T10:30:00Z", "board",
		"index out of range", "ui.hints()",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the log does not carry %q:\n%s", want, body)
		}
	}
}

// The second crash is usually the one that says what the first was about, so
// the file keeps both.
func TestCrashesAccumulateRatherThanReplacingEachOther(t *testing.T) {
	l := at(t.TempDir())
	l.Write("board", "first", nil)
	l.Write("board", "second", nil)

	body, _ := os.ReadFile(l.File())
	if !strings.Contains(string(body), "first") || !strings.Contains(string(body), "second") {
		t.Errorf("one crash replaced the other:\n%s", body)
	}
}

// Nowhere to write is not a reason to lose the message: the caller still gets a
// line it can put on the terminal.
func TestWithNowhereToWriteTheMessageSurvives(t *testing.T) {
	said := Log{}.Caught("board", "boom")
	if !strings.Contains(said, "boom") {
		t.Errorf("the message was lost: %q", said)
	}
	if strings.Contains(said, "written to") {
		t.Errorf("it claims to have written something: %q", said)
	}

	// And with somewhere, the line says where to look.
	l := at(t.TempDir())
	if got := l.Caught("board", "boom"); !strings.Contains(got, l.File()) {
		t.Errorf("the line does not say where to look: %q", got)
	}
}
