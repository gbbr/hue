[![hue](https://godoc.org/github.com/gbbr/hue?status.svg)](https://godoc.org/github.com/gbbr/hue) 
[![travis-ci](https://travis-ci.org/gbbr/hue.svg?branch=master)](https://travis-ci.org/gbbr/hue) 

# ![](http://i1253.photobucket.com/albums/hh588/gbbr/light-bulb-outline_318-50593%20copy_zpsexky6j6x.jpg) hue

hue is a small package for interacting with a [Phillips Hue](http://www.meethue.com/) bridge. It facilitates discovery, authentication and control of up to one brige in your local network.


### hello world

To discover a bridge, pair with it and turn on a light named _"Desk"_, the program would be:

```go
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
    	// link button must be pressed before calling
    	if err := b.Pair(); err != nil {
    		log.Fatal(err)
    	}
    }
    light, err := b.Lights().Get("Desk")
    if err != nil {
    	log.Fatal(err)
    }
    if err := light.On(); err != nil {
    	log.Fatal(err)
    }
}
```
hue attempts to discover a bridge using UPnP (for up to 5 seconds) or by falling back to a remote [endpoint](https://www.meethue.com/api/nupnp). On subsequent calls, discovery and pairing data is readily available from cache stored on the file system in `~/.hue`. It is best practice to check that the device has not already been paired with before calling `Pair`, for performance reasons.

Shall you ever need to reset the cache, simply remove the file.

There are still aspects of the API to be implemented, but the individual light interaction is complete. To see the full documentation, visit our [godoc](https://godoc.org/github.com/gbbr/hue) page.
 
