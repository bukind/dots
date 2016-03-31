package main

import (
	"flag"
	"fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"math"
	"math/cmplx"
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

type Vector complex128

func mkVec(r, i float64) Vector {
	return Vector(complex(r, i))
}

func vx(v Vector) float64 {
	return real(v)
}

func vy(v Vector) float64 {
	return imag(v)
}

func VAbs(v Vector) float64 {
	return cmplx.Abs(complex128(v))
}

func VScale(v Vector, scale float64) Vector {
	return mkVec(real(v)*scale, imag(v)*scale)
}

func mkRand(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

type Sprite struct {
	surface  *cairo.Surface
	position Vector
	speed    Vector
	force    Vector
	radius   float64
	mass     float64
}

func (v *Sprite) Speed() float64 {
	return VAbs(v.speed)
}

type SpriteMetric struct {
	speed    float64
	momentum float64
	energy   float64
	sume     float64
}

func GetMax(v *SpriteMetric, sprites []*Sprite) {
	for _, s := range sprites {
		speed := s.Speed()
		if speed > v.speed {
			v.speed = speed
		}
		momentum := speed * s.mass
		if momentum > v.momentum {
			v.momentum = momentum
		}
		energy := momentum * speed / 2
		if energy > v.energy {
			v.energy = energy
		}
		v.sume += energy
	}
}

func ScaleEnergy(sprites []*Sprite, scale float64) {
	ratio := math.Sqrt(scale)
	for _, s := range sprites {
		s.speed = VScale(s.speed, ratio)
	}
}

func NewSprite(maxx, maxy, minsize, maxsize int) *Sprite {
	res := new(Sprite)
	iwx := rand.Intn(maxsize-minsize+1) + minsize
	iwy := iwx
	res.radius = float64(iwx) / 1.2

	if res.radius >= float64(maxx/2) || res.radius >= float64(maxy/2) {
		panic("Too small drawing area")
	}

	var err error
	res.surface, err = cairo.NewSurfaceImage(cairo.FORMAT_RGB24, iwx, iwy)
	if err != nil {
		panic(err)
	}

	red := rand.Float64()
	green := rand.Float64()
	blue := rand.Float64()
	c := cairo.Create(res.surface)
	c.SetSourceRGB(red, green, blue)
	c.Paint()

	res.speed = mkVec(mkRand(-maxspeed, maxspeed), mkRand(-maxspeed, maxspeed))
	res.position = mkVec(mkRand(res.radius, float64(maxx)-res.radius),
		mkRand(res.radius, float64(maxy)-res.radius))
	res.mass = res.radius * res.radius
	return res
}

func (v *Sprite) UpdatePosition(maxx, maxy, dt float64) {
	v.speed += VScale(v.force, dt/v.mass)
	v.position += VScale(v.speed, dt)
	mx := maxx - v.radius
	if vx(v.position) < v.radius {
		v.position = mkVec(2*v.radius-vx(v.position), vy(v.position))
		if vx(v.speed) < 0 {
			v.speed = mkVec(-vx(v.speed), vy(v.speed))
		}
	} else if vx(v.position) > maxx-v.radius {
		v.position = mkVec(2*mx-vx(v.position), vy(v.position))
		if vx(v.speed) > 0 {
			v.speed = mkVec(-vx(v.speed), vy(v.speed))
		}
	}
	my := maxy - v.radius
	if vy(v.position) < v.radius {
		v.position = mkVec(vx(v.position), 2*v.radius-vy(v.position))
		if vy(v.speed) < 0 {
			v.speed = mkVec(vx(v.speed), -vy(v.speed))
		}
	} else if vy(v.position) > maxy-v.radius {
		v.position = mkVec(vx(v.position), 2*my-vy(v.position))
		if vy(v.speed) > 0 {
			v.speed = mkVec(vx(v.speed), -vy(v.speed))
		}
	}
}

func UpdateForces(sprites []*Sprite, matrix [][]float64) {
	for _, s := range sprites {
		s.force = mkVec(0, 0)
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
			r0 := v.radius + s.radius
			rvec := s.position - v.position
			rbig := VAbs(rvec)
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
			force := VScale(rvec, f)
			v.force += force
			s.force -= force
		}
	}
}

func (v *Sprite) Paint(cr *cairo.Context) {
	cr.SetSourceSurface(v.surface, vx(v.position)+v.radius, vy(v.position)+v.radius)
	cr.Paint()
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

	var sprites []*Sprite
	var matrix [][]float64
	var fwx, fwy float64

	now := time.Now()
	start := now
	prevstat := now

	var initialEnergySum float64 = 0.

	setup := func(da *gtk.DrawingArea) {
		sprites = make([]*Sprite, nsprites)
		matrix = make([][]float64, nsprites)
		wx := da.GetAllocatedWidth()
		wy := da.GetAllocatedHeight()
		fwx = float64(wx)
		fwy = float64(wy)
		makeSprites(sprites, wx, wy)
		makeFeeling(matrix)

		var m SpriteMetric
		GetMax(&m, sprites)
		initialEnergySum = m.sume
	}

	// Event handlers
	da.Connect("configure-event", func(da *gtk.DrawingArea, ev *gdk.Event) {
		// setup everything
		if sprites == nil {
			setup(da)
		}
	})

	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
		if sprites == nil {
			setup(da)
		}
		prevtime := now
		now = time.Now()
		dt := now.Sub(prevtime).Seconds()

		UpdateForces(sprites, matrix)

		var max SpriteMetric
		GetMax(&max, sprites)

		ScaleEnergy(sprites, initialEnergySum/max.sume)

		for _, s := range sprites {
			s.UpdatePosition(fwx, fwy, dt)
		}

		for _, s := range sprites {
			s.Paint(cr)
		}

		if now.Sub(prevstat) > time.Second*5 {
			prevstat = now
			fmt.Printf("pass %.0f dt=%.3f perf=%v maxv/p/e=%.1f/%.1e/%.1e sume=%.1e\n",
				now.Sub(start).Seconds(),
				dt, time.Now().Sub(now),
				max.speed, max.momentum, max.energy,
				max.sume)
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
