package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"

	"kero/internal/proto"
)

func projectDaemon(t *testing.T) *Daemon {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("KERO_CONFIG", filepath.Join(dir, "config"))
	d := &Daemon{dir: dir, changeCh: make(chan struct{}, 1), saveCh: make(chan struct{}, 1)}
	d.viewed.Store(new(string))
	return d
}

func createProject(t *testing.T, d *Daemon, p proto.ProjectParams) (proto.Project, error) {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.call(nil, "project.create", raw)
	if err != nil {
		return proto.Project{}, err
	}
	return result.(proto.Project), nil
}

func TestProjectCreateRejectsInvalidDirectories(t *testing.T) {
	d := projectDaemon(t)
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{file, filepath.Join(t.TempDir(), "missing")} {
		if p, err := createProject(t, d, proto.ProjectParams{Dir: dir}); err == nil {
			t.Errorf("project.create(%q) = %+v, want an error", dir, p)
		}
	}
	if got := d.state().Projects; len(got) != 0 {
		t.Errorf("projects after invalid drops = %+v, want none", got)
	}
}

func TestProjectCreateMatchesFolderPickerPackages(t *testing.T) {
	for _, tc := range []struct {
		name     string
		packaged bool
	}{
		{"ordinary folder", false},
		{"application.app-sources", false},
		{"Application.app/Contents/project", false},
		{"日本語 Application.app", true},
		{"Application.APP", true},
		{"Plugin.bundle", true},
		{"Document.rtfd", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := projectDaemon(t)
			dir := filepath.Join(t.TempDir(), tc.name)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			wantErr := runtime.GOOS == "darwin" && tc.packaged
			first, err := createProject(t, d, proto.ProjectParams{Dir: dir})
			if (err != nil) != wantErr {
				t.Fatalf("project.create(%q) = %+v, %v, want error %t", dir, first, err, wantErr)
			}
			if wantErr && len(d.state().Projects) != 0 {
				t.Fatal("a rejected file package registered a project")
			}
			t.Run("symlink", func(t *testing.T) {
				link := filepath.Join(t.TempDir(), "alias without a package extension")
				if err := os.Symlink(dir, link); err != nil {
					t.Skipf("directory symlinks unavailable: %v", err)
				}
				got, err := createProject(t, d, proto.ProjectParams{Dir: link})
				if (err != nil) != wantErr {
					t.Fatalf("project.create(%q) = %+v, %v, want error %t", link, got, err, wantErr)
				}
				if !wantErr && got != first {
					t.Errorf("symlink project = %+v, want %+v", got, first)
				}
				if wantErr && len(d.state().Projects) != 0 {
					t.Fatal("a symlink to a rejected file package registered a project")
				}
			})
		})
	}
}

func TestProjectCreateReusesDirectoryAliases(t *testing.T) {
	d := projectDaemon(t)
	name := "日本語 # % project"
	if runtime.GOOS != "windows" {
		name += " "
	}
	dir := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := createProject(t, d, proto.ProjectParams{Dir: dir, Name: "My project"})
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, link); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	for _, alias := range []string{dir, dir + string(os.PathSeparator), link} {
		got, err := createProject(t, d, proto.ProjectParams{Dir: alias})
		if err != nil || got != first {
			t.Errorf("project.create(%q) = %+v, %v, want %+v", alias, got, err, first)
		}
	}
	if got := d.state().Projects; !slices.Equal(got, []proto.Project{first}) {
		t.Errorf("projects after aliases = %+v, want [%+v]", got, first)
	}
}

func TestProjectCreateConcurrentAliases(t *testing.T) {
	d := projectDaemon(t)
	dir := t.TempDir()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, link); err != nil {
		t.Logf("directory symlinks unavailable; using a trailing separator: %v", err)
		link = dir + string(os.PathSeparator)
	}
	ids := make([]string, 24)
	var wg sync.WaitGroup
	for i := range ids {
		path := dir
		if i%2 != 0 {
			path = link
		}
		raw, err := json.Marshal(proto.ProjectParams{Dir: path})
		if err != nil {
			t.Fatal(err)
		}
		wg.Go(func() {
			p, err := d.call(nil, "project.create", raw)
			if err != nil {
				t.Errorf("concurrent project.create(%q): %v", path, err)
				return
			}
			ids[i] = p.(proto.Project).ID
		})
	}
	wg.Wait()
	for _, id := range ids {
		if id == "" || id != ids[0] {
			t.Errorf("concurrent project IDs = %v, want one nonempty ID", ids)
			break
		}
	}
	if got := d.state().Projects; len(got) != 1 {
		t.Errorf("projects after concurrent aliases = %+v, want one", got)
	}
}

func TestProjectCreateRejectsInaccessibleDirectories(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix directory permissions without root privileges")
	}
	for _, mode := range []os.FileMode{0, 0o400, 0o100} {
		t.Run(mode.String(), func(t *testing.T) {
			d := projectDaemon(t)
			dir := t.TempDir()
			if err := os.Chmod(dir, mode); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(dir, 0o700) })
			if p, err := createProject(t, d, proto.ProjectParams{Dir: dir}); err == nil {
				t.Errorf("project.create(%q, mode %v) = %+v, want an error", dir, mode, p)
			}
			if got := d.state().Projects; len(got) != 0 {
				t.Errorf("projects after inaccessible drop = %+v, want none", got)
			}
		})
	}
}

func TestProjectCreateRejectsInvalidSymlinks(t *testing.T) {
	d := projectDaemon(t)
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{file, filepath.Join(t.TempDir(), "missing")} {
		link := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if p, err := createProject(t, d, proto.ProjectParams{Dir: link}); err == nil {
			t.Errorf("project.create(symlink to %q) = %+v, want an error", target, p)
		}
	}
}

func TestProjectCreateNormalizesRelativeDirectory(t *testing.T) {
	d := projectDaemon(t)
	base := t.TempDir()
	cwd := filepath.Join(base, "cwd")
	dir := filepath.Join(base, "proj")
	if err := os.MkdirAll(cwd, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	rel, err := filepath.Rel(cwd, dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := createProject(t, d, proto.ProjectParams{Dir: rel})
	if err != nil || p.Dir != dir {
		t.Errorf("project.create(%q) = %+v, %v, want directory %q", rel, p, err, dir)
	}
}

func TestProjectCreateKeepsDistinctDirectories(t *testing.T) {
	d := projectDaemon(t)
	var ids []string
	for range 2 {
		dir := filepath.Join(t.TempDir(), "same-name")
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		p, err := createProject(t, d, proto.ProjectParams{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	if ids[0] == ids[1] || len(d.state().Projects) != 2 {
		t.Errorf("distinct directories with the same name have IDs %v, want two projects", ids)
	}
}

func TestProjectCreateReusesDirectoryCase(t *testing.T) {
	d := projectDaemon(t)
	parent := t.TempDir()
	dir, alias := filepath.Join(parent, "Project"), filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(alias); os.IsNotExist(err) {
		t.Skip("case-sensitive filesystem")
	} else if err != nil {
		t.Fatal(err)
	}
	first, err := createProject(t, d, proto.ProjectParams{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	got, err := createProject(t, d, proto.ProjectParams{Dir: alias})
	if err != nil || got != first {
		t.Errorf("project.create(%q) = %+v, %v, want %+v", alias, got, err, first)
	}
}

func TestProjectCreateNameOnly(t *testing.T) {
	d := projectDaemon(t)
	p, err := createProject(t, d, proto.ProjectParams{Name: "Named project"})
	if err != nil || p.ID == "" || p.Name != "Named project" || p.Dir != "" {
		t.Errorf("name-only project.create = %+v, %v, want a named project without a directory", p, err)
	}
}

func TestProjectCreateCommandAndShell(t *testing.T) {
	d := projectDaemon(t)
	p, err := createProject(t, d, proto.ProjectParams{
		Name:    "Scratch",
		Command: "  echo hi  ",
		Shell:   "  bash  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Command != "echo hi" || p.Shell != "bash" {
		t.Errorf("project.create command/shell = %+v, want trimmed Command and Shell", p)
	}
	dir := t.TempDir()
	withDir, err := createProject(t, d, proto.ProjectParams{
		Dir:     dir,
		Command: "npm run dev",
		Shell:   "zsh",
	})
	if err != nil {
		t.Fatal(err)
	}
	if withDir.Command != "npm run dev" || withDir.Shell != "zsh" {
		t.Errorf("project.create(dir) command/shell = %+v, want Command and Shell saved", withDir)
	}
}

func TestProjectCreatePersists(t *testing.T) {
	d := projectDaemon(t)
	first, err := createProject(t, d, proto.ProjectParams{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	d.saveState()
	restored := &Daemon{dir: d.dir}
	restored.viewed.Store(new(string))
	restored.restore()
	if got := restored.state().Projects; !slices.Equal(got, []proto.Project{first}) {
		t.Errorf("restored projects = %+v, want [%+v]", got, first)
	}
}

// project.move puts a project at another index, past either end at that
// end, and the order outlives kerod.
func TestProjectMovePersists(t *testing.T) {
	d := projectDaemon(t)
	var ids []string
	for _, name := range []string{"a", "b", "c"} {
		p, err := createProject(t, d, proto.ProjectParams{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	move := func(id string, index int) {
		raw, _ := json.Marshal(proto.MoveProjectParams{ID: id, Index: index})
		if _, err := d.call(nil, "project.move", raw); err != nil {
			t.Fatal(err)
		}
	}
	order := func(st proto.State) []string {
		var got []string
		for _, p := range st.Projects {
			got = append(got, p.Name)
		}
		return got
	}
	move(ids[2], 0)
	if got := order(d.state()); !slices.Equal(got, []string{"c", "a", "b"}) {
		t.Errorf("c to the top: %v", got)
	}
	move(ids[2], 99)
	if got := order(d.state()); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("c past the end: %v", got)
	}
	move(ids[0], 1)
	d.saveState()
	restored := &Daemon{dir: d.dir}
	restored.viewed.Store(new(string))
	restored.restore()
	if got := order(restored.state()); !slices.Equal(got, []string{"b", "a", "c"}) {
		t.Errorf("restored order %v, want b a c", got)
	}
}

// project.command sets and clears a project's default command, and persists
// it across kerod restarts.
func TestProjectCommandPersists(t *testing.T) {
	d := projectDaemon(t)
	p, err := createProject(t, d, proto.ProjectParams{Name: "app"})
	if err != nil {
		t.Fatal(err)
	}
	setCmd := func(id, cmd string) error {
		raw, err := json.Marshal(proto.ProjectParams{ID: id, Command: cmd})
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.call(nil, "project.command", raw)
		return err
	}
	if err := setCmd("missing", "claude"); err == nil {
		t.Fatal("project.command succeeded for unknown project")
	}
	if err := setCmd(p.ID, "  npm run dev  "); err != nil {
		t.Fatal(err)
	}
	if got := d.state().Projects[0].Command; got != "npm run dev" {
		t.Fatalf("command = %q, want %q", got, "npm run dev")
	}
	d.saveState()
	restored := &Daemon{dir: d.dir}
	restored.viewed.Store(new(string))
	restored.restore()
	if got := restored.state().Projects[0].Command; got != "npm run dev" {
		t.Fatalf("restored command = %q, want %q", got, "npm run dev")
	}
	if err := setCmd(p.ID, "   "); err != nil {
		t.Fatal(err)
	}
	if got := d.state().Projects[0].Command; got != "" {
		t.Fatalf("cleared command = %q, want empty", got)
	}
}

// New sessions in a project with a default command start that command unless
// another command or a browser URL was given.
func TestProjectDefaultCommandInCreate(t *testing.T) {
	d := testDaemon(t)
	d.projects[0].Command = "echo hello"

	s, err := d.create(proto.CreateParams{Project: "pr"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.removeSession(s.ID) })
	if got := s.Panes[0].Command; !slices.Equal(got, []string{"echo hello"}) {
		t.Errorf("default command = %v, want [echo hello]", got)
	}

	explicit, err := d.create(proto.CreateParams{Project: "pr", Command: []string{"echo", "custom"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.removeSession(explicit.ID) })
	if got := explicit.Panes[0].Command; !slices.Equal(got, []string{"echo", "custom"}) {
		t.Errorf("explicit command = %v, want [echo custom]", got)
	}

	browser, err := d.create(proto.CreateParams{Project: "pr", URL: "localhost:3000"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.removeSession(browser.ID) })
	if len(browser.Panes[0].Command) != 0 || browser.Panes[0].URL != "http://localhost:3000" {
		t.Errorf("browser pane = %+v, want URL without command", browser.Panes[0])
	}
}

// project.shell sets and clears a project's default shell, and persists
// it across kerod restarts.
func TestProjectShellPersists(t *testing.T) {
	d := projectDaemon(t)
	p, err := createProject(t, d, proto.ProjectParams{Name: "app"})
	if err != nil {
		t.Fatal(err)
	}
	setShell := func(id, sh string) error {
		raw, err := json.Marshal(proto.ProjectParams{ID: id, Shell: sh})
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.call(nil, "project.shell", raw)
		return err
	}
	if err := setShell("missing", "cmd"); err == nil {
		t.Fatal("project.shell succeeded for unknown project")
	}
	if err := setShell(p.ID, "  wsl:Ubuntu  "); err != nil {
		t.Fatal(err)
	}
	if got := d.state().Projects[0].Shell; got != "wsl:Ubuntu" {
		t.Fatalf("shell = %q, want %q", got, "wsl:Ubuntu")
	}
	d.saveState()
	restored := &Daemon{dir: d.dir}
	restored.viewed.Store(new(string))
	restored.restore()
	if got := restored.state().Projects[0].Shell; got != "wsl:Ubuntu" {
		t.Fatalf("restored shell = %q, want %q", got, "wsl:Ubuntu")
	}
	if err := setShell(p.ID, "   "); err != nil {
		t.Fatal(err)
	}
	if got := d.state().Projects[0].Shell; got != "" {
		t.Fatalf("cleared shell = %q, want empty", got)
	}
}

func TestHostShellsRPC(t *testing.T) {
	d := projectDaemon(t)
	res, err := d.call(nil, "host.shells", nil)
	if err != nil {
		t.Fatalf("host.shells error: %v", err)
	}
	hs, ok := res.(proto.HostShells)
	if !ok {
		t.Fatalf("host.shells result = %T, want proto.HostShells", res)
	}
	if len(hs.Shells) == 0 {
		t.Fatal("host.shells returned empty Shells")
	}
}

func TestResolveShellAndArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		wslSpec := resolveShell("wsl:Ubuntu")
		if wslSpec.wsl != "Ubuntu" || wslSpec.name != "Ubuntu" {
			t.Fatalf("resolveShell(wsl:Ubuntu) = %+v", wslSpec)
		}
		if args := shellSpecArgs(wslSpec, "", "", nil); !slices.Equal(args, []string{"wsl", "-d", "Ubuntu"}) {
			t.Fatalf("wsl args = %v, want [wsl -d Ubuntu]", args)
		}
		if args := shellSpecArgs(wslSpec, "", "", []string{"echo", "hi"}); len(args) < 4 || args[2] != "Ubuntu" || args[3] != "--" {
			t.Fatalf("wsl command args = %v", args)
		}
		if marksExitSpec(wslSpec) {
			t.Fatal("marksExitSpec(wsl) = true, want false")
		}

		cmdSpec := resolveShell("cmd")
		if cmdSpec.name != "cmd" || cmdSpec.wsl != "" {
			t.Fatalf("resolveShell(cmd) = %+v", cmdSpec)
		}
		if args := shellSpecArgs(cmdSpec, "", "", nil); !slices.Equal(args, []string{"cmd"}) {
			t.Fatalf("cmd args = %v", args)
		}
		psSpec := resolveShell("powershell")
		if psSpec.name != "powershell" {
			t.Fatalf("resolveShell(powershell) = %+v", psSpec)
		}
	}
}

// New sessions and splits in a project with a default shell launch that shell.
func TestProjectDefaultShellInCreateAndSplit(t *testing.T) {
	d := testDaemon(t)
	wantShell := "sh"
	wantPaneShell := "sh"
	if runtime.GOOS == "windows" {
		wantShell = "cmd"
		wantPaneShell = "cmd"
	}
	d.projects[0].Shell = wantShell

	s, err := d.create(proto.CreateParams{Project: "pr"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.removeSession(s.ID) })
	if got := s.Panes[0].Shell; got != wantPaneShell {
		t.Fatalf("created pane shell = %q, want %q", got, wantPaneShell)
	}

	split, err := d.create(proto.CreateParams{Split: s.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(split.Panes) != 2 || split.Panes[1].Shell != wantPaneShell {
		t.Fatalf("split pane shell = %q, want %q", split.Panes[len(split.Panes)-1].Shell, wantPaneShell)
	}
}

// Creating a session with NoDefaults skips the workspace's default folder,
// default command, and default shell fallbacks.
func TestCreateNoDefaultsSkipsWorkspaceDefaults(t *testing.T) {
	d := testDaemon(t)
	projDir := d.projects[0].Dir
	d.projects[0].Command = "echo hello"
	wantShell, otherShell := "sh", "bash"
	if runtime.GOOS == "windows" {
		wantShell, otherShell = "cmd", "powershell"
	}
	d.projects[0].Shell = otherShell

	s, err := d.create(proto.CreateParams{
		Project:    "pr",
		Shell:      wantShell,
		NoDefaults: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.removeSession(s.ID) })
	if s.Project != "pr" {
		t.Fatalf("project = %q, want pr", s.Project)
	}
	if got := s.Panes[0].Shell; got != wantShell {
		t.Fatalf("pane shell = %q, want %q", got, wantShell)
	}
	if got := s.Panes[0].Command; len(got) != 0 {
		t.Fatalf("pane command = %v, want empty", got)
	}
	if got := s.Panes[0].Dir; got == projDir || got != proto.Home() {
		t.Fatalf("pane dir = %q, want home %q (and not project dir %q)", got, proto.Home(), projDir)
	}
}
