package main

import (
	"log"

	"gbbr.io/hue"
)

func main() {
	b, err := hue.Discover()
	if err != nil {
		log.Fatal(err)
	}
	if !b.IsPaired() {
		// link button must be pressed for non-error response
		if err := b.Pair(); err != nil {
			log.Fatal(err)
		}
	}
	l, err := b.Lights().Get("Couch")
	if err != nil {
		log.Fatal(err)
	}
	err = l.Set(&hue.State{
		TransitionTime: 0,
		Brightness:     255,
		XY:             &[2]float64{1, 0.8},
	})
	if err != nil {
		log.Fatal(err)
	}
}
