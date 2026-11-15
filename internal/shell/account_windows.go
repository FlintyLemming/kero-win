package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// account returns the shell Windows Terminal opens: PowerShell 7 when it
// is installed, else Windows PowerShell, else cmd.exe.
func account() string {
	if p := FindPwsh(); p != "" {
		return p
	}
	if p := FindWindowsPowerShell(); p != "" {
		return p
	}
	return FindCmd()
}

// FindPwsh returns the installed PowerShell 7+ executable, if any.
func FindPwsh() string {
	candidates := []string{}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "PowerShell", "7", "pwsh.exe"))
	}
	return findCommand("pwsh.exe", candidates...)
}

// FindPwshPreview returns the installed PowerShell Preview executable, if any.
func FindPwshPreview() string {
	var candidates []string
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "PowerShell", "7-preview", "pwsh.exe"))
	}
	return firstExisting(candidates...)
}

// FindWindowsPowerShell returns Windows PowerShell 5.1, if present.
func FindWindowsPowerShell() string {
	return findCommand(
		"powershell.exe",
		filepath.Join(systemRoot(), "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
	)
}

// FindCmd returns cmd.exe.
func FindCmd() string {
	if p := os.Getenv("ComSpec"); exists(p) {
		return p
	}
	return filepath.Join(systemRoot(), "System32", "cmd.exe")
}

// FindWSL returns wsl.exe if installed.
func FindWSL() string {
	return findCommand("wsl.exe", filepath.Join(systemRoot(), "System32", "wsl.exe"))
}

// FindGitBash returns Git for Windows' bash.exe if installed, avoiding
// System32's legacy WSL bash.exe stub.
func FindGitBash() string {
	var candidates []string
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(env); root != "" {
			candidates = append(candidates, filepath.Join(root, "Git", "bin", "bash.exe"))
		}
	}
	if local := os.Getenv("LocalAppData"); local != "" {
		candidates = append(candidates, filepath.Join(local, "Programs", "Git", "bin", "bash.exe"))
	}
	if p := firstExisting(candidates...); p != "" {
		return p
	}
	if gitExe, err := exec.LookPath("git.exe"); err == nil {
		// git.exe usually lives in <Git>/cmd/git.exe or <Git>/bin/git.exe.
		dir := filepath.Dir(filepath.Dir(gitExe))
		if p := filepath.Join(dir, "bin", "bash.exe"); exists(p) && !strings.EqualFold(filepath.Base(dir), "System32") {
			return p
		}
	}
	return ""
}

func findCommand(exe string, candidates ...string) string {
	if p, err := exec.LookPath(exe); err == nil && exists(p) {
		return p
	}
	return firstExisting(candidates...)
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if exists(p) {
			return p
		}
	}
	return ""
}

func systemRoot() string {
	if root := os.Getenv("SystemRoot"); root != "" {
		return root
	}
	return `C:\Windows`
}

func exists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
