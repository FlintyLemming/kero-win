package main

import (
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"kero/internal/proto"
	"kero/internal/shell"
	"kero/locales"
)

// fetchShells queries h's kerod for its default shell and installed shell
// profiles asynchronously, publishing the result back to the UI thread.
func (a *App) fetchShells(h *host) {
	if h == nil || h.cl == nil {
		return
	}
	cl := h.cl
	h.shellsLoading = true
	go func() {
		var res proto.HostShells
		err := cl.Call("host.shells", nil, &res)
		a.update(func() {
			h.shellsLoading = false
			if h.cl != cl || err != nil {
				return
			}
			h.shells = res.Shells
			h.shellDefault = res.Default
			h.shellsLoaded = true
			h.shellsAt = time.Now()
		})
	}()
}

// ensureLocalShells probes local shells (including WSL distributions) in a
// background goroutine so opening the context menu never blocks the UI thread.
func (a *App) ensureLocalShells() {
	h := a.local()
	if h == nil || h.shellsLoaded || h.shellsLoading {
		return
	}
	h.shellsLoading = true
	go func() {
		opts := shell.Available()
		def := a.set.Shell
		if def == "" {
			def = shell.Login()
		}
		a.update(func() {
			h.shellsLoading = false
			if !h.shellsLoaded {
				h.shells = opts
				h.shellDefault = def
				h.shellsLoaded = true
				h.shellsAt = time.Now()
			}
		})
	}()
}

func (a *App) hostShellOptions(h *host) (opts []proto.ShellOption, defaultName string) {
	if h != nil && h.shellsLoaded {
		def := h.shellDefault
		if !h.remote() && a.set.Shell != "" {
			def = a.set.Shell
		}
		return h.shells, a.hostShellLabel(h, def)
	}
	if h != nil && h.remote() {
		osName := h.info.OS
		if osName == "" {
			osName = runtime.GOOS
		}
		return shell.DefaultOptions(osName), ""
	}
	a.ensureLocalShells()
	def := a.set.Shell
	if def == "" {
		def = shell.Login()
	}
	return shell.QuickAvailable(), shell.DefaultDisplayName(def)
}

func shellOptionLabel(opt proto.ShellOption) string {
	if opt.Kind == "cmd" {
		return locales.ShellCommandPrompt()
	}
	return opt.Name
}

func (a *App) hostShellLabel(h *host, sh string) string {
	sh = strings.TrimSpace(sh)
	if sh == "" {
		return ""
	}
	if h != nil {
		for _, opt := range h.shells {
			if opt.ID == sh || (opt.Path != "" && strings.EqualFold(opt.Path, sh)) {
				return shellOptionLabel(opt)
			}
		}
	}
	if strings.EqualFold(sh, "cmd") || strings.EqualFold(sh, "cmd.exe") || strings.EqualFold(shell.DefaultDisplayName(sh), "cmd.exe") {
		return locales.ShellCommandPrompt()
	}
	return shell.DefaultDisplayName(sh)
}

func (a *App) projectShellMenu(m *ui.Menu, g group) {
	id := g.project.ID
	if id == "" {
		// ui.Menu.Submenu builds immediately even when disabled, so use a plain disabled item.
		m.Item(locales.SidebarDefaultShellMenu()).Disabled(true)
		return
	}
	m.Submenu(locales.SidebarDefaultShellMenu(), func(m *ui.Menu) {
		opts, defName := a.hostShellOptions(g.host)
		autoLabel := locales.SidebarDefaultShellAuto()
		if defName != "" {
			autoLabel = locales.SidebarDefaultShellAutoNamed(defName)
		}
		cur := strings.TrimSpace(g.project.Shell)
		if m.Item(autoLabel).Checked(cur == "").Chosen() {
			a.setProjectShell(id, "")
		}
		if len(opts) > 0 || cur != "" {
			m.Separator()
		}
		matched := false
		for _, opt := range opts {
			checked := cur != "" && (cur == opt.ID || (opt.Path != "" && strings.EqualFold(cur, opt.Path)))
			if checked {
				matched = true
			}
			if m.Item(shellOptionLabel(opt)).Checked(checked).Chosen() {
				a.setProjectShell(id, opt.ID)
			}
		}
		if cur != "" && !matched {
			if m.Item(shell.DefaultDisplayName(cur)).Checked(true).Chosen() {
				a.setProjectShell(id, cur)
			}
		}
	})
}

func (a *App) setProjectShell(id, sh string) {
	sh = strings.TrimSpace(sh)
	for i := range a.st.Projects {
		if p := &a.st.Projects[i]; p.ID == id {
			if sh == p.Shell {
				return
			}
			p.Shell = sh // at once; the daemon's state follows
			a.call(id, "project.shell", proto.ProjectParams{ID: id, Shell: sh}, nil)
			return
		}
	}
}
