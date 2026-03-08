package routing

import (
	"fmt"
	"math"
	"strings"

	"gohopper/core/config"
	"gohopper/core/routing/querygraph"
	routingutil "gohopper/core/routing/util"
	"gohopper/core/routing/weighting"
	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
	webapi "gohopper/web-api"
)

type Router struct {
	graph            *storage.BaseGraph
	locationIdx      index.LocationIndex
	routerConfig     RouterConfig
	profilesByName   map[string]config.Profile
	weightingFactory weighting.WeightingFactory
	encodingManager  *routingutil.EncodingManager
}

func NewRouter(
	graph *storage.BaseGraph,
	locationIdx index.LocationIndex,
	routerConfig RouterConfig,
	profilesByName map[string]config.Profile,
	weightingFactory weighting.WeightingFactory,
	encodingManager *routingutil.EncodingManager,
) *Router {
	return &Router{
		graph:            graph,
		locationIdx:      locationIdx,
		routerConfig:     routerConfig,
		profilesByName:   profilesByName,
		weightingFactory: weightingFactory,
		encodingManager:  encodingManager,
	}
}

func (r *Router) Route(request webapi.GHRequest) webapi.GHResponse {
	response := webapi.NewGHResponse()

	for _, check := range []func(webapi.GHRequest) error{
		r.checkNoLegacyParameters,
		r.checkAtLeastOnePoint,
		r.checkHeadings,
		r.checkPointHints,
		r.checkCurbsides,
		r.checkNoBlockArea,
	} {
		if err := check(request); err != nil {
			response.AddError(err)
			return response
		}
	}
	if err := r.checkIfPointsAreInBoundsAndNotNull(request.Points); err != nil {
		response.AddError(err)
		return response
	}

	profile, ok := r.profilesByName[request.Profile]
	if !ok {
		response.AddError(fmt.Errorf("the requested profile '%s' does not exist.\nTo use this profile you need to add it to the configuration, see docs/core/profiles.md", request.Profile))
		return response
	}

	w := r.weightingFactory.CreateWeighting(profile, nil, false)
	tMode := routingutil.NodeBased
	if w.HasTurnCosts() {
		tMode = routingutil.EdgeBased
	}

	snaps := make([]*index.Snap, len(request.Points))
	for i, pt := range request.Points {
		snap := r.locationIdx.FindClosest(pt.Lat, pt.Lon, routingutil.AllEdges)
		if snap == nil || !snap.IsValid() {
			response.AddError(webapi.PointNotFoundError{
				Message: fmt.Sprintf("Cannot find point %d: %s", i, pt.String()),
				Point:   i,
			})
			return response
		}
		snaps[i] = snap
	}

	queryGraph := querygraph.CreateFromSnaps(r.graph, snaps)
	algoOpts := r.buildAlgoOpts(request, tMode)
	algoFactory := &RoutingAlgorithmFactorySimple{}
	pathCalculator := NewFlexiblePathCalculator(queryGraph, algoFactory, w, algoOpts)

	totalVisitedNodes := 0
	var allPaths []*Path
	for i := 0; i < len(snaps)-1; i++ {
		fromNode := snaps[i].GetClosestNode()
		toNode := snaps[i+1].GetClosestNode()
		paths := pathCalculator.CalcPaths(fromNode, toNode, NewEdgeRestrictions())
		if len(paths) == 0 || !paths[0].Found {
			response.AddError(webapi.ConnectionNotFoundError{
				Message: fmt.Sprintf("Connection between locations not found for leg %d", i),
			})
			return response
		}
		allPaths = append(allPaths, paths[0])
		totalVisitedNodes += pathCalculator.GetVisitedNodes()

		// FlexiblePathCalculator is single-use per algo; recreate for next leg.
		if i < len(snaps)-2 {
			pathCalculator = NewFlexiblePathCalculator(queryGraph, algoFactory, w, algoOpts)
		}
	}

	responsePath := r.buildResponsePath(request, allPaths, snaps)
	response.Add(responsePath)
	response.Hints.PutObject("visited_nodes.sum", totalVisitedNodes)
	avg := 0
	if len(snaps) > 1 {
		avg = totalVisitedNodes / (len(snaps) - 1)
	}
	response.Hints.PutObject("visited_nodes.average", avg)
	return response
}

func (r *Router) buildAlgoOpts(request webapi.GHRequest, tMode routingutil.TraversalMode) AlgorithmOptions {
	opts := NewAlgorithmOptions()
	opts.TraversalMode = tMode
	opts.MaxVisitedNodes = r.routerConfig.MaxVisitedNodes
	opts.TimeoutMillis = r.routerConfig.TimeoutMillis

	if request.Algorithm != "" {
		opts.Algorithm = strings.ToLower(request.Algorithm)
	}
	return opts
}

func (r *Router) buildResponsePath(request webapi.GHRequest, paths []*Path, snaps []*index.Snap) webapi.ResponsePath {
	var totalDistance float64
	var totalTime int64
	var totalWeight float64

	allPoints := util.NewPointList(0, false)
	// Track waypoint indices for simplification partitioning.
	waypointIndices := []int{0}
	for i, p := range paths {
		totalDistance += p.Distance
		totalTime += p.Time
		totalWeight += p.Weight

		if request.Options.CalcPoints {
			legPoints := p.CalcPoints()
			for j := 0; j < legPoints.Size(); j++ {
				if i > 0 && j == 0 {
					continue
				}
				pt := legPoints.Get(j)
				allPoints.Add(pt.Lat, pt.Lon)
			}
			waypointIndices = append(waypointIndices, allPoints.Size()-1)
		}
	}

	// Apply RDP simplification if configured.
	if r.routerConfig.SimplifyResponse && request.Options.CalcPoints && allPoints.Size() > 2 {
		wayPointMaxDist := request.Hints.GetFloat64("way_point_max_distance", 0.5)
		rdp := util.NewRamerDouglasPeucker()
		rdp.SetMaxDistance(wayPointMaxDist)

		// Build waypoint partition from indices.
		wpIntervals := make([]util.Interval, len(waypointIndices)-1)
		for i := range wpIntervals {
			wpIntervals[i] = util.Interval{Start: waypointIndices[i], End: waypointIndices[i+1]}
		}
		partition := &waypointPartition{intervals: wpIntervals}
		util.SimplifyPath(allPoints, []util.Partition{partition}, rdp)

		// Rebuild waypoint indices from simplified partition.
		waypointIndices = make([]int, 0, len(wpIntervals)+1)
		waypointIndices = append(waypointIndices, partition.intervals[0].Start)
		for _, iv := range partition.intervals {
			waypointIndices = append(waypointIndices, iv.End)
		}
	}

	waypoints := make([]util.GHPoint, len(snaps))
	for i, s := range snaps {
		sp := s.GetSnappedPoint()
		waypoints[i] = sp.GHPoint
	}

	rp := webapi.ResponsePath{
		Distance:      totalDistance,
		Time:          totalTime,
		Weight:        totalWeight,
		PointsEncoded: request.Options.PointsEncoded,
	}

	if request.Options.CalcPoints && allPoints.Size() > 0 {
		ghPoints := allPoints.ToGHPoints()

		if request.Options.PointsEncoded {
			rp.Points = util.EncodePolylineFromPoints(ghPoints, request.Options.PointsEncodedMultiplier)
			rp.SnappedWaypoints = util.EncodePolylineFromPoints(waypoints, request.Options.PointsEncodedMultiplier)
		} else {
			rp.Points = map[string]any{"type": "LineString", "coordinates": toCoordinates(ghPoints)}
			rp.SnappedWaypoints = map[string]any{"type": "LineString", "coordinates": toCoordinates(waypoints)}
		}

		bboxVal := util.CalcBBox(ghPoints)
		rp.BBox = bboxVal.ToArray()
	}

	if request.Options.Instructions {
		rp.Instructions = buildSimpleInstructions(totalDistance, totalTime, allPoints.Size())
	}

	return rp
}

// waypointPartition implements util.Partition for waypoint interval tracking.
type waypointPartition struct {
	intervals []util.Interval
}

func (w *waypointPartition) Size() int {
	return len(w.intervals)
}

func (w *waypointPartition) GetIntervalLength(index int) int {
	return w.intervals[index].End - w.intervals[index].Start
}

func (w *waypointPartition) SetInterval(index, start, end int) {
	w.intervals[index].Start = start
	w.intervals[index].End = end
}

func buildSimpleInstructions(distance float64, timeMs int64, numPoints int) []webapi.Instruction {
	lastIdx := max(0, numPoints-1)
	return []webapi.Instruction{
		{Text: "Continue", Distance: distance, Time: timeMs, Interval: [2]int{0, lastIdx}, Sign: 0},
		{Text: "Arrive at destination", Distance: 0, Time: 0, Interval: [2]int{lastIdx, lastIdx}, Sign: 4},
	}
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
		return fmt.Errorf("you have to pass at least one point")
	}
	return nil
}

func (r *Router) checkIfPointsAreInBoundsAndNotNull(points []util.GHPoint) error {
	bounds := r.graph.GetBounds()
	for i, point := range points {
		if math.IsNaN(point.Lat) || math.IsNaN(point.Lon) {
			return fmt.Errorf("point %d is null", i)
		}
		if !bounds.Contains(point.Lat, point.Lon) {
			return webapi.PointOutOfBoundsError{Message: fmt.Sprintf("Point %d is out of bounds: %s, the bounds are: %+v", i, point.String(), bounds), Point: i}
		}
	}
	return nil
}

func (r *Router) checkHeadings(request webapi.GHRequest) error {
	if len(request.Headings) > 1 && len(request.Headings) != len(request.Points) {
		return fmt.Errorf("the number of 'heading' parameters must be zero, one or equal to the number of points (%d)", len(request.Points))
	}
	for i, h := range request.Headings {
		if !webapi.IsAzimuthValue(h) {
			return fmt.Errorf("heading for point %d must be in range [0,360) or NaN, but was: %v", i, h)
		}
	}
	return nil
}

func (r *Router) checkPointHints(request webapi.GHRequest) error {
	if len(request.PointHints) > 0 && len(request.PointHints) != len(request.Points) {
		return fmt.Errorf("if you pass point_hint, you need to pass exactly one hint for every point, empty hints will be ignored")
	}
	return nil
}

func (r *Router) checkCurbsides(request webapi.GHRequest) error {
	if len(request.Curbsides) > 0 && len(request.Curbsides) != len(request.Points) {
		return fmt.Errorf("if you pass curbside, you need to pass exactly one curbside for every point, empty curbsides will be ignored")
	}
	return nil
}

func (r *Router) checkNoBlockArea(request webapi.GHRequest) error {
	if request.Hints.Has("block_area") {
		return fmt.Errorf("the `block_area` parameter is no longer supported, use a custom model with `areas` instead")
	}
	return nil
}

func toCoordinates(points []util.GHPoint) [][]float64 {
	coords := make([][]float64, len(points))
	for i, p := range points {
		coords[i] = []float64{p.Lon, p.Lat}
	}
	return coords
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
