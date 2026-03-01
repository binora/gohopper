package routing

import (
	"fmt"
	"time"

	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// FlexiblePathCalculator creates a fresh RoutingAlgorithm per path calculation.
type FlexiblePathCalculator struct {
	graph        storage.Graph
	algoFactory  RoutingAlgorithmFactory
	weighting    weighting.Weighting
	algoOpts     AlgorithmOptions
	debug        string
	visitedNodes int
}

func NewFlexiblePathCalculator(graph storage.Graph, algoFactory RoutingAlgorithmFactory, w weighting.Weighting, algoOpts AlgorithmOptions) *FlexiblePathCalculator {
	return &FlexiblePathCalculator{
		graph:       graph,
		algoFactory: algoFactory,
		weighting:   w,
		algoOpts:    algoOpts,
	}
}

func (f *FlexiblePathCalculator) CalcPaths(from, to int, restrictions EdgeRestrictions) []*Path {
	algo := f.createAlgo()
	return f.calcPaths(from, to, restrictions, algo)
}

func (f *FlexiblePathCalculator) createAlgo() RoutingAlgorithm {
	start := time.Now()
	algo := f.algoFactory.CreateAlgo(f.graph, f.weighting, f.algoOpts)
	elapsed := time.Since(start).Microseconds()
	f.debug = fmt.Sprintf(", algoInit:%d \u03bcs", elapsed)
	return algo
}

func (f *FlexiblePathCalculator) calcPaths(from, to int, _ EdgeRestrictions, algo RoutingAlgorithm) []*Path {
	start := time.Now()
	paths := algo.CalcPaths(from, to)

	if len(paths) == 0 {
		panic(fmt.Sprintf("Path list was empty for %d -> %d", from, to))
	}

	f.visitedNodes = algo.GetVisitedNodes()
	elapsed := time.Since(start).Milliseconds()
	f.debug += fmt.Sprintf(", %s-routing:%d ms", algo.GetName(), elapsed)
	return paths
}

func (f *FlexiblePathCalculator) GetDebugString() string {
	return f.debug
}

func (f *FlexiblePathCalculator) GetVisitedNodes() int {
	return f.visitedNodes
}

func (f *FlexiblePathCalculator) GetWeighting() weighting.Weighting {
	return f.weighting
}

func (f *FlexiblePathCalculator) SetWeighting(w weighting.Weighting) {
	f.weighting = w
}
