package routing

import (
	"math"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// HeadingResolver finds edges adjacent to a node whose heading differs from a
// given heading by more than a configurable tolerance.
type HeadingResolver struct {
	edgeExplorer util.EdgeExplorer
	toleranceRad float64
}

func NewHeadingResolver(graph storage.Graph) *HeadingResolver {
	return &HeadingResolver{
		edgeExplorer: graph.CreateEdgeExplorer(routingutil.AllEdges),
		toleranceRad: (math.Pi / 180) * 100, // default 100 degrees
	}
}

func (hr *HeadingResolver) SetTolerance(toleranceDeg float64) *HeadingResolver {
	hr.toleranceRad = (math.Pi / 180) * toleranceDeg
	return hr
}

// GetEdgesWithDifferentHeading returns edge IDs adjacent to baseNode whose
// heading differs from the given heading (north-based azimuth, 0..360) by more
// than the configured tolerance.
func (hr *HeadingResolver) GetEdgesWithDifferentHeading(baseNode int, heading float64) []int {
	xAxisAngle := util.ConvertAzimuth2XAxisAngle(heading)
	var edges []int
	iter := hr.edgeExplorer.SetBaseNode(baseNode)
	for iter.Next() {
		points := iter.FetchWayGeometry(util.FetchModeAll)
		orientation := util.CalcOrientation(
			points.GetLat(0), points.GetLon(0),
			points.GetLat(1), points.GetLon(1),
		)
		orientation = util.AlignOrientation(xAxisAngle, orientation)
		diff := math.Abs(orientation - xAxisAngle)
		if diff > hr.toleranceRad {
			edges = append(edges, iter.GetEdge())
		}
	}
	return edges
}
