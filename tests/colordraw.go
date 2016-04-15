package main

import (
	"fmt"
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"math"
	"time"
)

const fullscreen = false
const interval = time.Second / 40

/*
func makeColor(phi float64) []float64 {
  // Xlin = X^1.7
  // Y = 0.2126*Rlin + 0.7152*Glin + 0.0722*Blin
	// red - cyan
	// green - magenta
	// blue - yellow
	r := math.Cos(phi)

		(1-math.Cos(count*deltaPhase))/2,
			(1-math.Cos(count*deltaPhase*1.1))/2,
			(1-math.Cos(count*deltaPhase*1.3))/2)
}
*/


func main() {
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

	deltaPhase := math.Pi * 2 / 60

	timer := time.NewTimer(interval)
	done := make(chan bool)
	defer close(done)

	start := time.Now()

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

	da.Connect("configure-event", func(da *gtk.DrawingArea, ev *gdk.Event) {
		fmt.Println("configure", da.GetAllocatedWidth(), da.GetAllocatedHeight())
	})

	// Event handlers
	da.Connect("draw", func(da *gtk.DrawingArea, cr *cairo.Context) {
		count := time.Now().Sub(start).Seconds()
		wx := float64(da.GetAllocatedWidth())
		wy := float64(da.GetAllocatedHeight())
		// fmt.Println("draw wx=", wx, ", wy=", wy, ", count=", count)
		x0 := wx / 2
		y0 := wy / 2
		w := math.Min(x0, y0) * 0.8
		r := w * 0.05
		const nitems = 20
		// color := makeColor(count * deltaPhase)
		cr.SetSourceRGB(1, 0, 0)
		for i := 0; i < nitems; i++ {
		  phase := count*deltaPhase+float64(i)*math.Pi*2.0/nitems
			x := x0 + w*math.Sin(phase)
			y := y0 - w*math.Cos(phase)
			// if (i & 1) == 0 {
			// cr.Rectangle(x-r, y-r, r+r, r+r)
			// } else {
			cr.MoveTo(x0, y0)
			cr.Arc(x, y, r, 0., math.Pi*2.0001)
			// }
		}
		cr.Fill()
		// cr.Rectangle(x, y, 10.0, 10.0)
		// cr.SetLineWidth(10.0)
		// cr.SetSourceRGB(0, 0, 1)
		// cr.LineTo(x, y)
		// cr.Stroke()
	})

	// any key press lead to exit
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
		gtk.MainQuit()
	})

	gtk.Main()
}
