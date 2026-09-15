package journal

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// What an app did has to end up somewhere a person can read afterwards, because
// the screen it was drawn on is gone.
func TestWhatIsLoggedEndsUpInTheFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	t.Setenv("BELP_LOG", "")

	closer := Open(dir, "belp")
	slog.Info("resumed", "service", "web", "containers", 9)
	slog.Debug("this one is below the level")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(File(dir, "belp"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"level=INFO", "msg=resumed", "service=web", "containers=9"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the log does not carry %q:\n%s", want, body)
		}
	}
	if strings.Contains(string(body), "below the level") {
		t.Error("a debug line was written at the default level")
	}
}

// The level is asked for rather than assumed: a log nobody set up still answers
// "what did it do", and the debug line that would drown it is one you request.
func TestTheLevelComesFromTheEnvironment(t *testing.T) {
	for _, c := range []struct {
		set  string
		want slog.Level
	}{
		{"", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	} {
		t.Setenv("BELP_LOG", c.set)
		if got := Level(); got != c.want {
			t.Errorf("BELP_LOG=%q gives %v, want %v", c.set, got, c.want)
		}
	}
	// And a run that must not touch the disk can say so.
	t.Setenv("BELP_LOG", "off")
	if Level() <= slog.LevelError {
		t.Error("off still writes errors")
	}
}

// Nothing on a Mac rotates a user's own logs — newsyslog covers /Library/Logs
// and not ~/Library/Logs — so a file left alone grows until somebody notices.
func TestAFullFileIsRolledAside(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	path := File(dir, "belp")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, Cap+1), 0o644); err != nil {
		t.Fatal(err)
	}

	closer := Open(dir, "belp")
	slog.Info("after the roll")
	_ = closer.Close()

	fresh, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh) > 4096 {
		t.Errorf("the new file is %d bytes, so nothing was rolled", len(fresh))
	}
	if !strings.Contains(string(fresh), "after the roll") {
		t.Error("the new file did not get the new line")
	}
	// The old one is kept, because the thing before a failure is usually what
	// explains it.
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("the full file was not kept: %v", err)
	}
}

// A journal that cannot open must not stop the app: a log is a convenience and
// running is the job.
func TestAnAppRunsWithNowhereToWrite(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	closer := Open("/proc/nonexistent-and-unwritable", "belp")
	if err := closer.Close(); err != nil {
		t.Errorf("closing a journal that never opened failed: %v", err)
	}
	slog.Info("this must not panic")
}

// The place is the platform's own, so the logs turn up where a person looks.
func TestTheDirectoryIsWhereThePlatformKeepsLogs(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	got := Dir("/Users/me")
	if runtime.GOOS == "darwin" {
		if got != "/Users/me/Library/Logs/belp" {
			t.Errorf("on a Mac logs go to %q", got)
		}
		return
	}
	if !strings.HasPrefix(got, "/Users/me/.local/state") {
		t.Errorf("logs go to %q", got)
	}
}
