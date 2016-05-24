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
var pattern0 uint64 = 0x0
var pattern1 uint64 = 0x0
var initialConfig = ""

const lowBits64 uint64 = 0x5555555555555555

const totalStates = 3 // empty, young, old
const bitsPerCell = 4
const cellsPerInt = 64 / bitsPerCell
const cellMask uint64 = (1 << bitsPerCell) - 1

type ShiftType int

const (
	SHIFT_NONE ShiftType = iota
	SHIFT_FIRST
	SHIFT_TWO
	SHIFT_ALL
)

var colorNames = []string{"white", "red", "blue"}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Failure: %v", err)
	os.Exit(1)
}

type cellType struct {
	color    *gdk.RGBA
	cellMask uint64
}

type Playground struct {
	da             *gtk.DrawingArea
	cellSize       uint
	gapSize        uint
	area           [][]uint64
	cellTypes      []*cellType
	cellsPerRow    int
	lastIntMask    uint64
	lastCellOffset uint
	iterations     int
}

func NewPlayground(cellSize, gapSize uint) *Playground {
	pg := new(Playground)
	pg.cellSize = cellSize
	pg.gapSize = gapSize
	return pg
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
	// fmt.Printf("step %p\n", pg)
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
	// fmt.Println("step done\n")
}

func (pg *Playground) Init(da *gtk.DrawingArea, nx, ny int) {
	fmt.Println("configure-event")

	if nx <= 0 {
		panic("Too narrow area")
	}
	if ny <= 0 {
		panic("Too short area")
	}

	if initialConfig == "" {
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
	}

	rowLen := (nx + cellsPerInt - 1) / cellsPerInt
	pg.da = da
	pg.cellsPerRow = nx
	lastIntCells := nx - cellsPerInt*(rowLen-1)
	if lastIntCells <= 0 {
		panic("Invalid lastIntCells")
	}
	// the mask of the last int in the row
	pg.lastIntMask = ^(^uint64(0) << uint(lastIntCells*bitsPerCell))
	// the offset the the last cell in the last int
	pg.lastCellOffset = uint((lastIntCells - 1) * bitsPerCell)
	for i := 0; i < ny; i++ {
		row := make([]uint64, rowLen)
		if initialConfig == "" {
			pattern := pattern0
			if i%2 == 1 {
				pattern = pattern1
			}
			for j := 0; j < rowLen; j++ {
				row[j] = pattern
			}
			row[rowLen-1] &= pg.lastIntMask
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
		ct.cellMask = uint64(i)
	}
	pg.iterations = 0

	switch initialConfig {
	case "line":
		pg.setDots(ny/2, nx/2-3, "1222221")
	case "kaka":
		pg.setDots(ny/2+0, nx/2, "000000012")
		pg.setDots(ny/2+1, nx/2, "2100010021")
		pg.setDots(ny/2+2, nx/2, "0020210021")
		pg.setDots(ny/2+3, nx/2, "222002122")
		pg.setDots(ny/2+4, nx/2, "0110101")
	case "":
		// do nothing
	default:
		pg.setDots(ny/2, nx/2, "221")
		pg.setDots(ny/2+1, nx/2, "002")
		pg.setDots(ny/2+2, nx/2, "2")
	}
}

func (pg *Playground) setDots(y, x int, dots string) {
	for ; y < 0; y += len(pg.area) {
	}
	for ; y >= len(pg.area); y -= len(pg.area) {
	}
	for ; x < 0; x += pg.cellsPerRow {
	}
	for ; x >= pg.cellsPerRow; x -= pg.cellsPerRow {
	}
	// TODO(bukind): optimize the loop
	for i := 0; i < len(dots); i++ {
		ix := x / cellsPerInt
		shift := uint((x - ix*cellsPerInt) * bitsPerCell)
		var v uint64
		switch dots[i] {
		case '0':
			v = 0
		case '1':
			v = 1
		case '2':
			v = 2
		}
		nv := pg.area[y][ix] & ^(cellMask<<shift) + ((1 << shift) * v)
		pg.area[y][ix] = nv
		x++
	}
}

func (pg *Playground) Clean() {
	for iy := 0; iy < len(pg.area); iy++ {
		for ix := 0; ix < len(pg.area[iy]); ix++ {
			pg.area[iy][ix] = 0
		}
	}
}

func (pg *Playground) StepAndDraw() {
	if pg.iterations > 0 {
		pg.iterations--
		pg.Step()
		pg.da.QueueDraw()
	} else if pg.iterations == -1 {
		pg.Step()
		pg.da.QueueDraw()
	}
}

func (pg *Playground) ShowAll() {
	for iy := 0; iy < len(pg.area); iy++ {
		for ix := 0; ix < len(pg.area[iy]); ix++ {
			showbin(pg.area[iy][ix])
		}
	}
}

func areaSetup(da *gtk.DrawingArea, ev *gdk.Event, pg *Playground) {
	_ = ev
	nx := da.GetAllocatedWidth() / int(pg.cellSize+pg.gapSize)
	ny := da.GetAllocatedHeight() / int(pg.cellSize+pg.gapSize)
	pg.Init(da, nx, ny)
}

func areaDraw(da *gtk.DrawingArea, cr *cairo.Context, pg *Playground) {
	_ = cr
	if pg.area == nil {
		areaSetup(da, nil, pg)
	}
	// fmt.Printf("draw totx:%d\n", pg.cellsPerRow)
	dx := float64(pg.cellSize + pg.gapSize)
	cs := float64(pg.cellSize)
	for iy, row := range pg.area {
		// fmt.Printf("row #%d nx:%d, ct:%d\n", iy, len(row), len(pg.cellTypes))
		y := float64(iy) * dx
		for _, cellType := range pg.cellTypes {
			rgba := cellType.color.Floats()
			cr.SetSourceRGB(rgba[0], rgba[1], rgba[2])
			for ix, value := range row {
				idx0 := ix * cellsPerInt
				maxIdx := idx0 + cellsPerInt
				if maxIdx > pg.cellsPerRow {
					maxIdx = pg.cellsPerRow
				}
				for idx := idx0; idx < maxIdx; idx++ {
					if value&cellMask == cellType.cellMask {
						// fmt.Printf("x:%d, y:%d, rgb:%.1f/%.1f/%.1f\n", idx, iy,
						//           rgba[0], rgba[1], rgba[2])
						cr.Rectangle(dx*float64(idx), y, cs, cs)
					}
					value >>= bitsPerCell
				}
			}
			cr.Fill()
		}
	}
	if pg.iterations != 0 {
		pg.StepAndDraw()
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
		pg.da.QueueDraw()
	case gdk.KEY_C:
		pg.Clean()
		pg.da.QueueDraw()
	case gdk.KEY_t:
		pg.iterations += 10
		pg.StepAndDraw()
	case gdk.KEY_x:
		pg.iterations = 0
	case gdk.KEY_s:
		pg.iterations = -1
		pg.StepAndDraw()
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
		default:
			c = '$'
		}
		r = append(r, c)
		v >>= bitsPerCell
	}
	return string(r)
}

func mouseClicked(win *gtk.Window, evt *gdk.Event, pg *Playground) bool {
	ev := gdk.EventButton{evt}
	dx := float64(pg.cellSize + pg.gapSize)
	ix := int(ev.X() / dx)
	iy := int(ev.Y() / dx)
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
	pg.da.QueueDraw()
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

	var cellSize uint = 12
	var gapSize uint = 4

	flag.Uint64Var(&pattern0, "pattern0", pattern0, "Set the initial pattern #0")
	flag.Uint64Var(&pattern1, "pattern1", pattern1, "Set the initial pattern #1")
	flag.IntVar(&xsize, "xsize", xsize, "Set the X size, or -1")
	flag.IntVar(&ysize, "ysize", ysize, "Set the Y size, or -1")
	flag.UintVar(&cellSize, "cellsize", cellSize, "The size of the cell")
	flag.UintVar(&gapSize, "gapsize", gapSize, "The size of the gap")
	flag.StringVar(&initialConfig, "init", initialConfig, "The name of the initial configuration")

	flag.Parse()

	if xsize == -1 || ysize == -1 {
		fullscreen = true
	}

	gtk.Init(nil)

	playground := NewPlayground(cellSize, gapSize)

	if err := setupWindow(playground); err != nil {
		fail(err)
	}
	gtk.Main()
}
