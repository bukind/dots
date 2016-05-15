package main

import (
	"testing"
)

func mku(x ...uint64) []uint64 {
	return x
}

func eq(x, y []uint64) bool {
	if len(x) != len(y) {
		return false
	}
	for i := 0; i < len(x); i++ {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

const lowers uint64 = 0x5555555555555555
const allset uint64 = 0xffffffffffffffff

func TestBitor3(t *testing.T) {
	x := mku(lowers, lowers, lowers)
	if len(x) != 3 {
		t.Errorf("len(mku) != 3")
	}
	y := mku(0, 0, 0)
	r1 := bitor3(x, y, nil, SHIFT_NONE)
	r2 := bitor3(y, x, nil, SHIFT_NONE)
	r3 := bitor3(x, y, nil, SHIFT_FIRST)
	r4 := bitor3(x, y, nil, SHIFT_ALL)
	r5 := bitor3(x, x, nil, SHIFT_NONE)
	r6 := bitor3(x, x, nil, SHIFT_ALL)

	if !eq(r1, r2) {
		t.Errorf("1|0 != 0|1")
	}
	if !eq(r1, x) {
		t.Errorf("1|0 != 1")
	}
	if !eq(r3, y) {
		t.Errorf("shift all 1>>1 != 0")
	}
	if !eq(r4, y) {
		t.Errorf("shift first 1>>1 != 0")
	}
	if !eq(r5, x) {
		t.Errorf("1|1 != 1")
	}
	if !eq(r6, y) {
		t.Errorf("1|1 >> 1 != 0")
	}
}

func TestJoin2(t *testing.T) {
	x := mku(lowers, lowers, lowers)
	y := mku(allset, allset, allset)
	if !eq(join2(x, x), y) {
		t.Errorf("join(1,1) != 11")
	}
}

func TestDiv2(t *testing.T) {
	x := mku(lowers, lowers, lowers)
	s := lowers >> 1
	y := mku(s, s, s)
	if !eq(div2(x), y) {
		t.Errorf("div2: 1010101 >> 1 != 0101010")
	}
}

func TestSumup3(t *testing.T) {
	x := mku(lowers, lowers, lowers)
	y := mku(0, 0, 0)
	r1 := sumup3(x, y, nil, SHIFT_NONE)
	r2 := sumup3(y, x, nil, SHIFT_NONE)
	r3 := sumup3(x, x, nil, SHIFT_NONE)
	r4 := sumup3(x, x, &x, SHIFT_NONE)
	if !eq(r1, x) {
		t.Errorf("sum: 1+0 != 1")
	}
	if !eq(r1, r2) {
		t.Errorf("sum: 1+0 != 0+1")
	}
	if !eq(r3, join2(y, x)) {
		t.Errorf("sum: 1+1 != 10")
	}
	if !eq(r4, join2(x, x)) {
		t.Errorf("sum: 1+1+1 != 11")
	}
}

func ExpectInt(t *testing.T, s string, a, b int) {
	if a != b {
		t.Errorf("invalid %s: %d != %d", s, a, b)
	}
}

func ExpectUint(t *testing.T, s string, a, b uint) {
	if a != b {
		t.Errorf("invalid %s: %d != %d", s, a, b)
	}
}

func ExpectUint64(t *testing.T, s string, a, b uint64) {
	if a != b {
		t.Errorf("invalid %s: %x != %x", s, a, b)
	}
}

func TestPlaygroundInit7x5(t *testing.T) {
	var cellSize uint = 7
	var gapSize uint = 1
	pg := NewPlayground(cellSize, gapSize)
	if pg == nil {
		t.Fatal("cannot make a new playground")
	}
	ExpectUint(t, "pg.cellSize", pg.cellSize, cellSize)
	ExpectUint(t, "pg.gapSize", pg.gapSize, gapSize)

	nx := 7
	ny := 5
	pg.Init(nil, nx, ny)
	ExpectInt(t, "len(pg.area)", len(pg.area), ny)
	ExpectInt(t, "pg.cellsPerRow", pg.cellsPerRow, nx)
	ExpectInt(t, "len(pg.area[0])", len(pg.area[0]), 1)
	ExpectInt(t, "len(pg.cellTypes)", len(pg.cellTypes), 3)
	ExpectUint64(t, "pg.lastIntMask", pg.lastIntMask, 0x3fff)
	ExpectUint(t, "pg.lastCellOffset", pg.lastCellOffset, 12)
}

func TestPlaygroundInit800x5(t *testing.T) {
	var cellSize uint = 7
	var gapSize uint = 1
	pg := NewPlayground(cellSize, gapSize)
	if pg == nil {
		t.Fatal("cannot make a new playground")
	}
	ExpectUint(t, "pg.cellSize", pg.cellSize, cellSize)
	ExpectUint(t, "pg.gapSize", pg.gapSize, gapSize)

	nx := 800
	ny := 5
	pg.Init(nil, nx, ny)
	ExpectInt(t, "len(pg.area)", len(pg.area), ny)
	ExpectInt(t, "pg.cellsPerRow", pg.cellsPerRow, nx)
	ExpectInt(t, "len(pg.area[0])", len(pg.area[0]), 25) // 800/32
	ExpectInt(t, "len(pg.cellTypes)", len(pg.cellTypes), 3)
	ExpectUint64(t, "pg.lastIntMask", pg.lastIntMask, allset)
	ExpectUint(t, "pg.lastCellOffset", pg.lastCellOffset, 62)
}
