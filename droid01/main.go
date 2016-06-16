// +build darwin linux

package main

import (
  "log"
    "golang.org/x/mobile/app"
    "golang.org/x/mobile/asset"
    "golang.org/x/mobile/event/lifecycle"
    "golang.org/x/mobile/event/paint"
    "golang.org/x/mobile/event/size"
    "golang.org/x/mobile/event/touch"
    "golang.org/x/mobile/exp/audio"
    "golang.org/x/mobile/gl"
)


var (
    sz size.Event
    player *audio.Player
)


func onStart(glctx gl.Context) {
    // loadScene()
}


func onStop() {
    player.Close()
}


func onPaint(glctx gl.Context, sz size.Event) {
    glctx.ClearColor(1, 0, 0, 1)
    glctx.Clear(gl.COLOR_BUFFER_BIT)
}


func mainApp(a app.App) {
    var glctx gl.Context
    for e := range a.Events() {
        switch e := a.Filter(e).(type) {
        case lifecycle.Event:
            switch e.Crosses(lifecycle.StageVisible) {
            case lifecycle.CrossOn:
                glctx, _ = e.DrawContext.(gl.Context)
                onStart(glctx)
                a.Send(paint.Event{})
            case lifecycle.CrossOff:
                onStop()
                glctx = nil
            }
        case size.Event:
            sz = e
        case paint.Event:
            if glctx == nil || e.External {
                // As we are actively painting as fast as
                // we can (usually 60 FPS), skip any paint
                // events sent by the system.
                continue
            }
            onPaint(glctx, sz)
            a.Publish()
            // Drive the animation by preparing to paint the next frame
            // after this one is shown.
            a.Send(paint.Event{}) // keep animating
        case touch.Event:
            player.Seek(0)
            player.Play()
        }
    }
}


func main() {
	rc, err := asset.Open("boing.wav")
	if err != nil {
		log.Fatal(err)
	}
	player, err = audio.NewPlayer(rc, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
    app.Main(mainApp)
}
