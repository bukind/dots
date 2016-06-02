package main

import (
	"flag"
	"fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"os"
	"runtime/pprof"
)

var fullscreen bool = false
var xsize int = 400
var ysize int = 400
var pattern0 uint64 = 0x0
var pattern1 uint64 = 0x0
var initialConfig = ""

const lowBits64 uint64 = 0x5555555555555555

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

func fail(err error) {
	fmt.Fprintf(os.Stderr, "Failure: %v", err)
	os.Exit(1)
}

type cellType struct {
	color *gdk.RGBA
}

type cellValue struct {
	young uint64
	total uint64
}

type Playground struct {
	da             *gtk.DrawingArea
	cellSize       uint
	area           [][]uint64
	cellTypes      []*cellType
	cellsPerRow    int
	lastIntMask    uint64
	lastCellOffset uint
	repeats        int    // how many times to repeat
	iterations     uint64 // the number of steps passed
	viewX0         int    // the index of the top-left cell
	viewY0         int
}

func NewPlayground(cellSize uint) *Playground {
	pg := new(Playground)
	pg.cellSize = cellSize
	pg.viewX0 = 0
	pg.viewY0 = 0
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

func cellSplit(x uint64) cellValue {
	const lowMask uint64 = 0x3333333333333333
	y := x & lowMask
	return cellValue{y, (x>>2)&lowMask + y}
}

// Makes a running sum of the row.
// young, total
func tripleRow(orig []uint64, lco uint, lim uint64) []cellValue {
	nint := len(orig)
	result := make([]cellValue, nint)
	mask := cellMask
	ls := uint(bitsPerCell)
	rs := uint(64 - bitsPerCell)
	for i := 1; i < nint-1; i++ {
		o := orig[i]
		a := orig[i-1]
		b := orig[i+1]
		x := o + (o >> ls) + (b << rs) + (o << ls) + (a >> rs)
		result[i] = cellSplit(x)
	}
	if nint > 1 {
		o := orig[0]
		a := orig[nint-1]
		b := orig[1]
		x := o + (o >> ls) + (b << rs) + (o << ls) + ((a >> lco) & mask)
		result[0] = cellSplit(x)
		o = orig[nint-1]
		a = orig[nint-2]
		b = orig[0]
		x = o + (o >> ls) + ((b & mask) << lco) + (o << ls) + (a >> rs)
		x &= lim
		result[nint-1] = cellSplit(x)
	} else {
		o := orig[0]
		x := o + (o >> ls) + ((o & mask) << lco) + (o << ls) + ((o >> lco) & mask)
		x &= lim
		result[0] = cellSplit(x)
	}
	return result
}

// Sumup 8 adjacent cells together.
// Simple trick is to sumup all 9 cells, then subtrack the central one.
// Thus we can reuse the running sums of the rows.
//
func sumup8(arg [][]cellValue, orig []uint64) []cellValue {
	nint := len(orig)
	res := make([]cellValue, nint)
	a := arg[0]
	b := arg[1]
	c := arg[2]
	for i := 0; i < nint; i++ {
		v := cellSplit(orig[i])
		res[i].young = a[i].young + b[i].young + c[i].young - v.young
		res[i].total = a[i].total + b[i].total + c[i].total - v.total
	}
	return res
}

func (pg *Playground) Step() {
	// fmt.Printf("step %p\n", pg)
	nrows := len(pg.area)
	next := make([][]uint64, nrows) // the next state of the area
	roll := make([][]cellValue, 3)  // working area
	first := tripleRow(pg.area[0], pg.lastCellOffset, pg.lastIntMask)
	last := tripleRow(pg.area[nrows-1], pg.lastCellOffset, pg.lastIntMask)
	roll[1] = last
	roll[2] = first
	for iy := 0; iy < nrows; iy++ {
		// shift all rows
		roll[0] = roll[1]
		roll[1] = roll[2]
		// fill the next row
		idx := iy + 1
		if idx < nrows {
			roll[2] = tripleRow(pg.area[idx], pg.lastCellOffset, pg.lastIntMask)
		} else {
			roll[2] = first
		}
		// now sumup all young and total number of adjacent cells.
		// counts is an array of number of Y (young) and T(total) cells around.
		counts := sumup8(roll, pg.area[iy])
		// rules are:
		// 1. each young cell converts to old.
		// 2. an empty cell converts to young cell if Y<2 and T=3, otherwise is empty
		// 3. an old cell remains live if Y<2 and T=[2..3], otherwise is empty
		nint := len(pg.area[iy])
		next[iy] = make([]uint64, nint)
		const ones uint64 = 0x1111111111111111
		for ix := 0; ix < nint; ix++ {
			orig := pg.area[iy][ix]
			noto := ^orig
			notyoung := ^counts[ix].young
			total := counts[ix].total

			// condition if young less than 2
			yless2 := (notyoung >> 1) & (notyoung >> 2) & (notyoung >> 3)

			// condition if total is 2 or 3
			nott := ^total
			total23 := (total >> 1) & (nott >> 2) & (nott >> 3)

			// extract all young cells and convert them into old
			new1 := (orig & ones) << 2

			// extract all empty cells
			empt := noto & (noto >> 2)
			// convert them into youngs
			new2 := empt & yless2 & total & total23 & ones

			// extract all old cells
			olds := orig >> 2
			// convert them into old
			new3 := (olds & yless2 & total23 & ones) << 2

			// now combine all three outcomes
			next[iy][ix] = new1 | new2 | new3
		}
		next[iy][nint-1] &= pg.lastIntMask
	}
	pg.area = next
	pg.iterations++
	// fmt.Println("step done\n")
}

func makeCellType(colorName string) *cellType {
	ct := new(cellType)
	ct.color = gdk.NewRGBA()
	if !ct.color.Parse(colorName) {
		panic("failed to parse color name")
	}
	return ct
}

func (pg *Playground) Init(da *gtk.DrawingArea, nx, ny int) {
	fmt.Println("configure-event")

	if nx <= 0 {
		panic("Too narrow area")
	}
	if ny <= 0 {
		panic("Too short area")
	}

	// define cell types
	pg.cellTypes = make([]*cellType, cellMask+1)
	pg.cellTypes[0x0] = makeCellType("white")
	pg.cellTypes[0x1] = makeCellType("lightgreen")
	pg.cellTypes[0x4] = makeCellType("blue")

	if initialConfig == "" {
		// check pattern0, pattern1
		p0 := pattern0
		p1 := pattern1
		for i := 0; i < cellsPerInt; i++ {
			if pg.cellTypes[p0&cellMask] == nil {
				panic("Bad pattern")
			}
			if pg.cellTypes[p1&cellMask] == nil {
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
	pg.repeats = 0

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
			v = 0x1
		case '2':
			v = 0x4
		}
		nv := pg.area[y][ix] & ^(cellMask<<shift) + (v << shift)
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
	if pg.repeats > 0 {
		pg.repeats--
		pg.Step()
		pg.da.QueueDraw()
	} else if pg.repeats == -1 {
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

func areaSetupEvent(da *gtk.DrawingArea, ev *gdk.Event, pg *Playground) {
	_ = ev
	nx := da.GetAllocatedWidth() / int(pg.cellSize)
	ny := da.GetAllocatedHeight() / int(pg.cellSize)
	pg.Init(da, nx, ny)
}

func areaDrawEvent(da *gtk.DrawingArea, cr *cairo.Context, pg *Playground) {
	_ = cr
	if pg.area == nil {
		areaSetupEvent(da, nil, pg)
	}
	var gapSize uint = 0
	if pg.cellSize > 3 {
		gapSize = pg.cellSize / 4
	}
	dx := float64(pg.cellSize)
	cs := float64(pg.cellSize - gapSize)
	olds := 0
	news := 0
	// calculate the viewport parameters
	cellsX := da.GetAllocatedWidth() / int(pg.cellSize)
	cellsY := da.GetAllocatedHeight() / int(pg.cellSize)
	startY := pg.viewY0
	startX := pg.viewX0
	endY := startY + cellsY
	endX := startX + cellsX
	if endY > len(pg.area) {
		if cellsY > len(pg.area) {
			startY = 0
		} else {
			startY = len(pg.area) - cellsY
		}
		endY = len(pg.area)
	}
	if startX+cellsX > pg.cellsPerRow {
		if cellsX > pg.cellsPerRow {
			startX = 0
		} else {
			startX = pg.cellsPerRow - cellsX
		}
		endX = pg.cellsPerRow
	}
	// convert X cells into ints
	cellX0 := startX
	cellY0 := startY
	startX = startX / cellsPerInt
	endX = (endX + cellsPerInt - 1) / cellsPerInt

	for iy := startY; iy < endY; iy++ {
		row := pg.area[iy]
		y := float64(iy-cellY0) * dx
		for mask, cellType := range pg.cellTypes {
			if mask == 0 || cellType == nil {
				// optimization - skip empty cells
				continue
			}
			cnt := &olds
			if mask == 1 {
				cnt = &news
			}
			rgba := cellType.color.Floats()
			cr.SetSourceRGBA(rgba[0], rgba[1], rgba[2], rgba[3])
			for ix := startX; ix < endX; ix++ {
				value := row[ix]
				idx0 := ix * cellsPerInt
				maxIdx := idx0 + cellsPerInt
				if maxIdx > pg.cellsPerRow {
					maxIdx = pg.cellsPerRow
				}
				for idx := idx0; idx < maxIdx; idx++ {
					if int(value&cellMask) == mask {
						cr.Rectangle(dx*float64(idx-cellX0), y, cs, cs)
						(*cnt)++
					}
					value >>= bitsPerCell
				}
			}
			cr.Fill()
		}
	}
	cr.MoveTo(1., 14.)
	cr.SetSourceRGB(0., 0., 0.)
	cr.SetFontSize(12.)
	total := float64(pg.cellsPerRow * len(pg.area))
	cr.ShowText(fmt.Sprintf("steps:%d cells:%d/%.1f%%  old:%d/%.1f%%",
		pg.iterations, olds+news, float64(olds+news)*100/total,
		olds, float64(olds)*100/total))
	cr.Stroke()
	if pg.repeats != 0 {
		pg.StepAndDraw()
	}
}

func keyPressEvent(win *gtk.Window, evt *gdk.Event, pg *Playground) {
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
	case gdk.KEY_S:
		{
			// clean the lower half of the field
			nint := len(pg.area)
			for iy := nint / 2; iy < nint; iy++ {
				for ix := 0; ix < len(pg.area[iy]); ix++ {
					pg.area[iy][ix] = 0
				}
			}
		}
	case gdk.KEY_t:
		pg.repeats += 10
		pg.StepAndDraw()
	case gdk.KEY_x:
		pg.repeats = 0
	case gdk.KEY_s:
		pg.repeats = -1
		pg.StepAndDraw()
	}
}

func mouseScrollEvent(win *gtk.Window, evt *gdk.Event, pg *Playground) bool {
	ev := gdk.EventScroll{evt}
	dy := ev.DeltaY()
	var newcs = pg.cellSize
	if dy < 0 {
		// zoom in
		if newcs < 4 {
			newcs += 1
		} else if newcs < 10 {
			newcs += 2
		} else if newcs < 30 {
			newcs = uint(1.4 * float64(newcs))
		} else {
			// too large cell - not zooming
		}
	} else if dy > 0 {
		// zoom out
		if newcs > 10 {
			newcs = uint(float64(newcs) / 1.4)
			if newcs > 10 {
				newcs = 10
			}
		} else if newcs > 4 {
			newcs -= 2
		} else if newcs > 1 {
			newcs -= 1
		} else {
			// too small cell - not zooming
		}
	}
	if newcs == pg.cellSize {
		return true
	}

	// old cell index under the cursor
	oldX := float64(pg.viewX0) + ev.X()/float64(pg.cellSize)
	oldY := float64(pg.viewY0) + ev.Y()/float64(pg.cellSize)
	// find the 0 position so that the same cell is under the cursor
	newX0 := int(oldX - (ev.X() / float64(newcs)))
	newY0 := int(oldY - (ev.Y() / float64(newcs)))
	if newX0 < 0 {
		newX0 = 0
	} else if newX0 >= pg.cellsPerRow {
		newX0 = pg.cellsPerRow - 1
	}
	if newY0 < 0 {
		newY0 = 0
	} else if newY0 >= len(pg.area) {
		newY0 = len(pg.area) - 1
	}

	fmt.Printf("scroll: dy:%.1f, (x,y):%.1f,%.1f v0:%d,%d -> %d,%d\n",
		dy, ev.X(), ev.Y(), pg.viewX0, pg.viewY0, newX0, newY0)

	pg.viewX0 = newX0
	pg.viewY0 = newY0
	pg.cellSize = newcs
	pg.da.QueueDraw()
	return true
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

func mouseClickedEvent(win *gtk.Window, evt *gdk.Event, pg *Playground) bool {
	ev := gdk.EventButton{evt}
	dx := float64(pg.cellSize)
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

	da.AddEvents(int(gdk.SCROLL_MASK))

	win.Add(da)
	win.ShowAll()

	if _, err = da.Connect("configure-event", areaSetupEvent, playground); err != nil {
		return err
	}

	if _, err = da.Connect("draw", areaDrawEvent, playground); err != nil {
		return err
	}

	if _, err = win.Connect("key-press-event", keyPressEvent, playground); err != nil {
		return err
	}

	if _, err = win.Connect("button-press-event", mouseClickedEvent, playground); err != nil {
		return err
	}

	if _, err = win.Connect("scroll-event", mouseScrollEvent, playground); err != nil {
		return err
	}

	return nil
}

func main() {

	var cellSize uint = 12
	var prof string

	flag.Uint64Var(&pattern0, "pattern0", pattern0, "Set the initial pattern #0")
	flag.Uint64Var(&pattern1, "pattern1", pattern1, "Set the initial pattern #1")
	flag.IntVar(&xsize, "xsize", xsize, "Set the X size, or -1")
	flag.IntVar(&ysize, "ysize", ysize, "Set the Y size, or -1")
	flag.UintVar(&cellSize, "cellsize", cellSize, "The size of the cell")
	flag.StringVar(&initialConfig, "init", initialConfig, "The name of the initial configuration")
	flag.StringVar(&prof, "prof", "", "The name of the cpu profile output")

	flag.Parse()

	if xsize == -1 || ysize == -1 {
		fullscreen = true
	}

	gtk.Init(nil)

	playground := NewPlayground(cellSize)

	if err := setupWindow(playground); err != nil {
		fail(err)
	}
	if prof != "" {
		f, err := os.Create(prof)
		if err != nil {
			fail(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	gtk.Main()
}
