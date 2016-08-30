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
func (l *LightsService) GetByID(id string) (*Light, error) {
	list, err := l.idMap()
	if err != nil {
		return nil, ErrNotExist
	}
	v, ok := list[id]
	if !ok {
		return nil, ErrNotExist
	}
	return v, nil
}

// Get returns a light by name.
func (l *LightsService) Get(name string) (*Light, error) {
	list, err := l.idMap()
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		if l.Name == name {
			return l, nil
		}
	}
	return nil, ErrNotExist
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
	}
	return all, err
}

// Light holds information about a specific light, including its state.
type Light struct {
	bridge *Bridge

	// ID is the ID that the bridge returns for this light.
	ID string

	// UID is the unique id of the device. The MAC address of the device with
	// a unique endpoint id in the form: AA:BB:CC:DD:EE:FF:00:11-XX
	UID string `json:"uniqueid"`

	// SWVersion is an identifier for the software version running on the light.
	SWVersion string `json:"swversion"`

	// State details the state of the light.
	State LightState `json:"state"`

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
func (l *Light) On() error { return l.Set(&State{On: true}) }

// Off turns the light off.
func (l *Light) Off() error {
	_, err := l.bridge.call(http.MethodPut, map[string]bool{
		"on": false,
	}, "lights", l.ID, "state")
	if err == nil {
		l.State.On = false
	}
	return err
}

// Toggle toggles a light on/off.
func (l *Light) Toggle() error {
	if l.State.On {
		return l.Off()
	}
	return l.On()
}

// Rename sets the name by which this light can be addressed.
func (l *Light) Rename(name string) error {
	_, err := l.bridge.call(http.MethodPut, map[string]string{
		"name": name,
	}, "lights", l.ID)
	if err == nil {
		l.Name = name
	}
	return err
}

// Set sets the new state of the light. Note that Set can not turn the light off.
// In order to do that, use the provided Off method.
func (l *Light) Set(s *State) error {
	_, err := l.bridge.call(http.MethodPut, s, "lights", l.ID, "state")
	if err != nil {
		return err
	}
	r, err := l.bridge.call(http.MethodGet, nil, "lights", l.ID)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, l); err != nil {
		return err
	}
	return err
}

// State holds a structure that is used to update a light's state.
type State struct {
	// On, when true, will turn a light on.
	On bool `json:"on,omitempty"`

	// The brightness value to set the light to. Brightness is a scale from 1
	// (the minimum the light is capable of) to 254 (the maximum).
	// Note: a brightness of 1 is not off.
	// e.g. "brightness": 60 will set the light to a specific brightness
	Brightness uint8 `json:"bri,omitempty"`

	// The hue value to set light to. The hue value is a wrapping value between
	// 0 and 65535. Both 0 and 65535 are red, 25500 is green and 46920 is blue.
	// e.g. “brightness”: 60 will set the light to a specific brightness
	Hue uint16 `json:"hue,omitempty"`

	// Saturation of the light. 254 is the most saturated (colored) and 0 is
	// the least saturated (white).
	Saturation uint8 `json:"sat,omitempty"`

	// The x and y coordinates of a color in CIE color space. The first entry
	// is the x coordinate and the second entry is the y coordinate. Both x and
	// y must be between 0 and 1. If the specified coordinates are not in the
	// CIE color space, the closest color to the coordinates will be chosen.
	XY *[2]float64 `json:"xy,omitempty"`

	// The Mired Color temperature of the light. 2012 connected lights are
	// capable of 153 (6500K) to 500 (2000K).
	Ct float64 `json:"ct,omitempty"`

	// The alert effect, is a temporary change to the bulb’s state, and has one
	// of the following values:
	// 	"none" – The light is not performing an alert effect.
	// 	"select" – The light is performing one breathe cycle.
	// 	"lselect" – The light is performing breathe cycles for 15 seconds or
	// 		until an "alert": "none" command is received.
	// Note that this contains the last alert sent to the light and not its
	// current state. i.e. After the breathe cycle has finished the bridge does
	// not reset the alert to "none".
	Alert string `json:"alert,omitempty"`

	// The dynamic effect of the light. Currently "none" and "colorloop" are
	// supported. Other values will generate an error of type 7. Setting the
	// effect to colorloop will cycle through all hues using the current
	// brightness and saturation settings.
	Effect string `json:"effect,omitempty"`

	// The duration of the transition from the light’s current state to the new
	// state. This is given as a multiple of 100ms and defaults to 4 (400ms).
	// For example, setting transitiontime:10 will make the transition last 1
	// second.
	TransitionTime uint16 `json:"transitiontime,omitempty"`

	// As of 1.7. Increments or decrements the value of the brightness. It is
	// ignored if the Brightness field is provided. Any ongoing brightness
	// transition is stopped. Setting a value of 0 also stops any ongoing
	// transition.
	BriInc int `json:"bri_inc,omitempty"`

	// As of 1.7. Increments or decrements the value of Saturation. It is
	// ignored if the Saturation field is provided. Any ongoing Saturation
	// transition is stopped. Setting a value of 0 also stops any ongoing
	// transition.
	SatInc int `json:"sat_inc,omitempty"`

	// As of 1.7. Increments or decrements the value of the Hue. It is ignored
	// if the Hue field is provided. Any ongoing color transition is stopped.
	// Setting a value of 0 also stops any ongoing transition. Note if the
	// resulting values are < 0 or > 65535 the result is wrapped. For example:
	// HueInc with a value of 1 will result in 0 when applied to a Hue of 65535.
	// HueInc with a value of -2 will result in 65534 when applied to a Hue of 0.
	HueInc int `json:"hue_inc,omitempty"`

	// As of 1.7. Increments or decrements the value of Ct. It is ignored if
	// the Ct field is provided. Any ongoing color transition is stopped.
	// Setting a value of 0 also stops any ongoing transition.
	CtInc int `json:"ct_inc,omitempty"`

	// As of 1.7. Increments or decrements the value of the XY. It is ignored
	// if the XY attribute is provided. Any ongoing color transition is stopped.
	// Setting a value of 0 also stops any ongoing transition. Will stop at it's
	// gamut boundaries. Max value [0.5, 0.5].
	XYInc *[2]float64 `json:"xy_inc,omitempty"`
}

// LightState holds the active state of a specific light
type LightState struct {
	// On/Off state of the light. On=true, Off=false
	On bool `json:"on"`

	// Brightness of the light. This is a scale from the minimum brightness the
	// light is capable of, 1, to the maximum capable brightness, 254.
	Brightness uint8 `json:"bri"`

	// Hue of the light. This is a wrapping value between 0 and 65535. Both 0
	// and 65535 are red, 25500 is green and 46920 is blue.
	Hue uint16 `json:"hue"`

	// Hue of the light. This is a wrapping value between 0 and 65535. Both 0
	// and 65535 are red, 25500 is green and 46920 is blue.
	Saturation uint8 `json:"sat"`

	// The x and y coordinates of a color in CIE color space. The first entry
	// is the x coordinate and the second entry is the y coordinate. Both x and
	// y are between 0 and 1.
	XY [2]float64 `json:"xy"`

	// The Mired Color temperature of the light. 2012 connected lights are
	// capable of 153 (6500K) to 500 (2000K).
	ColorTemp float64 `json:"ct"`

	// The alert effect, which is a temporary change to the bulb’s state. This
	// can take one of the following values:
	// 	"none" – The light is not performing an alert effect.
	// 	"select" – The light is performing one breathe cycle.
	// 	"lselect" – The light is performing breathe cycles for 15 seconds or
	// 		until an "alert": "none" command is received.
	// Note that this contains the last alert sent to the light and not its
	// current state. i.e. After the breathe cycle has finished the bridge does
	// not reset the alert to "none".
	Alert string `json:"alert"`

	// The dynamic effect of the light, can either be "none" or "colorloop".
	Effect string `json:"effect"`

	// Indicates the color mode in which the light is working, this is the last
	// command type it received. Values are "hs" for Hue and Saturation, "xy"
	// for XY and "ct" for Color Temperature. This parameter is only present
	// when the light supports at least one of the values.
	ColorMode string `json:"colormode"`

	// Indicates if a light can be reached by the bridge.
	Reachable bool `json:"reachable"`
}
