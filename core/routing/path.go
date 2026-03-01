package routing

import (
	"fmt"
	"math"
	"slices"

	"gohopper/core/storage"
	"gohopper/core/util"
)

// EdgeVisitor is the callback used in ForEveryEdge.
type EdgeVisitor interface {
	Next(edge util.EdgeIteratorState, index int, prevEdgeID int)
	Finish()
}

// Path represents the result of a shortest path calculation.
type Path struct {
	Graph       storage.Graph
	nodeAccess  storage.NodeAccess
	Weight      float64
	Distance    float64
	Time        int64 // milliseconds
	EdgeIDs     []int
	FromNode    int
	EndNode     int
	Description []string
	Found       bool
	DebugInfo   string
}

// NewPath creates a Path with default values.
func NewPath(graph storage.Graph) *Path {
	return &Path{
		Graph:      graph,
		nodeAccess: graph.GetNodeAccess(),
		Weight:     math.MaxFloat64,
		FromNode:   -1,
		EndNode:    -1,
	}
}

func (p *Path) GetGraph() storage.Graph { return p.Graph }

// GetDescription returns the description of this route alternative.
func (p *Path) GetDescription() []string {
	if p.Description == nil {
		return []string{}
	}
	return p.Description
}

func (p *Path) SetDescription(desc []string) *Path {
	p.Description = desc
	return p
}

func (p *Path) GetEdges() []int { return p.EdgeIDs }

func (p *Path) SetEdges(edgeIDs []int) {
	p.EdgeIDs = edgeIDs
}

func (p *Path) AddEdge(edge int) {
	p.EdgeIDs = append(p.EdgeIDs, edge)
}

func (p *Path) GetEdgeCount() int { return len(p.EdgeIDs) }

func (p *Path) SetEndNode(end int) *Path {
	p.EndNode = end
	return p
}

func (p *Path) GetFromNode() int {
	if p.FromNode < 0 {
		panic("fromNode < 0 should not happen")
	}
	return p.FromNode
}

func (p *Path) SetFromNode(from int) *Path {
	p.FromNode = from
	return p
}

func (p *Path) SetFound(found bool) *Path {
	p.Found = found
	return p
}

func (p *Path) SetDistance(d float64) *Path {
	p.Distance = d
	return p
}

func (p *Path) AddDistance(d float64) *Path {
	p.Distance += d
	return p
}

func (p *Path) SetTime(t int64) *Path {
	p.Time = t
	return p
}

func (p *Path) AddTime(t int64) *Path {
	p.Time += t
	return p
}

func (p *Path) SetWeight(w float64) *Path {
	p.Weight = w
	return p
}

func (p *Path) SetDebugInfo(info string) {
	p.DebugInfo = info
}

// GetFinalEdge returns the final edge of the path.
func (p *Path) GetFinalEdge() util.EdgeIteratorState {
	return p.Graph.GetEdgeIteratorState(p.EdgeIDs[len(p.EdgeIDs)-1], p.EndNode)
}

// ForEveryEdge iterates over all edges in this path sorted from start to end
// and calls the visitor callback for every edge.
func (p *Path) ForEveryEdge(visitor EdgeVisitor) {
	tmpNode := p.GetFromNode()
	prevEdgeID := util.NoEdge
	for i, edgeID := range p.EdgeIDs {
		edgeBase := p.Graph.GetEdgeIteratorState(edgeID, tmpNode)
		if edgeBase == nil {
			panic(fmt.Sprintf("Edge %d was empty when requested with node %d, array index:%d, edges:%d",
				edgeID, tmpNode, i, len(p.EdgeIDs)))
		}

		tmpNode = edgeBase.GetBaseNode()
		// more efficient swap, currently not implemented for virtual edges
		edgeBase = p.Graph.GetEdgeIteratorState(edgeBase.GetEdge(), tmpNode)
		visitor.Next(edgeBase, i, prevEdgeID)

		prevEdgeID = edgeBase.GetEdge()
	}
	visitor.Finish()
}

// CalcEdges returns the list of all edge states.
func (p *Path) CalcEdges() []util.EdgeIteratorState {
	edges := make([]util.EdgeIteratorState, 0, len(p.EdgeIDs))
	if len(p.EdgeIDs) == 0 {
		return edges
	}
	p.ForEveryEdge(&edgeCollectorVisitor{edges: &edges})
	return edges
}

type edgeCollectorVisitor struct {
	edges *[]util.EdgeIteratorState
}

func (v *edgeCollectorVisitor) Next(edge util.EdgeIteratorState, _ int, _ int) {
	*v.edges = append(*v.edges, edge)
}

func (v *edgeCollectorVisitor) Finish() {}

// CalcNodes returns the tower node IDs in this path.
func (p *Path) CalcNodes() []int {
	nodes := make([]int, 0, len(p.EdgeIDs)+1)
	if len(p.EdgeIDs) == 0 {
		if p.Found {
			nodes = append(nodes, p.EndNode)
		}
		return nodes
	}

	tmpNode := p.GetFromNode()
	nodes = append(nodes, tmpNode)
	p.ForEveryEdge(&nodeCollectorVisitor{nodes: &nodes})
	return nodes
}

type nodeCollectorVisitor struct {
	nodes *[]int
}

func (v *nodeCollectorVisitor) Next(edge util.EdgeIteratorState, _ int, _ int) {
	*v.nodes = append(*v.nodes, edge.GetAdjNode())
}

func (v *nodeCollectorVisitor) Finish() {}

// CalcPoints builds the geometry of this path from edges.
func (p *Path) CalcPoints() *util.PointList {
	points := util.NewPointList(len(p.EdgeIDs)+1, p.nodeAccess.Is3D())
	if len(p.EdgeIDs) == 0 {
		if p.Found {
			p.addNodeToPointList(points, p.EndNode)
		}
		return points
	}

	tmpNode := p.GetFromNode()
	p.addNodeToPointList(points, tmpNode)
	p.ForEveryEdge(&pointCollectorVisitor{points: points})
	return points
}

func (p *Path) addNodeToPointList(pl *util.PointList, nodeID int) {
	if p.nodeAccess.Is3D() {
		pl.Add3D(p.nodeAccess.GetLat(nodeID), p.nodeAccess.GetLon(nodeID), p.nodeAccess.GetEle(nodeID))
	} else {
		pl.Add(p.nodeAccess.GetLat(nodeID), p.nodeAccess.GetLon(nodeID))
	}
}

type pointCollectorVisitor struct {
	points *util.PointList
}

func (v *pointCollectorVisitor) Next(edge util.EdgeIteratorState, _ int, _ int) {
	pl := edge.FetchWayGeometry(util.FetchModePillarAndAdj)
	for j := 0; j < pl.Size(); j++ {
		pt := pl.Get(j)
		if v.points.Is3D() {
			v.points.Add3D(pt.Lat, pt.Lon, pt.Ele)
		} else {
			v.points.Add(pt.Lat, pt.Lon)
		}
	}
}

func (v *pointCollectorVisitor) Finish() {}

// ReverseEdgeIDs reverses the edge ID list in-place.
func (p *Path) ReverseEdgeIDs() {
	slices.Reverse(p.EdgeIDs)
}

func (p *Path) String() string {
	return fmt.Sprintf("found: %t, weight: %v, time: %d, distance: %v, edges: %d",
		p.Found, p.Weight, p.Time, p.Distance, len(p.EdgeIDs))
}
