package main

import (
	"fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
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
	deltaPhase := math.Pi * 2 / 60

	timer := time.NewTimer(interval)
	done := make(chan bool)
	defer close(done)

	var count float64 = 0

	go func() {
		for {
			select {
			case <-timer.C:
				win.QueueDraw()
				count++
				timer.Reset(interval)
			case <-done:
				break
			}
		}
	}()

	da.Connect("configure-event", func(da *gtk.DrawingArea, ev *gdk.Event) {
		fmt.Println("configure", da.GetAllocatedWidth(), da.GetAllocatedHeight())
	})

	// Event handlers
	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
		cr.SetSourceRGB(0, 0, 0)
		wx := float64(da.GetAllocatedWidth())
		wy := float64(da.GetAllocatedHeight())
		fmt.Println("draw wx=", wx, ", wy=", wy, ", count=", count)
		x0 := wx / 2
		y0 := wy / 2
		w := math.Min(x0, y0) * 0.9
		x := x0 + w*math.Sin(count*deltaPhase)
		y := y0 - w*math.Cos(count*deltaPhase)
		// cr.Rectangle(x, y, 10.0, 10.0)
		// cr.Fill()
		cr.SetLineWidth(10.0)
		cr.SetSourceRGB(0, 0, 1)
		cr.MoveTo(x0, y0)
		cr.LineTo(x, y)
		cr.Stroke()
	})

	// any key press lead to exit
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
		gtk.MainQuit()
	})

	gtk.Main()
}
