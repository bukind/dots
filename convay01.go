package main

import (
    "fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"os"
)

const fullscreen bool = false
const winSizeX = 400
const winSizeY = 400


func fail(err error) {
    fmt.Fprintf(os.Stderr, "Failure: %v", err)
	os.Exit(1)
}


type Playground struct {
}


func NewPlayground() *Playground {
    return new(Playground)
}


func (pg *Playground) Step() {
    fmt.Printf("step %p\n", pg)
}


func areaSetup(da *gtk.DrawingArea, ev *gdk.Event, pg *Playground) {
    _ = da
	_ = ev
	_ = pg
}


func areaDraw(da *gtk.DrawingArea, cr *cairo.Context, pg *Playground) {
    _ = da
	_ = cr
	_ = pg
	pg.Step()
}


func winKeyPress(win *gtk.Window, evt *gdk.Event, pg *Playground) {
    _ = win
	ev := gdk.EventKey{evt}
	switch ev.KeyVal() {
	case gdk.KEY_Escape:
		gtk.MainQuit()
	}
}


func setupWindow(playground *Playground) error {

	var win *gtk.Window
	var err error

	if win, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL); err != nil {
	    return err
	}

    win.SetTitle("dots")
	win.Connect("destroy", gtk.MainQuit)
	if fullscreen {
	    win.Fullscreen()
	} else {
	    win.SetSizeRequest(winSizeX, winSizeY)
	}
	win.SetResizable(false)

	var da *gtk.DrawingArea
	if da, err = gtk.DrawingAreaNew(); err != nil {
	    return err
	}

	win.Add(da)
	win.ShowAll()

	if _, err = da.Connect("configure-event", areaSetup, playground); err != nil {
	    return err
	}

	if _, err = da.Connect("draw", areaDraw, playground); err != nil {
	    return err
	}

	if _, err = win.Connect("key-press-event", winKeyPress, playground); err != nil {
	    return err
	}

	return nil
}


func main() {
    gtk.Init(nil)

	playground := NewPlayground()
	playground.Step()

	if err := setupWindow(playground); err != nil {
	    fail(err)
	}
	gtk.Main()
}
