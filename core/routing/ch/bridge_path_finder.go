package ch

import (
	"container/heap"
	"fmt"
	"math"

	"gohopper/core/util"
)

// BridgePathEntry contains weight and CH entry information for a bridge path.
type BridgePathEntry struct {
	Weight  float64
	ChEntry *PrepareCHEntry
}

func (e *BridgePathEntry) String() string {
	return fmt.Sprintf("weight: %v, chEntry: %s", e.Weight, e.ChEntry)
}

// BridgePathFinder finds 'bridge-paths' during edge-based CH preparation. Bridge-paths are paths
// that start and end at neighbor nodes of the contracted node without visiting any other nodes.
type BridgePathFinder struct {
	graph           *CHPreparationGraph
	outExplorer     PrepareGraphEdgeExplorer
	origOutExplorer PrepareGraphOrigEdgeExplorer
	queue           prepareCHEntryHeap
	entries         map[int]*PrepareCHEntry
}

func NewBridgePathFinder(graph *CHPreparationGraph) *BridgePathFinder {
	return &BridgePathFinder{
		graph:           graph,
		outExplorer:     graph.CreateOutEdgeExplorer(),
		origOutExplorer: graph.CreateOutOrigEdgeExplorer(),
		entries:         make(map[int]*PrepareCHEntry),
	}
}

// Find finds all bridge paths starting at a given node and starting edge key.
// Returns a mapping between target edge keys reachable via bridge paths and their entries.
func (f *BridgePathFinder) Find(startInEdgeKey, startNode, centerNode int) map[int]*BridgePathEntry {
	f.queue = f.queue[:0]
	clear(f.entries)
	result := make(map[int]*BridgePathEntry)

	startEntry := NewPrepareCHEntry(util.NoEdge, startInEdgeKey, startInEdgeKey, startNode, 0, 0)
	f.entries[startInEdgeKey] = startEntry
	heap.Push(&f.queue, startEntry)

	for f.queue.Len() > 0 {
		currEntry := heap.Pop(&f.queue).(*PrepareCHEntry)
		iter := f.outExplorer.SetBaseNode(currEntry.AdjNode)
		for iter.Next() {
			adjNode := iter.GetAdjNode()
			firstKey := iter.GetOrigEdgeKeyFirst()
			lastKey := iter.GetOrigEdgeKeyLast()
			prepEdge := iter.GetPrepareEdge()
			origCount := iter.GetOrigEdgeCount()

			if adjNode == centerNode {
				// arrived at center node, keep expanding search
				weight := currEntry.Weight +
					f.graph.GetTurnWeight(currEntry.IncEdgeKey, currEntry.AdjNode, firstKey) +
					iter.GetWeight()
				if math.IsInf(weight, 1) {
					continue
				}
				entry, exists := f.entries[lastKey]
				if !exists {
					entry = NewPrepareCHEntry(prepEdge, firstKey, lastKey, adjNode, weight, currEntry.OrigEdges+origCount)
					entry.Parent = currEntry
					f.entries[lastKey] = entry
					heap.Push(&f.queue, entry)
				} else if weight < entry.Weight {
					f.queue.remove(entry)
					entry.PrepareEdge = prepEdge
					entry.OrigEdges = currEntry.OrigEdges + origCount
					entry.FirstEdgeKey = firstKey
					entry.Weight = weight
					entry.Parent = currEntry
					heap.Push(&f.queue, entry)
				}
			} else if currEntry.AdjNode == centerNode {
				// just left center node, arrived at neighbor. Record bridge path entries.
				weight := currEntry.Weight +
					f.graph.GetTurnWeight(currEntry.IncEdgeKey, currEntry.AdjNode, firstKey) +
					iter.GetWeight()
				if math.IsInf(weight, 1) {
					continue
				}
				origOutIter := f.origOutExplorer.SetBaseNode(adjNode)
				for origOutIter.Next() {
					outKey := origOutIter.GetOrigEdgeKeyFirst()
					totalWeight := weight + f.graph.GetTurnWeight(lastKey, adjNode, outKey)
					if math.IsInf(totalWeight, 1) {
						continue
					}
					resEntry, exists := result[outKey]
					if !exists {
						chEntry := NewPrepareCHEntry(prepEdge, firstKey, lastKey, adjNode, weight, currEntry.OrigEdges+origCount)
						chEntry.Parent = currEntry
						result[outKey] = &BridgePathEntry{Weight: totalWeight, ChEntry: chEntry}
					} else if totalWeight < resEntry.Weight {
						resEntry.Weight = totalWeight
						resEntry.ChEntry.PrepareEdge = prepEdge
						resEntry.ChEntry.FirstEdgeKey = firstKey
						resEntry.ChEntry.OrigEdges = currEntry.OrigEdges + origCount
						resEntry.ChEntry.IncEdgeKey = lastKey
						resEntry.ChEntry.Weight = weight
						resEntry.ChEntry.Parent = currEntry
					}
				}
			}
		}
	}
	return result
}

// prepareCHEntryHeap implements heap.Interface for PrepareCHEntry pointers.
type prepareCHEntryHeap []*PrepareCHEntry

func (h prepareCHEntryHeap) Len() int            { return len(h) }
func (h prepareCHEntryHeap) Less(i, j int) bool   { return h[i].Weight < h[j].Weight }
func (h prepareCHEntryHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i] }
func (h *prepareCHEntryHeap) Push(x any)          { *h = append(*h, x.(*PrepareCHEntry)) }
func (h *prepareCHEntryHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return x
}

// remove does an O(n) scan to find and remove an entry by pointer equality, matching
// Java's PriorityQueue.remove(Object) behavior.
func (h *prepareCHEntryHeap) remove(entry *PrepareCHEntry) {
	for i, e := range *h {
		if e == entry {
			heap.Remove(h, i)
			return
		}
	}
}
