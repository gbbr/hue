package hue

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

// ErrNotFound is returned when no bridge was discovered.
var ErrNotFound = errors.New("no bridge was found")

// Discover returns the (first) bridge that it finds on the local network.
func Discover() (*Bridge, error) {
	if b := fromCache(); b != nil {
		return b, nil
	}
	bid, err := discover()
	if err != nil {
		return nil, err
	}
	return &Bridge{bridgeID: bid}, err
}

// bridgeID stores discovered bridges.
type bridgeID struct {
	ID string `json:"id"`
	IP string `json:"internalipaddress"`
}

// discover runs UPNP discovery and falls back to the remote API on failure.
func discover() (bridgeID, error) {
	var (
		b   bridgeID
		err error
	)
	b, err = discoverLocal()
	if err != nil {
		log.Println("Didn't find any bridges via UPNP, attempting remote API...")
		b, err = discoverRemote()
		if err != nil {
			return b, ErrNotFound
		}
	}
	return b, err
}

var (
	mcastAddr    = &net.UDPAddr{IP: []byte{239, 255, 255, 250}, Port: 1900}
	connDeadline = 5 * time.Second
)

// discoverLocal attempts to discover any Hue bridges available via UPNP.
func discoverLocal() (bridgeID, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		return bridgeID{}, err
	}
	defer conn.Close()
	conn.WriteToUDP([]byte("M-SEARCH * HTTP/1.1\r\n"+
		"HOST: 239.255.255.250:1900\r\n"+
		"MAN: ssdp:discover\r\n"+
		"MX: 10\r\n"+
		"ST: ssdp:all\r\n"), mcastAddr)
	conn.SetDeadline(time.Now().Add(connDeadline))
	r := bufio.NewReader(conn)
	for {
		_, err := r.ReadString('\n') // HTTP/1.1 200 OK\r\n
		if err != nil {
			break
		}
		tp := textproto.NewReader(r)
		h, err := tp.ReadMIMEHeader()
		if err != nil {
			continue
		}
		v, ok := h["Location"]
		if !ok || len(v) == 0 {
			continue
		}
		bid, err := tryLocation(v[0])
		if err != nil {
			continue
		}
		return bid, err
	}
	return bridgeID{}, ErrNotFound
}

// tryLocation queries the passed url to check if it is the description of a Hue
// bridge, in which case it returns information about it. Any other outcome will
// result in an error.
func tryLocation(url string) (bridgeID, error) {
	resp, err := http.Get(url)
	if err != nil {
		return bridgeID{}, err
	}
	var body struct {
		URL    string `xml:"URLBase"`
		Device struct {
			Description string `xml:"modelDescription"`
			Name        string `xml:"modelName"`
			ID          string `xml:"serialNumber"`
		} `xml:"device"`
	}
	err = xml.NewDecoder(resp.Body).Decode(&body)
	defer resp.Body.Close()
	if err != nil {
		return bridgeID{}, err
	}
	if body.URL == "" ||
		!(strings.Contains(body.Device.Description, "Philips hue") ||
			strings.Contains(body.Device.Name, "Philips hue")) {
		return bridgeID{}, ErrNotFound
	}
	return bridgeID{
		ID: body.Device.ID,
		IP: body.URL,
	}, nil
}

var remoteAddr = "https://www.meethue.com/api/nupnp"

// discoverRemote uses the meethue.com API to discover local bridges.
func discoverRemote() (bridgeID, error) {
	resp, err := http.Get(remoteAddr)
	defer resp.Body.Close()
	if err != nil {
		return bridgeID{}, err
	}
	var b []bridgeID
	err = json.NewDecoder(resp.Body).Decode(&b)
	if err != nil {
		return bridgeID{}, err
	}
	if len(b) == 0 {
		return bridgeID{}, ErrNotFound
	}
	b[0].IP = fmt.Sprintf("http://%s/", b[0].IP) // sanitize
	return b[0], nil
}
