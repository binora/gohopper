package storage

import (
	"gohopper/core/routing/ev"
	"gohopper/core/util"
)

const (
	NoTurnEntry     = -1
	tcFrom          = 0
	tcTo            = 4
	tcFlags         = 8
	tcNext          = 12
	bytesPerTCEntry = 16
)

type TurnCostStorage struct {
	da    DataAccess
	count int
}

func NewTurnCostStorage(dir Directory, segmentSize int) *TurnCostStorage {
	return &TurnCostStorage{
		da: dir.CreateFull("turn_costs", dir.DefaultTypeFor("turn_costs", true), segmentSize),
	}
}

func (tc *TurnCostStorage) Create(initBytes int64) {
	tc.da.Create(initBytes)
}

func (tc *TurnCostStorage) LoadExisting() bool {
	if !tc.da.LoadExisting() {
		return false
	}
	checkDAVersion("turn_costs", util.VersionTurnCosts, int(tc.da.GetHeader(0*4)))
	if tc.da.GetHeader(1*4) != bytesPerTCEntry {
		panic("turn cost storage: unexpected bytes per entry")
	}
	tc.count = int(tc.da.GetHeader(2 * 4))
	return true
}

func (tc *TurnCostStorage) Flush() {
	tc.da.SetHeader(0*4, int32(util.VersionTurnCosts))
	tc.da.SetHeader(1*4, bytesPerTCEntry)
	tc.da.SetHeader(2*4, int32(tc.count))
	tc.da.Flush()
}

func (tc *TurnCostStorage) Close()        { tc.da.Close() }
func (tc *TurnCostStorage) IsClosed() bool { return tc.da.IsClosed() }

func (tc *TurnCostStorage) toPointer(index int) int64 {
	return int64(index) * bytesPerTCEntry
}

func (tc *TurnCostStorage) GetFromEdge(index int) int {
	return int(tc.da.GetInt(tc.toPointer(index) + tcFrom))
}

func (tc *TurnCostStorage) GetToEdge(index int) int {
	return int(tc.da.GetInt(tc.toPointer(index) + tcTo))
}

func (tc *TurnCostStorage) GetFlags(index int) int32 {
	return tc.da.GetInt(tc.toPointer(index) + tcFlags)
}

func (tc *TurnCostStorage) GetNext(index int) int {
	return int(tc.da.GetInt(tc.toPointer(index) + tcNext))
}

func (tc *TurnCostStorage) Count() int {
	return tc.count
}

func (tc *TurnCostStorage) GetCapacity() int64 {
	return tc.da.Capacity()
}

func (tc *TurnCostStorage) FindOrCreateEntry(na NodeAccess, fromEdge, viaNode, toEdge int) int {
	idx := na.GetTurnCostIndex(viaNode)
	prevIdx := NoTurnEntry
	for idx != NoTurnEntry {
		ptr := tc.toPointer(idx)
		if tc.da.GetInt(ptr+tcFrom) == int32(fromEdge) && tc.da.GetInt(ptr+tcTo) == int32(toEdge) {
			return idx
		}
		prevIdx = idx
		idx = int(tc.da.GetInt(ptr + tcNext))
	}

	newIdx := tc.count
	tc.count++
	tc.da.EnsureCapacity(int64(tc.count) * bytesPerTCEntry)
	ptr := tc.toPointer(newIdx)
	tc.da.SetInt(ptr+tcFrom, int32(fromEdge))
	tc.da.SetInt(ptr+tcTo, int32(toEdge))
	tc.da.SetInt(ptr+tcFlags, 0)
	tc.da.SetInt(ptr+tcNext, int32(NoTurnEntry))

	if prevIdx == NoTurnEntry {
		na.SetTurnCostIndex(viaNode, newIdx)
	} else {
		tc.da.SetInt(tc.toPointer(prevIdx)+tcNext, int32(newIdx))
	}
	return newIdx
}

func (tc *TurnCostStorage) SetFlags(index int, flags int32) {
	tc.da.SetInt(tc.toPointer(index)+tcFlags, flags)
}

type tcEdgeIntAccess struct {
	tc *TurnCostStorage
}

func (a *tcEdgeIntAccess) GetInt(entryIndex, index int) int32 {
	return a.tc.da.GetInt(a.tc.toPointer(entryIndex) + tcFlags)
}

func (a *tcEdgeIntAccess) SetInt(entryIndex, index int, value int32) {
	a.tc.da.SetInt(a.tc.toPointer(entryIndex)+tcFlags, value)
}

func (tc *TurnCostStorage) findIndex(na NodeAccess, fromEdge, viaNode, toEdge int) int {
	if !util.EdgeIsValid(fromEdge) || !util.EdgeIsValid(toEdge) {
		panic("from and to edge cannot be NO_EDGE")
	}
	if viaNode < 0 {
		panic("via node cannot be negative")
	}
	const maxEntries = 1000
	idx := na.GetTurnCostIndex(viaNode)
	for range maxEntries {
		if idx == NoTurnEntry {
			return -1
		}
		ptr := tc.toPointer(idx)
		if int(tc.da.GetInt(ptr+tcFrom)) == fromEdge && int(tc.da.GetInt(ptr+tcTo)) == toEdge {
			return idx
		}
		idx = int(tc.da.GetInt(ptr + tcNext))
	}
	panic("turn cost list is longer than expected")
}

func (tc *TurnCostStorage) GetDecimal(na NodeAccess, dev ev.DecimalEncodedValue, fromEdge, viaNode, toEdge int) float64 {
	idx := tc.findIndex(na, fromEdge, viaNode, toEdge)
	if idx < 0 {
		return 0
	}
	return dev.GetDecimal(false, idx, &tcEdgeIntAccess{tc})
}

func (tc *TurnCostStorage) GetBool(na NodeAccess, bev ev.BooleanEncodedValue, fromEdge, viaNode, toEdge int) bool {
	idx := tc.findIndex(na, fromEdge, viaNode, toEdge)
	if idx < 0 {
		return false
	}
	return bev.GetBool(false, idx, &tcEdgeIntAccess{tc})
}

func (tc *TurnCostStorage) SetBool(na NodeAccess, bev ev.BooleanEncodedValue, fromEdge, viaNode, toEdge int, value bool) {
	idx := tc.FindOrCreateEntry(na, fromEdge, viaNode, toEdge)
	if idx < 0 {
		panic("invalid turn cost entry index")
	}
	bev.SetBool(false, idx, &tcEdgeIntAccess{tc}, value)
}

func (tc *TurnCostStorage) SortEdges(getNewEdge func(int) int) {
	for i := range tc.count {
		ptr := tc.toPointer(i)
		tc.da.SetInt(ptr+tcFrom, int32(getNewEdge(int(tc.da.GetInt(ptr+tcFrom)))))
		tc.da.SetInt(ptr+tcTo, int32(getNewEdge(int(tc.da.GetInt(ptr+tcTo)))))
	}
}

type tcEntry struct {
	from, to, flags, next int32
}

func (tc *TurnCostStorage) SortNodes(nodeCount int, na NodeAccess) {
	entries := make([]tcEntry, tc.count)
	for i := range tc.count {
		ptr := tc.toPointer(i)
		entries[i] = tcEntry{
			from:  tc.da.GetInt(ptr + tcFrom),
			to:    tc.da.GetInt(ptr + tcTo),
			flags: tc.da.GetInt(ptr + tcFlags),
			next:  tc.da.GetInt(ptr + tcNext),
		}
	}

	countBefore := tc.count
	tc.count = 0
	for node := range nodeCount {
		firstForNode := true
		idx := na.GetTurnCostIndex(node)
		for idx != NoTurnEntry {
			if firstForNode {
				na.SetTurnCostIndex(node, tc.count)
			} else {
				tc.da.SetInt(tc.toPointer(tc.count-1)+tcNext, int32(tc.count))
			}
			e := entries[idx]
			ptr := tc.toPointer(tc.count)
			tc.da.SetInt(ptr+tcFrom, e.from)
			tc.da.SetInt(ptr+tcTo, e.to)
			tc.da.SetInt(ptr+tcFlags, e.flags)
			tc.da.SetInt(ptr+tcNext, int32(NoTurnEntry))
			tc.count++
			firstForNode = false
			idx = int(e.next)
		}
	}
	if countBefore != tc.count {
		panic("turn cost count changed unexpectedly")
	}
}

func (tc *TurnCostStorage) SetDecimal(na NodeAccess, dev ev.DecimalEncodedValue, fromEdge, viaNode, toEdge int, cost float64) {
	idx := tc.FindOrCreateEntry(na, fromEdge, viaNode, toEdge)
	if idx < 0 {
		panic("invalid turn cost entry index")
	}
	dev.SetDecimal(false, idx, &tcEdgeIntAccess{tc}, cost)
}

func (tc *TurnCostStorage) GetTurnCostsCount(na NodeAccess, node int) int {
	idx := na.GetTurnCostIndex(node)
	count := 0
	for idx != NoTurnEntry {
		ptr := tc.toPointer(idx)
		idx = int(tc.da.GetInt(ptr + tcNext))
		count++
	}
	return count
}

type TurnCostIterator struct {
	tc            *TurnCostStorage
	na            NodeAccess
	nodeCount     int
	viaNode       int
	turnCostIndex int
	intsRef       *IntsRef
	eia           ev.EdgeIntAccess
}

func (tc *TurnCostStorage) GetAllTurnCosts(na NodeAccess, nodeCount int) *TurnCostIterator {
	ref := NewIntsRef(1)
	return &TurnCostIterator{
		tc:            tc,
		na:            na,
		nodeCount:     nodeCount,
		viaNode:       -1,
		turnCostIndex: NoTurnEntry,
		intsRef:       ref,
		eia:           NewIntsRefEdgeIntAccess(ref),
	}
}

func (it *TurnCostIterator) turnCostPtr() int64 {
	return it.tc.toPointer(it.turnCostIndex)
}

func (it *TurnCostIterator) GetFromEdge() int {
	return int(it.tc.da.GetInt(it.turnCostPtr() + tcFrom))
}

func (it *TurnCostIterator) GetViaNode() int {
	return it.viaNode
}

func (it *TurnCostIterator) GetToEdge() int {
	return int(it.tc.da.GetInt(it.turnCostPtr() + tcTo))
}

func (it *TurnCostIterator) GetBool(bev ev.BooleanEncodedValue) bool {
	it.intsRef.Ints[0] = it.tc.da.GetInt(it.turnCostPtr() + tcFlags)
	return bev.GetBool(false, -1, it.eia)
}

func (it *TurnCostIterator) GetCost(dev ev.DecimalEncodedValue) float64 {
	it.intsRef.Ints[0] = it.tc.da.GetInt(it.turnCostPtr() + tcFlags)
	return dev.GetDecimal(false, -1, it.eia)
}

func (it *TurnCostIterator) Next() bool {
	if it.nextTCI() {
		return true
	}
	for it.nextNode() {
		if it.turnCostIndex != NoTurnEntry {
			return true
		}
	}
	return false
}

func (it *TurnCostIterator) nextNode() bool {
	it.viaNode++
	if it.viaNode >= it.nodeCount {
		return false
	}
	it.turnCostIndex = it.na.GetTurnCostIndex(it.viaNode)
	return true
}

func (it *TurnCostIterator) nextTCI() bool {
	if it.turnCostIndex == NoTurnEntry {
		return false
	}
	it.turnCostIndex = int(it.tc.da.GetInt(it.turnCostPtr() + tcNext))
	return it.turnCostIndex != NoTurnEntry
}
