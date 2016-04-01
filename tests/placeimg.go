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
var nsprites int = 2
var maxspeed float64 = 10.0
var minforce float64 = 10
var maxforce float64 = 100
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

type State struct {
	pos    Vector
	speed  Vector
	force  Vector
	energy float64 // potential only
}

type Sprite struct {
	surface *cairo.Surface
	radius  float64
	mass    float64
	s       State
}

func (v *Sprite) Speed() float64 {
	return VAbs(v.s.speed)
}

func ScaleEnergy(sprites []*Sprite, scale float64) {
	ratio := math.Sqrt(scale)
	for _, v := range sprites {
		v.s.speed = VScale(v.s.speed, ratio)
	}
}

func NewSprite(maxx, maxy, minsize, maxsize int) *Sprite {
	res := new(Sprite)
	iwx := rand.Intn(maxsize-minsize+1) + minsize
	iwy := iwx
	res.radius = float64(iwx) / 1.2
	res.mass = res.radius * res.radius

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

	res.s.speed = mkVec(mkRand(-maxspeed, maxspeed), mkRand(-maxspeed, maxspeed))
	res.s.pos = mkVec(mkRand(res.radius, float64(maxx)-res.radius),
		mkRand(res.radius, float64(maxy)-res.radius))
	return res
}

func (v *Sprite) UpdatePosition(maxx, maxy, dt float64) {
	v.s.pos += VScale(v.s.speed, dt)
	mx := maxx - v.radius
	if vx(v.s.pos) < v.radius {
		v.s.pos = mkVec(2*v.radius-vx(v.s.pos), vy(v.s.pos))
		if vx(v.s.speed) < 0 {
			v.s.speed = mkVec(-vx(v.s.speed), vy(v.s.speed))
		}
	} else if vx(v.s.pos) > maxx-v.radius {
		v.s.pos = mkVec(2*mx-vx(v.s.pos), vy(v.s.pos))
		if vx(v.s.speed) > 0 {
			v.s.speed = mkVec(-vx(v.s.speed), vy(v.s.speed))
		}
	}
	my := maxy - v.radius
	if vy(v.s.pos) < v.radius {
		v.s.pos = mkVec(vx(v.s.pos), 2*v.radius-vy(v.s.pos))
		if vy(v.s.speed) < 0 {
			v.s.speed = mkVec(vx(v.s.speed), -vy(v.s.speed))
		}
	} else if vy(v.s.pos) > maxy-v.radius {
		v.s.pos = mkVec(vx(v.s.pos), 2*my-vy(v.s.pos))
		if vy(v.s.speed) > 0 {
			v.s.speed = mkVec(vx(v.s.speed), -vy(v.s.speed))
		}
	}
}

func (v *Sprite) UpdateSpeed(old State) {
	scale := 0.5 / v.mass
	v.s.speed += VScale((v.s.force + old.force), scale)
}

func collectState(sprites []*Sprite) ([]State, float64, float64) {
	potential := 0.0
	kinetic := 0.0
	state := make([]State, len(sprites))
	for i, v := range sprites {
		state[i] = v.s
		potential += v.s.energy
		speed := VAbs(v.s.speed)
		kinetic += v.mass * speed * speed / 2
	}
	return state, potential, kinetic
}

// UpdateDynamics calculates the force and the potential energy of
// the sprite.
// The potential energy is by the formula:
// E = K / r,  r > R0
// E = K / R0, r < R0
// where R0 = r1 + r2, r = |Pos1 - Pos2|
// The force is then:
// F = K / r^2
func UpdateDynamics(sprites []*Sprite, matrix [][]float64) {
	for _, v := range sprites {
		v.s.force = mkVec(0, 0)
		v.s.energy = 0
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
			rmin := v.radius + s.radius
			rvec := s.s.pos - v.s.pos
			rlen := VAbs(rvec)
			r := rlen / rmin
			kf := matrix[i][j] * v.mass * s.mass
			var f float64
			if r < 1 {
				r = 1
				f = 0
			} else {
			    f = kf / (r * r) / rlen
			}
			force := VScale(rvec, f)
			v.s.force += force
			s.s.force -= force
			energy := kf / r
			v.s.energy += energy
			s.s.energy += energy
		}
	}
}

func UpdateKinematics(sprites []*Sprite, matrix [][]float64) {
	// keep the original state
	oldstate, oldpot, oldkin := collectState(sprites)

	// update the force and the energy of the sprites
	UpdateDynamics(sprites, matrix)

	// update speed
	for i, v := range sprites {
		v.UpdateSpeed(oldstate[i])
	}

	_, newpot, newkin := collectState(sprites)

	fmt.Printf("W/K=%f/%f -> W/K=%f,%f D=%f\n",
		oldpot, oldkin, newpot, newkin, newpot+newkin-oldpot-oldkin)
}

func (v *Sprite) Paint(cr *cairo.Context) {
	cr.SetSourceSurface(v.surface, vx(v.s.pos)+v.radius, vy(v.s.pos)+v.radius)
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
		for j := i; j < len(matrix); j++ {
			if i == j {
				matrix[i][j] = 0
			} else {
				f := mkRand(minforce, maxforce)
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
	flag.IntVar(&nsprites, "nsprites", nsprites, "Set the number of sprites")
	flag.IntVar(&minsize, "minsize", minsize, "Set the minsize")
	flag.IntVar(&maxsize, "maxsize", maxsize, "Set the maxsize")
	flag.Int64Var(&maxfps, "maxfps", maxfps, "Set the maximum fps")
	flag.Float64Var(&maxspeed, "maxspeed", maxspeed, "Set the maxspeed")
	flag.Float64Var(&minforce, "minforce", minforce, "Set the minforce")
	flag.Float64Var(&maxforce, "maxforce", maxforce, "Set the maxforce")
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

	setup := func(da *gtk.DrawingArea) {
		sprites = make([]*Sprite, nsprites)
		matrix = make([][]float64, nsprites)
		wx := da.GetAllocatedWidth()
		wy := da.GetAllocatedHeight()
		fwx = float64(wx)
		fwy = float64(wy)
		makeSprites(sprites, wx, wy)
		makeFeeling(matrix)
		UpdateDynamics(sprites, matrix)
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

		// update the location of each sprite
		for _, s := range sprites {
			s.UpdatePosition(fwx, fwy, dt)
		}

		// update the speed of the sprites keeping the balance
		// of the energy
		UpdateKinematics(sprites, matrix)

		for _, s := range sprites {
			s.Paint(cr)
		}

		if now.Sub(prevstat) > time.Second {
			prevstat = now
			fmt.Printf("pass %.0f dt=%.3f perf=%v\n",
				now.Sub(start).Seconds(),
				dt, time.Now().Sub(now))
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
