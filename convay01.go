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
var pattern0 int64 = 0x1818181818181818
var pattern1 int64 = 0x6060606060606060

const cellSize = 12
const gapSize = 4

var colorNames []string
var bitsPerCell uint
var cellsPerInt int
var cellMask int64

const totalStates = 3 // empty, young, old

func init() {
	bitsPerCell = 1
	for i := 1; i < totalStates-1; i <<= 1 {
		bitsPerCell++
	}
	cellMask = (int64(1) << bitsPerCell) - 1
	cellsPerInt = 64 / int(bitsPerCell)
	colorNames = []string{"white", "lightgreen", "blue", "white",
		"white", "white", "white", "white"}[0:totalStates]
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Failure: %v", err)
	os.Exit(1)
}

type cellType struct {
	color    *gdk.RGBA
	cellMask int64
}

type Playground struct {
	area        [][]int64
	cellTypes   []*cellType
	cellsPerRow int
}

func NewPlayground() *Playground {
	return new(Playground)
}

func (pg *Playground) Step() {
	fmt.Printf("step %p\n", pg)
}

func (pg *Playground) Init(da *gtk.DrawingArea) {
	fmt.Println("configure-event")
	wx := da.GetAllocatedWidth()
	wy := da.GetAllocatedHeight()

	nx := wx / (cellSize + gapSize)
	if nx <= 0 {
		panic("Too narrow area")
	}
	ny := wy / (cellSize + gapSize)
	if ny <= 0 {
		panic("Too short area")
	}
	rowLen := (nx + cellsPerInt - 1) / cellsPerInt
	pg.cellsPerRow = nx
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
	// define cell types
	pg.cellTypes = make([]*cellType, totalStates)
	for i := 0; i < len(pg.cellTypes); i++ {
		ct := new(cellType)
		pg.cellTypes[i] = ct
		ct.color = gdk.NewRGBA()
		if !ct.color.Parse(colorNames[i]) {
			panic("failed to parse color name")
		}
		ct.cellMask = int64(i)
	}
}

func areaSetup(da *gtk.DrawingArea, ev *gdk.Event, pg *Playground) {
	_ = ev
	pg.Init(da)
}

func areaDraw(da *gtk.DrawingArea, cr *cairo.Context, pg *Playground) {
	_ = cr
	if pg.area == nil {
		pg.Init(da)
	}
	fmt.Printf("draw totx:%d\n", pg.cellsPerRow)
	dx := float64(cellSize + gapSize)
	for iy, row := range pg.area {
		fmt.Printf("row #%d nx:%d, ct:%d\n", iy, len(row), len(pg.cellTypes))
		y := float64(iy) * dx
		for _, cellType := range pg.cellTypes {
			rgba := cellType.color.Floats()
			cr.SetSourceRGB(rgba[0], rgba[1], rgba[2])
			for ix, value := range row {
				maxIdx := ix + cellsPerInt
				if maxIdx > pg.cellsPerRow {
					maxIdx = pg.cellsPerRow
				}
				for idx := ix; idx < maxIdx; idx++ {
					if value&cellMask == cellType.cellMask {
						// fmt.Printf("x:%d, y:%d, rgb:%.1f/%.1f/%.1f\n", idx, iy,
						//           rgba[0], rgba[1], rgba[2])
						cr.Rectangle(dx*float64(idx), y, cellSize, cellSize)
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
