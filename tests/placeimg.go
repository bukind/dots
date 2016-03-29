package main

import (
	"flag"
	"fmt"
	"github.com/conformal/gotk3/cairo"
	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/gtk"
	"math"
	"math/rand"
	"time"
)

var minsize int = 5
var maxsize int = 20
var maxfps int64 = 20
var nsprites int = 50
var maxspeed float64 = 10.0
var minforce float64 = 0.1
var maxforce float64 = 0.5
var fullscreen bool = false

type Sprite struct {
	surface *cairo.Surface
	wx      float64
	wy      float64
	x       float64
	y       float64
	vx      float64
	vy      float64
	rad     float64
	fx      float64 // fx
	fy      float64 // fy
	mass    float64
}

func NewSprite(wx, wy, minsize, maxsize int) *Sprite {
	res := new(Sprite)
	iwx := rand.Intn(maxsize-minsize+1) + minsize
	iwy := iwx
	res.surface = cairo.ImageSurfaceCreate(cairo.FORMAT_RGB24, iwx, iwy)

	red := rand.Float64()
	green := rand.Float64()
	blue := rand.Float64()
	c := cairo.Create(res.surface)
	c.SetSourceRGB(red, green, blue)
	c.Paint()

	res.wx = float64(iwx)
	res.wy = float64(iwy)
	res.vx = maxspeed * 2 * (rand.Float64() - 0.5)
	res.vy = maxspeed * 2 * (rand.Float64() - 0.5)
	res.x = rand.Float64() * float64(wx-iwx)
	res.y = rand.Float64() * float64(wy-iwy)
	res.rad = res.wx + res.wy
	res.mass = res.wx * res.wy
	return res
}

func (v *Sprite) UpdatePosition(wx, wy, dt float64) float64 {
	v.vx += v.fx / v.mass * dt
	v.vy += v.fy / v.mass * dt
	v.x += v.vx * dt
	v.y += v.vy * dt
	mx := wx - v.wx
	if v.x < 0 {
		v.x = -v.x
		if v.vx < 0 {
			v.vx = -v.vx
		}
	} else if v.x > mx {
		v.x = 2*mx - v.x
		if v.vx > 0 {
			v.vx = -v.vx
		}
	}
	my := wy - v.wy
	if v.y < 0 {
		v.y = -v.y
		if v.vy < 0 {
			v.vy = -v.vy
		}
	} else if v.y > my {
		v.y = 2*my - v.y
		if v.vy > 0 {
			v.vy = -v.vy
		}
	}
	return math.Hypot(v.vx, v.vy)
}

func UpdateForces(sprites []*Sprite, matrix [][]float64) {
	for _, s := range sprites {
		s.fx = 0
		s.fy = 0
	}

	for i, v := range sprites {
		for j, s := range sprites[i+1:] {
			// d mv = f dt
			// for k > 0 that is attraction:
			// r = R/r0
			// f = k / r^2,  r > 1
			// f = k*(r-1)*2 + k, r < 1
			// for k < 0 that is distraction:
			// f = k / r^2,  r > 1
			// f = k*(r-1) + k, r < 1
			//
			// fx := f * rx / R
			kf := matrix[i][j] * v.mass * s.mass
			r0 := v.rad + s.rad
			rbig := math.Hypot(s.x-v.x, s.y-v.y)
			r := rbig / r0
			var f float64
			if r > 1 {
				f = kf / (r * r)
			} else {
				f = kf * (r - 1)
				if kf > 0 {
					f *= 2
				}
				f += kf
			}
			f /= rbig
			fx := f * (s.x - v.x)
			fy := f * (s.y - v.y)
			v.fx += fx
			v.fy += fy
			s.fx -= fx
			s.fy -= fy
		}
	}
}

func makeSprites(sprites []*Sprite, wx, wy int) {
	for i := 0; i < len(sprites); i++ {
		sprites[i] = NewSprite(wx, wy, minsize, maxsize)
	}
}

func makeFeeling(matrix [][]float64) {
	for i := 0; i < len(matrix); i++ {
		matrix[i] = make([]float64, len(matrix))
	}
	for i := 0; i < len(matrix); i++ {
		for j := 0; j < len(matrix); j++ {
			if i == j {
				matrix[i][j] = 0
			} else {
				f := rand.Float64()*(maxforce-minforce) + maxforce
				if rand.Intn(2) > 0 {
					f = -f
				}
				matrix[i][j] = f
				matrix[j][i] = f
			}
		}
	}
}

func main() {

	// options
	flag.IntVar(&nsprites, "nsprites", 10, "Set the number of sprites")
	flag.IntVar(&minsize, "minsize", 5, "Set the minsize")
	flag.IntVar(&maxsize, "maxsize", 40, "Set the maxsize")
	flag.Int64Var(&maxfps, "maxfps", 20, "Set the maximum fps")
	flag.Float64Var(&maxspeed, "maxspeed", 10., "Set the maxspeed")
	flag.Float64Var(&minforce, "minforce", 0., "Set the minforce")
	flag.Float64Var(&maxforce, "maxforce", 3., "Set the maxforce")
	flag.BoolVar(&fullscreen, "fullscreen", false, "Allow to run fullscreen")

	flag.Parse()

	// check flags
	if nsprites < 1 || minsize < 1 || minsize >= maxsize ||
		maxfps < 1 || maxfps > 100 || maxspeed < 0.1 || maxforce < 0 ||
		minforce >= maxforce {
		panic("bad nsprites")
	}

	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("dots")
	win.Connect("destroy", gtk.MainQuit)
	if fullscreen {
		win.Fullscreen()
	} else {
		win.SetSizeRequest(400, 400)
	}
	win.SetResizable(false)

	da, _ := gtk.DrawingAreaNew()
	win.Add(da)
	win.ShowAll()

	done := make(chan bool)
	defer close(done)

	sprites := make([]*Sprite, nsprites)
	matrix := make([][]float64, nsprites)
	var fwx, fwy float64

	now := time.Now()
	prevstat := now

	// Event handlers
	da.Connect("configure-event", func(da *gtk.DrawingArea, ev *gdk.Event) {
		// setup everything
		wx := da.GetAllocatedWidth()
		wy := da.GetAllocatedHeight()
		fwx = float64(wx)
		fwy = float64(wy)
		makeSprites(sprites, wx, wy)
		makeFeeling(matrix)
	})

	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {

		prevtime := now
		now = time.Now()
		dt := now.Sub(prevtime).Seconds()

		UpdateForces(sprites, matrix)

		maxv := 0.0
		for i := 0; i < len(sprites); i++ {
			v := sprites[i].UpdatePosition(fwx, fwy, dt)
			if v > maxv {
				maxv = v
			}
			cr.SetSourceSurface(sprites[i].surface, sprites[i].x, sprites[i].y)
			cr.Paint()
		}
		if now.Sub(prevstat) > time.Minute {
			fmt.Println("pass dt=", dt, ", perf=", time.Now().Sub(now), ", maxv=", maxv)
		}

		// restart the drawing
		go func() {
			interval := time.Duration(int64(time.Second) / maxfps)
			timer := time.NewTimer(interval)
			select {
			case <-done:
				break
			case <-timer.C:
				win.QueueDraw()
			}
		}()
	})

	// any key press lead to exit
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
		gtk.MainQuit()
	})

	gtk.Main()
}
