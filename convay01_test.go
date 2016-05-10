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
