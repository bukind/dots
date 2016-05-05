package main

import (
	"flag"
	"fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"os"
)

var fullscreen bool = false
var xsize int = 400
var ysize int = 400
var pattern0 int64 = 0x1b1b1b1b1b1b1b1b
var pattern1 int64 = 0x6c6c6c6c6c6c6c6c

const cellSize = 5
const gapSize = 0
const bitsPerCell = 2
const cellsPerInt = 64 / bitsPerCell
const cellMask = (1 << bitsPerCell) - 1

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Failure: %v", err)
	os.Exit(1)
}

type cellType struct {
	r, g, b  float64 // rgb
	cellMask int64
}

type Playground struct {
	area      [][]int64
	cellTypes []cellType
}

func NewPlayground() *Playground {
	return new(Playground)
}

func (pg *Playground) Step() {
	fmt.Printf("step %p\n", pg)
}

func (pg *Playground) Init(wx, wy int) {
	nx := wx / (cellSize + gapSize)
	if nx <= 0 {
		panic("Too narrow area")
	}
	ny := wy / (cellSize + gapSize)
	if ny <= 0 {
		panic("Too short area")
	}
	rowLen := (nx + cellsPerInt - 1) / cellsPerInt
	for i := 0; i < wy; i++ {
		row := make([]int64, rowLen)
		pattern := pattern0
		if i%2 == 1 {
			pattern = pattern1
		}
		for j := 0; j < rowLen; j++ {
			row[j] = pattern
		}
		pg.area = append(pg.area, row)
	}
}

func areaSetup(da *gtk.DrawingArea, ev *gdk.Event, pg *Playground) {
	wx := da.GetAllocatedWidth()
	wy := da.GetAllocatedHeight()
	pg.Init(wx, wy)
	_ = ev
}

func areaDraw(da *gtk.DrawingArea, cr *cairo.Context, pg *Playground) {
	_ = da
	_ = cr
	dx := float64(cellSize + gapSize)
	for iy, row := range pg.area {
		y := float64(iy) * dx
		for _, cellType := range pg.cellTypes {
			cr.SetSourceRGB(cellType.r, cellType.g, cellType.b)
			for ix, value := range row {
				for idx := 0; idx < cellsPerInt; idx++ {
					if value&cellMask == cellType.cellMask {
						cr.Rectangle(dx*float64(ix+idx), y, cellSize, cellSize)
					}
					value >>= bitsPerCell
				}
			}
			cr.Fill()
		}
	}
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
		win.SetSizeRequest(xsize, ysize)
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

	flag.Int64Var(&pattern0, "pattern0", pattern0, "Set the initial pattern #0")
	flag.Int64Var(&pattern1, "pattern1", pattern1, "Set the initial pattern #1")
	flag.IntVar(&xsize, "xsize", xsize, "Set the X size, or -1")
	flag.IntVar(&ysize, "ysize", ysize, "Set the Y size, or -1")

	flag.Parse()

	if xsize == -1 || ysize == -1 {
		fullscreen = true
	}

	gtk.Init(nil)

	playground := NewPlayground()
	playground.Step()

	if err := setupWindow(playground); err != nil {
		fail(err)
	}
	gtk.Main()
}
