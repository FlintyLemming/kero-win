package main

import (
	"fmt"
	"slices"
	"time"

	"kero/internal/actions"
	"kero/internal/proto"
	"kero/locales"

	"github.com/egoist/mygo/ui"
)

// tabDrag is a session tab dragged along the top tab strip to reorder
// sessions within its project.
type tabDrag struct {
	key string // project group key
	id  string // session id
}

// tabMove is the tab currently being dragged in the top strip and the
// slot index where dropping it would insert.
type tabMove struct {
	key string
	id  string
	at  int
}

func (a *App) groupKeyOf(s *proto.Session) string {
	if s == nil {
		return ""
	}
	h := a.hostOf(s.ID)
	for _, g := range a.hostGroups(h) {
		for _, o := range g.sessions {
			if o.ID == s.ID {
				return g.key()
			}
		}
	}
	return ""
}

func (a *App) selectedGroup() (group, bool) {
	gs := a.groups()
	if len(gs) == 0 {
		return group{}, false
	}
	if a.selectedProject != "" {
		for _, g := range gs {
			if g.key() == a.selectedProject {
				if s := a.session(a.selected); s != nil && a.groupKeyOf(s) != g.key() {
					break
				}
				return g, true
			}
		}
	}
	if s := a.session(a.selected); s != nil {
		key := a.groupKeyOf(s)
		for _, g := range gs {
			if g.key() == key {
				a.selectedProject = key
				return g, true
			}
		}
	}
	return gs[0], true
}

func (a *App) sessionTabsOrder() []string {
	g, ok := a.selectedGroup()
	if !ok {
		return nil
	}
	ids := make([]string, len(g.sessions))
	for i, s := range g.sessions {
		ids[i] = s.ID
	}
	return ids
}

func (a *App) selectProject(key string) {
	for _, g := range a.groups() {
		if g.key() != key {
			continue
		}
		a.renaming, a.renamingProject = "", ""
		a.selectedProject = key
		target := ""
		if prev := a.projectSession[key]; prev != "" {
			if slices.ContainsFunc(g.sessions, func(s *proto.Session) bool { return s.ID == prev }) {
				target = prev
			}
		}
		if target == "" && len(g.sessions) > 0 {
			target = g.sessions[0].ID
		}
		if target != "" {
			a.selectSession(target)
		} else {
			a.selected = ""
			a.reportView()
			a.updateTitle()
			a.savePrefs()
		}
		return
	}
}

func (a *App) moveSessionIndex(id, project string, index int) {
	s := a.session(id)
	if s == nil {
		return
	}
	h := a.hostOf(id)
	cur := slices.IndexFunc(a.st.Sessions, func(x proto.Session) bool { return x.ID == id })
	if cur >= 0 {
		moved := a.st.Sessions[cur]
		if project != "" {
			moved.Project = project
		}
		a.st.Sessions = slices.Delete(a.st.Sessions, cur, cur+1)
		at, n := len(a.st.Sessions), 0
		for j := range a.st.Sessions {
			o := &a.st.Sessions[j]
			if o.Project != moved.Project || a.hostOf(o.ID) != h {
				continue
			}
			if index >= 0 && n == index {
				at = j
				break
			}
			n++
			at = j + 1
		}
		a.st.Sessions = slices.Insert(a.st.Sessions, at, moved)
	}
	a.revealTab = id
	a.call(id, "move", proto.MoveParams{ID: id, Project: project, Index: index}, nil)
}

func (a *App) reorderTab(id string, g group, at int) {
	if at < 0 {
		return
	}
	cur := slices.IndexFunc(g.sessions, func(s *proto.Session) bool { return s.ID == id })
	if cur < 0 || at == cur || at == cur+1 {
		return
	}
	target := at
	if at > cur {
		target--
	}
	a.moveSessionIndex(id, g.project.ID, target)
}

func projectStatusSummary(sessions []*proto.Session) (st proto.Status, unread bool, progress int, needs int) {
	progress = -1
	st = proto.Idle
	hasSession := false
	rank := func(x proto.Status, u bool) int {
		switch {
		case x == proto.Waiting:
			return 5
		case x == proto.Failed:
			return 4
		case x == proto.Working:
			return 3
		case u:
			return 2
		case x == proto.Idle:
			return 1
		default:
			return 0
		}
	}
	for _, s := range sessions {
		hasSession = true
		if s.Status == proto.Waiting {
			needs++
		}
		if s.Unread {
			unread = true
		}
		if s.Status == proto.Working && s.Progress > progress {
			progress = s.Progress
		}
		if rank(s.Status, s.Unread) > rank(st, false) {
			st = s.Status
		}
	}
	if !hasSession {
		return "", false, -1, 0
	}
	return st, unread, progress, needs
}

// projectGroupView renders a project as a compact selectable row in the
// left sidebar when TopTabs mode is active, reusing project drag targets.
func (a *App) projectGroupView(c *ui.Context, pal *palette, g group, gi, at int, n *int) {
	grp := ui.Column(c.Key("group-" + g.key())).Shrink(0)
	a.projectTarget(grp, g, at)
	moving := g.project.ID != "" && a.projMove.id == g.project.ID
	grp.Opacity(grp.Animate("moving", 1-0.6*b2f(moving), 120*time.Millisecond))
	grp.Children(func() {
		a.projectRow(c, pal, g, gi, moving, n)
		if a.projMove.id != "" && a.projMove.host == g.host && at >= 0 {
			a.insertionLine(c, pal, g, gi, at)
		}
	})
}

func (a *App) projectRow(c *ui.Context, pal *palette, g group, gi int, moving bool, n *int) {
	*n++
	index := *n
	id, key := g.project.ID, g.key()
	activeG, hasActive := a.selectedGroup()
	selected := hasActive && activeG.key() == key
	renaming := id != "" && a.renamingProject == id
	st, unread, progress, needs := projectStatusSummary(g.sessions)

	top := float32(0)
	if gi == 0 {
		top = 2
	}
	row := ui.Row(c.Key("project-"+key)).Height(rowH2).Margin(top, 0, 2, 0).Padding(0, 10, 0, rowInset).Gap(10).AlignItems(ui.Center).Radius(8)
	if pal.dark {
		row.Border(1, ui.Transparent)
	}
	hovered := row.Hovered() && a.projMove.id == ""
	bg := ui.Transparent
	switch {
	case selected:
		bg = pal.rowSelected
		chosen(row, pal)
	case hovered:
		bg = pal.rowHover
	}
	row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 110 * time.Millisecond}))

	if g.host.remote() && g.host.conn != connReady {
		row.Opacity(0.55)
	}
	if id != "" && !renaming && len(a.hostProjects(g.host)) > 1 {
		row.Drag(projectDrag{g.host, id})
		if row.Dragging() {
			a.projMove.host, a.projMove.id = g.host, id
			floating(row.Background(pal.popover), pal)
		} else if moving {
			a.defer_(func() {
				if a.projMove.id == id {
					a.projMove = projectMove{at: -1}
				}
			})
		}
	}
	if (row.Pressed() || row.Clicked()) && !selected && !renaming {
		a.selectProject(key)
	}
	if row.DoubleClicked() && id != "" {
		a.startProjectRename(id)
	}
	pointerTooltip(c, row, "tip-project-"+key, a.projectTooltip(g))
	row.ContextMenu(func(m *ui.Menu) {
		a.projectContextMenu(m, g, false, false)
	})

	row.Children(func() {
		if renaming {
			in := ui.TextInputBase(c.Key("rename"), &a.renameBuf).Grow(1).MinWidth(0).FontSize(13.5).FontWeight(600).AutoFocus()
			if in.Shortcut(0, ui.KeyEscape) {
				a.renamingProject = ""
				a.focusTerminal()
			} else if in.Submitted() || (!in.Focused() && !in.Pressed() && in.Valid() && a.renameStarted(c)) {
				a.commitProjectRename()
			}
			return
		}
		ico := folderIcon
		icoColor := pal.textMuted
		if id == "" {
			ico = terminalIcon
		}
		if selected {
			icoColor = pal.accent
		}
		ui.Icon(c, ico).Size(16, 16).TextColor(icoColor).Shrink(0)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				title := ui.Text(c, g.project.Name).FontSize(13.5).FontWeight(600).SingleLine().Ellipsis("…").Shrink(1).MinWidth(0).TextColor(pal.text)
				if unread {
					title.FontWeight(650)
				} else if !selected && len(g.sessions) == 0 {
					title.TextColor(pal.textMuted)
				}
				if g.host.remote() {
					a.hostTag(c, pal, g.host)
				}
				ui.Spacer(c)
				if g.first && g.host.remote() && !a.hostNeedsRow(g.host) {
					a.hostState(c, pal, g.host)
				}
				if c.Modifiers() == ui.Cmd && index <= 9 {
					ui.Text(c, actions.Glyphs(fmt.Sprintf("CmdOrCtrl+%d", index))).FontSize(12).TextColor(pal.textMuted).FontFeatures("tnum").SingleLine().Shrink(0)
				}
			})
			subColor := pal.textFaint
			subText := ""
			switch {
			case needs > 0:
				subText = locales.SidebarNeedsYou(needs)
				subColor = pal.attention
			case len(g.sessions) == 0:
				subText = locales.SidebarNoSessions()
			case a.set.SidebarCount:
				subText = locales.SidebarSessionsCount(len(g.sessions))
			case g.project.Dir != "":
				subText = g.host.shortDir(g.project.Dir)
			default:
				subText = locales.SidebarSessionsCount(len(g.sessions))
			}
			sub := ui.Text(c, subText).FontSize(12).TextColor(subColor).SingleLine().Ellipsis("…").MinWidth(0)
			if needs > 0 {
				sub.FontWeight(500)
			}
		})
		ui.Box(c.Key("trailing")).Size(20, 20).Shrink(0).Center().Children(func() {
			if hovered && id != "" {
				b := ui.ButtonBase(c.Key("close-project")).Size(20, 20).Radius(5).Center().TextColor(pal.textMuted).
					Label(locales.SidebarCloseProject()).Tooltip(locales.SidebarCloseProject())
				if b.Hovered() {
					b.Background(pal.rowHover).TextColor(pal.text)
				}
				b.Children(func() { ui.Icon(c, closeIcon).Size(13, 13) })
				if b.Clicked() {
					name, count := g.project.Name, len(g.sessions)
					a.defer_(func() {
						if count == 0 {
							a.call(id, "project.close", proto.IDParams{ID: id}, nil)
							return
						}
						a.confirm = &confirmation{
							open:    true,
							title:   locales.ConfirmCloseTitle(name),
							detail:  locales.ConfirmCloseProjectDetail(count),
							action:  locales.SidebarCloseProject(),
							destroy: true,
							run:     func() { a.call(id, "project.close", proto.IDParams{ID: id}, nil) },
						}
					})
				}
				return
			}
			if st != "" {
				statusIconP(c, pal, st, unread, progress, 15)
			}
		})
	})
}

// topTabsBar renders the horizontal tab strip of sessions belonging to the
// selected project along the top of the workspace area.
func (a *App) topTabsBar(c *ui.Context, pal *palette, bar ui.TitleBar, shown float32) {
	padL := max(lead(bar)+34-(a.chrome.sideW+a.chrome.mainPad), 0)
	g, hasGroup := a.selectedGroup()
	at := a.tabMove.at
	a.tabMove.at = -1

	barRow := ui.Row(c.Key("top-tabs")).Height(headerH).Shrink(0).AlignItems(ui.Center).
		Padding(0, 0, 0, padL).Margin(0, 0, cardGap, 0).Gap(4).DragWindow()
	barRow.Children(func() {
		if shown < 0.5 {
			ui.Row(c.Key("tab-sidebar-btn")).Absolute().Left(a.chrome.buttonsLeft(bar)).Top(0).Bottom(0).
				AlignItems(ui.Center).Opacity(1 - 2*shown).Children(func() {
				iconButton(c, "show-sidebar", sidebarIcon, a.tip(locales.SidebarShow(), "sidebar.toggle"), pal, a.toggleSidebar)
			})
		}
		if hasGroup {
			strip := ui.ScrollHorizontal(c.Key("tab-strip")).Shrink(1).MinWidth(0).AlignSelf(ui.Stretch).
				AlignItems(ui.Center).Gap(4).Padding(0, 2, 0, 0).TrackScroll(&a.tabScroll)
			if len(g.sessions) > 1 {
				if d, ok := ui.Drop[tabDrag](strip); ok && d.key == g.key() && at >= 0 {
					a.tabMove = tabMove{at: -1}
					a.defer_(func() { a.reorderTab(d.id, g, at) })
				}
			}
			strip.Children(func() {
				for i, s := range g.sessions {
					a.sessionTabItem(c, pal, g, s, i, at)
				}
			})
			iconButton(c, "new-tab", plusIcon, a.tip(locales.ActionNewSession(), "session.new"), pal, func() {
				a.newSessionOn(g.host, g.project.ID)
			})
		} else {
			iconButton(c, "new-tab", plusIcon, a.tip(locales.ActionNewSession(), "session.new"), pal, func() {
				a.newSession("")
			})
		}
		ui.Box(c.Key("drag")).Grow(1).MinWidth(12).AlignSelf(ui.Stretch).DragWindow()
		if a.workSession() != nil {
			a.showRightButton(c, pal)
		}
		a.controlsRoom(c, 8)
	})
}

func (a *App) sessionTabItem(c *ui.Context, pal *palette, g group, s *proto.Session, i, at int) {
	key := g.key()
	sid := s.ID
	selected := sid == a.selected
	renaming := a.renaming == sid
	moving := a.tabMove.id == sid && a.tabMove.key == key

	slot := ui.Row(c.Key("tab-slot-" + sid)).Shrink(0).AlignItems(ui.Center)
	if len(g.sessions) > 1 {
		if d, over := ui.DragOver[tabDrag](slot); over && d.key == key {
			if x, _, in := slot.PointerPosition(); in {
				place := i
				if w := slot.Bounds().W; x >= w/2 {
					place = i + 1
				}
				a.tabMove.at = place
			}
		}
		if d, ok := ui.Drop[tabDrag](slot); ok && d.key == key && at >= 0 {
			a.tabMove = tabMove{at: -1}
			a.defer_(func() { a.reorderTab(d.id, g, at) })
		}
	}
	slot.Children(func() {
		if a.tabMove.id != "" && a.tabMove.key == key && at == i {
			ui.Box(c.Key("drop-l")).Width(2).Height(22).Radius(1).Background(pal.accent).Margin(0, 2, 0, 0).PassThrough()
		}
		tab := ui.Row(c.Key("tab-" + sid)).Height(34).MinWidth(112).MaxWidth(210).Shrink(0).
			Padding(0, 6, 0, 10).Gap(6).AlignItems(ui.Center).Radius(9)
		if a.revealTab == sid {
			a.revealTab = ""
			tab.ScrollIntoView()
		}
		hovered := tab.Hovered() && a.tabMove.id == ""
		tab.Border(1, ui.Transparent)
		bg := ui.Transparent
		switch {
		case selected:
			bg = pal.card
			tab.Border(1, pal.cardBorder)
		case hovered:
			bg = pal.rowHover
		}
		tab.Background(tab.AnimateColor("bg", bg, ui.Animation{Duration: 110 * time.Millisecond}))
		tab.Opacity(tab.Animate("moving", 1-0.6*b2f(moving), 120*time.Millisecond))
		if h := a.hostOf(sid); h.remote() && h.conn != connReady {
			tab.Opacity(0.55)
		}
		if !renaming && len(g.sessions) > 1 {
			tab.Drag(tabDrag{key, sid})
			if tab.Dragging() {
				a.tabMove.key, a.tabMove.id = key, sid
				floating(tab.Background(pal.popover), pal)
			} else if moving {
				a.defer_(func() {
					if a.tabMove.id == sid {
						a.tabMove = tabMove{at: -1}
					}
				})
			}
		}
		if (tab.Pressed() || tab.Clicked()) && !selected && !renaming {
			a.selectSession(sid)
		}
		if tab.DoubleClicked() {
			a.startRename(sid)
		}
		tip := a.hostOf(sid).shortDir(s.WorkDir())
		if s.Process != "" {
			tip = s.Process + " · " + tip
		}
		pointerTooltip(c, tab, "tip-tab-"+sid, tip)
		tab.ContextMenu(func(m *ui.Menu) {
			a.sessionMenu(m, s)
			if len(g.sessions) > 1 {
				var others, toRight []string
				for j, o := range g.sessions {
					if o.ID != sid {
						others = append(others, o.ID)
					}
					if j > i {
						toRight = append(toRight, o.ID)
					}
				}
				m.Separator()
				if m.Item(locales.TabsCloseOthers()).Disabled(len(others) == 0).Chosen() {
					a.defer_(func() { a.closeSessionIDs(locales.TabsCloseOthers(), others) })
				}
				if m.Item(locales.TabsCloseRight()).Disabled(len(toRight) == 0).Chosen() {
					a.defer_(func() { a.closeSessionIDs(locales.TabsCloseRight(), toRight) })
				}
			}
		})
		tab.Children(func() {
			if renaming {
				in := ui.TextInputBase(c.Key("rename"), &a.renameBuf).Grow(1).MinWidth(60).FontSize(13).AutoFocus()
				if in.Submitted() || in.Shortcut(0, ui.KeyEscape) {
					if in.Shortcut(0, ui.KeyEscape) {
						a.renaming = ""
						a.focusTerminal()
					} else {
						a.commitRename()
					}
				} else if !in.Focused() && !in.Pressed() && in.Valid() && a.renameStarted(c) {
					a.commitRename()
				}
				return
			}
			ui.Icon(c, programIcon(&s.Pane)).Size(14, 14).TextColor(programColor(pal, &s.Pane)).Shrink(0)
			name := ui.Text(c, s.DisplayName(a.set.SessionName)).FontSize(13).SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
			switch {
			case s.Unread:
				name.FontWeight(650).TextColor(pal.text)
			case selected:
				name.FontWeight(550).TextColor(pal.text)
			case s.Status == proto.Exited:
				name.TextColor(pal.textFaint)
			default:
				name.TextColor(pal.textMuted)
			}
			if len(s.Panes) > 1 {
				ui.Textf(c, "%d", len(s.Panes)).FontSize(11).TextColor(pal.textFaint).FontFeatures("tnum").
					Padding(1, 5).Radius(4).Background(pal.chip).Shrink(0)
			}
			ui.Box(c.Key("trailing")).Size(18, 18).Shrink(0).Center().Children(func() {
				if hovered {
					b := ui.ButtonBase(c.Key("close")).Size(18, 18).Radius(4).Center().TextColor(pal.textMuted).
						Label(locales.ActionCloseSession()).Tooltip(locales.ActionCloseSession())
					if b.Hovered() {
						b.Background(pal.rowHover).TextColor(pal.text)
					}
					b.Children(func() { ui.Icon(c, closeIcon).Size(12, 12) })
					if b.Clicked() {
						a.defer_(func() { a.closeSession(sid, true) })
					}
					return
				}
				if s.Status != proto.Idle || s.Unread {
					statusIconP(c, pal, s.Status, s.Unread, s.Progress, 13)
				}
			})
		})
		if a.tabMove.id != "" && a.tabMove.key == key && i == len(g.sessions)-1 && at == len(g.sessions) {
			ui.Box(c.Key("drop-r")).Width(2).Height(22).Radius(1).Background(pal.accent).Margin(0, 0, 0, 2).PassThrough()
		}
	})
}

func (a *App) closeSessionIDs(title string, ids []string) {
	if len(ids) == 0 {
		return
	}
	hasBusy := false
	for _, id := range ids {
		if s := a.session(id); s != nil && (busy(s) || s.Worktree != nil) {
			hasBusy = true
			break
		}
	}
	run := func() {
		for _, id := range ids {
			a.closeSession(id, false)
		}
	}
	if a.set.ConfirmClose && hasBusy {
		a.confirm = &confirmation{
			open:    true,
			title:   title,
			detail:  locales.ConfirmCloseAllSessionsDetail(len(ids)),
			action:  locales.ActionClose(),
			destroy: true,
			run:     run,
		}
		return
	}
	run()
}
