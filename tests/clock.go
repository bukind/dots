package main

import (
	"fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gtk"
	"math"
	"time"
)

func main() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("dots")
	win.Connect("destroy", gtk.MainQuit)
	win.SetSizeRequest(400, 400)
	win.SetResizable(false)

	da, _ := gtk.DrawingAreaNew()
	win.Add(da)
	win.ShowAll()

	interval := time.Second
	timer := time.NewTimer(interval)
	done := make(chan bool)
	var count float64 = 0
	deltaPhase := math.Pi * 2 / 60

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

	// Event handlers
	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
		cr.SetSourceRGB(0, 0, 0)
		wx := float64(da.GetAllocatedWidth())
		wy := float64(da.GetAllocatedHeight())
		fmt.Println("wx=", wx, ", wy=", wy, ", count=", count)
		x0 := wx / 2
		y0 := wy / 2
		x := x0 * (1.0 + 0.8*math.Sin(count*deltaPhase))
		y := y0 * (1.0 - 0.8*math.Cos(count*deltaPhase))
		// cr.Rectangle(x, y, 10.0, 10.0)
		// cr.Fill()
		cr.SetLineWidth(10.0)
		cr.SetSourceRGB(0, 0, 1)
		cr.MoveTo(x0, y0)
		cr.LineTo(x, y)
		cr.Stroke()
	})

	gtk.Main()
	close(done)
}
