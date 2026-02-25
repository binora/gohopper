package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gohopper/core/config"
	"gohopper/core/routing"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

var validProfileName = regexp.MustCompile(`^[a-z0-9_-]+$`)

type GraphHopper struct {
	profilesByName map[string]config.Profile
	graph          *storage.BaseGraph
	locationIndex  *index.LocationIndex
	routerConfig   routing.RouterConfig
	router         *routing.Router
	ghLocation     string
	fullyLoaded    bool
	properties     map[string]string
	initErr        error
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
	g.initErr = validateProfileConfig(cfg)
	return g
}

func (g *GraphHopper) ImportOrLoad() error {
	if g.initErr != nil {
		return g.initErr
	}
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

func validateProfileConfig(cfg GraphHopperConfig) error {
	seenProfiles := map[string]struct{}{}
	for _, p := range cfg.GetProfiles() {
		if p.Name == "" {
			return errors.New("Profile name cannot be empty")
		}
		if !validProfileName.MatchString(p.Name) {
			return fmt.Errorf("Profile names may only contain lower case letters, numbers and underscores, given: %s", p.Name)
		}
		if p.Weighting != "" && p.Weighting != "custom" {
			return fmt.Errorf("Could not create weighting for profile: '%s': Weighting '%s' not supported", p.Name, p.Weighting)
		}
		if _, ok := seenProfiles[p.Name]; ok {
			return fmt.Errorf("Profile names must be unique. Duplicate name: '%s'", p.Name)
		}
		seenProfiles[p.Name] = struct{}{}
	}

	seenCH := map[string]struct{}{}
	for _, p := range cfg.GetCHProfiles() {
		if _, ok := seenProfiles[p.Profile]; !ok {
			return fmt.Errorf("CH profile references unknown profile '%s'", p.Profile)
		}
		if _, ok := seenCH[p.Profile]; ok {
			return fmt.Errorf("Duplicate CH reference to profile '%s'", p.Profile)
		}
		seenCH[p.Profile] = struct{}{}
	}

	lmByProfile := map[string]config.LMProfile{}
	for _, p := range cfg.GetLMProfiles() {
		if _, ok := seenProfiles[p.Profile]; !ok {
			return fmt.Errorf("LM profile references unknown profile '%s'", p.Profile)
		}
		if _, ok := lmByProfile[p.Profile]; ok {
			return fmt.Errorf("Multiple LM profiles are using the same profile '%s'", p.Profile)
		}
		lmByProfile[p.Profile] = p
	}

	for _, p := range cfg.GetLMProfiles() {
		if p.PreparationProfile == "" {
			continue
		}
		if _, ok := seenProfiles[p.PreparationProfile]; !ok {
			return fmt.Errorf("LM profile references unknown preparation profile '%s'", p.PreparationProfile)
		}
		prepProfile, ok := lmByProfile[p.PreparationProfile]
		if !ok {
			return fmt.Errorf("Unknown LM preparation profile '%s' in LM profile '%s' cannot be used as preparation_profile", p.PreparationProfile, p.Profile)
		}
		if prepProfile.PreparationProfile != "" {
			return fmt.Errorf("Cannot use '%s' as preparation_profile for LM profile '%s', because it uses another profile for preparation itself.", p.PreparationProfile, p.Profile)
		}
	}

	return nil
}
