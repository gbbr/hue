package hue

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

// addrTestsuite is a suite of tests for the internal addr function.
var addrTestsuite = map[string]struct {
	In  []string
	Out string
}{
	"no-tokens": {
		In:  []string{},
		Out: "http://1.2.3.4/api",
	},
	"one-token": {
		In:  []string{"a"},
		Out: "http://1.2.3.4/api/user/a",
	},
	"three-tokens": {
		In:  []string{"a", "b", "c"},
		Out: "http://1.2.3.4/api/user/a/b/c",
	},
}

func TestAddr(t *testing.T) {
	b := Bridge{
		bridgeID: bridgeID{IP: "http://1.2.3.4/"},
		username: "user",
	}
	for name, tt := range addrTestsuite {
		t.Run(name, func(t *testing.T) {
			if got := b.addr(tt.In...); got != tt.Out {
				t.Fatalf("expected %v, got %v", tt.Out, got)
			}
		})
	}
}

// callTestsuite is a test suite for the internal call function.
var callTestsuite = map[string]struct {
	Response []byte
	Result   []byte
	Error    error
}{
	// simple JSON response
	"success": {
		Response: []byte(`{"some": "message"}`),
		Result:   []byte(`{"some": "message"}`),
	},
	// array response
	"success-array": {
		Response: []byte(`[{"some": "message"},{"some": "message"}]`),
		Result:   []byte(`[{"some": "message"},{"some": "message"}]`),
	},
	// invalid JSON
	"invalid-json": {
		Response: []byte(`not json`),
		Error:    &json.SyntaxError{Offset: 2},
	},
	// should return parsed error
	"failure": {
		Response: []byte(`[{"error": {"type":101,"address":"a/b/c","description":"blah"}}]`),
		Error:    APIError{Code: 101, URL: "a/b/c", Msg: "blah"},
	},
}

func TestCall(t *testing.T) {
	for name, tt := range callTestsuite {
		t.Run(name, func(t *testing.T) {
			srv := serverWithResponse(string(tt.Response))
			defer srv.Close()
			msg, err := (Bridge{
				bridgeID: bridgeID{IP: srv.URL + "/"},
			}).call(http.MethodGet, "some body")
			if tt.Error != nil {
				if err == nil {
					t.Fatalf("expected error")
				}
				if _, ok := tt.Error.(APIError); ok {
					if !reflect.DeepEqual(tt.Error, err) {
						t.Fatalf("expected error %v, got %v", tt.Error, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(tt.Response, msg) {
				t.Fatalf("expected %s, got %s", tt.Response, msg)
			}
		})
	}
}
