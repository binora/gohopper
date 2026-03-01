package index

import (
	"gohopper/core/geohash"
	"gohopper/core/util"
)

// InMemEntry is a node in the in-memory construction tree.
// It is either a leaf (holding edge IDs) or an interior node (holding
// sub-entries).
type InMemEntry interface {
	IsLeaf() bool
}

// InMemLeafEntry stores the edge IDs that belong to a single spatial cell.
type InMemLeafEntry struct {
	Results []int
}

// NewInMemLeafEntry creates a leaf entry with the given initial capacity.
func NewInMemLeafEntry(capacity int) *InMemLeafEntry {
	return &InMemLeafEntry{Results: make([]int, 0, capacity)}
}

// IsLeaf returns true.
func (e *InMemLeafEntry) IsLeaf() bool { return true }

// String returns a debug representation.
func (e *InMemLeafEntry) String() string { return "LEAF" }

// InMemTreeEntry is an interior node holding references to sub-entries.
type InMemTreeEntry struct {
	SubEntries []InMemEntry
}

// NewInMemTreeEntry creates a tree entry with the given number of sub-entry slots.
func NewInMemTreeEntry(subEntryNo int) *InMemTreeEntry {
	return &InMemTreeEntry{SubEntries: make([]InMemEntry, subEntryNo)}
}

// IsLeaf returns false.
func (e *InMemTreeEntry) IsLeaf() bool { return false }

// String returns a debug representation.
func (e *InMemTreeEntry) String() string { return "TREE" }

// InMemConstructionIndex builds an in-memory spatial index tree that is later
// serialized to the on-disk format.
type InMemConstructionIndex struct {
	pixelGridTraversal *PixelGridTraversal
	keyAlgo            *geohash.SpatialKeyAlgo
	Entries            []int
	Shifts             []byte
	Root               *InMemTreeEntry
}

// NewInMemConstructionIndex creates a new InMemConstructionIndex from the given
// IndexStructureInfo.
func NewInMemConstructionIndex(info *IndexStructureInfo) *InMemConstructionIndex {
	return &InMemConstructionIndex{
		Root:               NewInMemTreeEntry(info.Entries[0]),
		Entries:            info.Entries,
		Shifts:             info.Shifts,
		pixelGridTraversal: info.Grid,
		keyAlgo:            info.KeyAlgo,
	}
}

// AddToAllTilesOnLine rasterizes the edge segment (lat1,lon1)-(lat2,lon2) into
// the grid and inserts edgeID into every leaf cell the segment passes through.
func (idx *InMemConstructionIndex) AddToAllTilesOnLine(edgeID int, lat1, lon1, lat2, lon2 float64) {
	if !util.DistPlane.IsCrossBoundary(lon1, lon2) {
		// Traverse all grid cells on the line in tile coordinates (y, x).
		idx.pixelGridTraversal.Traverse(
			[2]float64{lon1, lat1},
			[2]float64{lon2, lat2},
			func(x, y int) {
				key := idx.keyAlgo.Encode(x, y)
				idx.put(key, edgeID)
			},
		)
	}
}

// put inserts a value into the tree under the given spatial key.
func (idx *InMemConstructionIndex) put(key int64, value int) {
	idx.putRecursive(key<<(64-idx.keyAlgo.Bits()), idx.Root, 0, value)
}

// putRecursive walks/creates the tree path defined by keyPart and inserts value
// into the appropriate leaf.
func (idx *InMemConstructionIndex) putRecursive(keyPart int64, entry InMemEntry, depth int, value int) {
	if entry.IsLeaf() {
		leaf := entry.(*InMemLeafEntry)
		// Avoid adding the same edge id multiple times.
		// Since each edge id is handled only once, this can only happen when
		// this method is called several times in a row with the same edge id,
		// so it is enough to check the last entry.
		if len(leaf.Results) == 0 || leaf.Results[len(leaf.Results)-1] != value {
			leaf.Results = append(leaf.Results, value)
		}
		return
	}

	shift := idx.Shifts[depth]
	index := int(uint64(keyPart) >> (64 - shift))
	keyPart <<= shift
	tree := entry.(*InMemTreeEntry)
	sub := tree.SubEntries[index]
	depth++

	if sub == nil {
		if depth == len(idx.Entries) {
			sub = NewInMemLeafEntry(4)
		} else {
			sub = NewInMemTreeEntry(idx.Entries[depth])
		}
		tree.SubEntries[index] = sub
	}

	idx.putRecursive(keyPart, sub, depth, value)
}
