package querygraph

import (
	"sort"

	"gohopper/core/storage"
	"gohopper/core/storage/index"
	"gohopper/core/util"
)

type flagsGetter interface {
	GetFlags() *storage.IntsRef
}

func BuildQueryOverlay(graph storage.Graph, snaps []*index.Snap) *QueryOverlay {
	return buildQueryOverlay(graph.GetNodes(), graph.GetEdges(), graph.GetNodeAccess().Is3D(), snaps)
}

func buildQueryOverlay(firstVirtualNodeID, firstVirtualEdgeID int, is3D bool, snaps []*index.Snap) *QueryOverlay {
	b := &queryOverlayBuilder{
		firstVirtualNodeID: firstVirtualNodeID,
		firstVirtualEdgeID: firstVirtualEdgeID,
		is3D:               is3D,
	}
	return b.build(snaps)
}

type queryOverlayBuilder struct {
	firstVirtualNodeID int
	firstVirtualEdgeID int
	is3D               bool
	queryOverlay       *QueryOverlay
}

func (b *queryOverlayBuilder) build(resList []*index.Snap) *QueryOverlay {
	b.queryOverlay = newQueryOverlay(len(resList), b.is3D)
	b.buildVirtualEdges(resList)
	b.buildEdgeChangesAtRealNodes()
	return b.queryOverlay
}

func (b *queryOverlayBuilder) buildVirtualEdges(snaps []*index.Snap) {
	edge2res := make(map[int][]*index.Snap, len(snaps))

	for _, snap := range snaps {
		if snap.GetSnappedPosition() == index.Tower {
			continue
		}
		closestEdge := snap.GetClosestEdge()
		if closestEdge == nil {
			panic("do not call QueryGraph.Create with invalid Snap")
		}
		base := closestEdge.GetBaseNode()

		doReverse := base > closestEdge.GetAdjNode()
		if base == closestEdge.GetAdjNode() {
			pl := closestEdge.FetchWayGeometry(util.FetchModePillarOnly)
			if pl.Size() > 1 {
				doReverse = pl.GetLat(0) > pl.GetLat(pl.Size()-1)
			}
		}

		if doReverse {
			closestEdge = closestEdge.Detach(true)
			fullPL := closestEdge.FetchWayGeometry(util.FetchModeAll)
			snap.SetClosestEdge(closestEdge)
			if snap.GetSnappedPosition() == index.Pillar {
				snap.SetWayIndex(fullPL.Size() - snap.GetWayIndex() - 1)
			} else {
				snap.SetWayIndex(fullPL.Size() - snap.GetWayIndex() - 2)
			}
			if snap.GetWayIndex() < 0 {
				panic("problem with wayIndex while reversing closest edge")
			}
		}

		edgeID := closestEdge.GetEdge()
		edge2res[edgeID] = append(edge2res[edgeID], snap)
	}

	for _, results := range edge2res {
		closestEdge := results[0].GetClosestEdge()
		fullPL := closestEdge.FetchWayGeometry(util.FetchModeAll)
		baseNode := closestEdge.GetBaseNode()

		sort.Slice(results, func(i, j int) bool {
			diff := results[i].GetWayIndex() - results[j].GetWayIndex()
			if diff != 0 {
				return diff < 0
			}
			di := distanceOfSnappedPointToPillarNode(results[i], fullPL)
			dj := distanceOfSnappedPointToPillarNode(results[j], fullPL)
			return di < dj
		})

		prevPoint := fullPL.Get(0)
		adjNode := closestEdge.GetAdjNode()
		origEdgeKey := closestEdge.GetEdgeKey()
		origRevEdgeKey := closestEdge.GetReverseEdgeKey()
		prevWayIndex := 1
		prevNodeID := baseNode
		virtNodeID := b.queryOverlay.getVirtualNodes().Size() + b.firstVirtualNodeID
		addedEdges := false

		for i, res := range results {
			if res.GetClosestEdge().GetBaseNode() != baseNode {
				panic("base nodes have to be identical but were not")
			}
			currSnapped := res.GetSnappedPoint()

			if index.ConsiderEqual(prevPoint.Lat, prevPoint.Lon, currSnapped.Lat, currSnapped.Lon) {
				res.SetClosestNode(prevNodeID)
				res.SetSnappedPoint(prevPoint)
				if i == 0 {
					res.SetWayIndex(0)
					res.SetSnappedPosition(index.Tower)
				} else {
					res.SetWayIndex(results[i-1].GetWayIndex())
					res.SetSnappedPosition(results[i-1].GetSnappedPosition())
				}
				res.SetQueryDistance(util.DistPlane.CalcDist(prevPoint.Lat, prevPoint.Lon, res.GetQueryPoint().Lat, res.GetQueryPoint().Lon))
				continue
			}

			b.queryOverlay.closestEdges = append(b.queryOverlay.closestEdges, res.GetClosestEdge().GetEdge())
			isPillar := res.GetSnappedPosition() == index.Pillar
			b.createEdges(origEdgeKey, origRevEdgeKey,
				prevPoint, prevWayIndex, isPillar,
				res.GetSnappedPoint(), res.GetWayIndex(),
				fullPL, closestEdge, prevNodeID, virtNodeID)

			b.queryOverlay.getVirtualNodes().Add3D(currSnapped.Lat, currSnapped.Lon, currSnapped.Ele)

			if addedEdges {
				b.queryOverlay.addVirtualEdge(b.queryOverlay.getVirtualEdge(b.queryOverlay.getNumVirtualEdges() - 2))
				b.queryOverlay.addVirtualEdge(b.queryOverlay.getVirtualEdge(b.queryOverlay.getNumVirtualEdges() - 2))
			}

			addedEdges = true
			res.SetClosestNode(virtNodeID)
			prevNodeID = virtNodeID
			prevWayIndex = res.GetWayIndex() + 1
			prevPoint = currSnapped
			virtNodeID++
		}

		if addedEdges {
			b.createEdges(origEdgeKey, origRevEdgeKey,
				prevPoint, prevWayIndex, false,
				fullPL.Get(fullPL.Size()-1), fullPL.Size()-2,
				fullPL, closestEdge, virtNodeID-1, adjNode)
		}
	}
}

func (b *queryOverlayBuilder) createEdges(origEdgeKey, origRevEdgeKey int,
	prevSnapped util.GHPoint3D, prevWayIndex int, isPillar bool, currSnapped util.GHPoint3D, wayIndex int,
	fullPL *util.PointList, closestEdge util.EdgeIteratorState,
	prevNodeID, nodeID int) {

	maxIdx := wayIndex + 1
	basePoints := util.NewPointList(maxIdx-prevWayIndex+1, b.is3D)
	basePoints.Add3D(prevSnapped.Lat, prevSnapped.Lon, prevSnapped.Ele)
	for i := prevWayIndex; i < maxIdx; i++ {
		basePoints.AddFrom(fullPL, i)
	}
	if !isPillar {
		basePoints.Add3D(currSnapped.Lat, currSnapped.Lon, currSnapped.Ele)
	}

	baseReversePoints := basePoints.Clone(true)
	baseDistance := util.DistPlane.CalcPointListDistance(basePoints)
	virtEdgeID := b.firstVirtualEdgeID + b.queryOverlay.getNumVirtualEdges()/2

	reverse := closestEdge.GetBool(util.ReverseState)

	var edgeFlags *storage.IntsRef
	if fg, ok := closestEdge.(flagsGetter); ok {
		edgeFlags = fg.GetFlags()
	} else {
		edgeFlags = storage.NewIntsRef(1)
	}
	keyValues := closestEdge.GetKeyValues()

	baseEdge := NewVirtualEdgeIteratorState(origEdgeKey, util.CreateEdgeKey(virtEdgeID, false),
		prevNodeID, nodeID, baseDistance, edgeFlags, keyValues, basePoints, reverse)
	baseReverseEdge := NewVirtualEdgeIteratorState(origRevEdgeKey, util.CreateEdgeKey(virtEdgeID, true),
		nodeID, prevNodeID, baseDistance, edgeFlags.DeepCopy(), keyValues, baseReversePoints, !reverse)

	baseEdge.SetReverseEdge(baseReverseEdge)
	baseReverseEdge.SetReverseEdge(baseEdge)
	b.queryOverlay.addVirtualEdge(baseEdge)
	b.queryOverlay.addVirtualEdge(baseReverseEdge)
}

func (b *queryOverlayBuilder) buildEdgeChangesAtRealNodes() {
	buildEdgeChanges(b.queryOverlay.closestEdges, b.queryOverlay.virtualEdges, b.firstVirtualNodeID, b.queryOverlay.edgeChangesAtRealNodes)
}

func distanceOfSnappedPointToPillarNode(snap *index.Snap, fullPL *util.PointList) float64 {
	sp := snap.GetSnappedPoint()
	wayIdx := snap.GetWayIndex()
	return util.DistPlane.CalcNormalizedDistCoords(fullPL.GetLat(wayIdx), fullPL.GetLon(wayIdx), sp.Lat, sp.Lon)
}
