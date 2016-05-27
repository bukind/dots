package main

import (
	"runtime"
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

func ExpectInt(t *testing.T, s string, a, b int) {
	if a != b {
		_, fn, ln, ok := runtime.Caller(1)
		if !ok {
			fn = "???"
			ln = 0
		}
		t.Errorf("@%s:%d invalid %s: %d != %d", fn, ln, s, a, b)
	}
}

func ExpectUint(t *testing.T, s string, a, b uint) {
	if a != b {
		_, fn, ln, ok := runtime.Caller(1)
		if !ok {
			fn = "???"
			ln = 0
		}
		t.Errorf("@%s:%d invalid %s: %d != %d", fn, ln, s, a, b)
	}
}

func ExpectUint64(t *testing.T, s string, a, b uint64) {
	if a != b {
		_, fn, ln, ok := runtime.Caller(1)
		if !ok {
			fn = "???"
			ln = 0
		}
		t.Errorf("@%s:%d invalid %s: %x != %x", fn, ln, s, a, b)
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
	ExpectUint64(t, "pg.lastIntMask", pg.lastIntMask, 0xfffffff)
	ExpectUint(t, "pg.lastCellOffset", pg.lastCellOffset, 24)
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
	ExpectInt(t, "len(pg.area[0])", len(pg.area[0]), 50) // 800/16
	ExpectInt(t, "len(pg.cellTypes)", len(pg.cellTypes), 3)
	ExpectUint64(t, "pg.lastIntMask", pg.lastIntMask, allset)
	ExpectUint(t, "pg.lastCellOffset", pg.lastCellOffset, 60)
}
