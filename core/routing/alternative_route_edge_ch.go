package routing

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

// AlternativeRouteEdgeCH is the edge-based counterpart of AlternativeRouteCH.
// Mirrors com.graphhopper.routing.AlternativeRouteEdgeCH.
type AlternativeRouteEdgeCH struct {
	*DijkstraBidirectionEdgeCHNoSOD

	maxWeightFactor       float64
	maxShareFactor        float64
	localOptimalityFactor float64
	maxPaths              int
	alternatives          []AlternativeInfo
	extraVisitedNodes     int
}

type potentialEdgeAlternativeInfo struct {
	v      int
	edgeIn int
	weight float64
}

// NewAlternativeRouteEdgeCH builds the edge-based alt-route algorithm,
// reading tuning parameters from the PMap (same keys as Java).
func NewAlternativeRouteEdgeCH(graph storage.RoutingCHGraph, hints webapi.PMap) *AlternativeRouteEdgeCH {
	a := &AlternativeRouteEdgeCH{
		DijkstraBidirectionEdgeCHNoSOD: NewDijkstraBidirectionEdgeCHNoSOD(graph),
		maxWeightFactor:                hints.GetFloat64(altMaxWeightFactorKey, 1.25),
		maxShareFactor:                 hints.GetFloat64(altMaxShareFactorKey, 0.8),
		localOptimalityFactor:          hints.GetFloat64(altLocalOptimalityFactorKey, 0.25),
		maxPaths:                       hints.GetInt(altMaxPathsKey, 3),
	}
	a.FinishedFn = a.finishedAlt
	return a
}

func (a *AlternativeRouteEdgeCH) finishedAlt() bool {
	if a.finishedFrom && a.finishedTo {
		return true
	}
	return a.currFrom.Weight >= a.BestWeight*a.maxWeightFactor &&
		a.currTo.Weight >= a.BestWeight*a.maxWeightFactor
}

func (a *AlternativeRouteEdgeCH) GetVisitedNodes() int {
	return a.visitedCountFrom + a.visitedCountTo + a.extraVisitedNodes
}

// CalcAlternatives returns up to maxPaths alternatives from s to t. The first
// entry is always the shortest path (when one exists).
func (a *AlternativeRouteEdgeCH) CalcAlternatives(s, t int) []AlternativeInfo {
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

	// In edge-based bidir CH, bestWeightMapTo is keyed by edge-key, not node.
	// Build a node→entry index so we can pair fwd/bwd entries that meet at the
	// same node regardless of incoming edge.
	bestByAdjNode := make(map[int]*SPTEntry, len(a.bestWeightMapTo))
	for _, e := range a.bestWeightMapTo {
		bestByAdjNode[e.AdjNode] = e
	}

	var potentials []potentialEdgeAlternativeInfo
	// Iterate bestWeightMapFrom in deterministic key order. Java relies on
	// hppc.IntObjectHashMap's seeded iteration; Go map iteration is randomized
	// per run, so without sorting the picked alternatives would be unstable
	// across runs when several candidates share the same weight bucket.
	weightLimit := bestPath.Weight * a.maxWeightFactor
	for _, k := range slices.Sorted(maps.Keys(a.bestWeightMapFrom)) {
		fromEntry := a.bestWeightMapFrom[k]
		toEntry, ok := bestByAdjNode[fromEntry.AdjNode]
		if !ok {
			continue
		}
		if fromEntry.GetWeightOfVisitedPath()+toEntry.GetWeightOfVisitedPath() > weightLimit {
			continue
		}
		preliminary := a.createPathExtractor().Extract(fromEntry, toEntry,
			fromEntry.GetWeightOfVisitedPath()+toEntry.GetWeightOfVisitedPath())
		share := a.calculateShare(preliminary)
		if share > a.maxShareFactor {
			continue
		}
		potentials = append(potentials, potentialEdgeAlternativeInfo{
			v:      fromEntry.AdjNode,
			edgeIn: a.getIncomingEdge(fromEntry),
			weight: 2*(fromEntry.GetWeightOfVisitedPath()+toEntry.GetWeightOfVisitedPath()) + share,
		})
	}
	slices.SortFunc(potentials, func(a, b potentialEdgeAlternativeInfo) int { return cmp.Compare(a.weight, b.weight) })

	baseGraph := a.Graph.GetBaseGraph()
	w, ok := a.Graph.GetWeighting().(weighting.Weighting)
	if !ok {
		panic(fmt.Sprintf("CH weighting %T does not implement weighting.Weighting", a.Graph.GetWeighting()))
	}
	for _, p := range potentials {
		v := p.v
		tailSv := p.edgeIn

		svRouter := NewDijkstraBidirectionEdgeCHNoSOD(a.Graph)
		svRouter.SetPathExtractorSupplier(a.createPathExtractor)
		suvPath := svRouter.CalcPathEdgeToEdge(s, v, util.AnyEdge, tailSv)
		a.extraVisitedNodes += svRouter.GetVisitedNodes()

		u := baseGraph.GetEdgeIteratorState(tailSv, v).GetBaseNode()

		vtRouter := NewDijkstraBidirectionEdgeCHNoSOD(a.Graph)
		vtRouter.SetPathExtractorSupplier(a.createPathExtractor)
		uvtPath := vtRouter.CalcPathEdgeToEdge(u, t, tailSv, util.AnyEdge)
		a.extraVisitedNodes += vtRouter.GetVisitedNodes()
		// If there's a turn restriction at u→v→x the uvt-path may not reach t;
		// dropping the candidate prevents returning a truncated alternative.
		if !uvtPath.Found {
			continue
		}

		path := concatEdgePaths(baseGraph, w, suvPath, uvtPath)

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
		vIndex := len(suvPath.CalcNodes()) - 1
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

func (a *AlternativeRouteEdgeCH) CalcPaths(from, to int) []*Path {
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

func (a *AlternativeRouteEdgeCH) calculateShare(path *Path) float64 {
	if path.Distance == 0 {
		return 0
	}
	return a.sharedDistance(path) / path.Distance
}

func (a *AlternativeRouteEdgeCH) sharedDistance(path *Path) float64 {
	var shared float64
	for _, edge := range path.CalcEdges() {
		if a.containsInAnyAlternative(edge.GetBaseNode()) && a.containsInAnyAlternative(edge.GetAdjNode()) {
			shared += edge.GetDistance()
		}
	}
	return shared
}

func (a *AlternativeRouteEdgeCH) sharedDistanceWithShortest(path *Path) float64 {
	var shared float64
	shortest := a.alternatives[0].Nodes
	for _, edge := range path.CalcEdges() {
		if slices.Contains(shortest, edge.GetBaseNode()) && slices.Contains(shortest, edge.GetAdjNode()) {
			shared += edge.GetDistance()
		}
	}
	return shared
}

func (a *AlternativeRouteEdgeCH) containsInAnyAlternative(v int) bool {
	for _, alt := range a.alternatives {
		if slices.Contains(alt.Nodes, v) {
			return true
		}
	}
	return false
}

func (a *AlternativeRouteEdgeCH) tTest(path *Path, vIndex int) bool {
	if path.GetEdgeCount() == 0 {
		return true
	}
	detour := path.Distance - a.sharedDistanceWithShortest(path)
	T := 0.5 * a.localOptimalityFactor * detour
	edges := path.CalcEdges()
	fromEdge := previousEdgeTMetersAway(edges, vIndex, T)
	toEdge := nextEdgeTMetersAway(edges, vIndex, T)
	tRouter := NewDijkstraBidirectionEdgeCHNoSOD(a.Graph)
	tRouter.SetPathExtractorSupplier(a.createPathExtractor)
	tPath := tRouter.CalcPathEdgeToEdge(fromEdge.GetBaseNode(), toEdge.GetAdjNode(), fromEdge.GetEdge(), toEdge.GetEdge())
	a.extraVisitedNodes += tRouter.GetVisitedNodes()
	v := path.CalcNodes()[vIndex]
	return slices.Contains(tPath.CalcNodes(), v)
}

func previousEdgeTMetersAway(edges []util.EdgeIteratorState, vIndex int, T float64) util.EdgeIteratorState {
	dist := 0.0
	i := vIndex
	for i > 0 && dist < T {
		dist += edges[i-1].GetDistance()
		i--
	}
	return edges[i]
}

func nextEdgeTMetersAway(edges []util.EdgeIteratorState, vIndex int, T float64) util.EdgeIteratorState {
	dist := 0.0
	i := vIndex
	for i < len(edges)-1 && dist < T {
		dist += edges[i].GetDistance()
		i++
	}
	return edges[i-1]
}

// concatEdgePaths glues s→u→v and u→v→t into s→u→v→...→t, subtracting the
// duplicated u→v edge so weight/distance/time aren't double-counted. Mirrors
// Java AlternativeRouteEdgeCH.concat.
func concatEdgePaths(baseGraph storage.Graph, w weighting.Weighting, suvPath, uvtPath *Path) *Path {
	p := NewPath(baseGraph)
	p.SetFromNode(suvPath.FromNode)
	uvEdge := uvtPath.EdgeIDs[0]
	p.EdgeIDs = slices.Concat(suvPath.EdgeIDs, uvtPath.EdgeIDs[1:])
	vuEdgeState := baseGraph.GetEdgeIteratorState(uvEdge, uvtPath.FromNode)
	p.SetEndNode(uvtPath.EndNode)
	p.SetWeight(suvPath.Weight + uvtPath.Weight - w.CalcEdgeWeight(vuEdgeState, true))
	p.SetDistance(suvPath.Distance + uvtPath.Distance - vuEdgeState.GetDistance())
	p.AddTime(suvPath.Time + uvtPath.Time - w.CalcEdgeMillis(vuEdgeState, true))
	p.SetFound(true)
	return p
}
