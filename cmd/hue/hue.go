package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/gbbr/hue"
)

func main() {
	b, err := hue.Discover()
	if err != nil {
		log.Fatalf("%v", err)
	}
	if !b.IsPaired() {
		fmt.Println("Press link button on Bridge, then press ENTER to continue...")
		r := bufio.NewReader(os.Stdin)
		r.ReadByte()
		err = b.Pair()
		if err != nil {
			log.Fatalf("%#v", err)
		}
	}

	err = b.Lights().Get("Couch").Switch()
	if err != nil {
		log.Printf("%#v", err)
	}
}
