package main

import (
	"log"

	"github.com/gbbr/hue"
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
	if err := l.Toggle(); err != nil {
		log.Fatal(err)
	}
}
