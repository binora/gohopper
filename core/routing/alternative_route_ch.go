package routing

import (
	"slices"
	"sort"

	"gohopper/core/storage"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

// AlternativeRouteCH is a minimum-moving-parts implementation of alternative
// route search on contraction hierarchies, after "Alternative Routes in Road
// Networks" (Abraham et al.). Mirrors com.graphhopper.routing.AlternativeRouteCH.
type AlternativeRouteCH struct {
	*DijkstraBidirectionCHNoSOD

	maxWeightFactor       float64
	maxShareFactor        float64
	localOptimalityFactor float64
	maxPaths              int
	alternatives          []AlternativeInfo
	extraVisitedNodes     int
}

// AlternativeInfo describes one alternative path and the fraction it shares
// with the shortest path.
type AlternativeInfo struct {
	Path        *Path
	ShareWeight float64
	Nodes       []int
}

func (a AlternativeInfo) GetPath() *Path { return a.Path }

type potentialAlternativeInfo struct {
	v      int
	weight float64
}

const (
	altMaxWeightFactorKey       = "alternative_route.max_weight_factor"
	altMaxShareFactorKey        = "alternative_route.max_share_factor"
	altLocalOptimalityFactorKey = "alternative_route.local_optimality_factor"
	altMaxPathsKey              = "alternative_route.max_paths"
)

// NewAlternativeRouteCH builds the alt-route algorithm reading tuning
// parameters from the PMap (same keys as Java).
func NewAlternativeRouteCH(graph storage.RoutingCHGraph, hints webapi.PMap) *AlternativeRouteCH {
	a := &AlternativeRouteCH{
		DijkstraBidirectionCHNoSOD: NewDijkstraBidirectionCHNoSOD(graph),
		maxWeightFactor:            hints.GetFloat64(altMaxWeightFactorKey, 1.25),
		maxShareFactor:             hints.GetFloat64(altMaxShareFactorKey, 0.8),
		localOptimalityFactor:      hints.GetFloat64(altLocalOptimalityFactorKey, 0.25),
		maxPaths:                   hints.GetInt(altMaxPathsKey, 3),
	}
	a.FinishedFn = a.finishedAlt
	return a
}

// finishedAlt continues bidir search past the optimal meeting weight (until
// both fronts exceed bestWeight*maxWeightFactor) so alt meeting points can
// be harvested.
func (a *AlternativeRouteCH) finishedAlt() bool {
	if a.finishedFrom && a.finishedTo {
		return true
	}
	return a.currFrom.Weight >= a.BestWeight*a.maxWeightFactor &&
		a.currTo.Weight >= a.BestWeight*a.maxWeightFactor
}

func (a *AlternativeRouteCH) GetVisitedNodes() int {
	return a.visitedCountFrom + a.visitedCountTo + a.extraVisitedNodes
}

// CalcAlternatives returns up to maxPaths alternatives from s to t. The first
// entry is always the shortest path (when one exists).
func (a *AlternativeRouteCH) CalcAlternatives(s, t int) []AlternativeInfo {
	a.checkAlreadyRun()
	a.init(s, 0, t, 0)
	a.runAlgo()
	bestPath := a.extractPath()
	if !bestPath.Found {
		return nil
	}
	a.alternatives = append(a.alternatives, AlternativeInfo{
		Path:  bestPath,
		Nodes: bestPath.CalcNodes(),
	})

	var potentials []potentialAlternativeInfo
	// Iterate in deterministic key order — see note in AlternativeRouteEdgeCH.
	keysFrom := make([]int, 0, len(a.bestWeightMapFrom))
	for k := range a.bestWeightMapFrom {
		keysFrom = append(keysFrom, k)
	}
	sort.Ints(keysFrom)
	for _, v := range keysFrom {
		fromEntry := a.bestWeightMapFrom[v]
		toEntry, ok := a.bestWeightMapTo[v]
		if !ok {
			continue
		}
		if fromEntry.GetWeightOfVisitedPath()+toEntry.GetWeightOfVisitedPath() > bestPath.Weight*a.maxWeightFactor {
			continue
		}
		preliminary := a.createPathExtractor().Extract(fromEntry, toEntry,
			fromEntry.GetWeightOfVisitedPath()+toEntry.GetWeightOfVisitedPath())
		share := a.calculateShare(preliminary)
		if share > a.maxShareFactor {
			continue
		}
		potentials = append(potentials, potentialAlternativeInfo{
			v:      v,
			weight: 2*(fromEntry.GetWeightOfVisitedPath()+toEntry.GetWeightOfVisitedPath()) + share,
		})
	}
	sort.Slice(potentials, func(i, j int) bool { return potentials[i].weight < potentials[j].weight })

	for _, p := range potentials {
		v := p.v
		svRouter := NewDijkstraBidirectionCH(a.Graph)
		svRouter.SetPathExtractorSupplier(a.createPathExtractor)
		svPath := svRouter.CalcPath(s, v)
		a.extraVisitedNodes += svRouter.GetVisitedNodes()

		vtRouter := NewDijkstraBidirectionCH(a.Graph)
		vtRouter.SetPathExtractorSupplier(a.createPathExtractor)
		vtPath := vtRouter.CalcPath(v, t)
		a.extraVisitedNodes += vtRouter.GetVisitedNodes()

		path := concatPaths(a.Graph.GetBaseGraph(), svPath, vtPath)

		shared := a.sharedDistanceWithShortest(path)
		detourLength := path.Distance - shared
		directLength := bestPath.Distance - shared
		if detourLength > directLength*a.maxWeightFactor {
			continue
		}
		share := a.calculateShare(path)
		if share > a.maxShareFactor {
			continue
		}
		vIndex := len(svPath.CalcNodes()) - 1
		if !a.tTest(path, vIndex) {
			continue
		}
		a.alternatives = append(a.alternatives, AlternativeInfo{
			Path:        path,
			ShareWeight: share,
			Nodes:       path.CalcNodes(),
		})
		if len(a.alternatives) >= a.maxPaths {
			break
		}
	}
	return a.alternatives
}

// CalcPaths shadows the parent's CalcPaths and returns the alternatives list
// (single empty path on no result, matching Java).
func (a *AlternativeRouteCH) CalcPaths(from, to int) []*Path {
	alts := a.CalcAlternatives(from, to)
	if len(alts) == 0 {
		return []*Path{NewPath(a.Graph.GetBaseGraph())}
	}
	out := make([]*Path, len(alts))
	for i, alt := range alts {
		out[i] = alt.Path
	}
	return out
}

func (a *AlternativeRouteCH) calculateShare(path *Path) float64 {
	if path.Distance == 0 {
		return 0
	}
	return a.sharedDistance(path) / path.Distance
}

func (a *AlternativeRouteCH) sharedDistance(path *Path) float64 {
	var shared float64
	for _, edge := range path.CalcEdges() {
		if a.containsInAnyAlternative(edge.GetBaseNode()) && a.containsInAnyAlternative(edge.GetAdjNode()) {
			shared += edge.GetDistance()
		}
	}
	return shared
}

func (a *AlternativeRouteCH) sharedDistanceWithShortest(path *Path) float64 {
	var shared float64
	shortest := a.alternatives[0].Nodes
	for _, edge := range path.CalcEdges() {
		if slices.Contains(shortest, edge.GetBaseNode()) && slices.Contains(shortest, edge.GetAdjNode()) {
			shared += edge.GetDistance()
		}
	}
	return shared
}

func (a *AlternativeRouteCH) containsInAnyAlternative(v int) bool {
	for _, alt := range a.alternatives {
		if slices.Contains(alt.Nodes, v) {
			return true
		}
	}
	return false
}

// tTest discards alternatives not locally shortest around the via node: from
// ~T meters before v to ~T meters after v, the straight shortest path must
// still pass through v.
func (a *AlternativeRouteCH) tTest(path *Path, vIndex int) bool {
	if path.GetEdgeCount() == 0 {
		return true
	}
	detour := path.Distance - a.sharedDistanceWithShortest(path)
	T := 0.5 * a.localOptimalityFactor * detour
	edges := path.CalcEdges()
	fromNode := previousNodeTMetersAway(edges, vIndex, T)
	toNode := nextNodeTMetersAway(edges, vIndex, T)
	tRouter := NewDijkstraBidirectionCH(a.Graph)
	tRouter.SetPathExtractorSupplier(a.createPathExtractor)
	tPath := tRouter.CalcPath(fromNode, toNode)
	a.extraVisitedNodes += tRouter.GetVisitedNodes()
	v := path.CalcNodes()[vIndex]
	return slices.Contains(tPath.CalcNodes(), v)
}

func previousNodeTMetersAway(edges []util.EdgeIteratorState, vIndex int, T float64) int {
	dist := 0.0
	i := vIndex
	for i > 0 && dist < T {
		dist += edges[i-1].GetDistance()
		i--
	}
	return edges[i].GetBaseNode()
}

func nextNodeTMetersAway(edges []util.EdgeIteratorState, vIndex int, T float64) int {
	dist := 0.0
	i := vIndex
	for i < len(edges)-1 && dist < T {
		dist += edges[i].GetDistance()
		i++
	}
	return edges[i-1].GetAdjNode()
}

func concatPaths(baseGraph storage.Graph, svPath, vtPath *Path) *Path {
	p := NewPath(baseGraph)
	p.EdgeIDs = append(p.EdgeIDs, svPath.EdgeIDs...)
	p.EdgeIDs = append(p.EdgeIDs, vtPath.EdgeIDs...)
	p.SetFromNode(svPath.CalcNodes()[0])
	p.SetEndNode(vtPath.EndNode)
	p.SetWeight(svPath.Weight + vtPath.Weight)
	p.SetDistance(svPath.Distance + vtPath.Distance)
	p.AddTime(svPath.Time + vtPath.Time)
	p.SetFound(true)
	return p
}
