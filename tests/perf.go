package main

import (
    "fmt"
	"github.com/conformal/gotk3/cairo"
	"github.com/conformal/gotk3/gtk"
	"github.com/conformal/gotk3/gdk"
	"math"
	"time"
)

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

	interval := time.Second

	timer := time.NewTimer(interval)
	done := make(chan bool)
	defer close(done)

	var basecolor float64 = 0
	var radius float64 = 1.5
	step := radius * 2.5

	go func() {
		for {
			select {
			case <-timer.C:
				win.QueueDraw()
				basecolor += 0.1
				timer.Reset(interval)
			case <-done:
				break
			}
		}
	}()

	// Event handlers
	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
		start := time.Now()
		wx := float64(da.GetAllocatedWidth())
		wy := float64(da.GetAllocatedHeight())
		i := basecolor
		diam := radius*2
	    for y := step/2; y < wy - step/2; y += step {
		    for x := step/2; x < wx - step/2; x += step {
			    i += 0.00001
			    red := (1. + math.Sin(i)) * 0.5
				grn := (1. + math.Sin(i*1.2)) * 0.5
				blu := (1. + math.Sin(i*1.7)) * 0.5
				cr.SetSourceRGB(red, grn, blu)
			    cr.Rectangle(x-radius, y-radius, diam, diam)
				// cr.Arc(x, y, radius, 0., math.Pi*2)
				cr.Fill()
			}
		}
		fmt.Println("pass", basecolor, time.Now().Sub(start))
	})

	// any key press lead to exit
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
	    gtk.MainQuit()
	})

	gtk.Main()
}
