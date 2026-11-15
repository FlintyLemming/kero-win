package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type fakeWindowControls struct {
	maximized  bool
	fullScreen bool
	minimized  int
	closed     int
}

func (f *fakeWindowControls) Minimize()          { f.minimized++ }
func (f *fakeWindowControls) Maximize()          { f.maximized = true }
func (f *fakeWindowControls) Unmaximize()        { f.maximized = false }
func (f *fakeWindowControls) IsMaximized() bool  { return f.maximized }
func (f *fakeWindowControls) IsFullScreen() bool { return f.fullScreen }
func (f *fakeWindowControls) Close()             { f.closed++ }

// Kero draws the window controls where the desktop's title bars have
// buttons, none under a tiling window manager, and the environment
// variable has the last word.
func TestControlsWanted(t *testing.T) {
	desktop := mygo.WindowButtons{End: []string{"minimize", "maximize", "close"}}
	gnome := mygo.WindowButtons{End: []string{"close"}}
	tiling := mygo.WindowButtons{}
	for _, tc := range []struct {
		env     string
		buttons mygo.WindowButtons
		want    bool
	}{
		{"", desktop, true},
		{"", gnome, true},
		{"", tiling, false},
		{"1", tiling, true},
		{"show", tiling, true},
		{"0", desktop, false},
		{"off", desktop, false},
		{"what", tiling, false},
	} {
		getenv := func(k string) string {
			if k == "KERO_WINDOW_CONTROLS" {
				return tc.env
			}
			return ""
		}
		if got := controlsWanted(getenv, tc.buttons); got != tc.want {
			t.Errorf("controlsWanted(KERO_WINDOW_CONTROLS=%q, %+v) = %v, want %v", tc.env, tc.buttons, got, tc.want)
		}
	}
}

func TestCornerCloseButton(t *testing.T) {
	prev := showControls
	showControls = true
	t.Cleanup(func() { showControls = prev })

	const winW, winH = 1100, 640
	cornerX, cornerY := float32(winW-1), float32(0)

	clickAt := func(tt *ui.Tester, x, y float32) {
		tt.Press(x, y)
		tt.Frame()
		tt.Release(x, y)
		tt.Frame()
	}

	for _, topTabs := range []bool{false, true} {
		for _, rightOpen := range []bool{false, true} {
			name := fmt.Sprintf("topTabs=%t/right=%t", topTabs, rightOpen)
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				a.set.TopTabs = topTabs
				if rightOpen {
					a.prefs.Side = sideChanges
				}
				w := &fakeWindowControls{}
				a.testWin = w

				tt := ui.NewTester(a.view, winW, winH)
				tt.Frame()
				tt.Advance(time.Second)
				tt.Frame()

				edge := float32(cardGap)
				if rightOpen {
					edge = 0
				}
				pillX := float32(winW) - (edge + 10 + controlBtnW/2)
				pillY := float32(controlTop + controlBtnH/2)

				// Restored window: clicking the extreme top-right blind spot must not close.
				clickAt(tt, cornerX, cornerY)
				if w.closed != 0 {
					t.Fatalf("restored window closed on top-right blind spot click")
				}

				// Restored window: clicking the visible close button pill closes.
				clickAt(tt, pillX, pillY)
				if w.closed != 1 {
					t.Fatalf("restored window close count = %d, want 1", w.closed)
				}

				// Maximized window: hovering the extreme top-right pixel highlights the pill
				// and clicking closes the window.
				w.closed = 0
				w.maximized = true
				tt.Move(100, 100)
				tt.Frame()

				restPill := tt.Image().RGBAAt(int(pillX-10), int(pillY-8))
				tt.Move(cornerX, cornerY)
				tt.Frame()
				hoverPill := tt.Image().RGBAAt(int(pillX-10), int(pillY-8))
				if hoverPill == restPill {
					t.Fatalf("hovering top-right blind spot did not highlight close pill (color stayed %+v)", restPill)
				}
				// Ensure hover highlight stays within the rounded pill and does not bleed into the outer corner.
				if cornerPx := tt.Image().RGBAAt(int(cornerX), int(cornerY)); cornerPx == hoverPill {
					t.Fatalf("close hover color bled into outer top-right corner (%+v)", cornerPx)
				}

				clickAt(tt, cornerX, cornerY)
				if w.closed != 1 {
					t.Fatalf("maximized window did not close on top-right blind spot click (closed=%d)", w.closed)
				}

				// Pressing at the corner and drifting onto the visible icon still closes.
				tt.Press(cornerX, cornerY)
				tt.Frame()
				tt.Release(pillX, pillY)
				tt.Frame()
				if w.closed != 2 {
					t.Fatalf("press-at-corner release-on-icon close count = %d, want 2", w.closed)
				}

				// Double-clicking the top-right corner closes rather than unmaximizing.
				clickAt(tt, cornerX, cornerY)
				clickAt(tt, cornerX, cornerY)
				if w.closed < 3 || !w.maximized {
					t.Fatalf("double-click at corner: closed=%d maximized=%t", w.closed, w.maximized)
				}

				// Clicking the gap above the maximize button does not close.
				aboveMaxX := float32(winW) - (edge + 10 + controlBtnW + 2 + controlBtnW/2)
				clickAt(tt, aboveMaxX, 2)
				if w.closed != 4 {
					t.Fatalf("clicking above maximize closed window (closed=%d)", w.closed)
				}

				// Minimize and restore buttons still work in their original slots.
				clickAt(tt, aboveMaxX, pillY)
				if w.maximized {
					t.Fatal("maximize/restore button did not unmaximize")
				}
				minX := aboveMaxX - (controlBtnW + 2)
				clickAt(tt, minX, pillY)
				if w.minimized != 1 {
					t.Fatalf("minimize count = %d, want 1", w.minimized)
				}

				// Full-screen window: top-right blind spot also closes.
				w.closed = 0
				w.fullScreen = true
				tt.Frame()
				clickAt(tt, cornerX, cornerY)
				if w.closed != 1 {
					t.Fatalf("fullscreen window did not close on top-right blind spot click (closed=%d)", w.closed)
				}
			})
		}
	}
}

func TestSettingsCornerCloseButton(t *testing.T) {
	prev := showControls
	showControls = true
	t.Cleanup(func() { showControls = prev })

	a := testApp(t)
	w := &fakeWindowControls{}
	a.testSetWin = w

	const winW, winH = 820, 640
	tt := ui.NewTester(a.settingsView, winW, winH)
	tt.Frame()

	cornerX, cornerY := float32(winW-1), float32(0)
	clickAt := func(x, y float32) {
		tt.Press(x, y)
		tt.Frame()
		tt.Release(x, y)
		tt.Frame()
	}

	// Restored settings window: corner blind spot does not close.
	clickAt(cornerX, cornerY)
	if w.closed != 0 {
		t.Fatalf("restored settings window closed on top-right blind spot click")
	}

	// Maximized settings window: corner blind spot closes.
	w.maximized = true
	tt.Frame()
	clickAt(cornerX, cornerY)
	if w.closed != 1 {
		t.Fatalf("maximized settings window did not close on top-right blind spot click")
	}

	// Fullscreen settings window: corner blind spot closes.
	w.maximized = false
	w.fullScreen = true
	tt.Frame()
	clickAt(cornerX, cornerY)
	if w.closed != 2 {
		t.Fatalf("fullscreen settings window did not close on top-right blind spot click")
	}
}
