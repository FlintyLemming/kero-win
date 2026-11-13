//go:build darwin || linux

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"kero/internal/proto"
)

// A restored pane starts its shell alone, not the command it ran, and the
// log exposes neither command arguments nor environment values.
func TestRestoreDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		browser bool
		plain   bool
	}{
		{name: "with a command"},
		{name: "ordinary shell", plain: true},
		{name: "browser", browser: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := testDaemon(t)
			var output bytes.Buffer
			d.log = log.New(&output, "", 0)
			argsPath := filepath.Join(d.dir, "args")
			t.Setenv("KERO_TEST_ARGS", argsPath)
			t.Setenv("KERO_TEST_PRIVATE", "private-environment-value")
			// Capture the actual PTY argv without loading the user's shell
			// profile. The name sh selects POSIX argv.
			shell := filepath.Join(d.dir, "sh")
			if err := os.WriteFile(shell, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$KERO_TEST_ARGS\"\nprintf 'END\\n' >> \"$KERO_TEST_ARGS\"\nexec /bin/cat\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			config := fmt.Sprintf("command = %s\n", shell)
			if err := os.WriteFile(os.Getenv("KERO_CONFIG"), []byte(config), 0o600); err != nil {
				t.Fatal(err)
			}
			sp := savedPane{ID: "pane", Dir: d.dir, Command: []string{"private-command-payload"}, Cols: 80, Rows: 24}
			if tc.plain {
				sp.Command = nil
			}
			if tc.browser {
				sp.URL = "https://example.test/private-page"
			}
			data, err := json.Marshal(savedState{Version: 2, Sessions: []savedSession{{ID: "session", Panes: []savedPane{sp}}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(d.statePath(), data, 0o600); err != nil {
				t.Fatal(err)
			}
			d.restore()
			p := d.panes["pane"]
			if p == nil {
				t.Fatalf("pane not restored: %s", output.String())
			}
			t.Cleanup(func() {
				p.close()
				if p.proc != nil {
					select {
					case <-p.proc.exited:
					case <-time.After(5 * time.Second):
						t.Error("fixture shell did not exit")
					}
				}
			})
			gotLog := output.String()
			for _, private := range []string{"private-command-payload", "private-environment-value", "private-page"} {
				if strings.Contains(gotLog, private) {
					t.Errorf("log exposed %q: %s", private, gotLog)
				}
			}
			if tc.browser {
				if p.proc != nil || p.info.URL != sp.URL {
					t.Fatalf("browser changed: %+v", p.info)
				}
				return
			}
			p.mu.Lock()
			unchanged := reflect.DeepEqual(p.info.Command, sp.Command)
			p.mu.Unlock()
			if !unchanged {
				t.Error("restoration changed the saved command")
			}
			var args string
			for deadline := time.Now().Add(5 * time.Second); ; {
				b, _ := os.ReadFile(argsPath)
				args = string(b)
				if strings.HasSuffix(args, "END\n") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("fixture shell did not record its arguments")
				}
				time.Sleep(time.Millisecond)
			}
			if args != "\nEND\n" {
				t.Errorf("the restored shell ran a command: %q", args)
			}
		})
	}
}

// Run owns signals and background goroutines, so exercise it in a subprocess.
func TestDaemonDiagnosticsProcess(t *testing.T) {
	if os.Getenv("KERO_TEST_DAEMON") != "1" {
		return
	}
	if err := Run(Options{Serve: true}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestDaemonRunDiagnostics(t *testing.T) {
	dir := t.TempDir()
	// A short directory keeps the Unix socket address within macOS's limit.
	socketDir, err := os.MkdirTemp("", "kd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(socketDir) })
	sock := filepath.Join(socketDir, "s")
	t.Setenv("KERO_HOME", dir)
	t.Setenv("KERO_CONFIG", filepath.Join(dir, "config"))
	t.Setenv("KERO_SOCKET", sock)
	t.Setenv("KERO_TEST_DAEMON", "1")
	t.Setenv("KERO_TAKEOVER", "")
	newCommand := func() *exec.Cmd {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		t.Cleanup(cancel)
		return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDaemonDiagnosticsProcess$")
	}
	start := func() (*exec.Cmd, <-chan error) {
		cmd := newCommand()
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		t.Cleanup(func() { cmd.Process.Kill() })
		for deadline := time.Now().Add(5 * time.Second); ; {
			conn, err := net.DialTimeout("unix", sock, 100*time.Millisecond)
			if err == nil {
				conn.SetDeadline(time.Now().Add(time.Second))
				json.NewEncoder(conn).Encode(proto.Hello{Version: proto.Version})
				var welcome proto.Welcome
				err = json.NewDecoder(conn).Decode(&welcome)
				conn.Close()
				if err == nil && welcome.PID == cmd.Process.Pid {
					return cmd, done
				}
			}
			select {
			case err := <-done:
				t.Fatalf("daemon exited before readiness: %v", err)
			default:
			}
			if time.Now().After(deadline) {
				t.Fatal("daemon did not become ready")
			}
			time.Sleep(time.Millisecond)
		}
	}
	stop := func(cmd *exec.Cmd, done <-chan error, sig syscall.Signal) {
		t.Helper()
		if err := cmd.Process.Signal(sig); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if sig == syscall.SIGTERM && err != nil {
				t.Fatalf("clean shutdown: %v", err)
			}
			if sig == syscall.SIGKILL && err == nil {
				t.Error("killed daemon exited successfully")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("daemon did not stop")
		}
	}
	readLog := func() string {
		b, err := os.ReadFile(filepath.Join(dir, "kerod.log"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	marker := filepath.Join(dir, "kerod.running")
	first, firstDone := start()
	if strings.Contains(readLog(), "previous daemon exit was unclean") {
		t.Error("first start reported an unclean exit")
	}
	before, err := os.Stat(marker)
	if err != nil {
		t.Errorf("running daemon has no run marker: %v", err)
	}
	out, err := newCommand().CombinedOutput()
	if err == nil || !strings.Contains(string(out), "kerod is already running") {
		t.Fatalf("duplicate daemon: %v, %s", err, out)
	}
	after, err := os.Stat(marker)
	if err != nil {
		t.Errorf("duplicate daemon removed run marker: %v", err)
	} else if before != nil && !before.ModTime().Equal(after.ModTime()) {
		t.Error("duplicate daemon changed the owner's run marker")
	}
	if got := strings.Count(readLog(), "--- daemon run ---"); got != 1 {
		t.Errorf("run separators after duplicate: got %d, want 1", got)
	}
	stop(first, firstDone, syscall.SIGTERM)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("clean shutdown left run marker: %v", err)
	}
	second, secondDone := start()
	if strings.Contains(readLog(), "previous daemon exit was unclean") {
		t.Error("clean predecessor reported as unclean")
	}
	stop(second, secondDone, syscall.SIGKILL)
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("killed daemon lost run marker: %v", err)
	}
	third, thirdDone := start()
	got := readLog()
	if strings.Count(got, "previous daemon exit was unclean") != 1 {
		t.Errorf("unclean predecessor not identified once: %s", got)
	}
	if strings.Contains(got, "crash") {
		t.Errorf("claimed a crash cause: %s", got)
	}
	if strings.Count(got, "--- daemon run ---") != 3 {
		t.Errorf("missing daemon run boundaries: %s", got)
	}
	stop(third, thirdDone, syscall.SIGTERM)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("final clean shutdown left run marker: %v", err)
	}
}
