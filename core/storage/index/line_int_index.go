package index

import (
	"fmt"

	"gohopper/core/geohash"
	"gohopper/core/storage"
	"gohopper/core/util"
)

// startPointer is the first int-pointer used in the DataAccess.
// We do not start with 0 because a positive value means leaf and a negative
// value means "entry with sub-entries".
const startPointer = 1

// LineIntIndex is a persistent spatial index backed by a DataAccess.
// It stores edges organized by their spatial key in a tree structure
// that mirrors the in-memory construction index.
type LineIntIndex struct {
	dataAccess           storage.DataAccess
	bounds               util.BBox
	minResolutionInMeter int
	size                 int
	leafs                int
	checksum             int32
	indexStructureInfo   *IndexStructureInfo
	entries              []int
	shifts               []byte
	initialized          bool
	keyAlgo              *geohash.SpatialKeyAlgo
}

// NewLineIntIndex creates a new LineIntIndex.
func NewLineIntIndex(bbox util.BBox, dir storage.Directory, name string) *LineIntIndex {
	return NewLineIntIndexWithType(bbox, dir, name, dir.DefaultTypeFor(name, true))
}

// NewLineIntIndexWithType creates a new LineIntIndex with an explicit DAType.
func NewLineIntIndexWithType(bbox util.BBox, dir storage.Directory, name string, daType storage.DAType) *LineIntIndex {
	return &LineIntIndex{
		bounds:               bbox,
		dataAccess:           dir.CreateWithType(name, daType),
		minResolutionInMeter: 300,
	}
}

// LoadExisting loads a previously stored index from the DataAccess.
// Returns true on success. Must be called at most once.
func (idx *LineIntIndex) LoadExisting() bool {
	if idx.initialized {
		panic("call LoadExisting only once")
	}
	if !idx.dataAccess.LoadExisting() {
		return false
	}
	storedVersion := int(idx.dataAccess.GetHeader(0))
	if storedVersion != util.VersionLocationIdx {
		panic(fmt.Sprintf("cannot load location_index - expected version %d, got %d. "+
			"Make sure you are using the correct version of GraphHopper",
			util.VersionLocationIdx, storedVersion))
	}
	idx.checksum = idx.dataAccess.GetHeader(1 * 4)
	idx.minResolutionInMeter = int(idx.dataAccess.GetHeader(2 * 4))
	idx.indexStructureInfo = CreateIndexStructureInfo(idx.bounds, idx.minResolutionInMeter)
	idx.keyAlgo = idx.indexStructureInfo.KeyAlgo
	idx.entries = idx.indexStructureInfo.Entries
	idx.shifts = idx.indexStructureInfo.Shifts
	idx.initialized = true
	return true
}

// Store serializes the in-memory construction index to the DataAccess.
func (idx *LineIntIndex) Store(inMem *InMemConstructionIndex) {
	idx.indexStructureInfo = CreateIndexStructureInfo(idx.bounds, idx.minResolutionInMeter)
	idx.keyAlgo = idx.indexStructureInfo.KeyAlgo
	idx.entries = idx.indexStructureInfo.Entries
	idx.shifts = idx.indexStructureInfo.Shifts
	idx.dataAccess.Create(64 * 1024)
	idx.storeEntry(inMem.Root, startPointer)
	idx.initialized = true
}

func (idx *LineIntIndex) storeEntry(entry InMemEntry, intPointer int) int {
	pointer := int64(intPointer) * 4
	if entry.IsLeaf() {
		leaf := entry.(*InMemLeafEntry)
		results := leaf.Results
		length := len(results)
		if length == 0 {
			return intPointer
		}
		idx.size += length
		intPointer++
		idx.leafs++
		idx.dataAccess.EnsureCapacity(int64(intPointer+length+1) * 4)
		if length == 1 {
			// less disc space for single entries
			idx.dataAccess.SetInt(pointer, int32(-results[0]-1))
		} else {
			for _, r := range results {
				idx.dataAccess.SetInt(int64(intPointer)*4, int32(r))
				intPointer++
			}
			idx.dataAccess.SetInt(pointer, int32(intPointer))
		}
	} else {
		treeEntry := entry.(*InMemTreeEntry)
		intPointer += len(treeEntry.SubEntries)
		for _, subEntry := range treeEntry.SubEntries {
			if subEntry == nil {
				pointer += 4
				continue
			}
			idx.dataAccess.EnsureCapacity(int64(intPointer+1) * 4)
			prevIntPointer := intPointer
			intPointer = idx.storeEntry(subEntry, prevIntPointer)
			if intPointer == prevIntPointer {
				idx.dataAccess.SetInt(pointer, 0)
			} else {
				idx.dataAccess.SetInt(pointer, int32(prevIntPointer))
			}
			pointer += 4
		}
	}
	return intPointer
}

// fillIDs walks the tree for the given key part and calls consumer for each
// found edge ID.
func (idx *LineIntIndex) fillIDs(keyPart int64, consumer func(edgeID int)) {
	intPointer := startPointer
	for _, shift := range idx.shifts {
		offset := int(uint64(keyPart) >> (64 - shift))
		nextIntPointer := int(idx.dataAccess.GetInt(int64(intPointer+offset) * 4))
		if nextIntPointer <= 0 {
			return
		}
		keyPart <<= shift
		intPointer = nextIntPointer
	}
	data := int(idx.dataAccess.GetInt(int64(intPointer) * 4))
	if data < 0 {
		// single data entry (less disc space)
		consumer(-(data + 1))
	} else {
		// "data" is index of last data item
		for leafIndex := intPointer + 1; leafIndex < data; leafIndex++ {
			consumer(int(idx.dataAccess.GetInt(int64(leafIndex) * 4)))
		}
	}
}

// Query traverses the spatial index tree, calling visitor.OnEdge for each edge
// that passes the tile filter. Each edge ID is reported at most once.
func (idx *LineIntIndex) Query(tileFilter TileFilter, visitor Visitor) {
	seen := make(map[int]struct{})
	dedup := &dedupVisitor{
		inner: visitor,
		seen:  seen,
	}
	idx.queryRecursive(startPointer, tileFilter,
		idx.bounds.MinLat, idx.bounds.MinLon,
		idx.bounds.MaxLat-idx.bounds.MinLat,
		idx.bounds.MaxLon-idx.bounds.MinLon,
		dedup, 0)
}

type dedupVisitor struct {
	inner Visitor
	seen  map[int]struct{}
}

func (d *dedupVisitor) IsTileInfo() bool          { return d.inner.IsTileInfo() }
func (d *dedupVisitor) OnTile(bbox util.BBox, w int) { d.inner.OnTile(bbox, w) }
func (d *dedupVisitor) OnEdge(edgeID int) {
	if _, ok := d.seen[edgeID]; !ok {
		d.seen[edgeID] = struct{}{}
		d.inner.OnEdge(edgeID)
	}
}

func (idx *LineIntIndex) queryRecursive(intPointer int, tileFilter TileFilter,
	minLat, minLon, deltaLatPerDepth, deltaLonPerDepth float64,
	visitor Visitor, depth int) {

	pointer := int64(intPointer) * 4
	if depth == len(idx.entries) {
		nextIntPointer := int(idx.dataAccess.GetInt(pointer))
		if nextIntPointer < 0 {
			// single data entry
			visitor.OnEdge(-(nextIntPointer + 1))
		} else {
			maxPointer := int64(nextIntPointer) * 4
			for leafPointer := pointer + 4; leafPointer < maxPointer; leafPointer += 4 {
				visitor.OnEdge(int(idx.dataAccess.GetInt(leafPointer)))
			}
		}
		return
	}

	maxCells := 1 << idx.shifts[depth]
	factor := 4
	if maxCells == 4 {
		factor = 2
	}
	deltaLonPerDepth /= float64(factor)
	deltaLatPerDepth /= float64(factor)

	for cellIndex := range maxCells {
		nextIntPointer := int(idx.dataAccess.GetInt(pointer + int64(cellIndex)*4))
		if nextIntPointer <= 0 {
			continue
		}
		x, y := idx.keyAlgo.Decode(int64(cellIndex))
		tmpMinLon := minLon + deltaLonPerDepth*float64(x)
		tmpMinLat := minLat + deltaLatPerDepth*float64(y)

		var bbox util.BBox
		needBBox := tileFilter != nil || visitor.IsTileInfo()
		if needBBox {
			bbox = util.NewBBox(tmpMinLon, tmpMinLon+deltaLonPerDepth, tmpMinLat, tmpMinLat+deltaLatPerDepth)
		}
		if visitor.IsTileInfo() {
			visitor.OnTile(bbox, depth)
		}
		if tileFilter == nil || tileFilter.AcceptAll(bbox) {
			idx.queryRecursive(nextIntPointer, nil,
				tmpMinLat, tmpMinLon, deltaLatPerDepth, deltaLonPerDepth,
				visitor, depth+1)
		} else if tileFilter.AcceptPartially(bbox) {
			idx.queryRecursive(nextIntPointer, tileFilter,
				tmpMinLat, tmpMinLon, deltaLatPerDepth, deltaLonPerDepth,
				visitor, depth+1)
		}
	}
}

// FindEdgeIdsInNeighborhood collects edge IDs from the neighborhood of a
// point. With iteration=0 it looks only in the tile containing the point.
// With iteration=1,2,... it expands the search to a growing square of tiles.
func (idx *LineIntIndex) FindEdgeIdsInNeighborhood(queryLat, queryLon float64, iteration int, consumer func(edgeID int)) {
	x := idx.keyAlgo.X(queryLon)
	y := idx.keyAlgo.Y(queryLat)
	parts := idx.indexStructureInfo.Parts
	bits := idx.keyAlgo.Bits()
	shift := 64 - bits

	tryFill := func(qx, qy int) {
		if qx >= 0 && qy >= 0 && qx < parts && qy < parts {
			idx.fillIDs(idx.keyAlgo.Encode(qx, qy)<<shift, consumer)
		}
	}

	// Left and right columns (full height).
	for yreg := -iteration; yreg <= iteration; yreg++ {
		tryFill(x-iteration, y+yreg)
		if iteration > 0 {
			tryFill(x+iteration, y+yreg)
		}
	}

	// Top and bottom rows (excluding corners already covered).
	for xreg := -iteration + 1; xreg <= iteration-1; xreg++ {
		tryFill(x+xreg, y-iteration)
		tryFill(x+xreg, y+iteration)
	}
}

// GetChecksum returns the stored checksum.
func (idx *LineIntIndex) GetChecksum() int32 { return idx.checksum }

// SetChecksum sets the checksum value.
func (idx *LineIntIndex) SetChecksum(checksum int32) { idx.checksum = checksum }

// GetMinResolutionInMeter returns the minimum resolution in meters.
func (idx *LineIntIndex) GetMinResolutionInMeter() int { return idx.minResolutionInMeter }

// SetMinResolutionInMeter sets the minimum resolution in meters.
func (idx *LineIntIndex) SetMinResolutionInMeter(m int) { idx.minResolutionInMeter = m }

// Flush writes header data and flushes the underlying DataAccess.
func (idx *LineIntIndex) Flush() {
	idx.dataAccess.SetHeader(0, int32(util.VersionLocationIdx))
	idx.dataAccess.SetHeader(1*4, idx.checksum)
	idx.dataAccess.SetHeader(2*4, int32(idx.minResolutionInMeter))
	idx.dataAccess.Flush()
}

// Close releases the underlying DataAccess.
func (idx *LineIntIndex) Close() { idx.dataAccess.Close() }

// IsClosed returns true if the DataAccess is closed.
func (idx *LineIntIndex) IsClosed() bool { return idx.dataAccess.IsClosed() }

// GetCapacity returns the capacity of the underlying DataAccess.
func (idx *LineIntIndex) GetCapacity() int64 { return idx.dataAccess.Capacity() }

// GetSize returns the total number of stored edge references.
func (idx *LineIntIndex) GetSize() int { return idx.size }

// GetLeafs returns the number of leaf nodes.
func (idx *LineIntIndex) GetLeafs() int { return idx.leafs }

// GetIndexStructureInfo returns the index structure configuration.
func (idx *LineIntIndex) GetIndexStructureInfo() *IndexStructureInfo { return idx.indexStructureInfo }
