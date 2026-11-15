package shell

import (
	"bytes"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"kero/internal/proto"
)

// WSLPrefix identifies a WSL distribution shell ID (e.g. "wsl:Ubuntu").
const WSLPrefix = "wsl:"

// DefaultOptions returns a conservative OS shell list when live discovery
// has not completed yet (such as before a remote host responds).
func DefaultOptions(goos string) []proto.ShellOption {
	if goos == "windows" {
		return []proto.ShellOption{
			{ID: "pwsh", Name: "PowerShell", Kind: "pwsh"},
			{ID: "powershell", Name: "Windows PowerShell", Kind: "powershell"},
			{ID: "cmd", Name: "Command Prompt", Kind: "cmd"},
		}
	}
	return []proto.ShellOption{
		{ID: "zsh", Name: "zsh", Kind: "unix"},
		{ID: "bash", Name: "bash", Kind: "unix"},
		{ID: "sh", Name: "sh", Kind: "unix"},
	}
}

// DefaultDisplayName returns a human-readable name for a shell path or ID.
func DefaultDisplayName(sh string) string {
	sh = strings.TrimSpace(sh)
	if sh == "" {
		return ""
	}
	if distro, ok := strings.CutPrefix(sh, WSLPrefix); ok {
		return distro
	}
	base := filepath.Base(strings.ReplaceAll(sh, `\`, "/"))
	switch strings.ToLower(base) {
	case "pwsh", "pwsh.exe":
		return "PowerShell"
	case "powershell", "powershell.exe":
		return "Windows PowerShell"
	case "cmd", "cmd.exe":
		return "cmd.exe"
	default:
		return strings.TrimSuffix(base, ".exe")
	}
}

// ParseWSLDistros extracts installed distribution names from `wsl.exe -l -q`
// output, handling both UTF-8 and Windows' default UTF-16LE output and
// filtering internal Docker Desktop helper distributions.
func ParseWSLDistros(out []byte) []string {
	text := decodeWSLOutput(out)
	var distros []string
	seen := make(map[string]struct{})
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "docker-desktop" || lower == "docker-desktop-data" {
			continue
		}
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		distros = append(distros, line)
	}
	return distros
}

func decodeWSLOutput(out []byte) string {
	// wsl.exe prints UTF-16LE when WSL_UTF8=1 is ignored or unsupported.
	if len(out) >= 2 && (bytes.IndexByte(out, 0) >= 0 || (out[0] == 0xff && out[1] == 0xfe)) {
		if out[0] == 0xff && out[1] == 0xfe {
			out = out[2:]
		}
		u16 := make([]uint16, len(out)/2)
		for i := range u16 {
			u16[i] = uint16(out[2*i]) | uint16(out[2*i+1])<<8
		}
		return string(utf16.Decode(u16))
	}
	return string(out)
}
