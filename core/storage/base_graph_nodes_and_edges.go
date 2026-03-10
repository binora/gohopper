package storage

import (
	"fmt"
	"math"

	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

var _ ev.EdgeIntAccess = (*BaseGraphNodesAndEdges)(nil)

const (
	intDistFactor = 1000.0
	noEdge        = -1
)

// MaxDist is the maximum distance per edge (~2147 km).
var MaxDist = float64(math.MaxInt32) / intDistFactor

// BaseGraphNodesAndEdges is the low-level storage engine for nodes and edges.
type BaseGraphNodesAndEdges struct {
	nodes DataAccess
	edges DataAccess

	// Node field offsets
	nEdgeRef, nLat, nLon, nEle, nTC int
	nodeEntryBytes                  int
	nodeCount                       int

	// Edge field offsets
	eNodeA, eNodeB, eLinkA, eLinkB, eDist, eKV, eFlags, eGeo int
	bytesForFlags                                              int
	edgeEntryBytes                                             int
	edgeCount                                                  int

	withTurnCosts bool
	withElevation bool
	Bounds        util.BBox
	frozen        bool
}

func NewBaseGraphNodesAndEdges(dir Directory, withElevation, withTurnCosts bool, segmentSize, bytesForFlags int) *BaseGraphNodesAndEdges {
	s := &BaseGraphNodesAndEdges{
		nodes:         dir.CreateFull("nodes", dir.DefaultTypeFor("nodes", true), segmentSize),
		edges:         dir.CreateFull("edges", dir.DefaultTypeFor("edges", false), segmentSize),
		bytesForFlags: bytesForFlags,
		withTurnCosts: withTurnCosts,
		withElevation: withElevation,
		Bounds:        util.CreateInverse(withElevation),
	}
	s.initNodeLayout()
	s.initEdgeLayout()
	return s
}

func (s *BaseGraphNodesAndEdges) initNodeLayout() {
	s.nEdgeRef = 0
	s.nLat = 4
	s.nLon = 8
	s.nEle = s.nLon
	if s.withElevation {
		s.nEle += 4
	}
	s.nTC = s.nEle
	if s.withTurnCosts {
		s.nTC += 4
	}
	s.nodeEntryBytes = s.nTC + 4
}

func (s *BaseGraphNodesAndEdges) initEdgeLayout() {
	s.eNodeA = 0
	s.eNodeB = 4
	s.eLinkA = 8
	s.eLinkB = 12
	s.eDist = 16
	s.eKV = 20
	s.eFlags = 24
	s.eGeo = s.eFlags + s.bytesForFlags
	s.edgeEntryBytes = s.eGeo + 5
}

func (s *BaseGraphNodesAndEdges) Create(initSize int64) {
	s.nodes.Create(initSize)
	s.edges.Create(initSize)
}

func (s *BaseGraphNodesAndEdges) LoadExisting() bool {
	if !s.nodes.LoadExisting() || !s.edges.LoadExisting() {
		return false
	}

	checkDAVersion("nodes", util.VersionNode, int(s.nodes.GetHeader(0*4)))
	s.nodeEntryBytes = int(s.nodes.GetHeader(1 * 4))
	s.nodeCount = int(s.nodes.GetHeader(2 * 4))
	s.Bounds.MinLon = util.IntToDegree(s.nodes.GetHeader(3 * 4))
	s.Bounds.MaxLon = util.IntToDegree(s.nodes.GetHeader(4 * 4))
	s.Bounds.MinLat = util.IntToDegree(s.nodes.GetHeader(5 * 4))
	s.Bounds.MaxLat = util.IntToDegree(s.nodes.GetHeader(6 * 4))

	hasElevation := s.nodes.GetHeader(7*4) == 1
	if hasElevation != s.withElevation {
		panic(fmt.Sprintf("configured dimension elevation=%v is not equal to dimension of loaded graph elevation=%v", s.withElevation, hasElevation))
	}
	if s.withElevation {
		s.Bounds.MinEle = util.UIntToEle(int(s.nodes.GetHeader(8 * 4)))
		s.Bounds.MaxEle = util.UIntToEle(int(s.nodes.GetHeader(9 * 4)))
	}
	s.frozen = s.nodes.GetHeader(10*4) == 1

	checkDAVersion("edges", util.VersionEdge, int(s.edges.GetHeader(0*4)))
	s.edgeEntryBytes = int(s.edges.GetHeader(1 * 4))
	s.edgeCount = int(s.edges.GetHeader(2 * 4))
	return true
}

func (s *BaseGraphNodesAndEdges) Flush() {
	s.nodes.SetHeader(0*4, int32(util.VersionNode))
	s.nodes.SetHeader(1*4, int32(s.nodeEntryBytes))
	s.nodes.SetHeader(2*4, int32(s.nodeCount))
	s.nodes.SetHeader(3*4, util.DegreeToInt(s.Bounds.MinLon))
	s.nodes.SetHeader(4*4, util.DegreeToInt(s.Bounds.MaxLon))
	s.nodes.SetHeader(5*4, util.DegreeToInt(s.Bounds.MinLat))
	s.nodes.SetHeader(6*4, util.DegreeToInt(s.Bounds.MaxLat))
	if s.withElevation {
		s.nodes.SetHeader(7*4, 1)
		s.nodes.SetHeader(8*4, int32(util.EleToUInt(s.Bounds.MinEle)))
		s.nodes.SetHeader(9*4, int32(util.EleToUInt(s.Bounds.MaxEle)))
	} else {
		s.nodes.SetHeader(7*4, 0)
	}
	s.nodes.SetHeader(10*4, boolToInt32(s.frozen))

	s.edges.SetHeader(0*4, int32(util.VersionEdge))
	s.edges.SetHeader(1*4, int32(s.edgeEntryBytes))
	s.edges.SetHeader(2*4, int32(s.edgeCount))

	s.edges.Flush()
	s.nodes.Flush()
}

func (s *BaseGraphNodesAndEdges) Close() {
	s.edges.Close()
	s.nodes.Close()
}

func (s *BaseGraphNodesAndEdges) GetNodes() int          { return s.nodeCount }
func (s *BaseGraphNodesAndEdges) GetEdges() int          { return s.edgeCount }
func (s *BaseGraphNodesAndEdges) WithElevation() bool    { return s.withElevation }
func (s *BaseGraphNodesAndEdges) WithTurnCosts() bool    { return s.withTurnCosts }
func (s *BaseGraphNodesAndEdges) GetBounds() util.BBox   { return s.Bounds }
func (s *BaseGraphNodesAndEdges) IsFrozen() bool         { return s.frozen }
func (s *BaseGraphNodesAndEdges) SetFrozen(f bool)       { s.frozen = f }
func (s *BaseGraphNodesAndEdges) IsClosed() bool         { return s.nodes.IsClosed() }
func (s *BaseGraphNodesAndEdges) GetBytesForFlags() int  { return s.bytesForFlags }

// EnsureNodeCapacity grows node storage if needed, initializing new nodes.
func (s *BaseGraphNodesAndEdges) EnsureNodeCapacity(node int) {
	if node < s.nodeCount {
		return
	}
	oldCount := s.nodeCount
	s.nodeCount = node + 1
	s.nodes.EnsureCapacity(int64(s.nodeCount) * int64(s.nodeEntryBytes))
	for n := oldCount; n < s.nodeCount; n++ {
		ptr := s.ToNodePointer(n)
		s.SetEdgeRef(ptr, noEdge)
		if s.withTurnCosts {
			s.SetTurnCostRef(ptr, NoTurnEntry)
		}
	}
}

func (s *BaseGraphNodesAndEdges) ToNodePointer(node int) int64 {
	if node < 0 || node >= s.nodeCount {
		panic(fmt.Sprintf("node: %d out of bounds [0,%d[", node, s.nodeCount))
	}
	return int64(node) * int64(s.nodeEntryBytes)
}

func (s *BaseGraphNodesAndEdges) ToEdgePointer(edge int) int64 {
	if edge < 0 || edge >= s.edgeCount {
		panic(fmt.Sprintf("edge: %d out of bounds [0,%d[", edge, s.edgeCount))
	}
	return int64(edge) * int64(s.edgeEntryBytes)
}

// Edge creates a new edge between nodeA and nodeB, returning the edge ID.
func (s *BaseGraphNodesAndEdges) Edge(nodeA, nodeB int) int {
	if s.edgeCount == math.MaxInt32 {
		panic(fmt.Sprintf("maximum edge count exceeded: %d", s.edgeCount))
	}
	if nodeA == nodeB {
		panic(fmt.Sprintf("loop edges are not supported, got: %d - %d", nodeA, nodeB))
	}
	s.EnsureNodeCapacity(max(nodeA, nodeB))

	edgeID := s.edgeCount
	edgePtr := int64(s.edgeCount) * int64(s.edgeEntryBytes)
	s.edgeCount++
	s.edges.EnsureCapacity(int64(s.edgeCount) * int64(s.edgeEntryBytes))

	s.SetNodeA(edgePtr, nodeA)
	s.SetNodeB(edgePtr, nodeB)

	// Prepend to node A's adjacency list
	ptrA := s.ToNodePointer(nodeA)
	prevA := s.GetEdgeRef(ptrA)
	if prevA >= 0 {
		s.SetLinkA(edgePtr, prevA)
	} else {
		s.SetLinkA(edgePtr, noEdge)
	}
	s.SetEdgeRef(ptrA, edgeID)

	// Prepend to node B's adjacency list
	ptrB := s.ToNodePointer(nodeB)
	prevB := s.GetEdgeRef(ptrB)
	if prevB >= 0 {
		s.SetLinkB(edgePtr, prevB)
	} else {
		s.SetLinkB(edgePtr, noEdge)
	}
	s.SetEdgeRef(ptrB, edgeID)

	return edgeID
}

// Node field accessors

func (s *BaseGraphNodesAndEdges) SetEdgeRef(nodePtr int64, edgeRef int) {
	s.nodes.SetInt(nodePtr+int64(s.nEdgeRef), int32(edgeRef))
}

func (s *BaseGraphNodesAndEdges) GetEdgeRef(nodePtr int64) int {
	return int(s.nodes.GetInt(nodePtr + int64(s.nEdgeRef)))
}

func (s *BaseGraphNodesAndEdges) SetLat(nodePtr int64, lat float64) {
	s.nodes.SetInt(nodePtr+int64(s.nLat), util.DegreeToInt(lat))
}

func (s *BaseGraphNodesAndEdges) GetLat(nodePtr int64) float64 {
	return util.IntToDegree(s.nodes.GetInt(nodePtr + int64(s.nLat)))
}

func (s *BaseGraphNodesAndEdges) SetLon(nodePtr int64, lon float64) {
	s.nodes.SetInt(nodePtr+int64(s.nLon), util.DegreeToInt(lon))
}

func (s *BaseGraphNodesAndEdges) GetLon(nodePtr int64) float64 {
	return util.IntToDegree(s.nodes.GetInt(nodePtr + int64(s.nLon)))
}

func (s *BaseGraphNodesAndEdges) SetEle(nodePtr int64, ele float64) {
	s.nodes.SetInt(nodePtr+int64(s.nEle), int32(util.EleToUInt(ele)))
}

func (s *BaseGraphNodesAndEdges) GetEle(nodePtr int64) float64 {
	return util.UIntToEle(int(s.nodes.GetInt(nodePtr + int64(s.nEle))))
}

func (s *BaseGraphNodesAndEdges) SetTurnCostRef(nodePtr int64, tcRef int) {
	s.nodes.SetInt(nodePtr+int64(s.nTC), int32(tcRef))
}

func (s *BaseGraphNodesAndEdges) GetTurnCostRef(nodePtr int64) int {
	return int(s.nodes.GetInt(nodePtr + int64(s.nTC)))
}

// Edge field accessors

func (s *BaseGraphNodesAndEdges) SetNodeA(edgePtr int64, nodeA int) {
	s.edges.SetInt(edgePtr+int64(s.eNodeA), int32(nodeA))
}

func (s *BaseGraphNodesAndEdges) GetNodeA(edgePtr int64) int {
	return int(s.edges.GetInt(edgePtr + int64(s.eNodeA)))
}

func (s *BaseGraphNodesAndEdges) SetNodeB(edgePtr int64, nodeB int) {
	s.edges.SetInt(edgePtr+int64(s.eNodeB), int32(nodeB))
}

func (s *BaseGraphNodesAndEdges) GetNodeB(edgePtr int64) int {
	return int(s.edges.GetInt(edgePtr + int64(s.eNodeB)))
}

func (s *BaseGraphNodesAndEdges) SetLinkA(edgePtr int64, linkA int) {
	s.edges.SetInt(edgePtr+int64(s.eLinkA), int32(linkA))
}

func (s *BaseGraphNodesAndEdges) GetLinkA(edgePtr int64) int {
	return int(s.edges.GetInt(edgePtr + int64(s.eLinkA)))
}

func (s *BaseGraphNodesAndEdges) SetLinkB(edgePtr int64, linkB int) {
	s.edges.SetInt(edgePtr+int64(s.eLinkB), int32(linkB))
}

func (s *BaseGraphNodesAndEdges) GetLinkB(edgePtr int64) int {
	return int(s.edges.GetInt(edgePtr + int64(s.eLinkB)))
}

func (s *BaseGraphNodesAndEdges) SetDist(edgePtr int64, distance float64) {
	s.edges.SetInt(edgePtr+int64(s.eDist), int32(distToInt(distance)))
}

func (s *BaseGraphNodesAndEdges) GetDist(edgePtr int64) float64 {
	return float64(s.edges.GetInt(edgePtr+int64(s.eDist))) / intDistFactor
}

func (s *BaseGraphNodesAndEdges) SetGeoRef(edgePtr int64, geoRef int64) {
	s.edges.SetInt(edgePtr+int64(s.eGeo), int32(geoRef))
	s.edges.SetByte(edgePtr+int64(s.eGeo)+4, byte(geoRef>>32))
}

func (s *BaseGraphNodesAndEdges) GetGeoRef(edgePtr int64) int64 {
	low := s.edges.GetInt(edgePtr + int64(s.eGeo))
	high := s.edges.GetByte(edgePtr + int64(s.eGeo) + 4)
	return int64(low)&0xFFFF_FFFF | int64(int8(high))<<32
}

func (s *BaseGraphNodesAndEdges) SetKeyValuesRef(edgePtr int64, ref int) {
	s.edges.SetInt(edgePtr+int64(s.eKV), int32(ref))
}

func (s *BaseGraphNodesAndEdges) GetKeyValuesRef(edgePtr int64) int {
	return int(s.edges.GetInt(edgePtr + int64(s.eKV)))
}

// Flag accessors

func (s *BaseGraphNodesAndEdges) ReadFlags(edgePtr int64, flags *IntsRef) {
	for i := 0; i < flags.Length; i++ {
		flags.Ints[i] = s.getFlagInt(edgePtr, i*4)
	}
}

func (s *BaseGraphNodesAndEdges) WriteFlags(edgePtr int64, flags *IntsRef) {
	for i := 0; i < flags.Length; i++ {
		s.setFlagInt(edgePtr, i*4, flags.Ints[i])
	}
}

func (s *BaseGraphNodesAndEdges) getFlagInt(edgePtr int64, byteOff int) int32 {
	if byteOff >= s.bytesForFlags {
		panic(fmt.Sprintf("too large byteOffset %d vs %d", byteOff, s.bytesForFlags))
	}
	pos := edgePtr + int64(byteOff) + int64(s.eFlags)
	remaining := s.bytesForFlags - byteOff
	switch {
	case remaining == 3:
		return (int32(s.edges.GetShort(pos)) << 8) & 0x00FF_FFFF | int32(s.edges.GetByte(pos+2))&0xFF
	case remaining == 2:
		return int32(s.edges.GetShort(pos)) & 0xFFFF
	case remaining == 1:
		return int32(s.edges.GetByte(pos)) & 0xFF
	default:
		return s.edges.GetInt(pos)
	}
}

func (s *BaseGraphNodesAndEdges) setFlagInt(edgePtr int64, byteOff int, value int32) {
	if byteOff >= s.bytesForFlags {
		panic(fmt.Sprintf("too large byteOffset %d vs %d", byteOff, s.bytesForFlags))
	}
	pos := edgePtr + int64(byteOff) + int64(s.eFlags)
	remaining := s.bytesForFlags - byteOff
	switch {
	case remaining == 3:
		s.edges.SetShort(pos, int16(value>>8))
		s.edges.SetByte(pos+2, byte(value))
	case remaining == 2:
		s.edges.SetShort(pos, int16(value))
	case remaining == 1:
		s.edges.SetByte(pos, byte(value))
	default:
		s.edges.SetInt(pos, value)
	}
}

func (s *BaseGraphNodesAndEdges) CreateEdgeFlags() *IntsRef {
	return NewIntsRef((s.bytesForFlags + 3) / 4)
}

// EdgeIntAccess implementation — bridges edgeID+index to raw byte storage.

func (s *BaseGraphNodesAndEdges) GetInt(edgeID, index int) int32 {
	return s.getFlagInt(s.ToEdgePointer(edgeID), index*4)
}

func (s *BaseGraphNodesAndEdges) SetInt(edgeID, index int, value int32) {
	s.setFlagInt(s.ToEdgePointer(edgeID), index*4, value)
}

// edgeData holds all fields of an edge for in-place permutation.
type edgeData struct {
	nodeA, nodeB int
	linkA, linkB int
	dist         int32
	kv           int
	flags        *IntsRef
	geo          int64
}

func (s *BaseGraphNodesAndEdges) loadEdge(ptr int64) edgeData {
	flags := s.CreateEdgeFlags()
	s.ReadFlags(ptr, flags)
	return edgeData{
		nodeA: s.GetNodeA(ptr),
		nodeB: s.GetNodeB(ptr),
		linkA: s.GetLinkA(ptr),
		linkB: s.GetLinkB(ptr),
		dist:  s.edges.GetInt(ptr + int64(s.eDist)),
		kv:    s.GetKeyValuesRef(ptr),
		flags: flags,
		geo:   s.GetGeoRef(ptr),
	}
}

func (s *BaseGraphNodesAndEdges) storeEdge(ptr int64, e edgeData, getNewEdge func(int) int) {
	s.SetNodeA(ptr, e.nodeA)
	s.SetNodeB(ptr, e.nodeB)
	s.SetLinkA(ptr, remapEdge(e.linkA, getNewEdge))
	s.SetLinkB(ptr, remapEdge(e.linkB, getNewEdge))
	s.edges.SetInt(ptr+int64(s.eDist), e.dist)
	s.SetKeyValuesRef(ptr, e.kv)
	s.WriteFlags(ptr, e.flags)
	s.SetGeoRef(ptr, e.geo)
}

// remapEdge translates an edge ID through the permutation, preserving noEdge sentinels.
func remapEdge(edge int, getNewEdge func(int) int) int {
	if edge == noEdge {
		return noEdge
	}
	return getNewEdge(edge)
}

// SortEdges reorders edges in-place using a cycle-following permutation algorithm.
// getNewEdge maps old edge ID to new edge ID.
func (s *BaseGraphNodesAndEdges) SortEdges(getNewEdge func(int) int) {
	visited := make([]bool, s.edgeCount)
	for edge := range s.edgeCount {
		if visited[edge] {
			continue
		}
		curr := edge
		carry := s.loadEdge(s.ToEdgePointer(curr))

		for {
			visited[curr] = true
			dest := getNewEdge(curr)
			destPtr := s.ToEdgePointer(dest)

			next := s.loadEdge(destPtr)
			s.storeEdge(destPtr, carry, getNewEdge)
			carry = next

			curr = dest
			if curr == edge {
				break
			}
		}
	}

	// Update edge references in nodes.
	for node := range s.nodeCount {
		ptr := s.ToNodePointer(node)
		edgeRef := s.GetEdgeRef(ptr)
		if edgeRef != noEdge {
			s.SetEdgeRef(ptr, getNewEdge(edgeRef))
		}
	}
}

// nodeData holds all fields of a node for in-place permutation.
type nodeData struct {
	edgeRef int
	lat     float64
	lon     float64
	ele     float64
	tcRef   int
}

func (s *BaseGraphNodesAndEdges) loadNode(ptr int64) nodeData {
	nd := nodeData{
		edgeRef: s.GetEdgeRef(ptr),
		lat:     s.GetLat(ptr),
		lon:     s.GetLon(ptr),
	}
	if s.withElevation {
		nd.ele = s.GetEle(ptr)
	}
	if s.withTurnCosts {
		nd.tcRef = s.GetTurnCostRef(ptr)
	}
	return nd
}

func (s *BaseGraphNodesAndEdges) storeNode(ptr int64, nd nodeData) {
	s.SetEdgeRef(ptr, nd.edgeRef)
	s.SetLat(ptr, nd.lat)
	s.SetLon(ptr, nd.lon)
	if s.withElevation {
		s.SetEle(ptr, nd.ele)
	}
	if s.withTurnCosts {
		s.SetTurnCostRef(ptr, nd.tcRef)
	}
}

// RelabelNodes reorders nodes in-place using a cycle-following permutation algorithm.
// getNewNode maps old node ID to new node ID.
func (s *BaseGraphNodesAndEdges) RelabelNodes(getNewNode func(int) int) {
	// Update all node references in edges.
	for edge := range s.edgeCount {
		ptr := s.ToEdgePointer(edge)
		s.SetNodeA(ptr, getNewNode(s.GetNodeA(ptr)))
		s.SetNodeB(ptr, getNewNode(s.GetNodeB(ptr)))
	}

	// Cycle-following permutation on node data.
	visited := make([]bool, s.nodeCount)
	for node := range s.nodeCount {
		if visited[node] {
			continue
		}
		curr := node
		carry := s.loadNode(s.ToNodePointer(curr))

		for {
			visited[curr] = true
			dest := getNewNode(curr)
			destPtr := s.ToNodePointer(dest)

			next := s.loadNode(destPtr)
			s.storeNode(destPtr, carry)
			carry = next

			curr = dest
			if curr == node {
				break
			}
		}
	}
}

func distToInt(distance float64) int {
	if distance < 0 {
		panic(fmt.Sprintf("distance cannot be negative: %f", distance))
	}
	if distance > MaxDist {
		distance = MaxDist
	}
	return int(math.Round(distance * intDistFactor))
}

func checkDAVersion(name string, expected, actual int) {
	if expected != actual {
		panic(fmt.Sprintf("cannot load %s - expected version %d, got %d. "+
			"Make sure you are using the correct version of GraphHopper", name, expected, actual))
	}
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
