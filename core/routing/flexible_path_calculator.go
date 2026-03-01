package routing

import (
	"fmt"
	"time"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// FlexiblePathCalculator implements PathCalculator by creating a fresh
// RoutingAlgorithm for each path calculation.
// Port of Java com.graphhopper.routing.FlexiblePathCalculator.
type FlexiblePathCalculator struct {
	graph        storage.Graph
	algoFactory  RoutingAlgorithmFactory
	weighting    weighting.Weighting
	algoOpts     AlgorithmOptions
	debug        string
	visitedNodes int
}

// NewFlexiblePathCalculator creates a new FlexiblePathCalculator.
func NewFlexiblePathCalculator(graph storage.Graph, algoFactory RoutingAlgorithmFactory, w weighting.Weighting, algoOpts AlgorithmOptions) *FlexiblePathCalculator {
	return &FlexiblePathCalculator{
		graph:       graph,
		algoFactory: algoFactory,
		weighting:   w,
		algoOpts:    algoOpts,
	}
}

// CalcPaths calculates paths from 'from' to 'to' with the given edge restrictions.
func (f *FlexiblePathCalculator) CalcPaths(from, to int, restrictions EdgeRestrictions) []*Path {
	algo := f.createAlgo()
	return f.calcPaths(from, to, restrictions, algo)
}

// createAlgo creates a new routing algorithm and records the init time.
func (f *FlexiblePathCalculator) createAlgo() RoutingAlgorithm {
	start := time.Now()
	algo := f.algoFactory.CreateAlgo(f.graph, f.weighting, f.algoOpts)
	elapsed := time.Since(start).Microseconds()
	f.debug = fmt.Sprintf(", algoInit:%d \u03bcs", elapsed)
	return algo
}

// calcPaths runs the algorithm and records debug info and visited nodes.
func (f *FlexiblePathCalculator) calcPaths(from, to int, restrictions EdgeRestrictions, algo RoutingAlgorithm) []*Path {
	start := time.Now()

	// TODO: when QueryGraph is ported, handle unfavored virtual edges here:
	// for _, edge := range restrictions.UnfavoredEdges {
	//     queryGraph.UnfavorVirtualEdge(edge)
	// }

	var paths []*Path
	if restrictions.SourceOutEdge != AnyEdge || restrictions.TargetInEdge != AnyEdge {
		// TODO: EdgeToEdgeRoutingAlgorithm curbside support not yet implemented.
		// For now, fall back to standard routing.
		paths = algo.CalcPaths(from, to)
	} else {
		paths = algo.CalcPaths(from, to)
	}

	// TODO: when QueryGraph is ported, clear unfavored status here:
	// queryGraph.ClearUnfavoredStatus()

	if len(paths) == 0 {
		panic(fmt.Sprintf("Path list was empty for %d -> %d", from, to))
	}

	f.visitedNodes = algo.GetVisitedNodes()
	elapsed := time.Since(start).Milliseconds()
	f.debug += fmt.Sprintf(", %s-routing:%d ms", algo.GetName(), elapsed)
	return paths
}

// GetDebugString returns debug information from the last path calculation.
func (f *FlexiblePathCalculator) GetDebugString() string {
	return f.debug
}

// GetVisitedNodes returns the number of visited nodes from the last path calculation.
func (f *FlexiblePathCalculator) GetVisitedNodes() int {
	return f.visitedNodes
}

// GetWeighting returns the current weighting.
func (f *FlexiblePathCalculator) GetWeighting() weighting.Weighting {
	return f.weighting
}

// SetWeighting sets the weighting for future path calculations.
func (f *FlexiblePathCalculator) SetWeighting(w weighting.Weighting) {
	f.weighting = w
}
