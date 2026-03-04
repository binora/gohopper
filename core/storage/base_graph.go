package storage

import (
	"fmt"

	routingutil "gohopper/core/routing/util"
	"gohopper/core/util"
)

// BaseGraph is the top-level graph structure managing nodes, edges, way geometry,
// key-value storage, and turn costs.
type BaseGraph struct {
	Store            *BaseGraphNodesAndEdges
	nodeAccess       NodeAccess
	EdgeKVStorage    *KVStorage
	TurnCostStorage  *TurnCostStorage
	wayGeometry      DataAccess
	dir              Directory
	segmentSize      int
	initialized      bool
	minGeoRef        int64
	maxGeoRef        int64
	eleBytesPerCoord int
}

func NewBaseGraph(dir Directory, withElevation, withTurnCosts bool, segmentSize, bytesForFlags int) *BaseGraph {
	store := NewBaseGraphNodesAndEdges(dir, withElevation, withTurnCosts, segmentSize, bytesForFlags)
	na := newGHNodeAccess(store)
	bg := &BaseGraph{
		Store:       store,
		nodeAccess:  na,
		EdgeKVStorage: NewKVStorage(dir, true),
		wayGeometry: dir.CreateWithSegmentSize("geometry", segmentSize),
		dir:         dir,
		segmentSize: segmentSize,
	}
	if na.Is3D() {
		bg.eleBytesPerCoord = 3
	}
	if withTurnCosts {
		bg.TurnCostStorage = NewTurnCostStorage(dir, segmentSize)
	}
	return bg
}

func (bg *BaseGraph) Create(initSize int64) *BaseGraph {
	bg.checkNotInitialized()
	bg.dir.Init()
	bg.Store.Create(initSize)
	geoInit := min(initSize, 2000)
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

func (bg *BaseGraph) GetNodes() int             { return bg.Store.GetNodes() }
func (bg *BaseGraph) GetEdges() int             { return bg.Store.GetEdges() }
func (bg *BaseGraph) GetNodeAccess() NodeAccess  { return bg.nodeAccess }
func (bg *BaseGraph) GetBounds() util.BBox       { return bg.Store.GetBounds() }
func (bg *BaseGraph) IsFrozen() bool             { return bg.Store.IsFrozen() }
func (bg *BaseGraph) SupportsTurnCosts() bool    { return bg.TurnCostStorage != nil }

func (bg *BaseGraph) Freeze() {
	bg.Store.SetFrozen(true)
}

func (bg *BaseGraph) GetTurnCostStorage() *TurnCostStorage { return bg.TurnCostStorage }
func (bg *BaseGraph) GetMaxGeoRef() int64                  { return bg.maxGeoRef }
func (bg *BaseGraph) GetDirectory() Directory              { return bg.dir }

func (bg *BaseGraph) checkNotInitialized() {
	if bg.initialized {
		panic("graph already initialized")
	}
}

func (bg *BaseGraph) setWayGeometryHeader() {
	bg.wayGeometry.SetHeader(0*4, int32(util.VersionGeometry))
	bg.wayGeometry.SetHeader(1*4, util.BitLE.GetIntLow(bg.minGeoRef))
	bg.wayGeometry.SetHeader(2*4, util.BitLE.GetIntHigh(bg.minGeoRef))
	bg.wayGeometry.SetHeader(3*4, util.BitLE.GetIntLow(bg.maxGeoRef))
	bg.wayGeometry.SetHeader(4*4, util.BitLE.GetIntHigh(bg.maxGeoRef))
}

func (bg *BaseGraph) loadWayGeometryHeader() {
	checkDAVersion("geometry", util.VersionGeometry, int(bg.wayGeometry.GetHeader(0*4)))
	bg.minGeoRef = util.BitLE.ToLongFromInts(bg.wayGeometry.GetHeader(1*4), bg.wayGeometry.GetHeader(2*4))
	bg.maxGeoRef = util.BitLE.ToLongFromInts(bg.wayGeometry.GetHeader(3*4), bg.wayGeometry.GetHeader(4*4))
}

// Edge creates a new edge between nodeA and nodeB, returning an EdgeIteratorState.
func (bg *BaseGraph) Edge(nodeA, nodeB int) util.EdgeIteratorState {
	if bg.IsFrozen() {
		panic("cannot create edge on frozen graph")
	}
	edgeID := bg.Store.Edge(nodeA, nodeB)
	edge := NewEdgeIteratorStateImpl(bg)
	edge.Init(edgeID, nodeB)
	return edge
}

func (bg *BaseGraph) GetBaseGraph() *BaseGraph { return bg }

// GetEdgeIteratorState returns an EdgeIteratorState for the given edge.
// adjNode can be math.MinInt32 to accept any direction.
// Returns nil if the edge doesn't connect to adjNode.
func (bg *BaseGraph) GetEdgeIteratorState(edgeID, adjNode int) util.EdgeIteratorState {
	edge := NewEdgeIteratorStateImpl(bg)
	if edge.Init(edgeID, adjNode) {
		return edge
	}
	return nil
}

// GetEdgeIteratorStateForKey returns an EdgeIteratorState initialized from an edge key.
func (bg *BaseGraph) GetEdgeIteratorStateForKey(edgeKey int) util.EdgeIteratorState {
	edge := NewEdgeIteratorStateImpl(bg)
	edge.InitEdgeKey(edgeKey)
	return edge
}

// CreateEdgeExplorer returns a new EdgeExplorer with the given filter.
func (bg *BaseGraph) CreateEdgeExplorer(filter routingutil.EdgeFilter) util.EdgeExplorer {
	return newEdgeIteratorImpl(bg, filter)
}

// GetAllEdges returns an iterator over all edges in the graph.
func (bg *BaseGraph) GetAllEdges() AllEdgesIterator {
	return newAllEdgeIterator(bg)
}

// GetOtherNode returns the node on the other end of the given edge.
func (bg *BaseGraph) GetOtherNode(edge, node int) int {
	edgePointer := bg.Store.ToEdgePointer(edge)
	nodeA := bg.Store.GetNodeA(edgePointer)
	if node == nodeA {
		return bg.Store.GetNodeB(edgePointer)
	}
	return nodeA
}

// IsAdjacentToNode returns true if the edge connects to the given node.
func (bg *BaseGraph) IsAdjacentToNode(edge, node int) bool {
	edgePointer := bg.Store.ToEdgePointer(edge)
	return bg.Store.GetNodeA(edgePointer) == node || bg.Store.GetNodeB(edgePointer) == node
}

func (bg *BaseGraph) SetDist(edgeID int, distance float64) {
	bg.Store.SetDist(bg.Store.ToEdgePointer(edgeID), distance)
}

func (bg *BaseGraph) GetDist(edgeID int) float64 {
	return bg.Store.GetDist(bg.Store.ToEdgePointer(edgeID))
}

func (bg *BaseGraph) setWayGeometry(pillarNodes *util.PointList, edgePointer int64, reverse bool) {
	if pillarNodes == nil || pillarNodes.IsEmpty() {
		bg.Store.SetGeoRef(edgePointer, 0)
		return
	}
	if pillarNodes.Is3D() != bg.nodeAccess.Is3D() {
		panic(fmt.Sprintf("cannot use pointlist which is 3D=%v for graph which is 3D=%v",
			pillarNodes.Is3D(), bg.nodeAccess.Is3D()))
	}
	existingGeoRef := bg.Store.GetGeoRef(edgePointer)
	if existingGeoRef < 0 {
		panic("this edge has already been copied")
	}
	count := pillarNodes.Size()
	if existingGeoRef > 0 {
		if count > bg.getPillarCount(existingGeoRef) {
			panic("this edge already has a way geometry so it cannot be changed to a bigger geometry")
		}
		bg.setWayGeometryAtGeoRef(pillarNodes, edgePointer, reverse, existingGeoRef)
		return
	}
	geoRef := bg.nextGeoRef(3 + count*(8+bg.eleBytesPerCoord))
	bg.setWayGeometryAtGeoRef(pillarNodes, edgePointer, reverse, geoRef)
}

func (bg *BaseGraph) setWayGeometryAtGeoRef(pillarNodes *util.PointList, edgePointer int64, reverse bool, geoRef int64) {
	bytes := bg.createWayGeometryBytes(pillarNodes, reverse)
	bg.wayGeometry.EnsureCapacity(geoRef + int64(len(bytes)))
	bg.wayGeometry.SetBytes(geoRef, bytes, len(bytes))
	bg.Store.SetGeoRef(edgePointer, geoRef)
}

func (bg *BaseGraph) createWayGeometryBytes(pillarNodes *util.PointList, reverse bool) []byte {
	count := pillarNodes.Size()
	coordBytes := 8 + bg.eleBytesPerCoord
	buf := make([]byte, 3+count*coordBytes)
	buf[0] = byte(count)
	buf[1] = byte(count >> 8)
	buf[2] = byte(count >> 16)

	is3D := bg.nodeAccess.Is3D()
	for i := range count {
		src := i
		if reverse {
			src = count - 1 - i
		}
		off := 3 + i*coordBytes
		util.BitLE.FromInt(buf, util.DegreeToInt(pillarNodes.GetLat(src)), off)
		util.BitLE.FromInt(buf, util.DegreeToInt(pillarNodes.GetLon(src)), off+4)
		if is3D {
			util.BitLE.FromUInt3(buf, int32(util.EleToUInt(pillarNodes.GetEle(src))), off+8)
		}
	}
	return buf
}

func (bg *BaseGraph) fetchWayGeometry(edgePointer int64, reverse bool, mode util.FetchMode, baseNode, adjNode int) *util.PointList {
	is3D := bg.nodeAccess.Is3D()
	if mode == util.FetchModeTowerOnly {
		pl := util.NewPointList(2, is3D)
		bg.addNodeToPointList(pl, baseNode)
		bg.addNodeToPointList(pl, adjNode)
		return pl
	}

	geoRef := bg.Store.GetGeoRef(edgePointer)
	count := 0
	coordBytes := 8 + bg.eleBytesPerCoord
	var buf []byte
	if geoRef > 0 {
		count = bg.getPillarCount(geoRef)
		buf = make([]byte, count*coordBytes)
		bg.wayGeometry.GetBytes(geoRef+3, buf, len(buf))
	} else if mode == util.FetchModePillarOnly {
		return util.NewPointList(0, is3D)
	}

	pl := util.NewPointList(getPointListLength(count, mode), is3D)
	if reverse {
		if mode == util.FetchModeAll || mode == util.FetchModePillarAndAdj {
			bg.addNodeToPointList(pl, adjNode)
		}
	} else if mode == util.FetchModeAll || mode == util.FetchModeBaseAndPillar {
		bg.addNodeToPointList(pl, baseNode)
	}

	off := 0
	for range count {
		lat := util.IntToDegree(util.BitLE.ToInt(buf, off))
		lon := util.IntToDegree(util.BitLE.ToInt(buf, off+4))
		if is3D {
			ele := util.UIntToEle(int(util.BitLE.ToUInt3(buf, off+8)))
			pl.Add3D(lat, lon, ele)
		} else {
			pl.Add(lat, lon)
		}
		off += coordBytes
	}

	if reverse {
		if mode == util.FetchModeAll || mode == util.FetchModeBaseAndPillar {
			bg.addNodeToPointList(pl, baseNode)
		}
		pl.Reverse()
	} else if mode == util.FetchModeAll || mode == util.FetchModePillarAndAdj {
		bg.addNodeToPointList(pl, adjNode)
	}

	return pl
}

func (bg *BaseGraph) addNodeToPointList(pl *util.PointList, nodeID int) {
	na := bg.nodeAccess
	lat, lon := na.GetLat(nodeID), na.GetLon(nodeID)
	if na.Is3D() {
		pl.Add3D(lat, lon, na.GetEle(nodeID))
	} else {
		pl.Add(lat, lon)
	}
}

func (bg *BaseGraph) getPillarCount(geoRef int64) int {
	return int(bg.wayGeometry.GetShort(geoRef)) | int(bg.wayGeometry.GetByte(geoRef+2))<<16
}

func (bg *BaseGraph) nextGeoRef(bytes int) int64 {
	tmp := bg.maxGeoRef
	bg.maxGeoRef += int64(bytes)
	bg.wayGeometry.EnsureCapacity(bg.maxGeoRef)
	return tmp
}

func getPointListLength(pillarNodes int, mode util.FetchMode) int {
	switch mode {
	case util.FetchModeTowerOnly:
		return 2
	case util.FetchModePillarOnly:
		return pillarNodes
	case util.FetchModeBaseAndPillar, util.FetchModePillarAndAdj:
		return pillarNodes + 1
	case util.FetchModeAll:
		return pillarNodes + 2
	}
	panic(fmt.Sprintf("unhandled FetchMode: %v", mode))
}

// copyProperties copies all properties from one edge state to another.
func (bg *BaseGraph) copyProperties(from util.EdgeIteratorState, to *EdgeIteratorStateImpl) util.EdgeIteratorState {
	if src, ok := from.(*EdgeIteratorStateImpl); ok {
		edgePointer := bg.Store.ToEdgePointer(to.GetEdge())
		bg.Store.WriteFlags(edgePointer, src.GetFlags())
	}
	to.SetDistance(from.GetDistance())
	to.SetKeyValues(from.GetKeyValues())
	to.SetWayGeometry(from.FetchWayGeometry(util.FetchModePillarOnly))
	return to
}

func (bg *BaseGraph) ForEdgeAndCopiesOfEdgeState(explorer util.EdgeExplorer, edge util.EdgeIteratorState, consumer func(util.EdgeIteratorState)) {
	geoRef := bg.Store.GetGeoRef(edge.(*EdgeIteratorStateImpl).edgePointer)
	if geoRef == 0 {
		consumer(edge)
		return
	}
	iter := explorer.SetBaseNode(edge.GetBaseNode())
	for iter.Next() {
		iterImpl := iter.(*edgeIteratorImpl)
		geoRefBefore := bg.Store.GetGeoRef(iterImpl.edgePointer)
		if geoRefBefore == geoRef {
			consumer(iter)
		}
		if bg.Store.GetGeoRef(iterImpl.edgePointer) != geoRefBefore {
			panic("the consumer must not change the geo ref")
		}
	}
}

func (bg *BaseGraph) ForEdgeAndCopiesOfEdge(explorer util.EdgeExplorer, node, edge int, consumer func(int)) {
	geoRef := bg.Store.GetGeoRef(bg.Store.ToEdgePointer(edge))
	if geoRef == 0 {
		consumer(edge)
		return
	}
	iter := explorer.SetBaseNode(node)
	for iter.Next() {
		iterImpl := iter.(*edgeIteratorImpl)
		geoRefBefore := bg.Store.GetGeoRef(iterImpl.edgePointer)
		if geoRefBefore == geoRef {
			consumer(iter.GetEdge())
		}
	}
}

// BaseGraphBuilder provides a convenient way to create BaseGraph instances.
type BaseGraphBuilder struct {
	dir           Directory
	withElevation bool
	withTurnCosts bool
	bytesForFlags int
	segmentSize   int
	initBytes     int64
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
