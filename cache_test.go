package hue

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/mitchellh/go-homedir"
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
	homeDir, err := homedir.Dir()
	if err != nil {
		t.Fatalf("failed to clean up: %v", err)
	}
	if err := os.Remove(path.Join(homeDir, cacheFile)); err != nil {
		t.Fatalf("failed to clean up: %v", err)
	}
	cacheFile = origCache
}
