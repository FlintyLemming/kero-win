//go:build !windows

package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kero/internal/proto"
)

// Resolve maps a stored shell ID or executable path to the executable path.
func Resolve(id string) (exe, wslDistro string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Login(), ""
	}
	if filepath.IsAbs(id) {
		return id, ""
	}
	if p, err := exec.LookPath(id); err == nil {
		return p, ""
	}
	return id, ""
}

// QuickAvailable returns the available POSIX/modern shells on Unix.
func QuickAvailable() []proto.ShellOption {
	return Available()
}

// Available discovers valid executable shells on Unix from the user's login
// shell, /etc/shells, and common shells in PATH.
func Available() []proto.ShellOption {
	var candidates []string
	if sh := Login(); sh != "" {
		candidates = append(candidates, sh)
	}
	if b, err := os.ReadFile("/etc/shells"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			candidates = append(candidates, line)
		}
	}
	for _, name := range []string{"zsh", "bash", "fish", "sh", "nu", "pwsh"} {
		if p, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, p)
		}
	}

	var out []proto.ShellOption
	seen := make(map[string]struct{})
	for _, p := range candidates {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, proto.ShellOption{
			ID:   p,
			Name: filepath.Base(p),
			Path: p,
			Kind: "unix",
		})
	}
	return out
}
