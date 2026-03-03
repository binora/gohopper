package coll

import "fmt"

// GHLongLongBTree is an in-memory B-Tree with configurable value size (1–8 bytes).
// Delete is not supported.
type GHLongLongBTree struct {
	emptyValue     int64
	maxLeafEntries int
	initLeafSize   int
	splitIndex     int
	factor         float32
	size           int64
	height         int
	root           *btreeEntry
	bytesPerValue  int
	maxValue       int64
}

func NewGHLongLongBTree(maxLeafEntries, bytesPerValue int, emptyValue int64) *GHLongLongBTree {
	if bytesPerValue > 8 {
		panic(fmt.Sprintf("Values can have 8 bytes maximum but requested was %d", bytesPerValue))
	}
	if maxLeafEntries < 1 {
		panic(fmt.Sprintf("illegal maxLeafEntries: %d", maxLeafEntries))
	}

	maxValue := int64(1<<(bytesPerValue*8-1)) - 1

	if maxLeafEntries%2 == 0 {
		maxLeafEntries++
	}

	splitIndex := maxLeafEntries / 2
	var factor float32
	var initLeafSize int
	switch {
	case maxLeafEntries < 10:
		factor = 2
		initLeafSize = 1
	case maxLeafEntries < 20:
		factor = 2
		initLeafSize = 4
	default:
		factor = 1.7
		initLeafSize = maxLeafEntries / 10
	}

	t := &GHLongLongBTree{
		emptyValue:     emptyValue,
		maxLeafEntries: maxLeafEntries,
		initLeafSize:   initLeafSize,
		splitIndex:     splitIndex,
		factor:         factor,
		bytesPerValue:  bytesPerValue,
		maxValue:       maxValue,
	}
	t.Clear()
	return t
}

func (t *GHLongLongBTree) Put(key, value int64) int64 {
	if value > t.maxValue {
		panic(fmt.Sprintf("Value %d exceeded max value: %d. Increase bytesPerValue (%d)", value, t.maxValue, t.bytesPerValue))
	}
	if value == t.emptyValue {
		panic(fmt.Sprintf("Value cannot be the 'empty value' %d", t.emptyValue))
	}

	rv := t.root.put(key, value, t)
	if rv.tree != nil {
		t.height++
		t.root = rv.tree
	}
	if rv.oldValue == nil {
		t.size++
		if t.size%1000000 == 0 {
			t.Optimize()
		}
	}
	if rv.oldValue == nil {
		return t.emptyValue
	}
	return t.toLong(rv.oldValue, 0)
}

func (t *GHLongLongBTree) Get(key int64) int64 {
	return t.root.get(key, t)
}

func (t *GHLongLongBTree) Height() int { return t.height }

func (t *GHLongLongBTree) GetSize() int64 { return t.size }

func (t *GHLongLongBTree) GetMemoryUsage() int {
	return int(t.root.getCapacity(t) / (1 << 20))
}

func (t *GHLongLongBTree) Clear() {
	t.size = 0
	t.height = 1
	t.root = newBTreeEntry(t.initLeafSize, true, t.bytesPerValue)
}

func (t *GHLongLongBTree) GetEmptyValue() int64 { return t.emptyValue }

func (t *GHLongLongBTree) GetMaxValue() int64 { return t.maxValue }

func (t *GHLongLongBTree) Optimize() {
	if t.GetSize() > 10000 {
		t.root.compact(t)
	}
}

// toLong decodes bytesPerValue bytes from b at the given offset into an int64.
// The topmost byte is sign-extended; lower bytes are zero-extended.
func (t *GHLongLongBTree) toLong(b []byte, offset int) int64 {
	var res int64
	bpv := t.bytesPerValue
	// topmost byte: sign-extended
	// lower bytes: zero-extended (& 0xFF)
	for i := bpv - 1; i >= 0; i-- {
		shift := uint(i * 8)
		if i == bpv-1 {
			res |= int64(int8(b[offset+i])) << shift
		} else {
			res |= int64(b[offset+i]&0xFF) << shift
		}
	}
	return res
}

// FromLong encodes a value into a new byte slice.
func (t *GHLongLongBTree) FromLong(value int64) []byte {
	bytes := make([]byte, t.bytesPerValue)
	t.fromLongInto(bytes, value, 0)
	return bytes
}

// ToLong decodes a byte slice to int64. Exported for testing.
func (t *GHLongLongBTree) ToLong(b []byte) int64 {
	return t.toLong(b, 0)
}

func (t *GHLongLongBTree) fromLongInto(bytes []byte, value int64, offset int) {
	for i := 0; i < t.bytesPerValue; i++ {
		bytes[offset+i] = byte(value >> uint(i*8))
	}
}

// binarySearch performs a binary search on keys[start..start+length).
// Returns the index if found, or ^insertionPoint if not found.
func binarySearch(keys []int64, start, length int, key int64) int {
	high := start + length
	low := start - 1
	for high-low > 1 {
		guess := int(uint(high+low) >> 1) // unsigned shift to avoid overflow
		if keys[guess] < key {
			low = guess
		} else {
			high = guess
		}
	}
	if high == start+length {
		return ^(start + length)
	}
	if keys[high] == key {
		return high
	}
	return ^high
}

// returnValue is the result of a put operation on a btreeEntry.
type returnValue struct {
	oldValue []byte      // nil means new insertion, non-nil means update
	tree     *btreeEntry // non-nil means a split occurred
}

type btreeEntry struct {
	entrySize int
	keys      []int64
	values    []byte
	children  []*btreeEntry
	isLeaf    bool
}

func newBTreeEntry(size int, leaf bool, bytesPerValue int) *btreeEntry {
	e := &btreeEntry{
		isLeaf: leaf,
		keys:   make([]int64, size),
		values: make([]byte, size*bytesPerValue),
	}
	if !leaf {
		e.children = make([]*btreeEntry, size+1)
	}
	return e
}

func (e *btreeEntry) put(key, newValue int64, t *GHLongLongBTree) returnValue {
	index := binarySearch(e.keys, 0, e.entrySize, key)
	if index >= 0 {
		// update existing key
		bpv := t.bytesPerValue
		oldValue := make([]byte, bpv)
		copy(oldValue, e.values[index*bpv:(index+1)*bpv])
		t.fromLongInto(e.values, newValue, index*bpv)
		return returnValue{oldValue: oldValue}
	}

	index = ^index
	if e.isLeaf || e.children[index] == nil {
		// insert into this node
		rv := returnValue{}
		rv.tree = e.checkSplitEntry(t)
		newValueBytes := t.FromLong(newValue)
		if rv.tree == nil {
			e.insertKeyValue(index, key, newValueBytes, t)
		} else if index <= t.splitIndex {
			rv.tree.children[0].insertKeyValue(index, key, newValueBytes, t)
		} else {
			rv.tree.children[1].insertKeyValue(index-t.splitIndex-1, key, newValueBytes, t)
		}
		return rv
	}

	// recurse into child
	downRV := e.children[index].put(key, newValue, t)
	if downRV.oldValue != nil {
		return downRV
	}

	if downRV.tree != nil {
		returnTree := e.checkSplitEntry(t)
		if returnTree == nil {
			e.insertTree(index, downRV.tree, t)
		} else if index <= t.splitIndex {
			returnTree.children[0].insertTree(index, downRV.tree, t)
		} else {
			returnTree.children[1].insertTree(index-t.splitIndex-1, downRV.tree, t)
		}
		downRV.tree = returnTree
	}
	return downRV
}

func (e *btreeEntry) checkSplitEntry(t *GHLongLongBTree) *btreeEntry {
	if e.entrySize < t.maxLeafEntries {
		return nil
	}

	bpv := t.bytesPerValue
	splitIdx := t.splitIndex

	// right child
	count := e.entrySize - splitIdx - 1
	rightSize := max(t.initLeafSize, count)
	right := newBTreeEntry(rightSize, e.isLeaf, bpv)
	copyEntry(e, right, splitIdx+1, count, bpv)

	// left child
	leftSize := max(t.initLeafSize, splitIdx)
	left := newBTreeEntry(leftSize, e.isLeaf, bpv)
	copyEntry(e, left, 0, splitIdx, bpv)

	// new parent with one key
	parent := newBTreeEntry(1, false, bpv)
	parent.entrySize = 1
	parent.keys[0] = e.keys[splitIdx]
	copy(parent.values[0:bpv], e.values[splitIdx*bpv:(splitIdx+1)*bpv])
	parent.children[0] = left
	parent.children[1] = right
	return parent
}

func copyEntry(from, to *btreeEntry, fromIdx, count, bpv int) {
	copy(to.keys[:count], from.keys[fromIdx:fromIdx+count])
	copy(to.values[:count*bpv], from.values[fromIdx*bpv:(fromIdx+count)*bpv])
	if !from.isLeaf {
		copy(to.children[:count+1], from.children[fromIdx:fromIdx+count+1])
	}
	to.entrySize = count
}

func (e *btreeEntry) insertKeyValue(index int, key int64, newValue []byte, t *GHLongLongBTree) {
	bpv := t.bytesPerValue
	e.ensureSize(e.entrySize+1, t)
	count := e.entrySize - index
	if count > 0 {
		copy(e.keys[index+1:index+1+count], e.keys[index:index+count])
		copy(e.values[(index+1)*bpv:(index+1+count)*bpv], e.values[index*bpv:(index+count)*bpv])
		if !e.isLeaf {
			copy(e.children[index+2:index+2+count], e.children[index+1:index+1+count])
		}
	}
	e.keys[index] = key
	copy(e.values[index*bpv:(index+1)*bpv], newValue)
	e.entrySize++
}

func (e *btreeEntry) insertTree(index int, tree *btreeEntry, t *GHLongLongBTree) {
	e.insertKeyValue(index, tree.keys[0], tree.values[:t.bytesPerValue], t)
	if !e.isLeaf {
		e.children[index] = tree.children[0]
		e.children[index+1] = tree.children[1]
	}
}

func (e *btreeEntry) get(key int64, t *GHLongLongBTree) int64 {
	index := binarySearch(e.keys, 0, e.entrySize, key)
	if index >= 0 {
		return t.toLong(e.values, index*t.bytesPerValue)
	}
	index = ^index
	if e.isLeaf || e.children[index] == nil {
		return t.emptyValue
	}
	return e.children[index].get(key, t)
}

func (e *btreeEntry) getCapacity(t *GHLongLongBTree) int64 {
	cap := int64(len(e.keys)*(8+4) + 3*12 + 4 + 1)
	if !e.isLeaf {
		cap += int64(len(e.children) * 8) // pointer size on 64-bit
		for _, c := range e.children {
			if c != nil {
				cap += c.getCapacity(t)
			}
		}
	}
	return cap
}

func (e *btreeEntry) getEntries() int {
	entries := 1
	if !e.isLeaf {
		for _, c := range e.children {
			if c != nil {
				entries += c.getEntries()
			}
		}
	}
	return entries
}

func (e *btreeEntry) ensureSize(size int, t *GHLongLongBTree) {
	if size <= len(e.keys) {
		return
	}
	grown := int(float32(size) * t.factor)
	newSize := min(t.maxLeafEntries, max(size+1, grown))

	newKeys := make([]int64, newSize)
	copy(newKeys, e.keys)
	e.keys = newKeys

	bpv := t.bytesPerValue
	newValues := make([]byte, newSize*bpv)
	copy(newValues, e.values)
	e.values = newValues

	if !e.isLeaf {
		newChildren := make([]*btreeEntry, newSize+1)
		copy(newChildren, e.children)
		e.children = newChildren
	}
}

func (e *btreeEntry) compact(t *GHLongLongBTree) {
	bpv := t.bytesPerValue
	tolerance := 1
	if e.entrySize+tolerance < len(e.keys) {
		newKeys := make([]int64, e.entrySize)
		copy(newKeys, e.keys)
		e.keys = newKeys

		newValues := make([]byte, e.entrySize*bpv)
		copy(newValues, e.values)
		e.values = newValues

		if !e.isLeaf {
			newChildren := make([]*btreeEntry, e.entrySize+1)
			copy(newChildren, e.children)
			e.children = newChildren
		}
	}

	if !e.isLeaf {
		for _, c := range e.children {
			if c != nil {
				c.compact(t)
			}
		}
	}
}
