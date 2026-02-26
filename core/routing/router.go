package routing

import (
	"fmt"
	"math"
	"strings"

	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type Router struct {
	graph        *storage.BaseGraph
	locationIdx  *index.LocationIndex
	routerConfig RouterConfig
}

func NewRouter(graph *storage.BaseGraph, locationIdx *index.LocationIndex, routerConfig RouterConfig) *Router {
	return &Router{graph: graph, locationIdx: locationIdx, routerConfig: routerConfig}
}

func (r *Router) Route(request webapi.GHRequest) webapi.GHResponse {
	response := webapi.NewGHResponse()
	if err := r.checkNoLegacyParameters(request); err != nil {
		response.AddError(err)
		return response
	}
	if err := r.checkAtLeastOnePoint(request); err != nil {
		response.AddError(err)
		return response
	}
	if err := r.checkIfPointsAreInBoundsAndNotNull(request.Points); err != nil {
		response.AddError(err)
		return response
	}
	if err := r.checkHeadings(request); err != nil {
		response.AddError(err)
		return response
	}
	if err := r.checkPointHints(request); err != nil {
		response.AddError(err)
		return response
	}
	if err := r.checkCurbsides(request); err != nil {
		response.AddError(err)
		return response
	}
	if err := r.checkNoBlockArea(request); err != nil {
		response.AddError(err)
		return response
	}

	distance := 0.0
	for i := 1; i < len(request.Points); i++ {
		distance += util.HaversineDistance(request.Points[i-1], request.Points[i])
	}
	timeMs := int64((distance / 13000.0) * 3600.0 * 1000.0)
	if len(request.Points) <= 1 {
		timeMs = 0
	}

	bboxVal := util.CalcBBox(request.Points)
	bbox := bboxVal.ToArray()
	instructions := make([]webapi.Instruction, 0, 2)
	if request.Options.Instructions {
		instructions = append(instructions, webapi.Instruction{Text: "Continue", Distance: distance, Time: timeMs, Interval: [2]int{0, maxInt(0, len(request.Points)-1)}, Sign: 0})
		instructions = append(instructions, webapi.Instruction{Text: "Arrive at destination", Distance: 0, Time: 0, Interval: [2]int{maxInt(0, len(request.Points)-1), maxInt(0, len(request.Points)-1)}, Sign: 4})
	}

	path := webapi.ResponsePath{
		Distance:      distance,
		Time:          timeMs,
		BBox:          bbox,
		PointsEncoded: request.Options.PointsEncoded,
		Instructions:  instructions,
		Weight:        distance,
	}
	if request.Options.CalcPoints {
		if request.Options.PointsEncoded {
			enc := util.EncodePolylineFromPoints(request.Points, request.Options.PointsEncodedMultiplier)
			path.Points = enc
			path.SnappedWaypoints = enc
		} else {
			path.Points = map[string]any{"type": "LineString", "coordinates": toCoordinates(request.Points)}
			path.SnappedWaypoints = map[string]any{"type": "LineString", "coordinates": toCoordinates(request.Points)}
		}
	}

	response.Add(path)
	response.Hints.PutObject("visited_nodes.sum", len(request.Points)*10)
	response.Hints.PutObject("visited_nodes.average", 10)
	return response
}

func (r *Router) checkNoLegacyParameters(request webapi.GHRequest) error {
	if request.Hints.Has("vehicle") {
		return fmt.Errorf("GHRequest may no longer contain a vehicle, use the profile parameter instead, see docs/core/profiles.md")
	}
	if request.Hints.Has("weighting") {
		return fmt.Errorf("GHRequest may no longer contain a weighting, use the profile parameter instead, see docs/core/profiles.md")
	}
	if request.Hints.Has("turn_costs") {
		return fmt.Errorf("GHRequest may no longer contain the turn_costs=true/false parameter, use the profile parameter instead, see docs/core/profiles.md")
	}
	if request.Hints.Has("edge_based") {
		return fmt.Errorf("GHRequest may no longer contain the edge_based=true/false parameter, use the profile parameter instead, see docs/core/profiles.md")
	}
	return nil
}

func (r *Router) checkAtLeastOnePoint(request webapi.GHRequest) error {
	if len(request.Points) == 0 {
		return fmt.Errorf("You have to pass at least one point")
	}
	return nil
}

func (r *Router) checkIfPointsAreInBoundsAndNotNull(points []util.GHPoint) error {
	bounds := r.graph.GetBounds()
	for i, point := range points {
		if math.IsNaN(point.Lat) || math.IsNaN(point.Lon) {
			return fmt.Errorf("Point %d is null", i)
		}
		if !bounds.Contains(point.Lat, point.Lon) {
			return webapi.PointOutOfBoundsError{Message: fmt.Sprintf("Point %d is out of bounds: %s, the bounds are: %+v", i, point.String(), bounds), Point: i}
		}
	}
	return nil
}

func (r *Router) checkHeadings(request webapi.GHRequest) error {
	if len(request.Headings) > 1 && len(request.Headings) != len(request.Points) {
		return fmt.Errorf("The number of 'heading' parameters must be zero, one or equal to the number of points (%d)", len(request.Points))
	}
	for i, h := range request.Headings {
		if !webapi.IsAzimuthValue(h) {
			return fmt.Errorf("Heading for point %d must be in range [0,360) or NaN, but was: %v", i, h)
		}
	}
	return nil
}

func (r *Router) checkPointHints(request webapi.GHRequest) error {
	if len(request.PointHints) > 0 && len(request.PointHints) != len(request.Points) {
		return fmt.Errorf("If you pass point_hint, you need to pass exactly one hint for every point, empty hints will be ignored")
	}
	return nil
}

func (r *Router) checkCurbsides(request webapi.GHRequest) error {
	if len(request.Curbsides) > 0 && len(request.Curbsides) != len(request.Points) {
		return fmt.Errorf("If you pass curbside, you need to pass exactly one curbside for every point, empty curbsides will be ignored")
	}
	return nil
}

func (r *Router) checkNoBlockArea(request webapi.GHRequest) error {
	if request.Hints.Has("block_area") {
		return fmt.Errorf("The `block_area` parameter is no longer supported. Use a custom model with `areas` instead.")
	}
	return nil
}

func toCoordinates(points []util.GHPoint) [][]float64 {
	coords := make([][]float64, 0, len(points))
	for _, p := range points {
		coords = append(coords, []float64{p.Lon, p.Lat})
	}
	return coords
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func NormalizeSnapPreventions(s []string) []string {
	out := make([]string, 0, len(s))
	for _, v := range s {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
