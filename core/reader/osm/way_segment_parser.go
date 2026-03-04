package osm

import (
	"fmt"
	"log"

	"gohopper/core/reader"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// includeIfNodeTags is the set of node tag keys that trigger tag storage.
var includeIfNodeTags = map[string]bool{
	"barrier":  true,
	"highway":  true,
	"railway":  true,
	"crossing": true,
	"ford":     true,
}

// EdgeHandler is called for each way segment (edge) after splitting.
type EdgeHandler func(from, to int, pointList *util.PointList, way *reader.ReaderWay, nodeTags []map[string]any)

// CoordinateSupplier returns coordinates for a given OSM node ID, or nil if missing.
type CoordinateSupplier func(osmNodeID int64) *GHPoint3D

// NodeTagSupplier returns tags for a given OSM node ID.
type NodeTagSupplier func(osmNodeID int64) map[string]any

// WayPreprocessor is called for each accepted way during pass 2.
type WayPreprocessor func(way *reader.ReaderWay, coordSupplier CoordinateSupplier, tagSupplier NodeTagSupplier)

// RelationProcessor is called for each relation during pass 2.
// getNodeID maps an OSM node ID to the internal tower node index, or -1 if not a tower node.
type RelationProcessor func(relation *reader.ReaderRelation, getNodeID func(int64) int)

// WaySegmentParser reads an OSM file in two passes, splitting ways into segments at junctions.
type WaySegmentParser struct {
	nodeData             *OSMNodeData
	elevationProvider    func(node *reader.ReaderNode) float64
	wayFilter            func(way *reader.ReaderWay) bool
	splitNodeFilter      func(node *reader.ReaderNode) bool
	wayPreprocessor      WayPreprocessor
	relationPreprocessor func(relation *reader.ReaderRelation)
	relationProcessor    RelationProcessor
	edgeHandler          EdgeHandler
}

// WaySegmentParserBuilder constructs a WaySegmentParser with callbacks.
type WaySegmentParserBuilder struct {
	parser *WaySegmentParser
}

// NewWaySegmentParserBuilder creates a builder with the required dependencies.
func NewWaySegmentParserBuilder(nodeAccess storage.NodeAccess, dir storage.Directory) *WaySegmentParserBuilder {
	return &WaySegmentParserBuilder{
		parser: &WaySegmentParser{
			nodeData:             NewOSMNodeData(nodeAccess, dir),
			elevationProvider:    func(node *reader.ReaderNode) float64 { return 0 },
			wayFilter:            func(way *reader.ReaderWay) bool { return true },
			splitNodeFilter:      func(node *reader.ReaderNode) bool { return false },
			wayPreprocessor:      func(way *reader.ReaderWay, cs CoordinateSupplier, ts NodeTagSupplier) {},
			relationPreprocessor: func(relation *reader.ReaderRelation) {},
			relationProcessor:    func(relation *reader.ReaderRelation, getNodeID func(int64) int) {},
			edgeHandler: func(from, to int, pointList *util.PointList, way *reader.ReaderWay, nodeTags []map[string]any) {
				fmt.Printf("edge %d->%d (%d points)\n", from, to, pointList.Size())
			},
		},
	}
}

func (b *WaySegmentParserBuilder) SetElevationProvider(ep func(*reader.ReaderNode) float64) *WaySegmentParserBuilder {
	b.parser.elevationProvider = ep
	return b
}

func (b *WaySegmentParserBuilder) SetWayFilter(wf func(*reader.ReaderWay) bool) *WaySegmentParserBuilder {
	b.parser.wayFilter = wf
	return b
}

func (b *WaySegmentParserBuilder) SetSplitNodeFilter(f func(*reader.ReaderNode) bool) *WaySegmentParserBuilder {
	b.parser.splitNodeFilter = f
	return b
}

func (b *WaySegmentParserBuilder) SetWayPreprocessor(wp WayPreprocessor) *WaySegmentParserBuilder {
	b.parser.wayPreprocessor = wp
	return b
}

func (b *WaySegmentParserBuilder) SetRelationPreprocessor(rp func(*reader.ReaderRelation)) *WaySegmentParserBuilder {
	b.parser.relationPreprocessor = rp
	return b
}

func (b *WaySegmentParserBuilder) SetRelationProcessor(rp RelationProcessor) *WaySegmentParserBuilder {
	b.parser.relationProcessor = rp
	return b
}

func (b *WaySegmentParserBuilder) SetEdgeHandler(eh EdgeHandler) *WaySegmentParserBuilder {
	b.parser.edgeHandler = eh
	return b
}

func (b *WaySegmentParserBuilder) Build() *WaySegmentParser {
	return b.parser
}

// ReadOSM reads the OSM file in two passes, splitting ways at intersections.
func (p *WaySegmentParser) ReadOSM(osmFile string) error {
	if p.nodeData.GetNodeCount() > 0 {
		return fmt.Errorf("you can only run way segment parser once")
	}

	log.Printf("Start reading OSM file: '%s'", osmFile)

	// Pass 1: classify nodes, preprocess relations (skip nodes for speed)
	log.Println("pass1 - start")
	pass1 := &pass1Handler{parser: p}
	if err := ReadFile(osmFile, pass1, SkipOptions{SkipNodes: true}); err != nil {
		return fmt.Errorf("pass1 failed: %w", err)
	}

	log.Printf("Creating graph. Node count (pillar+tower): %d", p.nodeData.GetNodeCount())

	// Pass 2: store coordinates, split ways, create edges
	log.Println("pass2 - start")
	pass2 := &pass2Handler{parser: p}
	if err := ReadFile(osmFile, pass2, SkipOptionsNone()); err != nil {
		return fmt.Errorf("pass2 failed: %w", err)
	}

	p.nodeData.Release()
	log.Println("Finished reading OSM file.")
	return nil
}

// --- Pass 1: classify nodes, preprocess relations ---

type pass1Handler struct {
	parser           *WaySegmentParser
	handledWays      bool
	handledRelations bool
	wayCounter       int64
	acceptedWays     int64
}

func (h *pass1Handler) HandleNode(node *reader.ReaderNode) {}

func (h *pass1Handler) HandleWay(way *reader.ReaderWay) {
	if !h.handledWays {
		log.Println("pass1 - start reading OSM ways")
		h.handledWays = true
	}
	if h.handledRelations {
		panic("OSM way elements must be located before relation elements in OSM file")
	}

	h.wayCounter++
	if h.wayCounter%10_000_000 == 0 {
		log.Printf("pass1 - processed ways: %d, accepted: %d, way nodes: %d",
			h.wayCounter, h.acceptedWays, h.parser.nodeData.GetNodeCount())
	}

	if !h.parser.wayFilter(way) {
		return
	}
	h.acceptedWays++

	nodes := way.Nodes
	lastIdx := len(nodes) - 1
	for i, osmID := range nodes {
		isEnd := i == 0 || i == lastIdx
		var nodeType int64
		if isEnd {
			nodeType = EndNode
		} else {
			nodeType = IntermediateNode
		}
		h.parser.nodeData.SetOrUpdateNodeType(osmID, nodeType, func(prev int64) int64 {
			if prev == EndNode && isEnd {
				return ConnectionNode
			}
			return JunctionNode
		})
	}
}

func (h *pass1Handler) HandleRelation(relation *reader.ReaderRelation) {
	if !h.handledRelations {
		log.Println("pass1 - start reading OSM relations")
		h.handledRelations = true
	}
	h.parser.relationPreprocessor(relation)
}

func (h *pass1Handler) HandleFileHeader(header *OSMFileHeader) {}

func (h *pass1Handler) OnFinish() {
	log.Printf("pass1 - finished, processed ways: %d, accepted: %d, way nodes: %d",
		h.wayCounter, h.acceptedWays, h.parser.nodeData.GetNodeCount())
}

// --- Pass 2: store coordinates, split ways, create edges ---

type pass2Handler struct {
	parser            *WaySegmentParser
	handledNodes      bool
	handledWays       bool
	handledRelations  bool
	nodeCounter       int64
	acceptedNodes     int64
	ignoredSplitNodes int64
	wayCounter        int64
}

func (h *pass2Handler) HandleNode(node *reader.ReaderNode) {
	if !h.handledNodes {
		log.Println("pass2 - start reading OSM nodes")
		h.handledNodes = true
	}
	if h.handledWays {
		panic("OSM node elements must be located before way elements in OSM file")
	}
	if h.handledRelations {
		panic("OSM node elements must be located before relation elements in OSM file")
	}

	h.nodeCounter++
	if h.nodeCounter%10_000_000 == 0 {
		log.Printf("pass2 - processed nodes: %d, accepted: %d", h.nodeCounter, h.acceptedNodes)
	}

	nodeType := h.parser.nodeData.AddCoordinatesIfMapped(node.GetID(), node.Lat, node.Lon,
		func() float64 { return h.parser.elevationProvider(node) })
	if nodeType == EmptyNode {
		return
	}
	h.acceptedNodes++

	// Remember barrier/split nodes
	if h.parser.splitNodeFilter(node) {
		if nodeType == JunctionNode {
			h.ignoredSplitNodes++
		} else {
			h.parser.nodeData.SetSplitNode(node.GetID())
		}
	}

	// Store node tags if at least one important tag is included
	tags := node.GetTags()
	for key := range tags {
		if includeIfNodeTags[key] {
			delete(tags, "created_by")
			delete(tags, "source")
			delete(tags, "note")
			delete(tags, "fixme")
			h.parser.nodeData.SetTags(node.GetID(), tags)
			break
		}
	}
}

func (h *pass2Handler) HandleWay(way *reader.ReaderWay) {
	if !h.handledWays {
		log.Println("pass2 - start reading OSM ways")
		h.handledWays = true
	}
	if h.handledRelations {
		panic("OSM way elements must be located before relation elements in OSM file")
	}

	h.wayCounter++
	if h.wayCounter%10_000_000 == 0 {
		log.Printf("pass2 - processed ways: %d", h.wayCounter)
	}

	if !h.parser.wayFilter(way) {
		return
	}

	nd := h.parser.nodeData
	segment := make([]SegmentNode, len(way.Nodes))
	for i, osmID := range way.Nodes {
		segment[i] = SegmentNode{
			OSMNodeID: osmID,
			ID:        nd.GetID(osmID),
			Tags:      nd.GetTags(osmID),
		}
	}

	h.parser.wayPreprocessor(way,
		func(osmNodeID int64) *GHPoint3D {
			return nd.GetCoordinates(nd.GetID(osmNodeID))
		},
		func(osmNodeID int64) map[string]any {
			return nd.GetTags(osmNodeID)
		},
	)

	h.splitWayAtJunctionsAndEmptySections(segment, way)
}

func (h *pass2Handler) splitWayAtJunctionsAndEmptySections(fullSegment []SegmentNode, way *reader.ReaderWay) {
	var segment []SegmentNode
	for _, node := range fullSegment {
		if !IsNodeID(node.ID) {
			// Node exists in ways but not in nodes (e.g. OSM extract boundary).
			// Split the way here to avoid connecting exit/entry points with a straight line.
			if len(segment) > 1 {
				h.splitLoopSegments(segment, way)
				segment = nil
			}
		} else if IsTowerNode(node.ID) {
			if len(segment) > 0 {
				segment = append(segment, node)
				h.splitLoopSegments(segment, way)
				segment = nil
			}
			segment = append(segment, node)
		} else {
			segment = append(segment, node)
		}
	}
	// The last segment might end at the end of the way.
	if len(segment) > 1 {
		h.splitLoopSegments(segment, way)
	}
}

func (h *pass2Handler) splitLoopSegments(segment []SegmentNode, way *reader.ReaderWay) {
	if len(segment) < 2 {
		panic(fmt.Sprintf("Segment size must be >= 2, but was: %d", len(segment)))
	}

	isLoop := segment[0].OSMNodeID == segment[len(segment)-1].OSMNodeID
	if len(segment) == 2 && isLoop {
		log.Printf("Loop in OSM way: %d, will be ignored, duplicate node: %d", way.GetID(), segment[0].OSMNodeID)
	} else if isLoop {
		// Split loop into two segments
		h.splitSegmentAtSplitNodes(segment[:len(segment)-1], way)
		h.splitSegmentAtSplitNodes(segment[len(segment)-2:], way)
	} else {
		h.splitSegmentAtSplitNodes(segment, way)
	}
}

func (h *pass2Handler) splitSegmentAtSplitNodes(parentSegment []SegmentNode, way *reader.ReaderWay) {
	nd := h.parser.nodeData
	var segment []SegmentNode

	for i, node := range parentSegment {
		if !nd.IsSplitNode(node.OSMNodeID) {
			segment = append(segment, node)
			continue
		}

		// Consume the split-node marker so the barrier edge is only created once.
		nd.UnsetSplitNode(node.OSMNodeID)

		// Create two copies: one stays with the preceding segment, one starts the next.
		// When the barrier is at the end of the parent segment, swap so the copy sits
		// on the inside (facing the preceding segment).
		barrierFrom := node
		barrierTo := nd.AddCopyOfNode(node)
		if i == len(parentSegment)-1 {
			barrierFrom, barrierTo = barrierTo, barrierFrom
		}

		// Finish the preceding segment up to and including barrierFrom.
		if len(segment) > 0 {
			segment = append(segment, barrierFrom)
			h.handleSegment(segment, way)
			// handleSegment may promote pillar to tower; read back the updated node.
			barrierFrom = segment[len(segment)-1]
			segment = nil
		}

		// Emit the zero-length barrier edge.
		way.SetTag("gh:barrier_edge", "true")
		barrierSegment := []SegmentNode{barrierFrom, barrierTo}
		h.handleSegment(barrierSegment, way)
		way.RemoveTag("gh:barrier_edge")

		// Start the next segment from the barrier's outgoing copy.
		// handleSegment may have promoted pillar to tower; read back updated IDs.
		segment = []SegmentNode{barrierSegment[1]}
	}

	if len(segment) > 1 {
		h.handleSegment(segment, way)
	}
}

func (h *pass2Handler) handleSegment(segment []SegmentNode, way *reader.ReaderWay) {
	nd := h.parser.nodeData
	pointList := util.NewPointList(len(segment), nd.Is3D())
	nodeTags := make([]map[string]any, len(segment))
	from := -1
	to := -1

	for i := range segment {
		node := &segment[i]
		id := node.ID
		if !IsNodeID(id) {
			panic(fmt.Sprintf("Invalid id for node: %d when handling segment for way: %d", node.OSMNodeID, way.GetID()))
		}

		// Promote pillar nodes at segment boundaries to tower nodes.
		if IsPillarNode(id) && (i == 0 || i == len(segment)-1) {
			id = nd.ConvertPillarToTowerNode(id, node.OSMNodeID)
			node.ID = id
		}

		if i == 0 {
			from = nd.IDToTowerNode(id)
		} else if i == len(segment)-1 {
			to = nd.IDToTowerNode(id)
		} else if IsTowerNode(id) {
			panic(fmt.Sprintf("Tower nodes should only appear at the end of segments, way: %d", way.GetID()))
		}

		nd.AddCoordinatesToPointList(id, pointList)
		nodeTags[i] = node.Tags
	}

	if from < 0 || to < 0 {
		panic(fmt.Sprintf("The first and last nodes of a segment must be tower nodes, way: %d", way.GetID()))
	}

	h.parser.edgeHandler(from, to, pointList, way, nodeTags)
}

func (h *pass2Handler) HandleRelation(relation *reader.ReaderRelation) {
	if !h.handledRelations {
		log.Println("pass2 - start reading OSM relations")
		h.handledRelations = true
	}
	h.parser.relationProcessor(relation, h.getInternalNodeIDOfOSMNode)
}

func (h *pass2Handler) getInternalNodeIDOfOSMNode(osmNodeID int64) int {
	id := h.parser.nodeData.GetID(osmNodeID)
	if IsTowerNode(id) {
		return int(-id) - 3
	}
	return -1
}

func (h *pass2Handler) HandleFileHeader(header *OSMFileHeader) {}

func (h *pass2Handler) OnFinish() {
	log.Printf("pass2 - finished, processed ways: %d, accepted nodes: %d, ignored barriers at junctions: %d",
		h.wayCounter, h.acceptedNodes, h.ignoredSplitNodes)
}
