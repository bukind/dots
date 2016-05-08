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
var pattern0 uint64 = 0x1818181818181818
var pattern1 uint64 = 0x6060606060606060

const lowBits64 uint64 = 0x5555555555555555
const cellSize = 12
const gapSize = 4

const totalStates = 3 // empty, young, old
const bitsPerCell = 2
const cellsPerInt = 64 / bitsPerCell
const cellMask = (1 << bitsPerCell) - 1

type ShiftType int

const (
    SHIFT_NONE  ShiftType = iota
    SHIFT_FIRST
    SHIFT_ALL
)

var colorNames = []string{"white", "lightgreen", "blue"}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Failure: %v", err)
	os.Exit(1)
}

type cellType struct {
	color    *gdk.RGBA
	cellMask uint64
}

type Playground struct {
	area        [][]uint64
	cellTypes   []*cellType
	cellsPerRow int
	lastIntMask uint64
	lastCellOffset uint
}

func NewPlayground() *Playground {
	return new(Playground)
}

//     01 01 01 01 prev
//     >> 01 01 01 01 prev+   -> 11 11 11 11
//  01 01 01 01 << prev-
//
//     01 01 01 01 this - ignored
//     >> 01 01 01 01 this+   -> 10 10 10 10
//  01 01 01 01 << this-
//
//     01 01 01 01 next       -> 11 11 11 11
//     >> 01 01 01 01 next+
//  01 01 01 01 << next-


// Returns three rows: normal, shifed left, shifted right.
func (pg *Playground) tripleRow(iy int) [][]uint64 {
	orig := pg.area[iy]
	nint := len(orig)
	plus := make([]uint64, nint)
	for i := 0; i < nint-1; i++ {
	    plus[i] = (orig[i] >> bitsPerCell) | (orig[i+1] << (64-bitsPerCell))
	}
	// wrap lowest int
	plus[nint-1] = (orig[nint-1] >> bitsPerCell) | ((orig[0] & cellMask) << pg.lastCellOffset)

	minus := make([]uint64, nint)
	for i := 1; i < nint; i++ {
	    minus[i] = (orig[i] << bitsPerCell) | (orig[i-1] >> (64-bitsPerCell))
	}
	minus[0] = (orig[0] << bitsPerCell) | ((orig[nint-1] >> pg.lastCellOffset) & cellMask)

	return [][]uint64{orig, minus, plus}
}

// 01,01,01 -> 11 -> 1a,a1
// 01,01,01 -> 11 -> 1b,b1
// 01,01    -> 11 -> 1c,c1
// a1,b1,c1 -> 11 -> 1d,x1
// 1a,1b,1c -> 11. -> 1e.,e1.
// 1d,e1    -> 11. -> 1f.,x1.
// 1e,1f    -> 11..
// Sumup 8 rows to count the number of bits.
func (pg *Playground) sumup8(arg [][]uint64) [][]uint64 {
    res := make([][]uint64, 4)
    // young cells - lower bits
	var nul []uint64
    a := pg.sumup3(arg[0], arg[1], arg[2], SHIFT_NONE)
	b := pg.sumup3(arg[6], arg[7], arg[8], SHIFT_NONE)
	c := pg.sumup3(arg[4], arg[5], nul, SHIFT_NONE)
	d := pg.sumup3(a, b, c, SHIFT_NONE) // bit0
	e := pg.sumup3(a, b, c, SHIFT_ALL)
	f := pg.sumup3(d, e, nul, SHIFT_FIRST) // bit1
	g := pg.sumup3(e, f, nul, SHIFT_ALL) // bits2,3
	res[0] = pg.join2(d, f)
	res[1] = g
	// older cells - higher bits
	a = pg.sumup3(arg[0], arg[1], arg[2], SHIFT_ALL)
	b = pg.sumup3(arg[6], arg[7], arg[8], SHIFT_ALL)
	c = pg.sumup3(arg[4], arg[5], nul, SHIFT_ALL)
	d = pg.sumup3(a, b, c, SHIFT_NONE) // bit0
	e = pg.sumup3(a, b, c, SHIFT_ALL)
	f = pg.sumup3(d, e, nul, SHIFT_FIRST) // bit1
	g = pg.sumup3(e, f, nul, SHIFT_ALL) // bit2,3
	res[2] = pg.join2(d, f)
	res[3] = g
	return res
}

func (pg *Playground) join2(x, y []uint64) []uint64 {
    nint := len(x)
    res := make([]uint64, nint)
	for i := 0; i < nint; i++ {
	    res[i] = (x[i] & lowBits64) | ((y[i] & lowBits64) << 1)
	}
	return res
}

func (pg *Playground) sumup3(x, y, z []uint64, shift ShiftType) []uint64 {
    nint := len(x)
	res := make([]uint64,nint)
	for i := 0; i < nint; i++ {
	    a := x[i]
		b := y[i]
		c := uint64(0)
		if len(z) > 0 {
		    c = z[i]
		}
		if shift == SHIFT_ALL {
		    a >>= 1
			b >>= 1
			c >>= 1
		} else if shift == SHIFT_FIRST {
		    a >>= 1
		}
		a &= lowBits64
		b &= lowBits64
		c &= lowBits64
		res[i] = a + b + c
	}
	return res
}

func (pg *Playground) Step() {
	fmt.Printf("step %p\n", pg)
	nrows := len(pg.area)
	next := make([][]uint64, nrows) // the next state of the area
	roll := make([][]uint64, 9) // working area
	first := pg.tripleRow(0)
	last := pg.tripleRow(nrows-1)
	copy(roll[3:6],last)
	copy(roll[6:9],first)
	for iy := 0; iy < nrows; iy++ {
	    // shift all rows
		copy(roll[0:3],roll[3:6])
		copy(roll[3:6],roll[6:9])
		// fill the next row
		idx := iy + 1
		if idx < nrows {
		    copy(roll[6:9],pg.tripleRow(idx))
		} else {
		    copy(roll[6:9],first)
		}
		// now sumup all rows
		bits := pg.sumup8(roll)
		_ = bits
	}
	_ = next
	fmt.Println("step done\n")
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
	extraCells := rowLen * cellsPerInt - nx
	// the mask of the last int in the row
	pg.lastIntMask = ^uint64(0) >> uint(extraCells * bitsPerCell)
	// the offset the the last cell in the last int
	pg.lastCellOffset = uint((extraCells - 1) * bitsPerCell)
	for i := 0; i < wy; i++ {
		row := make([]uint64, rowLen)
		pattern := pattern0
		if i%2 == 1 {
			pattern = pattern1
		}
		for j := 0; j < rowLen; j++ {
			row[j] = pattern
		}
		row[rowLen-1] &= pg.lastIntMask
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
		ct.cellMask = uint64(i)
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
		// fmt.Printf("row #%d nx:%d, ct:%d\n", iy, len(row), len(pg.cellTypes))
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
					if value & cellMask == cellType.cellMask {
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
	case gdk.KEY_space:
		pg.Step()
		win.QueueDraw()
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

	flag.Uint64Var(&pattern0, "pattern0", pattern0, "Set the initial pattern #0")
	flag.Uint64Var(&pattern1, "pattern1", pattern1, "Set the initial pattern #1")
	flag.IntVar(&xsize, "xsize", xsize, "Set the X size, or -1")
	flag.IntVar(&ysize, "ysize", ysize, "Set the Y size, or -1")

	flag.Parse()

	if xsize == -1 || ysize == -1 {
		fullscreen = true
	}

	// check pattern0, pattern1
	p0 := pattern0
	p1 := pattern1
	for i := 0; i < cellsPerInt; i++ {
		if (p0&cellMask) >= totalStates || (p1&cellMask) >= totalStates {
			panic("Bad pattern")
		}
		p0 >>= bitsPerCell
		p1 >>= bitsPerCell
	}

	gtk.Init(nil)

	playground := NewPlayground()
	playground.Step()

	if err := setupWindow(playground); err != nil {
		fail(err)
	}
	gtk.Main()
}
