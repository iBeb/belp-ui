// Package journal is where an app writes down what it did.
//
// A terminal app has nowhere to say anything: stdout is the screen it is
// drawing, and stderr lands on the same screen a moment before it is torn down.
// So the running account goes to a file, at the place the platform keeps them,
// and the screen stays for the person.
//
// Levels, because "what happened" and "what went wrong" are different questions
// asked at different times: a quiet log is one somebody will actually read when
// something breaks, and a noisy one is grep practice.
package journal

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Cap is how large a file grows before it is rolled over, and Keep how many
// older ones are kept.
//
// Rolled here rather than left to the system: newsyslog covers /Library/Logs
// and not ~/Library/Logs, so nothing on a Mac rotates a user's own logs and a
// file left alone grows until somebody notices.
const (
	Cap  = 4 << 20 // 4 MiB
	Keep = 1
)

// Dir is where this platform keeps a person's application logs.
//
// ~/Library/Logs on a Mac, which is the convention and is what Console.app
// reads. XDG's state directory elsewhere, and whatever XDG_STATE_HOME says when
// it says anything, so a test can put them somewhere else.
func Dir(home string) string {
	if s := os.Getenv("XDG_STATE_HOME"); s != "" {
		return filepath.Join(s, "logs")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Logs", "belp")
	}
	return filepath.Join(home, ".local", "state", "belp", "logs")
}

// File is where one app writes.
func File(home, app string) string {
	return filepath.Join(Dir(home), app+".log")
}

// Level is what gets written, read from BELP_LOG.
//
// Info by default: a log nobody set up should still answer "what did it do", and
// the debug line that would drown it is one you ask for.
func Level() slog.Level {
	switch strings.ToLower(os.Getenv("BELP_LOG")) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "off", "none":
		// Nothing is written at all, for a run that must not touch the disk.
		return slog.LevelError + 4
	default:
		return slog.LevelInfo
	}
}

// Open starts an app's journal and makes it the default for log/slog, so that
// everything an app calls can write to it without being handed anything.
//
// The closer is safe to call on a journal that never opened: an app that cannot
// write its log still has to run.
func Open(home, app string) io.Closer {
	path := File(home, app)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return shut{}
	}
	roll(path)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return shut{}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: Level()})))
	return f
}

// roll moves a full file aside so the new one starts empty.
func roll(path string) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < Cap {
		return
	}
	// Oldest first, so nothing is overwritten before it has been moved on.
	for i := Keep; i >= 1; i-- {
		older := path + "." + itoa(i+1)
		if i == Keep {
			_ = os.Remove(path + "." + itoa(i))
			continue
		}
		_ = os.Rename(path+"."+itoa(i), older)
	}
	_ = os.Rename(path, path+".1")
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// shut is a closer for a journal that never opened.
type shut struct{}

func (shut) Close() error { return nil }
