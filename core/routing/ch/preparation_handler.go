package ch

import (
	"time"

	"gohopper/core/config"
	"gohopper/core/storage"
	webapi "gohopper/web-api"
)

// PreparationHandler mirrors com.graphhopper.routing.ch.CHPreparationHandler.
// It orchestrates load/prepare of one or more CH profiles against a BaseGraph.
//
// Divergence from Java: Java runs Load/Prepare concurrently when
// preparationThreads > 1 via GHUtility.runConcurrently; gohopper currently has
// no concurrent task helper, so both Load and Prepare iterate sequentially.
// Java itself defaults to preparationThreads=1, so the default behavior is
// identical. A concurrent variant can be added when a caller needs it.
type PreparationHandler struct {
	chProfiles         []config.CHProfile
	preparationThreads int
	pMap               webapi.PMap
}

func NewPreparationHandler() *PreparationHandler {
	return &PreparationHandler{
		preparationThreads: 1,
		pMap:               webapi.NewPMap(),
	}
}

// InitConfig is the minimal GraphHopperConfig surface needed by Init. It
// exists to keep ch decoupled from core (where *GraphHopperConfig lives) —
// importing core here would create a cycle since core imports core/routing.
type InitConfig interface {
	Has(key string) bool
	GetInt(key string, def int) int
}

// Init mirrors Java CHPreparationHandler.init: it enforces the deprecated-key
// migrations, reads the preparation thread count, and accepts the CH profiles
// and PMap directly (Java's chConfig.getCHProfiles()/asPMap() would require
// extra accessors here that no Go caller needs yet).
func (h *PreparationHandler) Init(cfg InitConfig, chProfiles []config.CHProfile, pMap webapi.PMap) {
	if cfg.Has("prepare.threads") {
		panic("Use prepare.ch.threads instead of prepare.threads")
	}
	if cfg.Has("prepare.chWeighting") || cfg.Has("prepare.chWeightings") || cfg.Has("prepare.ch.weightings") {
		panic("Use profiles_ch instead of prepare.chWeighting, prepare.chWeightings or prepare.ch.weightings, see #1922 and docs/core/profiles.md")
	}
	if cfg.Has("prepare.ch.edge_based") {
		panic("Use profiles_ch instead of prepare.ch.edge_based, see #1922 and docs/core/profiles.md")
	}
	h.SetPreparationThreads(cfg.GetInt("prepare.ch.threads", h.preparationThreads))
	h.SetCHProfiles(chProfiles...)
	h.pMap = pMap
}

func (h *PreparationHandler) IsEnabled() bool {
	return len(h.chProfiles) > 0
}

func (h *PreparationHandler) SetCHProfiles(chProfiles ...config.CHProfile) *PreparationHandler {
	h.chProfiles = append(h.chProfiles[:0], chProfiles...)
	return h
}

func (h *PreparationHandler) GetCHProfiles() []config.CHProfile {
	return h.chProfiles
}

func (h *PreparationHandler) GetPreparationThreads() int {
	return h.preparationThreads
}

func (h *PreparationHandler) SetPreparationThreads(n int) {
	h.preparationThreads = n
}

// Load mirrors Java CHPreparationHandler.load: for each CH config, attempt to
// load the on-disk CH storage; on miss, remove any half-written files. The
// returned map keys are CHConfig.GetName().
func (h *PreparationHandler) Load(graph *storage.BaseGraph, chConfigs []*CHConfig) map[string]storage.RoutingCHGraph {
	loaded := make(map[string]storage.RoutingCHGraph, len(chConfigs))
	for _, c := range chConfigs {
		chStorage := storage.NewCHStorage(graph.GetDirectory(), c.GetName(), graph.GetSegmentSize(), c.IsEdgeBased())
		if chStorage.LoadExisting() {
			loaded[c.GetName()] = storage.NewRoutingCHGraph(graph, chStorage, c.GetWeighting())
		} else {
			graph.GetDirectory().Remove("nodes_ch_" + c.GetName())
			graph.GetDirectory().Remove("shortcuts_" + c.GetName())
		}
	}
	return loaded
}

// Prepare mirrors Java CHPreparationHandler.prepare: it runs the actual CH
// preparation for each CHConfig and writes a date stamp into properties. When
// closeEarly is true, the underlying CHStorage is closed after flush.
func (h *PreparationHandler) Prepare(graph *storage.BaseGraph, props *storage.StorableProperties, chConfigs []*CHConfig, closeEarly bool) map[string]*Result {
	if len(chConfigs) == 0 {
		return map[string]*Result{}
	}
	results := make(map[string]*Result, len(chConfigs))
	for _, c := range chConfigs {
		name := c.GetName()
		prepare := FromGraph(graph, c).SetParams(h.pMap)
		results[name] = prepare.DoWork()
		prepare.Flush()
		if closeEarly {
			prepare.Close()
		}
		props.Put("prepare.ch.date."+name, time.Now().UTC().Format("2006-01-02_15-04-05"))
	}
	return results
}
