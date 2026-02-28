package storage

import "gohopper/core/util"

const (
	NoTurnEntry     = -1
	tcFrom          = 0
	tcTo            = 4
	tcFlags         = 8
	tcNext          = 12
	bytesPerTCEntry = 16
)

// TurnCostStorage stores turn restrictions/costs using a linked-list per node.
// Each entry: |from_edge(4)|to_edge(4)|flags(4)|next(4)|
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

// FindOrCreateEntry finds an existing entry matching (fromEdge, viaNode, toEdge)
// using the node's turn cost linked list, or creates a new one.
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
