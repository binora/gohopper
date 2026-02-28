package storage

import "gohopper/core/util"

// BaseGraph is the top-level graph structure managing nodes, edges, way geometry,
// key-value storage, and turn costs.
type BaseGraph struct {
	Store           *BaseGraphNodesAndEdges
	nodeAccess      NodeAccess
	EdgeKVStorage   *KVStorage
	TurnCostStorage *TurnCostStorage
	wayGeometry     DataAccess
	dir             Directory
	segmentSize     int
	initialized     bool
	minGeoRef       int64
	maxGeoRef       int64
	eleBytesPerCoord int
}

// NewBaseGraph creates a new BaseGraph.
func NewBaseGraph(dir Directory, withElevation, withTurnCosts bool, segmentSize, bytesForFlags int) *BaseGraph {
	store := NewBaseGraphNodesAndEdges(dir, withElevation, withTurnCosts, segmentSize, bytesForFlags)
	na := newGHNodeAccess(store)
	bg := &BaseGraph{
		Store:         store,
		nodeAccess:    na,
		EdgeKVStorage: NewKVStorage(dir, true),
		wayGeometry:   dir.CreateWithSegmentSize("geometry", segmentSize),
		dir:           dir,
		segmentSize:   segmentSize,
	}
	if na.Is3D() {
		bg.eleBytesPerCoord = 3
	}
	if withTurnCosts {
		bg.TurnCostStorage = NewTurnCostStorage(dir, segmentSize)
	}
	return bg
}

// Create initializes new storage.
func (bg *BaseGraph) Create(initSize int64) *BaseGraph {
	bg.checkNotInitialized()
	bg.dir.Init()
	bg.Store.Create(initSize)
	geoInit := initSize
	if geoInit > 2000 {
		geoInit = 2000
	}
	bg.wayGeometry.Create(geoInit)
	bg.EdgeKVStorage.Create(geoInit)
	if bg.TurnCostStorage != nil {
		bg.TurnCostStorage.Create(geoInit)
	}
	bg.initialized = true
	bg.minGeoRef = -1
	bg.maxGeoRef = 1
	return bg
}

// LoadExisting loads all sub-stores from persistent storage.
func (bg *BaseGraph) LoadExisting() bool {
	bg.checkNotInitialized()
	if !bg.Store.LoadExisting() {
		return false
	}
	if !bg.wayGeometry.LoadExisting() {
		return false
	}
	if !bg.EdgeKVStorage.LoadExisting() {
		return false
	}
	if bg.TurnCostStorage != nil && !bg.TurnCostStorage.LoadExisting() {
		return false
	}
	bg.initialized = true
	bg.loadWayGeometryHeader()
	return true
}

// Flush writes all sub-stores to persistent storage.
func (bg *BaseGraph) Flush() {
	if !bg.wayGeometry.IsClosed() {
		bg.setWayGeometryHeader()
		bg.wayGeometry.Flush()
	}
	if !bg.EdgeKVStorage.IsClosed() {
		bg.EdgeKVStorage.Flush()
	}
	bg.Store.Flush()
	if bg.TurnCostStorage != nil {
		bg.TurnCostStorage.Flush()
	}
}

// Close releases all resources.
func (bg *BaseGraph) Close() {
	if !bg.wayGeometry.IsClosed() {
		bg.wayGeometry.Close()
	}
	if !bg.EdgeKVStorage.IsClosed() {
		bg.EdgeKVStorage.Close()
	}
	bg.Store.Close()
	if bg.TurnCostStorage != nil && !bg.TurnCostStorage.IsClosed() {
		bg.TurnCostStorage.Close()
	}
}

func (bg *BaseGraph) GetNodes() int            { return bg.Store.GetNodes() }
func (bg *BaseGraph) GetEdges() int            { return bg.Store.GetEdges() }
func (bg *BaseGraph) GetNodeAccess() NodeAccess { return bg.nodeAccess }
func (bg *BaseGraph) GetBounds() util.BBox      { return bg.Store.GetBounds() }
func (bg *BaseGraph) IsFrozen() bool            { return bg.Store.IsFrozen() }

func (bg *BaseGraph) Freeze() {
	bg.Store.SetFrozen(true)
}

func (bg *BaseGraph) SupportsTurnCosts() bool {
	return bg.TurnCostStorage != nil
}

func (bg *BaseGraph) checkNotInitialized() {
	if bg.initialized {
		panic("graph already initialized")
	}
}

// Way geometry header: version + minGeoRef/maxGeoRef stored as two longs (4 ints)
func (bg *BaseGraph) setWayGeometryHeader() {
	bg.wayGeometry.SetHeader(0*4, int32(util.VersionGeometry))
	bg.wayGeometry.SetHeader(1*4, util.BitLE.GetIntLow(bg.minGeoRef))
	bg.wayGeometry.SetHeader(2*4, util.BitLE.GetIntHigh(bg.minGeoRef))
	bg.wayGeometry.SetHeader(3*4, util.BitLE.GetIntLow(bg.maxGeoRef))
	bg.wayGeometry.SetHeader(4*4, util.BitLE.GetIntHigh(bg.maxGeoRef))
}

func (bg *BaseGraph) loadWayGeometryHeader() {
	version := bg.wayGeometry.GetHeader(0 * 4)
	checkDAVersion("geometry", util.VersionGeometry, int(version))
	bg.minGeoRef = util.BitLE.ToLongFromInts(bg.wayGeometry.GetHeader(1*4), bg.wayGeometry.GetHeader(2*4))
	bg.maxGeoRef = util.BitLE.ToLongFromInts(bg.wayGeometry.GetHeader(3*4), bg.wayGeometry.GetHeader(4*4))
}

// Edge creates a new edge between nodeA and nodeB.
func (bg *BaseGraph) Edge(nodeA, nodeB int) int {
	if bg.IsFrozen() {
		panic("cannot create edge on frozen graph")
	}
	return bg.Store.Edge(nodeA, nodeB)
}

// SetDist sets the distance for an edge.
func (bg *BaseGraph) SetDist(edgeId int, distance float64) {
	bg.Store.SetDist(bg.Store.ToEdgePointer(edgeId), distance)
}

// GetDist returns the distance of an edge.
func (bg *BaseGraph) GetDist(edgeId int) float64 {
	return bg.Store.GetDist(bg.Store.ToEdgePointer(edgeId))
}

// Builder provides a convenient way to create BaseGraph instances.
type BaseGraphBuilder struct {
	dir            Directory
	withElevation  bool
	withTurnCosts  bool
	bytesForFlags  int
	segmentSize    int
	initBytes      int64
}

func NewBaseGraphBuilder(bytesForFlags int) *BaseGraphBuilder {
	return &BaseGraphBuilder{
		bytesForFlags: bytesForFlags,
		segmentSize:   -1,
		initBytes:     100,
	}
}

func (b *BaseGraphBuilder) SetDir(dir Directory) *BaseGraphBuilder {
	b.dir = dir
	return b
}

func (b *BaseGraphBuilder) SetWithElevation(v bool) *BaseGraphBuilder {
	b.withElevation = v
	return b
}

func (b *BaseGraphBuilder) SetWithTurnCosts(v bool) *BaseGraphBuilder {
	b.withTurnCosts = v
	return b
}

func (b *BaseGraphBuilder) SetSegmentSize(v int) *BaseGraphBuilder {
	b.segmentSize = v
	return b
}

func (b *BaseGraphBuilder) SetBytes(v int64) *BaseGraphBuilder {
	b.initBytes = v
	return b
}

func (b *BaseGraphBuilder) Build() *BaseGraph {
	if b.dir == nil {
		b.dir = NewRAMDirectory("", false)
	}
	return NewBaseGraph(b.dir, b.withElevation, b.withTurnCosts, b.segmentSize, b.bytesForFlags)
}

func (b *BaseGraphBuilder) CreateGraph() *BaseGraph {
	bg := b.Build()
	bg.Create(b.initBytes)
	return bg
}
