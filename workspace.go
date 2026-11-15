package main

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"kero/internal/client"
	"kero/internal/proto"
	"kero/locales"
)

// workspaceAsk holds the modal form state for creating a new workspace,
// optionally backed by a folder, with default shell and command settings.
type workspaceAsk struct {
	open    bool
	host    *host
	name    string
	dir     string
	command string
	shell   string
	err     string
}

// startNewWorkspace opens the New Workspace dialog on h (or the selected host).
func (a *App) startNewWorkspace(h *host) {
	if h == nil {
		h = a.selectedHost()
	}
	if h == nil {
		h = a.local()
	}
	if h != nil {
		if h.remote() && h.cl == nil {
			a.toast(locales.HostNotConnected(h.name))
			return
		}
		if !h.remote() {
			a.ensureLocalShells()
		}
		if h.cl != nil {
			a.fetchShells(h)
		}
	}
	a.wsAsk = &workspaceAsk{open: true, host: h}
}

func (a *App) workspaceView(c *ui.Context, pal *palette) {
	d := a.wsAsk
	if d == nil {
		return
	}
	// When the remote folder picker is open for this dialog, let the picker
	// own keyboard focus and Escape while preserving the form state.
	if a.picker.open {
		return
	}
	if d.host == nil {
		d.host = a.selectedHost()
	}
	if d.host == nil {
		d.host = a.local()
	}

	const fieldWidth = 396
	var nameIn, dirIn, cmdIn ui.Element
	submit := false

	ui.DialogBase(c.Key("workspace"), &d.open, func(backdrop, panel ui.Element) {
		backdrop.Background(pal.overlay)
		floating(panel.Width(440).Radius(12).Background(pal.popover), pal).
			Padding(20, 20).Gap(14).Label(locales.WorkspaceNewTitle())

		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, locales.WorkspaceNewTitle()).FontSize(14).FontWeight(600).TextColor(pal.text)
			ui.Text(c, locales.WorkspaceNewDetail()).FontSize(12.5).TextColor(pal.textMuted).Wrap()
		})

		if len(a.hosts) > 1 {
			prevHost := d.host
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, locales.WorkspaceHostLabel()).FontSize(12).FontWeight(500).TextColor(pal.textMuted)
				popup(c, "host", &d.host, a.hosts, func(h *host) string {
					if h == nil {
						return ""
					}
					return h.label()
				}, fieldWidth, pal)
			})
			if d.host != prevHost {
				d.shell = ""
				d.err = ""
				if d.host != nil {
					if !d.host.remote() {
						a.ensureLocalShells()
					}
					if d.host.cl != nil {
						a.fetchShells(d.host)
					}
				}
			}
		}

		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, locales.WorkspaceNameLabel()).FontSize(12).FontWeight(500).TextColor(pal.textMuted)
			nameIn = textField(c, "name", &d.name, locales.WorkspaceNamePlaceholder(), fieldWidth, pal)
			nameIn.AutoFocus()
			if nameIn.Changed() {
				d.err = ""
			}
		})

		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, locales.WorkspaceFolderLabel()).FontSize(12).FontWeight(500).TextColor(pal.textMuted)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				dirIn = textField(c, "dir", &d.dir, locales.WorkspaceFolderPlaceholder(), 0, pal).Grow(1).MinWidth(0)
				if dirIn.Changed() {
					d.err = ""
				}
				if fieldButton(c, "choose", locales.SetChooseFolder(), false, pal) {
					a.chooseWorkspaceFolder(d)
				}
			})
		})

		opts, defName := a.hostShellOptions(d.host)
		shellIDs := make([]string, 0, len(opts)+1)
		shellIDs = append(shellIDs, "")
		autoLabel := locales.SidebarDefaultShellAuto()
		if defName != "" {
			autoLabel = locales.SidebarDefaultShellAutoNamed(defName)
		}
		shellLabels := map[string]string{"": autoLabel}
		for _, opt := range opts {
			shellIDs = append(shellIDs, opt.ID)
			shellLabels[opt.ID] = shellOptionLabel(opt)
		}
		if d.shell != "" && !slices.Contains(shellIDs, d.shell) {
			shellIDs = append(shellIDs, d.shell)
		}
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, locales.SidebarDefaultShellMenu()).FontSize(12).FontWeight(500).TextColor(pal.textMuted)
			popup(c, "shell", &d.shell, shellIDs, func(id string) string {
				if lbl, ok := shellLabels[id]; ok {
					return lbl
				}
				return a.hostShellLabel(d.host, id)
			}, fieldWidth, pal)
		})

		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, locales.WorkspaceCommandLabel()).FontSize(12).FontWeight(500).TextColor(pal.textMuted)
			cmdIn = textField(c, "command", &d.command, locales.ProjectCommandPlaceholder(), fieldWidth, pal)
		})

		if d.err != "" {
			ui.Text(c, d.err).FontSize(12).TextColor(pal.danger).Wrap()
		}

		ui.Row(c).Gap(8).Justify(ui.End).Margin(4, 0, 0, 0).Children(func() {
			if button(c, "cancel", locales.CommonCancel(), "esc", false, pal) {
				d.open = false
			}
			if button(c, "create", locales.WorkspaceCreate(), "↩", true, pal) {
				submit = true
			}
		})
	})

	if submit || nameIn.Submitted() || dirIn.Submitted() || cmdIn.Submitted() {
		a.submitNewWorkspace(d)
	}

	if !d.open && !a.picker.open {
		a.wsAsk = nil
		a.focusTerminal()
	}
}

func (a *App) submitNewWorkspace(d *workspaceAsk) {
	name := strings.TrimSpace(d.name)
	dir := strings.TrimSpace(d.dir)
	if name == "" && dir == "" {
		d.err = locales.WorkspaceNeedNameOrFolder()
		return
	}
	h := d.host
	if h == nil {
		h = a.local()
	}
	if h == nil || h.cl == nil {
		if h != nil {
			d.err = locales.HostNotConnected(h.name)
		}
		return
	}
	if dir != "" && !h.remote() {
		expanded := expandHome(dir)
		if abs, err := filepath.Abs(expanded); err == nil {
			expanded = abs
		}
		if !isDir(expanded) {
			d.err = locales.WorkspaceInvalidFolder()
			return
		}
		dir = expanded
	}
	if name == "" && dir != "" {
		name = proto.DirName(dir)
	}
	cmd := strings.TrimSpace(d.command)
	sh := strings.TrimSpace(d.shell)
	d.open = false
	go a.createWorkspace(h, h.cl, name, dir, cmd, sh)
}

func (a *App) chooseWorkspaceFolder(d *workspaceAsk) {
	h := d.host
	if h == nil {
		h = a.local()
	}
	if h != nil && h.remote() {
		a.pickFolderOn(h, func(pickedHost *host, dir string) {
			a.applyWorkspaceFolderPick(d, pickedHost, dir)
		})
		return
	}
	win := a.win
	if win == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: win, Title: locales.OpenProjectTitle(), Directory: true, ButtonLabel: locales.CommonOpen(),
			Message: locales.OpenProjectMessage(),
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.update(func() {
			if a.wsAsk != d {
				return
			}
			a.applyWorkspaceFolderPick(d, h, paths[0])
		})
	}()
}

func (a *App) applyWorkspaceFolderPick(d *workspaceAsk, h *host, dir string) {
	d.dir = dir
	d.err = ""
	if h != nil && h != d.host {
		d.host = h
		d.shell = ""
	}
	if strings.TrimSpace(d.name) == "" && dir != "" {
		d.name = proto.DirName(dir)
	}
}

func (a *App) createWorkspace(h *host, cl *client.Client, name, dir, command, shell string) {
	var p proto.Project
	err := cl.Call("project.create", proto.ProjectParams{
		Name:    name,
		Dir:     dir,
		Command: command,
		Shell:   shell,
	}, &p)
	if err != nil {
		a.update(func() {
			if h.cl == cl {
				a.toast(err.Error())
			}
		})
		return
	}
	// Apply command/shell if connecting to an older daemon or reusing an existing project.
	if command != "" && p.Command != command {
		if err := cl.Call("project.command", proto.ProjectParams{ID: p.ID, Command: command}, nil); err != nil {
			a.update(func() {
				if h.cl == cl {
					a.toast(err.Error())
				}
			})
			return
		}
		p.Command = command
	}
	if shell != "" && p.Shell != shell {
		if err := cl.Call("project.shell", proto.ProjectParams{ID: p.ID, Shell: shell}, nil); err != nil {
			a.update(func() {
				if h.cl == cl {
					a.toast(err.Error())
				}
			})
			return
		}
		p.Shell = shell
	}

	var s proto.Session
	if err := cl.Call("create", proto.CreateParams{Project: p.ID, Dir: p.Dir}, &s); err != nil {
		a.update(func() {
			if h.cl == cl {
				a.toast(err.Error())
			}
		})
		return
	}

	a.update(func() {
		if h.cl != cl {
			return
		}
		delete(a.prefs.Collapsed, p.ID)
		if a.project(p.ID) == nil {
			a.st.Projects = append(a.st.Projects, p)
		}
		a.addPending(h, s)
		a.selectProject(p.ID)
		a.selectSession(s.ID)
	})
}
