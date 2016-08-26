package hue

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestList(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()

	mb.nextResponse = map[string]*Light{
		"l1": &Light{UID: "l1uid", Type: "l1type", State: new(LightState)},
		"l2": &Light{UID: "l2uid", Type: "l2type", State: new(LightState)},
	}
	list, err := mb.b.Lights().List()
	if err != nil {
		t.Fatal(err)
	}
	if want, got := len(mb.nextResponse.(map[string]*Light)), len(list); want != got {
		t.Fatalf("expected %d entries, got %d", want, got)
	}
	if list[1].ID != "l2" || list[0].ID != "l1" {
		t.Fatalf("expected to link IDs")
	}
	if list[1].bridge != mb.b || list[0].bridge != mb.b {
		t.Fatalf("expected to link bridges")
	}
	if list[1].State.l != list[1] || list[0].State.l != list[0] {
		t.Fatalf("expected to link states to lights")
	}
}
