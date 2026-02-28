package storage

import (
	"fmt"
	"math"

	"gohopper/core/util"
)

const (
	intDistFactor = 1000.0
	noEdge        = -1
	noTurnEntry   = -1
)

// MaxDist is the maximum distance per edge (~2147 km).
var MaxDist = float64(math.MaxInt32) / intDistFactor

// BaseGraphNodesAndEdges is the low-level storage engine for nodes and edges.
// It manages two DataAccess instances with fixed memory layouts.
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

	// Node memory layout
	s.nEdgeRef = 0
	s.nLat = 4
	s.nLon = 8
	s.nEle = s.nLon
	if withElevation {
		s.nEle += 4
	}
	s.nTC = s.nEle
	if withTurnCosts {
		s.nTC += 4
	}
	s.nodeEntryBytes = s.nTC + 4

	// Edge memory layout
	s.eNodeA = 0
	s.eNodeB = 4
	s.eLinkA = 8
	s.eLinkB = 12
	s.eDist = 16
	s.eKV = 20
	s.eFlags = 24
	s.eGeo = s.eFlags + bytesForFlags
	s.edgeEntryBytes = s.eGeo + 5

	return s
}

func (s *BaseGraphNodesAndEdges) Create(initSize int64) {
	s.nodes.Create(initSize)
	s.edges.Create(initSize)
}

func (s *BaseGraphNodesAndEdges) LoadExisting() bool {
	if !s.nodes.LoadExisting() || !s.edges.LoadExisting() {
		return false
	}

	nodesVersion := s.nodes.GetHeader(0 * 4)
	checkDAVersion("nodes", util.VersionNode, int(nodesVersion))
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

	edgesVersion := s.edges.GetHeader(0 * 4)
	checkDAVersion("edges", util.VersionEdge, int(edgesVersion))
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
	if s.frozen {
		s.nodes.SetHeader(10*4, 1)
	} else {
		s.nodes.SetHeader(10*4, 0)
	}

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

func (s *BaseGraphNodesAndEdges) GetNodes() int { return s.nodeCount }
func (s *BaseGraphNodesAndEdges) GetEdges() int { return s.edgeCount }

func (s *BaseGraphNodesAndEdges) WithElevation() bool { return s.withElevation }
func (s *BaseGraphNodesAndEdges) WithTurnCosts() bool { return s.withTurnCosts }
func (s *BaseGraphNodesAndEdges) GetBounds() util.BBox { return s.Bounds }
func (s *BaseGraphNodesAndEdges) IsFrozen() bool       { return s.frozen }
func (s *BaseGraphNodesAndEdges) SetFrozen(f bool)     { s.frozen = f }

func (s *BaseGraphNodesAndEdges) IsClosed() bool {
	return s.nodes.IsClosed()
}

func (s *BaseGraphNodesAndEdges) GetBytesForFlags() int { return s.bytesForFlags }

// EnsureNodeCapacity grows node storage if needed, initializing new nodes.
func (s *BaseGraphNodesAndEdges) EnsureNodeCapacity(node int) {
	if node < s.nodeCount {
		return
	}
	oldNodes := s.nodeCount
	s.nodeCount = node + 1
	s.nodes.EnsureCapacity(int64(s.nodeCount) * int64(s.nodeEntryBytes))
	for n := oldNodes; n < s.nodeCount; n++ {
		s.SetEdgeRef(s.ToNodePointer(n), noEdge)
		if s.withTurnCosts {
			s.SetTurnCostRef(s.ToNodePointer(n), noTurnEntry)
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
	edge := s.edgeCount
	edgePointer := int64(s.edgeCount) * int64(s.edgeEntryBytes)
	s.edgeCount++
	s.edges.EnsureCapacity(int64(s.edgeCount) * int64(s.edgeEntryBytes))

	s.SetNodeA(edgePointer, nodeA)
	s.SetNodeB(edgePointer, nodeB)

	// Prepend edge to node A's linked list
	nodePointerA := s.ToNodePointer(nodeA)
	edgeRefA := s.GetEdgeRef(nodePointerA)
	if edgeRefA >= 0 {
		s.SetLinkA(edgePointer, edgeRefA)
	} else {
		s.SetLinkA(edgePointer, noEdge)
	}
	s.SetEdgeRef(nodePointerA, edge)

	// Prepend edge to node B's linked list
	if nodeA != nodeB {
		nodePointerB := s.ToNodePointer(nodeB)
		edgeRefB := s.GetEdgeRef(nodePointerB)
		if edgeRefB >= 0 {
			s.SetLinkB(edgePointer, edgeRefB)
		} else {
			s.SetLinkB(edgePointer, noEdge)
		}
		s.SetEdgeRef(nodePointerB, edge)
	}
	return edge
}

// Node field accessors

func (s *BaseGraphNodesAndEdges) SetEdgeRef(nodePointer int64, edgeRef int) {
	s.nodes.SetInt(nodePointer+int64(s.nEdgeRef), int32(edgeRef))
}

func (s *BaseGraphNodesAndEdges) GetEdgeRef(nodePointer int64) int {
	return int(s.nodes.GetInt(nodePointer + int64(s.nEdgeRef)))
}

func (s *BaseGraphNodesAndEdges) SetLat(nodePointer int64, lat float64) {
	s.nodes.SetInt(nodePointer+int64(s.nLat), util.DegreeToInt(lat))
}

func (s *BaseGraphNodesAndEdges) GetLat(nodePointer int64) float64 {
	return util.IntToDegree(s.nodes.GetInt(nodePointer + int64(s.nLat)))
}

func (s *BaseGraphNodesAndEdges) SetLon(nodePointer int64, lon float64) {
	s.nodes.SetInt(nodePointer+int64(s.nLon), util.DegreeToInt(lon))
}

func (s *BaseGraphNodesAndEdges) GetLon(nodePointer int64) float64 {
	return util.IntToDegree(s.nodes.GetInt(nodePointer + int64(s.nLon)))
}

func (s *BaseGraphNodesAndEdges) SetEle(nodePointer int64, ele float64) {
	s.nodes.SetInt(nodePointer+int64(s.nEle), int32(util.EleToUInt(ele)))
}

func (s *BaseGraphNodesAndEdges) GetEle(nodePointer int64) float64 {
	return util.UIntToEle(int(s.nodes.GetInt(nodePointer + int64(s.nEle))))
}

func (s *BaseGraphNodesAndEdges) SetTurnCostRef(nodePointer int64, tcRef int) {
	s.nodes.SetInt(nodePointer+int64(s.nTC), int32(tcRef))
}

func (s *BaseGraphNodesAndEdges) GetTurnCostRef(nodePointer int64) int {
	return int(s.nodes.GetInt(nodePointer + int64(s.nTC)))
}

// Edge field accessors

func (s *BaseGraphNodesAndEdges) SetNodeA(edgePointer int64, nodeA int) {
	s.edges.SetInt(edgePointer+int64(s.eNodeA), int32(nodeA))
}

func (s *BaseGraphNodesAndEdges) GetNodeA(edgePointer int64) int {
	return int(s.edges.GetInt(edgePointer + int64(s.eNodeA)))
}

func (s *BaseGraphNodesAndEdges) SetNodeB(edgePointer int64, nodeB int) {
	s.edges.SetInt(edgePointer+int64(s.eNodeB), int32(nodeB))
}

func (s *BaseGraphNodesAndEdges) GetNodeB(edgePointer int64) int {
	return int(s.edges.GetInt(edgePointer + int64(s.eNodeB)))
}

func (s *BaseGraphNodesAndEdges) SetLinkA(edgePointer int64, linkA int) {
	s.edges.SetInt(edgePointer+int64(s.eLinkA), int32(linkA))
}

func (s *BaseGraphNodesAndEdges) GetLinkA(edgePointer int64) int {
	return int(s.edges.GetInt(edgePointer + int64(s.eLinkA)))
}

func (s *BaseGraphNodesAndEdges) SetLinkB(edgePointer int64, linkB int) {
	s.edges.SetInt(edgePointer+int64(s.eLinkB), int32(linkB))
}

func (s *BaseGraphNodesAndEdges) GetLinkB(edgePointer int64) int {
	return int(s.edges.GetInt(edgePointer + int64(s.eLinkB)))
}

func (s *BaseGraphNodesAndEdges) SetDist(edgePointer int64, distance float64) {
	s.edges.SetInt(edgePointer+int64(s.eDist), int32(distToInt(distance)))
}

func (s *BaseGraphNodesAndEdges) GetDist(edgePointer int64) float64 {
	return float64(s.edges.GetInt(edgePointer+int64(s.eDist))) / intDistFactor
}

func (s *BaseGraphNodesAndEdges) SetGeoRef(edgePointer int64, geoRef int64) {
	s.edges.SetInt(edgePointer+int64(s.eGeo), int32(geoRef))
	s.edges.SetByte(edgePointer+int64(s.eGeo)+4, byte(geoRef>>32))
}

func (s *BaseGraphNodesAndEdges) GetGeoRef(edgePointer int64) int64 {
	low := s.edges.GetInt(edgePointer + int64(s.eGeo))
	high := s.edges.GetByte(edgePointer + int64(s.eGeo) + 4)
	// Sign-extend the byte to support negative georefs (#2985)
	return int64(low)&0xFFFF_FFFF | int64(int8(high))<<32
}

func (s *BaseGraphNodesAndEdges) SetKeyValuesRef(edgePointer int64, ref int) {
	s.edges.SetInt(edgePointer+int64(s.eKV), int32(ref))
}

func (s *BaseGraphNodesAndEdges) GetKeyValuesRef(edgePointer int64) int {
	return int(s.edges.GetInt(edgePointer + int64(s.eKV)))
}

// Flag accessors

func (s *BaseGraphNodesAndEdges) ReadFlags(edgePointer int64, flags *IntsRef) {
	for i := 0; i < flags.Length; i++ {
		flags.Ints[i] = s.getFlagInt(edgePointer, i*4)
	}
}

func (s *BaseGraphNodesAndEdges) WriteFlags(edgePointer int64, flags *IntsRef) {
	for i := 0; i < flags.Length; i++ {
		s.setFlagInt(edgePointer, i*4, flags.Ints[i])
	}
}

func (s *BaseGraphNodesAndEdges) getFlagInt(edgePointer int64, byteOffset int) int32 {
	if byteOffset >= s.bytesForFlags {
		panic(fmt.Sprintf("too large byteOffset %d vs %d", byteOffset, s.bytesForFlags))
	}
	ep := edgePointer + int64(byteOffset)
	if byteOffset+3 == s.bytesForFlags {
		return (int32(s.edges.GetShort(ep+int64(s.eFlags))) << 8) & 0x00FF_FFFF | int32(s.edges.GetByte(ep+int64(s.eFlags)+2))&0xFF
	} else if byteOffset+2 == s.bytesForFlags {
		return int32(s.edges.GetShort(ep+int64(s.eFlags))) & 0xFFFF
	} else if byteOffset+1 == s.bytesForFlags {
		return int32(s.edges.GetByte(ep+int64(s.eFlags))) & 0xFF
	}
	return s.edges.GetInt(ep + int64(s.eFlags))
}

func (s *BaseGraphNodesAndEdges) setFlagInt(edgePointer int64, byteOffset int, value int32) {
	if byteOffset >= s.bytesForFlags {
		panic(fmt.Sprintf("too large byteOffset %d vs %d", byteOffset, s.bytesForFlags))
	}
	ep := edgePointer + int64(byteOffset)
	if byteOffset+3 == s.bytesForFlags {
		s.edges.SetShort(ep+int64(s.eFlags), int16(value>>8))
		s.edges.SetByte(ep+int64(s.eFlags)+2, byte(value))
	} else if byteOffset+2 == s.bytesForFlags {
		s.edges.SetShort(ep+int64(s.eFlags), int16(value))
	} else if byteOffset+1 == s.bytesForFlags {
		s.edges.SetByte(ep+int64(s.eFlags), byte(value))
	} else {
		s.edges.SetInt(ep+int64(s.eFlags), value)
	}
}

func (s *BaseGraphNodesAndEdges) CreateEdgeFlags() *IntsRef {
	n := (s.bytesForFlags + 3) / 4
	return NewIntsRef(n)
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
