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
const cellMask uint64 = (1 << bitsPerCell) - 1

type ShiftType int

const (
	SHIFT_NONE ShiftType = iota
	SHIFT_FIRST
	SHIFT_TWO
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
	area           [][]uint64
	cellTypes      []*cellType
	cellsPerRow    int
	lastIntMask    uint64
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
		plus[i] = (orig[i] >> bitsPerCell) | (orig[i+1] << (64 - bitsPerCell))
	}
	// wrap lowest int
	plus[nint-1] = (orig[nint-1] >> bitsPerCell) | ((orig[0] & cellMask) << pg.lastCellOffset)

	minus := make([]uint64, nint)
	for i := 1; i < nint; i++ {
		minus[i] = (orig[i] << bitsPerCell) | (orig[i-1] >> (64 - bitsPerCell))
	}
	minus[0] = (orig[0] << bitsPerCell) | ((orig[nint-1] >> pg.lastCellOffset) & cellMask)

	return [][]uint64{orig, minus, plus}
}

// Sumup 8 rows to count the number of young and all (young+old) cells around.
//
// 01,01,01 -> 11 -> 1a,a1
// 01,01,01 -> 11 -> 1b,b1
// 01,01    -> 11 -> 1c,c1
// a1,b1,c1 -> 11 -> 1d,x1
// 1a,1b,1c -> 11. -> 1e.,e1.
// 1d,e1    -> 11. -> 1f.,x1.
// (1e,1f    -> 11..) - not needed as we dont care about exact values of higher bits
// Instead we use OR to combine them.
//
func sumup8(arg [][]uint64) [][]uint64 {
	res := make([][]uint64, 3)
	// young cells - lower bits
	a := sumup3(arg[0], arg[1], &arg[2], SHIFT_NONE)
	b := sumup3(arg[6], arg[7], &arg[8], SHIFT_NONE)
	c := sumup3(arg[4], arg[5], nil, SHIFT_NONE)
	d := sumup3(a, b, &c, SHIFT_NONE) // bit0
	e := sumup3(a, b, &c, SHIFT_ALL)
	f := sumup3(d, e, nil, SHIFT_FIRST) // bit1
	res[0] = d
	res[1] = f
	res[2] = bitor3(e, f, nil, SHIFT_ALL)
	// total cells
	a = sumup3(arg[0], arg[1], &arg[2], SHIFT_ALL)
	b = sumup3(arg[6], arg[7], &arg[8], SHIFT_ALL)
	c = sumup3(arg[4], arg[5], &res[0], SHIFT_TWO)
	d = sumup3(a, b, &c, SHIFT_NONE) // bit0
	e = sumup3(a, b, &c, SHIFT_ALL)
	f = sumup3(d, e, &res[1], SHIFT_FIRST) // bit1
	res[0] = join2(res[0], d)
	res[1] = join2(res[1], f)
	res[2] = join2(res[2], bitor3(e, f, &res[2], SHIFT_TWO))
	return res
}

func join2(x, y []uint64) []uint64 {
	nint := len(x)
	res := make([]uint64, nint)
	for i := 0; i < nint; i++ {
		res[i] = (x[i] & lowBits64) | ((y[i] & lowBits64) << 1)
	}
	return res
}

func div2(x []uint64) []uint64 {
	nint := len(x)
	res := make([]uint64, nint)
	for i := 0; i < nint; i++ {
		res[i] = x[i] >> 1
	}
	return res
}

var shifts = [][]uint{
	{0, 0, 0},
	{1, 0, 0},
	{1, 1, 0},
	{1, 1, 1},
}

func sumup3(x, y []uint64, z *[]uint64, shift ShiftType) []uint64 {
	nint := len(x)
	res := make([]uint64, nint)
	as := shifts[int(shift)][0]
	bs := shifts[int(shift)][1]
	cs := shifts[int(shift)][2]
	if z == nil {
		// we use a trick - res is initialized with 0.
		// so if len(z) == 0, then we will use res as a source for z
		z = &res
	}
	for i := 0; i < nint; i++ {
		a := (x[i] >> as) & lowBits64
		b := (y[i] >> bs) & lowBits64
		c := ((*z)[i] >> cs) & lowBits64
		res[i] = a + b + c
	}
	return res
}

func bitor3(x, y []uint64, z *[]uint64, shift ShiftType) []uint64 {
	nint := len(x)
	res := make([]uint64, nint)
	as := shifts[shift][0]
	bs := shifts[shift][1]
	cs := shifts[shift][2]
	if z == nil {
		z = &res
	}
	for i := 0; i < nint; i++ {
		res[i] = ((x[i] >> as) | (y[i] >> bs) | ((*z)[i] >> cs)) & lowBits64
	}
	return res
}

func (pg *Playground) Step() {
	fmt.Printf("step %p\n", pg)
	nrows := len(pg.area)
	next := make([][]uint64, nrows) // the next state of the area
	roll := make([][]uint64, 9)     // working area
	first := pg.tripleRow(0)
	last := pg.tripleRow(nrows - 1)
	copy(roll[3:6], last)
	copy(roll[6:9], first)
	for iy := 0; iy < nrows; iy++ {
		// shift all rows
		copy(roll[0:3], roll[3:6])
		copy(roll[3:6], roll[6:9])
		// fill the next row
		idx := iy + 1
		if idx < nrows {
			copy(roll[6:9], pg.tripleRow(idx))
		} else {
			copy(roll[6:9], first)
		}
		// now sumup all young and total number of adjacent cells.
		// counts has three arrays with T (total) and Y (young) bits
		// for every cell.  bit0 (TY), bit1 (TY) and bit2+ (TY).
		counts := sumup8(roll)
		// rules are:
		// 1. each young cell converts to old.
		// 2. an empty cell converts to young cell if Y<2 and T=3, otherwise is empty
		// 3. an old cell remains live if Y<2 and T=[2..3], otherwise is empty
		nint := len(pg.area[iy])
		next[iy] = make([]uint64, nint)
		for ix := 0; ix < nint; ix++ {
			orig := pg.area[iy][ix]
			bit0 := counts[0][ix]
			bit1 := counts[1][ix]
			not2 := ^counts[2][ix]
			//       empty young old
			// orig:  00,   01,   10
			// bit0:  1.    ..    ..
			// bit1:  10    ..    10
			// not2:  11    ..    11

			// out:   01    10    10

			// common mask for empty->young and old->old
			mask := not2 & (not2 >> 1) & ^bit1 & (bit1 >> 1)
			noto := ^orig
			newyng := noto & (noto >> 1) & (bit0 >> 1) & mask
			newold := (orig>>1)&mask | orig
			next[iy][ix] = (newyng & lowBits64) | ((newold & lowBits64) << 1)
		}
		next[iy][nint-1] &= pg.lastIntMask
	}
	pg.area = next
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
	extraCells := rowLen*cellsPerInt - nx
	// the mask of the last int in the row
	pg.lastIntMask = ^uint64(0) >> uint(extraCells*bitsPerCell)
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

func (pg *Playground) Clean() {
	for iy := 0; iy < len(pg.area); iy++ {
		for ix := 0; ix < len(pg.area[iy]); ix++ {
			pg.area[iy][ix] = 0
		}
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
	fmt.Printf("key: val:%d state:%d type:%v\n", ev.KeyVal(), ev.State(), ev.Type())
	switch ev.KeyVal() {
	case gdk.KEY_Escape:
		gtk.MainQuit()
	case gdk.KEY_space:
		pg.Step()
		win.QueueDraw()
	case gdk.KEY_C:
		pg.Clean()
		win.QueueDraw()
	}
}

func showbin(v uint64) string {
	var r []byte
	for i := 0; i < cellsPerInt; i++ {
		var c byte
		switch v & cellMask {
		case 0:
			c = '.'
		case 1:
			c = 'o'
		case 2:
			c = 'X'
		case 3:
			c = '?'
		}
		r = append(r, c)
		v >>= bitsPerCell
	}
	return string(r)
}

func mouseClicked(win *gtk.Window, evt *gdk.Event, pg *Playground) bool {
	ev := gdk.EventButton{evt}
	ix := int(ev.X() / (cellSize + gapSize))
	iy := int(ev.Y() / (cellSize + gapSize))
	idx := ix / cellsPerInt
	v := pg.area[iy][idx]
	mask := cellMask << uint(bitsPerCell*(ix%cellsPerInt))
	v0 := ^v & mask
	nv := (((v0 & (v0 >> 1)) | (v&mask)<<1) & mask) | (v & ^mask)
	fmt.Printf("mouse: btn:%d bnt-val:%d state:%d type:%v ix,iy,idx,i:%d,%d,%d,%d\n",
		ev.Button(), ev.ButtonVal(),
		ev.State(), ev.Type(),
		ix, iy, idx, ix%cellsPerInt)
	fmt.Printf("old: %s\n", showbin(v))
	fmt.Printf("msk: %s\n", showbin(mask))
	fmt.Printf("new: %s\n", showbin(nv))
	pg.area[iy][idx] = nv
	win.QueueDraw()
	return true
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

	if _, err = win.Connect("button-press-event", mouseClicked, playground); err != nil {
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

	if err := setupWindow(playground); err != nil {
		fail(err)
	}
	gtk.Main()
}
