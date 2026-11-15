package main

import (
	"encoding/json"
	"testing"
	"time"

	"kero/internal/proto"
	"kero/locales"

	"github.com/egoist/mygo/ui"
)

func topTabsTestApp(t *testing.T) *App {
	t.Helper()
	a := testApp(t)
	a.set.TopTabs = true
	a.st = proto.State{
		Projects: []proto.Project{
			{ID: "p1", Name: "Alpha", Dir: "/code/alpha"},
			{ID: "p2", Name: "Beta", Dir: "/code/beta"},
		},
		Sessions: []proto.Session{
			{ID: "s1", Project: "p1", Name: "one", Pane: proto.Pane{ID: "a", Process: "zsh", Dir: "/code/alpha", Status: proto.Working}, Panes: []proto.Pane{{ID: "a", Process: "zsh", Dir: "/code/alpha", Status: proto.Working}}},
			{ID: "s2", Project: "p1", Name: "two", Pane: proto.Pane{ID: "b", Process: "zsh", Dir: "/code/alpha"}, Panes: []proto.Pane{{ID: "b", Process: "zsh", Dir: "/code/alpha"}}},
			{ID: "s3", Project: "p2", Name: "three", Pane: proto.Pane{ID: "c", Process: "zsh", Dir: "/code/beta"}, Panes: []proto.Pane{{ID: "c", Process: "zsh", Dir: "/code/beta"}}},
			{ID: "s4", Project: "p2", Name: "four", Pane: proto.Pane{ID: "d", Process: "zsh", Dir: "/code/beta"}, Panes: []proto.Pane{{ID: "d", Process: "zsh", Dir: "/code/beta"}}},
		},
	}
	a.selected = "s1"
	a.selectedProject = a.local().name + "\x00p1"
	for _, s := range a.st.Sessions {
		a.own(a.local(), s)
		for _, p := range s.Panes {
			a.terms[p.ID] = &termEntry{}
		}
	}
	for _, p := range a.st.Projects {
		a.owner[p.ID] = a.local()
	}
	return a
}

func TestTopTabsSettingToggle(t *testing.T) {
	a := testApp(t)
	if a.set.TopTabs {
		t.Fatal("TopTabs should default to false")
	}
	tt := ui.NewTester(a.settingsView, 780, 900)
	a.ss.query = "top-tabs"
	tt.Frame()
	if !tt.HasText(locales.SetTopTabs()) {
		t.Fatalf("missing setting %q in search results; texts=%q", locales.SetTopTabs(), tt.Texts())
	}
	row, ok := tt.Find(locales.SetTopTabs())
	if !ok {
		t.Fatal("missing setting row bounds")
	}
	// Click the toggle switch near the right edge of the setting row.
	tt.ClickAt(716, row.Y+row.H)
	tt.Frame()
	if !a.set.TopTabs {
		t.Fatalf("clicking top-tabs switch did not enable TopTabs (row=%+v)", row)
	}
}

func TestTopTabsLayoutAndProjectSwitching(t *testing.T) {
	a := topTabsTestApp(t)
	tt := ui.NewTester(a.view, 1100, 640)
	tt.Frame()

	// Projects are in the left sidebar; non-selected project sessions are not.
	alpha, ok := tt.Find("Alpha")
	if !ok || alpha.X >= a.prefs.SidebarWidth {
		t.Fatalf("Alpha project not in sidebar: found=%v bounds=%+v", ok, alpha)
	}
	beta, ok := tt.Find("Beta")
	if !ok || beta.X >= a.prefs.SidebarWidth {
		t.Fatalf("Beta project not in sidebar: found=%v bounds=%+v", ok, beta)
	}
	if tt.HasText("three") || tt.HasText("four") {
		t.Fatalf("sessions of unselected project should not appear; texts=%q", tt.Texts())
	}

	// Selected project's sessions appear horizontally along the top bar.
	tab1, ok1 := tt.Find("one")
	tab2, ok2 := tt.Find("two")
	if !ok1 || !ok2 {
		t.Fatalf("missing top tabs: one=%v two=%v texts=%q", ok1, ok2, tt.Texts())
	}
	if tab1.X < a.prefs.SidebarWidth || tab2.X <= tab1.X || tab1.Y >= topBar || tab2.Y >= topBar {
		t.Fatalf("tabs not positioned horizontally in top bar: tab1=%+v tab2=%+v", tab1, tab2)
	}

	// Clicking tab two selects s2.
	if err := tt.Click("two"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.selected != "s2" {
		t.Fatalf("selected = %q, want s2", a.selected)
	}

	// Switching project in sidebar shows Beta's tabs along the top.
	if err := tt.Click("Beta"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.selected != "s3" {
		t.Fatalf("after clicking Beta, selected = %q, want s3", a.selected)
	}
	if tt.HasText("two") || !tt.HasText("three") || !tt.HasText("four") {
		t.Fatalf("top bar did not switch to Beta tabs: texts=%q", tt.Texts())
	}

	// Select tab four in Beta, then jump back to Alpha -> Alpha remembers s2.
	if err := tt.Click("four"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.selected != "s4" {
		t.Fatalf("selected = %q, want s4", a.selected)
	}
	if err := tt.Click("Alpha"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.selected != "s2" {
		t.Fatalf("returning to Alpha selected %q, want remembered s2", a.selected)
	}

	// Next/Prev session stays within the selected project's tabs.
	a.selectOffset(1)
	if a.selected != "s1" {
		t.Fatalf("selectOffset(1) from s2 = %q, want s1", a.selected)
	}

	// Cmd/Ctrl+2 switches to project 2 (Beta), restoring s4.
	tt.Key(ui.Cmd, ui.Key2)
	tt.Frame()
	if a.selected != "s4" {
		t.Fatalf("Cmd+2 selected %q, want Beta's remembered s4", a.selected)
	}
}

func TestTopTabsEmptyProjectAndNewSession(t *testing.T) {
	a := topTabsTestApp(t)
	a.st.Projects = append(a.st.Projects, proto.Project{ID: "p3", Name: "EmptyProj", Dir: "/code/empty"})
	a.owner["p3"] = a.local()

	created := make(chan proto.CreateParams, 1)
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		if method == "create" {
			var p proto.CreateParams
			_ = json.Unmarshal(raw, &p)
			created <- p
			return proto.Session{ID: "s5", Project: p.Project, Name: "five"}, nil
		}
		return nil, nil
	})
	a.local().conn = connReady
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	tt.Frame()
	if err := tt.Click("EmptyProj"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.selected != "" {
		t.Fatalf("empty project left session %q selected", a.selected)
	}
	if !tt.HasText(locales.NewSessionIn("EmptyProj")) {
		t.Fatalf("missing empty project CTA; texts=%q", tt.Texts())
	}
	if err := tt.Click(locales.NewSessionIn("EmptyProj")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	select {
	case p := <-created:
		if p.Project != "p3" {
			t.Fatalf("created project = %q, want p3", p.Project)
		}
	case <-time.After(time.Second):
		t.Fatal("clicking new session in empty project did not call create")
	}
}

func TestTopTabsHoverCloseAndHiddenSidebar(t *testing.T) {
	a := topTabsTestApp(t)
	a.st.Sessions[0].Process = ""
	a.st.Sessions[0].Status = proto.Idle // non-busy so close proceeds directly
	closed := make(chan string, 1)
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		if method == "close" {
			var p proto.CloseParams
			_ = json.Unmarshal(raw, &p)
			closed <- p.ID
		}
		return nil, nil
	})
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	tt.Frame()

	// Hover tab one and click its close button without selecting another project.
	tab1, ok := tt.Find("one")
	if !ok {
		t.Fatal("missing tab one")
	}
	tt.Move(tab1.X+tab1.W/2, tab1.Y+tab1.H/2)
	tt.Frame()
	btn, ok := tt.Find(locales.ActionCloseSession())
	if !ok {
		t.Fatal("tab close button did not appear on hover")
	}
	tt.ClickAt(btn.X+btn.W/2, btn.Y+btn.H/2)
	tt.Frame()
	select {
	case id := <-closed:
		if id != "s1" {
			t.Fatalf("closed = %q, want s1", id)
		}
	case <-time.After(time.Second):
		t.Fatal("close RPC was not called")
	}
	if a.selected != "s2" {
		t.Fatalf("after closing s1, selected = %q, want s2", a.selected)
	}

	// Hide left sidebar and confirm show-sidebar button sits left of tabs without overlap.
	a.prefs.SidebarHidden = true
	tt.Frame()
	tt.Advance(time.Second)
	tt.Frame()
	showBtn, ok := tt.Find(a.tip(locales.SidebarShow(), "sidebar.toggle"))
	if !ok {
		t.Fatalf("show-sidebar button missing from top bar when sidebar hidden; texts=%q", tt.Texts())
	}
	tab2, ok := tt.Find("two")
	if !ok {
		t.Fatal("missing remaining tab two")
	}
	if showBtn.Y >= topBar || showBtn.X+showBtn.W > tab2.X {
		t.Fatalf("show-sidebar %+v overlaps tab2 %+v or outside topBar", showBtn, tab2)
	}
	tt.ClickAt(showBtn.X+showBtn.W/2, showBtn.Y+showBtn.H/2)
	tt.Frame()
	if a.prefs.SidebarHidden {
		t.Fatal("clicking show-sidebar did not restore sidebar")
	}
}

func TestTopTabsDragReorder(t *testing.T) {
	a := topTabsTestApp(t)
	a.st.Sessions = []proto.Session{
		{ID: "s1", Project: "p1", Name: "one", Pane: proto.Pane{ID: "a", Process: "zsh", Dir: "/code/alpha"}, Panes: []proto.Pane{{ID: "a"}}},
		{ID: "s2", Project: "p1", Name: "two", Pane: proto.Pane{ID: "b", Process: "zsh", Dir: "/code/alpha"}, Panes: []proto.Pane{{ID: "b"}}},
		{ID: "s3", Project: "p1", Name: "three", Pane: proto.Pane{ID: "c", Process: "zsh", Dir: "/code/alpha"}, Panes: []proto.Pane{{ID: "c"}}},
	}
	a.terms["c"] = &termEntry{}
	moved := make(chan proto.MoveParams, 1)
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		if method == "move" {
			var p proto.MoveParams
			_ = json.Unmarshal(raw, &p)
			moved <- p
		}
		return nil, nil
	})
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	tt.Frame()
	tab1, ok1 := tt.Find("one")
	tab3, ok3 := tt.Find("three")
	if !ok1 || !ok3 {
		t.Fatalf("missing tabs: one=%v three=%v", ok1, ok3)
	}
	sx, sy := tab1.X+tab1.W/2, tab1.Y+tab1.H/2
	tx, ty := tab3.X+tab3.W-6, tab3.Y+tab3.H/2
	tt.Press(sx, sy)
	for step := 1; step <= 8; step++ {
		f := float32(step) / 8
		tt.Move(sx+(tx-sx)*f, sy+(ty-sy)*f)
		tt.Advance(16 * time.Millisecond)
		tt.Frame()
	}
	tt.Release(tx, ty)
	tt.Frame()
	tt.Frame()
	select {
	case p := <-moved:
		if p.ID != "s1" || p.Project != "p1" || p.Index != 2 {
			t.Fatalf("moved = %+v, want ID=s1 Project=p1 Index=2", p)
		}
	case <-time.After(time.Second):
		t.Fatal("dragging tab did not send move RPC")
	}
}
