package daemon

import (
	"path/filepath"
	"testing"
	"time"

	"kero/internal/proto"
)

// testPane is a pane with no program, whose output the test writes.
func testPane(t *testing.T) *pane {
	t.Helper()
	t.Setenv("KERO_CONFIG", filepath.Join(t.TempDir(), "config"))
	d := &Daemon{changeCh: make(chan struct{}, 1), saveCh: make(chan struct{}, 1)}
	d.viewed.Store(new(string))
	s, err := newPane(d, "s", proto.Pane{ID: "p", Status: proto.Idle, Cols: 96, Rows: 28})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.term.Close() })
	return s
}

func (s *pane) status(now time.Time) proto.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.look()
	return s.info.Status
}

// A program's bell and notification tell nothing of its status: vim
// beeps at a wrong key, a build notifies it is done.
func TestProgramNotifyIsNoWait(t *testing.T) {
	s := testPane(t)
	s.fgPGID, s.shellPID = 2, 1
	now := time.Now()
	s.output([]byte("\x07\x1b]9;Build finished\x07\x1b]777;notify;make;Build finished\x07"))
	if st := s.status(now); st != proto.Idle {
		t.Fatalf("after a bell and notifications: %s", st)
	}
}

// A program works while its title turns as a spinner: a clock it draws,
// however long it ticks, is not work. Work that ends unseen is unread.
func TestProgramWorksByTitle(t *testing.T) {
	s := testPane(t)
	now := time.Now()
	for i := range 30 {
		s.output([]byte("\x1b[32;1H" + now.Format("05")))
		now = now.Add(time.Second)
		if st := s.status(now); st != proto.Idle {
			t.Fatalf("clock tick %d: %s", i, st)
		}
	}
	s.output([]byte("\x1b]0;⠋ building\x07"))
	if st := s.status(now); st != proto.Working {
		t.Fatalf("title turning: %s", st)
	}
	// Redraws of the work, as a resize makes, are the same work.
	s.resize(80, 28, 0, 0)
	s.output([]byte("\x1b]0;⠙ building\x07"))
	if st := s.status(now); st != proto.Working {
		t.Fatalf("title turning, resized: %s", st)
	}
	s.output([]byte("\x1b]0;done\x07"))
	if st := s.status(now); st != proto.Idle {
		t.Fatalf("title still: %s", st)
	}
	if !s.info.Unread {
		t.Error("work that ended unseen is not unread")
	}
}

// "kero notify" needs the user until they type in the pane.
func TestNotifyNeedsUser(t *testing.T) {
	s := testPane(t)
	now := time.Now()
	if st := s.status(now); st != proto.Idle {
		t.Fatalf("at the prompt: %s", st)
	}
	s.mu.Lock()
	s.flag("", "Need your API key to continue")
	s.mu.Unlock()
	if st := s.status(now); st != proto.Waiting || s.info.Message != "Need your API key to continue" {
		t.Fatalf("kero notify: %s %q", st, s.info.Message)
	}
	s.input(nil)
	if st := s.status(now); st != proto.Idle || s.info.Message != "" {
		t.Fatalf("after a key: %s %q", st, s.info.Message)
	}
}
