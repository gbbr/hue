package hue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"
)

// xmlTest holds a information about what the resulting bridgeID of a certain
// XML Response should be, as well as if it should expect to trigger an error.
type xmlTest struct {
	Response string
	Result   bridgeID
	Error    bool
}

var xmlTestsuite = map[string]xmlTest{
	// good, has valid model name
	"good": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0">
			<URLBase>http://1.2.3.4/</URLBase><device>
			<serialNumber>00178829da0d</serialNumber>
			<modelName>Philips hue bridge 2012</modelName>
			</device></root>`,
		Result: bridgeID{ID: "00178829da0d", IP: "http://1.2.3.4/"},
	},
	// good, has valid model description
	"good-with-description": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0">
			<URLBase>http://1.2.3.4/</URLBase><device>
			<serialNumber>00178829da0d</serialNumber>
			<modelDescription>Philips hue Personal Wireless Lighting</modelDescription>
			</device></root>`,
		Result: bridgeID{ID: "00178829da0d", IP: "http://1.2.3.4/"},
	},
	// good, has both model description and model model
	"good-with-name-and-description": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0">
			<URLBase>http://1.2.3.4/</URLBase><device>
			<serialNumber>00178829da0d</serialNumber>
			<modelName>Philips hue bridge 2012</modelName>
			<modelDescription>Philips hue Personal Wireless Lighting</modelDescription>
			</device></root>`,
		Result: bridgeID{ID: "00178829da0d", IP: "http://1.2.3.4/"},
	},
	// bad response (missing URL)
	"no-url": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0"><device>
			<serialNumber>00178829da0d</serialNumber></device></root>`,
		Error: true,
	},
	// bad response (bad model name)
	"not-hue": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0"><device>
			<modelName>Remote controller carpet</modelName>
			<serialNumber>00178829da0d</serialNumber></device></root>`,
		Error: true,
	},
	// bad response (bad model description)
	"also-not-hue": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0"><device>
			<modelDescription>Remote controller Window</modelDescription>
			<serialNumber>00178829da0d</serialNumber></device></root>`,
		Error: true,
	},
	// erroneous response (bad XML format)
	"error": {
		Response: `<root xmlns="urn:schemas-upnp-org:device-1-0">Base>root>`,
		Error:    true,
	},
}

// serverWithResponse returns a fake server that responds with the passed string to all
// requests.
func serverWithResponse(resp string) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(resp))
		}))
}

func TestTryLocation(t *testing.T) {
	for name, tt := range xmlTestsuite {
		t.Run(name, func(t *testing.T) {
			srv := serverWithResponse(tt.Response)
			defer srv.Close()
			b, err := tryLocation(srv.URL)
			if tt.Error {
				if err == nil {
					t.Fatalf("expected error on test '%s'", name)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error on test '%s'", name)
			}
			if !reflect.DeepEqual(b, tt.Result) {
				t.Fatalf("expected %v, got %v", b, tt.Result)
			}
		})
	}
}

// discoverRemoteTestsuite holds a suite of tests for the discoverRemote
// function.
var discoverRemoteTestsuite = map[string]struct {
	// Response is an array of bridgeIDs that will be returned as JSON from
	// the remote server.
	Response []bridgeID

	// Result is the expected bridgeID that discoverRemote should return from
	// the response.
	Result bridgeID

	// Error, when true, signals that the response should trigger an error.
	Error bool
}{
	// single bridge
	"single": {
		Response: []bridgeID{{ID: "one-two-three", IP: "1.2.3"}},
		Result:   bridgeID{ID: "one-two-three", IP: "http://1.2.3/"},
	},
	// multiple bridges
	"multiple": {
		Response: []bridgeID{
			{ID: "three-four-five", IP: "3.4.5"},
			{ID: "one-two-three", IP: "1.2.3"},
		},
		Result: bridgeID{ID: "three-four-five", IP: "http://3.4.5/"},
	},
	// no bridges
	"not-found": {Response: []bridgeID{}, Error: true},
}

func TestDiscoverRemote(t *testing.T) {
	var origRemoteAddr string
	setup := func(h http.Handler) *httptest.Server {
		origRemoteAddr = remoteAddr
		srv := httptest.NewServer(h)
		remoteAddr = srv.URL
		return srv
	}
	teardown := func(srv *httptest.Server) {
		remoteAddr = origRemoteAddr
		srv.Close()
	}
	for name, tt := range discoverRemoteTestsuite {
		t.Run(name, func(t *testing.T) {
			srv := setup(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				err := json.NewEncoder(w).Encode(tt.Response)
				if err != nil {
					t.Fatal(err)
				}
			}))
			defer teardown(srv)
			bid, err := discoverRemote()
			if tt.Error {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(bid, tt.Result) {
				t.Fatalf("expected %v, got %v", tt.Result, bid)
			}
		})
	}
}

var discoverLocalTestsuite = map[string]struct {
	Reply       string
	Result      bridgeID
	Error       bool
	XMLResponse xmlTest
}{
	// contains a location that returns a good XML response
	"good": {
		Reply:       "HTTP/1.1 200 OK\r\nHue-Bridgeid: 12345\r\nLocation: %s\r\n\r\n",
		XMLResponse: xmlTestsuite["good"],
		Result:      bridgeID{ID: "00178829da0d", IP: "http://1.2.3.4/"},
	},
	// contains two responses, second one has a good location
	"good-multi-response": {
		Reply: "HTTP/1.1 200 OK\r\nSome-Header: 12345\r\n\r\n" +
			"HTTP/1.1 200 OK\r\nHue-Bridgeid: 12345\r\nLocation: %s\r\n\r\n",
		XMLResponse: xmlTestsuite["good"],
		Result:      bridgeID{ID: "00178829da0d", IP: "http://1.2.3.4/"},
	},
	// contains a location, but the response is not a hue bridge
	"not-hue": {
		Reply:       "HTTP/1.1 200 OK\r\nLocation: %s\r\n\r\n",
		XMLResponse: xmlTestsuite["not-hue"],
		Error:       true,
	},
	// no headers
	"no-headers": {
		Reply: "HTTP/1.1 200 OK\r\n",
		Error: true,
	},
	// no response
	"no-response": {
		Reply: "",
		Error: true,
	},
}

func TestDiscoverLocal(t *testing.T) {
	origAddr := mcastAddr
	origDeadline := connDeadline
	setup := func() *net.UDPConn {
		mcastAddr = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}
		// shorten deadline
		connDeadline = time.Second
		conn, err := net.ListenUDP("udp", mcastAddr)
		if err != nil {
			t.Fatal(err)
		}
		conn.SetDeadline(time.Now().Add(time.Second))
		return conn
	}
	teardown := func(conn *net.UDPConn) {
		mcastAddr = origAddr
		connDeadline = origDeadline
		conn.Close()
	}
	for name, tt := range discoverLocalTestsuite {
		t.Run(name, func(t *testing.T) {
			conn := setup()
			defer teardown(conn)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				bid, err := discoverLocal()
				if tt.Error {
					if err == nil {
						t.Fatal("expected error")
					}
					return
				}
				if err != nil {
					t.Fatalf("got unexpected error: %v", err)
				}
				if !reflect.DeepEqual(tt.Result, bid) {
					t.Fatalf("expected %v, got %v", tt.Result, bid)
				}
			}()
			b := make([]byte, 128)
			_, raddr, err := conn.ReadFromUDP(b)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(b, []byte("M-SEARCH * HTTP/1.1")) {
				t.Fatalf("expected upnp search head, got %s", string(b))
			}
			srv := serverWithResponse(tt.XMLResponse.Response)
			_, err = conn.WriteToUDP([]byte(fmt.Sprintf(tt.Reply, srv.URL)), raddr)
			if err != nil {
				t.Fatal(err)
			}
			wg.Wait()
		})
	}
}
