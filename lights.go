package hue

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrNotExist is returned when a light was not found.
var ErrNotExist = errors.New("light does not exist")

const (
	ColorLoop = "colorloop"
	NoEffect  = "none"
)

// Lights returns the service to interact with the lights on this bridge.
func (b *Bridge) Lights() *LightsService { return &LightsService{bridge: b} }

// LightsService is the service that allows interacting with the lights API
// of the bridge.
type LightsService struct{ bridge *Bridge }

// List returns a slice of all lights discovered by the bridge.
func (l *LightsService) List() ([]*Light, error) {
	all, err := l.idMap()
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

// Toggle toggles all lights "on" state.
func (l *LightsService) Toggle() error {
	return l.ForEach(func(l *Light) { l.Toggle() })
}

// ForEach traverses each light and passes it as an argument to the given function.
func (l *LightsService) ForEach(fn func(*Light)) error {
	list, err := l.idMap()
	if err != nil {
		return err
	}
	for _, l := range list {
		fn(l)
	}
	return nil
}

// GetByID returns a light by id.
func (l *LightsService) GetByID(id string) *Light {
	list, err := l.idMap()
	if err != nil {
		return &Light{error: err}
	}
	v, ok := list[id]
	if !ok {
		return &Light{error: ErrNotExist, State: &LightState{}}
	}
	return v
}

// Get returns a light by name.
func (l *LightsService) Get(name string) *Light {
	list, err := l.idMap()
	if err != nil {
		return &Light{error: err}
	}
	for _, l := range list {
		if l.Name == name {
			return l
		}
	}
	return &Light{error: ErrNotExist, State: &LightState{}}
}

// Scan searches for new lights on the system.
func (l *LightsService) Scan() error {
	_, err := l.bridge.call(http.MethodPost, nil, "lights")
	return err
}

func (l *LightsService) idMap() (map[string]*Light, error) {
	msg, err := l.bridge.call(http.MethodGet, nil, "lights")
	if err != nil {
		return nil, err
	}
	var all map[string]*Light
	err = json.Unmarshal(msg, &all)
	for id, ll := range all {
		ll.bridge = l.bridge
		ll.ID = id
		ll.State.l = ll
	}
	return all, err
}

// Light holds information about a specific light, including its state.
type Light struct {
	bridge *Bridge
	error  error

	// ID is the ID that the bridge returns for this light.
	ID string

	// UID is the unique id of the device. The MAC address of the device with
	// a unique endpoint id in the form: AA:BB:CC:DD:EE:FF:00:11-XX
	UID string `json:"uniqueid"`

	// SWVersion is an identifier for the software version running on the light.
	SWVersion string `json:"swversion"`

	// State details the state of the light.
	State *LightState `json:"state"`

	// Type is a fixed name describing the type of light.
	Type string `json:"type"`

	// Name is a unique, editable name given to the light. To change this, the
	// Rename method is provided.
	Name string `json:"name"`

	// ModelID is the hardware model of the light.
	ModelID string `json:"modelid"`

	// ManufacturerName is the manufacturer name.
	ManufacturerName string `json:"manufacturername"`
}

// On turns the light on.
func (l *Light) On() error { return l.onState(true) }

// Off turns the light off.
func (l *Light) Off() error { return l.onState(false) }

// Toggle toggles the light's "on" state.
func (l *Light) Toggle() error { return l.onState(!l.State.On) }

// Effect sets the dynamic effect of the light, can either be "none" or
// "colorloop". If set to colorloop, the light will cycle through all
// hues using the current brightness and saturation settings.
func (l *Light) Effect(name string) error {
	err := l.State.make(&LightState{Effect: name, On: true})
	if err == nil {
		l.State.Effect = name
	}
	return err
}

// Rename sets the name by which this light can be addressed.
func (l *Light) Rename(name string) error {
	if l.error != nil {
		return l.error
	}
	_, err := l.bridge.call(http.MethodPut, map[string]interface{}{
		"name": name,
	}, "lights", l.ID)
	if err == nil {
		l.Name = name
	}
	return err
}

func (l *Light) onState(b bool) error {
	err := l.State.make(&LightState{On: b})
	if err == nil {
		l.State.On = b
	}
	return err
}

// LightState holds the state of a specific light.
type LightState struct {
	l *Light

	// The dynamic effect of the light, can either be "colorloop" or "none".
	// If set to colorloop, the light will cycle through all hues using the
	// current brightness and saturation settings.
	Effect string `json:"effect,omitempty"`

	// The alert effect, is a temporary change to the bulb’s state, and has one
	// of the following values:
	//
	// 	"none" – The light is not performing an alert effect.
	// 	"select" – The light is performing one breathe cycle.
	// 	"lselect" – The light is performing breathe cycles for 15 seconds or
	// 	until an "alert": "none" command is received.
	//
	// Note that this contains the last alert sent to the light and not its
	// current state. i.e. After the breathe cycle has finished the bridge does
	// not reset the alert to "none".
	Alert string `json:"alert,omitempty"`

	// The Mired Color temperature of the light. 2012 connected lights are
	// capable of 153 (6500K) to 500 (2000K). https://en.wikipedia.org/wiki/Mired
	Ct float64 `json:"ct,omitempty"`

	// On/Off state of the light. On=true, Off=false
	On bool `json:"on"`

	// The brightness value to set the light to. Brightness is a scale from 1
	// (the minimum the light is capable of) to 254 (the maximum).
	// Note: a brightness of 1 is not off.
	// e.g. "brightness": 60 will set the light to a specific brightness
	Brightness uint8 `json:"bri,omitempty"`

	// The hue value to set light to. The hue value is a wrapping value between
	// 0 and 65535. Both 0 and 65535 are red, 25500 is green and 46920 is blue.
	// e.g. "hue": 50000 will set the light to a specific hue.
	Hue uint16 `json:"hue,omitempty"`

	// Saturation of the light. 254 is the most saturated (colored) and 0 is
	// the least saturated (white).
	Saturation uint8 `json:"sat,omitempty"`
}

// Commit reconciles the current state with the physical light.
func (ls *LightState) Commit() error { return ls.make(ls) }

func (ls *LightState) make(state *LightState) error {
	if ls.l.error != nil {
		return ls.l.error
	}
	_, err := ls.l.bridge.call(http.MethodPut, state, "lights", ls.l.ID, "state")
	return err
}
