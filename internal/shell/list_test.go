package shell

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestParseWSLDistros(t *testing.T) {
	t.Run("utf8 with docker-desktop and duplicates", func(t *testing.T) {
		raw := []byte("\xef\xbb\xbfUbuntu\r\ndocker-desktop\r\ndocker-desktop-data\r\nDebian\r\nubuntu\r\n\r\n")
		got := ParseWSLDistros(raw)
		want := []string{"Ubuntu", "Debian"}
		if !slices.Equal(got, want) {
			t.Fatalf("ParseWSLDistros = %v, want %v", got, want)
		}
	})

	t.Run("utf16le with BOM", func(t *testing.T) {
		text := "Ubuntu\r\ndocker-desktop\r\nUbuntu-24.04\r\n"
		u16 := utf16.Encode([]rune(text))
		raw := []byte{0xff, 0xfe}
		for _, v := range u16 {
			raw = append(raw, byte(v), byte(v>>8))
		}
		got := ParseWSLDistros(raw)
		want := []string{"Ubuntu", "Ubuntu-24.04"}
		if !slices.Equal(got, want) {
			t.Fatalf("ParseWSLDistros = %v, want %v", got, want)
		}
	})
}

func TestDefaultDisplayName(t *testing.T) {
	cases := map[string]string{
		`C:\Program Files\PowerShell\7\pwsh.exe`:                        "PowerShell",
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`:     "Windows PowerShell",
		`C:\Windows\System32\cmd.exe`:                                   "cmd.exe",
		"wsl:Ubuntu":                                                    "Ubuntu",
		"/bin/zsh":                                                      "zsh",
		"":                                                              "",
	}
	for in, want := range cases {
		if got := DefaultDisplayName(in); got != want {
			t.Fatalf("DefaultDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAvailableReturnsAtLeastOneShell(t *testing.T) {
	got := Available()
	if len(got) == 0 {
		t.Fatal("Available() returned no shells")
	}
	for _, opt := range got {
		if opt.ID == "" || opt.Name == "" {
			t.Fatalf("invalid option %+v", opt)
		}
	}
}
