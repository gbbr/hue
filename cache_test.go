package hue

import (
	"os"
	"os/user"
	"path"
	"reflect"
	"testing"
)

func TestToCacheFromCache(t *testing.T) {
	origCache := cacheFile
	cacheFile = ".hue-test"
	want := &Bridge{bridgeID: bridgeID{ID: "id", IP: "ip"}, username: "user"}
	toCache(want)
	b := fromCache()
	if b == nil {
		t.Fatal("expected non-nil response from cache")
	}
	if !reflect.DeepEqual(want, b) {
		t.Fatalf("expected %v, got %v", want, b)
	}
	// clean-up
	u, err := user.Current()
	if err != nil {
		t.Fatalf("failed to clean up: %v", err)
	}
	if err := os.Remove(path.Join(u.HomeDir, cacheFile)); err != nil {
		t.Fatalf("failed to clean up: %v", err)
	}
	cacheFile = origCache
}
