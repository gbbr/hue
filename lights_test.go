package hue

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testLights = map[string]*Light{
	"l1": &Light{UID: "l1uid", Name: "l1name", Type: "l1type"},
	"l2": &Light{UID: "l2uid", Name: "l2name", Type: "l2type"},
}

// serviceTestTools is a toolset that can be used to test a service on the bridge.
type serviceTestTools struct {
	// b is the bridge that should be used in tests.
	b *Bridge
	// srv is the test server that will act as the bridge API. It must be closed
	// when the test completes
	srv *httptest.Server
	// nextResponse is the next response that the server will provide.
	nextResponse interface{}
	// lastMethod is the last request method that the server received.
	lastMethod string
	// lastBody is the last request body that the server received.
	lastBody io.Reader
	// lastPath is the last path that was requested on the server.
	lastPath string
}

func (st *serviceTestTools) teardown() { st.srv.Close() }

// mockBridge returns a set of tools that allows testing services on the bridge.
func mockBridge(t *testing.T) *serviceTestTools {
	stt := new(serviceTestTools)
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			stt.lastMethod = r.Method
			stt.lastBody = r.Body
			stt.lastPath = r.URL.Path
			if err := json.NewEncoder(w).Encode(stt.nextResponse); err != nil {
				t.Fatal(err)
			}
		},
	))
	stt.b = &Bridge{
		bridgeID: bridgeID{ID: "bridge_id", IP: srv.URL + "/"},
		username: "bridge_username",
	}
	stt.srv = srv
	return stt
}

func TestLightsService(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()
	mb.nextResponse = testLights

	t.Run("List", func(t *testing.T) {
		list, err := mb.b.Lights().List()
		if err != nil {
			t.Fatal(err)
		}
		if want, got := len(mb.nextResponse.(map[string]*Light)), len(list); want != got {
			t.Fatalf("expected %d entries, got %d", want, got)
		}
		if list[1].ID == "" || list[0].ID == "" {
			t.Fatalf("expected to link IDs")
		}
		if list[1].bridge != mb.b || list[0].bridge != mb.b {
			t.Fatalf("expected to link lights to bridges")
		}
		if list[1].State.l == nil || list[0].State.l == nil {
			t.Fatalf("expected to link states to lights")
		}
	})

	t.Run("ForEach", func(t *testing.T) {
		var i int
		err := mb.b.Lights().ForEach(func(l *Light) {
			i++
			if _, ok := testLights[l.ID]; !ok {
				t.Fatal("invalid entry or did not link IDs")
			}
			if l.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if i != len(testLights) {
			t.Fatal("did not go through all lights")
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			l := mb.b.Lights().Get("l1name")
			if l.UID != testLights["l1"].UID {
				t.Fatalf("expected %v, got %v", l, testLights["l1"])
			}
			if l.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
			if l.State.l != l {
				t.Fatal("didn't link state")
			}
		})

		t.Run("error", func(t *testing.T) {
			l := mb.b.Lights().Get("some bogus")
			if l.error != ErrNotExist {
				t.Fatalf("expected ErrNotExist, instead got %v", l.error)
			}
		})
	})

	t.Run("GetByID", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			l := mb.b.Lights().GetByID("l1")
			if l.UID != testLights["l1"].UID {
				t.Fatalf("expected %v, got %v", l, testLights["l1"])
			}
			if l.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
			if l.State.l != l {
				t.Fatal("didn't link state")
			}
		})

		t.Run("error", func(t *testing.T) {
			l := mb.b.Lights().GetByID("some bogus")
			if l.error != ErrNotExist {
				t.Fatalf("expected ErrNotExist, instead got %v", l.error)
			}
		})
	})
}

func TestLight(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()
	mb.nextResponse = testLights

	t.Run("On", func(t *testing.T) {
		l := mb.b.Lights().Get("l1name")
		if l.State.On {
			t.Fatal("expected light to be off")
		}
		if err := l.On(); err != nil {
			t.Fatal(err)
		}
		if !l.State.On {
			t.Fatal("expected light to turn on")
		}
	})

	t.Run("Off", func(t *testing.T) {
		l := mb.b.Lights().Get("l1name")
		if l.State.On {
			t.Fatal("expected light to be off")
		}
		if err := l.On(); err != nil {
			t.Fatal(err)
		}
		if !l.State.On {
			t.Fatal("expected light to turn on")
		}
		if err := l.Off(); err != nil {
			t.Fatal(err)
		}
		if l.State.On {
			t.Fatal("expected light to be off")
		}
	})

	t.Run("Toggle", func(t *testing.T) {
		l := mb.b.Lights().Get("l1name")
		if err := l.Toggle(); err != nil {
			t.Fatal(err)
		}
		if !l.State.On {
			t.Fatal("expected light to turn on")
		}
		if err := l.Toggle(); err != nil {
			t.Fatal(err)
		}
		if l.State.On {
			t.Fatal("expected light to be off")
		}
	})

	t.Run("Effect", func(t *testing.T) {
		l := mb.b.Lights().Get("l1name")
		if err := l.Effect("asd"); err != nil {
			t.Fatal(err)
		}
		if l.State.Effect != "asd" {
			t.Fatalf("expected effect 'asd', got '%s'", l.State.Effect)
		}
	})

	t.Run("Rename", func(t *testing.T) {
		l := mb.b.Lights().Get("l1name")
		if err := l.Rename("asd"); err != nil {
			t.Fatal(err)
		}
		if l.Name != "asd" {
			t.Fatalf("expected name to become 'asd', got '%s'", l.Name)
		}
	})
}

func TestLightState(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()
	mb.nextResponse = testLights

	l1 := mb.b.Lights().GetByID("l1")
	l1.State.On = true
	l1.State.Saturation = 123
	l1.State.Commit()
	if mb.lastMethod != http.MethodPut {
		t.Fatalf("expected PUT, got %s", mb.lastMethod)
	}
	if mb.lastPath != "/api/bridge_username/lights/l1/state" {
		t.Fatalf("expected PUT, got %s", mb.lastMethod)
	}
}
