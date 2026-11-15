package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"kero/internal/proto"
	"kero/locales"
)

func TestTopLeftPlusOpensNewWorkspaceDialog(t *testing.T) {
	a := testApp(t)
	created := make(chan proto.CreateParams, 2)
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		if method == "create" {
			var p proto.CreateParams
			_ = json.Unmarshal(raw, &p)
			created <- p
			return proto.Session{ID: "s-new", Project: p.Project}, nil
		}
		return nil, nil
	})
	a.local().conn = connReady
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	tt.Frame()

	// Expanded sidebar top-left + opens New Workspace instead of creating a session.
	if err := tt.Click(a.tip(locales.ActionNewWorkspace(), "project.new")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.wsAsk == nil || !a.wsAsk.open {
		t.Fatal("expanded sidebar + did not open New Workspace dialog")
	}
	if !tt.HasText(locales.WorkspaceNewTitle()) {
		t.Fatalf("missing New Workspace dialog title; texts=%q", tt.Texts())
	}
	select {
	case p := <-created:
		t.Fatalf("top-left + unexpectedly started session directly: %+v", p)
	default:
	}

	// Cancel closes the dialog.
	if err := tt.Click(locales.CommonCancel()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.wsAsk != nil {
		t.Fatal("Cancel did not clear workspace dialog")
	}

	// Collapsed sidebar (standard mode) top-left + also opens New Workspace.
	a.prefs.SidebarHidden = true
	tt.Frame()
	if err := tt.Click(a.tip(locales.ActionNewWorkspace(), "project.new")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.wsAsk == nil || !a.wsAsk.open {
		t.Fatal("collapsed sidebar + did not open New Workspace dialog")
	}

	// Close dialog, enable TopTabs, and confirm the session bar + still starts a session.
	a.wsAsk = nil
	a.set.TopTabs = true
	a.selectedProject = "p1"
	tt.Frame()
	if err := tt.Click(a.tip(locales.ActionNewSession(), "session.new")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	select {
	case p := <-created:
		if p.Project != "p1" {
			t.Fatalf("TopTabs new-tab created for %q, want p1", p.Project)
		}
	case <-time.After(time.Second):
		t.Fatal("TopTabs new-tab + did not create a session")
	}
	if a.wsAsk != nil {
		t.Fatal("TopTabs new-tab + should not open New Workspace dialog")
	}
}

func TestNewWorkspaceCreateNameOnlyWithShellAndCommand(t *testing.T) {
	a := testApp(t)
	a.local().shellsLoaded = true
	a.local().shells = []proto.ShellOption{
		{ID: "pwsh", Name: "PowerShell", Kind: "pwsh"},
	}

	projCreated := make(chan proto.ProjectParams, 1)
	sessCreated := make(chan proto.CreateParams, 1)
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		switch method {
		case "project.create":
			var p proto.ProjectParams
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, err
			}
			projCreated <- p
			return proto.Project{
				ID:      "p-scratch",
				Name:    p.Name,
				Dir:     p.Dir,
				Command: p.Command,
				Shell:   p.Shell,
			}, nil
		case "create":
			var p proto.CreateParams
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, err
			}
			sessCreated <- p
			return proto.Session{ID: "s-scratch", Project: p.Project, Name: "Scratch"}, nil
		}
		return nil, nil
	})
	a.local().conn = connReady
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	tt.Frame()
	if err := tt.Click(a.tip(locales.ActionNewWorkspace(), "project.new")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	// Empty submission validates inline and stays open.
	if err := tt.Click(locales.WorkspaceCreate()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.wsAsk == nil || !a.wsAsk.open {
		t.Fatal("empty submission closed dialog instead of showing validation error")
	}
	if !tt.HasText(locales.WorkspaceNeedNameOrFolder()) {
		t.Fatalf("missing validation message %q; texts=%q", locales.WorkspaceNeedNameOrFolder(), tt.Texts())
	}

	// Name is auto-focused; typing clears error and sets workspace name.
	tt.Type("Scratch")
	tt.Frame()
	if a.wsAsk.err != "" {
		t.Fatalf("typing Name did not clear validation error: %q", a.wsAsk.err)
	}

	// Choose Default Shell and Default Command.
	a.wsAsk.shell = "pwsh"
	a.wsAsk.command = "npm run dev"
	tt.Frame()

	if err := tt.Click(locales.WorkspaceCreate()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.wsAsk != nil {
		t.Fatal("valid submission left New Workspace dialog open")
	}

	select {
	case got := <-projCreated:
		if got.Name != "Scratch" || got.Dir != "" || got.Shell != "pwsh" || got.Command != "npm run dev" {
			t.Fatalf("project.create params = %+v, want Name=Scratch Dir=\"\" Shell=pwsh Command=\"npm run dev\"", got)
		}
	case <-time.After(time.Second):
		t.Fatal("did not call project.create")
	}
	select {
	case got := <-sessCreated:
		if got.Project != "p-scratch" || got.Dir != "" {
			t.Fatalf("create params = %+v, want Project=p-scratch Dir=\"\"", got)
		}
	case <-time.After(time.Second):
		t.Fatal("did not call create for initial session")
	}
}

func TestNewWorkspaceFolderValidationAndCreation(t *testing.T) {
	a := testApp(t)
	projCreated := make(chan proto.ProjectParams, 1)
	sessCreated := make(chan proto.CreateParams, 1)
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		switch method {
		case "project.create":
			var p proto.ProjectParams
			_ = json.Unmarshal(raw, &p)
			projCreated <- p
			return proto.Project{ID: "p-dir", Name: p.Name, Dir: p.Dir}, nil
		case "create":
			var p proto.CreateParams
			_ = json.Unmarshal(raw, &p)
			sessCreated <- p
			return proto.Session{ID: "s-dir", Project: p.Project, Dir: p.Dir}, nil
		}
		return nil, nil
	})
	a.local().conn = connReady
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	a.startNewWorkspace(nil)
	a.wsAsk.dir = filepath.Join(t.TempDir(), "does-not-exist")
	tt.Frame()

	if err := tt.Click(locales.WorkspaceCreate()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.wsAsk == nil || !a.wsAsk.open || !tt.HasText(locales.WorkspaceInvalidFolder()) {
		t.Fatalf("missing invalid folder error; texts=%q", tt.Texts())
	}

	dir := t.TempDir()
	a.wsAsk.dir = dir
	a.wsAsk.err = ""
	tt.Frame()
	if err := tt.Click(locales.WorkspaceCreate()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	select {
	case got := <-projCreated:
		wantName := proto.DirName(dir)
		if got.Name != wantName || got.Dir != dir {
			t.Fatalf("project.create = %+v, want Name=%q Dir=%q", got, wantName, dir)
		}
	case <-time.After(time.Second):
		t.Fatal("did not call project.create for valid folder")
	}
	select {
	case got := <-sessCreated:
		if got.Project != "p-dir" || got.Dir != dir {
			t.Fatalf("create = %+v, want Project=p-dir Dir=%q", got, dir)
		}
	case <-time.After(time.Second):
		t.Fatal("did not call create for folder workspace")
	}
}

func TestSidebarDropRejectedWhileNewWorkspaceOpen(t *testing.T) {
	a := testApp(t)
	a.local().conn = connReady
	cl, stop := projectClient(t, func(string, json.RawMessage) (any, error) { return nil, nil })
	a.local().cl = cl
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	a.startNewWorkspace(nil)
	tt.Frame()
	if tt.DropFiles(80, 120, t.TempDir()) {
		t.Fatal("sidebar accepted folder drop while New Workspace dialog was open")
	}
}

func TestPaletteSkipsEmptyRemoteTrailingSeparator(t *testing.T) {
	a := testApp(t)
	remote := &host{name: "devbox"}
	a.hosts = append(a.hosts, remote)
	if a.owner == nil {
		a.owner = map[string]*host{}
	}
	a.owner["p-remote"] = remote
	a.st.Projects = append(a.st.Projects, proto.Project{ID: "p-remote", Name: "RemoteScratch", Dir: ""})

	items := a.paletteItems()
	for _, it := range items {
		if it.title == locales.NewSessionIn("RemoteScratch") {
			if it.detail != "devbox" {
				t.Fatalf("palette detail = %q, want %q", it.detail, "devbox")
			}
			return
		}
	}
	t.Fatal("missing palette item for RemoteScratch")
}

func TestNewWorkspaceRemotePickerReturnsFolder(t *testing.T) {
	a := testApp(t)
	remote := &host{name: "devbox", conn: connReady}
	cl, stop := projectClient(t, func(method string, raw json.RawMessage) (any, error) {
		if method == "fs.list" {
			return proto.FSList{
				Dir: "/srv",
				Entries: []proto.FSEntry{
					{Name: "app", Dir: true},
				},
			}, nil
		}
		return nil, nil
	})
	remote.cl = cl
	a.hosts = append(a.hosts, remote)
	defer stop()

	tt := ui.NewTester(a.view, 1100, 640)
	a.startNewWorkspace(remote)
	a.wsAsk.command = "make dev"
	tt.Frame()

	// Click Choose... -> opens remote folder picker while preserving wsAsk.
	if err := tt.Click(locales.SetChooseFolder()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.picker.open || a.picker.host != remote {
		t.Fatalf("picker did not open on remote host: %+v", a.picker)
	}

	// Simulate listing loaded on UI thread and click Open.
	a.picker.list = &proto.FSList{
		Dir: "/srv",
		Entries: []proto.FSEntry{
			{Name: "app", Dir: true},
		},
	}
	tt.Frame()
	if err := tt.Click(locales.CommonOpen()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	if a.picker.open {
		t.Fatal("picker stayed open after selecting folder")
	}
	if a.wsAsk == nil || !a.wsAsk.open {
		t.Fatal("New Workspace dialog did not return after folder pick")
	}
	if a.wsAsk.dir != "/srv/app" {
		t.Fatalf("picked dir = %q, want /srv/app", a.wsAsk.dir)
	}
	if a.wsAsk.name != "app" {
		t.Fatalf("auto-filled name = %q, want app", a.wsAsk.name)
	}
	if a.wsAsk.command != "make dev" {
		t.Fatalf("command lost across folder pick: %q", a.wsAsk.command)
	}
}
