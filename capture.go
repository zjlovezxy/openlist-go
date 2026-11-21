package main

import (
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
)

// captureOnStart writes a PNG of the window to $OPENLIST_CAPTURE or $BAIDUNETGO_CAPTURE a few
// seconds after it opens and quits, for looking at the interface from
// scripts. $OPENLIST_CAPTURE_AFTER sets the wait (a Go duration).
func captureOnStart(win *mygo.Window) {
	path := os.Getenv("OPENLIST_CAPTURE")
	if path == "" {
		path = os.Getenv("BAIDUNETGO_CAPTURE")
	}
	if path == "" {
		return
	}
	wait := 3 * time.Second
	after := os.Getenv("OPENLIST_CAPTURE_AFTER")
	if after == "" {
		after = os.Getenv("BAIDUNETGO_CAPTURE_AFTER")
	}
	if d, err := time.ParseDuration(after); err == nil {
		wait = d
	}
	time.AfterFunc(wait, func() {
		png, err := win.CapturePage()
		if err != nil {
			log.Printf("capture: %v", err)
		} else if err := os.WriteFile(path, png, 0o644); err != nil {
			log.Printf("capture: %v", err)
		}
		mygo.App.Quit()
	})
}
