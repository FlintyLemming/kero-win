package shell

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"kero/internal/proto"
	"golang.org/x/sys/windows"
)

// Resolve maps a stored shell ID or executable path to the executable path
// and, for WSL distributions, the distribution name.
func Resolve(id string) (exe, wslDistro string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Login(), ""
	}
	if distro, ok := strings.CutPrefix(id, WSLPrefix); ok && distro != "" {
		if w := FindWSL(); w != "" {
			return w, distro
		}
		return "wsl.exe", distro
	}
	switch strings.ToLower(id) {
	case "pwsh", "pwsh.exe":
		if p := FindPwsh(); p != "" {
			return p, ""
		}
		return "pwsh.exe", ""
	case "powershell", "powershell.exe":
		if p := FindWindowsPowerShell(); p != "" {
			return p, ""
		}
		return "powershell.exe", ""
	case "cmd", "cmd.exe":
		return FindCmd(), ""
	case "bash", "git-bash":
		if p := FindGitBash(); p != "" {
			return p, ""
		}
	}
	if exists(id) {
		return id, ""
	}
	if p, err := exec.LookPath(id); err == nil {
		return p, ""
	}
	return id, ""
}

const wslProbeTimeout = 2 * time.Second

// QuickAvailable returns the local Windows shells without invoking wsl.exe,
// suitable for immediate fallback before async shell probing finishes.
func QuickAvailable() []proto.ShellOption {
	var out []proto.ShellOption
	if p := FindPwsh(); p != "" {
		out = append(out, proto.ShellOption{
			ID:   "pwsh",
			Name: "PowerShell",
			Path: p,
			Kind: "pwsh",
		})
	}
	if p := FindPwshPreview(); p != "" {
		out = append(out, proto.ShellOption{
			ID:   p,
			Name: "PowerShell Preview",
			Path: p,
			Kind: "pwsh",
		})
	}
	if p := FindWindowsPowerShell(); p != "" {
		out = append(out, proto.ShellOption{
			ID:   "powershell",
			Name: "Windows PowerShell",
			Path: p,
			Kind: "powershell",
		})
	}
	if p := FindCmd(); p != "" {
		out = append(out, proto.ShellOption{
			ID:   "cmd",
			Name: "Command Prompt",
			Path: p,
			Kind: "cmd",
		})
	}
	if p := FindGitBash(); p != "" {
		out = append(out, proto.ShellOption{
			ID:   p,
			Name: "Git Bash",
			Path: p,
			Kind: "bash",
		})
	}
	return out
}

// Available returns all detected shells on Windows, matching Windows Terminal's
// built-in profiles: PowerShell 7, Windows PowerShell, Command Prompt,
// installed WSL distributions (such as Ubuntu), and Git Bash.
func Available() []proto.ShellOption {
	quick := QuickAvailable()
	wslDistros := availableWSLDistros()
	if len(wslDistros) == 0 {
		return quick
	}
	// Place WSL distros right after Command Prompt (before Git Bash), matching
	// Windows Terminal's default profile ordering.
	var out []proto.ShellOption
	inserted := false
	for _, opt := range quick {
		if opt.Kind == "bash" && !inserted {
			out = append(out, wslDistros...)
			inserted = true
		}
		out = append(out, opt)
	}
	if !inserted {
		out = append(out, wslDistros...)
	}
	return out
}

func availableWSLDistros() []proto.ShellOption {
	wslExe := FindWSL()
	if wslExe == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), wslProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, wslExe, "-l", "-q")
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	// Never flash a console window when the GUI probes WSL in the background.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil
	}
	var distros []proto.ShellOption
	for _, name := range ParseWSLDistros(out) {
		distros = append(distros, proto.ShellOption{
			ID:   WSLPrefix + name,
			Name: name,
			Path: wslExe,
			Kind: "wsl",
		})
	}
	return distros
}
