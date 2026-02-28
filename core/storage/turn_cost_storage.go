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
// Each entry is 16 bytes: |from_edge(4)|to_edge(4)|flags(4)|next(4)|
type TurnCostStorage struct {
	turnCosts      DataAccess
	turnCostsCount int
}

func NewTurnCostStorage(dir Directory, segmentSize int) *TurnCostStorage {
	return &TurnCostStorage{
		turnCosts: dir.CreateFull("turn_costs", dir.DefaultTypeFor("turn_costs", true), segmentSize),
	}
}

func (tc *TurnCostStorage) Create(initBytes int64) {
	tc.turnCosts.Create(initBytes)
}

func (tc *TurnCostStorage) LoadExisting() bool {
	if !tc.turnCosts.LoadExisting() {
		return false
	}
	version := tc.turnCosts.GetHeader(0 * 4)
	checkDAVersion("turn_costs", util.VersionTurnCosts, int(version))
	bytesPerEntry := tc.turnCosts.GetHeader(1 * 4)
	if bytesPerEntry != bytesPerTCEntry {
		panic("turn cost storage: unexpected bytes per entry")
	}
	tc.turnCostsCount = int(tc.turnCosts.GetHeader(2 * 4))
	return true
}

func (tc *TurnCostStorage) Flush() {
	tc.turnCosts.SetHeader(0*4, int32(util.VersionTurnCosts))
	tc.turnCosts.SetHeader(1*4, bytesPerTCEntry)
	tc.turnCosts.SetHeader(2*4, int32(tc.turnCostsCount))
	tc.turnCosts.Flush()
}

func (tc *TurnCostStorage) Close() {
	tc.turnCosts.Close()
}

func (tc *TurnCostStorage) IsClosed() bool {
	return tc.turnCosts.IsClosed()
}

func (tc *TurnCostStorage) toPointer(index int) int64 {
	return int64(index) * bytesPerTCEntry
}

// GetFromEdge returns the from-edge of the turn cost entry at the given index.
func (tc *TurnCostStorage) GetFromEdge(index int) int {
	return int(tc.turnCosts.GetInt(tc.toPointer(index) + tcFrom))
}

// GetToEdge returns the to-edge of the turn cost entry at the given index.
func (tc *TurnCostStorage) GetToEdge(index int) int {
	return int(tc.turnCosts.GetInt(tc.toPointer(index) + tcTo))
}

// GetFlags returns the flags of the turn cost entry at the given index.
func (tc *TurnCostStorage) GetFlags(index int) int32 {
	return tc.turnCosts.GetInt(tc.toPointer(index) + tcFlags)
}

// GetNext returns the next linked-list pointer of the turn cost entry.
func (tc *TurnCostStorage) GetNext(index int) int {
	return int(tc.turnCosts.GetInt(tc.toPointer(index) + tcNext))
}

// Count returns the number of turn cost entries.
func (tc *TurnCostStorage) Count() int {
	return tc.turnCostsCount
}

// FindOrCreateEntry finds an existing entry matching (fromEdge, viaNode, toEdge) using the
// node's turn cost linked list, or creates a new one.
func (tc *TurnCostStorage) FindOrCreateEntry(nodeAccess NodeAccess, fromEdge, viaNode, toEdge int) int {
	idx := nodeAccess.GetTurnCostIndex(viaNode)
	prevIdx := NoTurnEntry
	for idx != NoTurnEntry {
		ptr := tc.toPointer(idx)
		if tc.turnCosts.GetInt(ptr+tcFrom) == int32(fromEdge) && tc.turnCosts.GetInt(ptr+tcTo) == int32(toEdge) {
			return idx
		}
		prevIdx = idx
		idx = int(tc.turnCosts.GetInt(ptr + tcNext))
	}
	// Create new entry
	newIdx := tc.turnCostsCount
	tc.turnCostsCount++
	tc.turnCosts.EnsureCapacity(int64(tc.turnCostsCount) * bytesPerTCEntry)
	ptr := tc.toPointer(newIdx)
	tc.turnCosts.SetInt(ptr+tcFrom, int32(fromEdge))
	tc.turnCosts.SetInt(ptr+tcTo, int32(toEdge))
	tc.turnCosts.SetInt(ptr+tcFlags, 0)
	tc.turnCosts.SetInt(ptr+tcNext, int32(NoTurnEntry))

	if prevIdx == NoTurnEntry {
		nodeAccess.SetTurnCostIndex(viaNode, newIdx)
	} else {
		tc.turnCosts.SetInt(tc.toPointer(prevIdx)+tcNext, int32(newIdx))
	}
	return newIdx
}

// SetFlags sets the flags of the turn cost entry at the given index.
func (tc *TurnCostStorage) SetFlags(index int, flags int32) {
	tc.turnCosts.SetInt(tc.toPointer(index)+tcFlags, flags)
}
