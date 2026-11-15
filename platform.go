package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"kero/locales"
)

// fileManager is the system's file manager, as buttons that show a file
// in it name it.
func fileManager() string {
	switch runtime.GOOS {
	case "darwin":
		return locales.FileManagerFinder()
	case "windows":
		return locales.FileManagerExplorer()
	}
	return locales.FileManagerFiles()
}

// openPath opens a file or folder in the app that opens it.
func openPath(path string) { mygo.Shell.OpenPath(path) }

// revealPath shows a file in the file manager.
func revealPath(path string) { mygo.Shell.ShowItemInFolder(path) }

// openTextFile opens a text file, as the configuration, which has no
// extension to tell its app by, in the text editor.
func openTextFile(path string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("/usr/bin/open", "-t", path).Start()
	case "windows":
		exec.Command("notepad.exe", path).Start()
	default:
		exec.Command("xdg-open", path).Start()
	}
}

// ownControls tells that Kero draws the window controls itself: on
// Windows and Linux the top-right card's header holds minimize, maximize
// and close, beside Kero's own buttons, where the system's, at the
// window's edge, would straddle the card. macOS keeps its traffic lights.
var ownControls = runtime.GOOS != "darwin"

// showControls tells that the window controls show (ownControls): where
// the desktop's title bars have buttons (mygo.Theme.WindowButtons), not
// under a tiling compositor, as Hyprland or Sway, which gives windows none,
// or where the user emptied the button layout, as users of tiling window
// managers on X11 do; nor under KWin on Wayland, whose own title bar has
// them. KERO_WINDOW_CONTROLS=1 or 0 shows or hides them all the same.
// followWindowButtons keeps it in step. Main thread only.
var showControls = ownControls

// followWindowButtons sets showControls, and again as the desktop's button
// layout changes. Once the app is ready.
func followWindowButtons() {
	if !ownControls {
		return
	}
	update := func() { showControls = controlsWanted(os.Getenv, mygo.Theme.WindowButtons()) }
	update()
	mygo.Theme.OnUpdated(update)
}

// controlsWanted tells that Kero should draw the window controls, where
// the desktop's title bars have buttons, unless KERO_WINDOW_CONTROLS says
// otherwise.
func controlsWanted(getenv func(string) string, buttons mygo.WindowButtons) bool {
	switch strings.ToLower(strings.TrimSpace(getenv("KERO_WINDOW_CONTROLS"))) {
	case "1", "true", "yes", "on", "show":
		return true
	case "0", "false", "no", "off", "hide":
		return false
	}
	return !buttons.Empty()
}

// framed sets a window's frame: the title bar hidden, under the traffic
// lights on macOS and without its buttons elsewhere (ownControls), which
// keeps the edges that resize the window.
func framed(o mygo.WindowOptions) mygo.WindowOptions {
	if ownControls {
		o.TitleBarStyle = mygo.TitleBarHidden
		o.HideWindowButtons = true
		// No menu bar over the sidebar and the cards: Alt and F10 show the
		// menus, whose shortcuts work all the same.
		o.AutoHideMenuBar = true
		return o
	}
	o.TitleBarStyle = mygo.TitleBarHiddenInset
	o.TitleBarHeight = topBar
	// The lights' buttons are 16 tall, and MyGo puts them a DIP above the
	// position: centered on barMid, as the sidebar's buttons and the
	// panes' headers are.
	o.TrafficLightPosition = &mygo.Point{X: 18, Y: barMid - 7}
	return o
}

var (
	minimizeIcon = icon(`<path d="M3.5 8h9"/>`)
	maximizeIcon = icon(`<rect x="3.5" y="3.5" width="9" height="9" rx="1.5"/>`)
	restoreIcon  = icon(`<rect x="3.5" y="5.5" width="7" height="7" rx="1.2"/><path d="M5.5 3.5h5.5a1.5 1.5 0 0 1 1.5 1.5v5.5"/>`)
)

// controlsW is the window controls' width: three buttons, two apart.
const (
	controlsW   = 3*controlBtnW + 2*2
	controlBtnW = 28
	controlBtnH = 26
	controlTop  = barMid - controlBtnH/2
)

// windowControlTarget is the subset of *mygo.Window needed to render and drive
// the custom title bar window controls.
type windowControlTarget interface {
	Minimize()
	Maximize()
	Unmaximize()
	IsMaximized() bool
	IsFullScreen() bool
	Close()
}

func resolveWindowTarget(w *mygo.Window, fallback windowControlTarget) windowControlTarget {
	if w != nil {
		return w
	}
	return fallback
}

// cornerControls renders the window controls at the top right corner of a window.
// When the window is maximized or in full screen, the close button's hit area
// extends seamlessly to the top-right screen corner (0, width) so Fitts's law
// blind-spot clicks close the window while keeping the visual pill aligned.
func cornerControls(c *ui.Context, pal *palette, key string, w windowControlTarget, edge float32) {
	if !showControls || w == nil {
		return
	}
	maximized := w.IsMaximized()
	cornerClose := maximized || w.IsFullScreen()
	row := ui.Row(c.Key(key)).Absolute().Gap(2).DragWindow()
	var topPad, rightPad float32
	if cornerClose {
		topPad, rightPad = controlTop, edge+10
		row.Right(0).Top(0).AlignItems(ui.Start)
	} else {
		row.Right(edge + 10).Top(controlTop).AlignItems(ui.Center)
	}
	row.Children(func() {
		controlButtons(c, pal, w, maximized, topPad, rightPad)
	})
}

// mainControls are the main window's controls, over its top right corner,
// where they stay as the right sidebar slides in and out under them: in a
// header of the sidebar or of a card, they would slide with it, out of the
// window. edge is the right padding of the cards' column, which comes to
// nothing as the sidebar shows, the controls moving with it; the headers
// under them keep room for them (controlsRoom, rightSidebar).
func (a *App) mainControls(c *ui.Context, pal *palette, edge float32) {
	cornerControls(c, pal, "main-controls", resolveWindowTarget(a.win, a.testWin), edge)
}

// settingsControls renders the Settings window's controls over its top right
// corner so its close button also reaches the top-right blind spot when
// maximized or in full screen.
func (a *App) settingsControls(c *ui.Context, pal *palette) {
	cornerControls(c, pal, "settings-controls", resolveWindowTarget(a.setWin, a.testSetWin), cardGap)
}

// controlsRoom keeps room at the right of the header at the window's top
// right, whose gap is gap, for the window controls over it (mainControls),
// as far as they come over the header: less as the right sidebar comes
// in beside the card, and none once the sidebar is wider than they are.
func (a *App) controlsRoom(c *ui.Context, gap float32) {
	if !showControls || resolveWindowTarget(a.win, a.testWin) == nil {
		return
	}
	// The gap before the room is in it: it narrows to nothing too.
	w := max(controlsW+6+gap-a.side.barW, 0)
	ui.Box(c.Key("controls-room")).Width(w).Shrink(0).Margin(0, 0, 0, -gap)
}

// underControls tells that the pointer is over the window controls, over
// the header of card, w wide, at the window's top right: as they were the
// header's own, they keep its pane hovered, its buttons shown.
func underControls(card ui.Element, w float32) bool {
	if !showControls {
		return false
	}
	x, y, in := card.PointerPosition()
	return in && y < headerH && x >= w-(controlsW+10)
}

// controlButtons are the window controls' buttons.
func controlButtons(c *ui.Context, pal *palette, w windowControlTarget, maximized bool, topPad, rightPad float32) {
	ui.Row(c.Key("window-actions")).Gap(2).AlignItems(ui.Center).Margin(topPad, 0, 0, 0).Children(func() {
		iconButton(c, "minimize", minimizeIcon, locales.WindowMinimize(), pal, w.Minimize)
		// The system's maximize button for the pointer, which brings
		// Windows 11's snap layouts as it rests there: no tooltip over them.
		label, svg := locales.WindowMaximize(), maximizeIcon
		if maximized {
			label, svg = locales.WindowRestore(), restoreIcon
		}
		m := ui.ButtonBase(c.Key("maximize")).Size(controlBtnW, controlBtnH).Radius(6).Center().TextColor(pal.textMuted).Label(label).MaximizeButton()
		if m.Pressed() {
			m.Background(pal.rowHover.Alpha(0.12))
		} else if m.Hovered() {
			m.Background(pal.rowHover).TextColor(pal.text)
		}
		m.Children(func() { ui.Icon(c, svg).Size(16, 16) })
		if m.Clicked() {
			if maximized {
				w.Unmaximize()
			} else {
				w.Maximize()
			}
		}
	})
	b := ui.ButtonBase(c.Key("close-window")).Center().TextColor(pal.textMuted).Label(locales.CommonClose()).Tooltip(locales.CommonClose())
	if topPad > 0 || rightPad > 0 {
		b.Size(controlBtnW+rightPad, controlBtnH+topPad).Padding(topPad, rightPad, 0, 0)
		if b.Hovered() || b.Pressed() {
			b.TextColor(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) {
				p.Fill(ui.Rect{X: r.X, Y: r.Y + topPad, W: controlBtnW, H: controlBtnH}, pal.danger, 6)
			})
		}
	} else {
		b.Size(controlBtnW, controlBtnH).Radius(6)
		if b.Hovered() || b.Pressed() {
			b.Background(pal.danger).TextColor(ui.RGB(255, 255, 255))
		}
	}
	b.Children(func() { ui.Icon(c, closeIcon).Size(16, 16) })
	if b.Clicked() {
		w.Close()
	}
}
