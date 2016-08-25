package hue

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
)

// cacheFile stores the name of the file where bridge cache will be stored.
var cacheFile = ".hue"

// cacheBridge holds the format of the contents of the cache file.
type cachedBridge struct{ ID, IP, Username string }

// toCache writes bridge b to the cache file.
func toCache(b *Bridge) {
	u, err := user.Current()
	if err != nil {
		log.Printf("could not cache: %v", err)
		return
	}
	data, err := json.Marshal(cachedBridge{ID: b.ID, IP: b.IP, Username: b.username})
	if err != nil {
		log.Printf("could not cache: %v", err)
		return
	}
	err = ioutil.WriteFile(path.Join(u.HomeDir, cacheFile), data, 0666)
	if err != nil {
		log.Printf("could not cache: %v", err)
		return
	}
}

// fromCache returns the cached bridge or nil otherwise.
func fromCache() *Bridge {
	u, err := user.Current()
	if err != nil {
		log.Printf("could not retrieve cache: %v", err)
		return nil
	}
	data, err := ioutil.ReadFile(path.Join(u.HomeDir, cacheFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Printf("could not retrieve cache: %v", err)
		return nil
	}
	var b cachedBridge
	if err := json.Unmarshal(data, &b); err != nil {
		log.Printf("could not retrieve cache: %v", err)
		return nil
	}
	return &Bridge{
		bridgeID: bridgeID{ID: b.ID, IP: b.IP},
		username: b.Username,
	}
}
