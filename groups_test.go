package hue

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"encoding/json"
	"reflect"
)

var testGroups = map[string]*Group{
	"g1": &Group{Name: "g1name", Type: "g1type"},
	"g2": &Group{Name: "g2name", Type: "g2type"},
}

func TestGroupsService(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()

	mb.nextResponse = testGroups

	t.Run("List", func(t *testing.T) {
		list, err := mb.b.Groups().List()
		if err != nil {
			t.Fatal(err)
		}
		if want, got := len(mb.nextResponse.(map[string]*Group)), len(list); want != got {
			t.Fatalf("expected %d entries, got %d", want, got)
		}
		if list[1].ID == "" || list[0].ID == "" {
			t.Fatalf("expected to link IDs")
		}
		if list[1].bridge != mb.b || list[0].bridge != mb.b {
			t.Fatalf("expected to link lights to bridges")
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			g, err := mb.b.Groups().Get("g1name")
			if err != nil {
				t.Fatal(err)
			}
			if g.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
		})

		t.Run("error", func(t *testing.T) {
			_, err := mb.b.Groups().Get("some bogus")
			if err != ErrNotExist {
				t.Fatalf("expected error, got %v", err)
			}
		})
	})

	t.Run("GetByID", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			g, err := mb.b.Groups().GetByID("g1")
			if err != nil {
				t.Fatal(err)
			}
			if g.bridge != mb.b {
				t.Fatal("didn't link bridge")
			}
		})

		t.Run("error", func(t *testing.T) {
			_, err := mb.b.Groups().GetByID("some bogus")
			if err != ErrNotExist {
				t.Fatalf("expected error, got %v", err)
			}
		})
	})
}

func TestGroup(t *testing.T) {
	mb := mockBridge(t)
	defer mb.teardown()
	mb.nextResponse = testGroups

	t.Run("Set", func(t *testing.T) {
		mb := mockBridge(t)
		defer mb.teardown()
		mb.nextResponse = testGroups
		g, err := mb.b.Groups().Get("g1name")
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
		if err := g.Set(want); err != nil {
			t.Fatal(err)
		}
	})
}
