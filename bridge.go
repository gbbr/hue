package hue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
)

type Bridge struct {
	bridgeID
	username string
}

// Pair attempts to authentiticate with the bridge. The link button on the bridge
// must be pressed before calling.
func (b *Bridge) Pair() error { return b.pairAs("gbbr/hue") }

// PairAs has the same outcome as Pair, except it allows setting the name of the
// app which will be used to authenticate.
func (b *Bridge) PairAs(appName string) error { return b.pairAs(appName) }

// IsPaired will return true if this bridge has been paired with.
func (b *Bridge) IsPaired() bool { return b.username != "" }

// addr constructs the URL of the API using the passed tokens. Some examples:
//
// 	addr()              => '<base>/api'
// 	addr("lights")      => '<base>/api/<username>/lights'
// 	addr("lights", "1") => '<base>/api/<username>/lights/1'
//
func (b Bridge) addr(tokens ...string) string {
	buf := bytes.NewBufferString(fmt.Sprintf("%sapi", b.IP))
	if len(tokens) == 0 {
		return buf.String()
	}
	buf.WriteString("/")
	buf.WriteString(b.username)
	for _, t := range tokens {
		buf.WriteString("/")
		buf.WriteString(t)
	}
	return buf.String()
}

// APIError holds detailed information about a failed API call.
// For more information see: http://www.developers.meethue.com/documentation/error-messages
type APIError struct {
	Code int    `json:"type"`
	URL  string `json:"address"`
	Msg  string `json:"description"`
}

func (e APIError) Error() string { return e.Msg }

// call calls the API at the URL specified by tokens using the given method and
// request body.
func (b Bridge) call(method, body string, tokens ...string) ([]byte, error) {
	req, err := http.NewRequest(method, b.addr(tokens...), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	slurp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var errors []struct {
		Err APIError `json:"error"`
	}
	if err := json.Unmarshal(slurp, &errors); err != nil {
		if _, ok := err.(*json.UnmarshalTypeError); !ok {
			return nil, err
		}
	}
	if len(errors) > 0 && errors[0].Err.Code != 0 {
		return nil, errors[0].Err
	}
	return slurp, nil
}

func (b *Bridge) pairAs(appName string) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`{"devicetype":"%s#%s-%s"}`, appName, host, runtime.GOOS)
	msg, err := b.call(http.MethodPost, body)
	if err != nil {
		return err
	}
	var resp []struct {
		Success struct {
			Username string `json:"username"`
		} `json:"success"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		return err
	}
	if len(resp) == 0 || resp[0].Success.Username == "" {
		return fmt.Errorf("bad response: %v", resp)
	}
	b.username = resp[0].Success.Username
	toCache(b)
	return nil
}
