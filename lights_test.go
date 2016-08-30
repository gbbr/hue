package hue

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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
			l, err := mb.b.Lights().Get("l1name")
			if err != nil {
				t.Fatal(err)
			}
			if l.UID != testLights["l1"].UID {
				t.Fatalf("expected %v, got %v", l, testLights["l1"])
			}
			if l.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
		})

		t.Run("error", func(t *testing.T) {
			_, err := mb.b.Lights().Get("some bogus")
			if err != ErrNotExist {
				t.Fatalf("expected error, got %v", err)
			}
		})
	})

	t.Run("GetByID", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			l, err := mb.b.Lights().GetByID("l1")
			if err != nil {
				t.Fatal(err)
			}
			if l.UID != testLights["l1"].UID {
				t.Fatalf("expected %v, got %v", l, testLights["l1"])
			}
			if l.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
		})

		t.Run("error", func(t *testing.T) {
			_, err := mb.b.Lights().GetByID("some bogus")
			if err != ErrNotExist {
				t.Fatalf("expected error, got %v", err)
			}
		})
	})
}

func TestLight(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()
	mb.nextResponse = testLights

	t.Run("Rename", func(t *testing.T) {
		l, err := mb.b.Lights().Get("l1name")
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Rename("asd"); err != nil {
			t.Fatal(err)
		}
		if l.Name != "asd" {
			t.Fatalf("expected name to become 'asd', got '%s'", l.Name)
		}
	})

	t.Run("Set", func(t *testing.T) {
		mb := mockBridge(t)
		defer mb.teardown()
		mb.nextResponse = testLights
		l, err := mb.b.Lights().Get("l1name")
		if err != nil {
			t.Fatal(err)
		}

		want := &State{Alert: "alert123"}
		srv := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPut:
					// on PUT request check that it's correct and return
					// a random success string
					s := new(State)
					if err := json.NewDecoder(r.Body).Decode(s); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(s, want) {
						t.Fatalf("expected %v, got %v", want, s)
					}
					if err := json.NewEncoder(w).Encode(map[string]string{"success": "true"}); err != nil {
						t.Fatal(err)
					}
				case http.MethodGet:
					// on GET request return the new, altered state of the light
					if err := json.NewEncoder(w).Encode(Light{
						State: LightState{Alert: want.Alert},
					}); err != nil {
						t.Fatal(err)
					}
				default:
					t.Fatal("unexpected request")
				}
			}))
		defer srv.Close()

		mb.b.bridgeID.IP = srv.URL + "/"
		if err := l.Set(want); err != nil {
			t.Fatal(err)
		}
		if l.State.Alert != "alert123" {
			t.Fatalf("expected 'Alert' to be 'alert123', got '%s'", l.State.Alert)
		}
	})
}
