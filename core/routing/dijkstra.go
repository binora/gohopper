package routing

import (
	"container/heap"
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
)

// Dijkstra implements a single-source shortest path algorithm.
// See http://en.wikipedia.org/wiki/Dijkstra's_algorithm
type Dijkstra struct {
	AbstractRoutingAlgorithm
	fromMap  map[int]*SPTEntry
	fromHeap sptEntryHeap
	currEdge *SPTEntry
	to       int
}

// NewDijkstra creates a new Dijkstra algorithm instance.
func NewDijkstra(graph storage.Graph, w weighting.Weighting, tMode routingutil.TraversalMode) *Dijkstra {
	size := graph.GetNodes() / 10
	if size < 200 {
		size = 200
	}
	if size > 2000 {
		size = 2000
	}
	d := &Dijkstra{
		AbstractRoutingAlgorithm: NewAbstractRoutingAlgorithm(graph, w, tMode),
		to:                       -1,
	}
	d.initCollections(size)
	return d
}

func (d *Dijkstra) initCollections(size int) {
	d.fromHeap = make(sptEntryHeap, 0, size)
	d.fromMap = make(map[int]*SPTEntry, size)
}

// CalcPath calculates the best path between from and to.
func (d *Dijkstra) CalcPath(from, to int) *Path {
	d.CheckAlreadyRun()
	d.SetupFinishTime()
	d.to = to
	startEntry := NewSPTEntry(from, 0)
	heap.Push(&d.fromHeap, startEntry)
	if !d.TraversalMode.IsEdgeBased() {
		d.fromMap[from] = startEntry
	}
	d.runAlgo()
	return d.extractPath()
}

// CalcPaths returns CalcPath wrapped in a single-element slice.
func (d *Dijkstra) CalcPaths(from, to int) []*Path {
	return DefaultCalcPaths(d, from, to)
}

func (d *Dijkstra) runAlgo() {
	for d.fromHeap.Len() > 0 {
		d.currEdge = heap.Pop(&d.fromHeap).(*SPTEntry)
		if d.currEdge.Deleted {
			continue
		}
		d.VisitedNodes++
		if d.IsMaxVisitedNodesExceeded() || d.finished() || d.IsTimeoutExceeded() {
			break
		}

		currNode := d.currEdge.AdjNode
		iter := d.EdgeExplorer.SetBaseNode(currNode)
		for iter.Next() {
			if !d.Accept(iter, d.currEdge.Edge) {
				continue
			}

			tmpWeight := CalcWeightWithTurnWeight(d.Weighting, iter, false, d.currEdge.Edge) + d.currEdge.Weight
			if math.IsInf(tmpWeight, 1) {
				continue
			}

			traversalID := d.TraversalMode.CreateTraversalID(iter, false)

			nEdge, exists := d.fromMap[traversalID]
			if !exists {
				nEdge = NewSPTEntryFull(iter.GetEdge(), iter.GetAdjNode(), tmpWeight, d.currEdge)
				d.fromMap[traversalID] = nEdge
				heap.Push(&d.fromHeap, nEdge)
			} else if nEdge.Weight > tmpWeight {
				nEdge.Deleted = true
				nEdge = NewSPTEntryFull(iter.GetEdge(), iter.GetAdjNode(), tmpWeight, d.currEdge)
				d.fromMap[traversalID] = nEdge
				heap.Push(&d.fromHeap, nEdge)
			} else {
				continue
			}

			d.updateBestPath(iter, nEdge, traversalID)
		}
	}
}

func (d *Dijkstra) finished() bool {
	return d.currEdge.AdjNode == d.to
}

func (d *Dijkstra) extractPath() *Path {
	if d.currEdge == nil || !d.finished() {
		return d.CreateEmptyPath()
	}
	return ExtractPath(d.Graph, d.Weighting, d.currEdge)
}

// updateBestPath is a no-op hook for subclasses (e.g., alternative routes).
func (d *Dijkstra) updateBestPath(_ interface{}, _ *SPTEntry, _ int) {}

// GetName returns the algorithm name.
func (d *Dijkstra) GetName() string {
	return AlgoDijkstra
}
