package main

import (
    "fmt"
	"github.com/conformal/gotk3/cairo"
	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/gdk"
    "math/rand"
	"time"
)


const minsize = 5
const maxsize = 20
const maxspeed = 1
const maxfps = 20
const nsprites = 8

type Sprite struct {
    surface *cairo.Surface
    wx     float64
    wy     float64
    x      float64
    y      float64
    vx     float64
    vy     float64
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
    res.x = rand.Float64() * float64(wx - iwx)
    res.y = rand.Float64() * float64(wy - iwy)
    return res
}


func (v *Sprite) Update(wx, wy, dt float64) {
    v.x += v.vx * dt
    v.y += v.vy * dt
    mx := wx - v.wx
    if v.x < 0 {
        v.x = -v.x
        if v.vx < 0 {
            v.vx = -v.vx
        }
    } else if v.x > mx  {
        v.x = 2 * mx - v.x
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
    } else if v.y > my  {
        v.y = 2 * my - v.y
        if v.vy > 0 {
            v.vy = -v.vy
        }
    }
}


func makeSprites(wx, wy, amount int) []*Sprite {
    res := make([]*Sprite, amount)
    for i := 0; i < amount; i++ {
       res[i] = NewSprite(wx, wy, minsize, maxsize)
    }
    return res
}


func main() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("dots")
	win.Connect("destroy", gtk.MainQuit)
    // win.SetSizeRequest(400, 400)
	win.Fullscreen()
	win.SetResizable(false)

	da, _ := gtk.DrawingAreaNew()
	win.Add(da)
	win.ShowAll()

	interval := time.Second / maxfps

	timer := time.NewTimer(interval)
	done := make(chan bool)
	defer close(done)

    wx := da.GetAllocatedWidth()
    wy := da.GetAllocatedHeight()
    fmt.Println(wx, wy)

    fwx := float64(wx)
    fwy := float64(wy)
    sprites := makeSprites(wx, wy, nsprites)
    now := time.Now()

	go func() {
		for {
			select {
			case <-timer.C:
				win.QueueDraw()
				timer.Reset(interval)
			case <-done:
				break
			}
		}
	}()

	// Event handlers
	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {

        prevtime := now
        now := time.Now()
        dt := now.Sub(prevtime).Seconds()

	    for i := 0; i < len(sprites); i++ {
            sprites[i].Update(fwx, fwy, dt)
            cr.SetSourceSurface(sprites[i].surface, sprites[i].x, sprites[i].y)
            cr.Paint()
		}
		fmt.Println("pass dt=", dt, ", perf=", time.Now().Sub(now))
	})

	// any key press lead to exit
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
	    gtk.MainQuit()
	})

	gtk.Main()
}
