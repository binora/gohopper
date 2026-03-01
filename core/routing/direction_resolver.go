package routing

import (
	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// DirectionResolver determines pairs of in/out edges at a graph node, separated
// into "right" and "left" relative to a given location next to the road.
type DirectionResolver struct {
	edgeExplorer util.EdgeExplorer
	nodeAccess   storage.NodeAccess
	isAccessible routingutil.DirectedEdgeFilter
}

func NewDirectionResolver(graph storage.Graph, isAccessible routingutil.DirectedEdgeFilter) *DirectionResolver {
	return &DirectionResolver{
		edgeExplorer: graph.CreateEdgeExplorer(routingutil.AllEdges),
		nodeAccess:   graph.GetNodeAccess(),
		isAccessible: isAccessible,
	}
}

func (dr *DirectionResolver) ResolveDirections(node int, location util.GHPoint) DirectionResolverResult {
	adj := dr.calcAdjEdges(node)
	if adj.numStandardEdges == 0 || !adj.hasInEdges() || !adj.hasOutEdges() || len(adj.nextPoints) == 0 {
		return Impossible()
	}
	if adj.numZeroDistanceEdges > 0 {
		return Unrestricted()
	}

	snappedPoint := drPoint{lat: dr.nodeAccess.GetLat(node), lon: dr.nodeAccess.GetLon(node)}
	if adj.containsPoint(snappedPoint) {
		panic("pillar node of adjacent edge matches snapped point, this should not happen")
	}

	switch len(adj.nextPoints) {
	case 1:
		neighbor := adj.nextPoints[0]
		inEdges := adj.getInEdges(neighbor)
		outEdges := adj.getOutEdges(neighbor)
		if len(inEdges) > 1 || len(outEdges) > 1 {
			return Unrestricted()
		}
		return Restricted(inEdges[0].edgeID, outEdges[0].edgeID, inEdges[0].edgeID, outEdges[0].edgeID)

	case 2:
		p1 := adj.nextPoints[0]
		p2 := adj.nextPoints[1]
		in1 := adj.getInEdges(p1)
		in2 := adj.getInEdges(p2)
		out1 := adj.getOutEdges(p1)
		out2 := adj.getOutEdges(p2)
		if len(in1) > 1 || len(in2) > 1 || len(out1) > 1 || len(out2) > 1 {
			return Unrestricted()
		}
		if len(in1)+len(in2) == 0 || len(out1)+len(out2) == 0 {
			panic("there has to be at least one in and one out edge when there are two next points")
		}
		if len(in1)+len(out1) == 0 || len(in2)+len(out2) == 0 {
			panic("there has to be at least one in or one out edge for each of the two next points")
		}

		locationPoint := drPoint{lat: location.Lat, lon: location.Lon}
		if len(in1) == 0 || len(out2) == 0 {
			return dr.resolveDirections2(snappedPoint, locationPoint, in2[0], out1[0])
		}
		if len(in2) == 0 || len(out1) == 0 {
			return dr.resolveDirections2(snappedPoint, locationPoint, in1[0], out2[0])
		}
		return dr.resolveDirections4(snappedPoint, locationPoint, in1[0], out2[0], in2[0].edgeID, out1[0].edgeID)

	default:
		return Unrestricted()
	}
}

func (dr *DirectionResolver) resolveDirections2(snappedPoint, queryPoint drPoint, inEdge, outEdge drEdge) DirectionResolverResult {
	if isOnRightLane(queryPoint, snappedPoint, inEdge.nextPoint, outEdge.nextPoint) {
		return OnlyRight(inEdge.edgeID, outEdge.edgeID)
	}
	return OnlyLeft(inEdge.edgeID, outEdge.edgeID)
}

func (dr *DirectionResolver) resolveDirections4(snappedPoint, queryPoint drPoint, inEdge, outEdge drEdge, altInEdge, altOutEdge int) DirectionResolverResult {
	if isOnRightLane(queryPoint, snappedPoint, inEdge.nextPoint, outEdge.nextPoint) {
		return Restricted(inEdge.edgeID, outEdge.edgeID, altInEdge, altOutEdge)
	}
	return Restricted(altInEdge, altOutEdge, inEdge.edgeID, outEdge.edgeID)
}

func isOnRightLane(queryPoint, snappedPoint, inPoint, outPoint drPoint) bool {
	qX := queryPoint.lon - snappedPoint.lon
	qY := queryPoint.lat - snappedPoint.lat
	iX := inPoint.lon - snappedPoint.lon
	iY := inPoint.lat - snappedPoint.lat
	oX := outPoint.lon - snappedPoint.lon
	oY := outPoint.lat - snappedPoint.lat
	return !util.IsClockwise(iX, iY, oX, oY, qX, qY)
}

func (dr *DirectionResolver) calcAdjEdges(node int) *adjacentEdges {
	adj := &adjacentEdges{}
	iter := dr.edgeExplorer.SetBaseNode(node)
	for iter.Next() {
		isIn := dr.isAccessible(iter, true)
		isOut := dr.isAccessible(iter, false)
		if !isIn && !isOut {
			continue
		}

		geometry := iter.FetchWayGeometry(util.FetchModeAll)
		nextPointLat := geometry.GetLat(1)
		nextPointLon := geometry.GetLon(1)

		isZeroDistanceEdge := false
		if util.EqualsEps(nextPointLat, geometry.GetLat(0)) &&
			util.EqualsEps(nextPointLon, geometry.GetLon(0)) {
			if geometry.Size() > 2 {
				nextPointLat = geometry.GetLat(2)
				nextPointLon = geometry.GetLon(2)
			} else if geometry.Size() == 2 {
				isZeroDistanceEdge = true
			} else {
				panic("geometry has less than two points")
			}
		}

		nextPoint := drPoint{lat: nextPointLat, lon: nextPointLon}
		edge := drEdge{edgeID: iter.GetEdge(), adjNode: iter.GetAdjNode(), nextPoint: nextPoint}
		adj.addEdge(edge, isIn, isOut)

		if isZeroDistanceEdge {
			adj.numZeroDistanceEdges++
		} else {
			adj.numStandardEdges++
		}
	}
	return adj
}

type drPoint struct {
	lat, lon float64
}

func drPointsEqual(a, b drPoint) bool {
	return util.EqualsEps(a.lat, b.lat) && util.EqualsEps(a.lon, b.lon)
}

type drEdge struct {
	edgeID   int
	adjNode  int
	nextPoint drPoint
}

type adjacentEdges struct {
	inEdges              []drEdge
	inEdgePoints         []drPoint
	outEdges             []drEdge
	outEdgePoints        []drPoint
	nextPoints           []drPoint
	numStandardEdges     int
	numZeroDistanceEdges int
}

func (a *adjacentEdges) addEdge(edge drEdge, isIn, isOut bool) {
	if isIn {
		a.inEdges = append(a.inEdges, edge)
		a.inEdgePoints = append(a.inEdgePoints, edge.nextPoint)
	}
	if isOut {
		a.outEdges = append(a.outEdges, edge)
		a.outEdgePoints = append(a.outEdgePoints, edge.nextPoint)
	}
	a.addNextPoint(edge.nextPoint)
}

func (a *adjacentEdges) addNextPoint(p drPoint) {
	for _, existing := range a.nextPoints {
		if drPointsEqual(existing, p) {
			return
		}
	}
	a.nextPoints = append(a.nextPoints, p)
}

func (a *adjacentEdges) containsPoint(p drPoint) bool {
	for _, existing := range a.nextPoints {
		if drPointsEqual(existing, p) {
			return true
		}
	}
	return false
}

func (a *adjacentEdges) getInEdges(p drPoint) []drEdge {
	var result []drEdge
	for i, pt := range a.inEdgePoints {
		if drPointsEqual(pt, p) {
			result = append(result, a.inEdges[i])
		}
	}
	return result
}

func (a *adjacentEdges) getOutEdges(p drPoint) []drEdge {
	var result []drEdge
	for i, pt := range a.outEdgePoints {
		if drPointsEqual(pt, p) {
			result = append(result, a.outEdges[i])
		}
	}
	return result
}

func (a *adjacentEdges) hasInEdges() bool {
	return len(a.inEdges) > 0
}

func (a *adjacentEdges) hasOutEdges() bool {
	return len(a.outEdges) > 0
}
