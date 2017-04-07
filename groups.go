package hue

import (
	"encoding/json"
	"net/http"
)

func (b *Bridge) Groups() *GroupsService { return &GroupsService{b} }

type GroupsService struct{ bridge *Bridge }

func (gs *GroupsService) List() ([]*Group, error) {
	all, err := gs.idMap()
	if err != nil {
		return nil, err
	}
	list := make([]*Group, 0, len(all))
	for _, g := range all {
		list = append(list, g)
	}
	return list, nil
}

func (gs *GroupsService) Get(name string) (*Group, error) {
	list, err := gs.idMap()
	if err != nil {
		return nil, err
	}
	for _, g := range list {
		if g.Name == name {
			return g, nil
		}
	}
	return nil, ErrNotExist
}

func (gs *GroupsService) GetByID(id string) (*Group, error) {
	list, err := gs.idMap()
	if err != nil {
		return nil, err
	}
	g, ok := list[id]
	if !ok {
		return nil, ErrNotExist
	}
	return g, nil
}

func (gs *GroupsService) idMap() (map[string]*Group, error) {
	r, err := gs.bridge.call(http.MethodGet, nil, "groups")
	if err != nil {
		return nil, err
	}
	var all map[string]*Group
	err = json.Unmarshal(r, &all)
	for id, g := range all {
		g.ID = id
		g.bridge = gs.bridge
	}
	return all, err
}

type Group struct {
	bridge *Bridge

	ID     string
	Name   string      `json:"name"`
	Lights []string    `json:"lights"`
	Type   string      `json:"type"`
	Action *LightState `json:"action"`
}

// TODO(gbbr):
func (g *GroupsService) Create() {}

// TODO(gbbr):
func (g *Group) Rename()    {}
func (g *Group) SetLights() {}
func (g *Group) Set()       {}
func (g *Group) Delete()    {}
func (g *Group) On()        {}
func (g *Group) Off()       {}
func (g *Group) Toggle()    {}

func (g *Group) ForEachLight(fn func(l *Light)) error {
	all, err := g.bridge.Lights().idMap()
	if err != nil {
		return err
	}
	for _, l := range g.Lights {
		light, ok := all[l]
		if !ok {
			continue
		}
		fn(light)
	}
	return nil
}
