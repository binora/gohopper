package index

import (
	"log"

	routeutil "gohopper/core/routing/util"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// EdgeCheck is the callback signature used by TraverseEdge.
type EdgeCheck func(node int, normedDist float64, wayIndex int, pos Position)

// LocationIndexTree implements LocationIndex using a quadtree built on top of
// a LineIntIndex. It maps GPS coordinates to the closest node or edge in the
// graph.
type LocationIndexTree struct {
	directory  storage.Directory
	graph      storage.Graph
	nodeAccess storage.NodeAccess

	maxRegionSearch      int
	minResolutionInMeter int
	initialized          bool

	lineIntIndex       *LineIntIndex
	equalNormedDelta   float64
	indexStructureInfo  *IndexStructureInfo
}

// NewLocationIndexTree creates a new LocationIndexTree for the given graph and
// directory.
func NewLocationIndexTree(g storage.Graph, dir storage.Directory) *LocationIndexTree {
	bounds := g.GetBounds()
	if !bounds.IsValid() {
		bounds = util.NewBBox(-10.0, 10.0, -10.0, 10.0)
	}
	return &LocationIndexTree{
		directory:            dir,
		graph:                g,
		nodeAccess:           g.GetNodeAccess(),
		maxRegionSearch:      4,
		minResolutionInMeter: 300,
		lineIntIndex:         NewLineIntIndex(bounds, dir, "location_index"),
		equalNormedDelta:     util.DistPlane.CalcNormalizedDist(0.1),
	}
}

// SetMinResolutionInMeter sets the minimum tile width in meters. Decrease for
// faster queries, at the cost of potentially missing edges in neighboring
// tiles.
func (lit *LocationIndexTree) SetMinResolutionInMeter(m int) *LocationIndexTree {
	lit.minResolutionInMeter = m
	return lit
}

// GetMinResolutionInMeter returns the minimum resolution.
func (lit *LocationIndexTree) GetMinResolutionInMeter() int {
	return lit.minResolutionInMeter
}

// SetMaxRegionSearch sets the number of neighboring tile rings to search.
// Default is 4.
func (lit *LocationIndexTree) SetMaxRegionSearch(numTiles int) *LocationIndexTree {
	if numTiles < 1 {
		panic("Region of location index must be at least 1")
	}
	lit.maxRegionSearch = numTiles
	return lit
}

// SetResolution sets the minimum resolution in meters.
func (lit *LocationIndexTree) SetResolution(minResolutionInMeter int) *LocationIndexTree {
	if minResolutionInMeter <= 0 {
		panic("Negative precision is not allowed!")
	}
	lit.SetMinResolutionInMeter(minResolutionInMeter)
	return lit
}

// LoadExisting loads a previously stored index. Returns true on success.
func (lit *LocationIndexTree) LoadExisting() bool {
	if !lit.lineIntIndex.LoadExisting() {
		return false
	}
	if lit.lineIntIndex.GetChecksum() != lit.checksum() {
		panic("location index was opened with incorrect graph")
	}
	lit.minResolutionInMeter = lit.lineIntIndex.GetMinResolutionInMeter()
	lit.indexStructureInfo = CreateIndexStructureInfo(lit.graph.GetBounds(), lit.minResolutionInMeter)
	lit.initialized = true
	return true
}

// Flush persists the index.
func (lit *LocationIndexTree) Flush() {
	lit.lineIntIndex.Flush()
}

// PrepareIndex builds the index from the graph. Uses AllEdges filter.
func (lit *LocationIndexTree) PrepareIndex() LocationIndex {
	return lit.PrepareIndexWithFilter(routeutil.AllEdges)
}

// PrepareIndexWithFilter builds the index from edges accepted by the filter.
func (lit *LocationIndexTree) PrepareIndexWithFilter(edgeFilter routeutil.EdgeFilter) LocationIndex {
	if lit.initialized {
		panic("call PrepareIndex only once")
	}

	bounds := lit.graph.GetBounds()
	if !bounds.IsValid() {
		bounds = util.NewBBox(-10.0, 10.0, -10.0, 10.0)
	}

	inMem := lit.prepareInMemConstructionIndex(bounds, edgeFilter)

	lit.lineIntIndex.SetMinResolutionInMeter(lit.minResolutionInMeter)
	lit.lineIntIndex.Store(inMem)
	lit.lineIntIndex.SetChecksum(lit.checksum())
	lit.Flush()

	log.Printf("location index created, size:%d, leafs:%d, precision:%d, depth:%d, checksum:%d, entries:%v",
		lit.lineIntIndex.GetSize(),
		lit.lineIntIndex.GetLeafs(),
		lit.minResolutionInMeter,
		len(lit.indexStructureInfo.Entries),
		lit.checksum(),
		lit.indexStructureInfo.Entries,
	)

	return lit
}

func (lit *LocationIndexTree) prepareInMemConstructionIndex(bounds util.BBox, edgeFilter routeutil.EdgeFilter) *InMemConstructionIndex {
	lit.indexStructureInfo = CreateIndexStructureInfo(bounds, lit.minResolutionInMeter)
	inMem := NewInMemConstructionIndex(lit.indexStructureInfo)
	allIter := lit.graph.GetAllEdges()
	for allIter.Next() {
		if !edgeFilter(allIter) {
			continue
		}
		edge := allIter.GetEdge()
		nodeA := allIter.GetBaseNode()
		nodeB := allIter.GetAdjNode()
		lat1 := lit.nodeAccess.GetLat(nodeA)
		lon1 := lit.nodeAccess.GetLon(nodeA)
		points := allIter.FetchWayGeometry(util.FetchModePillarOnly)
		for i := range points.Size() {
			lat2 := points.GetLat(i)
			lon2 := points.GetLon(i)
			inMem.AddToAllTilesOnLine(edge, lat1, lon1, lat2, lon2)
			lat1 = lat2
			lon1 = lon2
		}
		lat2 := lit.nodeAccess.GetLat(nodeB)
		lon2 := lit.nodeAccess.GetLon(nodeB)
		inMem.AddToAllTilesOnLine(edge, lat1, lon1, lat2, lon2)
	}
	return inMem
}

func (lit *LocationIndexTree) checksum() int32 {
	return int32(lit.graph.GetNodes() ^ lit.graph.GetAllEdges().Length())
}

// Close releases resources.
func (lit *LocationIndexTree) Close() {
	lit.lineIntIndex.Close()
}

// IsClosed returns true if the index has been closed.
func (lit *LocationIndexTree) IsClosed() bool {
	return lit.lineIntIndex.IsClosed()
}

// GetCapacity returns the capacity of the underlying storage.
func (lit *LocationIndexTree) GetCapacity() int64 {
	return lit.lineIntIndex.GetCapacity()
}

// CalculateRMin calculates the distance from (lat,lon) to the nearest tile
// border of the rectangular region that is (2*paddingTiles+1) tiles wide,
// centered on the tile containing (lat,lon).
func (lit *LocationIndexTree) CalculateRMin(lat, lon float64, paddingTiles int) float64 {
	x := lit.indexStructureInfo.KeyAlgo.X(lon)
	y := lit.indexStructureInfo.KeyAlgo.Y(lat)

	bounds := lit.graph.GetBounds()
	minLat := bounds.MinLat + float64(y-paddingTiles)*lit.indexStructureInfo.DeltaLat()
	maxLat := bounds.MinLat + float64(y+paddingTiles+1)*lit.indexStructureInfo.DeltaLat()
	minLon := bounds.MinLon + float64(x-paddingTiles)*lit.indexStructureInfo.DeltaLon()
	maxLon := bounds.MinLon + float64(x+paddingTiles+1)*lit.indexStructureInfo.DeltaLon()

	dSouthernLat := lat - minLat
	dNorthernLat := maxLat - lat
	dWesternLon := lon - minLon
	dEasternLon := maxLon - lon

	var dMinLat, dMinLon float64
	if dSouthernLat < dNorthernLat {
		dMinLat = util.DistPlane.CalcDist(lat, lon, minLat, lon)
	} else {
		dMinLat = util.DistPlane.CalcDist(lat, lon, maxLat, lon)
	}
	if dWesternLon < dEasternLon {
		dMinLon = util.DistPlane.CalcDist(lat, lon, lat, minLon)
	} else {
		dMinLon = util.DistPlane.CalcDist(lat, lon, lat, maxLon)
	}
	return min(dMinLat, dMinLon)
}

// FindClosest finds the closest snap for the given coordinates and edge filter.
func (lit *LocationIndexTree) FindClosest(queryLat, queryLon float64, edgeFilter routeutil.EdgeFilter) *Snap {
	if lit.IsClosed() {
		panic("you need to create a new LocationIndex instance as it is already closed")
	}

	snap := NewSnap(queryLat, queryLon)
	seenEdges := make(map[int]struct{})
	for iteration := 0; iteration < lit.maxRegionSearch; iteration++ {
		lit.lineIntIndex.FindEdgeIdsInNeighborhood(queryLat, queryLon, iteration, func(edgeID int) {
			if _, seen := seenEdges[edgeID]; seen {
				return
			}
			seenEdges[edgeID] = struct{}{}
			edgeState := lit.graph.GetEdgeIteratorStateForKey(edgeID * 2)
			if !edgeFilter(edgeState) {
				return
			}
			lit.TraverseEdge(queryLat, queryLon, edgeState, func(node int, normedDist float64, wayIndex int, pos Position) {
				if normedDist < snap.queryDistance {
					snap.queryDistance = normedDist
					snap.closestNode = node
					snap.closestEdge = edgeState.Detach(false)
					snap.wayIndex = wayIndex
					snap.snappedPosition = pos
				}
			})
		})
		if snap.IsValid() {
			rMin := lit.CalculateRMin(queryLat, queryLon, iteration)
			if util.DistPlane.CalcDenormalizedDist(snap.queryDistance) < rMin {
				break
			}
		}
	}

	if snap.IsValid() {
		snap.CalcSnappedPoint(util.DistPlane)
		sp := snap.GetSnappedPoint()
		snap.queryDistance = util.DistPlane.CalcDist(sp.Lat, sp.Lon, queryLat, queryLon)
	}
	return snap
}

// Query explores the index with the specified TileFilter and Visitor.
func (lit *LocationIndexTree) Query(tileFilter TileFilter, visitor Visitor) {
	lit.lineIntIndex.Query(tileFilter, visitor)
}

// TraverseEdge checks the distance from the query point to each segment of
// the edge and calls edgeCheck for each candidate.
func (lit *LocationIndexTree) TraverseEdge(queryLat, queryLon float64, currEdge util.EdgeIteratorState, edgeCheck EdgeCheck) {
	baseNode := currEdge.GetBaseNode()
	baseLat := lit.nodeAccess.GetLat(baseNode)
	baseLon := lit.nodeAccess.GetLon(baseNode)
	baseDist := util.DistPlane.CalcNormalizedDistCoords(queryLat, queryLon, baseLat, baseLon)

	adjNode := currEdge.GetAdjNode()
	adjLat := lit.nodeAccess.GetLat(adjNode)
	adjLon := lit.nodeAccess.GetLon(adjNode)
	adjDist := util.DistPlane.CalcNormalizedDistCoords(queryLat, queryLon, adjLat, adjLon)

	pointList := currEdge.FetchWayGeometry(util.FetchModePillarAndAdj)
	length := pointList.Size()

	var closestTowerNode int
	var closestDist float64
	if baseDist < adjDist {
		closestTowerNode = baseNode
		closestDist = baseDist
		edgeCheck(baseNode, baseDist, 0, Tower)
	} else {
		closestTowerNode = adjNode
		closestDist = adjDist
		edgeCheck(adjNode, adjDist, length, Tower)
	}
	if closestDist <= lit.equalNormedDelta {
		return
	}

	lastLat := baseLat
	lastLon := baseLon
	for i := range length {
		lat := pointList.GetLat(i)
		lon := pointList.GetLon(i)
		if util.DistPlane.IsCrossBoundary(lastLon, lon) {
			lastLat = lat
			lastLon = lon
			continue
		}
		// +1 because we skipped the base node
		indexInFullPointList := i + 1
		if util.DistPlane.ValidEdgeDistance(queryLat, queryLon, lastLat, lastLon, lat, lon) {
			closestDist = util.DistPlane.CalcNormalizedEdgeDistance(queryLat, queryLon, lastLat, lastLon, lat, lon)
			edgeCheck(closestTowerNode, closestDist, indexInFullPointList-1, Edge)
		} else if i < length-1 {
			closestDist = util.DistPlane.CalcNormalizedDistCoords(queryLat, queryLon, lat, lon)
			edgeCheck(closestTowerNode, closestDist, indexInFullPointList, Pillar)
		}
		if closestDist <= lit.equalNormedDelta {
			return
		}
		lastLat = lat
		lastLon = lon
	}
}
