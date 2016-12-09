package hue

import (
	"encoding/json"
	"net/http"
)

type Group struct {
	bridge *Bridge

	ID     string
	Name   string
	Lights []string
	Type   string
	Class  string
	Action LightState
}

// Groups returns the service to interact with the groups on this bridge.
func (b *Bridge) Groups() *GroupsService { return &GroupsService{bridge: b} }

// GroupsService is the service that allows interacting with the groups API
// of the bridge.
type GroupsService struct{ bridge *Bridge }

func (g *GroupsService) idMap() (groups map[string]*Group, err error) {
	msg, err := g.bridge.call(http.MethodGet, nil, "groups")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(msg, &groups)
	if err != nil {
		return groups, err
	}

	for id, group := range groups {
		group.bridge = g.bridge
		group.ID = id
	}

	return groups, nil
}

// List returns a slice of all groups discovered by the bridge.
func (g GroupsService) List() ([]*Group, error) {
	groups, err := g.idMap()
	if err != nil {
		return nil, err
	}
	list := make([]*Group, 0, len(groups))
	for _, ll := range groups {
		list = append(list, ll)
	}
	return list, nil
}

// Get returns a light by name.
func (g *GroupsService) Get(name string) (*Group, error) {
	list, err := g.idMap()
	if err != nil {
		return nil, err
	}
	for _, group := range list {
		if group.Name == name {
			return group, nil
		}
	}
	return nil, ErrNotExist
}

// GetByID returns a light by id.
func (g *GroupsService) GetByID(id string) (*Group, error) {
	list, err := g.idMap()
	if err != nil {
		return nil, ErrNotExist
	}
	v, ok := list[id]
	if !ok {
		return nil, ErrNotExist
	}
	return v, nil
}

// On turns the group on.
func (g *Group) On() error {
	return g.Set(&State{On: true})
}

// Off turns the group off.
func (g *Group) Off() error {
	return g.Set(&State{On: false})
}

// Toggle toggles a group on/off.
func (g *Group) Toggle() error {
	return g.Set(&State{On: !g.Action.On})
}

// Set sets the new state of the group.
func (g *Group) Set(s *State) error {

	_, err := g.bridge.call(http.MethodPut, s, "groups", g.ID, "action")
	if err != nil {
		return err
	}

	r, err := g.bridge.call(http.MethodGet, nil, "groups", g.ID)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r, g); err != nil {
		return err
	}
	return err
}