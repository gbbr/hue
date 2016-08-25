package hue

import (
	"encoding/json"
	"net/http"
	"os"
)

// Lights returns the Lights API.
func (b *Bridge) Lights() *LightsService { return &LightsService{bridge: b} }

type LightsService struct{ bridge *Bridge }

// List returns a slice of all lights discovered by the bridge.
func (l *LightsService) List() ([]*Light, error) {
	all, err := l.byID()
	if err != nil {
		return nil, err
	}
	list := make([]*Light, 0, len(all))
	for _, ll := range all {
		list = append(list, ll)
	}
	return list, nil
}

// On turns all lights on.
func (l *LightsService) On() error {
	return l.ForEach(func(l *Light) { l.On() })
}

// Off turns all lights off.
func (l *LightsService) Off() error {
	return l.ForEach(func(l *Light) { l.Off() })
}

// Switch toggles all lights on state.
func (l *LightsService) Switch() error {
	return l.ForEach(func(l *Light) { l.Switch() })
}

// ForEach traverses each light and passes it as an argument to the given function.
func (l *LightsService) ForEach(fn func(*Light)) error {
	list, err := l.byID()
	if err != nil {
		return err
	}
	for _, l := range list {
		fn(l)
	}
	return nil
}

// ID returns a light by id.
func (l *LightsService) ID(id string) *Light {
	list, err := l.byID()
	if err != nil {
		return &Light{error: err}
	}
	return list[id]
}

// Name returns a light by name.
func (l *LightsService) Name(name string) *Light {
	list, err := l.byID()
	if err != nil {
		return &Light{error: err}
	}
	for _, l := range list {
		if l.Name == name {
			return l
		}
	}
	return &Light{error: os.ErrNotExist}
}

func (l *LightsService) byID() (map[string]*Light, error) {
	msg, err := l.bridge.call(http.MethodGet, "", "lights")
	if err != nil {
		return nil, err
	}
	var all map[string]*Light
	err = json.Unmarshal(msg, &all)
	for id, ll := range all {
		ll.bridge = l.bridge
		ll.ID = id
	}
	return all, err
}

type Light struct {
	bridge *Bridge
	error  error

	ID               string
	UID              string     `json:"uniqueid"`
	SWVersion        string     `json:"swversion"`
	State            LightState `json:"state"`
	Type             string     `json:"type"`
	Name             string     `json:"name"`
	ModelID          string     `json:"modelid"`
	ManufacturerName string     `json:"manufacturername"`
}

type LightState struct {
	Effect     string  `json:"effect,omitempty"`
	Alert      string  `json:"alert,omitempty"`
	Ct         float64 `json:"ct,omitempty"`
	ColorMode  string  `json:"colormode,omitempty"`
	Reachable  bool    `json:"reachable,omitempty"`
	On         bool    `json:"on"`
	Brightness float64 `json:"bri,omitempty"`
	Hue        float64 `json:"hue,omitempty"`
	Saturation float64 `json:"sat,omitempty"`
}

// On turns the light on.
func (l *Light) On() error { return l.onState(true) }

// Off turns the light off.
func (l *Light) Off() error { return l.onState(false) }

// Switch toggles the light's on state.
func (l *Light) Switch() error { return l.onState(!l.State.On) }

func (l *Light) onState(b bool) error {
	if l.error != nil {
		return l.error
	}
	body, err := json.Marshal(LightState{On: b})
	if err != nil {
		return err
	}
	_, err = l.bridge.call(http.MethodPut, string(body), "lights", l.ID, "state")
	if err == nil {
		l.State.On = b
	}
	return err
}
