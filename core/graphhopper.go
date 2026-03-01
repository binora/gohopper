package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"gohopper/core/config"
	"gohopper/core/routing"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

var validProfileName = regexp.MustCompile(`^[a-z0-9_-]+$`)

type GraphHopper struct {
	profilesByName  map[string]config.Profile
	graph           *storage.BaseGraph
	locationIndex   index.LocationIndex
	routerConfig    routing.RouterConfig
	router          *routing.Router
	encodingManager *routingutil.EncodingManager
	ghLocation      string
	fullyLoaded     bool
	properties      map[string]string
	initErr         error
}

func NewGraphHopper() *GraphHopper {
	dir := storage.NewRAMDirectory("", false)
	baseGraph := storage.NewBaseGraphBuilder(4).SetDir(dir).Build()
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
	for _, p := range cfg.Profiles {
		g.profilesByName[p.Name] = p
	}
	g.initErr = validateProfileConfig(cfg)
	return g
}

func (g *GraphHopper) ImportOrLoad() error {
	if g.initErr != nil {
		return g.initErr
	}
	return g.load()
}

func (g *GraphHopper) load() error {
	if g.ghLocation == "" {
		return errors.New("GraphHopperLocation is not specified. Call Init before")
	}
	if g.fullyLoaded {
		return errors.New("graph is already successfully loaded")
	}

	info, err := os.Stat(g.ghLocation)
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing to load yet — no graph cache directory.
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("GraphHopperLocation cannot be an existing file. Has to be either non-existing or a folder: %s", g.ghLocation)
	}

	dir := storage.NewGHDirectory(g.ghLocation, storage.DATypeRAMStore)
	props := storage.NewStorableProperties(dir)
	if !props.LoadExisting() {
		// The directory exists but has no properties file — treat as no prior import.
		return nil
	}

	em := routingutil.FromProperties(props)
	g.encodingManager = em

	bg := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetDir(dir).
		SetWithTurnCosts(em.NeedsTurnCostsSupport()).
		Build()

	if !bg.LoadExisting() {
		return fmt.Errorf("could not load existing graph from: %s", g.ghLocation)
	}

	g.graph = bg
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

func (g *GraphHopper) GetLocationIndex() index.LocationIndex {
	return g.locationIndex
}

func (g *GraphHopper) GetEncodingManager() *routingutil.EncodingManager {
	return g.encodingManager
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
	for _, p := range points {
		g.graph.Store.Bounds.Update(p.Lat, p.Lon)
	}
}

func validateProfileConfig(cfg GraphHopperConfig) error {
	seenProfiles := map[string]struct{}{}
	for _, p := range cfg.Profiles {
		if p.Name == "" {
			return errors.New("profile name cannot be empty")
		}
		if !validProfileName.MatchString(p.Name) {
			return fmt.Errorf("profile names may only contain lower case letters, numbers and underscores, given: %s", p.Name)
		}
		if p.Weighting != "" && p.Weighting != "custom" {
			return fmt.Errorf("could not create weighting for profile: '%s': weighting '%s' not supported", p.Name, p.Weighting)
		}
		if _, ok := seenProfiles[p.Name]; ok {
			return fmt.Errorf("profile names must be unique, duplicate name: '%s'", p.Name)
		}
		seenProfiles[p.Name] = struct{}{}
	}

	seenCH := map[string]struct{}{}
	for _, p := range cfg.CHProfiles {
		if _, ok := seenProfiles[p.Profile]; !ok {
			return fmt.Errorf("CH profile references unknown profile '%s'", p.Profile)
		}
		if _, ok := seenCH[p.Profile]; ok {
			return fmt.Errorf("duplicate CH reference to profile '%s'", p.Profile)
		}
		seenCH[p.Profile] = struct{}{}
	}

	lmByProfile := map[string]config.LMProfile{}
	for _, p := range cfg.LMProfiles {
		if _, ok := seenProfiles[p.Profile]; !ok {
			return fmt.Errorf("LM profile references unknown profile '%s'", p.Profile)
		}
		if _, ok := lmByProfile[p.Profile]; ok {
			return fmt.Errorf("multiple LM profiles are using the same profile '%s'", p.Profile)
		}
		lmByProfile[p.Profile] = p
	}

	for _, p := range cfg.LMProfiles {
		if p.PreparationProfile == "" {
			continue
		}
		if _, ok := seenProfiles[p.PreparationProfile]; !ok {
			return fmt.Errorf("LM profile references unknown preparation profile '%s'", p.PreparationProfile)
		}
		prepProfile, ok := lmByProfile[p.PreparationProfile]
		if !ok {
			return fmt.Errorf("unknown LM preparation profile '%s' in LM profile '%s' cannot be used as preparation_profile", p.PreparationProfile, p.Profile)
		}
		if prepProfile.PreparationProfile != "" {
			return fmt.Errorf("cannot use '%s' as preparation_profile for LM profile '%s', because it uses another profile for preparation itself", p.PreparationProfile, p.Profile)
		}
	}

	return nil
}
