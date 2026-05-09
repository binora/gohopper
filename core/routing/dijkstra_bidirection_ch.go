package routing

import "gohopper/core/storage"

const chStallOnDemandPrecision = 0.001

// DijkstraBidirectionCH is the node-based bidirectional CH Dijkstra query
// algorithm with stall-on-demand enabled.
type DijkstraBidirectionCH struct {
	*DijkstraBidirectionCHNoSOD
}

func NewDijkstraBidirectionCH(graph storage.RoutingCHGraph) *DijkstraBidirectionCH {
	algo := &DijkstraBidirectionCH{
		DijkstraBidirectionCHNoSOD: NewDijkstraBidirectionCHNoSOD(graph),
	}
	algo.Name = "dijkstrabi|ch"
	algo.FromEntryCanBeSkipped = algo.fromEntryCanBeSkipped
	algo.ToEntryCanBeSkipped = algo.toEntryCanBeSkipped
	return algo
}

func (a *DijkstraBidirectionCH) fromEntryCanBeSkipped() bool {
	return a.entryIsStallable(a.currFrom, a.bestWeightMapFrom, a.inEdgeExplorer, false)
}

func (a *DijkstraBidirectionCH) toEntryCanBeSkipped() bool {
	return a.entryIsStallable(a.currTo, a.bestWeightMapTo, a.outEdgeExplorer, true)
}

func (a *DijkstraBidirectionCH) entryIsStallable(entry *SPTEntry, bestWeightMap map[int]*SPTEntry, edgeExplorer storage.RoutingCHEdgeExplorer, reverse bool) bool {
	iter := edgeExplorer.SetBaseNode(entry.AdjNode)
	for iter.Next() {
		if iter.GetEdge() == entry.Edge {
			continue
		}
		adjNodeEntry := bestWeightMap[iter.GetAdjNode()]
		if adjNodeEntry != nil &&
			adjNodeEntry.Weight+a.calcEdgeWeight(iter, !reverse, a.getIncomingEdge(entry))-entry.Weight < -chStallOnDemandPrecision {
			return true
		}
	}
	return false
}
