package index

import (
	"fmt"
	"math"

	"gohopper/core/util"
)

// InvalidNode is the sentinel value for an uninitialized closest node.
const InvalidNode = -1

// Position indicates whether the query point is projected onto a tower node,
// pillar node, or somewhere within the closest edge.
//
// Due to precision differences it is hard to define when something is exactly
// 90 degrees or "on-node" like TOWER or PILLAR or if it is more "on-edge" (EDGE).
// The default mechanism is to prefer "on-edge" even if it could be 90 degrees.
// To prefer "on-node" you could use e.g. ConsiderEqual with a default precision
// of 1e-6.
type Position int

const (
	Edge   Position = iota // projected onto the interior of an edge segment
	Tower                  // snapped to a tower node (base or adj)
	Pillar                 // snapped to a pillar node (intermediate geometry point)
)

func (p Position) String() string {
	switch p {
	case Edge:
		return "EDGE"
	case Tower:
		return "TOWER"
	case Pillar:
		return "PILLAR"
	default:
		return "UNKNOWN"
	}
}

// Snap is the result of a LocationIndex lookup.
//
//	X = query coordinates
//	S = snapped coordinates: "snapping" real coords to road
//	N = tower or pillar node
//	T = closest tower node
//	XS = distance
//
//	X
//	|
//	T--S----N
type Snap struct {
	queryPoint      util.GHPoint
	queryDistance    float64
	wayIndex        int
	closestNode     int
	closestEdge     util.EdgeIteratorState
	snappedPoint    *util.GHPoint3D
	snappedPosition Position
}

// NewSnap creates a new Snap for the given query coordinates with default field values.
func NewSnap(queryLat, queryLon float64) *Snap {
	return &Snap{
		queryPoint:   util.GHPoint{Lat: queryLat, Lon: queryLon},
		queryDistance: math.MaxFloat64,
		wayIndex:     -1,
		closestNode:  InvalidNode,
	}
}

// GetClosestNode returns the closest matching node. This is either a tower node
// of the base graph or a virtual node. Returns InvalidNode if nothing was found;
// this should be avoided via a call to IsValid.
func (s *Snap) GetClosestNode() int { return s.closestNode }

// SetClosestNode sets the closest node.
func (s *Snap) SetClosestNode(node int) { s.closestNode = node }

// GetQueryDistance returns the distance from the query point to the snapped
// coordinates, in meters.
func (s *Snap) GetQueryDistance() float64 { return s.queryDistance }

// SetQueryDistance sets the query-to-snap distance.
func (s *Snap) SetQueryDistance(dist float64) { s.queryDistance = dist }

// GetWayIndex returns the way geometry index.
func (s *Snap) GetWayIndex() int { return s.wayIndex }

// SetWayIndex sets the way geometry index.
func (s *Snap) SetWayIndex(wayIndex int) { s.wayIndex = wayIndex }

// GetSnappedPosition returns the position type: Edge (0), Tower (1), or Pillar (2).
func (s *Snap) GetSnappedPosition() Position { return s.snappedPosition }

// SetSnappedPosition sets the snapped position type.
func (s *Snap) SetSnappedPosition(pos Position) { s.snappedPosition = pos }

// IsValid returns true if a closest node was found.
func (s *Snap) IsValid() bool { return s.closestNode >= 0 }

// GetClosestEdge returns the closest edge.
func (s *Snap) GetClosestEdge() util.EdgeIteratorState { return s.closestEdge }

// SetClosestEdge sets the closest edge.
func (s *Snap) SetClosestEdge(edge util.EdgeIteratorState) { s.closestEdge = edge }

// GetQueryPoint returns the original query point.
func (s *Snap) GetQueryPoint() util.GHPoint { return s.queryPoint }

// GetSnappedPoint returns the position of the query point "snapped" to a close
// road segment or node. CalcSnappedPoint must be called first; otherwise this
// method panics.
func (s *Snap) GetSnappedPoint() util.GHPoint3D {
	if s.snappedPoint == nil {
		panic("calculate snapped point before")
	}
	return *s.snappedPoint
}

// SetSnappedPoint sets the snapped point directly.
func (s *Snap) SetSnappedPoint(point util.GHPoint3D) { s.snappedPoint = &point }

// CalcSnappedPoint calculates the closest point on the edge from the query
// point. If the crossing point is too close to a tower or pillar node this
// method may change the snapped position and way index.
func (s *Snap) CalcSnappedPoint(distCalc util.DistanceCalc) {
	if s.closestEdge == nil {
		panic("no closest edge")
	}
	if s.snappedPoint != nil {
		panic("calculate snapped point only once")
	}

	fullPL := s.closestEdge.FetchWayGeometry(util.FetchModeAll)
	tmpLat := fullPL.GetLat(s.wayIndex)
	tmpLon := fullPL.GetLon(s.wayIndex)
	tmpEle := fullPL.GetEle(s.wayIndex)

	if s.snappedPosition != Edge {
		s.snappedPoint = &util.GHPoint3D{GHPoint: util.GHPoint{Lat: tmpLat, Lon: tmpLon}, Ele: tmpEle}
		return
	}

	queryLat := s.queryPoint.Lat
	queryLon := s.queryPoint.Lon
	adjLat := fullPL.GetLat(s.wayIndex + 1)
	adjLon := fullPL.GetLon(s.wayIndex + 1)

	if distCalc.ValidEdgeDistance(queryLat, queryLon, tmpLat, tmpLon, adjLat, adjLon) {
		crossingPoint := distCalc.CalcCrossingPointToEdge(queryLat, queryLon, tmpLat, tmpLon, adjLat, adjLon)
		adjEle := fullPL.GetEle(s.wayIndex + 1)

		// Prevent extra virtual nodes and very short virtual edges when the
		// snap/crossing point is very close to a tower node. Since we delayed
		// the calculation of the crossing point until here, we need to correct
		// the Snap position in these cases. Note that it is possible that the
		// query point is very far from the tower node, but the crossing point
		// is still very close to it.
		if ConsiderEqual(crossingPoint.Lat, crossingPoint.Lon, tmpLat, tmpLon) {
			s.snappedPoint = &util.GHPoint3D{GHPoint: util.GHPoint{Lat: tmpLat, Lon: tmpLon}, Ele: tmpEle}
			if s.wayIndex == 0 {
				s.snappedPosition = Tower
				s.closestNode = s.closestEdge.GetBaseNode()
			} else {
				s.snappedPosition = Pillar
			}
		} else if ConsiderEqual(crossingPoint.Lat, crossingPoint.Lon, adjLat, adjLon) {
			s.wayIndex++
			s.snappedPoint = &util.GHPoint3D{GHPoint: util.GHPoint{Lat: adjLat, Lon: adjLon}, Ele: adjEle}
			if s.wayIndex == fullPL.Size()-1 {
				s.snappedPosition = Tower
				s.closestNode = s.closestEdge.GetAdjNode()
			} else {
				s.snappedPosition = Pillar
			}
		} else {
			s.snappedPoint = &util.GHPoint3D{
				GHPoint: util.GHPoint{Lat: crossingPoint.Lat, Lon: crossingPoint.Lon},
				Ele:     (tmpEle + adjEle) / 2,
			}
		}
	}
	// If validEdgeDistance is false for an EDGE position, something is wrong
	// (should not happen). The Java source has an assert here; in Go we
	// silently leave snappedPoint unset, which will cause GetSnappedPoint to
	// panic if called.
}

// ConsiderEqual returns true if the two lat/lon pairs are within 1e-6 of each
// other in both dimensions.
func ConsiderEqual(lat, lon, lat2, lon2 float64) bool {
	return math.Abs(lat-lat2) < 1e-6 && math.Abs(lon-lon2) < 1e-6
}

// String returns a human-readable representation of the snap.
func (s *Snap) String() string {
	if s.closestEdge != nil {
		return fmt.Sprintf("%s, %d %d:%d-%d snap: [%v, %v], query: [%v,%v]",
			s.snappedPosition,
			s.closestNode,
			s.closestEdge.GetEdge(),
			s.closestEdge.GetBaseNode(),
			s.closestEdge.GetAdjNode(),
			util.Round6(s.snappedPoint.Lat),
			util.Round6(s.snappedPoint.Lon),
			util.Round6(s.queryPoint.Lat),
			util.Round6(s.queryPoint.Lon),
		)
	}
	return fmt.Sprintf("%d, %s, %d", s.closestNode, s.queryPoint, s.wayIndex)
}
