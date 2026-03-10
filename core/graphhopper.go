package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"gohopper/core/config"
	"gohopper/core/reader/osm"
	"gohopper/core/routing"
	"gohopper/core/routing/ev"
	"gohopper/core/routing/parsers"
	"gohopper/core/routing/subnetwork"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
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
	osmFile         string
	storeOnFlush    bool
	fullyLoaded     bool
	ignoredHighways []string
	minNetworkSize  int
	properties      map[string]string
	initErr         error
}

func NewGraphHopper() *GraphHopper {
	return &GraphHopper{
		profilesByName: make(map[string]config.Profile),
		routerConfig:   routing.NewRouterConfig(),
		properties:     make(map[string]string),
		storeOnFlush:   true,
		minNetworkSize: 200,
	}
}

func (g *GraphHopper) SetOSMFile(path string) *GraphHopper {
	g.osmFile = path
	return g
}

func (g *GraphHopper) SetGraphHopperLocation(path string) *GraphHopper {
	g.ghLocation = path
	return g
}

func (g *GraphHopper) SetStoreOnFlush(store bool) *GraphHopper {
	g.storeOnFlush = store
	return g
}

func (g *GraphHopper) SetProfiles(profiles ...config.Profile) *GraphHopper {
	for _, p := range profiles {
		g.profilesByName[p.Name] = p
	}
	return g
}

func (g *GraphHopper) Init(cfg GraphHopperConfig) *GraphHopper {
	g.ghLocation = cfg.GetString("graph.location", "graph-cache")
	if f := cfg.GetString("datareader.file", ""); f != "" {
		g.osmFile = f
	}
	for _, p := range cfg.Profiles {
		g.profilesByName[p.Name] = p
	}
	g.ignoredHighways = cfg.SplitCSV("import.osm.ignored_highways")
	g.minNetworkSize = cfg.GetInt("prepare.min_network_size", 200)
	g.initErr = validateProfileConfig(cfg)
	return g
}

func (g *GraphHopper) ImportOrLoad() error {
	if g.initErr != nil {
		return g.initErr
	}
	if g.fullyLoaded {
		return errors.New("graph is already successfully loaded")
	}
	if g.ghLocation == "" {
		return errors.New("GraphHopperLocation is not specified. Call Init or SetGraphHopperLocation before")
	}

	// Try to load an existing graph cache
	if loaded, err := g.load(); err != nil {
		return err
	} else if loaded {
		return nil
	}

	// No existing graph cache — import from OSM file if available
	if g.osmFile != "" {
		return g.process()
	}
	return nil
}

func (g *GraphHopper) load() (bool, error) {
	info, err := os.Stat(g.ghLocation)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("GraphHopperLocation cannot be an existing file. Has to be either non-existing or a folder: %s", g.ghLocation)
	}

	dir := storage.NewGHDirectory(g.ghLocation, storage.DATypeRAMStore)
	props := storage.NewStorableProperties(dir)
	if !props.LoadExisting() {
		return false, nil
	}

	em := routingutil.FromProperties(props)
	g.encodingManager = em

	bg := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetDir(dir).
		SetWithTurnCosts(em.NeedsTurnCostsSupport()).
		Build()

	if !bg.LoadExisting() {
		return false, fmt.Errorf("could not load existing graph from: %s", g.ghLocation)
	}

	g.graph = bg

	locationIdx := index.NewLocationIndexTree(bg, dir)
	if locationIdx.LoadExisting() {
		g.locationIndex = locationIdx
	}

	g.buildRouter()
	g.fullyLoaded = true
	return true, nil
}

// process imports an OSM file, builds the graph, location index, and optionally flushes to disk.
func (g *GraphHopper) process() error {
	log.Printf("Importing OSM file: %s", g.osmFile)

	em := g.buildEncodingManager()
	osmParsers := g.buildOSMParsers(em)

	dir := storage.NewGHDirectory(g.ghLocation, storage.DATypeRAMStore)
	graph := storage.NewBaseGraphBuilder(em.BytesForFlags).
		SetDir(dir).
		SetWithTurnCosts(em.NeedsTurnCostsSupport()).
		Build()
	graph.Create(1000)

	reader := osm.NewOSMReader(graph, osmParsers, routing.NewOSMReaderConfig())
	if err := reader.ReadGraph(g.osmFile); err != nil {
		return err
	}

	g.cleanUp(graph, em)

	props := storage.NewStorableProperties(dir)
	routingutil.PutEncodingManagerIntoProperties(em, props)

	locIndex := index.NewLocationIndexTree(graph, dir)
	locIndex.PrepareIndex()

	if g.storeOnFlush {
		graph.Flush()
		locIndex.Flush()
		props.Flush()
	}

	g.graph = graph
	g.encodingManager = em
	g.locationIndex = locIndex
	g.buildRouter()

	g.fullyLoaded = true
	log.Printf("Import complete. nodes: %d, edges: %d", graph.GetNodes(), graph.GetEdges())
	return nil
}

func (g *GraphHopper) cleanUp(graph *storage.BaseGraph, em *routingutil.EncodingManager) {
	wf := weighting.NewDefaultWeightingFactory(graph, em)
	var jobs []subnetwork.PrepareJob
	for _, profile := range g.profilesByName {
		subnetworkEnc := em.GetBooleanEncodedValue(ev.SubnetworkKey(profile.Name))
		w := wf.CreateWeighting(profile, nil, false)
		jobs = append(jobs, subnetwork.PrepareJob{SubnetworkEnc: subnetworkEnc, Weighting: w})
	}
	prs := subnetwork.NewPrepareRoutingSubnetworks(graph, jobs)
	prs.SetMinNetworkSize(g.minNetworkSize)
	prs.DoWork()
}

func (g *GraphHopper) buildRouter() {
	wf := weighting.NewDefaultWeightingFactory(g.graph, g.encodingManager)
	g.router = routing.NewRouter(g.graph, g.locationIndex, g.routerConfig, g.profilesByName, wf, g.encodingManager)
}

func (g *GraphHopper) buildEncodingManager() *routingutil.EncodingManager {
	return routingutil.Start().
		Add(ev.VehicleAccessCreate("car")).
		Add(ev.VehicleSpeedCreate("car", 7, 2, true)).
		AddTurnCostEncodedValue(ev.TurnCostCreate("car", 1)).
		Add(ev.SubnetworkCreate("car")).
		Add(ev.RoundaboutCreate()).
		Add(ev.RoadClassCreate()).
		Add(ev.RoadClassLinkCreate()).
		Add(ev.RoadEnvironmentCreate()).
		Add(ev.MaxSpeedCreate()).
		Add(ev.RoadAccessCreate()).
		Add(ev.FerrySpeedCreate()).
		Add(ev.OSMWayIDCreate()).
		Build()
}

func (g *GraphHopper) buildOSMParsers(em *routingutil.EncodingManager) *routing.OSMParsers {
	p := routing.NewOSMParsers()
	for _, h := range g.ignoredHighways {
		p.AddIgnoredHighway(h)
	}
	p.AddWayTagParser(parsers.NewOSMRoundaboutParser(em.GetBooleanEncodedValue(ev.RoundaboutKey)))
	p.AddWayTagParser(parsers.NewOSMRoadClassParser(em.GetEncodedValue(ev.RoadClassKey).(*ev.EnumEncodedValue[ev.RoadClass])))
	p.AddWayTagParser(parsers.NewOSMRoadClassLinkParser(em.GetBooleanEncodedValue(ev.RoadClassLinkKey)))
	p.AddWayTagParser(parsers.NewOSMRoadEnvironmentParser(em.GetEncodedValue(ev.RoadEnvironmentKey).(*ev.EnumEncodedValue[ev.RoadEnvironment])))
	p.AddWayTagParser(parsers.NewOSMMaxSpeedParser(em.GetDecimalEncodedValue(ev.MaxSpeedKey)))
	p.AddWayTagParser(parsers.NewOSMRoadAccessParser(
		em.GetEncodedValue(ev.RoadAccessKey).(*ev.EnumEncodedValue[ev.RoadAccess]),
		parsers.ToOSMRestrictions(routingutil.TransportationModeCar),
	))
	p.AddWayTagParser(parsers.NewOSMWayIDParser(em.GetIntEncodedValue(ev.OSMWayIDKey)))
	p.AddWayTagParser(parsers.NewFerrySpeedCalculator(em.GetDecimalEncodedValue(ev.FerrySpeedKey)))
	p.AddWayTagParser(parsers.NewCarAccessParser(em, true, true))
	p.AddWayTagParser(parsers.NewCarAverageSpeedParser(em))
	return p
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
