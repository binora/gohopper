package osm

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"

	"gohopper/core/reader"
	"gohopper/core/routing"
	"gohopper/core/routing/parsers"
	"gohopper/core/storage"
	"gohopper/core/util"
)

const (
	streetNameKey       = "street_name"
	streetRefKey        = "street_ref"
	streetDestKey       = "street_destination"
	streetDestRefKey    = "street_destination_ref"
	motorwayJunctionKey = "motorway_junction"
)

var wayNamePattern = regexp.MustCompile(`; *`)

// OSMReader reads an OSM file (PBF or XML) and builds a BaseGraph.
type OSMReader struct {
	graph      *storage.BaseGraph
	osmParsers *routing.OSMParsers
	config     routing.OSMReaderConfig
	distCalc   util.DistanceCalc
	nodeAccess storage.NodeAccess

	zeroCounter int
}

// NewOSMReader creates a new OSMReader that will populate the given graph.
func NewOSMReader(graph *storage.BaseGraph, osmParsers *routing.OSMParsers, config routing.OSMReaderConfig) *OSMReader {
	return &OSMReader{
		graph:      graph,
		osmParsers: osmParsers,
		config:     config,
		distCalc:   util.DistEarth,
		nodeAccess: graph.GetNodeAccess(),
	}
}

// ReadGraph reads the given OSM file and populates the graph.
func (r *OSMReader) ReadGraph(osmFile string) error {
	wsp := NewWaySegmentParserBuilder(r.graph.GetNodeAccess(), r.graph.GetDirectory()).
		SetElevationProvider(func(node *reader.ReaderNode) float64 { return 0 }).
		SetWayFilter(r.acceptWay).
		SetSplitNodeFilter(r.isBarrierNode).
		SetWayPreprocessor(r.preprocessWay).
		SetRelationPreprocessor(func(rel *reader.ReaderRelation) {}).
		SetRelationProcessor(func(rel *reader.ReaderRelation, getNodeID func(int64) int) {}).
		SetEdgeHandler(r.addEdge).
		Build()

	if err := wsp.ReadOSM(osmFile); err != nil {
		return fmt.Errorf("error reading OSM file %s: %w", osmFile, err)
	}

	if r.graph.GetNodes() == 0 {
		return fmt.Errorf("graph after reading OSM must not be empty")
	}

	log.Printf("Finished reading OSM file: %s, nodes: %d, edges: %d, zero distance edges: %d",
		osmFile, r.graph.GetNodes(), r.graph.GetEdges(), r.zeroCounter)
	return nil
}

// acceptWay returns true if the way should be included in the graph.
func (r *OSMReader) acceptWay(way *reader.ReaderWay) bool {
	if len(way.Nodes) < 2 {
		return false
	}
	if !way.HasTags() {
		return false
	}
	return r.osmParsers.AcceptWay(way)
}

// isBarrierNode returns true if the node should cause a way split.
func (r *OSMReader) isBarrierNode(node *reader.ReaderNode) bool {
	return node.HasTag("barrier") || node.HasTag("ford")
}

// preprocessWay enriches way tags before edge creation (names, ferry speed from duration).
func (r *OSMReader) preprocessWay(way *reader.ReaderWay, coordSupplier CoordinateSupplier, nodeTagSupplier NodeTagSupplier) {
	keyValues := make(map[string]any)
	if r.config.ParseWayNames {
		r.parseWayNames(way, keyValues, nodeTagSupplier)
	}
	way.SetTag("key_values", keyValues)

	if !r.isCalculateWayDistance(way) {
		return
	}

	distance := r.calcWayDistance(way, coordSupplier)
	if math.IsNaN(distance) {
		log.Printf("Could not determine distance for OSM way: %d", way.GetID())
		return
	}
	way.SetTag("way_distance", distance)

	durationTag := way.GetTag("duration")
	if durationTag == "" {
		if parsers.IsFerry(way) && distance > 500_000 {
			log.Printf("Long ferry OSM way without duration tag: %d, distance: %d km", way.GetID(), int(distance/1000.0))
		}
		return
	}

	durationSeconds, err := parseDuration(durationTag)
	if err != nil {
		log.Printf("Could not parse duration tag '%s' in OSM way: %d", durationTag, way.GetID())
		return
	}

	durationHours := float64(durationSeconds) / 3600.0
	speedKmH := (distance / 1000.0) / durationHours
	if speedKmH < 0.1 {
		log.Printf("Unrealistic low speed from duration. OSM way: %d, duration=%s, distance=%f m", way.GetID(), durationTag, distance)
		return
	}
	way.SetTag("speed_from_duration", speedKmH)
}

func (r *OSMReader) parseWayNames(way *reader.ReaderWay, keyValues map[string]any, nodeTagSupplier NodeTagSupplier) {
	name := r.resolveWayName(way)
	if name != "" {
		keyValues[streetNameKey] = name
	}

	if ref := fixWayName(way.GetTag("ref")); ref != "" {
		keyValues[streetRefKey] = ref
	}
	if way.HasTag("destination:ref") {
		keyValues[streetDestRefKey] = fixWayName(way.GetTag("destination:ref"))
	}
	if way.HasTag("destination") {
		keyValues[streetDestKey] = fixWayName(way.GetTag("destination"))
	}

	r.copyMotorwayJunctionName(way, keyValues, nodeTagSupplier)
}

// resolveWayName returns the best available name for the way, preferring the
// configured language over the default "name" tag.
func (r *OSMReader) resolveWayName(way *reader.ReaderWay) string {
	if r.config.PreferredLanguage != "" {
		if name := fixWayName(way.GetTag("name:" + r.config.PreferredLanguage)); name != "" {
			return name
		}
	}
	return fixWayName(way.GetTag("name"))
}

// copyMotorwayJunctionName copies the junction node name into the way's key-values
// when the first node of a motorway/motorway_link is tagged as a motorway_junction.
func (r *OSMReader) copyMotorwayJunctionName(way *reader.ReaderWay, keyValues map[string]any, nodeTagSupplier NodeTagSupplier) {
	if len(way.Nodes) == 0 {
		return
	}
	if !way.HasTag("highway", "motorway") && !way.HasTag("highway", "motorway_link") {
		return
	}
	nodeTags := nodeTagSupplier(way.Nodes[0])
	nodeName, _ := nodeTags["name"].(string)
	nodeHighway, _ := nodeTags["highway"].(string)
	if nodeName != "" && nodeHighway == "motorway_junction" {
		keyValues[motorwayJunctionKey] = nodeName
	}
}

func (r *OSMReader) isCalculateWayDistance(way *reader.ReaderWay) bool {
	return parsers.IsFerry(way)
}

func (r *OSMReader) calcWayDistance(way *reader.ReaderWay, coordSupplier CoordinateSupplier) float64 {
	nodes := way.Nodes
	prevPoint := coordSupplier(nodes[0])
	if prevPoint == nil {
		return math.NaN()
	}
	distance := 0.0
	for i := 1; i < len(nodes); i++ {
		point := coordSupplier(nodes[i])
		if point == nil {
			return math.NaN()
		}
		distance += r.distCalc.CalcDist(prevPoint.Lat, prevPoint.Lon, point.Lat, point.Lon)
		prevPoint = point
	}
	return distance
}

// addEdge creates an edge in the graph for a way segment.
func (r *OSMReader) addEdge(fromIndex, toIndex int, pointList *util.PointList, way *reader.ReaderWay, nodeTags []map[string]any) {
	if fromIndex < 0 || toIndex < 0 {
		panic(fmt.Sprintf("to or from index is invalid for this edge %d->%d", fromIndex, toIndex))
	}

	distance := r.distCalc.CalcPointListDistance(pointList)

	if distance < 0.001 {
		r.zeroCounter++
		distance = 0.001
	}

	maxDistance := float64(math.MaxInt32-1) / 1000.0
	if math.IsNaN(distance) {
		log.Printf("Bug in OSM or GoHopper. Illegal tower node distance %f reset to 1m, osm way %d", distance, way.GetID())
		distance = 1
	}

	if math.IsInf(distance, 0) || distance > maxDistance {
		log.Printf("Bug in OSM or GoHopper. Too big tower node distance %f reset to large value, osm way %d", distance, way.GetID())
		distance = maxDistance
	}

	// Set artificial way tags
	way.SetTag("node_tags", nodeTags)
	way.SetTag("edge_distance", distance)
	way.SetTag("point_list", pointList)

	edge := r.graph.Edge(fromIndex, toIndex).SetDistance(distance)
	r.osmParsers.HandleWayTags(edge.GetEdge(), r.graph.Store, way, storage.EmptyIntsRef)

	if keyValues, ok := way.GetTagWithDefault("key_values", nil).(map[string]any); ok && len(keyValues) > 0 {
		edge.SetKeyValues(keyValues)
	}

	// Store pillar node geometry (strip first and last tower nodes).
	if pointList.Size() > 2 {
		edge.SetWayGeometry(pointList.Copy(1, pointList.Size()-1))
	}
}

// fixWayName normalizes a way name by replacing semicolons with commas.
func fixWayName(str string) string {
	if str == "" {
		return ""
	}
	s := wayNamePattern.ReplaceAllString(str, ", ")
	// Truncate overly long strings
	if len(s) > 256 {
		s = s[:256]
	}
	return strings.TrimSpace(s)
}
