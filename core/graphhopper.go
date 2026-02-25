package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gohopper/core/config"
	"gohopper/core/routing"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type GraphHopper struct {
	profilesByName map[string]config.Profile
	graph          *storage.BaseGraph
	locationIndex  *index.LocationIndex
	routerConfig   routing.RouterConfig
	router         *routing.Router
	ghLocation     string
	fullyLoaded    bool
	properties     map[string]string
}

func NewGraphHopper() *GraphHopper {
	baseGraph := storage.NewBaseGraph()
	locationIndex := index.NewLocationIndex()
	routerConfig := routing.NewRouterConfig()
	return &GraphHopper{
		profilesByName: make(map[string]config.Profile),
		graph:          baseGraph,
		locationIndex:  locationIndex,
		routerConfig:   routerConfig,
		router:         routing.NewRouter(baseGraph, locationIndex, routerConfig),
		properties:     make(map[string]string),
	}
}

func (g *GraphHopper) Init(cfg GraphHopperConfig) *GraphHopper {
	g.ghLocation = cfg.GetString("graph.location", "graph-cache")
	for _, p := range cfg.GetProfiles() {
		g.profilesByName[p.Name] = p
	}
	return g
}

func (g *GraphHopper) ImportOrLoad() error {
	if g.ghLocation == "" {
		g.ghLocation = "graph-cache"
	}
	if err := os.MkdirAll(g.ghLocation, 0o755); err != nil {
		return err
	}
	// This placeholder marker lets contributors see where cache compatibility work plugs in.
	markerPath := filepath.Join(g.ghLocation, "gohopper.marker")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		if err := os.WriteFile(markerPath, []byte("placeholder cache marker; replace with GH11 binary-compatible cache files\n"), 0o644); err != nil {
			return err
		}
	}
	g.properties["datareader.import.date"] = time.Now().UTC().Format(time.RFC3339)
	g.properties["datareader.data.date"] = ""
	g.fullyLoaded = true
	return nil
}

func (g *GraphHopper) Route(request webapi.GHRequest) webapi.GHResponse {
	if !g.fullyLoaded {
		resp := webapi.NewGHResponse()
		resp.AddError(fmt.Errorf("GraphHopper is not fully loaded"))
		return resp
	}
	return g.router.Route(request)
}

func (g *GraphHopper) GetBaseGraph() *storage.BaseGraph {
	return g.graph
}

func (g *GraphHopper) GetLocationIndex() *index.LocationIndex {
	return g.locationIndex
}

func (g *GraphHopper) GetProfiles() []config.Profile {
	profiles := make([]config.Profile, 0, len(g.profilesByName))
	for _, p := range g.profilesByName {
		profiles = append(profiles, p)
	}
	return profiles
}

func (g *GraphHopper) IsFullyLoaded() bool {
	return g.fullyLoaded
}

func (g *GraphHopper) GetProperties() map[string]string {
	return g.properties
}

func (g *GraphHopper) SetBounds(points []util.GHPoint) {
	g.graph.SetBounds(points)
}
