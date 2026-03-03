package osm

import (
	"fmt"
	"math"

	"gohopper/core/coll"
	"gohopper/core/storage"
	"gohopper/core/util"
)

const (
	JunctionNode     int64 = -2
	EmptyNode        int64 = -1
	EndNode          int64 = 0
	IntermediateNode int64 = 1
	ConnectionNode   int64 = 2
)

// OSMNodeData stores OSM node data during import. Tower nodes (junctions/connections) get
// negative IDs, pillar nodes (intermediate) get positive IDs. A few reserved IDs (-2..2)
// indicate the node type before coordinates are assigned in pass 2.
type OSMNodeData struct {
	// Maps OSM node ID → internal ID (or node type constant during pass 1).
	idsByOsmNodeIds coll.LongLongMap

	pillarNodes *PillarInfo
	towerNodes  storage.NodeAccess

	// Maps OSM node ID → tag pointer (index into nodeTags slice). -1 means no tags.
	nodeTagIndices coll.LongLongMap
	nodeTags       []map[string]any

	nodesToBeSplit map[int64]bool

	nextTowerId             int
	nextPillarId            int64
	nextArtificialOSMNodeId int64
}

func NewOSMNodeData(nodeAccess storage.NodeAccess, dir storage.Directory) *OSMNodeData {
	return &OSMNodeData{
		idsByOsmNodeIds:         coll.NewGHLongLongBTree(200, 5, EmptyNode),
		towerNodes:              nodeAccess,
		pillarNodes:             NewPillarInfo(nodeAccess.Is3D(), dir),
		nodeTagIndices:          coll.NewGHLongLongBTree(200, 4, -1),
		nodeTags:                nil,
		nodesToBeSplit:          make(map[int64]bool),
		nextArtificialOSMNodeId: -math.MaxInt64,
	}
}

func (d *OSMNodeData) Is3D() bool { return d.towerNodes.Is3D() }

// GetID returns the internal id stored for the given OSM node id.
func (d *OSMNodeData) GetID(osmNodeID int64) int64 {
	return d.idsByOsmNodeIds.Get(osmNodeID)
}

// IsTowerNode returns true if the internal id represents a tower node.
func IsTowerNode(id int64) bool { return id < JunctionNode }

// IsPillarNode returns true if the internal id represents a pillar node.
func IsPillarNode(id int64) bool { return id > ConnectionNode }

// IsNodeID returns true if the id represents an actual node (tower or pillar).
func IsNodeID(id int64) bool { return id > ConnectionNode || id < JunctionNode }

// SetOrUpdateNodeType sets or updates the node type for the given OSM node.
// If the node doesn't exist yet, newNodeType is stored. Otherwise nodeTypeUpdate
// is called with the current type to compute the new type.
func (d *OSMNodeData) SetOrUpdateNodeType(osmNodeID int64, newNodeType int64, nodeTypeUpdate func(int64) int64) {
	curr := d.idsByOsmNodeIds.Get(osmNodeID)
	if curr == EmptyNode {
		d.idsByOsmNodeIds.Put(osmNodeID, newNodeType)
	} else {
		d.idsByOsmNodeIds.Put(osmNodeID, nodeTypeUpdate(curr))
	}
}

// GetNodeCount returns the total number of mapped nodes.
func (d *OSMNodeData) GetNodeCount() int64 { return d.idsByOsmNodeIds.GetSize() }

// AddCoordinatesIfMapped stores coordinates for the given OSM node ID, but only if
// a non-empty node type was previously set. Returns the previous node type.
func (d *OSMNodeData) AddCoordinatesIfMapped(osmNodeID int64, lat, lon float64, getEle func() float64) int64 {
	nodeType := d.GetID(osmNodeID)
	switch nodeType {
	case EmptyNode:
		return nodeType
	case JunctionNode, ConnectionNode:
		d.addTowerNode(osmNodeID, lat, lon, getEle())
	case IntermediateNode, EndNode:
		d.addPillarNode(osmNodeID, lat, lon, getEle())
	default:
		panic(fmt.Sprintf("Unknown node type: %d, or coordinates already set. Possibly duplicate OSM node ID: %d", nodeType, osmNodeID))
	}
	return nodeType
}

func (d *OSMNodeData) addTowerNode(osmID int64, lat, lon, ele float64) int64 {
	d.towerNodes.SetNode(d.nextTowerId, lat, lon, ele)
	id := d.TowerNodeToID(d.nextTowerId)
	d.idsByOsmNodeIds.Put(osmID, id)
	d.nextTowerId++
	if d.nextTowerId == math.MaxInt32 {
		panic("Tower node id overflow, too many tower nodes")
	}
	return id
}

func (d *OSMNodeData) addPillarNode(osmID int64, lat, lon, ele float64) int64 {
	id := d.PillarNodeToID(d.nextPillarId)
	if id > d.idsByOsmNodeIds.GetMaxValue() {
		panic(fmt.Sprintf("id for pillar node cannot be bigger than %d", d.idsByOsmNodeIds.GetMaxValue()))
	}
	d.pillarNodes.SetNode(d.nextPillarId, lat, lon, ele)
	d.idsByOsmNodeIds.Put(osmID, id)
	d.nextPillarId++
	return id
}

// AddCopyOfNode creates a copy of a node's coordinates under a new artificial OSM ID.
func (d *OSMNodeData) AddCopyOfNode(node SegmentNode) SegmentNode {
	pt := d.GetCoordinates(node.ID)
	if pt == nil {
		panic(fmt.Sprintf("Cannot copy node: %d, because it is missing", node.OSMNodeID))
	}
	newOsmID := d.nextArtificialOSMNodeId
	d.nextArtificialOSMNodeId++
	if d.idsByOsmNodeIds.Put(newOsmID, IntermediateNode) != EmptyNode {
		panic(fmt.Sprintf("Artificial osm node id already exists: %d", newOsmID))
	}
	id := d.addPillarNode(newOsmID, pt.Lat, pt.Lon, pt.Ele)
	return SegmentNode{OSMNodeID: newOsmID, ID: id, Tags: node.Tags}
}

// ConvertPillarToTowerNode promotes a pillar node to a tower node.
func (d *OSMNodeData) ConvertPillarToTowerNode(id int64, osmNodeID int64) int64 {
	if !IsPillarNode(id) {
		panic(fmt.Sprintf("Not a pillar node: %d", id))
	}
	pillar := d.IDToPillarNode(id)
	lat := d.pillarNodes.GetLat(pillar)
	lon := d.pillarNodes.GetLon(pillar)
	ele := d.pillarNodes.GetEle(pillar)
	if lat == math.MaxFloat64 || lon == math.MaxFloat64 {
		panic(fmt.Sprintf("Pillar node was already converted to tower node: %d", id))
	}
	// Mark the pillar as converted.
	d.pillarNodes.SetNode(pillar, math.MaxFloat64, math.MaxFloat64, math.MaxFloat64)
	return d.addTowerNode(osmNodeID, lat, lon, ele)
}

// GHPoint3D holds coordinates for a node.
type GHPoint3D struct {
	Lat, Lon, Ele float64
}

// GetCoordinates returns the coordinates for the given internal node id, or nil if invalid.
func (d *OSMNodeData) GetCoordinates(id int64) *GHPoint3D {
	if IsTowerNode(id) {
		tower := d.IDToTowerNode(id)
		ele := math.NaN()
		if d.towerNodes.Is3D() {
			ele = d.towerNodes.GetEle(tower)
		}
		return &GHPoint3D{d.towerNodes.GetLat(tower), d.towerNodes.GetLon(tower), ele}
	}
	if IsPillarNode(id) {
		pillar := d.IDToPillarNode(id)
		ele := math.NaN()
		if d.pillarNodes.Is3D() {
			ele = d.pillarNodes.GetEle(pillar)
		}
		return &GHPoint3D{d.pillarNodes.GetLat(pillar), d.pillarNodes.GetLon(pillar), ele}
	}
	return nil
}

func (d *OSMNodeData) AddCoordinatesToPointList(id int64, pointList *util.PointList) {
	pt := d.GetCoordinates(id)
	if pt == nil {
		panic("invalid node id")
	}
	pointList.Add3D(pt.Lat, pt.Lon, pt.Ele)
}

// SetTags stores tags for the given node. Can only be called once per node.
func (d *OSMNodeData) SetTags(osmNodeID int64, tags map[string]any) {
	tagIndex := d.nodeTagIndices.Get(osmNodeID)
	if tagIndex != -1 {
		panic(fmt.Sprintf("Cannot add tags twice, duplicate node OSM ID: %d", osmNodeID))
	}
	idx := int64(len(d.nodeTags))
	d.nodeTags = append(d.nodeTags, tags)
	d.nodeTagIndices.Put(osmNodeID, idx)
}

// GetTags returns the tags for the given OSM node, or nil if none stored.
func (d *OSMNodeData) GetTags(osmNodeID int64) map[string]any {
	tagIndex := d.nodeTagIndices.Get(osmNodeID)
	if tagIndex < 0 {
		return nil
	}
	return d.nodeTags[tagIndex]
}

// Release frees temporary storage.
func (d *OSMNodeData) Release() {
	d.idsByOsmNodeIds.Clear()
	d.pillarNodes.Clear()
	d.nodeTagIndices.Clear()
	d.nodeTags = nil
	clear(d.nodesToBeSplit)
}

// TowerNodeToID converts a tower node index to its internal ID (negative).
func (d *OSMNodeData) TowerNodeToID(towerId int) int64 { return -int64(towerId) - 3 }

// IDToTowerNode converts an internal ID to a tower node index.
func (d *OSMNodeData) IDToTowerNode(id int64) int { return int(-id - 3) }

// PillarNodeToID converts a pillar node index to its internal ID (positive).
func (d *OSMNodeData) PillarNodeToID(pillarId int64) int64 { return pillarId + 3 }

// IDToPillarNode converts an internal ID to a pillar node index.
func (d *OSMNodeData) IDToPillarNode(id int64) int64 { return id - 3 }

// SetSplitNode marks a node as needing to be split (barrier).
func (d *OSMNodeData) SetSplitNode(osmNodeID int64) bool {
	if d.nodesToBeSplit[osmNodeID] {
		return false
	}
	d.nodesToBeSplit[osmNodeID] = true
	return true
}

// UnsetSplitNode removes the split-node mark.
func (d *OSMNodeData) UnsetSplitNode(osmNodeID int64) {
	if !d.nodesToBeSplit[osmNodeID] {
		panic(fmt.Sprintf("Node %d was not a split node", osmNodeID))
	}
	delete(d.nodesToBeSplit, osmNodeID)
}

// IsSplitNode returns true if the node is marked for splitting.
func (d *OSMNodeData) IsSplitNode(osmNodeID int64) bool {
	return d.nodesToBeSplit[osmNodeID]
}
